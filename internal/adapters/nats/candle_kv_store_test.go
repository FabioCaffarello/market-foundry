package nats

import (
	"context"
	"testing"
	"time"

	"internal/domain/evidence"
	"internal/shared/problem"
)

func TestPutResult_String(t *testing.T) {
	tests := []struct {
		result PutResult
		want   string
	}{
		{PutWritten, "written"},
		{PutSkippedStale, "skipped_stale"},
		{PutSkippedDuplicate, "skipped_duplicate"},
		{PutResult(99), "unknown"},
	}

	for _, tc := range tests {
		if got := tc.result.String(); got != tc.want {
			t.Errorf("PutResult(%d).String() = %q, want %q", tc.result, got, tc.want)
		}
	}
}

func TestCandleKey(t *testing.T) {
	got := candleKey("binancef", "btcusdt", 60)
	want := "binancef.btcusdt.60"
	if got != want {
		t.Errorf("candleKey = %q, want %q", got, want)
	}
}

func TestCandleHistoryKey(t *testing.T) {
	ts := time.Unix(1710000000, 0)
	got := candleHistoryKey("binancef", "btcusdt", 60, ts)
	want := "binancef.btcusdt.60.1710000000"
	if got != want {
		t.Errorf("candleHistoryKey = %q, want %q", got, want)
	}
}

func TestCandleHistoryKey_DeterministicForSameOpenTime(t *testing.T) {
	ts := time.Unix(1710000060, 0)
	k1 := candleHistoryKey("binancef", "btcusdt", 60, ts)
	k2 := candleHistoryKey("binancef", "btcusdt", 60, ts)
	if k1 != k2 {
		t.Errorf("same OpenTime must produce same key: %q != %q", k1, k2)
	}
}

func TestCandleHistoryKey_DifferentForDifferentOpenTimes(t *testing.T) {
	ts1 := time.Unix(1710000000, 0)
	ts2 := time.Unix(1710000060, 0)
	k1 := candleHistoryKey("binancef", "btcusdt", 60, ts1)
	k2 := candleHistoryKey("binancef", "btcusdt", 60, ts2)
	if k1 == k2 {
		t.Errorf("different OpenTimes must produce different keys: %q", k1)
	}
}

func TestCandleKVStore_NilGuard_Put(t *testing.T) {
	var store *CandleKVStore
	candle := evidence.EvidenceCandle{
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Open:       "100.00",
		High:       "105.00",
		Low:        "99.00",
		Close:      "102.00",
		Volume:     "1000.00",
		TradeCount: 42,
		OpenTime:   time.Now().Add(-time.Minute),
		CloseTime:  time.Now(),
	}

	_, prob := store.Put(context.Background(), candle)
	if prob == nil {
		t.Fatal("expected problem on nil store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestCandleKVStore_NilGuard_Get(t *testing.T) {
	var store *CandleKVStore

	result, prob := store.Get(context.Background(), "binancef", "btcusdt", 60)
	if prob == nil {
		t.Fatal("expected problem on nil store")
	}
	if result != nil {
		t.Error("expected nil result on nil store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestCandleKVStore_NilGuard_PutHistory(t *testing.T) {
	var store *CandleKVStore
	candle := evidence.EvidenceCandle{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Open:      "100.00", High: "105.00", Low: "99.00", Close: "102.00",
		Volume: "1000.00", TradeCount: 42,
		OpenTime: time.Now().Add(-time.Minute), CloseTime: time.Now(),
	}

	prob := store.PutHistory(context.Background(), candle)
	if prob == nil {
		t.Fatal("expected problem on nil store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestCandleKVStore_NilGuard_GetHistory(t *testing.T) {
	var store *CandleKVStore

	result, prob := store.GetHistory(context.Background(), "binancef", "btcusdt", 60, 10, 0, 0)
	if prob == nil {
		t.Fatal("expected problem on nil store")
	}
	if result != nil {
		t.Error("expected nil result on nil store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestCandleKVStore_UninitializedGuard_Put(t *testing.T) {
	store := NewCandleKVStore("nats://invalid:4222")

	candle := evidence.EvidenceCandle{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
		Open: "100.00", High: "105.00", Low: "99.00", Close: "102.00",
		Volume: "1000.00", TradeCount: 42,
		OpenTime: time.Now().Add(-time.Minute), CloseTime: time.Now(),
	}

	_, prob := store.Put(context.Background(), candle)
	if prob == nil {
		t.Fatal("expected problem on uninitialized store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestCandleKVStore_UninitializedGuard_Get(t *testing.T) {
	store := NewCandleKVStore("nats://invalid:4222")

	result, prob := store.Get(context.Background(), "binancef", "btcusdt", 60)
	if prob == nil {
		t.Fatal("expected problem on uninitialized store")
	}
	if result != nil {
		t.Error("expected nil result on uninitialized store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestCandleKVStore_Constructor(t *testing.T) {
	store := NewCandleKVStore("nats://localhost:4222")
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.url != "nats://localhost:4222" {
		t.Errorf("expected url nats://localhost:4222, got %s", store.url)
	}
}

func TestCandleKVStore_BucketConstants(t *testing.T) {
	if CandleLatestBucket != "CANDLE_LATEST" {
		t.Errorf("expected CANDLE_LATEST, got %s", CandleLatestBucket)
	}
	if CandleHistoryBucket != "CANDLE_HISTORY" {
		t.Errorf("expected CANDLE_HISTORY, got %s", CandleHistoryBucket)
	}
}

func TestCandleKVStore_Close_NilSafe(t *testing.T) {
	var store *CandleKVStore
	if err := store.Close(); err != nil {
		t.Errorf("Close on nil store should not error: %v", err)
	}

	store2 := NewCandleKVStore("nats://localhost:4222")
	if err := store2.Close(); err != nil {
		t.Errorf("Close on unstarted store should not error: %v", err)
	}
}

func TestCandleKey_MultiSymbolIsolation(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	timeframes := []int{60, 300}
	keys := make(map[string]bool)

	for _, sym := range symbols {
		for _, tf := range timeframes {
			key := candleKey("binancef", sym, tf)
			if keys[key] {
				t.Fatalf("key collision for symbol=%s tf=%d: key=%s", sym, tf, key)
			}
			keys[key] = true
		}
	}

	expected := len(symbols) * len(timeframes)
	if len(keys) != expected {
		t.Errorf("expected %d unique keys, got %d", expected, len(keys))
	}
}

func TestCandleHistoryKey_MultiSymbolIsolation(t *testing.T) {
	ts := time.Unix(1710000000, 0)
	symbols := []string{"btcusdt", "ethusdt"}
	timeframes := []int{60, 300}
	keys := make(map[string]bool)

	for _, sym := range symbols {
		for _, tf := range timeframes {
			key := candleHistoryKey("binancef", sym, tf, ts)
			if keys[key] {
				t.Fatalf("history key collision for symbol=%s tf=%d", sym, tf)
			}
			keys[key] = true
		}
	}
}
