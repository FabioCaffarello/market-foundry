//go:build integration

package execute_test

import (
	"sync/atomic"
	"testing"
	"time"

	"internal/shared/healthz"
)

// eventuallyAtLeast polls counter every 10ms until it reaches >= min, or
// timeout. Used in tests that read a counter set by an actor goroutine
// shortly after a NATS subscriber callback returns — the publish→increment
// ordering in venue_adapter_actor.onIntent creates a sub-microsecond race
// window where the test resumes before the counter is updated.
func eventuallyAtLeast(t *testing.T, counter *atomic.Int64, min int64, timeout time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if counter.Load() >= min {
			return
		}
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s: got %d after %s timeout", msg, counter.Load(), timeout)
}

// s343EventuallyInvariant polls the (processed, filled, skipped_halt)
// snapshot every 10ms until processed == filled + skipped_halt, or timeout.
// Returns the final snapshot. Same purpose as eventuallyAtLeast but for the
// counter-invariant assertion used by endurance/extended-observation tests.
func s343EventuallyInvariant(t *testing.T, label string, tracker *healthz.Tracker, venueReqs *atomic.Int64, timeout time.Duration) s343Snapshot {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var snap s343Snapshot
	for {
		snap = s343TakeSnapshot(tracker, venueReqs)
		if snap.processed == snap.filled+snap.skippedHalt {
			return snap
		}
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("[%s] counter invariant violated after %s: processed=%d != filled(%d) + skipped_halt(%d) = %d",
		label, timeout, snap.processed, snap.filled, snap.skippedHalt, snap.filled+snap.skippedHalt)
	return snap
}
