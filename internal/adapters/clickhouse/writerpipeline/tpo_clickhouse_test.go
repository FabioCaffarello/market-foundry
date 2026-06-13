//go:build requireclickhouse

package writerpipeline

// tpo_clickhouse_test.go — H-8.b.1 canary: the TPOProfile write path
// round-trips through ClickHouse with periods and levels intact as two
// sets of PARALLEL Array columns. The load-bearing proof of the
// sub-onda (two repeated structures in one row). Maps a TPOProfile
// through the real mapper + INSERT SQL (verbatim from
// cmd/writer/pipeline.go), then SELECTs back and asserts the arrays
// survived index-aligned. Reuses skipUnlessClickHouseCanonical /
// resetTable from canonical_columns_integration_test.go and
// sampleTPOEvent / mapTPOProfileRow from the unit tests.

import (
	"context"
	"testing"
)

const insightsTPODDL = `
CREATE TABLE IF NOT EXISTS insights_tpo (
    event_id             String,
    occurred_at          DateTime64(3),
    correlation_id       String DEFAULT '',
    causation_id         String DEFAULT '',
    source               LowCardinality(String),
    symbol               LowCardinality(String),
    base                 LowCardinality(String),
    quote                LowCardinality(String),
    contract             LowCardinality(String),
    timeframe            UInt32,
    bucket_size          String,
    period_seconds       UInt32,
    period_letter        Array(String),
    period_high          Array(String),
    period_low           Array(String),
    level_price          Array(String),
    level_letters        Array(String),
    level_count          Array(Int32),
    poc_price            String,
    value_area_high      String,
    value_area_low       String,
    initial_balance_high String,
    initial_balance_low  String,
    range_high           String,
    range_low            String,
    trade_count          Int64,
    overload             LowCardinality(String),
    open_time            DateTime64(3),
    close_time           DateTime64(3),
    final                Bool,
    ingested_at          DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY (timeframe, toYYYYMM(open_time))
ORDER BY (source, symbol, timeframe, open_time)`

const insertInsightsTPOSQL = "INSERT INTO insights_tpo (event_id, occurred_at, correlation_id, causation_id, source, symbol, base, quote, contract, timeframe, bucket_size, period_seconds, period_letter, period_high, period_low, level_price, level_letters, level_count, trade_count, overload, poc_price, value_area_high, value_area_low, initial_balance_high, initial_balance_low, range_high, range_low, open_time, close_time, final)"

func TestWriter_TPO_ArrayColumnsRoundTrip(t *testing.T) {
	client := skipUnlessClickHouseCanonical(t)
	resetTable(t, client, "insights_tpo", insightsTPODDL)

	ev := sampleTPOEvent(t)
	row := mapTPOProfileRow(ev)

	ctx := context.Background()
	if err := client.InsertBatch(ctx, insertInsightsTPOSQL, [][]any{row}); err != nil {
		t.Fatalf("InsertBatch insights_tpo: %v", err)
	}

	rows, err := client.Query(ctx,
		"SELECT period_letter, period_high, period_low, level_price, level_letters, level_count, poc_price, overload FROM insights_tpo WHERE event_id = ?",
		ev.Metadata.ID,
	)
	if err != nil {
		t.Fatalf("SELECT insights_tpo: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatalf("no row found for event_id=%s", ev.Metadata.ID)
	}

	var (
		periodLetter, periodHigh, periodLow []string
		levelPrice, levelLetters            []string
		levelCount                          []int32
		poc, overload                       string
	)
	if err := rows.Scan(&periodLetter, &periodHigh, &periodLow, &levelPrice, &levelLetters, &levelCount, &poc, &overload); err != nil {
		t.Fatalf("scan: %v", err)
	}

	// Period arrays survived index-aligned.
	wantP := ev.TPOProfile.Periods
	if len(periodLetter) != len(wantP) || len(periodHigh) != len(wantP) || len(periodLow) != len(wantP) {
		t.Fatalf("period arrays length mismatch: %d/%d/%d, want %d", len(periodLetter), len(periodHigh), len(periodLow), len(wantP))
	}
	for i, p := range wantP {
		if periodLetter[i] != p.Letter || periodHigh[i] != p.HighPrice || periodLow[i] != p.LowPrice {
			t.Errorf("period %d round-trip mismatch", i)
		}
	}

	// Level arrays survived index-aligned (incl. Int32 counts).
	wantL := ev.TPOProfile.Levels
	if len(levelPrice) != len(wantL) || len(levelLetters) != len(wantL) || len(levelCount) != len(wantL) {
		t.Fatalf("level arrays length mismatch")
	}
	for i, l := range wantL {
		if levelPrice[i] != l.PriceLevel || levelLetters[i] != l.Letters || int(levelCount[i]) != l.Count {
			t.Errorf("level %d round-trip mismatch: got (%s,%s,%d) want (%s,%s,%d)",
				i, levelPrice[i], levelLetters[i], levelCount[i], l.PriceLevel, l.Letters, l.Count)
		}
	}

	if poc != ev.TPOProfile.POCPrice {
		t.Errorf("poc = %s, want %s", poc, ev.TPOProfile.POCPrice)
	}
	if overload != ev.TPOProfile.Overload.Label() {
		t.Errorf("overload = %s, want %s", overload, ev.TPOProfile.Overload.Label())
	}
}
