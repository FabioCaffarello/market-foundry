package nats

import (
	"context"
	"testing"
	"time"

	"internal/domain/evidence"
	"internal/shared/problem"
)

func TestVolumeKey(t *testing.T) {
	got := volumeKey("binancef", "btcusdt", 60)
	want := "binancef.btcusdt.60"
	if got != want {
		t.Errorf("volumeKey = %q, want %q", got, want)
	}
}

func TestVolumeKey_DeterministicForSameInputs(t *testing.T) {
	k1 := volumeKey("binancef", "ethusdt", 300)
	k2 := volumeKey("binancef", "ethusdt", 300)
	if k1 != k2 {
		t.Errorf("same inputs must produce same key: %q != %q", k1, k2)
	}
}

func TestVolumeKey_DifferentForDifferentInputs(t *testing.T) {
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
			k1 := volumeKey(tc.s1, tc.sym1, tc.tf1)
			k2 := volumeKey(tc.s2, tc.sym2, tc.tf2)
			if k1 == k2 {
				t.Errorf("different inputs must produce different keys: %q", k1)
			}
		})
	}
}

func TestVolumeKVStore_NilGuard_Put(t *testing.T) {
	var store *VolumeKVStore
	vol := evidence.EvidenceVolume{
		Source:      "binancef",
		Symbol:      "btcusdt",
		Timeframe:   60,
		BuyVolume:   "1000.0",
		SellVolume:  "800.0",
		TotalVolume: "1800.0",
		VWAP:        "50000.5",
		TradeCount:  200,
		OpenTime:    time.Now().Add(-time.Minute),
		CloseTime:   time.Now(),
	}

	_, prob := store.Put(context.Background(), vol)
	if prob == nil {
		t.Fatal("expected problem on nil store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestVolumeKVStore_NilGuard_Get(t *testing.T) {
	var store *VolumeKVStore

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

func TestVolumeKVStore_UninitializedGuard_Put(t *testing.T) {
	store := NewVolumeKVStore("nats://invalid:4222")

	vol := evidence.EvidenceVolume{
		Source:      "binancef",
		Symbol:      "btcusdt",
		Timeframe:   60,
		BuyVolume:   "500.0",
		SellVolume:  "400.0",
		TotalVolume: "900.0",
		VWAP:        "49500.0",
		TradeCount:  100,
		OpenTime:    time.Now().Add(-time.Minute),
		CloseTime:   time.Now(),
	}

	_, prob := store.Put(context.Background(), vol)
	if prob == nil {
		t.Fatal("expected problem on uninitialized store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestVolumeKVStore_UninitializedGuard_Get(t *testing.T) {
	store := NewVolumeKVStore("nats://invalid:4222")

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

func TestVolumeKVStore_Constructor(t *testing.T) {
	store := NewVolumeKVStore("nats://localhost:4222")
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.url != "nats://localhost:4222" {
		t.Errorf("expected url nats://localhost:4222, got %s", store.url)
	}
}

func TestVolumeKVStore_BucketConstant(t *testing.T) {
	if VolumeLatestBucket != "VOLUME_LATEST" {
		t.Errorf("expected VOLUME_LATEST, got %s", VolumeLatestBucket)
	}
}

func TestVolumeKVStore_Close_NilSafe(t *testing.T) {
	var store *VolumeKVStore
	if err := store.Close(); err != nil {
		t.Errorf("Close on nil store should not error: %v", err)
	}

	store2 := NewVolumeKVStore("nats://localhost:4222")
	if err := store2.Close(); err != nil {
		t.Errorf("Close on unstarted store should not error: %v", err)
	}
}
