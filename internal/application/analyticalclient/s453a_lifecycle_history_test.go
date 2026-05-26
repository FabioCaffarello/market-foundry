package analyticalclient_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/execution"
)

// stubLifecycleReader is a test double for LifecycleHistoryReader.
type stubLifecycleReader struct {
	intents []execution.ExecutionIntent
	err     error
}

func (s *stubLifecycleReader) QueryLifecycleHistory(_ context.Context, _, _ string, _ int, _, _ string, _, _ int64, _ int) ([]execution.ExecutionIntent, error) {
	return s.intents, s.err
}

func TestGetLifecycleHistoryUseCase_HappyPath(t *testing.T) {
	now := time.Now().UTC()
	reader := &stubLifecycleReader{
		intents: []execution.ExecutionIntent{
			{Type: "venue_market_order", Source: "derive", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60, Side: "buy", Status: "filled", Final: true, Timestamp: now},
			{Type: "paper_order", Source: "derive", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60, Side: "buy", Status: "submitted", Final: true, Timestamp: now.Add(-time.Second)},
		},
	}

	uc := analyticalclient.NewGetLifecycleHistoryUseCase(reader, nil)
	reply, prob := uc.Execute(context.Background(), analyticalclient.LifecycleHistoryQuery{
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
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
	if reply.Entries[0].Type != "venue_market_order" {
		t.Errorf("expected first entry type=venue_market_order, got %q", reply.Entries[0].Type)
	}
	if reply.Entries[1].Type != "paper_order" {
		t.Errorf("expected second entry type=paper_order, got %q", reply.Entries[1].Type)
	}
	if reply.Meta.RowCount != 2 {
		t.Errorf("expected row_count=2, got %d", reply.Meta.RowCount)
	}
}

func TestGetLifecycleHistoryUseCase_EmptyResult(t *testing.T) {
	reader := &stubLifecycleReader{intents: nil}

	uc := analyticalclient.NewGetLifecycleHistoryUseCase(reader, nil)
	reply, prob := uc.Execute(context.Background(), analyticalclient.LifecycleHistoryQuery{
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(reply.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(reply.Entries))
	}
	if reply.Meta.RowCount != 0 {
		t.Errorf("expected row_count=0, got %d", reply.Meta.RowCount)
	}
}

func TestGetLifecycleHistoryUseCase_ReaderError(t *testing.T) {
	reader := &stubLifecycleReader{err: errors.New("connection refused")}

	uc := analyticalclient.NewGetLifecycleHistoryUseCase(reader, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.LifecycleHistoryQuery{
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})

	if prob == nil {
		t.Fatal("expected problem on reader error")
	}
}

func TestGetLifecycleHistoryUseCase_MissingSource(t *testing.T) {
	uc := analyticalclient.NewGetLifecycleHistoryUseCase(&stubLifecycleReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.LifecycleHistoryQuery{
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing source")
	}
}

func TestGetLifecycleHistoryUseCase_MissingSymbol(t *testing.T) {
	uc := analyticalclient.NewGetLifecycleHistoryUseCase(&stubLifecycleReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.LifecycleHistoryQuery{
		Source:    "derive",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing symbol")
	}
}

func TestGetLifecycleHistoryUseCase_InvalidTimeframe(t *testing.T) {
	uc := analyticalclient.NewGetLifecycleHistoryUseCase(&stubLifecycleReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.LifecycleHistoryQuery{
		Source: "derive",
		Symbol: "btcusdt",
	})
	if prob == nil {
		t.Fatal("expected problem for zero timeframe")
	}
}

func TestGetLifecycleHistoryUseCase_InvalidTimeRange(t *testing.T) {
	uc := analyticalclient.NewGetLifecycleHistoryUseCase(&stubLifecycleReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.LifecycleHistoryQuery{
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Since:     2000,
		Until:     1000,
	})
	if prob == nil {
		t.Fatal("expected problem when since > until")
	}
}

func TestGetLifecycleHistoryUseCase_LimitClamping(t *testing.T) {
	reader := &stubLifecycleReader{intents: nil}

	uc := analyticalclient.NewGetLifecycleHistoryUseCase(reader, nil)

	// Zero limit should default to 50 (not fail).
	reply, prob := uc.Execute(context.Background(), analyticalclient.LifecycleHistoryQuery{
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Limit:     0,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	_ = reply

	// Over-limit should be clamped to 500 (not fail).
	_, prob = uc.Execute(context.Background(), analyticalclient.LifecycleHistoryQuery{
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Limit:     9999,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
}

func TestGetLifecycleHistoryUseCase_NilUseCase(t *testing.T) {
	var uc *analyticalclient.GetLifecycleHistoryUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.LifecycleHistoryQuery{
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
}

func TestGetLifecycleHistoryUseCase_EntryTimestampFormat(t *testing.T) {
	now := time.Date(2026, 3, 24, 15, 30, 0, 0, time.UTC)
	reader := &stubLifecycleReader{
		intents: []execution.ExecutionIntent{
			{Type: "paper_order", Source: "derive", Instrument: instrumentFromVenue("btcusdt"), Timeframe: 60, Status: "submitted", Timestamp: now},
		},
	}

	uc := analyticalclient.NewGetLifecycleHistoryUseCase(reader, nil)
	reply, prob := uc.Execute(context.Background(), analyticalclient.LifecycleHistoryQuery{
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	expected := "2026-03-24T15:30:00Z"
	if reply.Entries[0].Timestamp != expected {
		t.Errorf("expected timestamp=%q, got %q", expected, reply.Entries[0].Timestamp)
	}
}
