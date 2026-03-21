package analyticalclient_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"internal/application/analyticalclient"
	"internal/shared/problem"
)

func TestGetDispositionBreakdown_Success(t *testing.T) {
	reader := &stubAggregationReader{
		dispositions: []analyticalclient.DispositionCount{
			{Disposition: "approved", Count: 80},
			{Disposition: "rejected", Count: 15},
			{Disposition: "modified", Count: 5},
		},
	}
	uc := analyticalclient.NewGetDispositionBreakdownUseCase(reader, slog.Default())

	reply, prob := uc.Execute(context.Background(), analyticalclient.DispositionBreakdownQuery{
		Type: "position_exposure", Source: "binance", Symbol: "btcusdt", Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(reply.Dispositions) != 3 {
		t.Fatalf("expected 3 dispositions, got %d", len(reply.Dispositions))
	}
	if reply.Total != 100 {
		t.Errorf("expected total 100, got %d", reply.Total)
	}
	if reply.Source != "clickhouse" {
		t.Errorf("expected source clickhouse, got %s", reply.Source)
	}

	// Verify percentages are computed.
	for _, d := range reply.Dispositions {
		if d.Percentage <= 0 {
			t.Errorf("expected positive percentage for %s, got %f", d.Disposition, d.Percentage)
		}
	}

	// Find approved and check percentage.
	for _, d := range reply.Dispositions {
		if d.Disposition == "approved" && d.Percentage != 80.0 {
			t.Errorf("expected approved percentage 80.0, got %f", d.Percentage)
		}
	}
}

func TestGetDispositionBreakdown_EmptyResult(t *testing.T) {
	reader := &stubAggregationReader{dispositions: nil}
	uc := analyticalclient.NewGetDispositionBreakdownUseCase(reader, slog.Default())

	reply, prob := uc.Execute(context.Background(), analyticalclient.DispositionBreakdownQuery{
		Type: "position_exposure", Source: "binance", Symbol: "btcusdt", Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if reply.Total != 0 {
		t.Errorf("expected total 0, got %d", reply.Total)
	}
	if len(reply.Dispositions) != 0 {
		t.Errorf("expected 0 dispositions, got %d", len(reply.Dispositions))
	}
}

func TestGetDispositionBreakdown_MissingType(t *testing.T) {
	uc := analyticalclient.NewGetDispositionBreakdownUseCase(&stubAggregationReader{}, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.DispositionBreakdownQuery{
		Source: "binance", Symbol: "btcusdt", Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing type")
	}
	if prob.Code != problem.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %q", prob.Code)
	}
}

func TestGetDispositionBreakdown_ReaderError(t *testing.T) {
	reader := &stubAggregationReader{dispErr: errors.New("connection refused")}
	uc := analyticalclient.NewGetDispositionBreakdownUseCase(reader, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.DispositionBreakdownQuery{
		Type: "position_exposure", Source: "binance", Symbol: "btcusdt", Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for reader error")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %q", prob.Code)
	}
}

func TestGetDispositionBreakdown_NilUseCase(t *testing.T) {
	var uc *analyticalclient.GetDispositionBreakdownUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.DispositionBreakdownQuery{
		Type: "position_exposure", Source: "binance", Symbol: "btcusdt", Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %q", prob.Code)
	}
}
