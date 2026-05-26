package strategyclient_test

import (
	"context"
	"testing"
	"time"

	"internal/application/strategyclient"
	"internal/domain/instrument"
	"internal/domain/strategy"
	"internal/shared/problem"
)

func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

type mockStrategyGateway struct {
	strat *strategy.Strategy
	prob  *problem.Problem
}

func (m *mockStrategyGateway) GetLatestStrategy(_ context.Context, _ strategyclient.StrategyLatestQuery) (strategyclient.StrategyLatestReply, *problem.Problem) {
	return strategyclient.StrategyLatestReply{Strategy: m.strat}, m.prob
}

func TestGetLatestStrategyUseCase_ValidatesInput(t *testing.T) {
	uc := strategyclient.NewGetLatestStrategyUseCase(&mockStrategyGateway{})

	tests := []struct {
		name  string
		query strategyclient.StrategyLatestQuery
	}{
		{"empty type", strategyclient.StrategyLatestQuery{Type: "", Source: "binancef", Symbol: "btcusdt", Timeframe: 60}},
		{"empty source", strategyclient.StrategyLatestQuery{Type: "mean_reversion_entry", Source: "", Symbol: "btcusdt", Timeframe: 60}},
		{"empty symbol", strategyclient.StrategyLatestQuery{Type: "mean_reversion_entry", Source: "binancef", Symbol: "", Timeframe: 60}},
		{"zero timeframe", strategyclient.StrategyLatestQuery{Type: "mean_reversion_entry", Source: "binancef", Symbol: "btcusdt", Timeframe: 0}},
		{"negative timeframe", strategyclient.StrategyLatestQuery{Type: "mean_reversion_entry", Source: "binancef", Symbol: "btcusdt", Timeframe: -1}},
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

func TestGetLatestStrategyUseCase_ReturnsStrategy(t *testing.T) {
	now := time.Now().UTC()
	strat := &strategy.Strategy{
		Type:       "mean_reversion_entry",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Direction:  strategy.DirectionLong,
		Confidence: "0.85",
		Decisions: []strategy.DecisionInput{
			{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Timeframe: 60},
		},
		Parameters: map[string]string{"entry": "market"},
		Final:      true,
		Timestamp:  now,
	}

	uc := strategyclient.NewGetLatestStrategyUseCase(&mockStrategyGateway{strat: strat})
	reply, prob := uc.Execute(context.Background(), strategyclient.StrategyLatestQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if reply.Strategy == nil {
		t.Fatal("expected strategy in reply")
	}
	if reply.Strategy.Direction != strategy.DirectionLong {
		t.Fatalf("expected long, got %s", reply.Strategy.Direction)
	}
}

func TestGetLatestStrategyUseCase_NilGateway(t *testing.T) {
	var uc *strategyclient.GetLatestStrategyUseCase
	_, prob := uc.Execute(context.Background(), strategyclient.StrategyLatestQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected unavailable error")
	}
}
