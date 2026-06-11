package analyticalclient_test

import (
	"internal/domain/instrument"

	"context"
	"errors"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/execution"
)

// -- GetExecutionListUseCase -------------------------------------------------

type stubExecutionListReader struct {
	intents []execution.ExecutionIntent
	err     error
}

func (s *stubExecutionListReader) QueryExecutionList(_ context.Context, _, _ string, _ instrument.CanonicalInstrument, _ int, _, _ string, _, _ int64, _ int) ([]execution.ExecutionIntent, error) {
	return s.intents, s.err
}

func TestGetExecutionListUseCase_HappyPath(t *testing.T) {
	now := time.Now().UTC()
	reader := &stubExecutionListReader{
		intents: []execution.ExecutionIntent{
			{Type: "paper_order", Source: "derive", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60, Side: "buy", Status: "submitted", Final: true, Timestamp: now},
		},
	}

	uc := analyticalclient.NewGetExecutionListUseCase(reader, nil)
	reply, prob := uc.Execute(context.Background(), analyticalclient.ExecutionListQuery{
		Status: "submitted",
	})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if reply.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", reply.Source)
	}
	if len(reply.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(reply.Entries))
	}
	if reply.Entries[0].Type != "paper_order" {
		t.Errorf("expected type=paper_order, got %q", reply.Entries[0].Type)
	}
	if reply.Meta.RowCount != 1 {
		t.Errorf("expected row_count=1, got %d", reply.Meta.RowCount)
	}
}

func TestGetExecutionListUseCase_NoFilter_ReturnsError(t *testing.T) {
	uc := analyticalclient.NewGetExecutionListUseCase(&stubExecutionListReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionListQuery{})
	if prob == nil {
		t.Fatal("expected problem when no filters provided")
	}
}

func TestGetExecutionListUseCase_ReaderError(t *testing.T) {
	reader := &stubExecutionListReader{err: errors.New("connection refused")}
	uc := analyticalclient.NewGetExecutionListUseCase(reader, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionListQuery{
		Source: "derive",
	})
	if prob == nil {
		t.Fatal("expected problem on reader error")
	}
}

func TestGetExecutionListUseCase_InvalidTimeRange(t *testing.T) {
	uc := analyticalclient.NewGetExecutionListUseCase(&stubExecutionListReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionListQuery{
		Source: "derive",
		Since:  2000,
		Until:  1000,
	})
	if prob == nil {
		t.Fatal("expected problem when since > until")
	}
}

func TestGetExecutionListUseCase_LimitClamping(t *testing.T) {
	reader := &stubExecutionListReader{intents: nil}
	uc := analyticalclient.NewGetExecutionListUseCase(reader, nil)

	// Zero limit should not fail.
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionListQuery{
		Source: "derive",
		Limit:  0,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	// Over-limit should not fail.
	_, prob = uc.Execute(context.Background(), analyticalclient.ExecutionListQuery{
		Source: "derive",
		Limit:  9999,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
}

func TestGetExecutionListUseCase_NilUseCase(t *testing.T) {
	var uc *analyticalclient.GetExecutionListUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionListQuery{
		Source: "derive",
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
}

// -- GetExecutionSummaryUseCase ----------------------------------------------

type stubExecutionSummaryReader struct {
	rows []analyticalclient.ExecutionSummaryRawRow
	err  error
}

func (s *stubExecutionSummaryReader) QueryExecutionSummary(_ context.Context, _ string, _ instrument.CanonicalInstrument, _ int, _, _ int64) ([]analyticalclient.ExecutionSummaryRawRow, error) {
	return s.rows, s.err
}

func TestGetExecutionSummaryUseCase_HappyPath(t *testing.T) {
	now := time.Now().UTC()
	reader := &stubExecutionSummaryReader{
		rows: []analyticalclient.ExecutionSummaryRawRow{
			{Type: "paper_order", Status: "submitted", Count: 10, LatestAt: now},
			{Type: "venue_market_order", Status: "filled", Count: 5, LatestAt: now},
		},
	}

	uc := analyticalclient.NewGetExecutionSummaryUseCase(reader, nil)
	reply, prob := uc.Execute(context.Background(), analyticalclient.ExecutionSummaryQuery{
		Source: "derive",
	})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if reply.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", reply.Source)
	}
	if len(reply.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(reply.Entries))
	}
	if reply.Entries[0].Count != 10 {
		t.Errorf("expected count=10, got %d", reply.Entries[0].Count)
	}
}

func TestGetExecutionSummaryUseCase_NoFilter_ReturnsError(t *testing.T) {
	uc := analyticalclient.NewGetExecutionSummaryUseCase(&stubExecutionSummaryReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionSummaryQuery{})
	if prob == nil {
		t.Fatal("expected problem when no filters provided")
	}
}

func TestGetExecutionSummaryUseCase_ReaderError(t *testing.T) {
	reader := &stubExecutionSummaryReader{err: errors.New("connection refused")}
	uc := analyticalclient.NewGetExecutionSummaryUseCase(reader, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionSummaryQuery{
		Source: "derive",
	})
	if prob == nil {
		t.Fatal("expected problem on reader error")
	}
}

func TestGetExecutionSummaryUseCase_InvalidTimeRange(t *testing.T) {
	uc := analyticalclient.NewGetExecutionSummaryUseCase(&stubExecutionSummaryReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionSummaryQuery{
		Source: "derive",
		Since:  2000,
		Until:  1000,
	})
	if prob == nil {
		t.Fatal("expected problem when since > until")
	}
}

func TestGetExecutionSummaryUseCase_NilUseCase(t *testing.T) {
	var uc *analyticalclient.GetExecutionSummaryUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionSummaryQuery{
		Source: "derive",
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
}

func TestGetExecutionSummaryUseCase_TimestampFormat(t *testing.T) {
	ts := time.Date(2026, 3, 24, 15, 30, 0, 0, time.UTC)
	reader := &stubExecutionSummaryReader{
		rows: []analyticalclient.ExecutionSummaryRawRow{
			{Type: "paper_order", Status: "submitted", Count: 1, LatestAt: ts},
		},
	}

	uc := analyticalclient.NewGetExecutionSummaryUseCase(reader, nil)
	reply, prob := uc.Execute(context.Background(), analyticalclient.ExecutionSummaryQuery{
		Source: "derive",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if reply.Entries[0].LatestAt != "2026-03-24T15:30:00Z" {
		t.Errorf("expected timestamp=2026-03-24T15:30:00Z, got %q", reply.Entries[0].LatestAt)
	}
}
