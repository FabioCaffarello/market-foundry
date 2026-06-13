package insights_test

import (
	"testing"
	"time"

	"internal/domain/insights"
)

func sampleCrossVenue(t *testing.T) insights.CrossVenueSnapshot {
	t.Helper()
	open := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	return insights.CrossVenueSnapshot{
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Venues: []insights.VenueRow{
			{Venue: "binancef", TradeCount: 5, Notional: "1000.00000000", LastPrice: "65000", HighPrice: "65020", LowPrice: "64990"},
			{Venue: "bybitf", TradeCount: 3, Notional: "2000.00000000", LastPrice: "65010", HighPrice: "65030", LowPrice: "65000"},
		},
		SpreadAbs:     "10.00000000",
		SpreadBps:     "1.5383",
		MidPrice:      "65005.00000000",
		DominantVenue: "bybitf",
		TradeCount:    8,
		OpenTime:      open,
		CloseTime:     open.Add(time.Minute),
		Final:         true,
	}
}

func TestCrossVenueSnapshot_VenueSymbolAndValidate(t *testing.T) {
	s := sampleCrossVenue(t)
	if s.VenueSymbol() != "btcusdt" {
		t.Errorf("VenueSymbol() = %q, want btcusdt", s.VenueSymbol())
	}
	if prob := s.Validate(); prob != nil {
		t.Fatalf("Validate() unexpected problem: %v", prob)
	}
}

func TestCrossVenueSnapshot_Validate_Rejections(t *testing.T) {
	t.Run("no venues", func(t *testing.T) {
		s := sampleCrossVenue(t)
		s.Venues = nil
		if s.Validate() == nil {
			t.Error("expected rejection for zero venues")
		}
	})
	t.Run("incomplete venue row", func(t *testing.T) {
		s := sampleCrossVenue(t)
		s.Venues = append(s.Venues, insights.VenueRow{Venue: "krakenf"})
		if s.Validate() == nil {
			t.Error("expected rejection for incomplete venue row")
		}
	})
}

func TestConsolidatedSpread(t *testing.T) {
	venues := []insights.VenueRow{
		{Venue: "binancef", LastPrice: "65000"},
		{Venue: "bybitf", LastPrice: "65010"},
	}
	spreadAbs, spreadBps, mid := insights.ConsolidatedSpread(venues)
	if spreadAbs != "10.00000000" {
		t.Errorf("spreadAbs = %q, want 10.00000000", spreadAbs)
	}
	if mid != "65005.00000000" {
		t.Errorf("mid = %q, want 65005.00000000", mid)
	}
	if spreadBps != "1.5383" {
		t.Errorf("spreadBps = %q, want 1.5383", spreadBps)
	}
}

func TestConsolidatedSpread_SingleVenue(t *testing.T) {
	venues := []insights.VenueRow{{Venue: "binancef", LastPrice: "65000"}}
	spreadAbs, _, mid := insights.ConsolidatedSpread(venues)
	if spreadAbs != "0.00000000" {
		t.Errorf("single-venue spreadAbs = %q, want 0.00000000", spreadAbs)
	}
	if mid != "65000.00000000" {
		t.Errorf("single-venue mid = %q, want 65000.00000000", mid)
	}
}

func TestDominantVenue(t *testing.T) {
	venues := []insights.VenueRow{
		{Venue: "binancef", Notional: "1000.00000000"},
		{Venue: "bybitf", Notional: "2000.00000000"},
	}
	if got := insights.DominantVenue(venues); got != "bybitf" {
		t.Errorf("DominantVenue = %q, want bybitf", got)
	}
	if got := insights.DominantVenue(nil); got != "" {
		t.Errorf("DominantVenue(nil) = %q, want empty", got)
	}
}
