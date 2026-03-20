package analyticalclient_test

import (
	"context"
	"errors"
	"testing"

	"internal/application/analyticalclient"
	"internal/domain/strategy"
)

type stubStrategyReader struct {
	strategies []strategy.Strategy
	err        error
}

func (s *stubStrategyReader) QueryStrategyHistory(_ context.Context, _, _, _ string, _ int, _ string, _, _ int64, _ int) ([]strategy.Strategy, error) {
	return s.strategies, s.err
}

func TestGetStrategyHistoryUseCase_MissingType(t *testing.T) {
	uc := analyticalclient.NewGetStrategyHistoryUseCase(&stubStrategyReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.StrategyHistoryQuery{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing type")
	}
}

func TestGetStrategyHistoryUseCase_MissingSource(t *testing.T) {
	uc := analyticalclient.NewGetStrategyHistoryUseCase(&stubStrategyReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.StrategyHistoryQuery{
		Type:      "mean_reversion_entry",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing source")
	}
}

func TestGetStrategyHistoryUseCase_MissingSymbol(t *testing.T) {
	uc := analyticalclient.NewGetStrategyHistoryUseCase(&stubStrategyReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.StrategyHistoryQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing symbol")
	}
}

func TestGetStrategyHistoryUseCase_InvalidTimeframe(t *testing.T) {
	uc := analyticalclient.NewGetStrategyHistoryUseCase(&stubStrategyReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.StrategyHistoryQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 0,
	})
	if prob == nil {
		t.Fatal("expected problem for zero timeframe")
	}
}

func TestGetStrategyHistoryUseCase_SinceAfterUntil(t *testing.T) {
	uc := analyticalclient.NewGetStrategyHistoryUseCase(&stubStrategyReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.StrategyHistoryQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Since:     2000,
		Until:     1000,
	})
	if prob == nil {
		t.Fatal("expected problem for since > until")
	}
}

func TestGetStrategyHistoryUseCase_DefaultLimit(t *testing.T) {
	reader := &stubStrategyReader{strategies: []strategy.Strategy{}}
	uc := analyticalclient.NewGetStrategyHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.StrategyHistoryQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
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

func TestGetStrategyHistoryUseCase_LimitClamped(t *testing.T) {
	reader := &stubStrategyReader{strategies: []strategy.Strategy{}}
	uc := analyticalclient.NewGetStrategyHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.StrategyHistoryQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
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

func TestGetStrategyHistoryUseCase_WithDirection(t *testing.T) {
	reader := &stubStrategyReader{strategies: []strategy.Strategy{}}
	uc := analyticalclient.NewGetStrategyHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.StrategyHistoryQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Direction: "long",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetStrategyHistoryUseCase_ReaderError(t *testing.T) {
	reader := &stubStrategyReader{err: errors.New("connection refused")}
	uc := analyticalclient.NewGetStrategyHistoryUseCase(reader, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.StrategyHistoryQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for reader error")
	}
}

func TestGetStrategyHistoryUseCase_NilReader(t *testing.T) {
	uc := analyticalclient.NewGetStrategyHistoryUseCase(nil, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.StrategyHistoryQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil reader")
	}
}

func TestGetStrategyHistoryUseCase_NilUseCaseExecute(t *testing.T) {
	var uc *analyticalclient.GetStrategyHistoryUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.StrategyHistoryQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
}

func TestGetStrategyHistoryUseCase_NegativeSince(t *testing.T) {
	uc := analyticalclient.NewGetStrategyHistoryUseCase(&stubStrategyReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.StrategyHistoryQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Since:     -1,
	})
	if prob == nil {
		t.Fatal("expected problem for negative since")
	}
}
