package engine

import (
	"context"
	"database/sql"
	"fmt"
	"os"
)

const bootstrapDDL = `
CREATE TABLE IF NOT EXISTS _migrations (
    version    UInt32,
    name       String,
    applied_at DateTime64(3) DEFAULT now64(3),
    checksum   String
) ENGINE = MergeTree()
ORDER BY version
`

// Runner orchestrates migration operations against a ClickHouse database.
// It expects a *sql.DB connected to the target database.
type Runner struct {
	db  *sql.DB
	dir string
}

// NewRunner creates a runner that reads migrations from dir and applies them via db.
func NewRunner(db *sql.DB, dir string) *Runner {
	return &Runner{db: db, dir: dir}
}

// bootstrap ensures the _migrations metadata table exists.
func (r *Runner) bootstrap(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, bootstrapDDL)
	return err
}

// applied returns all previously applied migrations keyed by version.
func (r *Runner) applied(ctx context.Context) (map[uint32]AppliedMigration, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT version, name, applied_at, checksum FROM _migrations ORDER BY version",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uint32]AppliedMigration)
	for rows.Next() {
		var m AppliedMigration
		if err := rows.Scan(&m.Version, &m.Name, &m.AppliedAt, &m.Checksum); err != nil {
			return nil, err
		}
		result[m.Version] = m
	}
	return result, rows.Err()
}

// Up applies all pending migrations in catalog order.
// If dryRun is true, it prints what would be applied without executing.
// Execution stops on the first failure.
func (r *Runner) Up(ctx context.Context, dryRun bool) error {
	if err := r.bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap metadata table: %w", err)
	}

	catalog, err := ReadCatalog(r.dir)
	if err != nil {
		return fmt.Errorf("read catalog: %w", err)
	}

	appliedMap, err := r.applied(ctx)
	if err != nil {
		return fmt.Errorf("read applied migrations: %w", err)
	}

	pending := 0
	for _, m := range catalog {
		if _, ok := appliedMap[m.Version]; ok {
			continue
		}
		pending++

		checksum, err := FileChecksum(m.Path)
		if err != nil {
			return fmt.Errorf("checksum %s: %w", m.Name, err)
		}

		if dryRun {
			fmt.Printf("  [pending] %s  (%s)\n", m.Name, checksum[:12])
			continue
		}

		content, err := os.ReadFile(m.Path)
		if err != nil {
			return fmt.Errorf("read %s: %w", m.Name, err)
		}

		fmt.Printf("  applying %s ... ", m.Name)
		// One ExecContext per statement: clickhouse-go/v2 rejects
		// multi-statement payloads (code 62) — see SplitStatements.
		// No transactional rollback across statements (ClickHouse DDL
		// is non-transactional); a mid-file failure stops the run
		// before the migration is recorded, and idempotent statements
		// (IF NOT EXISTS) make the retry safe.
		stmts := SplitStatements(string(content))
		for i, stmt := range stmts {
			if _, err := r.db.ExecContext(ctx, stmt); err != nil {
				fmt.Println("FAILED")
				return fmt.Errorf("apply %s (statement %d/%d): %w", m.Name, i+1, len(stmts), err)
			}
		}

		if _, err := r.db.ExecContext(ctx,
			"INSERT INTO _migrations (version, name, checksum) VALUES (?, ?, ?)",
			m.Version, m.Name, checksum,
		); err != nil {
			fmt.Println("FAILED (record)")
			return fmt.Errorf("record %s in _migrations: %w", m.Name, err)
		}

		fmt.Println("OK")
	}

	if dryRun && pending > 0 {
		fmt.Printf("\n  %d migration(s) would be applied\n", pending)
	} else if pending == 0 {
		fmt.Println("  no pending migrations")
	}

	return nil
}

// Status prints the state of all catalog migrations (applied or pending).
func (r *Runner) Status(ctx context.Context) error {
	if err := r.bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap metadata table: %w", err)
	}

	catalog, err := ReadCatalog(r.dir)
	if err != nil {
		return fmt.Errorf("read catalog: %w", err)
	}

	appliedMap, err := r.applied(ctx)
	if err != nil {
		return fmt.Errorf("read applied migrations: %w", err)
	}

	appliedCount := 0
	pendingCount := 0

	for _, m := range catalog {
		checksum, err := FileChecksum(m.Path)
		if err != nil {
			return fmt.Errorf("checksum %s: %w", m.Name, err)
		}

		if applied, ok := appliedMap[m.Version]; ok {
			appliedCount++
			drift := ""
			if applied.Checksum != checksum {
				drift = "  ** DRIFT **"
			}
			fmt.Printf("  [applied]  %s  applied_at=%s  checksum=%s%s\n",
				m.Name, applied.AppliedAt.Format("2006-01-02 15:04:05"), applied.Checksum[:12], drift)
		} else {
			pendingCount++
			fmt.Printf("  [pending]  %s  checksum=%s\n", m.Name, checksum[:12])
		}
	}

	fmt.Printf("\n  %d applied, %d pending\n", appliedCount, pendingCount)
	return nil
}

// Validate checks that all applied migrations have checksums matching the current
// catalog files. Returns a non-nil error if any drift is detected.
func (r *Runner) Validate(ctx context.Context) error {
	if err := r.bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap metadata table: %w", err)
	}

	catalog, err := ReadCatalog(r.dir)
	if err != nil {
		return fmt.Errorf("read catalog: %w", err)
	}

	appliedMap, err := r.applied(ctx)
	if err != nil {
		return fmt.Errorf("read applied migrations: %w", err)
	}

	driftCount := 0
	for _, m := range catalog {
		applied, ok := appliedMap[m.Version]
		if !ok {
			continue // not applied yet, nothing to validate
		}

		checksum, err := FileChecksum(m.Path)
		if err != nil {
			return fmt.Errorf("checksum %s: %w", m.Name, err)
		}

		if applied.Checksum != checksum {
			driftCount++
			fmt.Printf("  DRIFT  %s\n", m.Name)
			fmt.Printf("         applied:  %s\n", applied.Checksum[:12])
			fmt.Printf("         current:  %s\n", checksum[:12])
		}
	}

	if driftCount > 0 {
		return fmt.Errorf("%d migration(s) have checksum drift", driftCount)
	}

	fmt.Println("  all checksums valid")
	return nil
}
