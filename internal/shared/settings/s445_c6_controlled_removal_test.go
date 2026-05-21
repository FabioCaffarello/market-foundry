package settings

import (
	"testing"
)

// s445_c6_controlled_removal_test.go -- S445: C-6 Controlled dry_run=false Removal.
//
// Validates that the S433 rejection of dry_run=false for mainnet adapters has been
// removed (C-6 from S443 evidence gate), while all other fail-closed guards remain
// intact: IsDryRun() defaults true, paper_simulator+dry_run=false still rejected.

// -- C-6 removal: dry_run=false with mainnet is now valid --

func TestS445_C6_MainnetSpot_DryRunFalse_Valid(t *testing.T) {
	dryFalse := false
	cfg := VenueConfig{
		DryRun: &dryFalse,
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotMainnet},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("C-6: dry_run=false with mainnet spot should be valid: %s", prob.Message)
	}
}

func TestS445_C6_MainnetFutures_DryRunFalse_Valid(t *testing.T) {
	dryFalse := false
	cfg := VenueConfig{
		DryRun: &dryFalse,
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesMainnet},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("C-6: dry_run=false with mainnet futures should be valid: %s", prob.Message)
	}
}

func TestS445_C6_MainnetBothSegments_DryRunFalse_Valid(t *testing.T) {
	dryFalse := false
	cfg := VenueConfig{
		DryRun: &dryFalse,
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot:    {Enabled: true, Adapter: VenueTypeBinanceSpotMainnet},
			MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesMainnet},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("C-6: dry_run=false with both mainnet segments should be valid: %s", prob.Message)
	}
}

// -- Fail-closed guards that MUST remain intact --

func TestS445_FailClosed_IsDryRun_NilDefaultsTrue(t *testing.T) {
	cfg := VenueConfig{DryRun: nil}
	if !cfg.IsDryRun() {
		t.Fatal("fail-closed violated: IsDryRun() must return true when DryRun is nil")
	}
}

func TestS445_FailClosed_PaperSimulator_DryRunFalse_StillRejected(t *testing.T) {
	dryFalse := false
	cfg := VenueConfig{
		Type:   VenueTypePaperSimulator,
		DryRun: &dryFalse,
	}
	if prob := cfg.Validate(); prob == nil {
		t.Fatal("fail-closed violated: dry_run=false with paper_simulator must still be rejected")
	}
}

func TestS445_FailClosed_PaperExplicit_DryRunFalse_StillRejected(t *testing.T) {
	dryFalse := false
	cfg := VenueConfig{
		Type:   VenueTypePaperSimulator,
		DryRun: &dryFalse,
	}
	if prob := cfg.Validate(); prob == nil {
		t.Fatal("fail-closed violated: dry_run=false with explicit paper_simulator must be rejected")
	}
}

func TestS445_FailClosed_MainnetDryRunTrue_StillValid(t *testing.T) {
	dryTrue := true
	cfg := VenueConfig{
		DryRun: &dryTrue,
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotMainnet},
		},
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("mainnet with dry_run=true must remain valid: %s", prob.Message)
	}
}

func TestS445_FailClosed_MainnetDryRunOmitted_DefaultsToTrue(t *testing.T) {
	cfg := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotMainnet},
		},
	}
	if !cfg.IsDryRun() {
		t.Fatal("fail-closed violated: omitted dry_run on mainnet must default to true")
	}
	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("mainnet with omitted dry_run must be valid: %s", prob.Message)
	}
}

// -- Testnet adapters: dry_run=false remains valid (no regression) --

func TestS445_Testnet_DryRunFalse_NoRegression(t *testing.T) {
	dryFalse := false
	cases := []struct {
		name string
		cfg  VenueConfig
	}{
		{
			"spot_testnet",
			VenueConfig{
				DryRun: &dryFalse,
				Segments: map[MarketSegment]*SegmentVenueConfig{
					MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
				},
			},
		},
		{
			"futures_testnet",
			VenueConfig{
				DryRun: &dryFalse,
				Segments: map[MarketSegment]*SegmentVenueConfig{
					MarketSegmentFutures: {Enabled: true, Adapter: VenueTypeBinanceFuturesTestnet},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if prob := tc.cfg.Validate(); prob != nil {
				t.Fatalf("testnet dry_run=false must remain valid: %s", prob.Message)
			}
		})
	}
}
