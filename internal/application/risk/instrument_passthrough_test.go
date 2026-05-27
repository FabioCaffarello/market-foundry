package risk_test

import (
	"strings"
	"testing"
	"time"

	apprisk "internal/application/risk"
	"internal/domain/instrument"
)

// btcUSDTSpot / btcUSDTPerp / ethUSDTPerp / solUSDTPerp are the
// canonical fixtures for the H-6.c.1 risk evaluator tests after
// commit 7d removed the legacy (source, symbol) constructors. Spot is
// the pass-through canary contrast; Perp variants reflect the
// original ("binancef", "<base>usdt") tuple semantics produced by the
// H-6.b sunset boundary helper.
var (
	btcUSDTSpot = mustPerpOrSpot("BTC", instrument.ContractSpot)
	btcUSDTPerp = mustPerpOrSpot("BTC", instrument.ContractPerpetual)
	ethUSDTPerp = mustPerpOrSpot("ETH", instrument.ContractPerpetual)
	solUSDTPerp = mustPerpOrSpot("SOL", instrument.ContractPerpetual)
)

func mustPerpOrSpot(base string, contract instrument.ContractType) instrument.CanonicalInstrument {
	inst, prob := instrument.New(base, "USDT", contract)
	if prob != nil {
		panic("test setup: " + base + "/USDT: " + prob.Message)
	}
	return inst
}

// instrumentForSymbol maps a venue-native lowercase symbol ("btcusdt")
// to its canonical perpetual Instrument. Used by the multi-symbol
// concurrency tests where the symbol is a parameterized struct field
// (so a named fixture cannot be inlined per call site).
func instrumentForSymbol(sym string) instrument.CanonicalInstrument {
	base := strings.ToUpper(strings.TrimSuffix(sym, "usdt"))
	return mustPerpOrSpot(base, instrument.ContractPerpetual)
}

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
