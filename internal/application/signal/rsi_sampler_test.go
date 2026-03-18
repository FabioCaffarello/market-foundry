package signal_test

import (
	"strconv"
	"testing"
	"time"

	appsignal "internal/application/signal"
)

func TestRSISampler_WarmUp(t *testing.T) {
	s := appsignal.NewRSISampler("binancef", "btcusdt", 300)
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	// Feed 14 prices — should NOT produce a signal (need period+1 = 15).
	for i := 0; i < 14; i++ {
		price := strconv.FormatFloat(100.0+float64(i), 'f', 2, 64)
		_, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
		if ok {
			t.Fatalf("expected no signal at candle %d", i+1)
		}
	}

	// 15th price: should produce first signal.
	sig, ok := s.AddClose("114.00", ts.Add(14*time.Minute))
	if !ok {
		t.Fatal("expected signal after period+1 candles")
	}
	if sig.Type != "rsi" {
		t.Fatalf("expected type rsi, got %s", sig.Type)
	}
	if sig.Source != "binancef" || sig.Symbol != "btcusdt" || sig.Timeframe != 300 {
		t.Fatalf("unexpected scope: %s/%s/%d", sig.Source, sig.Symbol, sig.Timeframe)
	}
	if !sig.Final {
		t.Fatal("expected Final=true")
	}
}

func TestRSISampler_AllGains(t *testing.T) {
	s := appsignal.NewRSISampler("binancef", "btcusdt", 60)
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	// 15 monotonically increasing prices → all gains, zero losses → RSI = 100.
	for i := 0; i <= 14; i++ {
		price := strconv.FormatFloat(100.0+float64(i), 'f', 2, 64)
		sig, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
		if i < 14 {
			if ok {
				t.Fatalf("unexpected signal at candle %d", i+1)
			}
			continue
		}
		if !ok {
			t.Fatal("expected signal at candle 15")
		}

		rsi, err := strconv.ParseFloat(sig.Value, 64)
		if err != nil {
			t.Fatalf("parse rsi value: %v", err)
		}
		if rsi != 100.0 {
			t.Fatalf("expected RSI=100 for all gains, got %f", rsi)
		}
	}
}

func TestRSISampler_AllLosses(t *testing.T) {
	s := appsignal.NewRSISampler("binancef", "btcusdt", 60)
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	// 15 monotonically decreasing prices → all losses, zero gains → RSI = 0.
	for i := 0; i <= 14; i++ {
		price := strconv.FormatFloat(200.0-float64(i), 'f', 2, 64)
		sig, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
		if i < 14 {
			if ok {
				t.Fatalf("unexpected signal at candle %d", i+1)
			}
			continue
		}
		if !ok {
			t.Fatal("expected signal at candle 15")
		}

		rsi, err := strconv.ParseFloat(sig.Value, 64)
		if err != nil {
			t.Fatalf("parse rsi value: %v", err)
		}
		if rsi != 0.0 {
			t.Fatalf("expected RSI=0 for all losses, got %f", rsi)
		}
	}
}

func TestRSISampler_SmoothedUpdate(t *testing.T) {
	s := appsignal.NewRSISampler("binancef", "btcusdt", 300)
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	// Feed 15 prices for warm-up.
	prices := []string{
		"44.00", "44.34", "44.09", "43.61", "44.33",
		"44.83", "45.10", "45.42", "45.84", "46.08",
		"45.89", "46.03", "45.61", "46.28", "46.28",
	}
	var lastSig string
	for i, p := range prices {
		sig, ok := s.AddClose(p, ts.Add(time.Duration(i)*time.Minute))
		if i == 14 && !ok {
			t.Fatal("expected signal at candle 15")
		}
		if ok {
			lastSig = sig.Value
		}
	}

	// 16th price — smoothed update should still produce a signal.
	sig, ok := s.AddClose("46.00", ts.Add(15*time.Minute))
	if !ok {
		t.Fatal("expected signal at candle 16")
	}

	rsi, _ := strconv.ParseFloat(sig.Value, 64)
	if rsi <= 0 || rsi >= 100 {
		t.Fatalf("expected RSI between 0 and 100, got %f", rsi)
	}

	// RSI should have changed from the warm-up value.
	if sig.Value == lastSig {
		t.Fatal("expected RSI to change after smoothed update")
	}
}

func TestRSISampler_InvalidPrice(t *testing.T) {
	s := appsignal.NewRSISampler("binancef", "btcusdt", 60)
	_, ok := s.AddClose("not-a-number", time.Now())
	if ok {
		t.Fatal("expected no signal for invalid price")
	}
}

func TestRSISampler_Metadata(t *testing.T) {
	s := appsignal.NewRSISampler("binancef", "btcusdt", 300)
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	// Feed period+1 prices.
	for i := 0; i <= 14; i++ {
		price := strconv.FormatFloat(100.0+float64(i%3), 'f', 2, 64)
		sig, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
		if i < 14 {
			continue
		}
		if !ok {
			t.Fatal("expected signal")
		}
		if sig.Metadata["period"] != "14" {
			t.Fatalf("expected period=14 in metadata, got %s", sig.Metadata["period"])
		}
		if _, exists := sig.Metadata["avg_gain"]; !exists {
			t.Fatal("expected avg_gain in metadata")
		}
		if _, exists := sig.Metadata["avg_loss"]; !exists {
			t.Fatal("expected avg_loss in metadata")
		}
	}
}
