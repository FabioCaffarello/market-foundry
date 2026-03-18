package derive_test

import (
	"testing"
	"time"

	"internal/application/derive"
	"internal/domain/observation"
)

func tradeBurstTrade(ts time.Time, qty string, buyerMaker bool) observation.ObservationTrade {
	return observation.ObservationTrade{
		Source:     "binancef",
		Symbol:     "btcusdt",
		Price:      "100.00",
		Quantity:   qty,
		TradeID:    "t1",
		BuyerMaker: buyerMaker,
		Timestamp:  ts,
	}
}

func TestTradeBurstSampler_FirstWindow(t *testing.T) {
	s := derive.NewTradeBurstSampler("binancef", "btcusdt", 60*time.Second)

	base := time.Unix(1710000000, 0)

	// First trade — no finalization.
	_, didFinalize := s.AddTrade(tradeBurstTrade(base, "1.0", true))
	if didFinalize {
		t.Fatal("first trade should not finalize")
	}

	// Second trade in same window.
	_, didFinalize = s.AddTrade(tradeBurstTrade(base.Add(10*time.Second), "2.0", false))
	if didFinalize {
		t.Fatal("second trade in same window should not finalize")
	}

	// Third trade in new window — finalizes first window.
	burst, didFinalize := s.AddTrade(tradeBurstTrade(base.Add(60*time.Second), "1.0", true))
	if !didFinalize {
		t.Fatal("trade in new window should finalize previous")
	}
	if burst.TradeCount != 2 {
		t.Fatalf("expected 2 trades, got %d", burst.TradeCount)
	}
	if !burst.Final {
		t.Fatal("expected Final=true")
	}
	if burst.Burst {
		t.Fatal("first window should not be burst (no baseline)")
	}
}

func TestTradeBurstSampler_BurstDetection(t *testing.T) {
	s := derive.NewTradeBurstSampler("binancef", "btcusdt", 60*time.Second)

	base := time.Unix(1710000000, 0)

	// Window 1: 2 trades.
	s.AddTrade(tradeBurstTrade(base, "1.0", true))
	s.AddTrade(tradeBurstTrade(base.Add(10*time.Second), "1.0", false))

	// Window 2: 5 trades (>2× window 1) — should be burst.
	s.AddTrade(tradeBurstTrade(base.Add(60*time.Second), "1.0", true))
	s.AddTrade(tradeBurstTrade(base.Add(61*time.Second), "1.0", false))
	s.AddTrade(tradeBurstTrade(base.Add(62*time.Second), "1.0", true))
	s.AddTrade(tradeBurstTrade(base.Add(63*time.Second), "1.0", false))
	s.AddTrade(tradeBurstTrade(base.Add(64*time.Second), "1.0", true))

	// Trigger window 2 finalization.
	burst, didFinalize := s.AddTrade(tradeBurstTrade(base.Add(120*time.Second), "1.0", true))
	if !didFinalize {
		t.Fatal("expected finalization")
	}
	if burst.TradeCount != 5 {
		t.Fatalf("expected 5 trades, got %d", burst.TradeCount)
	}
	if !burst.Burst {
		t.Fatal("expected Burst=true (5 > 2×2)")
	}
}

func TestTradeBurstSampler_NoBurstWhenBelowThreshold(t *testing.T) {
	s := derive.NewTradeBurstSampler("binancef", "btcusdt", 60*time.Second)

	base := time.Unix(1710000000, 0)

	// Window 1: 5 trades.
	for i := 0; i < 5; i++ {
		s.AddTrade(tradeBurstTrade(base.Add(time.Duration(i)*time.Second), "1.0", true))
	}

	// Window 2: 6 trades (< 2×5 = 10) — not burst.
	for i := 0; i < 6; i++ {
		s.AddTrade(tradeBurstTrade(base.Add(60*time.Second+time.Duration(i)*time.Second), "1.0", false))
	}

	// Trigger window 2 finalization.
	burst, didFinalize := s.AddTrade(tradeBurstTrade(base.Add(120*time.Second), "1.0", true))
	if !didFinalize {
		t.Fatal("expected finalization")
	}
	if burst.Burst {
		t.Fatal("expected Burst=false (6 < 2×5)")
	}
}

func TestTradeBurstSampler_BuySellVolumeSplit(t *testing.T) {
	s := derive.NewTradeBurstSampler("binancef", "btcusdt", 60*time.Second)

	base := time.Unix(1710000000, 0)

	// 2 buy trades (buyer maker), 1 sell trade.
	s.AddTrade(tradeBurstTrade(base, "1.0", true))        // buy: 100 * 1 = 100
	s.AddTrade(tradeBurstTrade(base.Add(5*time.Second), "2.0", true))  // buy: 100 * 2 = 200
	s.AddTrade(tradeBurstTrade(base.Add(10*time.Second), "3.0", false)) // sell: 100 * 3 = 300

	// Finalize.
	burst, didFinalize := s.AddTrade(tradeBurstTrade(base.Add(60*time.Second), "1.0", true))
	if !didFinalize {
		t.Fatal("expected finalization")
	}
	if burst.BuyVolume != "300.00000000" {
		t.Fatalf("expected buy volume 300.00000000, got %s", burst.BuyVolume)
	}
	if burst.SellVolume != "300.00000000" {
		t.Fatalf("expected sell volume 300.00000000, got %s", burst.SellVolume)
	}
}

func TestTradeBurstSampler_WindowAlignment(t *testing.T) {
	s := derive.NewTradeBurstSampler("binancef", "btcusdt", 60*time.Second)

	ts := time.Unix(1710000035, 0) // 35 seconds into a window
	openTime, closeTime := s.WindowFor(ts)

	expectedOpen := time.Unix(1710000000, 0).UTC()
	expectedClose := time.Unix(1710000060, 0).UTC()

	if openTime != expectedOpen {
		t.Fatalf("expected open %v, got %v", expectedOpen, openTime)
	}
	if closeTime != expectedClose {
		t.Fatalf("expected close %v, got %v", expectedClose, closeTime)
	}
}
