package derive

import (
	"testing"
	"time"

	"internal/domain/instrument"
	"internal/domain/observation"
)

func volumeBTCUSDTPerp() instrument.CanonicalInstrument {
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		panic("test setup: failed to build canonical BTC/USDT-perpetual: " + prob.Message)
	}
	return inst
}

func TestVolumeSampler_FirstWindow(t *testing.T) {
	t.Parallel()

	sampler := NewVolumeSampler("binancef", "btcusdt", 60*time.Second)
	ts := time.Unix(1700000000, 0)

	trade := observation.ObservationTrade{
		Source: "binancef", Instrument: volumeBTCUSDTPerp(),
		Price: "50000.00", Quantity: "1.5",
		TradeID: "1", BuyerMaker: true, Timestamp: ts,
	}

	_, didFinalize := sampler.AddTrade(trade)
	if didFinalize {
		t.Fatal("should not finalize on first trade")
	}
	if !sampler.Active() {
		t.Fatal("sampler should be active")
	}
}

func TestVolumeSampler_WindowRollover(t *testing.T) {
	t.Parallel()

	sampler := NewVolumeSampler("binancef", "btcusdt", 60*time.Second)

	ts1 := time.Unix(1700000010, 0) // window 0
	ts2 := time.Unix(1700000070, 0) // window 1 — triggers finalization

	sampler.AddTrade(observation.ObservationTrade{
		Source: "binancef", Instrument: volumeBTCUSDTPerp(),
		Price: "50000.00", Quantity: "2.0",
		TradeID: "1", BuyerMaker: true, Timestamp: ts1,
	})
	sampler.AddTrade(observation.ObservationTrade{
		Source: "binancef", Instrument: volumeBTCUSDTPerp(),
		Price: "50000.00", Quantity: "1.0",
		TradeID: "2", BuyerMaker: false, Timestamp: ts1,
	})

	vol, didFinalize := sampler.AddTrade(observation.ObservationTrade{
		Source: "binancef", Instrument: volumeBTCUSDTPerp(),
		Price: "51000.00", Quantity: "0.5",
		TradeID: "3", BuyerMaker: true, Timestamp: ts2,
	})

	if !didFinalize {
		t.Fatal("should finalize on window rollover")
	}
	if !vol.Final {
		t.Fatal("finalized volume should have Final=true")
	}
	if vol.TradeCount != 2 {
		t.Fatalf("expected 2 trades, got %d", vol.TradeCount)
	}
	// BuyVolume: 50000*2 = 100000, SellVolume: 50000*1 = 50000
	if vol.BuyVolume != "100000.00000000" {
		t.Fatalf("expected buy volume 100000.00000000, got %s", vol.BuyVolume)
	}
	if vol.SellVolume != "50000.00000000" {
		t.Fatalf("expected sell volume 50000.00000000, got %s", vol.SellVolume)
	}
	if vol.TotalVolume != "150000.00000000" {
		t.Fatalf("expected total volume 150000.00000000, got %s", vol.TotalVolume)
	}
	// VWAP: 150000 / 3.0 = 50000
	if vol.VWAP != "50000.00000000" {
		t.Fatalf("expected VWAP 50000.00000000, got %s", vol.VWAP)
	}
}

func TestVolumeSampler_VWAPWithMixedPrices(t *testing.T) {
	t.Parallel()

	sampler := NewVolumeSampler("binancef", "btcusdt", 60*time.Second)
	ts1 := time.Unix(1700000010, 0)
	ts2 := time.Unix(1700000070, 0)

	// Trade 1: price=40000, qty=1 → notional=40000
	sampler.AddTrade(observation.ObservationTrade{
		Source: "binancef", Instrument: volumeBTCUSDTPerp(),
		Price: "40000.00", Quantity: "1.0",
		TradeID: "1", BuyerMaker: true, Timestamp: ts1,
	})
	// Trade 2: price=60000, qty=1 → notional=60000
	sampler.AddTrade(observation.ObservationTrade{
		Source: "binancef", Instrument: volumeBTCUSDTPerp(),
		Price: "60000.00", Quantity: "1.0",
		TradeID: "2", BuyerMaker: false, Timestamp: ts1,
	})

	vol, _ := sampler.AddTrade(observation.ObservationTrade{
		Source: "binancef", Instrument: volumeBTCUSDTPerp(),
		Price: "50000.00", Quantity: "1.0",
		TradeID: "3", BuyerMaker: true, Timestamp: ts2,
	})

	// VWAP = (40000 + 60000) / (1 + 1) = 50000
	if vol.VWAP != "50000.00000000" {
		t.Fatalf("expected VWAP 50000.00000000, got %s", vol.VWAP)
	}
}

func TestVolumeSampler_WindowAlignment(t *testing.T) {
	t.Parallel()

	sampler := NewVolumeSampler("binancef", "btcusdt", 60*time.Second)
	ts := time.Unix(1700000045, 0)

	open, close := sampler.WindowFor(ts)
	expectedOpen := time.Unix(1700000040, 0).UTC()
	// 1700000040 / 60 = 28333333.xx → not aligned. Let me recalculate.
	// 1700000045 / 60 = 28333334.08... → floor = 28333334 → 28333334*60 = 1700000040
	expectedClose := expectedOpen.Add(60 * time.Second)

	if open != expectedOpen {
		t.Fatalf("expected open %v, got %v", expectedOpen, open)
	}
	if close != expectedClose {
		t.Fatalf("expected close %v, got %v", expectedClose, close)
	}
}
