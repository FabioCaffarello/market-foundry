package clickhouse_test

import (
	"testing"
	"time"

	"internal/adapters/clickhouse"
)

// -- BuildExecutionQuery ------------------------------------------------------

func TestBuildExecutionQuery_BasicFilters(t *testing.T) {
	q, args := clickhouse.BuildExecutionQuery("paper_order", "derive", "btcusdt", 60, "", "", 0, 0, 50)

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

	assertQueryArg(t, "type", args[0], "paper_order")
	assertQueryArg(t, "source", args[1], "derive")
	assertQueryArg(t, "symbol", args[2], "btcusdt")
	assertQueryArg(t, "timeframe", args[3], uint32(60))
	assertQueryArg(t, "limit", args[4], 50)
}

func TestBuildExecutionQuery_NoTimeFilters(t *testing.T) {
	q, _ := clickhouse.BuildExecutionQuery("paper_order", "derive", "btcusdt", 60, "", "", 0, 0, 50)

	expectNotContains(t, q, "timestamp >=")
	expectNotContains(t, q, "timestamp <=")
}

func TestBuildExecutionQuery_WithSide(t *testing.T) {
	q, args := clickhouse.BuildExecutionQuery("paper_order", "derive", "btcusdt", 60, "buy", "", 0, 0, 50)

	expectContains(t, q, "side = ?")

	// 4 base + 1 side + 1 limit = 6.
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d: %v", len(args), args)
	}

	assertQueryArg(t, "side", args[4], "buy")
	assertQueryArg(t, "limit", args[5], 50)
}

func TestBuildExecutionQuery_WithStatus(t *testing.T) {
	q, args := clickhouse.BuildExecutionQuery("paper_order", "derive", "btcusdt", 60, "", "filled", 0, 0, 50)

	expectContains(t, q, "status = ?")

	// 4 base + 1 status + 1 limit = 6.
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d: %v", len(args), args)
	}

	assertQueryArg(t, "status", args[4], "filled")
	assertQueryArg(t, "limit", args[5], 50)
}

func TestBuildExecutionQuery_WithSideAndStatus(t *testing.T) {
	q, args := clickhouse.BuildExecutionQuery("paper_order", "derive", "btcusdt", 60, "buy", "filled", 0, 0, 50)

	expectContains(t, q, "side = ?")
	expectContains(t, q, "status = ?")

	// 4 base + 1 side + 1 status + 1 limit = 7.
	if len(args) != 7 {
		t.Fatalf("expected 7 args, got %d: %v", len(args), args)
	}

	assertQueryArg(t, "side", args[4], "buy")
	assertQueryArg(t, "status", args[5], "filled")
	assertQueryArg(t, "limit", args[6], 50)
}

