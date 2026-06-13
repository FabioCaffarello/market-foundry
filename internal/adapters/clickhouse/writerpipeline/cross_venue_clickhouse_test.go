//go:build requireclickhouse

package writerpipeline

// cross_venue_clickhouse_test.go — H-8.c.1 canary: the cross-venue
// snapshot write path round-trips through ClickHouse with the per-venue
// rows intact as six PARALLEL Array columns. Maps through the real
// mapper + INSERT SQL (verbatim from cmd/writer/pipeline.go), then
// SELECTs back. Reuses skipUnlessClickHouseCanonical / resetTable and
// sampleCrossVenueEvent / mapCrossVenueRow.

import (
	"context"
	"testing"
)

const insightsCrossVenueDDL = `
CREATE TABLE IF NOT EXISTS insights_cross_venue (
    event_id           String,
    occurred_at        DateTime64(3),
    correlation_id     String DEFAULT '',
    causation_id       String DEFAULT '',
    symbol             LowCardinality(String),
    base               LowCardinality(String),
    quote              LowCardinality(String),
    contract           LowCardinality(String),
    timeframe          UInt32,
    venue_name         Array(String),
    venue_trade_count  Array(Int64),
    venue_notional     Array(String),
    venue_last_price   Array(String),
    venue_high_price   Array(String),
    venue_low_price    Array(String),
    spread_abs         String,
    spread_bps         String,
    mid_price          String,
    dominant_venue     LowCardinality(String),
    trade_count        Int64,
    open_time          DateTime64(3),
    close_time         DateTime64(3),
    final              Bool,
    ingested_at        DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY (timeframe, toYYYYMM(open_time))
ORDER BY (symbol, timeframe, open_time)`

const insertInsightsCrossVenueSQL = "INSERT INTO insights_cross_venue (event_id, occurred_at, correlation_id, causation_id, symbol, base, quote, contract, timeframe, venue_name, venue_trade_count, venue_notional, venue_last_price, venue_high_price, venue_low_price, spread_abs, spread_bps, mid_price, dominant_venue, trade_count, open_time, close_time, final)"

func TestWriter_CrossVenue_ArrayColumnsRoundTrip(t *testing.T) {
	client := skipUnlessClickHouseCanonical(t)
	resetTable(t, client, "insights_cross_venue", insightsCrossVenueDDL)

	ev := sampleCrossVenueEvent(t)
	row := mapCrossVenueRow(ev)

	ctx := context.Background()
	if err := client.InsertBatch(ctx, insertInsightsCrossVenueSQL, [][]any{row}); err != nil {
		t.Fatalf("InsertBatch insights_cross_venue: %v", err)
	}

	rows, err := client.Query(ctx,
		"SELECT venue_name, venue_trade_count, venue_notional, venue_last_price, spread_abs, mid_price, dominant_venue FROM insights_cross_venue WHERE event_id = ?",
		ev.Metadata.ID,
	)
	if err != nil {
		t.Fatalf("SELECT insights_cross_venue: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatalf("no row found for event_id=%s", ev.Metadata.ID)
	}

	var (
		venueName, venueNotional, venueLast []string
		venueCount                          []int64
		spreadAbs, mid, dominant            string
	)
	if err := rows.Scan(&venueName, &venueCount, &venueNotional, &venueLast, &spreadAbs, &mid, &dominant); err != nil {
		t.Fatalf("scan: %v", err)
	}

	want := ev.CrossVenueSnapshot.Venues
	if len(venueName) != len(want) || len(venueCount) != len(want) || len(venueNotional) != len(want) || len(venueLast) != len(want) {
		t.Fatalf("venue arrays length mismatch")
	}
	for i, v := range want {
		if venueName[i] != v.Venue || venueCount[i] != v.TradeCount || venueNotional[i] != v.Notional || venueLast[i] != v.LastPrice {
			t.Errorf("venue %d round-trip mismatch: got (%s,%d,%s,%s) want (%s,%d,%s,%s)",
				i, venueName[i], venueCount[i], venueNotional[i], venueLast[i],
				v.Venue, v.TradeCount, v.Notional, v.LastPrice)
		}
	}

	if spreadAbs != ev.CrossVenueSnapshot.SpreadAbs || mid != ev.CrossVenueSnapshot.MidPrice || dominant != ev.CrossVenueSnapshot.DominantVenue {
		t.Errorf("consolidated metrics mismatch: spread=%s mid=%s dominant=%s", spreadAbs, mid, dominant)
	}
}
