package nats

import (
	"context"
	"testing"
	"time"

	"internal/domain/risk"
	"internal/shared/problem"
)

func TestRiskKVStore_NilGuard_Put(t *testing.T) {
	var store *RiskKVStore
	assessment := risk.RiskAssessment{
		Type:        "position_exposure",
		Source:      "binancef",
		Symbol:      "btcusdt",
		Timeframe:   60,
		Disposition: risk.DispositionApproved,
		Confidence:  "0.85",
		Final:       true,
		Timestamp:   time.Now(),
	}

	_, prob := store.Put(context.Background(), assessment)
	if prob == nil {
		t.Fatal("expected problem on nil store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestRiskKVStore_NilGuard_Get(t *testing.T) {
	var store *RiskKVStore

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

func TestRiskKVStore_UninitializedGuard_Put(t *testing.T) {
	store := NewRiskKVStore("nats://invalid:4222", RiskPositionExposureLatestBucket)

	assessment := risk.RiskAssessment{
		Type:        "position_exposure",
		Source:      "binancef",
		Symbol:      "btcusdt",
		Timeframe:   60,
		Disposition: risk.DispositionApproved,
		Confidence:  "0.85",
		Final:       true,
		Timestamp:   time.Now(),
	}

	_, prob := store.Put(context.Background(), assessment)
	if prob == nil {
		t.Fatal("expected problem on uninitialized store")
	}
	if !problem.IsCode(prob, problem.Unavailable) {
		t.Errorf("expected Unavailable, got %v", prob)
	}
}

func TestRiskKVStore_UninitializedGuard_Get(t *testing.T) {
	store := NewRiskKVStore("nats://invalid:4222", RiskPositionExposureLatestBucket)

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

func TestRiskKVStore_Constructor(t *testing.T) {
	store := NewRiskKVStore("nats://localhost:4222", RiskPositionExposureLatestBucket)
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.url != "nats://localhost:4222" {
		t.Errorf("expected url nats://localhost:4222, got %s", store.url)
	}
	if store.bucket != RiskPositionExposureLatestBucket {
		t.Errorf("expected bucket %s, got %s", RiskPositionExposureLatestBucket, store.bucket)
	}
}

func TestRiskKVStore_BucketConstant(t *testing.T) {
	if RiskPositionExposureLatestBucket != "RISK_POSITION_EXPOSURE_LATEST" {
		t.Errorf("expected RISK_POSITION_EXPOSURE_LATEST, got %s", RiskPositionExposureLatestBucket)
	}
}

func TestRiskKVStore_MultiSymbol_KeyIsolation(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	timeframes := []int{60, 300}
	keys := make(map[string]bool)

	for _, sym := range symbols {
		for _, tf := range timeframes {
			assessment := risk.RiskAssessment{
				Type:      "position_exposure",
				Source:    "binancef",
				Symbol:    sym,
				Timeframe: tf,
			}
			key := assessment.PartitionKey()
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

func TestRiskKVStore_Close_NilSafe(t *testing.T) {
	var store *RiskKVStore
	if err := store.Close(); err != nil {
		t.Errorf("Close on nil store should not error: %v", err)
	}

	store2 := NewRiskKVStore("nats://localhost:4222", RiskPositionExposureLatestBucket)
	if err := store2.Close(); err != nil {
		t.Errorf("Close on unstarted store should not error: %v", err)
	}
}

func TestRiskKVStore_GetKeyFormat(t *testing.T) {
	assessment := risk.RiskAssessment{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	}
	partitionKey := assessment.PartitionKey()
	expectedKey := "binancef.btcusdt.60"

	if partitionKey != expectedKey {
		t.Fatalf("PartitionKey() = %q, want %q", partitionKey, expectedKey)
	}
}
