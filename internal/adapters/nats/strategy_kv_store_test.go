package nats

import (
	"context"
	"testing"
	"time"

	"internal/domain/strategy"
	"internal/shared/problem"
)

func TestStrategyKVStore_NilGuard_Put(t *testing.T) {
	var store *StrategyKVStore
	strat := strategy.Strategy{
		Type:       "mean_reversion_entry",
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Direction:  strategy.DirectionLong,
		Confidence: "0.85",
		Final:      true,
		Timestamp:  time.Now(),
	}

	_, prob := store.Put(context.Background(), strat)
	if prob == nil {
		t.Fatal("expected problem on nil store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestStrategyKVStore_NilGuard_Get(t *testing.T) {
	var store *StrategyKVStore

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

func TestStrategyKVStore_UninitializedGuard_Put(t *testing.T) {
	store := NewStrategyKVStore("nats://invalid:4222", StrategyMeanReversionEntryLatestBucket)
	// Not calling Start — latest is nil.

	strat := strategy.Strategy{
		Type:       "mean_reversion_entry",
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Direction:  strategy.DirectionLong,
		Confidence: "0.85",
		Final:      true,
		Timestamp:  time.Now(),
	}

	_, prob := store.Put(context.Background(), strat)
	if prob == nil {
		t.Fatal("expected problem on uninitialized store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestStrategyKVStore_UninitializedGuard_Get(t *testing.T) {
	store := NewStrategyKVStore("nats://invalid:4222", StrategyMeanReversionEntryLatestBucket)
	// Not calling Start — latest is nil.

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

func TestStrategyKVStore_Constructor(t *testing.T) {
	store := NewStrategyKVStore("nats://localhost:4222", StrategyMeanReversionEntryLatestBucket)
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.url != "nats://localhost:4222" {
		t.Errorf("expected url nats://localhost:4222, got %s", store.url)
	}
	if store.bucket != StrategyMeanReversionEntryLatestBucket {
		t.Errorf("expected bucket %s, got %s", StrategyMeanReversionEntryLatestBucket, store.bucket)
	}
}

func TestStrategyKVStore_BucketConstant(t *testing.T) {
	if StrategyMeanReversionEntryLatestBucket != "STRATEGY_MEAN_REVERSION_ENTRY_LATEST" {
		t.Errorf("expected STRATEGY_MEAN_REVERSION_ENTRY_LATEST, got %s", StrategyMeanReversionEntryLatestBucket)
	}
}

func TestStrategyKVStore_MultiSymbol_KeyIsolation(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	timeframes := []int{60, 300}
	keys := make(map[string]bool)

	for _, sym := range symbols {
		for _, tf := range timeframes {
			strat := strategy.Strategy{
				Type:      "mean_reversion_entry",
				Source:    "binancef",
				Symbol:    sym,
				Timeframe: tf,
			}
			key := strat.PartitionKey()
			if keys[key] {
				t.Fatalf("key collision detected for symbol=%s tf=%d: key=%s", sym, tf, key)
			}
			keys[key] = true
		}
	}

	expectedCount := len(symbols) * len(timeframes)
	if len(keys) != expectedCount {
		t.Errorf("expected %d unique keys, got %d", expectedCount, len(keys))
	}
}

func TestStrategyKVStore_Close_NilSafe(t *testing.T) {
	var store *StrategyKVStore
	if err := store.Close(); err != nil {
		t.Errorf("Close on nil store should not error: %v", err)
	}

	store2 := NewStrategyKVStore("nats://localhost:4222", StrategyMeanReversionEntryLatestBucket)
	if err := store2.Close(); err != nil {
		t.Errorf("Close on unstarted store should not error: %v", err)
	}
}

func TestStrategyKVStore_GetKeyFormat(t *testing.T) {
	strat := strategy.Strategy{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	}
	partitionKey := strat.PartitionKey()
	expectedKey := "binancef.btcusdt.60"

	if partitionKey != expectedKey {
		t.Fatalf("PartitionKey() = %q, want %q", partitionKey, expectedKey)
	}
}
