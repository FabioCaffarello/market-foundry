package derive_test

import (
	"testing"
	"time"

	appderive "internal/application/derive"
	"internal/domain/observation"
)

func cvTrade(t *testing.T, source, price, qty string, ts time.Time) observation.ObservationTrade {
	t.Helper()
	return observation.ObservationTrade{
		Source:     source,
		Instrument: vpInstrument(t), // BTC/USDT-perpetual — same canonical across venues
		Price:      price,
		Quantity:   qty,
		TradeID:    source + price + ts.String(),
		Timestamp:  ts,
	}
}

func TestCrossVenueFusion_FusesAcrossVenuesAndFinalizes(t *testing.T) {
	f := appderive.NewCrossVenueFusion(time.Minute)
	open := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC) // 60s-aligned

	// Window 1: binancef + bybitf trade the same canonical instrument.
	if _, did := f.AddTrade(cvTrade(t, "binancef", "65000", "1", open)); did {
		t.Fatal("unexpected early finalize")
	}
	f.AddTrade(cvTrade(t, "bybitf", "65010", "1", open.Add(10*time.Second)))
	f.AddTrade(cvTrade(t, "binancef", "65020", "1", open.Add(20*time.Second)))

	// A trade for the same instrument in the next window finalizes window 1.
	snap, did := f.AddTrade(cvTrade(t, "binancef", "65030", "1", open.Add(70*time.Second)))
	if !did {
		t.Fatal("expected finalize on window boundary")
	}
	if prob := snap.Validate(); prob != nil {
		t.Fatalf("snapshot failed validation: %v", prob)
	}

	if !snap.Final || !snap.OpenTime.Equal(open) {
		t.Errorf("unexpected window: final=%v open=%v", snap.Final, snap.OpenTime)
	}
	if snap.TradeCount != 3 {
		t.Errorf("trade_count = %d, want 3", snap.TradeCount)
	}

	// Venues ascending by name: binancef, bybitf.
	if len(snap.Venues) != 2 || snap.Venues[0].Venue != "binancef" || snap.Venues[1].Venue != "bybitf" {
		t.Fatalf("venues wrong: %+v", snap.Venues)
	}
	bf := snap.Venues[0]
	if bf.TradeCount != 2 || bf.LastPrice != "65020" || bf.HighPrice != "65020" || bf.LowPrice != "65000" || bf.Notional != "130020.00000000" {
		t.Errorf("binancef row wrong: %+v", bf)
	}
	bb := snap.Venues[1]
	if bb.TradeCount != 1 || bb.LastPrice != "65010" || bb.Notional != "65010.00000000" {
		t.Errorf("bybitf row wrong: %+v", bb)
	}

	// Consolidated: last prices 65020 (binancef) / 65010 (bybitf).
	if snap.SpreadAbs != "10.00000000" || snap.MidPrice != "65015.00000000" {
		t.Errorf("spread/mid = %s / %s, want 10 / 65015", snap.SpreadAbs, snap.MidPrice)
	}
	if snap.DominantVenue != "binancef" {
		t.Errorf("dominant = %s, want binancef (more notional)", snap.DominantVenue)
	}
}

func TestCrossVenueFusion_SingleVenueWindow(t *testing.T) {
	f := appderive.NewCrossVenueFusion(time.Minute)
	open := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	f.AddTrade(cvTrade(t, "binancef", "65000", "2", open))
	snap, did := f.AddTrade(cvTrade(t, "binancef", "65000", "1", open.Add(70*time.Second)))
	if !did {
		t.Fatal("expected finalize")
	}
	if len(snap.Venues) != 1 || snap.DominantVenue != "binancef" {
		t.Errorf("single-venue window wrong: %+v", snap.Venues)
	}
	if snap.SpreadAbs != "0.00000000" {
		t.Errorf("single-venue spread = %s, want 0", snap.SpreadAbs)
	}
}
