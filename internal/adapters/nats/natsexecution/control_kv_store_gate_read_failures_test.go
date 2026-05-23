//go:build integration

package natsexecution_test

import (
	"context"
	"testing"
	"time"

	"internal/adapters/nats/natsexecution"
	domainexec "internal/domain/execution"
	"internal/shared/metrics"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// control_kv_store_gate_read_failures_test.go — P4.4.a: integration
// coverage for the KV-driven IsHalted failure modes whose unit tests
// would require mocking the entire jetstream.KeyValue interface. Each
// mode must (a) return IsHalted=false (fail-open per ADR 0012) and
// (b) increment gate_read_failures_total under the matching reason.

// IsHalted on a fresh KV bucket (no gate ever written) must report
// "not halted" and bump the key_not_found counter.
func TestIsHalted_KeyNotFound_FailsOpenAndCountsKeyNotFound(t *testing.T) {
	url := natsURL(t)

	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("start control KV store: %v", err)
	}
	defer store.Close()

	// Ensure the bucket starts empty for this run by deleting any
	// pre-existing gate key. We talk to JetStream directly because the
	// store exposes no Delete method; this isolates the test from
	// previous test runs that may have left state behind.
	resetControlBucket(t, url)

	before := metrics.GateReadFailureCount(metrics.GateReadFailureKeyNotFound)
	if halted := store.IsHalted(context.Background()); halted {
		t.Fatalf("empty bucket must fail-open (IsHalted=false)")
	}
	after := metrics.GateReadFailureCount(metrics.GateReadFailureKeyNotFound)
	if delta := after - before; delta != 1 {
		t.Fatalf("expected key_not_found counter delta=1, got %v", delta)
	}
}

// IsHalted called with an already-expired context must fail-open and
// bump ctx_timeout.
func TestIsHalted_CtxCancelled_FailsOpenAndCountsCtxTimeout(t *testing.T) {
	url := natsURL(t)

	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("start control KV store: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled — Get must fail with context.Canceled

	before := metrics.GateReadFailureCount(metrics.GateReadFailureCtxTimeout)
	if halted := store.IsHalted(ctx); halted {
		t.Fatalf("cancelled ctx must fail-open (IsHalted=false)")
	}
	after := metrics.GateReadFailureCount(metrics.GateReadFailureCtxTimeout)
	if delta := after - before; delta != 1 {
		t.Fatalf("expected ctx_timeout counter delta=1, got %v", delta)
	}
}

// IsHalted on a bucket whose gate entry contains invalid JSON must
// fail-open and bump unmarshal_error. The corrupt entry is injected
// via a direct JetStream Put.
func TestIsHalted_CorruptEntry_FailsOpenAndCountsUnmarshal(t *testing.T) {
	url := natsURL(t)

	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("start control KV store: %v", err)
	}
	defer store.Close()

	// Write garbage directly into the key the store reads.
	writeRawControlValue(t, url, []byte("{not valid json"))

	before := metrics.GateReadFailureCount(metrics.GateReadFailureUnmarshal)
	if halted := store.IsHalted(context.Background()); halted {
		t.Fatalf("corrupt entry must fail-open (IsHalted=false)")
	}
	after := metrics.GateReadFailureCount(metrics.GateReadFailureUnmarshal)
	if delta := after - before; delta != 1 {
		t.Fatalf("expected unmarshal_error counter delta=1, got %v", delta)
	}
}

// Sanity: on the happy path (gate explicitly active), IsHalted must
// return false without incrementing any failure counter.
func TestIsHalted_HappyPath_NoCounterIncrement(t *testing.T) {
	url := natsURL(t)

	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("start control KV store: %v", err)
	}
	defer store.Close()

	gate := domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    "p4.4.a-happy-path",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "test",
	}
	if prob := store.Put(context.Background(), gate); prob != nil {
		t.Fatalf("put active gate: %s", prob.Message)
	}

	beforeKNF := metrics.GateReadFailureCount(metrics.GateReadFailureKeyNotFound)
	beforeKV := metrics.GateReadFailureCount(metrics.GateReadFailureKVError)
	beforeCtx := metrics.GateReadFailureCount(metrics.GateReadFailureCtxTimeout)
	beforeUM := metrics.GateReadFailureCount(metrics.GateReadFailureUnmarshal)
	beforeNB := metrics.GateReadFailureCount(metrics.GateReadFailureNilBucket)

	if halted := store.IsHalted(context.Background()); halted {
		t.Fatalf("active gate must report not halted")
	}

	if got := metrics.GateReadFailureCount(metrics.GateReadFailureKeyNotFound); got != beforeKNF {
		t.Fatalf("key_not_found counter changed on happy path: before=%v after=%v", beforeKNF, got)
	}
	if got := metrics.GateReadFailureCount(metrics.GateReadFailureKVError); got != beforeKV {
		t.Fatalf("kv_error counter changed on happy path: before=%v after=%v", beforeKV, got)
	}
	if got := metrics.GateReadFailureCount(metrics.GateReadFailureCtxTimeout); got != beforeCtx {
		t.Fatalf("ctx_timeout counter changed on happy path: before=%v after=%v", beforeCtx, got)
	}
	if got := metrics.GateReadFailureCount(metrics.GateReadFailureUnmarshal); got != beforeUM {
		t.Fatalf("unmarshal_error counter changed on happy path: before=%v after=%v", beforeUM, got)
	}
	if got := metrics.GateReadFailureCount(metrics.GateReadFailureNilBucket); got != beforeNB {
		t.Fatalf("nil_bucket counter changed on happy path: before=%v after=%v", beforeNB, got)
	}
}

// resetControlBucket clears the gate key from the EXECUTION_CONTROL
// bucket so the next IsHalted call observes ErrKeyNotFound.
func resetControlBucket(t *testing.T, url string) {
	t.Helper()
	nc, err := nats.Connect(url, nats.Timeout(2*time.Second))
	if err != nil {
		t.Fatalf("dial NATS for cleanup: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("create jetstream: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	kv, err := js.KeyValue(ctx, natsexecution.ControlBucket)
	if err != nil {
		// Bucket may not exist yet — Start() will create it.
		return
	}
	_ = kv.Delete(ctx, natsexecution.ControlKey)
}

// writeRawControlValue puts an arbitrary byte payload into the gate
// key, bypassing the store's JSON encoding. Used to simulate
// corrupted entries.
func writeRawControlValue(t *testing.T, url string, payload []byte) {
	t.Helper()
	nc, err := nats.Connect(url, nats.Timeout(2*time.Second))
	if err != nil {
		t.Fatalf("dial NATS for corrupt write: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("create jetstream: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	kv, err := js.KeyValue(ctx, natsexecution.ControlBucket)
	if err != nil {
		t.Fatalf("open KV bucket: %v", err)
	}
	if _, err := kv.Put(ctx, natsexecution.ControlKey, payload); err != nil {
		t.Fatalf("put corrupt payload: %v", err)
	}
}
