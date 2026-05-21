package clickhouse_test

import (
	"testing"
	"time"

	"internal/adapters/clickhouse"
)

// -- BuildExecutionListQuery -------------------------------------------------

func TestBuildExecutionListQuery_SingleFilter_Type(t *testing.T) {
	q, args, err := clickhouse.BuildExecutionListQuery("paper_order", "", "", 0, "", "", 0, 0, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectContains(t, q, "type = ?")
	expectContains(t, q, "ORDER BY timestamp DESC")
	expectContains(t, q, "LIMIT ?")
	// 1 type + 1 limit = 2.
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
}

func TestBuildExecutionListQuery_SingleFilter_Status(t *testing.T) {
	q, args, err := clickhouse.BuildExecutionListQuery("", "", "", 0, "", "rejected", 0, 0, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectContains(t, q, "status = ?")
	expectNotContains(t, q, "type = ?")
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
	assertQueryArg(t, "status", args[0], "rejected")
}

func TestBuildExecutionListQuery_NoFilter_ReturnsError(t *testing.T) {
	_, _, err := clickhouse.BuildExecutionListQuery("", "", "", 0, "", "", 0, 0, 50)
	if err == nil {
		t.Fatal("expected error when no filters provided")
	}
}

func TestBuildExecutionListQuery_MultipleFilters(t *testing.T) {
	since := int64(1710500000)
	q, args, err := clickhouse.BuildExecutionListQuery("venue_market_order", "derive", "btcusdt", 60, "buy", "filled", since, 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectContains(t, q, "type = ?")
	expectContains(t, q, "source = ?")
	expectContains(t, q, "symbol = ?")
	expectContains(t, q, "timeframe = ?")
	expectContains(t, q, "side = ?")
	expectContains(t, q, "status = ?")
	expectContains(t, q, "timestamp >= ?")
	// 6 filters + 1 since + 1 limit = 8.
	if len(args) != 8 {
		t.Fatalf("expected 8 args, got %d: %v", len(args), args)
	}
}

func TestBuildExecutionListQuery_TimeRangeOnly(t *testing.T) {
	since := int64(1710500000)
	until := int64(1710600000)
	q, args, err := clickhouse.BuildExecutionListQuery("", "", "", 0, "", "", since, until, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectContains(t, q, "timestamp >= ?")
	expectContains(t, q, "timestamp <= ?")
	// 1 since + 1 until + 1 limit = 3.
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}

	sinceArg := args[0].(time.Time)
	if sinceArg.Unix() != since {
		t.Errorf("since arg: expected unix %d, got %d", since, sinceArg.Unix())
	}
}

func TestBuildExecutionListQuery_SelectColumns(t *testing.T) {
	q, _, err := clickhouse.BuildExecutionListQuery("paper_order", "", "", 0, "", "", 0, 0, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

// -- BuildExecutionSummaryQuery ----------------------------------------------

func TestBuildExecutionSummaryQuery_SingleFilter_Source(t *testing.T) {
	q, args, err := clickhouse.BuildExecutionSummaryQuery("derive", "", 0, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectContains(t, q, "source = ?")
	expectContains(t, q, "GROUP BY type, status")
	expectContains(t, q, "ORDER BY cnt DESC")
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d: %v", len(args), args)
	}
}

func TestBuildExecutionSummaryQuery_NoFilter_ReturnsError(t *testing.T) {
	_, _, err := clickhouse.BuildExecutionSummaryQuery("", "", 0, 0, 0)
	if err == nil {
		t.Fatal("expected error when no filters provided")
	}
}

func TestBuildExecutionSummaryQuery_TimeRange(t *testing.T) {
	since := int64(1710500000)
	until := int64(1710600000)
	q, args, err := clickhouse.BuildExecutionSummaryQuery("", "", 0, since, until)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectContains(t, q, "timestamp >= ?")
	expectContains(t, q, "timestamp <= ?")
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
}

func TestBuildExecutionSummaryQuery_AllFilters(t *testing.T) {
	q, args, err := clickhouse.BuildExecutionSummaryQuery("derive", "btcusdt", 60, 1710500000, 1710600000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectContains(t, q, "source = ?")
	expectContains(t, q, "symbol = ?")
	expectContains(t, q, "timeframe = ?")
	expectContains(t, q, "timestamp >= ?")
	expectContains(t, q, "timestamp <= ?")
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d: %v", len(args), args)
	}
}

func TestBuildExecutionSummaryQuery_SelectColumns(t *testing.T) {
	q, _, err := clickhouse.BuildExecutionSummaryQuery("derive", "", 0, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectContains(t, q, "type")
	expectContains(t, q, "status")
	expectContains(t, q, "count()")
	expectContains(t, q, "max(timestamp)")
}
