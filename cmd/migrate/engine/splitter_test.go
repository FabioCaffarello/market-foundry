package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every shipped migration (000–013) is single-statement by the
// H-6.d.1 per-table split convention; the splitter must return each
// verbatim as exactly one statement (comment headers preserved).
// This pins the splitter against the real on-disk shapes per
// H-6.f.1 Decisão #5.
func TestSplitStatements_RealMigrationsAreSingleStatement(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "deploy", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("migrations dir not available: %v", err)
	}

	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		stmts := SplitStatements(string(raw))
		if len(stmts) != 1 {
			t.Errorf("%s: expected 1 statement, got %d", e.Name(), len(stmts))
			continue
		}
		checked++
	}
	if checked < 14 {
		t.Errorf("expected >= 14 migration files checked, got %d", checked)
	}
}

func TestSplitStatements_SyntheticMultiStatement(t *testing.T) {
	content := `-- Migration: 099_synthetic_multi
-- Description: two ALTERs in one logical unit (the H-6.d.1 shape
--              that forced the per-table split).

ALTER TABLE evidence_candles
    ADD COLUMN IF NOT EXISTS base LowCardinality(String) DEFAULT '' AFTER symbol;

ALTER TABLE signals
    ADD COLUMN IF NOT EXISTS base LowCardinality(String) DEFAULT '' AFTER symbol;
`
	stmts := SplitStatements(content)
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %q", len(stmts), stmts)
	}
	if !strings.Contains(stmts[0], "evidence_candles") || !strings.HasPrefix(stmts[0], "-- Migration") {
		t.Errorf("statement 1 lost content or comment header: %q", stmts[0])
	}
	if !strings.Contains(stmts[1], "ALTER TABLE signals") {
		t.Errorf("statement 2 mismatch: %q", stmts[1])
	}
	for i, s := range stmts {
		if strings.Contains(s, ";") {
			t.Errorf("statement %d retains separator: %q", i+1, s)
		}
	}
}

func TestSplitStatements_SeparatorsInStringsAndComments(t *testing.T) {
	content := `-- header comment; with a semicolon
/* block; comment */
INSERT INTO t (a, b) VALUES ('x;y', 'it''s; fine'); -- trailing; note
SELECT 1; /* tail block;
spanning lines */
`
	stmts := SplitStatements(content)
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %q", len(stmts), stmts)
	}
	if !strings.Contains(stmts[0], "'x;y'") || !strings.Contains(stmts[0], "it''s; fine") {
		t.Errorf("string literals mangled: %q", stmts[0])
	}
	if !strings.HasPrefix(stmts[1], "-- trailing; note") || !strings.Contains(stmts[1], "SELECT 1") {
		t.Errorf("statement 2 mismatch: %q", stmts[1])
	}
}

func TestSplitStatements_CommentOnlyAndEmptySegmentsDropped(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    int
	}{
		{"empty", "", 0},
		{"whitespace", "  \n\t\n", 0},
		{"comments_only", "-- just a header\n/* and a block */\n", 0},
		{"trailing_semicolons", "SELECT 1;;\n;\n-- done\n", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SplitStatements(tc.content); len(got) != tc.want {
				t.Errorf("expected %d statements, got %d: %q", tc.want, len(got), got)
			}
		})
	}
}

func TestSplitStatements_NoTrailingSeparator(t *testing.T) {
	// 000–013 end without a trailing ';' today — the final segment
	// must still be emitted.
	stmts := SplitStatements("ALTER TABLE t ADD COLUMN IF NOT EXISTS c String")
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
}
