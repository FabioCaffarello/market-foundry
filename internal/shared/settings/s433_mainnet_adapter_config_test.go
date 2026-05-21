package settings_test

import (
	"testing"

	"internal/shared/settings"
)

func TestVenueType_Segment_MainnetAdapters(t *testing.T) {
	tests := []struct {
		vt   settings.VenueType
		want settings.MarketSegment
	}{
		{settings.VenueTypeBinanceSpotMainnet, settings.MarketSegmentSpot},
		{settings.VenueTypeBinanceFuturesMainnet, settings.MarketSegmentFutures},
		{settings.VenueTypeBinanceSpotTestnet, settings.MarketSegmentSpot},
		{settings.VenueTypeBinanceFuturesTestnet, settings.MarketSegmentFutures},
		{settings.VenueTypePaperSimulator, ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.vt), func(t *testing.T) {
			if got := tt.vt.Segment(); got != tt.want {
				t.Errorf("Segment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVenueType_Environment(t *testing.T) {
	tests := []struct {
		vt   settings.VenueType
		want string
	}{
		{settings.VenueTypeBinanceSpotTestnet, "testnet"},
		{settings.VenueTypeBinanceFuturesTestnet, "testnet"},
		{settings.VenueTypeBinanceSpotMainnet, "mainnet"},
		{settings.VenueTypeBinanceFuturesMainnet, "mainnet"},
		{settings.VenueTypePaperSimulator, ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.vt), func(t *testing.T) {
			if got := tt.vt.Environment(); got != tt.want {
				t.Errorf("Environment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVenueType_IsMainnet(t *testing.T) {
	if !settings.VenueTypeBinanceSpotMainnet.IsMainnet() {
		t.Error("binance_spot_mainnet should be mainnet")
	}
	if !settings.VenueTypeBinanceFuturesMainnet.IsMainnet() {
		t.Error("binance_futures_mainnet should be mainnet")
	}
	if settings.VenueTypeBinanceSpotTestnet.IsMainnet() {
		t.Error("binance_spot_testnet should not be mainnet")
	}
	if settings.VenueTypePaperSimulator.IsMainnet() {
		t.Error("paper_simulator should not be mainnet")
	}
}

func TestMainnetAdapter_KnownVenueType(t *testing.T) {
	cfg := settings.VenueConfig{
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentSpot: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceSpotMainnet,
			},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("mainnet spot adapter should be a valid config: %s", prob.Message)
	}
}

func TestMainnetAdapter_SegmentCompatibility(t *testing.T) {
	// Spot mainnet adapter on futures segment should fail.
	cfg := settings.VenueConfig{
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentFutures: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceSpotMainnet,
			},
		},
	}
	if prob := cfg.Validate(); prob == nil {
		t.Fatal("spot mainnet adapter on futures segment should be invalid")
	}
}

func TestMainnetAdapter_DryRunFalse_NowValid(t *testing.T) {
	dryRunFalse := false

	// S445: dry_run=false with mainnet adapter is now authorized (C-6 removal).
	// Previously rejected by S433. Authorized by S443 evidence gate.
	cfg := settings.VenueConfig{
		DryRun: &dryRunFalse,
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentSpot: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceSpotMainnet,
			},
		},
	}
	prob := cfg.Validate()
	if prob != nil {
		t.Fatalf("S445: dry_run=false with mainnet adapter should now be valid: %s", prob.Message)
	}
}

func TestMainnetAdapter_DryRunTrue_Valid(t *testing.T) {
	dryRunTrue := true

	cfg := settings.VenueConfig{
		DryRun: &dryRunTrue,
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentSpot: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceSpotMainnet,
			},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("dry_run=true with mainnet adapter should be valid: %s", prob.Message)
	}
}

func TestMainnetAdapter_DryRunOmitted_Valid(t *testing.T) {
	// Omitted dry_run defaults to true (fail-closed).
	cfg := settings.VenueConfig{
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentFutures: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceFuturesMainnet,
			},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("omitted dry_run with mainnet adapter should be valid (defaults to true): %s", prob.Message)
	}
}

func TestMainnetAdapter_DualSegment_Valid(t *testing.T) {
	cfg := settings.VenueConfig{
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentSpot: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceSpotMainnet,
			},
			settings.MarketSegmentFutures: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceFuturesMainnet,
			},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("dual mainnet segments should be valid: %s", prob.Message)
	}

	segs := cfg.EnabledSegments()
	if len(segs) != 2 {
		t.Fatalf("expected 2 enabled segments, got %d", len(segs))
	}
}

func TestMainnetAdapter_MixedTestnetMainnet_Valid(t *testing.T) {
	// Mixed testnet/mainnet within different segments is technically valid
	// (e.g., spot on testnet, futures on mainnet) — validation allows it.
	// Operational safety is guaranteed by dry_run=true (fail-closed default).
	cfg := settings.VenueConfig{
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentSpot: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceSpotTestnet,
			},
			settings.MarketSegmentFutures: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceFuturesMainnet,
			},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("mixed testnet/mainnet segments should be valid: %s", prob.Message)
	}
}

func TestTestnetAdapter_DryRunFalse_StillValid(t *testing.T) {
	// Testnet adapters with dry_run=false should remain valid (no regression).
	dryRunFalse := false
	cfg := settings.VenueConfig{
		DryRun: &dryRunFalse,
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentSpot: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceSpotTestnet,
			},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("dry_run=false with testnet adapter should remain valid: %s", prob.Message)
	}
}
