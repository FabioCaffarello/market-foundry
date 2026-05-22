package settings

import (
	"testing"
)

// s436_mainnet_dryrun_config_test.go — S436/S445: Mainnet Dry-Run Config Validation.
//
// Originally S436: proved fail-closed config rules preventing mainnet dry_run=false.
// Updated S445: C-6 removal authorized dry_run=false for mainnet adapters.
// Remaining fail-closed guards: IsDryRun() defaults to true when omitted,
// dry_run=false with paper_simulator still rejected.

// ── Config validation: mainnet + dry_run=false is now valid (S445 C-6) ──

func TestS436_MainnetDryRunFalse_NowValid(t *testing.T) {
	dryFalse := false

	cases := []struct {
		name string
		cfg  VenueConfig
	}{
		{
			"spot_segment_mainnet",
			VenueConfig{
				DryRun: &dryFalse,
				Segments: map[MarketSegment]*SegmentVenueConfig{
					MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotMainnet},
				},
			},
		},
		{
			"futures_segment_mainnet",
			VenueConfig{
				DryRun: &dryFalse,
				Segments: map[MarketSegment]*SegmentVenueConfig{
					MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesMainnet},
				},
			},
		},
		{
			"both_segments_mainnet",
			VenueConfig{
				DryRun: &dryFalse,
				Segments: map[MarketSegment]*SegmentVenueConfig{
					MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotMainnet},
					MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesMainnet},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prob := tc.cfg.Validate()
			if prob != nil {
				t.Fatalf("S445: mainnet + dry_run=false should now be valid: %s", prob.Message)
			}
			t.Logf("[s445] correctly accepted: %s", tc.name)
		})
	}
}

// ── Config validation: mainnet + dry_run=true must be accepted ──────

func TestS436_MainnetDryRunTrue_Accepted(t *testing.T) {
	dryTrue := true

	cases := []struct {
		name string
		cfg  VenueConfig
	}{
		{
			"spot_segment_dryrun",
			VenueConfig{
				DryRun: &dryTrue,
				Segments: map[MarketSegment]*SegmentVenueConfig{
					MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotMainnet},
				},
			},
		},
		{
			"futures_segment_dryrun",
			VenueConfig{
				DryRun: &dryTrue,
				Segments: map[MarketSegment]*SegmentVenueConfig{
					MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesMainnet},
				},
			},
		},
		{
			"both_segments_dryrun",
			VenueConfig{
				DryRun: &dryTrue,
				Segments: map[MarketSegment]*SegmentVenueConfig{
					MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotMainnet},
					MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesMainnet},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prob := tc.cfg.Validate()
			if prob != nil {
				t.Fatalf("unexpected validation error for mainnet + dry_run=true: %s", prob.Message)
			}
			t.Logf("[s436] correctly accepted: %s", tc.name)
		})
	}
}

// ── Config validation: mainnet + dry_run omitted (nil) defaults to true ─

func TestS436_MainnetDryRunOmitted_DefaultsToTrue(t *testing.T) {
	cfg := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotMainnet},
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesMainnet},
		},
	}

	// DryRun is nil — IsDryRun must return true (fail-closed).
	if !cfg.IsDryRun() {
		t.Fatal("IsDryRun must return true when DryRun is nil (fail-closed)")
	}

	prob := cfg.Validate()
	if prob != nil {
		t.Fatalf("mainnet with omitted dry_run should be valid (defaults to true): %s", prob.Message)
	}

	t.Log("[s436] PASS — nil DryRun defaults to true, mainnet config accepted")
}

// ── VenueType helpers: IsMainnet, Environment, Segment ──────────────

func TestS436_VenueTypeMainnetHelpers(t *testing.T) {
	cases := []struct {
		vt          VenueType
		isMainnet   bool
		environment string
		segment     MarketSegment
	}{
		{VenueTypeBinanceSpotMainnet, true, "mainnet", MarketSegmentSpot},
		{VenueTypeBinanceFuturesMainnet, true, "mainnet", MarketSegmentFutures},
		{VenueTypeBinanceSpotTestnet, false, "testnet", MarketSegmentSpot},
		{VenueTypeBinanceFuturesTestnet, false, "testnet", MarketSegmentFutures},
		{VenueTypePaperSimulator, false, "", ""},
	}

	for _, tc := range cases {
		t.Run(string(tc.vt), func(t *testing.T) {
			if got := tc.vt.IsMainnet(); got != tc.isMainnet {
				t.Errorf("IsMainnet: expected %v, got %v", tc.isMainnet, got)
			}
			if got := tc.vt.Environment(); got != tc.environment {
				t.Errorf("Environment: expected %q, got %q", tc.environment, got)
			}
			if got := tc.vt.Segment(); got != tc.segment {
				t.Errorf("Segment: expected %q, got %q", tc.segment, got)
			}
		})
	}
}

// ── IsDryRun fail-closed semantics ──────────────────────────────────

func TestS436_IsDryRun_FailClosed(t *testing.T) {
	trueVal := true
	falseVal := false

	cases := []struct {
		name     string
		dryRun   *bool
		expected bool
	}{
		{"nil_defaults_true", nil, true},
		{"explicit_true", &trueVal, true},
		{"explicit_false", &falseVal, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := VenueConfig{DryRun: tc.dryRun}
			if got := cfg.IsDryRun(); got != tc.expected {
				t.Errorf("IsDryRun: expected %v, got %v", tc.expected, got)
			}
		})
	}
}

