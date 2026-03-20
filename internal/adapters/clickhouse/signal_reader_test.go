package clickhouse_test

import (
	"testing"
	"time"

	"internal/adapters/clickhouse"
)

// ── BuildSignalQuery ────────────────────────────────────────────

func TestBuildSignalQuery_BasicFilters(t *testing.T) {
	q, args := clickhouse.BuildSignalQuery("rsi", "binancef", "btcusdt", 60, 0, 0, 50)

	expectContains(t, q, "type = ?")
	expectContains(t, q, "source = ?")
	expectContains(t, q, "symbol = ?")
	expectContains(t, q, "timeframe = ?")
	expectContains(t, q, "ORDER BY timestamp DESC")
	expectContains(t, q, "LIMIT ?")

	// 4 base args (type, source, symbol, timeframe) + 1 limit.
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d: %v", len(args), args)
	}

	assertQueryArg(t, "type", args[0], "rsi")
	assertQueryArg(t, "source", args[1], "binancef")
	assertQueryArg(t, "symbol", args[2], "btcusdt")
	assertQueryArg(t, "timeframe", args[3], uint32(60))
	assertQueryArg(t, "limit", args[4], 50)
}

func TestBuildSignalQuery_NoTimeFilters(t *testing.T) {
	q, _ := clickhouse.BuildSignalQuery("rsi", "binancef", "btcusdt", 60, 0, 0, 50)

	expectNotContains(t, q, "timestamp >=")
	expectNotContains(t, q, "timestamp <=")
}

func TestBuildSignalQuery_WithSince(t *testing.T) {
	since := int64(1710500000)
	q, args := clickhouse.BuildSignalQuery("rsi", "binancef", "btcusdt", 60, since, 0, 50)

	expectContains(t, q, "timestamp >= ?")
	expectNotContains(t, q, "timestamp <= ?")

	// 4 base + 1 since + 1 limit = 6.
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d", len(args))
	}

	sinceArg := args[4].(time.Time)
	if sinceArg.Unix() != since {
		t.Errorf("since arg: expected unix %d, got %d", since, sinceArg.Unix())
	}
}

func TestBuildSignalQuery_WithUntil(t *testing.T) {
	until := int64(1710600000)
	q, args := clickhouse.BuildSignalQuery("rsi", "binancef", "btcusdt", 60, 0, until, 50)

	expectNotContains(t, q, "timestamp >= ?")
	expectContains(t, q, "timestamp <= ?")

	// 4 base + 1 until + 1 limit = 6.
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d", len(args))
	}

	untilArg := args[4].(time.Time)
	if untilArg.Unix() != until {
		t.Errorf("until arg: expected unix %d, got %d", until, untilArg.Unix())
	}
}

func TestBuildSignalQuery_WithSinceAndUntil(t *testing.T) {
	since := int64(1710500000)
	until := int64(1710600000)
	q, args := clickhouse.BuildSignalQuery("rsi", "binancef", "btcusdt", 300, since, until, 100)

	expectContains(t, q, "timestamp >= ?")
	expectContains(t, q, "timestamp <= ?")

	// 4 base + 1 since + 1 until + 1 limit = 7.
	if len(args) != 7 {
		t.Fatalf("expected 7 args, got %d", len(args))
	}

	sinceArg := args[4].(time.Time)
	untilArg := args[5].(time.Time)
	if sinceArg.Unix() != since {
		t.Errorf("since arg: expected unix %d, got %d", since, sinceArg.Unix())
	}
	if untilArg.Unix() != until {
		t.Errorf("until arg: expected unix %d, got %d", until, untilArg.Unix())
	}
	assertQueryArg(t, "limit", args[6], 100)
}

func TestBuildSignalQuery_TimeframeAsUint32(t *testing.T) {
	_, args := clickhouse.BuildSignalQuery("rsi", "binancef", "btcusdt", 300, 0, 0, 50)

	tf, ok := args[3].(uint32)
	if !ok {
		t.Fatalf("timeframe arg should be uint32, got %T", args[3])
	}
	if tf != 300 {
		t.Errorf("expected timeframe 300, got %d", tf)
	}
}

func TestBuildSignalQuery_SelectColumns(t *testing.T) {
	q, _ := clickhouse.BuildSignalQuery("rsi", "binancef", "btcusdt", 60, 0, 0, 50)

	// Verify the 8 columns in the SELECT match the DDL read path.
	expectedCols := []string{
		"type", "source", "symbol", "timeframe",
		"value", "metadata", "final", "timestamp",
	}
	for _, col := range expectedCols {
		expectContains(t, q, col)
	}
}

// ── parseMetadataJSON ───────────────────────────────────────────

func TestParseMetadataJSON_ValidJSON(t *testing.T) {
	result := clickhouse.ParseMetadataJSON(`{"period":"14","avg_gain":"1.5"}`)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result["period"] != "14" {
		t.Errorf("expected period=14, got %q", result["period"])
	}
	if result["avg_gain"] != "1.5" {
		t.Errorf("expected avg_gain=1.5, got %q", result["avg_gain"])
	}
}

func TestParseMetadataJSON_EmptyString(t *testing.T) {
	result := clickhouse.ParseMetadataJSON("")
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestParseMetadataJSON_EmptyObject(t *testing.T) {
	result := clickhouse.ParseMetadataJSON("{}")
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestParseMetadataJSON_InvalidJSON(t *testing.T) {
	result := clickhouse.ParseMetadataJSON("not json")
	if len(result) != 0 {
		t.Errorf("expected empty map on invalid JSON, got %v", result)
	}
}
