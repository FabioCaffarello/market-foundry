package settings

import (
	"testing"
)

// ── S416: Config surface consolidation invariants ──────────────────
// These tests validate that the canonical config model is coherent
// after the S416 consolidation. They cover fail-closed semantics,
// segment enablement, and the relationship between dry_run and adapter type.

func TestCanonicalPaperConfigIsValid(t *testing.T) {
	// execute.jsonc: paper_simulator, no segments, dry_run=true
	cfg := VenueConfig{
		Type:            VenueTypePaperSimulator,
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("canonical paper config should be valid: %v", prob)
	}
	if !cfg.IsDryRun() {
		t.Fatal("omitted dry_run should default to true (fail-closed)")
	}
	if cfg.HasUnifiedSegments() {
		t.Fatal("paper config should not have unified segments")
	}
}

func TestCanonicalUnifiedDryRunConfigIsValid(t *testing.T) {
	// execute-unified.jsonc: both segments, dry_run=true
	tr := true
	cfg := VenueConfig{
		DryRun:          &tr,
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("canonical unified dry-run config should be valid: %v", prob)
	}
	if !cfg.IsDryRun() {
		t.Fatal("dry_run=true should report dry-run active")
	}
	if !cfg.HasUnifiedSegments() {
		t.Fatal("config with segments should report unified segments")
	}
	segs := cfg.EnabledSegments()
	if len(segs) != 2 {
		t.Fatalf("expected 2 enabled segments, got %d", len(segs))
	}
	if segs[0] != MarketSegmentSpot || segs[1] != MarketSegmentFutures {
		t.Fatalf("expected [spot, futures], got %v", segs)
	}
}

func TestCanonicalVenueLiveConfigIsValid(t *testing.T) {
	// execute-venue-live.jsonc: both segments, dry_run=false
	f := false
	cfg := VenueConfig{
		DryRun:          &f,
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("canonical venue-live config should be valid: %v", prob)
	}
	if cfg.IsDryRun() {
		t.Fatal("dry_run=false should report dry-run inactive")
	}
}

func TestSingleSegmentDisablementIsValid(t *testing.T) {
	// S416: The canonical way to run a single segment is to disable the other
	// in the unified config, NOT to use a per-segment config file.
	tr := true
	cfg := VenueConfig{
		DryRun:          &tr,
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
			// futures intentionally absent
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("single-segment (spot only) config should be valid: %v", prob)
	}
	segs := cfg.EnabledSegments()
	if len(segs) != 1 || segs[0] != MarketSegmentSpot {
		t.Fatalf("expected [spot], got %v", segs)
	}
	if cfg.IsSegmentEnabled(MarketSegmentFutures) {
		t.Fatal("futures should not be enabled")
	}
}

func TestDryRunFalseWithPaperSimulatorIsRejected(t *testing.T) {
	// Invariant: dry_run=false requires a real adapter, not paper_simulator.
	f := false
	cfg := VenueConfig{
		Type:   VenueTypePaperSimulator,
		DryRun: &f,
	}
	if prob := cfg.Validate(); prob == nil {
		t.Fatal("dry_run=false with paper_simulator should be rejected")
	}
}

func TestSegmentsWithSegmentRequiringTypeIsRejected(t *testing.T) {
	// Invariant: segments + segment-requiring type creates ambiguity.
	cfg := VenueConfig{
		Type:            VenueTypeBinanceSpotTestnet,
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	if prob := cfg.Validate(); prob == nil {
		t.Fatal("segment-requiring type with segments map should be rejected")
	}
}

func TestAdapterSegmentMismatchIsRejected(t *testing.T) {
	// Invariant: adapter must match its segment.
	cfg := VenueConfig{
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := cfg.Validate(); prob == nil {
		t.Fatal("adapter/segment mismatch should be rejected")
	}
}

func TestEmptySegmentsMapIsRejected(t *testing.T) {
	// Invariant: segments map present but none enabled.
	cfg := VenueConfig{
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: false, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	if prob := cfg.Validate(); prob == nil {
		t.Fatal("segments map with no enabled segments should be rejected")
	}
}

func TestEnabledSegmentSourcesReturnsCanonicalPrefixes(t *testing.T) {
	cfg := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	sources := cfg.EnabledSegmentSources()
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	if sources[0] != "binances" || sources[1] != "binancef" {
		t.Fatalf("expected [binances, binancef], got %v", sources)
	}
}
