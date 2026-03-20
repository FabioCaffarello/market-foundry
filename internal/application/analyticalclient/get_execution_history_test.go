package analyticalclient_test

import (
	"context"
	"errors"
	"testing"

	"internal/application/analyticalclient"
	"internal/domain/execution"
)

type stubExecutionReader struct {
	executions []execution.ExecutionIntent
	err        error
}

func (s *stubExecutionReader) QueryExecutionHistory(_ context.Context, _, _, _ string, _ int, _, _ string, _, _ int64, _ int) ([]execution.ExecutionIntent, error) {
	return s.executions, s.err
}

func TestGetExecutionHistoryUseCase_MissingType(t *testing.T) {
	uc := analyticalclient.NewGetExecutionHistoryUseCase(&stubExecutionReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing type")
	}
}

func TestGetExecutionHistoryUseCase_MissingSource(t *testing.T) {
	uc := analyticalclient.NewGetExecutionHistoryUseCase(&stubExecutionReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Type:      "paper_order",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing source")
	}
}

func TestGetExecutionHistoryUseCase_MissingSymbol(t *testing.T) {
	uc := analyticalclient.NewGetExecutionHistoryUseCase(&stubExecutionReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Type:      "paper_order",
		Source:    "derive",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing symbol")
	}
}

func TestGetExecutionHistoryUseCase_InvalidTimeframe(t *testing.T) {
	uc := analyticalclient.NewGetExecutionHistoryUseCase(&stubExecutionReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Type:      "paper_order",
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 0,
	})
	if prob == nil {
		t.Fatal("expected problem for zero timeframe")
	}
}

func TestGetExecutionHistoryUseCase_SinceAfterUntil(t *testing.T) {
	uc := analyticalclient.NewGetExecutionHistoryUseCase(&stubExecutionReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Type:      "paper_order",
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Since:     2000,
		Until:     1000,
	})
	if prob == nil {
		t.Fatal("expected problem for since > until")
	}
}

func TestGetExecutionHistoryUseCase_DefaultLimit(t *testing.T) {
	reader := &stubExecutionReader{executions: []execution.ExecutionIntent{}}
	uc := analyticalclient.NewGetExecutionHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Type:      "paper_order",
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetExecutionHistoryUseCase_LimitClamped(t *testing.T) {
	reader := &stubExecutionReader{executions: []execution.ExecutionIntent{}}
	uc := analyticalclient.NewGetExecutionHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Type:      "paper_order",
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Limit:     9999,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetExecutionHistoryUseCase_WithSideFilter(t *testing.T) {
	reader := &stubExecutionReader{executions: []execution.ExecutionIntent{}}
	uc := analyticalclient.NewGetExecutionHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Type:      "paper_order",
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Side:      "buy",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetExecutionHistoryUseCase_WithStatusFilter(t *testing.T) {
	reader := &stubExecutionReader{executions: []execution.ExecutionIntent{}}
	uc := analyticalclient.NewGetExecutionHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Type:      "paper_order",
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Status:    "filled",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetExecutionHistoryUseCase_ReaderError(t *testing.T) {
	reader := &stubExecutionReader{err: errors.New("connection refused")}
	uc := analyticalclient.NewGetExecutionHistoryUseCase(reader, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Type:      "paper_order",
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for reader error")
	}
}

func TestGetExecutionHistoryUseCase_NilReader(t *testing.T) {
	uc := analyticalclient.NewGetExecutionHistoryUseCase(nil, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Type:      "paper_order",
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil reader")
	}
}

func TestGetExecutionHistoryUseCase_NilUseCaseExecute(t *testing.T) {
	var uc *analyticalclient.GetExecutionHistoryUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Type:      "paper_order",
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
}

func TestGetExecutionHistoryUseCase_NegativeSince(t *testing.T) {
	uc := analyticalclient.NewGetExecutionHistoryUseCase(&stubExecutionReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.ExecutionHistoryQuery{
		Type:      "paper_order",
		Source:    "derive",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Since:     -1,
	})
	if prob == nil {
		t.Fatal("expected problem for negative since")
	}
}
