package signal

import (
	"math"
	"strconv"
	"testing"
	"time"

	domsignal "internal/domain/signal"
)

func TestMACDSampler_WarmUp(t *testing.T) {
	s := NewMACDSampler("binancef", "btcusdt", 300)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Feed 33 prices (slowPeriod + signalPeriod - 2 = 33) — should not produce a signal.
	for i := 0; i < 33; i++ {
		price := strconv.FormatFloat(100.0+float64(i)*0.5, 'f', 4, 64)
		sig, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
		if ok {
			t.Fatalf("expected no signal at candle %d, got %+v", i+1, sig)
		}
	}

	// 34th price (slowPeriod + signalPeriod - 1 reached) — should produce first signal.
	sig, ok := s.AddClose("116.5000", ts.Add(33*time.Minute))
	if !ok {
		t.Fatal("expected signal after 34 candles")
	}
	if sig.Type != "macd" {
		t.Errorf("expected type macd, got %s", sig.Type)
	}
	if sig.Source != "binancef" {
		t.Errorf("expected source binancef, got %s", sig.Source)
	}
	if sig.Symbol != "btcusdt" {
		t.Errorf("expected symbol btcusdt, got %s", sig.Symbol)
	}
	if sig.Timeframe != 300 {
		t.Errorf("expected timeframe 300, got %d", sig.Timeframe)
	}
}

func TestMACDSampler_BullishDivergence(t *testing.T) {
	s := NewMACDSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Feed 34 flat prices to warm up.
	for i := 0; i < 34; i++ {
		s.AddClose("100.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	// Feed sustained rising prices — MACD line should exceed signal line → positive histogram.
	var lastHistogram float64
	for i := 0; i < 40; i++ {
		price := strconv.FormatFloat(100.0+float64(i+1)*3.0, 'f', 4, 64)
		sig, ok := s.AddClose(price, ts.Add(time.Duration(34+i)*time.Minute))
		if ok {
			h, _ := strconv.ParseFloat(sig.Value, 64)
			lastHistogram = h
		}
	}

	if lastHistogram <= 0 {
		t.Errorf("expected positive histogram after sustained rise, got %f", lastHistogram)
	}
}

func TestMACDSampler_BearishDivergence(t *testing.T) {
	s := NewMACDSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Feed 34 flat prices to warm up.
	for i := 0; i < 34; i++ {
		s.AddClose("100.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	// Feed sustained falling prices — MACD line should be below signal line → negative histogram.
	var lastHistogram float64
	for i := 0; i < 40; i++ {
		price := strconv.FormatFloat(100.0-float64(i+1)*3.0, 'f', 4, 64)
		sig, ok := s.AddClose(price, ts.Add(time.Duration(34+i)*time.Minute))
		if ok {
			h, _ := strconv.ParseFloat(sig.Value, 64)
			lastHistogram = h
		}
	}

	if lastHistogram >= 0 {
		t.Errorf("expected negative histogram after sustained fall, got %f", lastHistogram)
	}
}

func TestMACDSampler_ConstantPrices(t *testing.T) {
	s := NewMACDSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// With constant prices, both EMAs converge to the same value → MACD = 0.
	for i := 0; i < 50; i++ {
		sig, ok := s.AddClose("42.0000", ts.Add(time.Duration(i)*time.Minute))
		if ok {
			h, _ := strconv.ParseFloat(sig.Value, 64)
			if math.Abs(h) > 1e-6 {
				t.Errorf("expected histogram ≈ 0 for constant prices, got %f at candle %d", h, i+1)
			}
		}
	}
}

func TestMACDSampler_Metadata(t *testing.T) {
	s := NewMACDSampler("binancef", "ethusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Warm up + produce one signal.
	var sig domsignal.Signal
	for i := 0; i < 35; i++ {
		price := strconv.FormatFloat(100.0+float64(i)*0.1, 'f', 4, 64)
		got, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
		if ok {
			sig = got
		}
	}

	requiredKeys := []string{"fast_period", "slow_period", "signal_period", "fast_ema", "slow_ema", "macd_line", "signal_line", "histogram"}
	for _, key := range requiredKeys {
		if _, ok := sig.Metadata[key]; !ok {
			t.Errorf("missing metadata key %q", key)
		}
	}

	if sig.Metadata["fast_period"] != "12" {
		t.Errorf("expected fast_period=12, got %s", sig.Metadata["fast_period"])
	}
	if sig.Metadata["slow_period"] != "26" {
		t.Errorf("expected slow_period=26, got %s", sig.Metadata["slow_period"])
	}
	if sig.Metadata["signal_period"] != "9" {
		t.Errorf("expected signal_period=9, got %s", sig.Metadata["signal_period"])
	}
}

func TestMACDSampler_InvalidPrice(t *testing.T) {
	s := NewMACDSampler("binancef", "btcusdt", 60)
	ts := time.Now()
	_, ok := s.AddClose("not-a-number", ts)
	if ok {
		t.Error("expected false for invalid price")
	}
}

func TestMACDSampler_Validate(t *testing.T) {
	s := NewMACDSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Warm up.
	for i := 0; i < 34; i++ {
		s.AddClose("100.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	sig, ok := s.AddClose("105.0000", ts.Add(34*time.Minute))
	if !ok {
		t.Fatal("expected signal")
	}
	if prob := sig.Validate(); prob != nil {
		t.Errorf("signal validation failed: %s", prob.Message)
	}
}

func TestMACDSampler_MultiSymbol(t *testing.T) {
	btc := NewMACDSampler("binancef", "btcusdt", 300)
	eth := NewMACDSampler("binancef", "ethusdt", 300)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 35; i++ {
		btcPrice := strconv.FormatFloat(40000.0+float64(i)*100.0, 'f', 4, 64)
		ethPrice := strconv.FormatFloat(2500.0+float64(i)*10.0, 'f', 4, 64)
		btcSig, btcOk := btc.AddClose(btcPrice, ts.Add(time.Duration(i)*time.Minute))
		ethSig, ethOk := eth.AddClose(ethPrice, ts.Add(time.Duration(i)*time.Minute))

		if btcOk && ethOk {
			if btcSig.Symbol != "btcusdt" {
				t.Errorf("expected btcusdt, got %s", btcSig.Symbol)
			}
			if ethSig.Symbol != "ethusdt" {
				t.Errorf("expected ethusdt, got %s", ethSig.Symbol)
			}
			// Partition keys must be isolated.
			if btcSig.PartitionKey() == ethSig.PartitionKey() {
				t.Error("partition keys must differ across symbols")
			}
			// Deduplication keys must be isolated.
			if btcSig.DeduplicationKey() == ethSig.DeduplicationKey() {
				t.Error("deduplication keys must differ across symbols")
			}
		}
	}
}

func TestMACDSampler_ContinuousProduction(t *testing.T) {
	s := NewMACDSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// After warm-up, every candle must produce a signal.
	signalCount := 0
	for i := 0; i < 60; i++ {
		price := strconv.FormatFloat(100.0+float64(i)*0.3, 'f', 4, 64)
		_, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
		if ok {
			signalCount++
		}
	}

	// 60 candles - 33 warmup = 27 expected signals (first signal at candle 34).
	// Candle 34 is the first signal, so 60 - 34 + 1 = 27 signals.
	expected := 60 - 34 + 1
	if signalCount != expected {
		t.Errorf("expected %d signals after warm-up, got %d", expected, signalCount)
	}
}
