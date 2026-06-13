package insights_test

import (
	"testing"
	"time"

	"internal/domain/insights"
)

func TestPeriodLetter(t *testing.T) {
	cases := []struct {
		idx  int
		want string
	}{
		{0, "A"}, {1, "B"}, {23, "X"}, {24, ""}, {-1, ""},
	}
	for _, tc := range cases {
		if got := insights.PeriodLetter(tc.idx); got != tc.want {
			t.Errorf("PeriodLetter(%d) = %q, want %q", tc.idx, got, tc.want)
		}
	}
}

func sampleTPO(t *testing.T) insights.TPOProfile {
	t.Helper()
	open := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	return insights.TPOProfile{
		Source:        "binancef",
		Instrument:    btcUSDTPerp(t),
		Timeframe:     3600,
		BucketSize:    "10",
		PeriodSeconds: 600,
		Periods: []insights.TPOPeriod{
			{Letter: "A", StartTime: open, EndTime: open.Add(10 * time.Minute), HighPrice: "65020", LowPrice: "65000"},
			{Letter: "B", StartTime: open.Add(10 * time.Minute), EndTime: open.Add(20 * time.Minute), HighPrice: "65040", LowPrice: "65010"},
		},
		Levels: []insights.TPOLevel{
			{PriceLevel: "65000", Letters: "A", Count: 1},
			{PriceLevel: "65010", Letters: "AB", Count: 2},
			{PriceLevel: "65020", Letters: "AB", Count: 2},
			{PriceLevel: "65030", Letters: "B", Count: 1},
		},
		POCPrice:   "65010",
		TradeCount: 9,
		Overload:   insights.OverloadL0,
		OpenTime:   open,
		CloseTime:  open.Add(time.Hour),
		Final:      true,
	}
}

func TestTPOProfile_VenueSymbol(t *testing.T) {
	if got := sampleTPO(t).VenueSymbol(); got != "btcusdt" {
		t.Errorf("VenueSymbol() = %q, want btcusdt", got)
	}
}

func TestTPOProfile_Validate_OK(t *testing.T) {
	if prob := sampleTPO(t).Validate(); prob != nil {
		t.Fatalf("Validate() unexpected problem: %v", prob)
	}
}

func TestTPOProfile_Validate_Rejections(t *testing.T) {
	t.Run("zero period_seconds", func(t *testing.T) {
		tp := sampleTPO(t)
		tp.PeriodSeconds = 0
		if tp.Validate() == nil {
			t.Error("expected rejection for period_seconds=0")
		}
	})
	t.Run("too many periods", func(t *testing.T) {
		tp := sampleTPO(t)
		tp.Periods = make([]insights.TPOPeriod, insights.TPOMaxPeriods+1)
		for i := range tp.Periods {
			tp.Periods[i] = insights.TPOPeriod{Letter: "A", HighPrice: "1", LowPrice: "1"}
		}
		if tp.Validate() == nil {
			t.Error("expected rejection for period count over the A..X cap")
		}
	})
	t.Run("incomplete level", func(t *testing.T) {
		tp := sampleTPO(t)
		tp.Levels = append(tp.Levels, insights.TPOLevel{PriceLevel: "65040", Letters: "", Count: 0})
		if tp.Validate() == nil {
			t.Error("expected rejection for incomplete level")
		}
	})
}

func TestPointOfControl(t *testing.T) {
	// Ascending levels; the max-count level wins, ties resolve to the
	// lowest price (first encountered ascending).
	levels := []insights.TPOLevel{
		{PriceLevel: "100", Count: 1},
		{PriceLevel: "110", Count: 2},
		{PriceLevel: "120", Count: 2},
		{PriceLevel: "130", Count: 1},
	}
	if got := insights.PointOfControl(levels); got != "110" {
		t.Errorf("PointOfControl = %q, want 110 (lowest of the tied max)", got)
	}
	if got := insights.PointOfControl(nil); got != "" {
		t.Errorf("PointOfControl(nil) = %q, want empty", got)
	}
}

func TestValueArea(t *testing.T) {
	// price:count — POC=120(5), total=12, 70% → ceil(8.4)=9.
	// Expand from 120: +130(3)→8, +110(2)→10 ≥ 9 → VA=[110,130].
	levels := []insights.TPOLevel{
		{PriceLevel: "100", Count: 1},
		{PriceLevel: "110", Count: 2},
		{PriceLevel: "120", Count: 5},
		{PriceLevel: "130", Count: 3},
		{PriceLevel: "140", Count: 1},
	}
	high, low := insights.ValueArea(levels, insights.DefaultValueAreaFraction)
	if high != "130" || low != "110" {
		t.Errorf("ValueArea = [%s, %s], want [110, 130]", low, high)
	}
}

func TestInitialBalanceAndRange(t *testing.T) {
	periods := []insights.TPOPeriod{
		{Letter: "A", HighPrice: "65020", LowPrice: "65000"},
		{Letter: "B", HighPrice: "65040", LowPrice: "65010"},
		{Letter: "C", HighPrice: "65060", LowPrice: "64990"},
	}
	ibHigh, ibLow := insights.InitialBalance(periods, 2)
	if ibHigh != "65040" || ibLow != "65000" {
		t.Errorf("InitialBalance(2) = [%s, %s], want [65000, 65040]", ibLow, ibHigh)
	}
	rHigh, rLow := insights.PriceRange(periods)
	if rHigh != "65060" || rLow != "64990" {
		t.Errorf("PriceRange = [%s, %s], want [64990, 65060]", rLow, rHigh)
	}
}
