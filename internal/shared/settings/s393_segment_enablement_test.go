package settings

import (
	"testing"
)

// ── S393/S399: VenueType.Segment() ──────────────────────────────────

func TestVenueTypeSegmentReturnsFuturesForBinanceFutures(t *testing.T) {
	if seg := VenueTypeBinanceFuturesTestnet.Segment(); seg != MarketSegmentFutures {
		t.Fatalf("expected futures, got %q", seg)
	}
}

func TestVenueTypeSegmentReturnsSpotForBinanceSpot(t *testing.T) {
	if seg := VenueTypeBinanceSpotTestnet.Segment(); seg != MarketSegmentSpot {
		t.Fatalf("expected spot, got %q", seg)
	}
}

func TestVenueTypeSegmentReturnsEmptyForPaper(t *testing.T) {
	if seg := VenueTypePaperSimulator.Segment(); seg != "" {
		t.Fatalf("expected empty, got %q", seg)
	}
}

func TestVenueTypeRequiresSegmentConfig(t *testing.T) {
	if !VenueTypeBinanceFuturesTestnet.RequiresSegmentConfig() {
		t.Fatal("futures testnet should require segment config")
	}
	if !VenueTypeBinanceSpotTestnet.RequiresSegmentConfig() {
		t.Fatal("spot testnet should require segment config")
	}
	if VenueTypePaperSimulator.RequiresSegmentConfig() {
		t.Fatal("paper_simulator should NOT require segment config")
	}
}

// ── S399: Unified segment config — helpers ──────────────────────────

func TestHasUnifiedSegmentsNilMap(t *testing.T) {
	v := VenueConfig{Type: VenueTypePaperSimulator}
	if v.HasUnifiedSegments() {
		t.Fatal("nil segments map should not be unified")
	}
}

func TestHasUnifiedSegmentsEmptyMap(t *testing.T) {
	v := VenueConfig{Segments: map[MarketSegment]*SegmentVenueConfig{}}
	if v.HasUnifiedSegments() {
		t.Fatal("empty segments map should not be unified")
	}
}

func TestHasUnifiedSegmentsWithEntries(t *testing.T) {
	v := VenueConfig{Segments: map[MarketSegment]*SegmentVenueConfig{
		MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
	}}
	if !v.HasUnifiedSegments() {
		t.Fatal("segments map with entries should be unified")
	}
}

func TestEnabledSegmentsCanonicalOrder(t *testing.T) {
	v := VenueConfig{Segments: map[MarketSegment]*SegmentVenueConfig{
		MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
	}}
	segs := v.EnabledSegments()
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segs))
	}
	if segs[0] != MarketSegmentSpot {
		t.Fatalf("first segment should be spot, got %s", segs[0])
	}
	if segs[1] != MarketSegmentFutures {
		t.Fatalf("second segment should be futures, got %s", segs[1])
	}
}

func TestEnabledSegmentsSkipsDisabled(t *testing.T) {
	v := VenueConfig{Segments: map[MarketSegment]*SegmentVenueConfig{
		MarketSegmentSpot:    {Enabled: false, Adapter: VenueTypeBinanceSpotTestnet},
		MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
	}}
	segs := v.EnabledSegments()
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if segs[0] != MarketSegmentFutures {
		t.Fatalf("expected futures, got %s", segs[0])
	}
}

func TestIsSegmentEnabledTrue(t *testing.T) {
	v := VenueConfig{Segments: map[MarketSegment]*SegmentVenueConfig{
		MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
	}}
	if !v.IsSegmentEnabled(MarketSegmentSpot) {
		t.Fatal("spot should be enabled")
	}
}

func TestIsSegmentEnabledFalseWhenDisabled(t *testing.T) {
	v := VenueConfig{Segments: map[MarketSegment]*SegmentVenueConfig{
		MarketSegmentSpot: {Enabled: false, Adapter: VenueTypeBinanceSpotTestnet},
	}}
	if v.IsSegmentEnabled(MarketSegmentSpot) {
		t.Fatal("spot should be disabled")
	}
}

func TestIsSegmentEnabledFalseWhenAbsent(t *testing.T) {
	v := VenueConfig{Segments: map[MarketSegment]*SegmentVenueConfig{
		MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
	}}
	if v.IsSegmentEnabled(MarketSegmentSpot) {
		t.Fatal("spot should not be enabled when not in map")
	}
}

func TestAdapterForSegment(t *testing.T) {
	v := VenueConfig{Segments: map[MarketSegment]*SegmentVenueConfig{
		MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
	}}
	if a := v.AdapterForSegment(MarketSegmentSpot); a != VenueTypeBinanceSpotTestnet {
		t.Fatalf("expected binance_spot_testnet, got %s", a)
	}
	if a := v.AdapterForSegment(MarketSegmentFutures); a != "" {
		t.Fatalf("expected empty for absent segment, got %s", a)
	}
}

// ── S399: VenueConfig validation — unified segments ─────────────────

