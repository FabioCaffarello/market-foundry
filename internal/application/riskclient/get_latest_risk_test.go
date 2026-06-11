package riskclient_test

import (
	"context"
	"testing"
	"time"

	"internal/application/riskclient"
	"internal/domain/instrument"
	"internal/domain/risk"
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

type mockRiskGateway struct {
	assessment *risk.RiskAssessment
	prob       *problem.Problem
}

func (m *mockRiskGateway) GetLatestRisk(_ context.Context, _ riskclient.RiskLatestQuery) (riskclient.RiskLatestReply, *problem.Problem) {
	return riskclient.RiskLatestReply{RiskAssessment: m.assessment}, m.prob
}

func TestGetLatestRiskUseCase_ValidatesInput(t *testing.T) {
	uc := riskclient.NewGetLatestRiskUseCase(&mockRiskGateway{})

	tests := []struct {
		name  string
		query riskclient.RiskLatestQuery
	}{
		{"empty type", riskclient.RiskLatestQuery{Type: "", Source: "binancef", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60}},
		{"empty source", riskclient.RiskLatestQuery{Type: "position_exposure", Source: "", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60}},
		{"zero instrument", riskclient.RiskLatestQuery{Type: "position_exposure", Source: "binancef", Instrument: instrument.CanonicalInstrument{}, Timeframe: 60}},
		{"zero timeframe", riskclient.RiskLatestQuery{Type: "position_exposure", Source: "binancef", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 0}},
		{"negative timeframe", riskclient.RiskLatestQuery{Type: "position_exposure", Source: "binancef", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: -1}},
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

func TestGetLatestRiskUseCase_ReturnsRisk(t *testing.T) {
	now := time.Now().UTC()
	assessment := &risk.RiskAssessment{
		Type:        "position_exposure",
		Source:      "binancef",
		Instrument:  btcUSDTPerp(t),
		Timeframe:   60,
		Disposition: risk.DispositionApproved,
		Confidence:  "0.85",
		Strategies: []risk.StrategyInput{
			{Type: "mean_reversion_entry", Direction: "long", Confidence: "0.85", Timeframe: 60},
		},
		Constraints: risk.Constraints{MaxPositionSize: "0.01", MaxExposure: "0.05"},
		Rationale:   "Position size within exposure limits",
		Final:       true,
		Timestamp:   now,
	}

	uc := riskclient.NewGetLatestRiskUseCase(&mockRiskGateway{assessment: assessment})
	reply, prob := uc.Execute(context.Background(), riskclient.RiskLatestQuery{
		Type:       "position_exposure",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if reply.RiskAssessment == nil {
		t.Fatal("expected risk assessment in reply")
	}
	if reply.RiskAssessment.Disposition != risk.DispositionApproved {
		t.Fatalf("expected approved, got %s", reply.RiskAssessment.Disposition)
	}
}

func TestGetLatestRiskUseCase_NilGateway(t *testing.T) {
	var uc *riskclient.GetLatestRiskUseCase
	_, prob := uc.Execute(context.Background(), riskclient.RiskLatestQuery{
		Type:       "position_exposure",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob == nil {
		t.Fatal("expected unavailable error")
	}
}
