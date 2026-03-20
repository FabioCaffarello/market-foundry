package clickhouse_test

import (
	"testing"
	"time"

	"internal/adapters/clickhouse"
)

// -- BuildDecisionQuery -------------------------------------------------------

func TestBuildDecisionQuery_BasicFilters(t *testing.T) {
	q, args := clickhouse.BuildDecisionQuery("rsi_oversold", "binancef", "btcusdt", 60, "", 0, 0, 50)

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

	assertQueryArg(t, "type", args[0], "rsi_oversold")
	assertQueryArg(t, "source", args[1], "binancef")
	assertQueryArg(t, "symbol", args[2], "btcusdt")
	assertQueryArg(t, "timeframe", args[3], uint32(60))
	assertQueryArg(t, "limit", args[4], 50)
}

func TestBuildDecisionQuery_NoTimeFilters(t *testing.T) {
	q, _ := clickhouse.BuildDecisionQuery("rsi_oversold", "binancef", "btcusdt", 60, "", 0, 0, 50)

	expectNotContains(t, q, "timestamp >=")
	expectNotContains(t, q, "timestamp <=")
}

func TestBuildDecisionQuery_WithOutcome(t *testing.T) {
	q, args := clickhouse.BuildDecisionQuery("rsi_oversold", "binancef", "btcusdt", 60, "triggered", 0, 0, 50)

	expectContains(t, q, "outcome = ?")

	// 4 base + 1 outcome + 1 limit = 6.
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d: %v", len(args), args)
	}

	assertQueryArg(t, "outcome", args[4], "triggered")
	assertQueryArg(t, "limit", args[5], 50)
}

func TestBuildDecisionQuery_WithSince(t *testing.T) {
	since := int64(1710500000)
	q, args := clickhouse.BuildDecisionQuery("rsi_oversold", "binancef", "btcusdt", 60, "", since, 0, 50)

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

func TestBuildDecisionQuery_WithUntil(t *testing.T) {
	until := int64(1710600000)
	q, args := clickhouse.BuildDecisionQuery("rsi_oversold", "binancef", "btcusdt", 60, "", 0, until, 50)

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

func TestBuildDecisionQuery_WithSinceAndUntil(t *testing.T) {
	since := int64(1710500000)
	until := int64(1710600000)
	q, args := clickhouse.BuildDecisionQuery("rsi_oversold", "binancef", "btcusdt", 300, "", since, until, 100)

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

func TestBuildDecisionQuery_WithOutcomeAndTimeRange(t *testing.T) {
	since := int64(1710500000)
	until := int64(1710600000)
	q, args := clickhouse.BuildDecisionQuery("rsi_oversold", "binancef", "btcusdt", 60, "triggered", since, until, 50)

	expectContains(t, q, "outcome = ?")
	expectContains(t, q, "timestamp >= ?")
	expectContains(t, q, "timestamp <= ?")

	// 4 base + 1 outcome + 1 since + 1 until + 1 limit = 8.
	if len(args) != 8 {
		t.Fatalf("expected 8 args, got %d", len(args))
	}
}

func TestBuildDecisionQuery_TimeframeAsUint32(t *testing.T) {
	_, args := clickhouse.BuildDecisionQuery("rsi_oversold", "binancef", "btcusdt", 300, "", 0, 0, 50)

	tf, ok := args[3].(uint32)
	if !ok {
		t.Fatalf("timeframe arg should be uint32, got %T", args[3])
	}
	if tf != 300 {
		t.Errorf("expected timeframe 300, got %d", tf)
	}
}

func TestBuildDecisionQuery_SelectColumns(t *testing.T) {
	q, _ := clickhouse.BuildDecisionQuery("rsi_oversold", "binancef", "btcusdt", 60, "", 0, 0, 50)

	// Verify the 10 columns in the SELECT match the DDL read path.
	expectedCols := []string{
		"type", "source", "symbol", "timeframe",
		"outcome", "confidence", "signals", "metadata",
		"final", "timestamp",
	}
	for _, col := range expectedCols {
		expectContains(t, q, col)
	}
}

// -- ParseSignalInputsJSON ---------------------------------------------------

func TestParseSignalInputsJSON_ValidArray(t *testing.T) {
	result := clickhouse.ParseSignalInputsJSON(`[{"type":"rsi","value":"28.5","timeframe":60}]`)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if result[0].Type != "rsi" {
		t.Errorf("expected type=rsi, got %q", result[0].Type)
	}
	if result[0].Value != "28.5" {
		t.Errorf("expected value=28.5, got %q", result[0].Value)
	}
	if result[0].Timeframe != 60 {
		t.Errorf("expected timeframe=60, got %d", result[0].Timeframe)
	}
}

func TestParseSignalInputsJSON_MultipleEntries(t *testing.T) {
	result := clickhouse.ParseSignalInputsJSON(`[{"type":"rsi","value":"28.5","timeframe":60},{"type":"ema","value":"100.5","timeframe":300}]`)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
}

func TestParseSignalInputsJSON_EmptyString(t *testing.T) {
	result := clickhouse.ParseSignalInputsJSON("")
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

func TestParseSignalInputsJSON_EmptyArray(t *testing.T) {
	result := clickhouse.ParseSignalInputsJSON("[]")
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

func TestParseSignalInputsJSON_EmptyObject(t *testing.T) {
	result := clickhouse.ParseSignalInputsJSON("{}")
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty object, got %v", result)
	}
}

func TestParseSignalInputsJSON_InvalidJSON(t *testing.T) {
	result := clickhouse.ParseSignalInputsJSON("not json")
	if len(result) != 0 {
		t.Errorf("expected empty slice on invalid JSON, got %v", result)
	}
}
