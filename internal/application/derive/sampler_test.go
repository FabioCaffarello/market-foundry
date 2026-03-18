package derive_test

import (
	"testing"
	"time"

	"internal/application/derive"
	"internal/domain/observation"
)

func makeTrade(price, qty string, ts time.Time) observation.ObservationTrade {
	return observation.ObservationTrade{
		Source:     "binancef",
		Symbol:     "btcusdt",
		Price:      price,
		Quantity:   qty,
		TradeID:    "1",
		BuyerMaker: false,
		Timestamp:  ts,
	}
}

func TestCandleSampler_SingleWindow(t *testing.T) {
	sampler := derive.NewCandleSampler("binancef", "btcusdt", 60*time.Second)

	base := time.Date(2026, 3, 16, 12, 0, 5, 0, time.UTC) // 5 seconds into the minute

	_, did := sampler.AddTrade(makeTrade("100.00", "1.0", base))
	if did {
		t.Fatal("should not finalize on first trade")
	}

	_, did = sampler.AddTrade(makeTrade("102.00", "2.0", base.Add(10*time.Second)))
	if did {
		t.Fatal("should not finalize within same window")
	}

	_, did = sampler.AddTrade(makeTrade("99.00", "0.5", base.Add(30*time.Second)))
	if did {
		t.Fatal("should not finalize within same window")
	}

	snap, ok := sampler.Snapshot()
	if !ok {
		t.Fatal("expected active sampler")
	}
	if snap.Final {
		t.Fatal("snapshot should not be final")
	}
	if snap.TradeCount != 3 {
		t.Fatalf("expected 3 trades, got %d", snap.TradeCount)
	}
	if snap.Open != "100.00000000" {
		t.Fatalf("expected open 100.00000000, got %s", snap.Open)
	}
	if snap.High != "102.00000000" {
		t.Fatalf("expected high 102.00000000, got %s", snap.High)
	}
	if snap.Low != "99.00000000" {
		t.Fatalf("expected low 99.00000000, got %s", snap.Low)
	}
	if snap.Close != "99.00000000" {
		t.Fatalf("expected close 99.00000000, got %s", snap.Close)
	}
}

func TestCandleSampler_WindowRollover(t *testing.T) {
	sampler := derive.NewCandleSampler("binancef", "btcusdt", 60*time.Second)

	// Window 1: 12:00:00 - 12:01:00
	base := time.Date(2026, 3, 16, 12, 0, 10, 0, time.UTC)
	sampler.AddTrade(makeTrade("100.00", "1.0", base))
	sampler.AddTrade(makeTrade("105.00", "2.0", base.Add(20*time.Second)))

	// This trade belongs to the next window — should finalize window 1.
	nextWindow := time.Date(2026, 3, 16, 12, 1, 5, 0, time.UTC)
	candle, did := sampler.AddTrade(makeTrade("110.00", "1.0", nextWindow))
	if !did {
		t.Fatal("expected finalization on window rollover")
	}
	if !candle.Final {
		t.Fatal("finalized candle should be final")
	}
	if candle.TradeCount != 2 {
		t.Fatalf("expected 2 trades in finalized candle, got %d", candle.TradeCount)
	}
	if candle.Open != "100.00000000" {
		t.Fatalf("expected open 100.00000000, got %s", candle.Open)
	}
	if candle.High != "105.00000000" {
		t.Fatalf("expected high 105.00000000, got %s", candle.High)
	}

	expectedOpen := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	if !candle.OpenTime.Equal(expectedOpen) {
		t.Fatalf("expected open time %v, got %v", expectedOpen, candle.OpenTime)
	}
	expectedClose := time.Date(2026, 3, 16, 12, 1, 0, 0, time.UTC)
	if !candle.CloseTime.Equal(expectedClose) {
		t.Fatalf("expected close time %v, got %v", expectedClose, candle.CloseTime)
	}

	// The new trade should be in the new window.
	snap, ok := sampler.Snapshot()
	if !ok {
		t.Fatal("expected active sampler after rollover")
	}
	if snap.TradeCount != 1 {
		t.Fatalf("expected 1 trade in new window, got %d", snap.TradeCount)
	}
	if snap.Open != "110.00000000" {
		t.Fatalf("expected new window open 110.00000000, got %s", snap.Open)
	}
}

