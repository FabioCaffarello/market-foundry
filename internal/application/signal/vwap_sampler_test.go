package signal

import (
	"math"
	"strconv"
	"testing"
	"time"

	domsignal "internal/domain/signal"
)

func TestVWAPSampler_WarmUp(t *testing.T) {
	s := NewVWAPSampler("binancef", "btcusdt", 300)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Feed 19 candles (period-1) — should not produce a signal.
	for i := 0; i < 19; i++ {
		price := strconv.FormatFloat(100.0+float64(i)*0.5, 'f', 4, 64)
		volume := strconv.FormatFloat(1000.0+float64(i)*10.0, 'f', 4, 64)
		sig, ok := s.AddCandle(price, volume, ts.Add(time.Duration(i)*time.Minute))
		if ok {
			t.Fatalf("expected no signal at candle %d, got %+v", i+1, sig)
		}
	}

	// 20th candle (period reached) — should produce first signal.
	sig, ok := s.AddCandle("109.5000", "1190.0000", ts.Add(19*time.Minute))
	if !ok {
		t.Fatal("expected signal after 20 candles")
	}
	if sig.Type != "vwap" {
		t.Errorf("expected type vwap, got %s", sig.Type)
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

func TestVWAPSampler_ConstantPrices(t *testing.T) {
	s := NewVWAPSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// With constant prices, VWAP = price → deviation = 0.
	for i := 0; i < 30; i++ {
		sig, ok := s.AddCandle("42.0000", "100.0000", ts.Add(time.Duration(i)*time.Minute))
		if ok {
			dev, _ := strconv.ParseFloat(sig.Value, 64)
			if math.Abs(dev) > 1e-8 {
				t.Errorf("expected deviation ≈ 0 for constant prices, got %f at candle %d", dev, i+1)
			}
		}
	}
}

func TestVWAPSampler_PriceAboveVWAP(t *testing.T) {
	s := NewVWAPSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Feed 20 flat candles to warm up.
	for i := 0; i < 20; i++ {
		s.AddCandle("100.0000", "1000.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	// Feed a sustained rise with constant volume — price should exceed VWAP.
	var lastDeviation float64
	for i := 0; i < 20; i++ {
		price := strconv.FormatFloat(100.0+float64(i+1)*5.0, 'f', 4, 64)
		sig, ok := s.AddCandle(price, "1000.0000", ts.Add(time.Duration(20+i)*time.Minute))
		if ok {
			d, _ := strconv.ParseFloat(sig.Value, 64)
			lastDeviation = d
		}
	}

	if lastDeviation <= 0 {
		t.Errorf("expected positive deviation after sustained rise, got %f", lastDeviation)
	}
}

func TestVWAPSampler_PriceBelowVWAP(t *testing.T) {
	s := NewVWAPSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Feed 20 flat candles to warm up.
	for i := 0; i < 20; i++ {
		s.AddCandle("100.0000", "1000.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	// Feed a sustained drop — price should be below VWAP.
	var lastDeviation float64
	for i := 0; i < 20; i++ {
		price := strconv.FormatFloat(100.0-float64(i+1)*3.0, 'f', 4, 64)
		sig, ok := s.AddCandle(price, "1000.0000", ts.Add(time.Duration(20+i)*time.Minute))
		if ok {
			d, _ := strconv.ParseFloat(sig.Value, 64)
			lastDeviation = d
		}
	}

	if lastDeviation >= 0 {
		t.Errorf("expected negative deviation after sustained fall, got %f", lastDeviation)
	}
}

func TestVWAPSampler_VolumeWeighting(t *testing.T) {
	s := NewVWAPSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// First 19 candles: low price with low volume.
	for i := 0; i < 19; i++ {
		s.AddCandle("50.0000", "10.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	// 20th candle: high price with massive volume — VWAP should be pulled toward 200.
	sig, ok := s.AddCandle("200.0000", "10000.0000", ts.Add(19*time.Minute))
	if !ok {
		t.Fatal("expected signal after 20 candles")
	}

	vwap, _ := strconv.ParseFloat(sig.Metadata["vwap"], 64)
	// With 19×(50×10) + 1×(200×10000) = 9500 + 2000000 = 2009500
	// totalVolume = 19×10 + 10000 = 10190
	// VWAP ≈ 2009500 / 10190 ≈ 197.2
	// VWAP should be much closer to 200 than to 50 due to volume weighting.
	if vwap < 150 {
		t.Errorf("expected VWAP pulled toward high-volume candle, got %f", vwap)
	}
}

func TestVWAPSampler_ZeroVolume(t *testing.T) {
	s := NewVWAPSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// All zero volume — VWAP = 0, deviation = 0.
	for i := 0; i < 25; i++ {
		sig, ok := s.AddCandle("100.0000", "0.0000", ts.Add(time.Duration(i)*time.Minute))
		if ok {
			dev, _ := strconv.ParseFloat(sig.Value, 64)
			if math.Abs(dev) > 1e-8 {
				t.Errorf("expected zero deviation with zero volume, got %f at candle %d", dev, i+1)
			}
		}
	}
}

func TestVWAPSampler_Metadata(t *testing.T) {
	s := NewVWAPSampler("binancef", "ethusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Warm up + produce one signal.
	var sig domsignal.Signal
	for i := 0; i < 21; i++ {
		price := strconv.FormatFloat(100.0+float64(i)*0.1, 'f', 4, 64)
		volume := strconv.FormatFloat(500.0+float64(i)*5.0, 'f', 4, 64)
		got, ok := s.AddCandle(price, volume, ts.Add(time.Duration(i)*time.Minute))
		if ok {
			sig = got
		}
	}

	requiredKeys := []string{"period", "vwap", "close", "total_volume", "deviation"}
	for _, key := range requiredKeys {
		if _, ok := sig.Metadata[key]; !ok {
			t.Errorf("missing metadata key %q", key)
		}
	}

	if sig.Metadata["period"] != "20" {
		t.Errorf("expected period=20, got %s", sig.Metadata["period"])
	}
}

func TestVWAPSampler_InvalidPrice(t *testing.T) {
	s := NewVWAPSampler("binancef", "btcusdt", 60)
	ts := time.Now()
	_, ok := s.AddCandle("not-a-number", "100.0000", ts)
	if ok {
		t.Error("expected false for invalid price")
	}
}

func TestVWAPSampler_InvalidVolume(t *testing.T) {
	s := NewVWAPSampler("binancef", "btcusdt", 60)
	ts := time.Now()
	_, ok := s.AddCandle("100.0000", "bad-volume", ts)
	if ok {
		t.Error("expected false for invalid volume")
	}
}

func TestVWAPSampler_Validate(t *testing.T) {
	s := NewVWAPSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Warm up.
	for i := 0; i < 20; i++ {
		s.AddCandle("100.0000", "500.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	sig, ok := s.AddCandle("105.0000", "600.0000", ts.Add(20*time.Minute))
	if !ok {
		t.Fatal("expected signal")
	}
	if prob := sig.Validate(); prob != nil {
		t.Errorf("signal validation failed: %s", prob.Message)
	}
}

func TestVWAPSampler_MultiSymbol(t *testing.T) {
	btc := NewVWAPSampler("binancef", "btcusdt", 300)
	eth := NewVWAPSampler("binancef", "ethusdt", 300)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 21; i++ {
		btcPrice := strconv.FormatFloat(40000.0+float64(i)*100.0, 'f', 4, 64)
		ethPrice := strconv.FormatFloat(2500.0+float64(i)*10.0, 'f', 4, 64)
		vol := strconv.FormatFloat(1000.0+float64(i)*50.0, 'f', 4, 64)

		btcSig, btcOk := btc.AddCandle(btcPrice, vol, ts.Add(time.Duration(i)*time.Minute))
		ethSig, ethOk := eth.AddCandle(ethPrice, vol, ts.Add(time.Duration(i)*time.Minute))

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

func TestVWAPSampler_ContinuousProduction(t *testing.T) {
	s := NewVWAPSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// After warm-up, every candle must produce a signal.
	signalCount := 0
	for i := 0; i < 40; i++ {
		price := strconv.FormatFloat(100.0+float64(i)*0.3, 'f', 4, 64)
		volume := strconv.FormatFloat(500.0+float64(i)*10.0, 'f', 4, 64)
		_, ok := s.AddCandle(price, volume, ts.Add(time.Duration(i)*time.Minute))
		if ok {
			signalCount++
		}
	}

	// 40 candles - 19 warmup = 21 expected signals (first signal at candle 20).
	expected := 40 - 20 + 1
	if signalCount != expected {
		t.Errorf("expected %d signals after warm-up, got %d", expected, signalCount)
	}
}

func TestVWAPSampler_RollingWindow(t *testing.T) {
	s := NewVWAPSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Feed 20 candles at price=50, then 20 more at price=100.
	// After 40 candles, the rolling window should contain only the last 20 (all at 100).
	for i := 0; i < 20; i++ {
		s.AddCandle("50.0000", "100.0000", ts.Add(time.Duration(i)*time.Minute))
	}
	var lastSig domsignal.Signal
	for i := 0; i < 20; i++ {
		sig, ok := s.AddCandle("100.0000", "100.0000", ts.Add(time.Duration(20+i)*time.Minute))
		if ok {
			lastSig = sig
		}
	}

	// After the window fully rotates, VWAP ≈ 100, deviation ≈ 0.
	dev, _ := strconv.ParseFloat(lastSig.Value, 64)
	if math.Abs(dev) > 1e-6 {
		t.Errorf("expected deviation ≈ 0 after full window rotation, got %f", dev)
	}
}
