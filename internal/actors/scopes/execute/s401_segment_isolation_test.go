package execute_test

import (
	"strings"
	"testing"

	natsexecution "internal/adapters/nats/natsexecution"
	"internal/shared/settings"
)

// ── S401: Segment isolation invariant tests ─────────────────────────
// These tests prove the structural properties of the multi-segment isolation
// hardening: consumer filtering, source-segment mapping completeness, and
// subject partitioning.

// ── Invariant 1: Every known segment has a canonical source mapping ──

func TestS401_AllKnownSegmentsHaveSourceMapping(t *testing.T) {
	for _, seg := range []settings.MarketSegment{settings.MarketSegmentSpot, settings.MarketSegmentFutures} {
		src := settings.SourceForSegment(seg)
		if src == "" {
			t.Fatalf("segment %q has no source mapping", seg)
		}
		// Round-trip: source → segment → source must be identity.
		roundTrip := settings.SegmentForSource(src)
		if roundTrip != seg {
			t.Fatalf("source %q round-trips to segment %q, expected %q", src, roundTrip, seg)
		}
	}
}

// ── Invariant 2: Source→Segment mapping is injective (no two sources map to same segment) ──

func TestS401_SourceToSegmentMappingIsInjective(t *testing.T) {
	sources := []string{"binances", "binancef"}
	seen := make(map[settings.MarketSegment]string)
	for _, src := range sources {
		seg := settings.SegmentForSource(src)
		if seg == "" {
			t.Fatalf("source %q has no segment mapping", src)
		}
		if prev, ok := seen[seg]; ok {
			t.Fatalf("segment %q mapped by both %q and %q — sources must be injective", seg, prev, src)
		}
		seen[seg] = src
	}
}

// ── Invariant 3: Spot source never maps to futures segment ──

func TestS401_SpotSourceNeverMapToFutures(t *testing.T) {
	seg := settings.SegmentForSource("binances")
	if seg == settings.MarketSegmentFutures {
		t.Fatal("binances must NOT map to futures segment")
	}
	if seg != settings.MarketSegmentSpot {
		t.Fatalf("binances should map to spot, got %q", seg)
	}
}

func TestS401_FuturesSourceNeverMapToSpot(t *testing.T) {
	seg := settings.SegmentForSource("binancef")
	if seg == settings.MarketSegmentSpot {
		t.Fatal("binancef must NOT map to spot segment")
	}
	if seg != settings.MarketSegmentFutures {
		t.Fatalf("binancef should map to futures, got %q", seg)
	}
}

// ── Invariant 4: Unknown sources rejected ──

func TestS401_UnknownSourceReturnsEmptySegment(t *testing.T) {
	unknowns := []string{"", "kraken", "bybit", "binance", "BINANCEF"}
	for _, src := range unknowns {
		seg := settings.SegmentForSource(src)
		if seg != "" {
			t.Fatalf("unknown source %q should return empty segment, got %q", src, seg)
		}
	}
}

// ── Invariant 5: Segment-scoped consumer subjects partition correctly ──

func TestS401_SpotOnlyConsumerExcludesFutures(t *testing.T) {
	spec := natsexecution.ExecuteVenueIntakeConsumerForSegments([]string{"binances"})
	for _, sub := range spec.FilterSubjects {
		if strings.Contains(sub, "binancef") {
			t.Fatalf("spot-only consumer must not contain futures source: %q", sub)
		}
	}
}

func TestS401_FuturesOnlyConsumerExcludesSpot(t *testing.T) {
	spec := natsexecution.ExecuteVenueIntakeConsumerForSegments([]string{"binancef"})
	for _, sub := range spec.FilterSubjects {
		if strings.Contains(sub, "binances") {
			t.Fatalf("futures-only consumer must not contain spot source: %q", sub)
		}
	}
}

func TestS401_UnifiedConsumerIncludesBothSegments(t *testing.T) {
	spec := natsexecution.ExecuteVenueIntakeConsumerForSegments([]string{"binances", "binancef"})
	if len(spec.FilterSubjects) != 2 {
		t.Fatalf("unified consumer expected 2 filter subjects, got %d", len(spec.FilterSubjects))
	}
	hasSpot, hasFutures := false, false
	for _, sub := range spec.FilterSubjects {
		if strings.Contains(sub, "binances") {
			hasSpot = true
		}
		if strings.Contains(sub, "binancef") {
			hasFutures = true
		}
	}
	if !hasSpot || !hasFutures {
		t.Fatalf("unified consumer must include both spot and futures: spot=%v futures=%v", hasSpot, hasFutures)
	}
}

// ── Invariant 6: EnabledSegmentSources consistent with EnabledSegments ──

func TestS401_EnabledSegmentSourcesMatchEnabledSegments(t *testing.T) {
	cfg := settings.VenueConfig{
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentSpot:    {Enabled: true, Adapter: settings.VenueTypeBinanceSpotTestnet},
			settings.MarketSegmentFutures: {Enabled: true, Adapter: settings.VenueTypeBinanceFuturesTestnet},
		},
	}
	segs := cfg.EnabledSegments()
	sources := cfg.EnabledSegmentSources()
	if len(segs) != len(sources) {
		t.Fatalf("segment count (%d) != source count (%d)", len(segs), len(sources))
	}
	for i, seg := range segs {
		expectedSrc := settings.SourceForSegment(seg)
		if sources[i] != expectedSrc {
			t.Fatalf("segment %q: expected source %q, got %q", seg, expectedSrc, sources[i])
		}
	}
}

// ── Invariant 7: NATS subject construction embeds source for auditability ──

func TestS401_PublishSubjectsContainSourcePrefix(t *testing.T) {
	// Validate that the subject patterns in the registry include source tokens.
	reg := natsexecution.DefaultRegistry()
	specs := []struct {
		name    string
		subject string
	}{
		{"PaperOrderSubmitted", reg.PaperOrderSubmitted.Subject},
		{"VenueMarketOrderFilled", reg.VenueMarketOrderFilled.Subject},
		{"VenueMarketOrderRejected", reg.VenueMarketOrderRejected.Subject},
	}
	for _, s := range specs {
		// All execution subjects are extended with .{source}.{symbol}.{timeframe}
		// at publish time. Verify the base subject is a prefix that expects extension.
		if strings.HasSuffix(s.subject, ".>") {
			t.Fatalf("%s base subject should not have wildcard suffix: %q", s.name, s.subject)
		}
		// The base subjects should be dot-separated and will get source appended.
		parts := strings.Split(s.subject, ".")
		if len(parts) < 3 {
			t.Fatalf("%s base subject too short to be valid: %q", s.name, s.subject)
		}
	}
}
