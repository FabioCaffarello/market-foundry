package risk_test

import (
	"testing"
	"time"

	apprisk "internal/application/risk"
	"internal/domain/instrument"
)

var btcUSDTSpot = func() instrument.CanonicalInstrument {
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractSpot)
	if prob != nil {
		panic("test setup: BTC/USDT-spot: " + prob.Message)
	}
	return inst
}()

// TestRiskEvaluators_NewForInstrument_Passthrough exercises the H-6.c.1
// pass-through constructors for each of the 2 risk evaluators. See
// internal/application/signal/instrument_passthrough_test.go for the
// rationale and canary contract.
func TestRiskEvaluators_NewForInstrument_Passthrough(t *testing.T) {
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	t.Run("DrawdownLimit", func(t *testing.T) {
		e := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTSpot, 60)
		r, ok := e.Evaluate("mean_reversion_entry", "long", "0.85", "high", "rsi oversold", 60, ts)
		if !ok {
			t.Fatal("DrawdownLimitEvaluator.Evaluate returned ok=false")
		}
		if r.Instrument != btcUSDTSpot {
			t.Errorf("Instrument = %+v, want %+v (pass-through broken)", r.Instrument, btcUSDTSpot)
		}
	})

	t.Run("PositionExposure", func(t *testing.T) {
		e := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTSpot, 60)
		r, ok := e.Evaluate("mean_reversion_entry", "long", "0.85", "high", "rsi oversold", 60, ts)
		if !ok {
			t.Fatal("PositionExposureEvaluator.Evaluate returned ok=false")
		}
		if r.Instrument != btcUSDTSpot {
			t.Errorf("Instrument = %+v, want %+v (pass-through broken)", r.Instrument, btcUSDTSpot)
		}
	})
}
