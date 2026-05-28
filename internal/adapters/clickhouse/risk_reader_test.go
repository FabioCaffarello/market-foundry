package clickhouse_test

import (
	"testing"
	"time"

	"internal/adapters/clickhouse"
)

// -- BuildRiskQuery -----------------------------------------------------------

func TestBuildRiskQuery_BasicFilters(t *testing.T) {
	q, args := clickhouse.BuildRiskQuery("position_exposure", "binancef", "btcusdt", 60, "", 0, 0, 50)

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

	assertQueryArg(t, "type", args[0], "position_exposure")
	assertQueryArg(t, "source", args[1], "binancef")
	assertQueryArg(t, "symbol", args[2], "btcusdt")
	assertQueryArg(t, "timeframe", args[3], uint32(60))
	assertQueryArg(t, "limit", args[4], 50)
}

func TestBuildRiskQuery_NoTimeFilters(t *testing.T) {
	q, _ := clickhouse.BuildRiskQuery("position_exposure", "binancef", "btcusdt", 60, "", 0, 0, 50)

	expectNotContains(t, q, "timestamp >=")
	expectNotContains(t, q, "timestamp <=")
}

func TestBuildRiskQuery_WithDisposition(t *testing.T) {
	q, args := clickhouse.BuildRiskQuery("position_exposure", "binancef", "btcusdt", 60, "approved", 0, 0, 50)

	expectContains(t, q, "disposition = ?")

	// 4 base + 1 disposition + 1 limit = 6.
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d: %v", len(args), args)
	}

	assertQueryArg(t, "disposition", args[4], "approved")
	assertQueryArg(t, "limit", args[5], 50)
}

