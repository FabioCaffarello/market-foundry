package bootstrap

import (
	"testing"

	"internal/shared/settings"
)

func TestMainnetCredentialCheck_NoMainnet_Passes(t *testing.T) {
	cfg := settings.AppConfig{
		Venue: settings.VenueConfig{
			Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
				settings.MarketSegmentSpot: {
					Enabled: true,
					Adapter: settings.VenueTypeBinanceSpotTestnet,
				},
			},
		},
	}
	resolve := func(venueType, key string) string { return "" }
	check := MainnetCredentialCheck(cfg, resolve)
	if err := check.Check(); err != nil {
		t.Fatalf("expected no error for testnet config, got %v", err)
	}
}

func TestMainnetCredentialCheck_MainnetPresent_Passes(t *testing.T) {
	cfg := settings.AppConfig{
		Venue: settings.VenueConfig{
			Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
				settings.MarketSegmentSpot: {
					Enabled: true,
					Adapter: settings.VenueTypeBinanceSpotMainnet,
				},
			},
		},
	}
	resolve := func(venueType, key string) string { return "resolved-value" }
	check := MainnetCredentialCheck(cfg, resolve)
	if err := check.Check(); err != nil {
		t.Fatalf("expected no error when credentials present, got %v", err)
	}
}

func TestMainnetCredentialCheck_MainnetMissing_FailsClosed(t *testing.T) {
	cfg := settings.AppConfig{
		Venue: settings.VenueConfig{
			Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
				settings.MarketSegmentSpot: {
					Enabled: true,
					Adapter: settings.VenueTypeBinanceSpotMainnet,
				},
			},
		},
	}
	resolve := func(venueType, key string) string { return "" }
	check := MainnetCredentialCheck(cfg, resolve)
	if err := check.Check(); err == nil {
		t.Fatal("expected error for missing mainnet credentials")
	}
}

func TestMainnetCredentialCheck_PartialMissing_FailsClosed(t *testing.T) {
	cfg := settings.AppConfig{
		Venue: settings.VenueConfig{
			Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
				settings.MarketSegmentSpot: {
					Enabled: true,
					Adapter: settings.VenueTypeBinanceSpotMainnet,
				},
			},
		},
	}
	resolve := func(venueType, key string) string {
		if key == "API_KEY" {
			return "present"
		}
		return "" // API_SECRET missing
	}
	check := MainnetCredentialCheck(cfg, resolve)
	if err := check.Check(); err == nil {
		t.Fatal("expected error for partial credentials")
	}
}

func TestMainnetCredentialCheck_MultiSegment_BothChecked(t *testing.T) {
	cfg := settings.AppConfig{
		Venue: settings.VenueConfig{
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
		},
	}
	// Only spot credentials present
	resolve := func(venueType, key string) string {
		if venueType == "binance_spot_mainnet" {
			return "present"
		}
		return ""
	}
	check := MainnetCredentialCheck(cfg, resolve)
	if err := check.Check(); err == nil {
		t.Fatal("expected error when futures mainnet credentials missing")
	}
}

func TestMainnetCredentialCheck_PaperSimulator_Passes(t *testing.T) {
	cfg := settings.AppConfig{
		Venue: settings.VenueConfig{
			Type: settings.VenueTypePaperSimulator,
		},
	}
	resolve := func(venueType, key string) string { return "" }
	check := MainnetCredentialCheck(cfg, resolve)
	if err := check.Check(); err != nil {
		t.Fatalf("expected no error for paper simulator, got %v", err)
	}
}

func TestMainnetCredentialCheck_NoSegments_Passes(t *testing.T) {
	cfg := settings.AppConfig{}
	resolve := func(venueType, key string) string { return "" }
	check := MainnetCredentialCheck(cfg, resolve)
	if err := check.Check(); err != nil {
		t.Fatalf("expected no error for empty config, got %v", err)
	}
}