func TestBuildExecutionQuery_WithSince(t *testing.T) {
	since := int64(1710500000)
	q, args := clickhouse.BuildExecutionQuery("paper_order", "derive", "btcusdt", 60, "", "", since, 0, 50)

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

func TestBuildExecutionQuery_WithUntil(t *testing.T) {
	until := int64(1710600000)
	q, args := clickhouse.BuildExecutionQuery("paper_order", "derive", "btcusdt", 60, "", "", 0, until, 50)

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

func TestBuildExecutionQuery_WithAllFilters(t *testing.T) {
	since := int64(1710500000)
	until := int64(1710600000)
	q, args := clickhouse.BuildExecutionQuery("paper_order", "derive", "btcusdt", 60, "sell", "rejected", since, until, 100)

	expectContains(t, q, "side = ?")
	expectContains(t, q, "status = ?")
	expectContains(t, q, "timestamp >= ?")
	expectContains(t, q, "timestamp <= ?")

	// 4 base + 1 side + 1 status + 1 since + 1 until + 1 limit = 9.
	if len(args) != 9 {
		t.Fatalf("expected 9 args, got %d", len(args))
	}

	assertQueryArg(t, "side", args[4], "sell")
	assertQueryArg(t, "status", args[5], "rejected")
	assertQueryArg(t, "limit", args[8], 100)
}

func TestBuildExecutionQuery_TimeframeAsUint32(t *testing.T) {
	_, args := clickhouse.BuildExecutionQuery("paper_order", "derive", "btcusdt", 300, "", "", 0, 0, 50)

	tf, ok := args[3].(uint32)
	if !ok {
		t.Fatalf("timeframe arg should be uint32, got %T", args[3])
	}
	if tf != 300 {
		t.Errorf("expected timeframe 300, got %d", tf)
	}
}

func TestBuildExecutionQuery_SelectColumns(t *testing.T) {
	q, _ := clickhouse.BuildExecutionQuery("paper_order", "derive", "btcusdt", 60, "", "", 0, 0, 50)

	// Verify the 19 columns in the SELECT match the DDL read path
	// (H-6.d.2: +base/quote/contract canonical columns).
	expectedCols := []string{
		"type", "source", "symbol",
		"base", "quote", "contract",
		"timeframe",
		"side", "quantity", "filled_quantity", "status",
		"risk", "fills", "parameters", "metadata",
		"exec_correlation_id", "exec_causation_id", "final", "timestamp",
	}
	for _, col := range expectedCols {
		expectContains(t, q, col)
	}
}

// -- ParseRiskInputJSON -------------------------------------------------------

func TestParseRiskInputJSON_ValidStruct(t *testing.T) {
	result := clickhouse.ParseRiskInputJSON(`{"type":"position_exposure","disposition":"approved","confidence":"0.85","timeframe":60}`)
	if result.Type != "position_exposure" {
		t.Errorf("expected type=position_exposure, got %q", result.Type)
	}
	if result.Disposition != "approved" {
		t.Errorf("expected disposition=approved, got %q", result.Disposition)
	}
	if result.Confidence != "0.85" {
		t.Errorf("expected confidence=0.85, got %q", result.Confidence)
	}
	if result.Timeframe != 60 {
		t.Errorf("expected timeframe=60, got %d", result.Timeframe)
	}
}

func TestParseRiskInputJSON_EmptyString(t *testing.T) {
	result := clickhouse.ParseRiskInputJSON("")
	if result.Type != "" || result.Disposition != "" {
		t.Errorf("expected zero struct, got %+v", result)
	}
}

func TestParseRiskInputJSON_EmptyObject(t *testing.T) {
	result := clickhouse.ParseRiskInputJSON("{}")
	if result.Type != "" || result.Disposition != "" {
		t.Errorf("expected zero struct, got %+v", result)
	}
}

func TestParseRiskInputJSON_MalformedJSON(t *testing.T) {
	result := clickhouse.ParseRiskInputJSON("not json")
	if result.Type != "" || result.Disposition != "" {
		t.Errorf("expected zero struct on invalid JSON, got %+v", result)
	}
}

// -- ParseFillsJSON -----------------------------------------------------------

func TestParseFillsJSON_ValidArray(t *testing.T) {
	result := clickhouse.ParseFillsJSON(`[{"price":"67500.00","quantity":"0.001","fee":"0.00","simulated":true,"timestamp":"2026-03-20T10:01:00Z"}]`)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if result[0].Price != "67500.00" {
		t.Errorf("expected price=67500.00, got %q", result[0].Price)
	}
	if result[0].Quantity != "0.001" {
		t.Errorf("expected quantity=0.001, got %q", result[0].Quantity)
	}
	if result[0].Fee != "0.00" {
		t.Errorf("expected fee=0.00, got %q", result[0].Fee)
	}
	if !result[0].Simulated {
		t.Error("expected simulated=true")
	}
}

func TestParseFillsJSON_MultipleFills(t *testing.T) {
	result := clickhouse.ParseFillsJSON(`[{"price":"67500.00","quantity":"0.001","fee":"0.00","simulated":true,"timestamp":"2026-03-20T10:01:00Z"},{"price":"67550.00","quantity":"0.002","fee":"0.01","simulated":false,"timestamp":"2026-03-20T10:02:00Z"}]`)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
}

func TestParseFillsJSON_EmptyString(t *testing.T) {
	result := clickhouse.ParseFillsJSON("")
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

func TestParseFillsJSON_EmptyArray(t *testing.T) {
	result := clickhouse.ParseFillsJSON("[]")
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

func TestParseFillsJSON_EmptyObject(t *testing.T) {
	result := clickhouse.ParseFillsJSON("{}")
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty object, got %v", result)
	}
}

func TestParseFillsJSON_InvalidJSON(t *testing.T) {
	result := clickhouse.ParseFillsJSON("not json")
	if len(result) != 0 {
		t.Errorf("expected empty slice on invalid JSON, got %v", result)
	}
}
