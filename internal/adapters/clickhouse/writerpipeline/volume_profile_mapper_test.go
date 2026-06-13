package writerpipeline

import (
	"testing"
	"time"

	"internal/domain/insights"
)

// volume_profile_mapper_test.go — H-8.a.1: locks the VolumeProfile →
// ClickHouse row mapping. The load-bearing assertion is that the
// per-window price buckets become three PARALLEL, index-aligned
// Array(String) columns while the event still produces exactly ONE row
// (1-event→1-row preserved). Reuses helpers from support_test.go
// (testMetadata, btcUSDTPerp, assertEq, fixedTime).

func sampleVolumeProfileEvent(t *testing.T) insights.VolumeProfileSampledEvent {
	t.Helper()
	return insights.VolumeProfileSampledEvent{
		Metadata: testMetadata(),
		VolumeProfile: insights.VolumeProfile{
			Source:     "binancef",
			Instrument: btcUSDTPerp(t),
			Timeframe:  60,
			BucketSize: "1",
			Buckets: []insights.PriceBucket{
				{PriceLevel: "65000", BuyVolume: "1200.50000000", SellVolume: "800.00000000"},
				{PriceLevel: "65010", BuyVolume: "300.00000000", SellVolume: "450.25000000"},
			},
			TradeCount: 7,
			Overload:   insights.OverloadL2,
			OpenTime:   fixedTime,
			CloseTime:  fixedTime.Add(time.Minute),
			Final:      true,
		},
	}
}

func TestMapVolumeProfileRow_ColumnCount(t *testing.T) {
	row := mapVolumeProfileRow(sampleVolumeProfileEvent(t))
	if len(row) != 19 {
		t.Fatalf("expected 19 columns, got %d", len(row))
	}
}

func TestMapVolumeProfileRow_MetadataAndDomainFields(t *testing.T) {
	row := mapVolumeProfileRow(sampleVolumeProfileEvent(t))

	assertEq(t, "event_id", row[0], "abc123")
	assertEq(t, "occurred_at", row[1], fixedTime)
	assertEq(t, "correlation_id", row[2], "corr-1")
	assertEq(t, "causation_id", row[3], "caus-1")
	assertEq(t, "source", row[4], "binancef")
	assertEq(t, "symbol", row[5], "btcusdt")
	assertEq(t, "base", row[6], "BTC")
	assertEq(t, "quote", row[7], "USDT")
	assertEq(t, "contract", row[8], "perpetual")
	assertEq(t, "timeframe", row[9], uint32(60))
	assertEq(t, "bucket_size", row[10], "1")
	assertEq(t, "trade_count", row[14], int64(7))
	assertEq(t, "overload", row[15], "L2")
	assertEq(t, "open_time", row[16], fixedTime)
	assertEq(t, "close_time", row[17], fixedTime.Add(time.Minute))
	assertEq(t, "final", row[18], true)
}

// TestMapVolumeProfileRow_ParallelArrays is the load-bearing assertion:
// the buckets become three index-aligned []string columns, so
// bucket_price_level[i] pairs with buy[i] and sell[i].
func TestMapVolumeProfileRow_ParallelArrays(t *testing.T) {
	row := mapVolumeProfileRow(sampleVolumeProfileEvent(t))

	priceLevels, ok := row[11].([]string)
	if !ok {
		t.Fatalf("bucket_price_level: expected []string, got %T", row[11])
	}
	buyVolumes, ok := row[12].([]string)
	if !ok {
		t.Fatalf("bucket_buy_volume: expected []string, got %T", row[12])
	}
	sellVolumes, ok := row[13].([]string)
	if !ok {
		t.Fatalf("bucket_sell_volume: expected []string, got %T", row[13])
	}

	if len(priceLevels) != 2 || len(buyVolumes) != 2 || len(sellVolumes) != 2 {
		t.Fatalf("expected 3 parallel arrays of length 2, got %d/%d/%d",
			len(priceLevels), len(buyVolumes), len(sellVolumes))
	}

	want := []insights.PriceBucket{
		{PriceLevel: "65000", BuyVolume: "1200.50000000", SellVolume: "800.00000000"},
		{PriceLevel: "65010", BuyVolume: "300.00000000", SellVolume: "450.25000000"},
	}
	for i, w := range want {
		if priceLevels[i] != w.PriceLevel || buyVolumes[i] != w.BuyVolume || sellVolumes[i] != w.SellVolume {
			t.Errorf("bucket %d misaligned: got (%s,%s,%s) want (%s,%s,%s)",
				i, priceLevels[i], buyVolumes[i], sellVolumes[i],
				w.PriceLevel, w.BuyVolume, w.SellVolume)
		}
	}
}

// TestMapVolumeProfileRow_EmptyBuckets — a window with no buckets still
// produces one row with empty (non-nil) arrays.
func TestMapVolumeProfileRow_EmptyBuckets(t *testing.T) {
	e := sampleVolumeProfileEvent(t)
	e.VolumeProfile.Buckets = nil
	row := mapVolumeProfileRow(e)

	for _, idx := range []int{11, 12, 13} {
		arr, ok := row[idx].([]string)
		if !ok {
			t.Fatalf("col %d: expected []string, got %T", idx, row[idx])
		}
		if len(arr) != 0 {
			t.Errorf("col %d: expected empty array, got %v", idx, arr)
		}
	}
}