func TestCandleSampler_WindowFor(t *testing.T) {
	sampler := derive.NewCandleSampler("binancef", "btcusdt", 60*time.Second)

	ts := time.Date(2026, 3, 16, 12, 0, 45, 0, time.UTC)
	openTime, closeTime := sampler.WindowFor(ts)

	expectedOpen := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	expectedClose := time.Date(2026, 3, 16, 12, 1, 0, 0, time.UTC)

	if !openTime.Equal(expectedOpen) {
		t.Fatalf("expected open %v, got %v", expectedOpen, openTime)
	}
	if !closeTime.Equal(expectedClose) {
		t.Fatalf("expected close %v, got %v", expectedClose, closeTime)
	}
}

func TestCandleSampler_FiveMinuteWindow(t *testing.T) {
	sampler := derive.NewCandleSampler("binancef", "btcusdt", 300*time.Second)

	// Window: 12:00:00 - 12:05:00
	base := time.Date(2026, 3, 16, 12, 0, 10, 0, time.UTC)
	sampler.AddTrade(makeTrade("100.00", "1.0", base))
	sampler.AddTrade(makeTrade("105.00", "2.0", base.Add(2*time.Minute)))

	// Still within 5-minute window.
	_, did := sampler.AddTrade(makeTrade("103.00", "1.5", base.Add(4*time.Minute)))
	if did {
		t.Fatal("should not finalize within 5-minute window")
	}

	// Crosses into next 5-minute window.
	nextWindow := time.Date(2026, 3, 16, 12, 5, 5, 0, time.UTC)
	candle, did := sampler.AddTrade(makeTrade("110.00", "1.0", nextWindow))
	if !did {
		t.Fatal("expected finalization on 5-minute window rollover")
	}
	if candle.Timeframe != 300 {
		t.Fatalf("expected timeframe 300, got %d", candle.Timeframe)
	}
	if candle.TradeCount != 3 {
		t.Fatalf("expected 3 trades, got %d", candle.TradeCount)
	}

	expectedOpen := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	if !candle.OpenTime.Equal(expectedOpen) {
		t.Fatalf("expected open time %v, got %v", expectedOpen, candle.OpenTime)
	}
	expectedClose := time.Date(2026, 3, 16, 12, 5, 0, 0, time.UTC)
	if !candle.CloseTime.Equal(expectedClose) {
		t.Fatalf("expected close time %v, got %v", expectedClose, candle.CloseTime)
	}
}

func TestCandleSampler_MultiTimeframeIndependence(t *testing.T) {
	sampler60 := derive.NewCandleSampler("binancef", "btcusdt", 60*time.Second)
	sampler300 := derive.NewCandleSampler("binancef", "btcusdt", 300*time.Second)

	// Same trade fed to both samplers (like fan-out in SourceScopeActor).
	base := time.Date(2026, 3, 16, 12, 0, 10, 0, time.UTC)
	trade1 := makeTrade("100.00", "1.0", base)
	sampler60.AddTrade(trade1)
	sampler300.AddTrade(trade1)

	// Trade in next 1-minute window but same 5-minute window.
	trade2 := makeTrade("105.00", "2.0", base.Add(55*time.Second))
	candle60, did60 := sampler60.AddTrade(trade2)
	_, did300 := sampler300.AddTrade(trade2)

	if !did60 {
		t.Fatal("60s sampler should finalize on window rollover")
	}
	if did300 {
		t.Fatal("300s sampler should NOT finalize yet")
	}
	if candle60.Timeframe != 60 {
		t.Fatalf("expected 60s timeframe, got %d", candle60.Timeframe)
	}

	// Snapshot of 300s sampler shows both trades accumulated.
	snap300, ok := sampler300.Snapshot()
	if !ok {
		t.Fatal("300s sampler should be active")
	}
	if snap300.TradeCount != 2 {
		t.Fatalf("expected 2 trades in 300s window, got %d", snap300.TradeCount)
	}
}

func TestCandleSampler_DomainValidation(t *testing.T) {
	sampler := derive.NewCandleSampler("binancef", "btcusdt", 60*time.Second)

	base := time.Date(2026, 3, 16, 12, 0, 5, 0, time.UTC)
	sampler.AddTrade(makeTrade("100.00", "1.0", base))

	snap, ok := sampler.Snapshot()
	if !ok {
		t.Fatal("expected active sampler")
	}
	if prob := snap.Validate(); prob != nil {
		t.Fatalf("snapshot should produce valid candle, got: %v", prob)
	}
}
