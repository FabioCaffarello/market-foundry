package clickhouse_test

import (
	"testing"
	"time"

	"internal/adapters/clickhouse"
)

// ── BuildCandleQuery ────────────────────────────────────────────

func TestBuildCandleQuery_BasicFilters(t *testing.T) {
	q, args := clickhouse.BuildCandleQuery("binancef", "btcusdt", 60, 0, 0, 50)

	expectContains(t, q, "source = ?")
	expectContains(t, q, "symbol = ?")
	expectContains(t, q, "timeframe = ?")
	expectContains(t, q, "ORDER BY open_time DESC")
	expectContains(t, q, "LIMIT ?")

	// 3 base args (source, symbol, timeframe) + 1 limit.
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}

	assertQueryArg(t, "source", args[0], "binancef")
	assertQueryArg(t, "symbol", args[1], "btcusdt")
	assertQueryArg(t, "timeframe", args[2], uint32(60))
	assertQueryArg(t, "limit", args[3], 50)
}

func TestBuildCandleQuery_NoTimeFilters(t *testing.T) {
	q, _ := clickhouse.BuildCandleQuery("binancef", "btcusdt", 60, 0, 0, 50)

	expectNotContains(t, q, "open_time >=")
	expectNotContains(t, q, "open_time <=")
}

func TestBuildCandleQuery_WithSince(t *testing.T) {
	since := int64(1710500000)
	q, args := clickhouse.BuildCandleQuery("binancef", "btcusdt", 60, since, 0, 50)

	expectContains(t, q, "open_time >= ?")
	expectNotContains(t, q, "open_time <= ?")

	// 3 base + 1 since + 1 limit = 5.
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d", len(args))
	}

	sinceArg := args[3].(time.Time)
	if sinceArg.Unix() != since {
		t.Errorf("since arg: expected unix %d, got %d", since, sinceArg.Unix())
	}
}

func TestBuildCandleQuery_WithUntil(t *testing.T) {
	until := int64(1710600000)
	q, args := clickhouse.BuildCandleQuery("binancef", "btcusdt", 60, 0, until, 50)

	expectNotContains(t, q, "open_time >= ?")
	expectContains(t, q, "open_time <= ?")

	// 3 base + 1 until + 1 limit = 5.
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d", len(args))
	}

	untilArg := args[3].(time.Time)
	if untilArg.Unix() != until {
		t.Errorf("until arg: expected unix %d, got %d", until, untilArg.Unix())
	}
}

func TestBuildCandleQuery_WithSinceAndUntil(t *testing.T) {
	since := int64(1710500000)
	until := int64(1710600000)
	q, args := clickhouse.BuildCandleQuery("binancef", "btcusdt", 300, since, until, 100)

	expectContains(t, q, "open_time >= ?")
	expectContains(t, q, "open_time <= ?")

	// 3 base + 1 since + 1 until + 1 limit = 6.
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d", len(args))
	}

	sinceArg := args[3].(time.Time)
	untilArg := args[4].(time.Time)
	if sinceArg.Unix() != since {
		t.Errorf("since arg: expected unix %d, got %d", since, sinceArg.Unix())
	}
	if untilArg.Unix() != until {
		t.Errorf("until arg: expected unix %d, got %d", until, untilArg.Unix())
	}
	assertQueryArg(t, "limit", args[5], 100)
}

func TestBuildCandleQuery_TimeframeAsUint32(t *testing.T) {
	_, args := clickhouse.BuildCandleQuery("binancef", "btcusdt", 300, 0, 0, 50)

	tf, ok := args[2].(uint32)
	if !ok {
		t.Fatalf("timeframe arg should be uint32, got %T", args[2])
	}
	if tf != 300 {
		t.Errorf("expected timeframe 300, got %d", tf)
	}
}

func TestBuildCandleQuery_SelectColumns(t *testing.T) {
	q, _ := clickhouse.BuildCandleQuery("binancef", "btcusdt", 60, 0, 0, 50)

	// Verify the 12 columns in the SELECT match the DDL read path.
	expectedCols := []string{
		"source", "symbol", "timeframe",
		"open", "high", "low", "close",
		"volume", "trade_count",
		"open_time", "close_time", "final",
	}
	for _, col := range expectedCols {
		expectContains(t, q, col)
	}
}

// ── FormatFloat ─────────────────────────────────────────────────

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{100.5, "100.5"},
		{0, "0"},
		{-1.23, "-1.23"},
		{0.000001, "0.000001"},
		{99999999.99, "99999999.99"},
		{1.0, "1"},
		{0.123456789, "0.123456789"},
	}

	for _, tt := range tests {
		got := clickhouse.FormatFloat(tt.input)
		if got != tt.expected {
			t.Errorf("FormatFloat(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// ── Helpers ─────────────────────────────────────────────────────

func expectContains(t *testing.T, s, substr string) {
	t.Helper()
	if !contains(s, substr) {
		t.Errorf("expected query to contain %q, got:\n%s", substr, s)
	}
}

func expectNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if contains(s, substr) {
		t.Errorf("expected query NOT to contain %q, got:\n%s", substr, s)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func assertQueryArg(t *testing.T, name string, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("arg %s: got %v (%T), want %v (%T)", name, got, got, want, want)
	}
}
