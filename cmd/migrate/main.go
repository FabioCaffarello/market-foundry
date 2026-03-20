package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"

	"cmd/migrate/migrate"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		usage()
		return
	}

	flags := flag.NewFlagSet(cmd, flag.ExitOnError)
	dryRun := flags.Bool("dry-run", false, "show pending migrations without applying (up only)")
	migrationsDir := flags.String("migrations-dir", envOrDefault("MIGRATIONS_DIR", "deploy/migrations"), "path to migrations directory")
	flags.Parse(os.Args[2:])

	switch cmd {
	case "up", "status", "validate":
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := openClickHouse(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "clickhouse: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	runner := migrate.NewRunner(db, *migrationsDir)

	fmt.Printf("migrate %s\n", cmd)
	switch cmd {
	case "up":
		err = runner.Up(ctx, *dryRun)
	case "status":
		err = runner.Status(ctx)
	case "validate":
		err = runner.Validate(ctx)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
		os.Exit(1)
	}
}

// openClickHouse establishes a connection to ClickHouse.
// It first ensures the target database exists, then reconnects to it.
func openClickHouse(ctx context.Context) (*sql.DB, error) {
	host := envOrDefault("CLICKHOUSE_HOST", "localhost")
	port := envOrDefault("CLICKHOUSE_PORT", "9000")
	user := envOrDefault("CLICKHOUSE_USER", "default")
	pass := envOrDefault("CLICKHOUSE_PASSWORD", "")
	database := envOrDefault("CLICKHOUSE_DATABASE", "market_foundry")

	// Connect without database to bootstrap it.
	baseDSN := fmt.Sprintf("clickhouse://%s:%s@%s:%s/?dial_timeout=5s", user, pass, host, port)
	base, err := sql.Open("clickhouse", baseDSN)
	if err != nil {
		return nil, fmt.Errorf("open base connection: %w", err)
	}
	defer base.Close()

	if err := base.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}

	if _, err := base.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+database); err != nil {
		return nil, fmt.Errorf("create database %s: %w", database, err)
	}

	// Reconnect to target database.
	targetDSN := fmt.Sprintf("clickhouse://%s:%s@%s:%s/%s?dial_timeout=5s", user, pass, host, port, database)
	db, err := sql.Open("clickhouse", targetDSN)
	if err != nil {
		return nil, fmt.Errorf("open target connection: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping target database: %w", err)
	}

	return db, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage: migrate <command> [options]

Commands:
  up         Apply pending migrations in order
  status     Show applied/pending migration status
  validate   Verify checksums of applied migrations

Options:
  --dry-run          Show what would be applied without executing (up only)
  --migrations-dir   Path to migrations directory (default: deploy/migrations)

Environment:
  CLICKHOUSE_HOST      ClickHouse host (default: localhost)
  CLICKHOUSE_PORT      ClickHouse native port (default: 9000)
  CLICKHOUSE_DATABASE  Target database (default: market_foundry)
  CLICKHOUSE_USER      ClickHouse user (default: default)
  CLICKHOUSE_PASSWORD  ClickHouse password (default: empty)
  MIGRATIONS_DIR       Override --migrations-dir default
`)
}
