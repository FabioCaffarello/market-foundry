package insights_test

import (
	"testing"
	"time"

	"internal/domain/insights"
	"internal/domain/instrument"
)

func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("New: %v", prob)
	}
	return inst
}

// ── BucketLevel (binning) ────────────────────────────────────────

func TestBucketLevel_Canonical(t *testing.T) {
	cases := []struct {
		price, width, want string
	}{
		{"65000.50", "10", "65000"},
		{"65007", "10", "65000"},
		{"65000.50", "0.5", "65000.5"},
		{"65000.49", "0.5", "65000"},
		{"100", "25", "100"},
		{"249.99", "25", "225"},
		{"0.123", "0.05", "0.1"},
	}
	for _, tc := range cases {
		got, prob := insights.BucketLevel(tc.price, tc.width)
		if prob != nil {
			t.Errorf("BucketLevel(%s,%s): %v", tc.price, tc.width, prob)
			continue
		}
		if got != tc.want {
			t.Errorf("BucketLevel(%s,%s) = %q, want %q", tc.price, tc.width, got, tc.want)
		}
	}
}

// Determinism: same inputs → identical output across repeated calls
// (replay-safety; no float drift).
func TestBucketLevel_Deterministic(t *testing.T) {
	first, _ := insights.BucketLevel("65000.50", "0.5")
	for i := 0; i < 100; i++ {
		got, _ := insights.BucketLevel("65000.50", "0.5")
		if got != first {
			t.Fatalf("non-deterministic: %q vs %q", got, first)
		}
	}
}

func TestBucketLevel_Rejections(t *testing.T) {
	cases := []struct{ price, width string }{
		{"notnum", "10"},
		{"100", "0"},
		{"100", "-5"},
		{"100", "notnum"},
	}
	for _, tc := range cases {
		if _, prob := insights.BucketLevel(tc.price, tc.width); prob == nil {
			t.Errorf("BucketLevel(%s,%s) accepted, want rejection", tc.price, tc.width)
		}
	}
}

// ── ClassifyOverload ─────────────────────────────────────────────

func TestClassifyOverload_Levels(t *testing.T) {
	cap := 100
	cases := []struct {
		count int
		want  insights.OverloadLevel
	}{
		{0, insights.OverloadL0},
		{49, insights.OverloadL0},
		{50, insights.OverloadL1},
		{79, insights.OverloadL1},
		{80, insights.OverloadL2},
		{99, insights.OverloadL2},
		{100, insights.OverloadL3},
		{150, insights.OverloadL3},
	}
	for _, tc := range cases {
		if got := insights.ClassifyOverload(tc.count, cap); got != tc.want {
			t.Errorf("ClassifyOverload(%d,%d) = %d, want %d", tc.count, cap, got, tc.want)
		}
	}
}

func TestClassifyOverload_NoCap(t *testing.T) {
	if got := insights.ClassifyOverload(99999, 0); got != insights.OverloadL0 {
		t.Errorf("no-cap must be L0, got %d", got)
	}
}

func TestOverloadLevel_AdmitsNewLevel(t *testing.T) {
	for _, l := range []insights.OverloadLevel{insights.OverloadL0, insights.OverloadL1, insights.OverloadL2} {
		if !l.AdmitsNewLevel() {
			t.Errorf("L%d must admit new levels", l)
		}
	}
	if insights.OverloadL3.AdmitsNewLevel() {
		t.Error("L3 must NOT admit new levels (existing buckets still accumulate)")
	}
}

func TestOverloadLevel_Validate(t *testing.T) {
	if insights.OverloadLevel(7).Validate() == nil {
		t.Error("out-of-range level must fail validation")
	}
	if insights.OverloadL2.Validate() != nil {
		t.Error("L2 must validate")
	}
}

// ── VolumeProfile.Validate ───────────────────────────────────────

func validProfile(t *testing.T) insights.VolumeProfile {
	t.Helper()
	return insights.VolumeProfile{
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		BucketSize: "10",
		Buckets: []insights.PriceBucket{
			{PriceLevel: "65000", BuyVolume: "1200.5", SellVolume: "800.0"},
			{PriceLevel: "65010", BuyVolume: "300.0", SellVolume: "450.25"},
		},
		TradeCount: 42,
		Overload:   insights.OverloadL0,
		OpenTime:   time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC),
		CloseTime:  time.Date(2026, 6, 13, 12, 1, 0, 0, time.UTC),
		Final:      true,
	}
}

func TestVolumeProfile_ValidPasses(t *testing.T) {
	if prob := validProfile(t).Validate(); prob != nil {
		t.Fatalf("valid profile failed: %v", prob)
	}
}

// An empty window (no trades) is valid.
func TestVolumeProfile_EmptyBucketsValid(t *testing.T) {
	vp := validProfile(t)
	vp.Buckets = nil
	vp.TradeCount = 0
	if prob := vp.Validate(); prob != nil {
		t.Errorf("empty-window profile must be valid: %v", prob)
	}
}

func TestVolumeProfile_Rejections(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*insights.VolumeProfile)
	}{
		{"no_source", func(vp *insights.VolumeProfile) { vp.Source = "" }},
		{"zero_instrument", func(vp *insights.VolumeProfile) { vp.Instrument = instrument.CanonicalInstrument{} }},
		{"bad_timeframe", func(vp *insights.VolumeProfile) { vp.Timeframe = 0 }},
		{"no_bucket_size", func(vp *insights.VolumeProfile) { vp.BucketSize = "" }},
		{"bad_overload", func(vp *insights.VolumeProfile) { vp.Overload = insights.OverloadLevel(9) }},
		{"incomplete_bucket", func(vp *insights.VolumeProfile) { vp.Buckets[0].BuyVolume = "" }},
		{"close_before_open", func(vp *insights.VolumeProfile) { vp.CloseTime = vp.OpenTime.Add(-time.Minute) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vp := validProfile(t)
			tc.mutate(&vp)
			if prob := vp.Validate(); prob == nil {
				t.Errorf("%s: expected rejection", tc.name)
			}
		})
	}
}

func TestVolumeProfile_VenueSymbol(t *testing.T) {
	if got := validProfile(t).VenueSymbol(); got != "btcusdt" {
		t.Errorf("VenueSymbol = %q, want btcusdt", got)
	}
}
