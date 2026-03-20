package clickhouse_test

import (
	"testing"
	"time"

	"internal/adapters/clickhouse"
)

// -- BuildStrategyQuery -------------------------------------------------------

func TestBuildStrategyQuery_BasicFilters(t *testing.T) {
	q, args := clickhouse.BuildStrategyQuery("mean_reversion_entry", "binancef", "btcusdt", 60, "", 0, 0, 50)

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

	assertQueryArg(t, "type", args[0], "mean_reversion_entry")
	assertQueryArg(t, "source", args[1], "binancef")
	assertQueryArg(t, "symbol", args[2], "btcusdt")
	assertQueryArg(t, "timeframe", args[3], uint32(60))
	assertQueryArg(t, "limit", args[4], 50)
}

func TestBuildStrategyQuery_NoTimeFilters(t *testing.T) {
	q, _ := clickhouse.BuildStrategyQuery("mean_reversion_entry", "binancef", "btcusdt", 60, "", 0, 0, 50)

	expectNotContains(t, q, "timestamp >=")
	expectNotContains(t, q, "timestamp <=")
}

func TestBuildStrategyQuery_WithDirection(t *testing.T) {
	q, args := clickhouse.BuildStrategyQuery("mean_reversion_entry", "binancef", "btcusdt", 60, "long", 0, 0, 50)

	expectContains(t, q, "direction = ?")

	// 4 base + 1 direction + 1 limit = 6.
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d: %v", len(args), args)
	}

	assertQueryArg(t, "direction", args[4], "long")
	assertQueryArg(t, "limit", args[5], 50)
}

func TestBuildStrategyQuery_WithSince(t *testing.T) {
	since := int64(1710500000)
	q, args := clickhouse.BuildStrategyQuery("mean_reversion_entry", "binancef", "btcusdt", 60, "", since, 0, 50)

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

func TestBuildStrategyQuery_WithUntil(t *testing.T) {
	until := int64(1710600000)
	q, args := clickhouse.BuildStrategyQuery("mean_reversion_entry", "binancef", "btcusdt", 60, "", 0, until, 50)

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

func TestBuildStrategyQuery_WithDirectionAndTimeRange(t *testing.T) {
	since := int64(1710500000)
	until := int64(1710600000)
	q, args := clickhouse.BuildStrategyQuery("mean_reversion_entry", "binancef", "btcusdt", 60, "short", since, until, 100)

	expectContains(t, q, "direction = ?")
	expectContains(t, q, "timestamp >= ?")
	expectContains(t, q, "timestamp <= ?")

	// 4 base + 1 direction + 1 since + 1 until + 1 limit = 8.
	if len(args) != 8 {
		t.Fatalf("expected 8 args, got %d", len(args))
	}

	assertQueryArg(t, "direction", args[4], "short")
	assertQueryArg(t, "limit", args[7], 100)
}

func TestBuildStrategyQuery_TimeframeAsUint32(t *testing.T) {
	_, args := clickhouse.BuildStrategyQuery("mean_reversion_entry", "binancef", "btcusdt", 300, "", 0, 0, 50)

	tf, ok := args[3].(uint32)
	if !ok {
		t.Fatalf("timeframe arg should be uint32, got %T", args[3])
	}
	if tf != 300 {
		t.Errorf("expected timeframe 300, got %d", tf)
	}
}

func TestBuildStrategyQuery_SelectColumns(t *testing.T) {
	q, _ := clickhouse.BuildStrategyQuery("mean_reversion_entry", "binancef", "btcusdt", 60, "", 0, 0, 50)

	// Verify the 11 columns in the SELECT match the DDL read path.
	expectedCols := []string{
		"type", "source", "symbol", "timeframe",
		"direction", "confidence", "decisions", "parameters", "metadata",
		"final", "timestamp",
	}
	for _, col := range expectedCols {
		expectContains(t, q, col)
	}
}

// -- ParseDecisionInputsJSON -------------------------------------------------

func TestParseDecisionInputsJSON_ValidArray(t *testing.T) {
	result := clickhouse.ParseDecisionInputsJSON(`[{"type":"rsi_oversold","outcome":"triggered","confidence":"0.85","timeframe":60}]`)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if result[0].Type != "rsi_oversold" {
		t.Errorf("expected type=rsi_oversold, got %q", result[0].Type)
	}
	if result[0].Outcome != "triggered" {
		t.Errorf("expected outcome=triggered, got %q", result[0].Outcome)
	}
	if result[0].Confidence != "0.85" {
		t.Errorf("expected confidence=0.85, got %q", result[0].Confidence)
	}
	if result[0].Timeframe != 60 {
		t.Errorf("expected timeframe=60, got %d", result[0].Timeframe)
	}
}

func TestParseDecisionInputsJSON_MultipleEntries(t *testing.T) {
	result := clickhouse.ParseDecisionInputsJSON(`[{"type":"rsi_oversold","outcome":"triggered","confidence":"0.85","timeframe":60},{"type":"ema_crossover","outcome":"not_triggered","confidence":"0.40","timeframe":300}]`)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
}

func TestParseDecisionInputsJSON_EmptyString(t *testing.T) {
	result := clickhouse.ParseDecisionInputsJSON("")
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

func TestParseDecisionInputsJSON_EmptyArray(t *testing.T) {
	result := clickhouse.ParseDecisionInputsJSON("[]")
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

func TestParseDecisionInputsJSON_EmptyObject(t *testing.T) {
	result := clickhouse.ParseDecisionInputsJSON("{}")
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty object, got %v", result)
	}
}

func TestParseDecisionInputsJSON_InvalidJSON(t *testing.T) {
	result := clickhouse.ParseDecisionInputsJSON("not json")
	if len(result) != 0 {
		t.Errorf("expected empty slice on invalid JSON, got %v", result)
	}
}
