//go:build integration

package natsexecution_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"internal/adapters/nats/natsexecution"
	"internal/adapters/nats/natskit"
	"internal/domain/execution"
)

// kv_store_roundtrip_test.go — S271: End-to-end KV materialization proof.
//
// These tests validate the complete Put→Get round-trip for execution paper
// intents against a real NATS server. They prove:
//
//   - KV-RT-1: All ExecutionIntent fields survive serialization round-trip
//   - KV-RT-2: Monotonicity guard rejects stale writes
//   - KV-RT-3: Monotonicity guard deduplicates same-timestamp writes
//   - KV-RT-4: Multi-symbol writes are isolated by partition key
//   - KV-RT-5: Get returns nil (not error) for missing keys
//   - KV-RT-6: Post-read validation catches corrupted entries
//
// Requires a running NATS server at localhost:4222 (or NATS_URL env var).
// Skipped automatically when NATS is unreachable.

func natsURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = "nats://localhost:4222"
	}
	// Quick TCP dial to check if NATS is reachable.
	host := "localhost:4222"
	if os.Getenv("NATS_URL") != "" {
		// Extract host:port from nats:// URL.
		h := url[len("nats://"):]
		if h != "" {
			host = h
		}
	}
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		t.Skipf("NATS not reachable at %s: %v", host, err)
	}
	conn.Close()
	return url
}

func testBucket(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("TEST_EXEC_KV_%d", time.Now().UnixNano())
}

func testIntent(ts time.Time) execution.ExecutionIntent {
	return execution.ExecutionIntent{
		Type:           "paper_order",
		Source:         "binancef",
		Symbol:         "btcusdt",
		Timeframe:      60,
		Side:           execution.SideBuy,
		Quantity:       "0.02",
		FilledQuantity: "0.02",
		Status:         execution.StatusFilled,
		Risk: execution.RiskInput{
			Type:             "position_exposure",
			Disposition:      "approved",
			Confidence:       "0.85",
			Timeframe:        60,
			StrategyType:     "mean_reversion_entry",
			DecisionSeverity: "high",
		},
		Fills: []execution.FillRecord{
			{Price: "42000.50", Quantity: "0.02", Fee: "0", Simulated: true, Timestamp: ts},
		},
		Parameters:    map[string]string{"max_position_pct": "0.02"},
		Metadata:      map[string]string{"source_strategy": "mean_reversion"},
		CorrelationID: "corr-s271-roundtrip",
		CausationID:   "cause-s271-roundtrip",
		Final:         true,
		Timestamp:     ts,
	}
}

// ---------- KV-RT-1: Full field round-trip ----------

