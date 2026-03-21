package signal_test

import (
	"strconv"
	"testing"
	"time"

	appsignal "internal/application/signal"
)

func TestBollingerSampler_WarmUp(t *testing.T) {
	s := appsignal.NewBollingerSampler("binancef", "btcusdt", 300)
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	// Feed 19 prices — should NOT produce a signal (need period=20).
	for i := 0; i < 19; i++ {
		price := strconv.FormatFloat(100.0+float64(i), 'f', 2, 64)
		_, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
		if ok {
			t.Fatalf("expected no signal at candle %d", i+1)
		}
	}

	// 20th price: should produce first signal.
	sig, ok := s.AddClose("119.00", ts.Add(19*time.Minute))
	if !ok {
		t.Fatal("expected signal after period candles")
	}
	if sig.Type != "bollinger" {
		t.Fatalf("expected type bollinger, got %s", sig.Type)
	}
	if sig.Source != "binancef" || sig.Symbol != "btcusdt" || sig.Timeframe != 300 {
		t.Fatalf("unexpected scope: %s/%s/%d", sig.Source, sig.Symbol, sig.Timeframe)
	}
	if !sig.Final {
		t.Fatal("expected Final=true")
	}
}

func TestBollingerSampler_ConstantPrices(t *testing.T) {
	s := appsignal.NewBollingerSampler("binancef", "btcusdt", 60)
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	// All prices identical → stddev = 0 → bands collapse → %B = 0.5.
	var lastValue string
	for i := 0; i < 20; i++ {
		result, ok := s.AddClose("100.00", ts.Add(time.Duration(i)*time.Minute))
		if i < 19 {
			if ok {
				t.Fatalf("unexpected signal at candle %d", i+1)
			}
			continue
		}
		if !ok {
			t.Fatal("expected signal at candle 20")
		}
		lastValue = result.Value
	}

	pctB, err := strconv.ParseFloat(lastValue, 64)
	if err != nil {
		t.Fatalf("parse pctB value: %v", err)
	}
	if pctB != 0.5 {
		t.Fatalf("expected %%B=0.5 for constant prices (collapsed bands), got %f", pctB)
	}
}

func TestBollingerSampler_PriceAtUpperBand(t *testing.T) {
	s := appsignal.NewBollingerSampler("binancef", "btcusdt", 60)
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	// Feed 19 prices around 100, then push the 20th price high.
	// With a tight distribution, an extreme price should give %B > 1.
	for i := 0; i < 19; i++ {
		s.AddClose("100.00", ts.Add(time.Duration(i)*time.Minute))
	}
	sig, ok := s.AddClose("200.00", ts.Add(19*time.Minute))
	if !ok {
		t.Fatal("expected signal at candle 20")
	}

	pctB, _ := strconv.ParseFloat(sig.Value, 64)
	if pctB <= 1.0 {
		t.Fatalf("expected %%B > 1 for extreme high price, got %f", pctB)
	}
}

func TestBollingerSampler_RollingWindow(t *testing.T) {
	s := appsignal.NewBollingerSampler("binancef", "btcusdt", 300)
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	// Feed 21 prices — after 20th, window should drop oldest price.
	for i := 0; i < 20; i++ {
		s.AddClose(strconv.FormatFloat(100.0+float64(i), 'f', 2, 64), ts.Add(time.Duration(i)*time.Minute))
	}

	// 21st price should still produce a signal.
	sig, ok := s.AddClose("120.00", ts.Add(20*time.Minute))
	if !ok {
		t.Fatal("expected signal at candle 21")
	}

	// SMA should be based on prices 1-20 (dropped price 0=100), not 0-19.
	sma, err := strconv.ParseFloat(sig.Metadata["sma"], 64)
	if err != nil {
		t.Fatalf("parse sma: %v", err)
	}
	// Expected SMA: (101+102+...+119+120)/20 = (sum of 101..120)/20
	// = (101+120)*20/2 / 20 = 110.5
	if sma < 110.0 || sma > 111.0 {
		t.Fatalf("expected SMA around 110.5 (rolling window), got %f", sma)
	}
}

func TestBollingerSampler_Metadata(t *testing.T) {
	s := appsignal.NewBollingerSampler("binancef", "btcusdt", 300)
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 20; i++ {
		price := strconv.FormatFloat(100.0+float64(i%5), 'f', 2, 64)
		sig, ok := s.AddClose(price, ts.Add(time.Duration(i)*time.Minute))
		if i < 19 {
			continue
		}
		if !ok {
			t.Fatal("expected signal")
		}
		if sig.Metadata["period"] != "20" {
			t.Fatalf("expected period=20 in metadata, got %s", sig.Metadata["period"])
		}
		if sig.Metadata["k"] != "2.0" {
			t.Fatalf("expected k=2.0 in metadata, got %s", sig.Metadata["k"])
		}
		for _, key := range []string{"sma", "upper", "lower", "bandwidth"} {
			if _, exists := sig.Metadata[key]; !exists {
				t.Fatalf("expected %s in metadata", key)
			}
		}
	}
}

func TestBollingerSampler_InvalidPrice(t *testing.T) {
	s := appsignal.NewBollingerSampler("binancef", "btcusdt", 60)
	_, ok := s.AddClose("not-a-number", time.Now())
	if ok {
		t.Fatal("expected no signal for invalid price")
	}
}
