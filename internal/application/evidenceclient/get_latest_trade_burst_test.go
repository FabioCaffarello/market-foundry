package evidenceclient_test

import (
	"context"
	"testing"
	"time"

	"internal/application/evidenceclient"
	"internal/domain/evidence"
	"internal/shared/problem"
)

type mockTradeBurstGateway struct {
	burst *evidence.EvidenceTradeBurst
	prob  *problem.Problem
}

func (m *mockTradeBurstGateway) GetLatestTradeBurst(_ context.Context, _ evidenceclient.TradeBurstLatestQuery) (evidenceclient.TradeBurstLatestReply, *problem.Problem) {
	return evidenceclient.TradeBurstLatestReply{TradeBurst: m.burst}, m.prob
}

func TestGetLatestTradeBurstUseCase_ValidatesInput(t *testing.T) {
	uc := evidenceclient.NewGetLatestTradeBurstUseCase(&mockTradeBurstGateway{})

	tests := []struct {
		name  string
		query evidenceclient.TradeBurstLatestQuery
	}{
		{"empty source", evidenceclient.TradeBurstLatestQuery{Source: "", Symbol: "btcusdt", Timeframe: 60}},
		{"empty symbol", evidenceclient.TradeBurstLatestQuery{Source: "binancef", Symbol: "", Timeframe: 60}},
		{"zero timeframe", evidenceclient.TradeBurstLatestQuery{Source: "binancef", Symbol: "btcusdt", Timeframe: 0}},
		{"negative timeframe", evidenceclient.TradeBurstLatestQuery{Source: "binancef", Symbol: "btcusdt", Timeframe: -1}},
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

func TestGetLatestTradeBurstUseCase_ReturnsTradeBurst(t *testing.T) {
	now := time.Now().UTC().Truncate(60 * time.Second)
	burst := &evidence.EvidenceTradeBurst{
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		TradeCount: 42,
		BuyVolume:  "500.00",
		SellVolume: "500.00",
		OpenTime:   now,
		CloseTime:  now.Add(60 * time.Second),
		Burst:      false,
		Final:      false,
	}

	uc := evidenceclient.NewGetLatestTradeBurstUseCase(&mockTradeBurstGateway{burst: burst})
	reply, prob := uc.Execute(context.Background(), evidenceclient.TradeBurstLatestQuery{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if reply.TradeBurst == nil {
		t.Fatal("expected trade burst in reply")
	}
	if reply.TradeBurst.Source != "binancef" {
		t.Fatalf("expected source binancef, got %s", reply.TradeBurst.Source)
	}
}

func TestGetLatestTradeBurstUseCase_NilGateway(t *testing.T) {
	var uc *evidenceclient.GetLatestTradeBurstUseCase
	_, prob := uc.Execute(context.Background(), evidenceclient.TradeBurstLatestQuery{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected unavailable error")
	}
}