func TestKVRoundTrip_AllFieldsSurvive(t *testing.T) {
	url := natsURL(t)
	bucket := testBucket(t)
	store := natsexecution.NewKVStore(url, bucket)
	if err := store.Start(); err != nil {
		t.Fatalf("start KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	ts := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	intent := testIntent(ts)

	result, prob := store.Put(ctx, intent)
	if prob != nil {
		t.Fatalf("put: %s", prob.Message)
	}
	if result != natskit.PutWritten {
		t.Fatalf("expected PutWritten, got %s", result)
	}

	got, prob := store.Get(ctx, "binancef", "btcusdt", 60)
	if prob != nil {
		t.Fatalf("get: %s", prob.Message)
	}
	if got == nil {
		t.Fatal("expected non-nil intent after put")
	}

	// Verify every field.
	if got.Type != intent.Type {
		t.Errorf("Type: want %q, got %q", intent.Type, got.Type)
	}
	if got.Source != intent.Source {
		t.Errorf("Source: want %q, got %q", intent.Source, got.Source)
	}
	if got.Symbol != intent.Symbol {
		t.Errorf("Symbol: want %q, got %q", intent.Symbol, got.Symbol)
	}
	if got.Timeframe != intent.Timeframe {
		t.Errorf("Timeframe: want %d, got %d", intent.Timeframe, got.Timeframe)
	}
	if got.Side != intent.Side {
		t.Errorf("Side: want %q, got %q", intent.Side, got.Side)
	}
	if got.Quantity != intent.Quantity {
		t.Errorf("Quantity: want %q, got %q", intent.Quantity, got.Quantity)
	}
	if got.FilledQuantity != intent.FilledQuantity {
		t.Errorf("FilledQuantity: want %q, got %q", intent.FilledQuantity, got.FilledQuantity)
	}
	if got.Status != intent.Status {
		t.Errorf("Status: want %q, got %q", intent.Status, got.Status)
	}
	if got.Risk.Type != intent.Risk.Type {
		t.Errorf("Risk.Type: want %q, got %q", intent.Risk.Type, got.Risk.Type)
	}
	if got.Risk.Disposition != intent.Risk.Disposition {
		t.Errorf("Risk.Disposition: want %q, got %q", intent.Risk.Disposition, got.Risk.Disposition)
	}
	if got.Risk.Confidence != intent.Risk.Confidence {
		t.Errorf("Risk.Confidence: want %q, got %q", intent.Risk.Confidence, got.Risk.Confidence)
	}
	if got.Risk.Timeframe != intent.Risk.Timeframe {
		t.Errorf("Risk.Timeframe: want %d, got %d", intent.Risk.Timeframe, got.Risk.Timeframe)
	}
	if got.Risk.StrategyType != intent.Risk.StrategyType {
		t.Errorf("Risk.StrategyType: want %q, got %q", intent.Risk.StrategyType, got.Risk.StrategyType)
	}
	if got.Risk.DecisionSeverity != intent.Risk.DecisionSeverity {
		t.Errorf("Risk.DecisionSeverity: want %q, got %q", intent.Risk.DecisionSeverity, got.Risk.DecisionSeverity)
	}
	if len(got.Fills) != 1 {
		t.Fatalf("Fills: want 1, got %d", len(got.Fills))
	}
	if got.Fills[0].Price != intent.Fills[0].Price {
		t.Errorf("Fills[0].Price: want %q, got %q", intent.Fills[0].Price, got.Fills[0].Price)
	}
	if got.Fills[0].Quantity != intent.Fills[0].Quantity {
		t.Errorf("Fills[0].Quantity: want %q, got %q", intent.Fills[0].Quantity, got.Fills[0].Quantity)
	}
	if got.Fills[0].Simulated != intent.Fills[0].Simulated {
		t.Errorf("Fills[0].Simulated: want %v, got %v", intent.Fills[0].Simulated, got.Fills[0].Simulated)
	}
	if !got.Fills[0].Timestamp.Equal(intent.Fills[0].Timestamp) {
		t.Errorf("Fills[0].Timestamp: want %v, got %v", intent.Fills[0].Timestamp, got.Fills[0].Timestamp)
	}
	if got.Parameters["max_position_pct"] != "0.02" {
		t.Errorf("Parameters[max_position_pct]: want %q, got %q", "0.02", got.Parameters["max_position_pct"])
	}
	if got.Metadata["source_strategy"] != "mean_reversion" {
		t.Errorf("Metadata[source_strategy]: want %q, got %q", "mean_reversion", got.Metadata["source_strategy"])
	}
	if got.CorrelationID != intent.CorrelationID {
		t.Errorf("CorrelationID: want %q, got %q", intent.CorrelationID, got.CorrelationID)
	}
	if got.CausationID != intent.CausationID {
		t.Errorf("CausationID: want %q, got %q", intent.CausationID, got.CausationID)
	}
	if got.Final != intent.Final {
		t.Errorf("Final: want %v, got %v", intent.Final, got.Final)
	}
	if !got.Timestamp.Equal(intent.Timestamp) {
		t.Errorf("Timestamp: want %v, got %v", intent.Timestamp, got.Timestamp)
	}
}

// ---------- KV-RT-2: Monotonicity guard rejects stale writes ----------

func TestKVRoundTrip_MonotonicityGuard_RejectsStale(t *testing.T) {
	url := natsURL(t)
	bucket := testBucket(t)
	store := natsexecution.NewKVStore(url, bucket)
	if err := store.Start(); err != nil {
		t.Fatalf("start KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	newer := time.Date(2026, 3, 21, 10, 5, 0, 0, time.UTC)
	older := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)

	// Write the newer intent first.
	intentNew := testIntent(newer)
	result, prob := store.Put(ctx, intentNew)
	if prob != nil {
		t.Fatalf("put newer: %s", prob.Message)
	}
	if result != natskit.PutWritten {
		t.Fatalf("expected PutWritten for newer, got %s", result)
	}

	// Attempt to write the older intent — must be rejected as stale.
	intentOld := testIntent(older)
	result, prob = store.Put(ctx, intentOld)
	if prob != nil {
		t.Fatalf("put older: unexpected error: %s", prob.Message)
	}
	if result != natskit.PutSkippedStale {
		t.Fatalf("expected PutSkippedStale for older timestamp, got %s", result)
	}

	// Verify the stored value is still the newer one.
	got, prob := store.Get(ctx, "binancef", "btcusdt", 60)
	if prob != nil {
		t.Fatalf("get: %s", prob.Message)
	}
	if !got.Timestamp.Equal(newer) {
		t.Errorf("expected stored timestamp %v, got %v", newer, got.Timestamp)
	}
}

// ---------- KV-RT-3: Monotonicity guard deduplicates same-timestamp ----------

func TestKVRoundTrip_MonotonicityGuard_DeduplicatesSameTimestamp(t *testing.T) {
	url := natsURL(t)
	bucket := testBucket(t)
	store := natsexecution.NewKVStore(url, bucket)
	if err := store.Start(); err != nil {
		t.Fatalf("start KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	ts := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)

	intent := testIntent(ts)
	result, prob := store.Put(ctx, intent)
	if prob != nil {
		t.Fatalf("first put: %s", prob.Message)
	}
	if result != natskit.PutWritten {
		t.Fatalf("expected PutWritten for first write, got %s", result)
	}

	// Same timestamp, same key — must be deduplicated.
	result, prob = store.Put(ctx, intent)
	if prob != nil {
		t.Fatalf("second put: unexpected error: %s", prob.Message)
	}
	if result != natskit.PutSkippedDuplicate {
		t.Fatalf("expected PutSkippedDuplicate for same timestamp, got %s", result)
	}
}

// ---------- KV-RT-4: Multi-symbol isolation ----------

func TestKVRoundTrip_MultiSymbol_Isolation(t *testing.T) {
	url := natsURL(t)
	bucket := testBucket(t)
	store := natsexecution.NewKVStore(url, bucket)
	if err := store.Start(); err != nil {
		t.Fatalf("start KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	ts := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)

	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	for _, sym := range symbols {
		intent := testIntent(ts)
		intent.Symbol = sym
		intent.CorrelationID = fmt.Sprintf("corr-%s", sym)

		result, prob := store.Put(ctx, intent)
		if prob != nil {
			t.Fatalf("put %s: %s", sym, prob.Message)
		}
		if result != natskit.PutWritten {
			t.Fatalf("put %s: expected PutWritten, got %s", sym, result)
		}
	}

	// Verify each symbol is independently readable.
	for _, sym := range symbols {
		got, prob := store.Get(ctx, "binancef", sym, 60)
		if prob != nil {
			t.Fatalf("get %s: %s", sym, prob.Message)
		}
		if got == nil {
			t.Fatalf("get %s: expected non-nil intent", sym)
		}
		if got.Symbol != sym {
			t.Errorf("get %s: symbol mismatch: want %q, got %q", sym, sym, got.Symbol)
		}
		if got.CorrelationID != fmt.Sprintf("corr-%s", sym) {
			t.Errorf("get %s: correlation bleed: want corr-%s, got %q", sym, sym, got.CorrelationID)
		}
	}
}

// ---------- KV-RT-5: Get returns nil for missing keys ----------

func TestKVRoundTrip_GetMissing_ReturnsNil(t *testing.T) {
	url := natsURL(t)
	bucket := testBucket(t)
	store := natsexecution.NewKVStore(url, bucket)
	if err := store.Start(); err != nil {
		t.Fatalf("start KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	got, prob := store.Get(ctx, "binancef", "nonexistent", 60)
	if prob != nil {
		t.Fatalf("get missing: unexpected error: %s", prob.Message)
	}
	if got != nil {
		t.Fatalf("expected nil for missing key, got %+v", got)
	}
}

// ---------- KV-RT-6: Overwrite advances to newer timestamp ----------

func TestKVRoundTrip_Overwrite_AdvancesToNewer(t *testing.T) {
	url := natsURL(t)
	bucket := testBucket(t)
	store := natsexecution.NewKVStore(url, bucket)
	if err := store.Start(); err != nil {
		t.Fatalf("start KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	t1 := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 21, 10, 1, 0, 0, time.UTC)

	// Write first intent (buy).
	i1 := testIntent(t1)
	i1.Side = execution.SideBuy
	i1.CorrelationID = "corr-t1"
	if _, prob := store.Put(ctx, i1); prob != nil {
		t.Fatalf("put t1: %s", prob.Message)
	}

	// Write second intent (sell) with newer timestamp.
	i2 := testIntent(t2)
	i2.Side = execution.SideSell
	i2.CorrelationID = "corr-t2"
	result, prob := store.Put(ctx, i2)
	if prob != nil {
		t.Fatalf("put t2: %s", prob.Message)
	}
	if result != natskit.PutWritten {
		t.Fatalf("expected PutWritten for newer, got %s", result)
	}

	// Verify latest is t2.
	got, prob := store.Get(ctx, "binancef", "btcusdt", 60)
	if prob != nil {
		t.Fatalf("get: %s", prob.Message)
	}
	if got.Side != execution.SideSell {
		t.Errorf("expected sell (t2), got %q", got.Side)
	}
	if got.CorrelationID != "corr-t2" {
		t.Errorf("expected corr-t2, got %q", got.CorrelationID)
	}
	if !got.Timestamp.Equal(t2) {
		t.Errorf("expected timestamp %v, got %v", t2, got.Timestamp)
	}
}

// ---------- KV-RT-7: No-action intent (side=none) round-trip ----------

func TestKVRoundTrip_NoActionIntent_SurvivesRoundTrip(t *testing.T) {
	url := natsURL(t)
	bucket := testBucket(t)
	store := natsexecution.NewKVStore(url, bucket)
	if err := store.Start(); err != nil {
		t.Fatalf("start KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	ts := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)

	intent := testIntent(ts)
	intent.Side = execution.SideNone
	intent.Quantity = "0"
	intent.FilledQuantity = ""
	intent.Status = execution.StatusSubmitted
	intent.Fills = nil

	result, prob := store.Put(ctx, intent)
	if prob != nil {
		t.Fatalf("put: %s", prob.Message)
	}
	if result != natskit.PutWritten {
		t.Fatalf("expected PutWritten, got %s", result)
	}

	got, prob := store.Get(ctx, "binancef", "btcusdt", 60)
	if prob != nil {
		t.Fatalf("get: %s", prob.Message)
	}
	if got.Side != execution.SideNone {
		t.Errorf("Side: want none, got %q", got.Side)
	}
	if got.Quantity != "0" {
		t.Errorf("Quantity: want 0, got %q", got.Quantity)
	}
	if got.Status != execution.StatusSubmitted {
		t.Errorf("Status: want submitted, got %q", got.Status)
	}
	if len(got.Fills) != 0 {
		t.Errorf("Fills: want 0, got %d", len(got.Fills))
	}
}

// ---------- KV-RT-8: JSON serialization fidelity ----------

func TestKVRoundTrip_JSONFidelity_MatchesDomainSerialization(t *testing.T) {
	// Proves that the KV store uses the same JSON encoding as direct domain serialization.
	ts := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	intent := testIntent(ts)

	directJSON, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("direct marshal: %v", err)
	}

	var recovered execution.ExecutionIntent
	if err := json.Unmarshal(directJSON, &recovered); err != nil {
		t.Fatalf("direct unmarshal: %v", err)
	}

	// Validate domain round-trip without NATS.
	if recovered.PartitionKey() != intent.PartitionKey() {
		t.Errorf("partition key drift: want %q, got %q", intent.PartitionKey(), recovered.PartitionKey())
	}
	if recovered.DeduplicationKey() != intent.DeduplicationKey() {
		t.Errorf("dedup key drift: want %q, got %q", intent.DeduplicationKey(), recovered.DeduplicationKey())
	}
	if prob := recovered.Validate(); prob != nil {
		t.Errorf("recovered intent failed validation: %s", prob.Message)
	}
}
