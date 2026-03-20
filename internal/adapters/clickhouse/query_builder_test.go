package clickhouse_test

import (
	"testing"
	"time"

	"internal/adapters/clickhouse"
)

// ── BuildQuery ──────────────────────────────────────────────────

func TestBuildQuery_MandatoryOnly(t *testing.T) {
	q, args := clickhouse.BuildQuery(
		"a, b, c", "my_table",
		"a = ? AND b = ?", []any{"x", "y"},
		nil,
		"ts", 0, 0, "ts", 10,
	)

	expectContains(t, q, "SELECT a, b, c")
	expectContains(t, q, "FROM my_table")
	expectContains(t, q, "WHERE a = ? AND b = ?")
	expectContains(t, q, "ORDER BY ts DESC LIMIT ?")
	expectNotContains(t, q, "ts >= ?")
	expectNotContains(t, q, "ts <= ?")

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	assertQueryArg(t, "a", args[0], "x")
	assertQueryArg(t, "b", args[1], "y")
	assertQueryArg(t, "limit", args[2], 10)
}

func TestBuildQuery_WithOptionalFilters(t *testing.T) {
	q, args := clickhouse.BuildQuery(
		"a, b", "tbl",
		"a = ?", []any{"v1"},
		[]clickhouse.OptionalFilter{
			{Column: "status", Value: "active"},
			{Column: "side", Value: "buy"},
		},
		"ts", 0, 0, "ts", 25,
	)

	expectContains(t, q, "AND status = ?")
	expectContains(t, q, "AND side = ?")

	// 1 mandatory + 2 optional + 1 limit = 4.
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}
	assertQueryArg(t, "mandatory", args[0], "v1")
	assertQueryArg(t, "status", args[1], "active")
	assertQueryArg(t, "side", args[2], "buy")
	assertQueryArg(t, "limit", args[3], 25)
}

func TestBuildQuery_WithSinceOnly(t *testing.T) {
	since := int64(1710500000)
	q, args := clickhouse.BuildQuery(
		"col", "tbl",
		"x = ?", []any{"a"},
		nil,
		"created_at", since, 0, "created_at", 50,
	)

	expectContains(t, q, "AND created_at >= ?")
	expectNotContains(t, q, "created_at <= ?")

	// 1 mandatory + 1 since + 1 limit = 3.
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}

	sinceArg := args[1].(time.Time)
	if sinceArg.Unix() != since {
		t.Errorf("since arg: expected unix %d, got %d", since, sinceArg.Unix())
	}
}

func TestBuildQuery_WithUntilOnly(t *testing.T) {
	until := int64(1710600000)
	q, args := clickhouse.BuildQuery(
		"col", "tbl",
		"x = ?", []any{"a"},
		nil,
		"created_at", 0, until, "created_at", 50,
	)

	expectNotContains(t, q, "created_at >= ?")
	expectContains(t, q, "AND created_at <= ?")

	// 1 mandatory + 1 until + 1 limit = 3.
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}

	untilArg := args[1].(time.Time)
	if untilArg.Unix() != until {
		t.Errorf("until arg: expected unix %d, got %d", until, untilArg.Unix())
	}
}

func TestBuildQuery_WithSinceAndUntil(t *testing.T) {
	since := int64(1710500000)
	until := int64(1710600000)
	q, args := clickhouse.BuildQuery(
		"col", "tbl",
		"x = ?", []any{"a"},
		nil,
		"ts", since, until, "ts", 100,
	)

	expectContains(t, q, "AND ts >= ?")
	expectContains(t, q, "AND ts <= ?")

	// 1 mandatory + 1 since + 1 until + 1 limit = 4.
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}

	sinceArg := args[1].(time.Time)
	untilArg := args[2].(time.Time)
	if sinceArg.Unix() != since {
		t.Errorf("since arg: expected unix %d, got %d", since, sinceArg.Unix())
	}
	if untilArg.Unix() != until {
		t.Errorf("until arg: expected unix %d, got %d", until, untilArg.Unix())
	}
	assertQueryArg(t, "limit", args[3], 100)
}

func TestBuildQuery_FiltersAndTimeRange(t *testing.T) {
	since := int64(1710500000)
	until := int64(1710600000)
	q, args := clickhouse.BuildQuery(
		"a, b", "tbl",
		"a = ?", []any{"v"},
		[]clickhouse.OptionalFilter{
			{Column: "outcome", Value: "triggered"},
		},
		"timestamp", since, until, "timestamp", 50,
	)

	expectContains(t, q, "AND outcome = ?")
	expectContains(t, q, "AND timestamp >= ?")
	expectContains(t, q, "AND timestamp <= ?")

	// 1 mandatory + 1 filter + 1 since + 1 until + 1 limit = 5.
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d", len(args))
	}

	assertQueryArg(t, "mandatory", args[0], "v")
	assertQueryArg(t, "outcome", args[1], "triggered")
	assertQueryArg(t, "limit", args[4], 50)
}

func TestBuildQuery_DoesNotMutateMandatoryArgs(t *testing.T) {
	original := []any{"a", "b"}
	origLen := len(original)

	clickhouse.BuildQuery(
		"col", "tbl",
		"x = ? AND y = ?", original,
		[]clickhouse.OptionalFilter{{Column: "z", Value: "c"}},
		"ts", 1, 2, "ts", 10,
	)

	if len(original) != origLen {
		t.Errorf("mandatoryArgs was mutated: expected len %d, got %d", origLen, len(original))
	}
}

func TestBuildQuery_DifferentTimeAndOrderColumns(t *testing.T) {
	q, _ := clickhouse.BuildQuery(
		"col", "tbl",
		"x = ?", []any{"a"},
		nil,
		"open_time", 1, 0, "open_time", 10,
	)

	expectContains(t, q, "AND open_time >= ?")
	expectContains(t, q, "ORDER BY open_time DESC")
}
