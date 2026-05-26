package signal

import (
	"math"
	"strconv"
	"testing"
	"time"

	domsignal "internal/domain/signal"
)

func TestATRSampler_WarmUp(t *testing.T) {
	s := NewATRSampler("binancef", "btcusdt", 300)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Feed 14 candles (period + 1 - 1 = 14) — should not produce a signal.
	// First candle establishes prevClose, then 13 true ranges (< period 14).
	for i := 0; i < 14; i++ {
		high := strconv.FormatFloat(101.0+float64(i)*0.5, 'f', 4, 64)
		low := strconv.FormatFloat(99.0+float64(i)*0.5, 'f', 4, 64)
		close := strconv.FormatFloat(100.0+float64(i)*0.5, 'f', 4, 64)
		sig, ok := s.AddCandle(high, low, close, ts.Add(time.Duration(i)*time.Minute))
		if ok {
			t.Fatalf("expected no signal at candle %d, got %+v", i+1, sig)
		}
	}

	// 15th candle (period + 1 reached) — should produce first signal.
	sig, ok := s.AddCandle("108.0000", "106.0000", "107.0000", ts.Add(14*time.Minute))
	if !ok {
		t.Fatal("expected signal after 15 candles")
	}
	if sig.Type != "atr" {
		t.Errorf("expected type atr, got %s", sig.Type)
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
}

func TestATRSampler_HighVolatility(t *testing.T) {
	s := NewATRSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Warm up with stable candles (TR = 2.0 each).
	for i := 0; i < 15; i++ {
		s.AddCandle("101.0000", "99.0000", "100.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	// Feed high-volatility candles (TR >> 2.0).
	var lastATR float64
	for i := 0; i < 20; i++ {
		high := strconv.FormatFloat(110.0+float64(i)*2.0, 'f', 4, 64)
		low := strconv.FormatFloat(90.0-float64(i)*2.0, 'f', 4, 64)
		close := strconv.FormatFloat(100.0, 'f', 4, 64)
		sig, ok := s.AddCandle(high, low, close, ts.Add(time.Duration(15+i)*time.Minute))
		if ok {
			v, _ := strconv.ParseFloat(sig.Value, 64)
			lastATR = v
		}
	}

	// ATR should be substantially above the initial 2.0 baseline.
	if lastATR <= 5.0 {
		t.Errorf("expected elevated ATR after high-volatility candles, got %f", lastATR)
	}
}

func TestATRSampler_LowVolatility(t *testing.T) {
	s := NewATRSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Warm up with moderate candles (TR = 4.0).
	for i := 0; i < 15; i++ {
		s.AddCandle("102.0000", "98.0000", "100.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	// Feed narrow-range candles (TR ≈ 0.2).
	var lastATR float64
	for i := 0; i < 40; i++ {
		s.AddCandle("100.1000", "99.9000", "100.0000", ts.Add(time.Duration(15+i)*time.Minute))
	}
	sig, ok := s.AddCandle("100.1000", "99.9000", "100.0000", ts.Add(55*time.Minute))
	if ok {
		lastATR, _ = strconv.ParseFloat(sig.Value, 64)
	}

	// After sustained narrow ranges, ATR should contract toward 0.2.
	if lastATR >= 2.0 {
		t.Errorf("expected contracted ATR after low-volatility candles, got %f", lastATR)
	}
}

func TestATRSampler_ConstantPrices(t *testing.T) {
	s := NewATRSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// With constant high=low=close, true range is always 0 → ATR = 0.
	for i := 0; i < 30; i++ {
		sig, ok := s.AddCandle("42.0000", "42.0000", "42.0000", ts.Add(time.Duration(i)*time.Minute))
		if ok {
			v, _ := strconv.ParseFloat(sig.Value, 64)
			if math.Abs(v) > 1e-6 {
				t.Errorf("expected ATR ≈ 0 for constant prices, got %f at candle %d", v, i+1)
			}
		}
	}
}

func TestATRSampler_GapUp(t *testing.T) {
	s := NewATRSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Warm up with stable candles.
	for i := 0; i < 15; i++ {
		s.AddCandle("101.0000", "99.0000", "100.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	// Gap up: previous close was 100, new candle opens at 120.
	// TR = max(125-115, |125-100|, |115-100|) = max(10, 25, 15) = 25.
	sig, ok := s.AddCandle("125.0000", "115.0000", "120.0000", ts.Add(15*time.Minute))
	if !ok {
		t.Fatal("expected signal")
	}
	tr := sig.Metadata["true_range"]
	trVal, _ := strconv.ParseFloat(tr, 64)
	if math.Abs(trVal-25.0) > 0.01 {
		t.Errorf("expected true_range=25.0 for gap-up candle, got %s", tr)
	}
}

func TestATRSampler_Metadata(t *testing.T) {
	s := NewATRSampler("binancef", "ethusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	var sig domsignal.Signal
	for i := 0; i < 20; i++ {
		high := strconv.FormatFloat(101.0+float64(i)*0.1, 'f', 4, 64)
		low := strconv.FormatFloat(99.0+float64(i)*0.1, 'f', 4, 64)
		close := strconv.FormatFloat(100.0+float64(i)*0.1, 'f', 4, 64)
		got, ok := s.AddCandle(high, low, close, ts.Add(time.Duration(i)*time.Minute))
		if ok {
			sig = got
		}
	}

	requiredKeys := []string{"period", "atr", "true_range"}
	for _, key := range requiredKeys {
		if _, ok := sig.Metadata[key]; !ok {
			t.Errorf("missing metadata key %q", key)
		}
	}

	if sig.Metadata["period"] != "14" {
		t.Errorf("expected period=14, got %s", sig.Metadata["period"])
	}
}

func TestATRSampler_InvalidPrice(t *testing.T) {
	s := NewATRSampler("binancef", "btcusdt", 60)
	ts := time.Now()

	_, ok := s.AddCandle("not-a-number", "99.0000", "100.0000", ts)
	if ok {
		t.Error("expected false for invalid high")
	}

	_, ok = s.AddCandle("101.0000", "not-a-number", "100.0000", ts)
	if ok {
		t.Error("expected false for invalid low")
	}

	_, ok = s.AddCandle("101.0000", "99.0000", "not-a-number", ts)
	if ok {
		t.Error("expected false for invalid close")
	}
}

func TestATRSampler_Validate(t *testing.T) {
	s := NewATRSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Warm up.
	for i := 0; i < 15; i++ {
		s.AddCandle("101.0000", "99.0000", "100.0000", ts.Add(time.Duration(i)*time.Minute))
	}

	sig, ok := s.AddCandle("102.0000", "98.0000", "100.0000", ts.Add(15*time.Minute))
	if !ok {
		t.Fatal("expected signal")
	}
	if prob := sig.Validate(); prob != nil {
		t.Errorf("signal validation failed: %s", prob.Message)
	}
}

func TestATRSampler_MultiSymbol(t *testing.T) {
	btc := NewATRSampler("binancef", "btcusdt", 300)
	eth := NewATRSampler("binancef", "ethusdt", 300)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 20; i++ {
		btcHigh := strconv.FormatFloat(40100.0+float64(i)*100.0, 'f', 4, 64)
		btcLow := strconv.FormatFloat(39900.0+float64(i)*100.0, 'f', 4, 64)
		btcClose := strconv.FormatFloat(40000.0+float64(i)*100.0, 'f', 4, 64)
		ethHigh := strconv.FormatFloat(2510.0+float64(i)*10.0, 'f', 4, 64)
		ethLow := strconv.FormatFloat(2490.0+float64(i)*10.0, 'f', 4, 64)
		ethClose := strconv.FormatFloat(2500.0+float64(i)*10.0, 'f', 4, 64)

		btcSig, btcOk := btc.AddCandle(btcHigh, btcLow, btcClose, ts.Add(time.Duration(i)*time.Minute))
		ethSig, ethOk := eth.AddCandle(ethHigh, ethLow, ethClose, ts.Add(time.Duration(i)*time.Minute))

		if btcOk && ethOk {
			if btcSig.VenueSymbol() != "btcusdt" {
				t.Errorf("expected btcusdt, got %s", btcSig.VenueSymbol())
			}
			if ethSig.VenueSymbol() != "ethusdt" {
				t.Errorf("expected ethusdt, got %s", ethSig.VenueSymbol())
			}
			if btcSig.PartitionKey() == ethSig.PartitionKey() {
				t.Error("partition keys must differ across symbols")
			}
			if btcSig.DeduplicationKey() == ethSig.DeduplicationKey() {
				t.Error("deduplication keys must differ across symbols")
			}
		}
	}
}

func TestATRSampler_ContinuousProduction(t *testing.T) {
	s := NewATRSampler("binancef", "btcusdt", 60)
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// After warm-up, every candle must produce a signal.
	signalCount := 0
	for i := 0; i < 40; i++ {
		high := strconv.FormatFloat(101.0+float64(i)*0.3, 'f', 4, 64)
		low := strconv.FormatFloat(99.0+float64(i)*0.3, 'f', 4, 64)
		close := strconv.FormatFloat(100.0+float64(i)*0.3, 'f', 4, 64)
		_, ok := s.AddCandle(high, low, close, ts.Add(time.Duration(i)*time.Minute))
		if ok {
			signalCount++
		}
	}

	// 40 candles - 14 warmup = 26 expected signals (first signal at candle 15).
	expected := 40 - 15 + 1
	if signalCount != expected {
		t.Errorf("expected %d signals after warm-up, got %d", expected, signalCount)
	}
}

func TestATRSampler_TrueRangeCalculation(t *testing.T) {
	// Verify true range handles all three cases correctly.
	tests := []struct {
		name      string
		high      float64
		low       float64
		prevClose float64
		expected  float64
	}{
		{"normal range", 105.0, 95.0, 100.0, 10.0}, // high-low dominates
		{"gap up", 115.0, 110.0, 100.0, 15.0},      // |high-prevClose| dominates
		{"gap down", 90.0, 85.0, 100.0, 15.0},      // |low-prevClose| dominates
		{"no range", 100.0, 100.0, 100.0, 0.0},     // all equal
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trueRange(tt.high, tt.low, tt.prevClose)
			if math.Abs(got-tt.expected) > 1e-10 {
				t.Errorf("trueRange(%f, %f, %f) = %f, want %f", tt.high, tt.low, tt.prevClose, got, tt.expected)
			}
		})
	}
}
