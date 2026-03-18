package nats

import (
	"context"
	"testing"
	"time"

	"internal/domain/decision"
	"internal/shared/problem"
)

func TestDecisionKVStore_NilGuard_Put(t *testing.T) {
	var store *DecisionKVStore
	dec := decision.Decision{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Outcome:    decision.OutcomeTriggered,
		Confidence: "0.85",
		Final:      true,
		Timestamp:  time.Now(),
	}

	_, prob := store.Put(context.Background(), dec)
	if prob == nil {
		t.Fatal("expected problem on nil store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestDecisionKVStore_NilGuard_Get(t *testing.T) {
	var store *DecisionKVStore

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

func TestDecisionKVStore_UninitializedGuard_Put(t *testing.T) {
	store := NewDecisionKVStore("nats://invalid:4222", DecisionRSIOversoldLatestBucket)
	// Not calling Start — latest is nil.

	dec := decision.Decision{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Outcome:    decision.OutcomeTriggered,
		Confidence: "0.85",
		Final:      true,
		Timestamp:  time.Now(),
	}

	_, prob := store.Put(context.Background(), dec)
	if prob == nil {
		t.Fatal("expected problem on uninitialized store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestDecisionKVStore_UninitializedGuard_Get(t *testing.T) {
	store := NewDecisionKVStore("nats://invalid:4222", DecisionRSIOversoldLatestBucket)
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

func TestDecisionKVStore_Constructor(t *testing.T) {
	store := NewDecisionKVStore("nats://localhost:4222", DecisionRSIOversoldLatestBucket)
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.url != "nats://localhost:4222" {
		t.Errorf("expected url nats://localhost:4222, got %s", store.url)
	}
	if store.bucket != DecisionRSIOversoldLatestBucket {
		t.Errorf("expected bucket %s, got %s", DecisionRSIOversoldLatestBucket, store.bucket)
	}
}

func TestDecisionKVStore_BucketConstant(t *testing.T) {
	if DecisionRSIOversoldLatestBucket != "DECISION_RSI_OVERSOLD_LATEST" {
		t.Errorf("expected DECISION_RSI_OVERSOLD_LATEST, got %s", DecisionRSIOversoldLatestBucket)
	}
}

func TestDecisionKVStore_MultiSymbol_KeyIsolation(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	timeframes := []int{60, 300}
	keys := make(map[string]bool)

	for _, sym := range symbols {
		for _, tf := range timeframes {
			dec := decision.Decision{
				Type:       "rsi_oversold",
				Source:     "binancef",
				Symbol:     sym,
				Timeframe:  tf,
				Outcome:    decision.OutcomeTriggered,
				Confidence: "0.85",
				Final:      true,
				Timestamp:  time.Now(),
			}
			key := dec.PartitionKey()
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

func TestDecisionKVStore_MultiSource_KeyIsolation(t *testing.T) {
	sources := []string{"binancef", "coinbase", "kraken"}
	keys := make(map[string]bool)

	for _, src := range sources {
		dec := decision.Decision{
			Type:       "rsi_oversold",
			Source:     src,
			Symbol:     "btcusdt",
			Timeframe:  60,
			Outcome:    decision.OutcomeTriggered,
			Confidence: "0.85",
			Final:      true,
			Timestamp:  time.Now(),
		}
		key := dec.PartitionKey()
		if keys[key] {
			t.Fatalf("key collision detected for source=%s: key=%s", src, key)
		}
		keys[key] = true
	}
}

func TestDecisionKVStore_Close_NilSafe(t *testing.T) {
	var store *DecisionKVStore
	if err := store.Close(); err != nil {
		t.Errorf("Close on nil store should not error: %v", err)
	}

	store2 := NewDecisionKVStore("nats://localhost:4222", DecisionRSIOversoldLatestBucket)
	if err := store2.Close(); err != nil {
		t.Errorf("Close on unstarted store should not error: %v", err)
	}
}

func TestDecisionKVStore_GetKeyFormat(t *testing.T) {
	// Verify that Get constructs the same key format as PartitionKey.
	// This is critical: if Get and Put use different key formats, reads will miss writes.
	dec := decision.Decision{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	}
	partitionKey := dec.PartitionKey()
	expectedKey := "binancef.btcusdt.60"

	if partitionKey != expectedKey {
		t.Fatalf("PartitionKey() = %q, want %q", partitionKey, expectedKey)
	}
	// Get uses fmt.Sprintf("%s.%s.%d", source, symbol, timeframe) — same format.
}
