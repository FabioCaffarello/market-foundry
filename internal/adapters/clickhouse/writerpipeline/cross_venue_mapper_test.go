package writerpipeline

import (
	"testing"
	"time"

	"internal/domain/insights"
)

// cross_venue_mapper_test.go — H-8.c.1: locks the CrossVenueSnapshot →
// ClickHouse row mapping. Load-bearing: the per-venue rows become six
// PARALLEL, index-aligned arrays while the event produces ONE row.
// Reuses helpers from support_test.go.

func sampleCrossVenueEvent(t *testing.T) insights.CrossVenueSampledEvent {
	t.Helper()
	return insights.CrossVenueSampledEvent{
		Metadata: testMetadata(),
		CrossVenueSnapshot: insights.CrossVenueSnapshot{
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
			OpenTime:      fixedTime,
			CloseTime:     fixedTime.Add(time.Minute),
			Final:         true,
		},
	}
}

func TestMapCrossVenueRow_ColumnCount(t *testing.T) {
	row := mapCrossVenueRow(sampleCrossVenueEvent(t))
	if len(row) != 23 {
		t.Fatalf("expected 23 columns, got %d", len(row))
	}
}

func TestMapCrossVenueRow_ScalarFields(t *testing.T) {
	row := mapCrossVenueRow(sampleCrossVenueEvent(t))
	assertEq(t, "event_id", row[0], "abc123")
	assertEq(t, "symbol", row[4], "btcusdt")
	assertEq(t, "base", row[5], "BTC")
	assertEq(t, "contract", row[7], "perpetual")
	assertEq(t, "timeframe", row[8], uint32(60))
	assertEq(t, "spread_abs", row[15], "10.00000000")
	assertEq(t, "spread_bps", row[16], "1.5383")
	assertEq(t, "mid_price", row[17], "65005.00000000")
	assertEq(t, "dominant_venue", row[18], "bybitf")
	assertEq(t, "trade_count", row[19], int64(8))
	assertEq(t, "final", row[22], true)
}

// TestMapCrossVenueRow_ParallelArrays — the load-bearing assertion: six
// index-aligned venue arrays, correct element types (counts are Int64).
func TestMapCrossVenueRow_ParallelArrays(t *testing.T) {
	row := mapCrossVenueRow(sampleCrossVenueEvent(t))

	names, ok := row[9].([]string)
	if !ok || len(names) != 2 || names[0] != "binancef" || names[1] != "bybitf" {
		t.Fatalf("venue_name array wrong: %#v", row[9])
	}
	counts, ok := row[10].([]int64)
	if !ok || counts[0] != 5 || counts[1] != 3 {
		t.Fatalf("venue_trade_count array (Int64) wrong: %#v", row[10])
	}
	notionals, _ := row[11].([]string)
	lasts, _ := row[12].([]string)
	if notionals[1] != "2000.00000000" || lasts[0] != "65000" {
		t.Errorf("notional/last arrays misaligned: %v / %v", notionals, lasts)
	}
	highs, _ := row[13].([]string)
	lows, _ := row[14].([]string)
	if highs[1] != "65030" || lows[0] != "64990" {
		t.Errorf("high/low arrays misaligned: %v / %v", highs, lows)
	}
}

func TestMapCrossVenueRow_SingleVenue(t *testing.T) {
	e := sampleCrossVenueEvent(t)
	e.CrossVenueSnapshot.Venues = e.CrossVenueSnapshot.Venues[:1]
	row := mapCrossVenueRow(e)
	names, ok := row[9].([]string)
	if !ok || len(names) != 1 || names[0] != "binancef" {
		t.Errorf("single-venue array wrong: %#v", row[9])
	}
	counts, _ := row[10].([]int64)
	if len(counts) != 1 {
		t.Errorf("expected 1 count, got %v", counts)
	}
}
