package settings

import "testing"

// ── S401: EnabledSegmentSources unit tests ───────────────────────────

func TestEnabledSegmentSourcesReturnsBothSources(t *testing.T) {
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
	// Canonical order: spot before futures.
	if sources[0] != "binances" {
		t.Fatalf("expected first source binances, got %q", sources[0])
	}
	if sources[1] != "binancef" {
		t.Fatalf("expected second source binancef, got %q", sources[1])
	}
}

func TestEnabledSegmentSourcesSpotOnly(t *testing.T) {
	cfg := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	sources := cfg.EnabledSegmentSources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0] != "binances" {
		t.Fatalf("expected binances, got %q", sources[0])
	}
}

func TestEnabledSegmentSourcesFuturesOnly(t *testing.T) {
	cfg := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	sources := cfg.EnabledSegmentSources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0] != "binancef" {
		t.Fatalf("expected binancef, got %q", sources[0])
	}
}

func TestEnabledSegmentSourcesNilForStandaloneConfig(t *testing.T) {
	cfg := VenueConfig{Type: VenueTypePaperSimulator}
	sources := cfg.EnabledSegmentSources()
	if sources != nil {
		t.Fatalf("expected nil sources for standalone config, got %v", sources)
	}
}

func TestEnabledSegmentSourcesDisabledSegmentExcluded(t *testing.T) {
	cfg := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
			MarketSegmentFutures: {Enabled: false, Adapter: VenueTypeBinanceFuturesTestnet},
		},
	}
	sources := cfg.EnabledSegmentSources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 source (disabled futures excluded), got %d", len(sources))
	}
	if sources[0] != "binances" {
		t.Fatalf("expected binances, got %q", sources[0])
	}
}
