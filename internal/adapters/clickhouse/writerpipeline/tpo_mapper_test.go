package writerpipeline

import (
	"testing"
	"time"

	"internal/domain/insights"
)

// tpo_mapper_test.go — H-8.b.1: locks the TPOProfile → ClickHouse row
// mapping. Load-bearing assertion: periods and levels become two sets of
// PARALLEL, index-aligned arrays while the event still produces ONE row.
// Reuses helpers from support_test.go (testMetadata, btcUSDTPerp,
// assertEq, fixedTime).

func sampleTPOEvent(t *testing.T) insights.TPOProfileSampledEvent {
	t.Helper()
	return insights.TPOProfileSampledEvent{
		Metadata: testMetadata(),
		TPOProfile: insights.TPOProfile{
			Source:        "binancef",
			Instrument:    btcUSDTPerp(t),
			Timeframe:     3600,
			BucketSize:    "1",
			PeriodSeconds: 600,
			Periods: []insights.TPOPeriod{
				{Letter: "A", StartTime: fixedTime, EndTime: fixedTime.Add(10 * time.Minute), HighPrice: "65020", LowPrice: "65000"},
				{Letter: "B", StartTime: fixedTime.Add(10 * time.Minute), EndTime: fixedTime.Add(20 * time.Minute), HighPrice: "65040", LowPrice: "65010"},
			},
			Levels: []insights.TPOLevel{
				{PriceLevel: "65000", Letters: "A", Count: 1},
				{PriceLevel: "65010", Letters: "AB", Count: 2},
			},
			POCPrice:           "65010",
			ValueAreaHigh:      "65010",
			ValueAreaLow:       "65000",
			InitialBalanceHigh: "65040",
			InitialBalanceLow:  "65000",
			RangeHigh:          "65040",
			RangeLow:           "65000",
			TradeCount:         9,
			Overload:           insights.OverloadL1,
			OpenTime:           fixedTime,
			CloseTime:          fixedTime.Add(time.Hour),
			Final:              true,
		},
	}
}

func TestMapTPOProfileRow_ColumnCount(t *testing.T) {
	row := mapTPOProfileRow(sampleTPOEvent(t))
	if len(row) != 30 {
		t.Fatalf("expected 30 columns, got %d", len(row))
	}
}

func TestMapTPOProfileRow_ScalarFields(t *testing.T) {
	row := mapTPOProfileRow(sampleTPOEvent(t))
	assertEq(t, "event_id", row[0], "abc123")
	assertEq(t, "source", row[4], "binancef")
	assertEq(t, "symbol", row[5], "btcusdt")
	assertEq(t, "base", row[6], "BTC")
	assertEq(t, "contract", row[8], "perpetual")
	assertEq(t, "timeframe", row[9], uint32(3600))
	assertEq(t, "period_seconds", row[11], uint32(600))
	assertEq(t, "trade_count", row[18], int64(9))
	assertEq(t, "overload", row[19], "L1")
	assertEq(t, "poc_price", row[20], "65010")
	assertEq(t, "value_area_high", row[21], "65010")
	assertEq(t, "final", row[29], true)
}

// TestMapTPOProfileRow_ParallelArrays — the load-bearing assertion: 3
// period arrays + 3 level arrays, index-aligned, correct element types.
func TestMapTPOProfileRow_ParallelArrays(t *testing.T) {
	row := mapTPOProfileRow(sampleTPOEvent(t))

	periodLetters, ok := row[12].([]string)
	if !ok || len(periodLetters) != 2 || periodLetters[0] != "A" || periodLetters[1] != "B" {
		t.Fatalf("period_letter array wrong: %#v", row[12])
	}
	periodHighs, _ := row[13].([]string)
	periodLows, _ := row[14].([]string)
	if periodHighs[1] != "65040" || periodLows[0] != "65000" {
		t.Errorf("period high/low arrays misaligned: %v / %v", periodHighs, periodLows)
	}

	levelPrices, ok := row[15].([]string)
	if !ok || len(levelPrices) != 2 || levelPrices[0] != "65000" || levelPrices[1] != "65010" {
		t.Fatalf("level_price array wrong: %#v", row[15])
	}
	levelLetters, _ := row[16].([]string)
	if levelLetters[1] != "AB" {
		t.Errorf("level_letters misaligned: %v", levelLetters)
	}
	levelCounts, ok := row[17].([]int32)
	if !ok || levelCounts[0] != 1 || levelCounts[1] != 2 {
		t.Fatalf("level_count array (Int32) wrong: %#v", row[17])
	}
}

func TestMapTPOProfileRow_EmptyArrays(t *testing.T) {
	e := sampleTPOEvent(t)
	e.TPOProfile.Periods = nil
	e.TPOProfile.Levels = nil
	row := mapTPOProfileRow(e)
	for _, idx := range []int{12, 13, 14, 15, 16} {
		if arr, ok := row[idx].([]string); !ok || len(arr) != 0 {
			t.Errorf("col %d: expected empty []string, got %#v", idx, row[idx])
		}
	}
	if arr, ok := row[17].([]int32); !ok || len(arr) != 0 {
		t.Errorf("level_count: expected empty []int32, got %#v", row[17])
	}
}
