package signal_test

import (
	"strconv"
	"testing"
	"time"

	appsignal "internal/application/signal"
	"internal/domain/instrument"
	domainsignal "internal/domain/signal"
)

// btcUSDTSpot is the canonical fixture used across the H-6.c.1
// passthrough tests. Constructed once at init via IIFE; panic on
// invalid setup mirrors the precedent in
// internal/application/analyticalclient/get_composite_chain_test.go.
var btcUSDTSpot = func() instrument.CanonicalInstrument {
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractSpot)
	if prob != nil {
		panic("test setup: BTC/USDT-spot: " + prob.Message)
	}
	return inst
}()

// TestSamplers_NewForInstrument_Passthrough exercises the H-6.c.1
// pass-through constructors for each of the 6 signal samplers.
//
// The canary check is twofold:
//
//  1. The constructor accepts a canonical Instrument directly and
//     does not invoke instrumentFromBinding internally (verified by
//     the analyzer's anti-patterns scan over the production .go
//     files; this test verifies the runtime contract).
//  2. The Instrument carried by the produced signal matches the one
//     passed at construction, asserting zero source-string
//     reconstruction in between (the regression-shape from H-6.b'
//     commit 37f8ddd would surface as a zero Instrument here).
//
// Each sampler is fed enough data to emit its first signal; the
// signal's Instrument is asserted equal to btcUSDTSpot.
func TestSamplers_NewForInstrument_Passthrough(t *testing.T) {
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	t.Run("RSI", func(t *testing.T) {
		s := appsignal.NewRSISamplerForInstrument("binancef", btcUSDTSpot, 60)
		var last domainsignal.Signal
		var got bool
		for i := 0; i <= 14; i++ {
			price := strconv.FormatFloat(100.0+float64(i), 'f', 2, 64)
			sig, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
			if ok {
				last = sig
				got = true
			}
		}
		assertInstrumentPassthrough(t, "rsi", last, got)
	})

	t.Run("EMACrossover", func(t *testing.T) {
		s := appsignal.NewEMACrossoverSamplerForInstrument("binancef", btcUSDTSpot, 60)
		var last domainsignal.Signal
		var got bool
		for i := 0; i < 25; i++ {
			price := strconv.FormatFloat(100.0+float64(i), 'f', 2, 64)
			sig, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
			if ok {
				last = sig
				got = true
			}
		}
		assertInstrumentPassthrough(t, "ema_crossover", last, got)
	})

	t.Run("Bollinger", func(t *testing.T) {
		s := appsignal.NewBollingerSamplerForInstrument("binancef", btcUSDTSpot, 60)
		var last domainsignal.Signal
		var got bool
		for i := 0; i < 22; i++ {
			price := strconv.FormatFloat(100.0+float64(i), 'f', 2, 64)
			sig, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
			if ok {
				last = sig
				got = true
			}
		}
		assertInstrumentPassthrough(t, "bollinger", last, got)
	})

	t.Run("MACD", func(t *testing.T) {
		s := appsignal.NewMACDSamplerForInstrument("binancef", btcUSDTSpot, 60)
		var last domainsignal.Signal
		var got bool
		for i := 0; i < 40; i++ {
			price := strconv.FormatFloat(100.0+float64(i), 'f', 2, 64)
			sig, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
			if ok {
				last = sig
				got = true
			}
		}
		assertInstrumentPassthrough(t, "macd", last, got)
	})

	t.Run("VWAP", func(t *testing.T) {
		s := appsignal.NewVWAPSamplerForInstrument("binancef", btcUSDTSpot, 60)
		var last domainsignal.Signal
		var got bool
		for i := 0; i < 22; i++ {
			price := 100.0 + float64(i)
			sig, ok := s.AddCandle(
				strconv.FormatFloat(price, 'f', 2, 64),
				"100.0",
				ts.Add(time.Duration(i)*time.Minute),
			)
			if ok {
				last = sig
				got = true
			}
		}
		assertInstrumentPassthrough(t, "vwap", last, got)
	})

	t.Run("ATR", func(t *testing.T) {
		s := appsignal.NewATRSamplerForInstrument("binancef", btcUSDTSpot, 60)
		var last domainsignal.Signal
		var got bool
		for i := 0; i < 16; i++ {
			price := 100.0 + float64(i)
			sig, ok := s.AddCandle(
				strconv.FormatFloat(price+1, 'f', 2, 64),
				strconv.FormatFloat(price-1, 'f', 2, 64),
				strconv.FormatFloat(price, 'f', 2, 64),
				ts.Add(time.Duration(i)*time.Minute),
			)
			if ok {
				last = sig
				got = true
			}
		}
		assertInstrumentPassthrough(t, "atr", last, got)
	})
}

func assertInstrumentPassthrough(t *testing.T, sigType string, sig domainsignal.Signal, got bool) {
	t.Helper()
	if !got {
		t.Fatalf("%s sampler emitted no signal during warm-up", sigType)
	}
	if sig.Instrument != btcUSDTSpot {
		t.Errorf("%s sampler: Instrument = %+v, want %+v (pass-through broken)", sigType, sig.Instrument, btcUSDTSpot)
	}
	if sig.Instrument.Base != "BTC" || sig.Instrument.Quote != "USDT" {
		t.Errorf("%s sampler: Instrument fields wrong: base=%s quote=%s", sigType, sig.Instrument.Base, sig.Instrument.Quote)
	}
}
