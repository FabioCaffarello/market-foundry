package analyticalclient_test

import (
	"context"
	"errors"
	"testing"

	"internal/application/analyticalclient"
	"internal/domain/evidence"
)

type stubCandleReader struct {
	candles []evidence.EvidenceCandle
	err     error
}

func (s *stubCandleReader) QueryCandleHistory(_ context.Context, _, _ string, _ int, _, _ int64, _ int) ([]evidence.EvidenceCandle, error) {
	return s.candles, s.err
}

func TestGetCandleHistoryUseCase_MissingSource(t *testing.T) {
	uc := analyticalclient.NewGetCandleHistoryUseCase(&stubCandleReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.CandleHistoryQuery{
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing source")
	}
}

func TestGetCandleHistoryUseCase_MissingSymbol(t *testing.T) {
	uc := analyticalclient.NewGetCandleHistoryUseCase(&stubCandleReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.CandleHistoryQuery{
		Source:    "binancef",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing symbol")
	}
}

func TestGetCandleHistoryUseCase_InvalidTimeframe(t *testing.T) {
	uc := analyticalclient.NewGetCandleHistoryUseCase(&stubCandleReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.CandleHistoryQuery{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 0,
	})
	if prob == nil {
		t.Fatal("expected problem for zero timeframe")
	}
}

func TestGetCandleHistoryUseCase_SinceAfterUntil(t *testing.T) {
	uc := analyticalclient.NewGetCandleHistoryUseCase(&stubCandleReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.CandleHistoryQuery{
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

func TestGetCandleHistoryUseCase_DefaultLimit(t *testing.T) {
	reader := &stubCandleReader{candles: []evidence.EvidenceCandle{}}
	uc := analyticalclient.NewGetCandleHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.CandleHistoryQuery{
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

func TestGetCandleHistoryUseCase_LimitClamped(t *testing.T) {
	reader := &stubCandleReader{candles: []evidence.EvidenceCandle{}}
	uc := analyticalclient.NewGetCandleHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.CandleHistoryQuery{
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

func TestGetCandleHistoryUseCase_ReaderError(t *testing.T) {
	reader := &stubCandleReader{err: errors.New("connection refused")}
	uc := analyticalclient.NewGetCandleHistoryUseCase(reader, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.CandleHistoryQuery{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for reader error")
	}
}

func TestGetCandleHistoryUseCase_NilReader(t *testing.T) {
	uc := analyticalclient.NewGetCandleHistoryUseCase(nil, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.CandleHistoryQuery{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil reader")
	}
}

func TestGetCandleHistoryUseCase_NilUseCaseExecute(t *testing.T) {
	var uc *analyticalclient.GetCandleHistoryUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.CandleHistoryQuery{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
}
