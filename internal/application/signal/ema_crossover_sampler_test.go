package signal

import (
	"strconv"
	"testing"
	"time"
)

func TestEMACrossoverSampler_WarmUp(t *testing.T) {
	s := NewEMACrossoverSamplerForInstrument("binancef", btcUSDTPerp, 300)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Feed slowPeriod-1 prices — should not produce a signal.
	for i := 0; i < 20; i++ {
		price := strconv.FormatFloat(100.0+float64(i)*0.5, 'f', 4, 64)
		sig, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
		if ok {
			t.Fatalf("expected no signal at candle %d, got %+v", i+1, sig)
		}
	}

	// 21st price (slowPeriod reached) — should produce first signal.
	sig, ok := s.AddClose("110.0000", ts.Add(20*time.Minute))
	if !ok {
		t.Fatal("expected signal after slowPeriod candles")
	}
	if sig.Type != "ema_crossover" {
		t.Errorf("expected type ema_crossover, got %s", sig.Type)
	}
	if sig.Source != "binancef" {
		t.Errorf("expected source binancef, got %s", sig.Source)
	}
	if sig.VenueSymbol() != "btcusdt" {
		t.Errorf("expected symbol btcusdt, got %s", sig.VenueSymbol())
	}
	if sig.Timeframe != 300 {
		t.Errorf("expected timeframe 300, got %d", sig.Timeframe)
	}
	if sig.Value != "bullish" && sig.Value != "bearish" && sig.Value != "neutral" {
		t.Errorf("unexpected value %q", sig.Value)
	}
	if sig.Metadata["fast_period"] != "9" {
		t.Errorf("expected fast_period=9, got %s", sig.Metadata["fast_period"])
	}
	if sig.Metadata["slow_period"] != "21" {
		t.Errorf("expected slow_period=21, got %s", sig.Metadata["slow_period"])
	}
}

func TestEMACrossoverSampler_BullishCrossover(t *testing.T) {
	s := NewEMACrossoverSamplerForInstrument("binancef", btcUSDTPerp, 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Feed 21 flat prices to warm up.
	for i := 0; i < 21; i++ {
		s.AddClose("100.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	// Feed a series of rising prices — fast EMA should rise faster than slow EMA.
	var lastSig string
	for i := 0; i < 30; i++ {
		price := strconv.FormatFloat(100.0+float64(i+1)*2.0, 'f', 4, 64)
		sig, ok := s.AddClose(price, ts.Add(time.Duration(21+i)*time.Minute))
		if ok {
			lastSig = sig.Value
		}
	}

	if lastSig != "bullish" {
		t.Errorf("expected bullish after sustained rise, got %s", lastSig)
	}
}

func TestEMACrossoverSampler_BearishCrossover(t *testing.T) {
	s := NewEMACrossoverSamplerForInstrument("binancef", btcUSDTPerp, 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Feed 21 flat prices to warm up.
	for i := 0; i < 21; i++ {
		s.AddClose("100.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	// Feed a series of falling prices — fast EMA should fall faster than slow EMA.
	var lastSig string
	for i := 0; i < 30; i++ {
		price := strconv.FormatFloat(100.0-float64(i+1)*2.0, 'f', 4, 64)
		sig, ok := s.AddClose(price, ts.Add(time.Duration(21+i)*time.Minute))
		if ok {
			lastSig = sig.Value
		}
	}

	if lastSig != "bearish" {
		t.Errorf("expected bearish after sustained fall, got %s", lastSig)
	}
}

func TestEMACrossoverSampler_InvalidPrice(t *testing.T) {
	s := NewEMACrossoverSamplerForInstrument("binancef", btcUSDTPerp, 60)
	ts := time.Now()
	_, ok := s.AddClose("not-a-number", ts)
	if ok {
		t.Error("expected false for invalid price")
	}
}

func TestEMACrossoverSampler_Validate(t *testing.T) {
	s := NewEMACrossoverSamplerForInstrument("binancef", btcUSDTPerp, 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Warm up.
	for i := 0; i < 21; i++ {
		s.AddClose("100.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	sig, ok := s.AddClose("105.0000", ts.Add(21*time.Minute))
	if !ok {
		t.Fatal("expected signal")
	}
	if prob := sig.Validate(); prob != nil {
		t.Errorf("signal validation failed: %s", prob.Message)
	}
}

func TestSMA(t *testing.T) {
	result := sma([]float64{10, 20, 30})
	if result != 20.0 {
		t.Errorf("expected 20.0, got %f", result)
	}

	result = sma(nil)
	if result != 0 {
		t.Errorf("expected 0, got %f", result)
	}
}
