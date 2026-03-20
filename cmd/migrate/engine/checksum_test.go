package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileChecksum_Deterministic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sql")
	if err := os.WriteFile(path, []byte("CREATE TABLE foo (id UInt32) ENGINE = MergeTree() ORDER BY id"), 0o644); err != nil {
		t.Fatal(err)
	}

	c1, err := FileChecksum(path)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := FileChecksum(path)
	if err != nil {
		t.Fatal(err)
	}

	if c1 != c2 {
		t.Errorf("checksums differ: %q vs %q", c1, c2)
	}
	if len(c1) != 64 {
		t.Errorf("expected 64-char hex digest, got %d chars", len(c1))
	}
}

func TestFileChecksum_DifferentContent(t *testing.T) {
	dir := t.TempDir()

	p1 := filepath.Join(dir, "a.sql")
	p2 := filepath.Join(dir, "b.sql")
	os.WriteFile(p1, []byte("SELECT 1"), 0o644)
	os.WriteFile(p2, []byte("SELECT 2"), 0o644)

	c1, _ := FileChecksum(p1)
	c2, _ := FileChecksum(p2)
	if c1 == c2 {
		t.Error("different content produced same checksum")
	}
}

func TestFileChecksum_MissingFile(t *testing.T) {
	_, err := FileChecksum("/nonexistent/file.sql")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
