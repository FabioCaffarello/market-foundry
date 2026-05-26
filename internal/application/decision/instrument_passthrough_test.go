package decision_test

import (
	"testing"
	"time"

	appdecision "internal/application/decision"
	"internal/domain/instrument"
)

// btcUSDTSpot is the canonical fixture for H-6.c.1 passthrough tests.
var btcUSDTSpot = func() instrument.CanonicalInstrument {
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractSpot)
	if prob != nil {
		panic("test setup: BTC/USDT-spot: " + prob.Message)
	}
	return inst
}()

// TestEvaluators_NewForInstrument_Passthrough exercises the H-6.c.1
// pass-through constructors for each of the 3 decision evaluators.
// See internal/application/signal/instrument_passthrough_test.go for
// the rationale and canary contract.
func TestEvaluators_NewForInstrument_Passthrough(t *testing.T) {
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	t.Run("RSIOversold", func(t *testing.T) {
		e := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTSpot, 60)
		// RSI=25 → below 30 threshold → triggered.
		dec, ok := e.Evaluate("rsi", "25.0", 60, ts)
		if !ok {
			t.Fatal("RSIOversoldEvaluator returned ok=false")
		}
		if dec.Instrument != btcUSDTSpot {
			t.Errorf("Instrument = %+v, want %+v (pass-through broken)", dec.Instrument, btcUSDTSpot)
		}
	})

	t.Run("EMACrossover", func(t *testing.T) {
		e := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTSpot, 60)
		dec, ok := e.Evaluate("ema_crossover", "bullish", 60, ts)
		if !ok {
			t.Fatal("EMACrossoverEvaluator returned ok=false")
		}
		if dec.Instrument != btcUSDTSpot {
			t.Errorf("Instrument = %+v, want %+v (pass-through broken)", dec.Instrument, btcUSDTSpot)
		}
	})

	t.Run("BollingerSqueeze", func(t *testing.T) {
		e := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTSpot, 60)
		// Bollinger squeeze requires metadata fields.
		meta := map[string]string{"bandwidth": "0.005", "sma": "100.0"}
		dec, ok := e.Evaluate("bollinger", "0.5", 60, ts, meta)
		if !ok {
			t.Fatal("BollingerSqueezeEvaluator returned ok=false")
		}
		if dec.Instrument != btcUSDTSpot {
			t.Errorf("Instrument = %+v, want %+v (pass-through broken)", dec.Instrument, btcUSDTSpot)
		}
	})
}
