package natsexecution_test

import (
	"context"
	"testing"

	"internal/adapters/nats/natsexecution"
	"internal/shared/metrics"
)

// control_kv_store_unit_test.go — P4.4.a: unit coverage for IsHalted's
// fail-open paths that don't require a running NATS server. The
// integration-tagged sibling file covers the remaining KV-driven modes
// (key_not_found, ctx_timeout, unmarshal_error, happy path).
//
// Per ADR 0012, IsHalted always returns false on any read failure but
// must increment gate_read_failures_total with the matching reason
// label so the silent failure mode is observable.

func TestIsHalted_NilReceiver_FailsOpenAndCountsNilBucket(t *testing.T) {
	before := metrics.GateReadFailureCount(metrics.GateReadFailureNilBucket)

	var s *natsexecution.ControlKVStore // nil receiver
	if halted := s.IsHalted(context.Background()); halted {
		t.Fatalf("nil receiver must fail-open (IsHalted=false)")
	}

	after := metrics.GateReadFailureCount(metrics.GateReadFailureNilBucket)
	if delta := after - before; delta != 1 {
		t.Fatalf("nil receiver: expected nil_bucket counter delta=1, got %v", delta)
	}
}

func TestIsHalted_UnstartedStore_FailsOpenAndCountsNilBucket(t *testing.T) {
	before := metrics.GateReadFailureCount(metrics.GateReadFailureNilBucket)

	// NewControlKVStore returns a store whose bucket field is nil until
	// Start() succeeds. Skipping Start() reproduces the init-race path.
	s := natsexecution.NewControlKVStore("nats://localhost:4222")
	if halted := s.IsHalted(context.Background()); halted {
		t.Fatalf("unstarted store must fail-open (IsHalted=false)")
	}

	after := metrics.GateReadFailureCount(metrics.GateReadFailureNilBucket)
	if delta := after - before; delta != 1 {
		t.Fatalf("unstarted store: expected nil_bucket counter delta=1, got %v", delta)
	}
}
