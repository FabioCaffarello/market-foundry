package settings

import (
	"testing"
)

// s419_consolidated_runtime_preflight_test.go — S419: Consolidated runtime preflight.
//
// Validates that the post-S416/S417/S418 config surface is coherent:
//   - All 3 canonical execute configs parse and validate
//   - Futures preflight: segment enablement, adapter resolution, source mapping
//   - No regression in standalone mode (paper_simulator)
//   - Fail-closed invariants hold across all canonical combinations
//   - Single-segment disablement still works (futures-only, spot-only)

// ── Phase 1: Canonical config surface integrity ─────────────────────

func TestS419_CanonicalPaperConfig_PostConsolidation(t *testing.T) {
	// Matches execute.jsonc: standalone paper_simulator, no segments.
	cfg := VenueConfig{
		Type:            VenueTypePaperSimulator,
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("canonical paper config invalid post-consolidation: %v", prob)
	}
	if !cfg.IsDryRun() {
		t.Fatal("paper config: omitted dry_run must default to true")
	}
	if cfg.HasUnifiedSegments() {
		t.Fatal("paper config must not report unified segments")
	}
}

func TestS419_CanonicalUnifiedDryRun_PostConsolidation(t *testing.T) {
	// Matches execute-unified.jsonc: both segments, dry_run=true.
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
		t.Fatalf("canonical unified dry-run config invalid: %v", prob)
	}
	if !cfg.IsDryRun() {
		t.Fatal("unified dry-run config must report dry_run=true")
	}
	if !cfg.HasUnifiedSegments() {
		t.Fatal("unified config must report unified segments")
	}
	segs := cfg.EnabledSegments()
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segs))
	}
}

func TestS419_CanonicalVenueLive_PostConsolidation(t *testing.T) {
	// Matches execute-venue-live.jsonc: both segments, dry_run=false.
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
		t.Fatalf("canonical venue-live config invalid: %v", prob)
	}
	if cfg.IsDryRun() {
		t.Fatal("venue-live config must report dry_run=false")
	}
}

// ── Phase 2: Futures preflight — segment readiness ──────────────────

func TestS419_FuturesPreflight_SegmentEnablement(t *testing.T) {
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

	if !cfg.IsSegmentEnabled(MarketSegmentFutures) {
		t.Fatal("futures segment must be enabled for Futures proof")
	}
	if !cfg.IsSegmentEnabled(MarketSegmentSpot) {
		t.Fatal("spot segment must remain enabled (coexistence)")
	}
}

func TestS419_FuturesPreflight_AdapterResolution(t *testing.T) {
	cfg := VenueConfig{
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}

	adapter := cfg.AdapterForSegment(MarketSegmentFutures)
	if adapter != VenueTypeBinanceFuturesTestnet {
		t.Fatalf("expected binance_futures_testnet, got %s", adapter)
	}
	if seg := adapter.Segment(); seg != MarketSegmentFutures {
		t.Fatalf("adapter segment: expected futures, got %s", seg)
	}
}

func TestS419_FuturesPreflight_SourceMapping(t *testing.T) {
	cfg := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}

	sources := cfg.EnabledSegmentSources()
	sourceSet := make(map[string]bool)
	for _, s := range sources {
		sourceSet[s] = true
	}

	if !sourceSet["binancef"] {
		t.Fatal("binancef must be in enabled sources for Futures proof")
	}
	if !sourceSet["binances"] {
		t.Fatal("binances must be in enabled sources (coexistence)")
	}

	// Verify SegmentForSource round-trips
	if seg := SegmentForSource("binancef"); seg != MarketSegmentFutures {
		t.Fatalf("SegmentForSource(binancef): expected futures, got %s", seg)
	}
	if seg := SegmentForSource("binances"); seg != MarketSegmentSpot {
		t.Fatalf("SegmentForSource(binances): expected spot, got %s", seg)
	}
}

func TestS419_FuturesPreflight_FuturesOnlyConfig(t *testing.T) {
	// S416: canonical way to run futures-only is disable spot in unified config.
	tr := true
	cfg := VenueConfig{
		DryRun:          &tr,
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("futures-only config should be valid: %v", prob)
	}
	segs := cfg.EnabledSegments()
	if len(segs) != 1 || segs[0] != MarketSegmentFutures {
		t.Fatalf("expected [futures], got %v", segs)
	}
	if cfg.IsSegmentEnabled(MarketSegmentSpot) {
		t.Fatal("spot must not be enabled in futures-only config")
	}
}

// ── Phase 3: Fail-closed invariants post-consolidation ──────────────

func TestS419_FailClosed_DryRunFalseWithPaper(t *testing.T) {
	f := false
	cfg := VenueConfig{
		Type:   VenueTypePaperSimulator,
		DryRun: &f,
	}
	if prob := cfg.Validate(); prob == nil {
		t.Fatal("dry_run=false with paper_simulator must be rejected")
	}
}

func TestS419_FailClosed_FuturesAdapterOnSpotSegment(t *testing.T) {
	cfg := VenueConfig{
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := cfg.Validate(); prob == nil {
		t.Fatal("futures adapter on spot segment must be rejected")
	}
}

func TestS419_FailClosed_SpotAdapterOnFuturesSegment(t *testing.T) {
	cfg := VenueConfig{
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	if prob := cfg.Validate(); prob == nil {
		t.Fatal("spot adapter on futures segment must be rejected")
	}
}

func TestS419_FailClosed_NoEnabledSegments(t *testing.T) {
	cfg := VenueConfig{
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot:    {Enabled: false, Adapter: VenueTypeBinanceSpotTestnet},
			MarketSegmentFutures: {Enabled: false, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := cfg.Validate(); prob == nil {
		t.Fatal("segments map with no enabled segments must be rejected")
	}
}

func TestS419_FailClosed_AmbiguousTypeWithSegments(t *testing.T) {
	cfg := VenueConfig{
		Type:            VenueTypeBinanceFuturesTestnet,
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := cfg.Validate(); prob == nil {
		t.Fatal("segment-requiring type with segments map must be rejected")
	}
}

// ── Phase 4: Taxonomy sanity (S418 cleanup verification) ────────────

func TestS419_Taxonomy_StandaloneAndSegmentsCoexist(t *testing.T) {
	// Standalone mode: paper_simulator type, no segments
	standalone := VenueConfig{
		Type:            VenueTypePaperSimulator,
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
	}
	if prob := standalone.Validate(); prob != nil {
		t.Fatalf("standalone mode invalid: %v", prob)
	}
	if standalone.HasUnifiedSegments() {
		t.Fatal("standalone mode must not have unified segments")
	}

	// Segments mode: unified config, no type
	tr := true
	segments := VenueConfig{
		DryRun:          &tr,
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	if prob := segments.Validate(); prob != nil {
		t.Fatalf("segments mode invalid: %v", prob)
	}
	if !segments.HasUnifiedSegments() {
		t.Fatal("segments mode must have unified segments")
	}
}
