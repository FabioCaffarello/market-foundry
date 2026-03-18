package nats

import (
	"context"
	"testing"
	"time"

	"internal/domain/signal"
	"internal/shared/problem"
)

func TestSignalKVStore_NilGuard_Put(t *testing.T) {
	var store *SignalKVStore
	sig := signal.Signal{
		Type:      "rsi",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Value:     "55.3",
		Timestamp: time.Now(),
	}

	_, prob := store.Put(context.Background(), sig)
	if prob == nil {
		t.Fatal("expected problem on nil store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestSignalKVStore_NilGuard_Get(t *testing.T) {
	var store *SignalKVStore

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

func TestSignalKVStore_UninitializedGuard_Put(t *testing.T) {
	store := NewSignalKVStore("nats://invalid:4222", SignalRSILatestBucket)
	// Not calling Start — latest is nil.

	sig := signal.Signal{
		Type:      "rsi",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Value:     "55.3",
		Timestamp: time.Now(),
	}

	_, prob := store.Put(context.Background(), sig)
	if prob == nil {
		t.Fatal("expected problem on uninitialized store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestSignalKVStore_UninitializedGuard_Get(t *testing.T) {
	store := NewSignalKVStore("nats://invalid:4222", SignalRSILatestBucket)
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

func TestSignalKVStore_Constructor(t *testing.T) {
	store := NewSignalKVStore("nats://localhost:4222", SignalRSILatestBucket)
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.url != "nats://localhost:4222" {
		t.Errorf("expected url nats://localhost:4222, got %s", store.url)
	}
	if store.bucket != SignalRSILatestBucket {
		t.Errorf("expected bucket %s, got %s", SignalRSILatestBucket, store.bucket)
	}
}

func TestSignalKVStore_BucketConstant(t *testing.T) {
	if SignalRSILatestBucket != "SIGNAL_RSI_LATEST" {
		t.Errorf("expected SIGNAL_RSI_LATEST, got %s", SignalRSILatestBucket)
	}
}

func TestSignalKVStore_MultiSymbol_KeyIsolation(t *testing.T) {
	// Verifies that different symbols produce distinct KV keys via PartitionKey.
	// This is the foundational guarantee that prevents cross-symbol bleed in the KV store.
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	timeframes := []int{60, 300}
	keys := make(map[string]bool)

	for _, sym := range symbols {
		for _, tf := range timeframes {
			sig := signal.Signal{
				Type:      "rsi",
				Source:    "binancef",
				Symbol:    sym,
				Timeframe: tf,
				Value:     "55.00",
				Timestamp: time.Now(),
				Final:     true,
			}
			key := sig.PartitionKey()
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

func TestSignalKVStore_Close_NilSafe(t *testing.T) {
	var store *SignalKVStore
	if err := store.Close(); err != nil {
		t.Errorf("Close on nil store should not error: %v", err)
	}

	store2 := NewSignalKVStore("nats://localhost:4222", SignalRSILatestBucket)
	if err := store2.Close(); err != nil {
		t.Errorf("Close on unstarted store should not error: %v", err)
	}
}
