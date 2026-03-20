package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCatalog_SortsAndParses(t *testing.T) {
	dir := t.TempDir()

	// Create files out of order.
	files := []string{
		"003_create_decisions.sql",
		"001_create_evidence_candles.sql",
		"002_create_signals.sql",
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("-- "+f), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	catalog, err := ReadCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(catalog) != 3 {
		t.Fatalf("expected 3 migrations, got %d", len(catalog))
	}

	// Verify sorted order.
	want := []struct {
		version uint32
		name    string
	}{
		{1, "001_create_evidence_candles"},
		{2, "002_create_signals"},
		{3, "003_create_decisions"},
	}

	for i, w := range want {
		if catalog[i].Version != w.version {
			t.Errorf("catalog[%d].Version = %d, want %d", i, catalog[i].Version, w.version)
		}
		if catalog[i].Name != w.name {
			t.Errorf("catalog[%d].Name = %q, want %q", i, catalog[i].Name, w.name)
		}
	}
}

func TestReadCatalog_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	catalog, err := ReadCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) != 0 {
		t.Fatalf("expected 0 migrations, got %d", len(catalog))
	}
}

func TestReadCatalog_RejectsDuplicateVersion(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"001_create_foo.sql", "001_create_bar.sql"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("--"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	_, err := ReadCatalog(dir)
	if err == nil {
		t.Fatal("expected error for duplicate version, got nil")
	}
}

func TestReadCatalog_RejectsInvalidFilename(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad_name.sql"), []byte("--"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadCatalog(dir)
	if err == nil {
		t.Fatal("expected error for invalid filename, got nil")
	}
}

func TestReadCatalog_IgnoresNonSQL(t *testing.T) {
	dir := t.TempDir()
	// SQL file.
	if err := os.WriteFile(filepath.Join(dir, "001_create_foo.sql"), []byte("--"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Non-SQL file should be ignored (not matched by *.sql glob).
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	catalog, err := ReadCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) != 1 {
		t.Fatalf("expected 1 migration, got %d", len(catalog))
	}
}
