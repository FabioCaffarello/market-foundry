package settings

import "testing"

// ── S400: Source-Segment mapping tests ───────────────────────────────

func TestSourceForSegmentFutures(t *testing.T) {
	if src := SourceForSegment(MarketSegmentFutures); src != "binancef" {
		t.Fatalf("expected binancef, got %q", src)
	}
}

func TestSourceForSegmentSpot(t *testing.T) {
	if src := SourceForSegment(MarketSegmentSpot); src != "binances" {
		t.Fatalf("expected binances, got %q", src)
	}
}

func TestSourceForSegmentUnknown(t *testing.T) {
	if src := SourceForSegment("options"); src != "" {
		t.Fatalf("expected empty for unknown segment, got %q", src)
	}
}

func TestSegmentForSourceBinancef(t *testing.T) {
	if seg := SegmentForSource("binancef"); seg != MarketSegmentFutures {
		t.Fatalf("expected futures, got %q", seg)
	}
}

func TestSegmentForSourceBinances(t *testing.T) {
	if seg := SegmentForSource("binances"); seg != MarketSegmentSpot {
		t.Fatalf("expected spot, got %q", seg)
	}
}

func TestSegmentForSourceUnknown(t *testing.T) {
	if seg := SegmentForSource("unknown"); seg != "" {
		t.Fatalf("expected empty for unknown source, got %q", seg)
	}
}

func TestUnifiedConfigBothSegmentsValidates(t *testing.T) {
	cfg := VenueConfig{
		DryRun: boolPtr(true),
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {
				Enabled: true,
				Adapter: VenueTypeBinanceSpotTestnet,
			},
			MarketSegmentFutures: {
				Enabled: true,
				Adapter: VenueTypeBinanceFuturesTestnet,
			},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("unified config should validate: %s", prob.Message)
	}

	segs := cfg.EnabledSegments()
	if len(segs) != 2 {
		t.Fatalf("expected 2 enabled segments, got %d", len(segs))
	}
	// Canonical order: spot before futures.
	if segs[0] != MarketSegmentSpot {
		t.Fatalf("expected spot first, got %q", segs[0])
	}
	if segs[1] != MarketSegmentFutures {
		t.Fatalf("expected futures second, got %q", segs[1])
	}
}

func TestUnifiedConfigSourceSegmentRoundTrip(t *testing.T) {
	for _, seg := range []MarketSegment{MarketSegmentSpot, MarketSegmentFutures} {
		src := SourceForSegment(seg)
		if src == "" {
			t.Fatalf("no source for segment %q", seg)
		}
		roundTrip := SegmentForSource(src)
		if roundTrip != seg {
			t.Fatalf("round-trip failed: %q → %q → %q", seg, src, roundTrip)
		}
	}
}

func boolPtr(b bool) *bool { return &b }
