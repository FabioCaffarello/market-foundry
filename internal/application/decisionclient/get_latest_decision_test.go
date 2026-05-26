package decisionclient_test

import (
	"context"
	"testing"
	"time"

	"internal/application/decisionclient"
	"internal/domain/decision"
	"internal/domain/instrument"
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

type mockDecisionGateway struct {
	dec  *decision.Decision
	prob *problem.Problem
}

func (m *mockDecisionGateway) GetLatestDecision(_ context.Context, _ decisionclient.DecisionLatestQuery) (decisionclient.DecisionLatestReply, *problem.Problem) {
	return decisionclient.DecisionLatestReply{Decision: m.dec}, m.prob
}

func TestGetLatestDecisionUseCase_ValidatesInput(t *testing.T) {
	uc := decisionclient.NewGetLatestDecisionUseCase(&mockDecisionGateway{})

	tests := []struct {
		name  string
		query decisionclient.DecisionLatestQuery
	}{
		{"empty type", decisionclient.DecisionLatestQuery{Type: "", Source: "binancef", Symbol: "btcusdt", Timeframe: 60}},
		{"empty source", decisionclient.DecisionLatestQuery{Type: "rsi_oversold", Source: "", Symbol: "btcusdt", Timeframe: 60}},
		{"empty symbol", decisionclient.DecisionLatestQuery{Type: "rsi_oversold", Source: "binancef", Symbol: "", Timeframe: 60}},
		{"zero timeframe", decisionclient.DecisionLatestQuery{Type: "rsi_oversold", Source: "binancef", Symbol: "btcusdt", Timeframe: 0}},
		{"negative timeframe", decisionclient.DecisionLatestQuery{Type: "rsi_oversold", Source: "binancef", Symbol: "btcusdt", Timeframe: -1}},
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

func TestGetLatestDecisionUseCase_ReturnsDecision(t *testing.T) {
	now := time.Now().UTC()
	dec := &decision.Decision{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Outcome:    decision.OutcomeTriggered,
		Severity:   decision.SeverityLow,
		Confidence: "0.85",
		Rationale:  "RSI 25.00 below oversold threshold 30.0 (distance 16.7%); severity low",
		Signals: []decision.SignalInput{
			{Type: "rsi", Value: "25.00", Timeframe: 60},
		},
		Metadata:  map[string]string{"threshold": "30.0"},
		Final:     true,
		Timestamp: now,
	}

	uc := decisionclient.NewGetLatestDecisionUseCase(&mockDecisionGateway{dec: dec})
	reply, prob := uc.Execute(context.Background(), decisionclient.DecisionLatestQuery{
		Type:      "rsi_oversold",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if reply.Decision == nil {
		t.Fatal("expected decision in reply")
	}
	if reply.Decision.Outcome != decision.OutcomeTriggered {
		t.Fatalf("expected triggered, got %s", reply.Decision.Outcome)
	}
}

func TestGetLatestDecisionUseCase_NilGateway(t *testing.T) {
	var uc *decisionclient.GetLatestDecisionUseCase
	_, prob := uc.Execute(context.Background(), decisionclient.DecisionLatestQuery{
		Type:      "rsi_oversold",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected unavailable error")
	}
}
