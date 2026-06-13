//go:build requireclickhouse

package writerpipeline

// volume_profile_clickhouse_test.go — H-8.a.1 canary: the VolumeProfile
// write path round-trips through ClickHouse with the price buckets intact
// as PARALLEL Array(String) columns.
//
// This is the load-bearing proof of the sub-onda's new risk: Array columns
// are the first of their kind in the foundry (Decisão #6, Opção B). The
// test maps a VolumeProfile through the real mapper + INSERT SQL (verbatim
// from cmd/writer/pipeline.go), then SELECTs back and asserts the three
// bucket arrays survived index-aligned, alongside the canonical instrument
// columns and the overload label.
//
// Reuses skipUnlessClickHouseCanonical / resetTable / envOrDefault from
// canonical_columns_integration_test.go and sampleVolumeProfileEvent /
// mapVolumeProfileRow from the unit tests (same package).
//
// Requirements (same as the H-6.d.1 canaries):
//   CLICKHOUSE_DSN=clickhouse://default:@localhost:9000/market_foundry_test
//   Skipped when CLICKHOUSE_DSN is not set.

import (
	"context"
	"testing"
)

// insightsVolumeProfileDDL mirrors deploy/migrations/014 (TTL omitted — not
// needed for test isolation; resetTable gives a clean slate per run).
const insightsVolumeProfileDDL = `
CREATE TABLE IF NOT EXISTS insights_volume_profile (
    event_id           String,
    occurred_at        DateTime64(3),
    correlation_id     String DEFAULT '',
    causation_id       String DEFAULT '',
    source             LowCardinality(String),
    symbol             LowCardinality(String),
    base               LowCardinality(String),
    quote              LowCardinality(String),
    contract           LowCardinality(String),
    timeframe          UInt32,
    bucket_size        String,
    bucket_price_level Array(String),
    bucket_buy_volume  Array(String),
    bucket_sell_volume Array(String),
    trade_count        Int64,
    overload           LowCardinality(String),
    open_time          DateTime64(3),
    close_time         DateTime64(3),
    final              Bool,
    ingested_at        DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY (timeframe, toYYYYMM(open_time))
ORDER BY (source, symbol, timeframe, open_time)`

// insertInsightsVolumeProfileSQL must match the codegen pipeline_entry in
// cmd/writer/pipeline.go verbatim (column order == mapVolumeProfileRow order).
const insertInsightsVolumeProfileSQL = "INSERT INTO insights_volume_profile (event_id, occurred_at, correlation_id, causation_id, source, symbol, base, quote, contract, timeframe, bucket_size, bucket_price_level, bucket_buy_volume, bucket_sell_volume, trade_count, overload, open_time, close_time, final)"

func TestWriter_VolumeProfile_ArrayColumnsRoundTrip(t *testing.T) {
	client := skipUnlessClickHouseCanonical(t)
	resetTable(t, client, "insights_volume_profile", insightsVolumeProfileDDL)

	ev := sampleVolumeProfileEvent(t) // 2 buckets, OverloadL2, btcUSDTPerp
	row := mapVolumeProfileRow(ev)

	ctx := context.Background()
	if err := client.InsertBatch(ctx, insertInsightsVolumeProfileSQL, [][]any{row}); err != nil {
		t.Fatalf("InsertBatch insights_volume_profile: %v", err)
	}

	rows, err := client.Query(ctx,
		"SELECT bucket_price_level, bucket_buy_volume, bucket_sell_volume, base, quote, contract, trade_count, overload FROM insights_volume_profile WHERE event_id = ?",
		ev.Metadata.ID,
	)
	if err != nil {
		t.Fatalf("SELECT insights_volume_profile: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatalf("no row found for event_id=%s", ev.Metadata.ID)
	}

	var (
		priceLevels, buyVolumes, sellVolumes []string
		base, quote, contract, overload      string
		tradeCount                           int64
	)
	if err := rows.Scan(&priceLevels, &buyVolumes, &sellVolumes, &base, &quote, &contract, &tradeCount, &overload); err != nil {
		t.Fatalf("scan: %v", err)
	}

	// Array columns survived index-aligned.
	want := ev.VolumeProfile.Buckets
	if len(priceLevels) != len(want) || len(buyVolumes) != len(want) || len(sellVolumes) != len(want) {
		t.Fatalf("bucket arrays length mismatch: got %d/%d/%d, want %d",
			len(priceLevels), len(buyVolumes), len(sellVolumes), len(want))
	}
	for i, b := range want {
		if priceLevels[i] != b.PriceLevel || buyVolumes[i] != b.BuyVolume || sellVolumes[i] != b.SellVolume {
			t.Errorf("bucket %d round-trip mismatch: got (%s,%s,%s) want (%s,%s,%s)",
				i, priceLevels[i], buyVolumes[i], sellVolumes[i],
				b.PriceLevel, b.BuyVolume, b.SellVolume)
		}
	}

	// Canonical instrument + scalar fields survived.
	if base != "BTC" || quote != "USDT" || contract != "perpetual" {
		t.Errorf("canonical columns: got (%s,%s,%s), want (BTC,USDT,perpetual)", base, quote, contract)
	}
	if tradeCount != ev.VolumeProfile.TradeCount {
		t.Errorf("trade_count: got %d, want %d", tradeCount, ev.VolumeProfile.TradeCount)
	}
	if overload != ev.VolumeProfile.Overload.Label() {
		t.Errorf("overload: got %q, want %q", overload, ev.VolumeProfile.Overload.Label())
	}
}
