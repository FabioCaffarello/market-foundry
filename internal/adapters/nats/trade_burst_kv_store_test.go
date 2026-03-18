package nats

import (
	"context"
	"testing"
	"time"

	"internal/domain/evidence"
	"internal/shared/problem"
)

func TestTradeBurstKey(t *testing.T) {
	got := tradeBurstKey("binancef", "btcusdt", 60)
	want := "binancef.btcusdt.60"
	if got != want {
		t.Errorf("tradeBurstKey = %q, want %q", got, want)
	}
}

func TestTradeBurstKey_DeterministicForSameInputs(t *testing.T) {
	k1 := tradeBurstKey("binancef", "ethusdt", 300)
	k2 := tradeBurstKey("binancef", "ethusdt", 300)
	if k1 != k2 {
		t.Errorf("same inputs must produce same key: %q != %q", k1, k2)
	}
}

func TestTradeBurstKey_DifferentForDifferentInputs(t *testing.T) {
	tests := []struct {
		name                         string
		s1, sym1                     string
		tf1                          int
		s2, sym2                     string
		tf2                          int
	}{
		{"different source", "binancef", "btcusdt", 60, "coinbase", "btcusdt", 60},
		{"different symbol", "binancef", "btcusdt", 60, "binancef", "ethusdt", 60},
		{"different timeframe", "binancef", "btcusdt", 60, "binancef", "btcusdt", 300},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k1 := tradeBurstKey(tc.s1, tc.sym1, tc.tf1)
			k2 := tradeBurstKey(tc.s2, tc.sym2, tc.tf2)
			if k1 == k2 {
				t.Errorf("different inputs must produce different keys: %q", k1)
			}
		})
	}
}

func TestTradeBurstKVStore_NilGuard_Put(t *testing.T) {
	var store *TradeBurstKVStore
	burst := evidence.EvidenceTradeBurst{
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		TradeCount: 100,
		BuyVolume:  "500.0",
		SellVolume: "300.0",
		OpenTime:   time.Now().Add(-time.Minute),
		CloseTime:  time.Now(),
	}

	_, prob := store.Put(context.Background(), burst)
	if prob == nil {
		t.Fatal("expected problem on nil store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestTradeBurstKVStore_NilGuard_Get(t *testing.T) {
	var store *TradeBurstKVStore

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

func TestTradeBurstKVStore_UninitializedGuard_Put(t *testing.T) {
	store := NewTradeBurstKVStore("nats://invalid:4222")

	burst := evidence.EvidenceTradeBurst{
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		TradeCount: 50,
		BuyVolume:  "250.0",
		SellVolume: "150.0",
		OpenTime:   time.Now().Add(-time.Minute),
		CloseTime:  time.Now(),
	}

	_, prob := store.Put(context.Background(), burst)
	if prob == nil {
		t.Fatal("expected problem on uninitialized store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestTradeBurstKVStore_UninitializedGuard_Get(t *testing.T) {
	store := NewTradeBurstKVStore("nats://invalid:4222")

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

func TestTradeBurstKVStore_Constructor(t *testing.T) {
	store := NewTradeBurstKVStore("nats://localhost:4222")
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.url != "nats://localhost:4222" {
		t.Errorf("expected url nats://localhost:4222, got %s", store.url)
	}
}

func TestTradeBurstKVStore_BucketConstant(t *testing.T) {
	if TradeBurstLatestBucket != "TRADE_BURST_LATEST" {
		t.Errorf("expected TRADE_BURST_LATEST, got %s", TradeBurstLatestBucket)
	}
}

func TestTradeBurstKVStore_Close_NilSafe(t *testing.T) {
	var store *TradeBurstKVStore
	if err := store.Close(); err != nil {
		t.Errorf("Close on nil store should not error: %v", err)
	}

	store2 := NewTradeBurstKVStore("nats://localhost:4222")
	if err := store2.Close(); err != nil {
		t.Errorf("Close on unstarted store should not error: %v", err)
	}
}
