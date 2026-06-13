package derive_test

import (
	"testing"
	"time"

	appderive "internal/application/derive"
	"internal/domain/observation"
)

func tpoTrade(t *testing.T, price, qty string, ts time.Time) observation.ObservationTrade {
	t.Helper()
	return observation.ObservationTrade{
		Source:     "binancef",
		Instrument: vpInstrument(t), // reuses helper from volume_profile_sampler_test.go
		Price:      price,
		Quantity:   qty,
		BuyerMaker: false,
		TradeID:    price + qty + ts.String(),
		Timestamp:  ts,
	}
}

func TestTPOSampler_PeriodsLevelsAndFinalize(t *testing.T) {
	// 1h window, 10-min periods → 6 periods (A..F).
	s := appderive.NewTPOSampler("binancef", time.Hour, "1", 600, 0)
	open := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC) // 3600s-aligned

	// Period A (offset 0): price 65000.
	if _, did := s.AddTrade(tpoTrade(t, "65000", "1", open)); did {
		t.Fatal("unexpected early finalize on first trade")
	}
	// Period B (offset 700s): price 65000 (same level, new period) + 65010.
	s.AddTrade(tpoTrade(t, "65000", "1", open.Add(700*time.Second)))
	s.AddTrade(tpoTrade(t, "65010", "1", open.Add(700*time.Second)))

	// Crossing into the next window finalizes the first.
	tp, did := s.AddTrade(tpoTrade(t, "65000", "1", open.Add(time.Hour+10*time.Second)))
	if !did {
		t.Fatal("expected finalize when crossing the window boundary")
	}

	if prob := tp.Validate(); prob != nil {
		t.Fatalf("finalized TPO failed validation: %v", prob)
	}
	if !tp.Final {
		t.Error("finalized profile should have Final=true")
	}
	if tp.TradeCount != 3 {
		t.Errorf("trade_count = %d, want 3", tp.TradeCount)
	}
	if !tp.OpenTime.Equal(open) {
		t.Errorf("open_time = %v, want %v", tp.OpenTime, open)
	}

	// Periods: A and B present.
	if len(tp.Periods) != 2 {
		t.Fatalf("periods = %d, want 2 (A,B)", len(tp.Periods))
	}
	if tp.Periods[0].Letter != "A" || tp.Periods[1].Letter != "B" {
		t.Errorf("period letters = %q,%q, want A,B", tp.Periods[0].Letter, tp.Periods[1].Letter)
	}
	// Period B saw 65000 and 65010.
	if tp.Periods[1].HighPrice != "65010" || tp.Periods[1].LowPrice != "65000" {
		t.Errorf("period B high/low = %s/%s, want 65010/65000", tp.Periods[1].HighPrice, tp.Periods[1].LowPrice)
	}

	// Levels ascending: 65000 (AB, 2), 65010 (B, 1).
	if len(tp.Levels) != 2 {
		t.Fatalf("levels = %d, want 2", len(tp.Levels))
	}
	if tp.Levels[0].PriceLevel != "65000" || tp.Levels[0].Letters != "AB" || tp.Levels[0].Count != 2 {
		t.Errorf("level[0] = %+v, want {65000, AB, 2}", tp.Levels[0])
	}
	if tp.Levels[1].PriceLevel != "65010" || tp.Levels[1].Letters != "B" || tp.Levels[1].Count != 1 {
		t.Errorf("level[1] = %+v, want {65010, B, 1}", tp.Levels[1])
	}

	// POC is the most-touched level.
	if tp.POCPrice != "65000" {
		t.Errorf("poc = %s, want 65000", tp.POCPrice)
	}
}

func TestTPOSampler_EmptyUntilTrade(t *testing.T) {
	s := appderive.NewTPOSampler("binancef", time.Hour, "1", 600, 0)
	if _, ok := s.Snapshot(); ok {
		t.Error("snapshot should report inactive before any trade")
	}
	if s.Active() {
		t.Error("sampler should be inactive before any trade")
	}
}
