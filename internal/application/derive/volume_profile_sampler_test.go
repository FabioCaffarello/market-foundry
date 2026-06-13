package derive_test

import (
	"testing"
	"time"

	appderive "internal/application/derive"
	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/domain/observation"
)

func vpInstrument(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("New: %v", prob)
	}
	return inst
}

func vpTrade(t *testing.T, price, qty string, buyerMaker bool, ts time.Time) observation.ObservationTrade {
	t.Helper()
	return observation.ObservationTrade{
		Source:     "binancef",
		Instrument: vpInstrument(t),
		Price:      price,
		Quantity:   qty,
		BuyerMaker: buyerMaker,
		TradeID:    price + qty,
		Timestamp:  ts,
	}
}

func TestVolumeProfileSampler_BinsAndFinalizes(t *testing.T) {
	s := appderive.NewVolumeProfileSampler("binancef", time.Minute, "10", 0)
	base := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)

	// Window 1: two trades into bucket 65000, one into 65010.
	s.AddTrade(vpTrade(t, "65000.50", "1.0", true, base))                 // buy 65000.50 → bucket 65000
	s.AddTrade(vpTrade(t, "65007", "2.0", false, base.Add(time.Second)))  // sell 130014 → bucket 65000
	s.AddTrade(vpTrade(t, "65012", "1.0", true, base.Add(2*time.Second))) // buy 65012 → bucket 65010

	// Trade in next window finalizes window 1.
	finalized, did := s.AddTrade(vpTrade(t, "65000", "1.0", true, base.Add(time.Minute)))
	if !did {
		t.Fatal("expected finalize at window boundary")
	}
	if !finalized.Final {
		t.Error("finalized profile must have Final=true")
	}
	if prob := finalized.Validate(); prob != nil {
		t.Fatalf("finalized profile invalid: %v", prob)
	}
	if len(finalized.Buckets) != 2 {
		t.Fatalf("expected 2 price buckets, got %d", len(finalized.Buckets))
	}
	// Ascending price order.
	if finalized.Buckets[0].PriceLevel != "65000" || finalized.Buckets[1].PriceLevel != "65010" {
		t.Errorf("buckets not in ascending order: %+v", finalized.Buckets)
	}
	if finalized.TradeCount != 3 {
		t.Errorf("trade count = %d, want 3", finalized.TradeCount)
	}
	// Bucket 65000: buy 65000.50, sell 130014.00.
	if finalized.Buckets[0].BuyVolume != "65000.50000000" {
		t.Errorf("bucket 65000 buy = %q", finalized.Buckets[0].BuyVolume)
	}
	if finalized.Buckets[0].SellVolume != "130014.00000000" {
		t.Errorf("bucket 65000 sell = %q", finalized.Buckets[0].SellVolume)
	}
}

func TestVolumeProfileSampler_Snapshot(t *testing.T) {
	s := appderive.NewVolumeProfileSampler("binancef", time.Minute, "10", 0)
	if _, ok := s.Snapshot(); ok {
		t.Error("inactive sampler must not snapshot")
	}
	s.AddTrade(vpTrade(t, "65000", "1.0", true, time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)))
	snap, ok := s.Snapshot()
	if !ok {
		t.Fatal("active sampler must snapshot")
	}
	if snap.Final {
		t.Error("snapshot must be interim (Final=false)")
	}
	if len(snap.Buckets) != 1 {
		t.Errorf("expected 1 bucket, got %d", len(snap.Buckets))
	}
}

// Overload: with a tiny cap, new price levels beyond the cap are
// dropped (L3), but existing buckets keep accumulating — bounded.
func TestVolumeProfileSampler_OverloadBoundsBuckets(t *testing.T) {
	s := appderive.NewVolumeProfileSampler("binancef", time.Hour, "1", 3) // cap 3
	base := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)

	// 5 distinct price levels; cap is 3 → only 3 buckets admitted.
	for i, p := range []string{"100", "200", "300", "400", "500"} {
		s.AddTrade(vpTrade(t, p, "1.0", true, base.Add(time.Duration(i)*time.Second)))
	}
	// Re-hit an existing bucket — must still accumulate.
	s.AddTrade(vpTrade(t, "100", "2.0", true, base.Add(10*time.Second)))

	snap, _ := s.Snapshot()
	if len(snap.Buckets) != 3 {
		t.Fatalf("expected 3 buckets (cap), got %d", len(snap.Buckets))
	}
	if snap.Overload != insights.OverloadL3 {
		t.Errorf("overload = %d, want L3 (at cap)", snap.Overload)
	}
	// Bucket 100 accumulated both hits: 100*1 + 100*2 = 300.
	if snap.Buckets[0].PriceLevel != "100" || snap.Buckets[0].BuyVolume != "300.00000000" {
		t.Errorf("bucket 100 should accumulate re-hits: %+v", snap.Buckets[0])
	}
}
