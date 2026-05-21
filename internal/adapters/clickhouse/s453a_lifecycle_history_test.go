package clickhouse_test

import (
	"testing"
	"time"

	"internal/adapters/clickhouse"
)

// -- BuildLifecycleHistoryQuery -----------------------------------------------

func TestBuildLifecycleHistoryQuery_BasicFilters(t *testing.T) {
	q, args := clickhouse.BuildLifecycleHistoryQuery("derive", "btcusdt", 60, "", "", 0, 0, 50)

	// Must NOT filter by type — that's the key difference from BuildExecutionQuery.
	expectNotContains(t, q, "type = ?")

	expectContains(t, q, "source = ?")
	expectContains(t, q, "symbol = ?")
	expectContains(t, q, "timeframe = ?")
	expectContains(t, q, "ORDER BY timestamp DESC")
	expectContains(t, q, "LIMIT ?")

	// 3 base args (source, symbol, timeframe) + 1 limit.
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}

	assertQueryArg(t, "source", args[0], "derive")
	assertQueryArg(t, "symbol", args[1], "btcusdt")
	assertQueryArg(t, "timeframe", args[2], uint32(60))
	assertQueryArg(t, "limit", args[3], 50)
}

func TestBuildLifecycleHistoryQuery_NoTypeFilter(t *testing.T) {
	q, _ := clickhouse.BuildLifecycleHistoryQuery("derive", "btcusdt", 60, "", "", 0, 0, 50)

	// The lifecycle query must return all event types (paper_order, venue_market_order, venue_rejection).
	expectNotContains(t, q, "type = ?")
	// But the type column should be in the SELECT for identification.
	expectContains(t, q, "type")
}

func TestBuildLifecycleHistoryQuery_WithSide(t *testing.T) {
	q, args := clickhouse.BuildLifecycleHistoryQuery("derive", "btcusdt", 60, "buy", "", 0, 0, 50)

	expectContains(t, q, "side = ?")

	// 3 base + 1 side + 1 limit = 5.
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d: %v", len(args), args)
	}

	assertQueryArg(t, "side", args[3], "buy")
	assertQueryArg(t, "limit", args[4], 50)
}

func TestBuildLifecycleHistoryQuery_WithStatus(t *testing.T) {
	q, args := clickhouse.BuildLifecycleHistoryQuery("derive", "btcusdt", 60, "", "filled", 0, 0, 50)

	expectContains(t, q, "status = ?")

	// 3 base + 1 status + 1 limit = 5.
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d: %v", len(args), args)
	}

	assertQueryArg(t, "status", args[3], "filled")
	assertQueryArg(t, "limit", args[4], 50)
}

func TestBuildLifecycleHistoryQuery_WithSince(t *testing.T) {
	since := int64(1710500000)
	q, args := clickhouse.BuildLifecycleHistoryQuery("derive", "btcusdt", 60, "", "", since, 0, 50)

	expectContains(t, q, "timestamp >= ?")
	expectNotContains(t, q, "timestamp <= ?")

	// 3 base + 1 since + 1 limit = 5.
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d", len(args))
	}

	sinceArg := args[3].(time.Time)
	if sinceArg.Unix() != since {
		t.Errorf("since arg: expected unix %d, got %d", since, sinceArg.Unix())
	}
}

func TestBuildLifecycleHistoryQuery_WithUntil(t *testing.T) {
	until := int64(1710600000)
	q, args := clickhouse.BuildLifecycleHistoryQuery("derive", "btcusdt", 60, "", "", 0, until, 50)

	expectNotContains(t, q, "timestamp >= ?")
	expectContains(t, q, "timestamp <= ?")

	// 3 base + 1 until + 1 limit = 5.
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d", len(args))
	}

	untilArg := args[3].(time.Time)
	if untilArg.Unix() != until {
		t.Errorf("until arg: expected unix %d, got %d", until, untilArg.Unix())
	}
}

func TestBuildLifecycleHistoryQuery_WithAllFilters(t *testing.T) {
	since := int64(1710500000)
	until := int64(1710600000)
	q, args := clickhouse.BuildLifecycleHistoryQuery("derive", "btcusdt", 60, "sell", "rejected", since, until, 100)

	expectContains(t, q, "side = ?")
	expectContains(t, q, "status = ?")
	expectContains(t, q, "timestamp >= ?")
	expectContains(t, q, "timestamp <= ?")
	expectNotContains(t, q, "type = ?")

	// 3 base + 1 side + 1 status + 1 since + 1 until + 1 limit = 8.
	if len(args) != 8 {
		t.Fatalf("expected 8 args, got %d", len(args))
	}

	assertQueryArg(t, "side", args[3], "sell")
	assertQueryArg(t, "status", args[4], "rejected")
	assertQueryArg(t, "limit", args[7], 100)
}

func TestBuildLifecycleHistoryQuery_SelectColumns(t *testing.T) {
	q, _ := clickhouse.BuildLifecycleHistoryQuery("derive", "btcusdt", 60, "", "", 0, 0, 50)

	expectedCols := []string{
		"type", "source", "symbol", "timeframe",
		"side", "quantity", "filled_quantity", "status",
		"risk", "fills", "parameters", "metadata",
		"exec_correlation_id", "exec_causation_id", "final", "timestamp",
	}
	for _, col := range expectedCols {
		expectContains(t, q, col)
	}
}

func TestBuildLifecycleHistoryQuery_TimeframeAsUint32(t *testing.T) {
	_, args := clickhouse.BuildLifecycleHistoryQuery("derive", "btcusdt", 300, "", "", 0, 0, 50)

	tf, ok := args[2].(uint32)
	if !ok {
		t.Fatalf("timeframe arg should be uint32, got %T", args[2])
	}
	if tf != 300 {
		t.Errorf("expected timeframe 300, got %d", tf)
	}
}