func TestVenueValidateAcceptsPaperWithoutSegments(t *testing.T) {
	v := VenueConfig{Type: VenueTypePaperSimulator}
	if prob := v.Validate(); prob != nil {
		t.Fatalf("paper_simulator without segments should be valid, got %v", prob)
	}
}

func TestVenueValidateAcceptsSingleSegmentSpot(t *testing.T) {
	v := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	if prob := v.Validate(); prob != nil {
		t.Fatalf("single spot segment should be valid, got %v", prob)
	}
}

func TestVenueValidateAcceptsSingleSegmentFutures(t *testing.T) {
	v := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := v.Validate(); prob != nil {
		t.Fatalf("single futures segment should be valid, got %v", prob)
	}
}

func TestVenueValidateAcceptsBothSegmentsEnabled(t *testing.T) {
	v := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := v.Validate(); prob != nil {
		t.Fatalf("both segments enabled should be valid, got %v", prob)
	}
}

func TestVenueValidateAcceptsPaperTypeWithSegments(t *testing.T) {
	v := VenueConfig{
		Type: VenueTypePaperSimulator,
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	if prob := v.Validate(); prob != nil {
		t.Fatalf("paper type with segments should be valid, got %v", prob)
	}
}

func TestVenueValidateRejectsSegmentTypeWithSegmentsMap(t *testing.T) {
	v := VenueConfig{
		Type: VenueTypeBinanceFuturesTestnet,
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	prob := v.Validate()
	if prob == nil {
		t.Fatal("segment-requiring Type with segments map should be rejected")
	}
}

func TestVenueValidateRejectsSegmentTypeWithoutSegmentsMap(t *testing.T) {
	v := VenueConfig{Type: VenueTypeBinanceFuturesTestnet}
	prob := v.Validate()
	if prob == nil {
		t.Fatal("segment-requiring Type without segments map should be rejected")
	}
}

func TestVenueValidateRejectsNoEnabledSegments(t *testing.T) {
	v := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: false, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	prob := v.Validate()
	if prob == nil {
		t.Fatal("segments map with no enabled segments should be rejected")
	}
}

func TestVenueValidateRejectsEnabledSegmentWithoutAdapter(t *testing.T) {
	v := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true},
		},
	}
	prob := v.Validate()
	if prob == nil {
		t.Fatal("enabled segment without adapter should be rejected")
	}
}

func TestVenueValidateRejectsUnknownAdapter(t *testing.T) {
	v := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: "unknown_venue"},
		},
	}
	prob := v.Validate()
	if prob == nil {
		t.Fatal("unknown adapter should be rejected")
	}
}

func TestVenueValidateRejectsAdapterSegmentMismatch(t *testing.T) {
	v := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	prob := v.Validate()
	if prob == nil {
		t.Fatal("futures adapter on spot segment should be rejected")
	}
}

func TestVenueValidateRejectsPaperAsSegmentAdapter(t *testing.T) {
	v := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypePaperSimulator},
		},
	}
	prob := v.Validate()
	if prob == nil {
		t.Fatal("paper_simulator as segment adapter should be rejected")
	}
}

func TestVenueValidateRejectsUnknownSegmentKey(t *testing.T) {
	v := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			"options": {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	prob := v.Validate()
	if prob == nil {
		t.Fatal("unknown segment key should be rejected")
	}
}

// ── S399: Preserves existing dry_run controls ───────────────────────

func TestVenueValidatePreservesDryRunFalseRejectsOnPaper(t *testing.T) {
	f := false
	v := VenueConfig{
		Type:   VenueTypePaperSimulator,
		DryRun: &f,
	}
	prob := v.Validate()
	if prob == nil {
		t.Fatal("dry_run=false on paper should still be rejected")
	}
}

func TestVenueValidateAcceptsDryRunTrueWithSegments(t *testing.T) {
	tr := true
	v := VenueConfig{
		DryRun: &tr,
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := v.Validate(); prob != nil {
		t.Fatalf("dry_run=true with segments should be valid, got %v", prob)
	}
}

func TestVenueValidateAcceptsDryRunFalseWithSegments(t *testing.T) {
	f := false
	v := VenueConfig{
		DryRun: &f,
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := v.Validate(); prob != nil {
		t.Fatalf("dry_run=false with segments should be valid, got %v", prob)
	}
}

func TestVenueIsDryRunDefaultsToTrue(t *testing.T) {
	v := VenueConfig{}
	if !v.IsDryRun() {
		t.Fatal("absent dry_run should default to true (fail-closed)")
	}
}

// ── S399: Disabled segment in map is tolerated when another is enabled ──

func TestVenueValidateAcceptsOneEnabledOneDisabled(t *testing.T) {
	v := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot:    {Enabled: false, Adapter: VenueTypeBinanceSpotTestnet},
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := v.Validate(); prob != nil {
		t.Fatalf("one enabled + one disabled should be valid, got %v", prob)
	}
}
