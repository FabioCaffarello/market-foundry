package analyticalclient_test

import (
	"internal/domain/instrument"

	"context"
	"errors"
	"testing"

	"internal/application/analyticalclient"
	"internal/domain/decision"
)

type stubDecisionReader struct {
	decisions []decision.Decision
	err       error
}

func (s *stubDecisionReader) QueryDecisionHistory(_ context.Context, _, _ string, _ instrument.CanonicalInstrument, _ int, _ string, _, _ int64, _ int) ([]decision.Decision, error) {
	return s.decisions, s.err
}

func TestGetDecisionHistoryUseCase_MissingType(t *testing.T) {
	uc := analyticalclient.NewGetDecisionHistoryUseCase(&stubDecisionReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.DecisionHistoryQuery{
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing type")
	}
}

func TestGetDecisionHistoryUseCase_MissingSource(t *testing.T) {
	uc := analyticalclient.NewGetDecisionHistoryUseCase(&stubDecisionReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.DecisionHistoryQuery{
		Type:       "rsi_oversold",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing source")
	}
}

func TestGetDecisionHistoryUseCase_MissingSymbol(t *testing.T) {
	uc := analyticalclient.NewGetDecisionHistoryUseCase(&stubDecisionReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.DecisionHistoryQuery{
		Type:      "rsi_oversold",
		Source:    "binancef",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for missing symbol")
	}
}

func TestGetDecisionHistoryUseCase_InvalidTimeframe(t *testing.T) {
	uc := analyticalclient.NewGetDecisionHistoryUseCase(&stubDecisionReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.DecisionHistoryQuery{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  0,
	})
	if prob == nil {
		t.Fatal("expected problem for zero timeframe")
	}
}

func TestGetDecisionHistoryUseCase_SinceAfterUntil(t *testing.T) {
	uc := analyticalclient.NewGetDecisionHistoryUseCase(&stubDecisionReader{}, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.DecisionHistoryQuery{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		Since:      2000,
		Until:      1000,
	})
	if prob == nil {
		t.Fatal("expected problem for since > until")
	}
}

func TestGetDecisionHistoryUseCase_DefaultLimit(t *testing.T) {
	reader := &stubDecisionReader{decisions: []decision.Decision{}}
	uc := analyticalclient.NewGetDecisionHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.DecisionHistoryQuery{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetDecisionHistoryUseCase_LimitClamped(t *testing.T) {
	reader := &stubDecisionReader{decisions: []decision.Decision{}}
	uc := analyticalclient.NewGetDecisionHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.DecisionHistoryQuery{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		Limit:      9999,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetDecisionHistoryUseCase_WithOutcome(t *testing.T) {
	reader := &stubDecisionReader{decisions: []decision.Decision{}}
	uc := analyticalclient.NewGetDecisionHistoryUseCase(reader, nil)
	result, prob := uc.Execute(context.Background(), analyticalclient.DecisionHistoryQuery{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		Outcome:    "triggered",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", result.Source)
	}
}

func TestGetDecisionHistoryUseCase_ReaderError(t *testing.T) {
	reader := &stubDecisionReader{err: errors.New("connection refused")}
	uc := analyticalclient.NewGetDecisionHistoryUseCase(reader, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.DecisionHistoryQuery{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob == nil {
		t.Fatal("expected problem for reader error")
	}
}

func TestGetDecisionHistoryUseCase_NilReader(t *testing.T) {
	uc := analyticalclient.NewGetDecisionHistoryUseCase(nil, nil)
	_, prob := uc.Execute(context.Background(), analyticalclient.DecisionHistoryQuery{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil reader")
	}
}

func TestGetDecisionHistoryUseCase_NilUseCaseExecute(t *testing.T) {
	var uc *analyticalclient.GetDecisionHistoryUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.DecisionHistoryQuery{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
}
