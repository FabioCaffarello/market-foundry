package strategy_test

import (
	"testing"
	"time"

	appstrategy "internal/application/strategy"
	"internal/domain/instrument"
)

var btcUSDTSpot = func() instrument.CanonicalInstrument {
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractSpot)
	if prob != nil {
		panic("test setup: BTC/USDT-spot: " + prob.Message)
	}
	return inst
}()

// TestEntryResolvers_NewForInstrument_Passthrough exercises the H-6.c.1
// pass-through constructors for each of the 3 strategy entry resolvers.
// See internal/application/signal/instrument_passthrough_test.go for
// the rationale and canary contract.
func TestEntryResolvers_NewForInstrument_Passthrough(t *testing.T) {
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	t.Run("MeanReversion", func(t *testing.T) {
		r := appstrategy.NewMeanReversionEntryResolverForInstrument("binancef", btcUSDTSpot, 60)
		s, ok := r.Resolve("rsi_oversold", "triggered", "0.85", "high", "rsi below 30", 60, ts)
		if !ok {
			t.Fatal("Resolve returned ok=false")
		}
		if s.Instrument != btcUSDTSpot {
			t.Errorf("Instrument = %+v, want %+v (pass-through broken)", s.Instrument, btcUSDTSpot)
		}
	})

	t.Run("TrendFollowing", func(t *testing.T) {
		r := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTSpot, 60)
		s, ok := r.Resolve("ema_crossover", "triggered", "0.85", "high", "bullish crossover", 60, ts)
		if !ok {
			t.Fatal("Resolve returned ok=false")
		}
		if s.Instrument != btcUSDTSpot {
			t.Errorf("Instrument = %+v, want %+v (pass-through broken)", s.Instrument, btcUSDTSpot)
		}
	})

	t.Run("SqueezeBreakout", func(t *testing.T) {
		r := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTSpot, 60)
		s, ok := r.Resolve("bollinger_squeeze", "triggered", "0.85", "high", "squeeze detected", 60, ts)
		if !ok {
			t.Fatal("Resolve returned ok=false")
		}
		if s.Instrument != btcUSDTSpot {
			t.Errorf("Instrument = %+v, want %+v (pass-through broken)", s.Instrument, btcUSDTSpot)
		}
	})
}
