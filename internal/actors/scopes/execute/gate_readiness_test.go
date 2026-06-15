//go:build integration

package execute_test

import (
	"context"
	"testing"
	"time"

	natsexecution "internal/adapters/nats/natsexecution"
)

// waitGateObserved polls the control gate through ControlKVStore.IsHalted
// — the exact read path the live SafetyGate uses on every event
// (internal/application/execution/safety_gate.go → control_kv_store.go:
// IsHalted) — until it observes the expected halted state, or fails.
//
// This replaces the fixed `time.Sleep(200ms)` that previously followed a
// gate Put before publishing the triggering event. A confirmed (acked)
// Put already makes the value server-visible to the actor's per-event
// read, so this is defense-in-depth: it deterministically confirms the
// gate is observable through the actor's own read path, and is faster
// than the fixed sleep (usually returns on the first poll).
//
// NOTE: the primary G9 flake is NOT a gate-visibility race — it is the
// documented "filled" counter lag (the adapter increments Counter("filled")
// AFTER PublishFill; see venue_adapter_actor.go), so a test reading the
// counter right after waitForFill can see the pre-increment value. That is
// fixed separately by waiting on the counter (s341WaitCounter) before
// snapshotting it.
func waitGateObserved(t *testing.T, store *natsexecution.ControlKVStore, wantHalted bool, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		got := store.IsHalted(ctx)
		cancel()
		if got == wantHalted {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("control gate did not become halted=%v within %s (observed halted=%v)", wantHalted, timeout, got)
		case <-time.After(25 * time.Millisecond):
		}
	}
}
