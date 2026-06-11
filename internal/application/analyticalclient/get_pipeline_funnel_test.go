package analyticalclient_test

import (
	"internal/domain/instrument"

	"context"
	"errors"
	"log/slog"
	"testing"

	"internal/application/analyticalclient"
	"internal/shared/problem"
)

// stubAggregationReader implements analyticalclient.AggregationReader for unit tests.
type stubAggregationReader struct {
	funnel       []analyticalclient.StageFunnelCount
	dispositions []analyticalclient.DispositionCount
	funnelErr    error
	dispErr      error
}

func (s *stubAggregationReader) QueryPipelineFunnel(_ context.Context, _, _ string, _ instrument.CanonicalInstrument, _ int, _, _ int64) ([]analyticalclient.StageFunnelCount, error) {
	return s.funnel, s.funnelErr
}

func (s *stubAggregationReader) QueryDispositionBreakdown(_ context.Context, _, _ string, _ instrument.CanonicalInstrument, _ int, _, _ int64) ([]analyticalclient.DispositionCount, error) {
	return s.dispositions, s.dispErr
}

func TestGetPipelineFunnel_Success(t *testing.T) {
	reader := &stubAggregationReader{
		funnel: []analyticalclient.StageFunnelCount{
			{Stage: "signal", Count: 100},
			{Stage: "decision", Count: 80},
			{Stage: "strategy", Count: 60},
			{Stage: "risk", Count: 55},
			{Stage: "execution", Count: 50},
		},
	}
	uc := analyticalclient.NewGetPipelineFunnelUseCase(reader, slog.Default())

	reply, prob := uc.Execute(context.Background(), analyticalclient.PipelineFunnelQuery{
		Type: "ema_crossover", Source: "binance", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(reply.Stages) != 5 {
		t.Fatalf("expected 5 stages, got %d", len(reply.Stages))
	}
	if reply.Source != "clickhouse" {
		t.Errorf("expected source clickhouse, got %s", reply.Source)
	}
}

func TestGetPipelineFunnel_MissingType(t *testing.T) {
	uc := analyticalclient.NewGetPipelineFunnelUseCase(&stubAggregationReader{}, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.PipelineFunnelQuery{
		Source: "binance", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing type")
	}
	if prob.Code != problem.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %q", prob.Code)
	}
}

func TestGetPipelineFunnel_MissingSource(t *testing.T) {
	uc := analyticalclient.NewGetPipelineFunnelUseCase(&stubAggregationReader{}, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.PipelineFunnelQuery{
		Type: "ema", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing source")
	}
}

func TestGetPipelineFunnel_InvalidTimeframe(t *testing.T) {
	uc := analyticalclient.NewGetPipelineFunnelUseCase(&stubAggregationReader{}, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.PipelineFunnelQuery{
		Type: "ema", Source: "binance", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: -1,
	})
	if prob == nil {
		t.Fatal("expected problem for invalid timeframe")
	}
}

func TestGetPipelineFunnel_ReaderError(t *testing.T) {
	reader := &stubAggregationReader{funnelErr: errors.New("connection refused")}
	uc := analyticalclient.NewGetPipelineFunnelUseCase(reader, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.PipelineFunnelQuery{
		Type: "ema", Source: "binance", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for reader error")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %q", prob.Code)
	}
}

func TestGetPipelineFunnel_NilUseCase(t *testing.T) {
	var uc *analyticalclient.GetPipelineFunnelUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.PipelineFunnelQuery{
		Type: "ema", Source: "binance", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %q", prob.Code)
	}
}