func TestBuildRiskQuery_WithSince(t *testing.T) {
	since := int64(1710500000)
	q, args := clickhouse.BuildRiskQuery("position_exposure", "binancef", "btcusdt", 60, "", since, 0, 50)

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

func TestBuildRiskQuery_WithUntil(t *testing.T) {
	until := int64(1710600000)
	q, args := clickhouse.BuildRiskQuery("position_exposure", "binancef", "btcusdt", 60, "", 0, until, 50)

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

func TestBuildRiskQuery_WithDispositionAndTimeRange(t *testing.T) {
	since := int64(1710500000)
	until := int64(1710600000)
	q, args := clickhouse.BuildRiskQuery("position_exposure", "binancef", "btcusdt", 60, "rejected", since, until, 100)

	expectContains(t, q, "disposition = ?")
	expectContains(t, q, "timestamp >= ?")
	expectContains(t, q, "timestamp <= ?")

	// 4 base + 1 disposition + 1 since + 1 until + 1 limit = 8.
	if len(args) != 8 {
		t.Fatalf("expected 8 args, got %d", len(args))
	}

	assertQueryArg(t, "disposition", args[4], "rejected")
	assertQueryArg(t, "limit", args[7], 100)
}

func TestBuildRiskQuery_TimeframeAsUint32(t *testing.T) {
	_, args := clickhouse.BuildRiskQuery("position_exposure", "binancef", "btcusdt", 300, "", 0, 0, 50)

	tf, ok := args[3].(uint32)
	if !ok {
		t.Fatalf("timeframe arg should be uint32, got %T", args[3])
	}
	if tf != 300 {
		t.Errorf("expected timeframe 300, got %d", tf)
	}
}

func TestBuildRiskQuery_SelectColumns(t *testing.T) {
	q, _ := clickhouse.BuildRiskQuery("position_exposure", "binancef", "btcusdt", 60, "", 0, 0, 50)

	// Verify the 16 columns in the SELECT match the DDL read path
	// (H-6.d.2: +base/quote/contract canonical columns).
	expectedCols := []string{
		"type", "source", "symbol",
		"base", "quote", "contract",
		"timeframe",
		"disposition", "confidence", "strategies", "constraints", "rationale",
		"parameters", "metadata", "final", "timestamp",
	}
	for _, col := range expectedCols {
		expectContains(t, q, col)
	}
}

// -- ParseStrategyInputsJSON --------------------------------------------------

func TestParseStrategyInputsJSON_ValidArray(t *testing.T) {
	result := clickhouse.ParseStrategyInputsJSON(`[{"type":"mean_reversion_entry","direction":"long","confidence":"0.85","timeframe":60}]`)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if result[0].Type != "mean_reversion_entry" {
		t.Errorf("expected type=mean_reversion_entry, got %q", result[0].Type)
	}
	if result[0].Direction != "long" {
		t.Errorf("expected direction=long, got %q", result[0].Direction)
	}
	if result[0].Confidence != "0.85" {
		t.Errorf("expected confidence=0.85, got %q", result[0].Confidence)
	}
	if result[0].Timeframe != 60 {
		t.Errorf("expected timeframe=60, got %d", result[0].Timeframe)
	}
}

func TestParseStrategyInputsJSON_MultipleEntries(t *testing.T) {
	result := clickhouse.ParseStrategyInputsJSON(`[{"type":"mean_reversion_entry","direction":"long","confidence":"0.85","timeframe":60},{"type":"momentum_entry","direction":"short","confidence":"0.40","timeframe":300}]`)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
}

func TestParseStrategyInputsJSON_EmptyString(t *testing.T) {
	result := clickhouse.ParseStrategyInputsJSON("")
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

func TestParseStrategyInputsJSON_EmptyArray(t *testing.T) {
	result := clickhouse.ParseStrategyInputsJSON("[]")
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

func TestParseStrategyInputsJSON_EmptyObject(t *testing.T) {
	result := clickhouse.ParseStrategyInputsJSON("{}")
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty object, got %v", result)
	}
}

func TestParseStrategyInputsJSON_InvalidJSON(t *testing.T) {
	result := clickhouse.ParseStrategyInputsJSON("not json")
	if len(result) != 0 {
		t.Errorf("expected empty slice on invalid JSON, got %v", result)
	}
}

func TestParseStrategyInputsJSON_WithDecisionContext(t *testing.T) {
	result := clickhouse.ParseStrategyInputsJSON(`[{"type":"mean_reversion_entry","direction":"long","confidence":"0.85","timeframe":60,"decision_severity":"high","decision_rationale":"RSI 10.00 below threshold"}]`)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if result[0].DecisionSeverity != "high" {
		t.Errorf("expected decision_severity=high, got %q", result[0].DecisionSeverity)
	}
	if result[0].DecisionRationale != "RSI 10.00 below threshold" {
		t.Errorf("expected decision_rationale, got %q", result[0].DecisionRationale)
	}
}

func TestParseStrategyInputsJSON_WithoutDecisionContext_BackwardCompatible(t *testing.T) {
	result := clickhouse.ParseStrategyInputsJSON(`[{"type":"mean_reversion_entry","direction":"long","confidence":"0.85","timeframe":60}]`)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if result[0].DecisionSeverity != "" {
		t.Errorf("expected empty decision_severity for legacy data, got %q", result[0].DecisionSeverity)
	}
	if result[0].DecisionRationale != "" {
		t.Errorf("expected empty decision_rationale for legacy data, got %q", result[0].DecisionRationale)
	}
}

// -- ParseConstraintsJSON -----------------------------------------------------

func TestParseConstraintsJSON_ValidStruct(t *testing.T) {
	result := clickhouse.ParseConstraintsJSON(`{"max_position_size":"0.1","max_exposure":"1000.00","stop_distance":"50.00"}`)
	if result.MaxPositionSize != "0.1" {
		t.Errorf("expected max_position_size=0.1, got %q", result.MaxPositionSize)
	}
	if result.MaxExposure != "1000.00" {
		t.Errorf("expected max_exposure=1000.00, got %q", result.MaxExposure)
	}
	if result.StopDistance != "50.00" {
		t.Errorf("expected stop_distance=50.00, got %q", result.StopDistance)
	}
}

func TestParseConstraintsJSON_EmptyString(t *testing.T) {
	result := clickhouse.ParseConstraintsJSON("")
	if result.MaxPositionSize != "" || result.MaxExposure != "" || result.StopDistance != "" {
		t.Errorf("expected zero struct, got %+v", result)
	}
}

func TestParseConstraintsJSON_EmptyObject(t *testing.T) {
	result := clickhouse.ParseConstraintsJSON("{}")
	if result.MaxPositionSize != "" || result.MaxExposure != "" || result.StopDistance != "" {
		t.Errorf("expected zero struct, got %+v", result)
	}
}

func TestParseConstraintsJSON_MalformedJSON(t *testing.T) {
	result := clickhouse.ParseConstraintsJSON("not json")
	if result.MaxPositionSize != "" || result.MaxExposure != "" || result.StopDistance != "" {
		t.Errorf("expected zero struct on invalid JSON, got %+v", result)
	}
}
