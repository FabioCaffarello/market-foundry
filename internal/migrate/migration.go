package migrate

import "time"

// Migration represents a single migration file discovered in the catalog.
type Migration struct {
	Version uint32
	Name    string
	Path    string
}

// AppliedMigration represents a migration that has been applied to the database.
type AppliedMigration struct {
	Version   uint32
	Name      string
	AppliedAt time.Time
	Checksum  string
}

// MigrationStatus combines catalog and applied state for reporting.
type MigrationStatus struct {
	Migration
	Applied  bool
	Checksum string
	Drift    bool // true if applied checksum != current file checksum
}
