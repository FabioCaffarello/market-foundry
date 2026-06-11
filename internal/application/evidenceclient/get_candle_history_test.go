package evidenceclient_test

import (
	"internal/domain/instrument"

	"context"
	"testing"
	"time"

	"internal/application/evidenceclient"
	"internal/domain/evidence"
	"internal/shared/problem"
)

type mockCandleHistoryGateway struct {
	candles   []evidence.EvidenceCandle
	prob      *problem.Problem
	lastQuery evidenceclient.CandleHistoryQuery
}

func (m *mockCandleHistoryGateway) GetCandleHistory(_ context.Context, q evidenceclient.CandleHistoryQuery) (evidenceclient.CandleHistoryReply, *problem.Problem) {
	m.lastQuery = q
	return evidenceclient.CandleHistoryReply{Candles: m.candles}, m.prob
}

func TestGetCandleHistoryUseCase_ValidatesInput(t *testing.T) {
	uc := evidenceclient.NewGetCandleHistoryUseCase(&mockCandleHistoryGateway{})

	tests := []struct {
		name  string
		query evidenceclient.CandleHistoryQuery
	}{
		{"empty source", evidenceclient.CandleHistoryQuery{Source: "", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60}},
		{"zero instrument", evidenceclient.CandleHistoryQuery{Source: "binancef", Instrument: instrument.CanonicalInstrument{}, Timeframe: 60}},
		{"zero timeframe", evidenceclient.CandleHistoryQuery{Source: "binancef", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 0}},
		{"negative timeframe", evidenceclient.CandleHistoryQuery{Source: "binancef", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: -1}},
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

func TestGetCandleHistoryUseCase_ValidatesRange(t *testing.T) {
	uc := evidenceclient.NewGetCandleHistoryUseCase(&mockCandleHistoryGateway{})

	tests := []struct {
		name    string
		since   int64
		until   int64
		wantErr bool
	}{
		{"since only", 1710000000, 0, false},
		{"until only", 0, 1710003600, false},
		{"valid range", 1710000000, 1710003600, false},
		{"equal bounds", 1710000000, 1710000000, false},
		{"since after until", 1710003600, 1710000000, true},
		{"negative since", -1, 0, true},
		{"negative until", 0, -1, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, prob := uc.Execute(context.Background(), evidenceclient.CandleHistoryQuery{
				Source:     "binancef",
				Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
				Timeframe:  60,
				Since:      tc.since,
				Until:      tc.until,
			})
			if tc.wantErr && prob == nil {
				t.Fatal("expected validation error")
			}
			if !tc.wantErr && prob != nil {
				t.Fatalf("unexpected error: %v", prob)
			}
		})
	}
}

func TestGetCandleHistoryUseCase_DefaultsLimit(t *testing.T) {
	gw := &mockCandleHistoryGateway{}
	uc := evidenceclient.NewGetCandleHistoryUseCase(gw)

	// Zero limit should not cause validation error — it gets defaulted.
	_, prob := uc.Execute(context.Background(), evidenceclient.CandleHistoryQuery{
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		Limit:      0,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if gw.lastQuery.Limit != 10 {
		t.Fatalf("expected default limit 10, got %d", gw.lastQuery.Limit)
	}
}

func TestGetCandleHistoryUseCase_ClampsLimit(t *testing.T) {
	gw := &mockCandleHistoryGateway{}
	uc := evidenceclient.NewGetCandleHistoryUseCase(gw)

	_, prob := uc.Execute(context.Background(), evidenceclient.CandleHistoryQuery{
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		Limit:      999,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if gw.lastQuery.Limit != 100 {
		t.Fatalf("expected clamped limit 100, got %d", gw.lastQuery.Limit)
	}
}

func TestGetCandleHistoryUseCase_PassesSinceUntil(t *testing.T) {
	gw := &mockCandleHistoryGateway{}
	uc := evidenceclient.NewGetCandleHistoryUseCase(gw)

	_, prob := uc.Execute(context.Background(), evidenceclient.CandleHistoryQuery{
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		Limit:      5,
		Since:      1710000000,
		Until:      1710003600,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if gw.lastQuery.Since != 1710000000 {
		t.Fatalf("expected since 1710000000, got %d", gw.lastQuery.Since)
	}
	if gw.lastQuery.Until != 1710003600 {
		t.Fatalf("expected until 1710003600, got %d", gw.lastQuery.Until)
	}
}

func TestGetCandleHistoryUseCase_ReturnsCandles(t *testing.T) {
	now := time.Now().UTC().Truncate(60 * time.Second)
	candles := []evidence.EvidenceCandle{
		{
			Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Open: "102.00", High: "106.00", Low: "101.00", Close: "104.00",
			Volume: "500.00", TradeCount: 20,
			OpenTime: now, CloseTime: now.Add(60 * time.Second), Final: true,
		},
		{
			Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Open: "100.00", High: "105.00", Low: "99.00", Close: "102.00",
			Volume: "1000.00", TradeCount: 42,
			OpenTime: now.Add(-60 * time.Second), CloseTime: now, Final: true,
		},
	}

	uc := evidenceclient.NewGetCandleHistoryUseCase(&mockCandleHistoryGateway{candles: candles})
	reply, prob := uc.Execute(context.Background(), evidenceclient.CandleHistoryQuery{
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		Limit:      10,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if len(reply.Candles) != 2 {
		t.Fatalf("expected 2 candles, got %d", len(reply.Candles))
	}
}

func TestGetCandleHistoryUseCase_NilGateway(t *testing.T) {
	var uc *evidenceclient.GetCandleHistoryUseCase
	_, prob := uc.Execute(context.Background(), evidenceclient.CandleHistoryQuery{
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob == nil {
		t.Fatal("expected unavailable error")
	}
}
