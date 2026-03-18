package evidenceclient_test

import (
	"context"
	"testing"
	"time"

	"internal/application/evidenceclient"
	"internal/domain/evidence"
	"internal/shared/problem"
)

type mockEvidenceGateway struct {
	candle *evidence.EvidenceCandle
	prob   *problem.Problem
}

func (m *mockEvidenceGateway) GetLatestCandle(_ context.Context, _ evidenceclient.CandleLatestQuery) (evidenceclient.CandleLatestReply, *problem.Problem) {
	return evidenceclient.CandleLatestReply{Candle: m.candle}, m.prob
}

func TestGetLatestCandleUseCase_ValidatesInput(t *testing.T) {
	uc := evidenceclient.NewGetLatestCandleUseCase(&mockEvidenceGateway{})

	tests := []struct {
		name  string
		query evidenceclient.CandleLatestQuery
	}{
		{"empty source", evidenceclient.CandleLatestQuery{Source: "", Symbol: "btcusdt", Timeframe: 60}},
		{"empty symbol", evidenceclient.CandleLatestQuery{Source: "binancef", Symbol: "", Timeframe: 60}},
		{"zero timeframe", evidenceclient.CandleLatestQuery{Source: "binancef", Symbol: "btcusdt", Timeframe: 0}},
		{"negative timeframe", evidenceclient.CandleLatestQuery{Source: "binancef", Symbol: "btcusdt", Timeframe: -1}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, prob := uc.Execute(context.Background(), tc.query)
			if prob == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestGetLatestCandleUseCase_ReturnsCandle(t *testing.T) {
	now := time.Now().UTC().Truncate(60 * time.Second)
	candle := &evidence.EvidenceCandle{
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Open:       "100.00",
		High:       "105.00",
		Low:        "99.00",
		Close:      "102.00",
		Volume:     "1000.00",
		TradeCount: 42,
		OpenTime:   now,
		CloseTime:  now.Add(60 * time.Second),
		Final:      false,
	}

	uc := evidenceclient.NewGetLatestCandleUseCase(&mockEvidenceGateway{candle: candle})
	reply, prob := uc.Execute(context.Background(), evidenceclient.CandleLatestQuery{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if reply.Candle == nil {
		t.Fatal("expected candle in reply")
	}
	if reply.Candle.Source != "binancef" {
		t.Fatalf("expected source binancef, got %s", reply.Candle.Source)
	}
}

func TestGetLatestCandleUseCase_NilGateway(t *testing.T) {
	var uc *evidenceclient.GetLatestCandleUseCase
	_, prob := uc.Execute(context.Background(), evidenceclient.CandleLatestQuery{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected unavailable error")
	}
}
