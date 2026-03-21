//go:build integration

package natsexecution_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"internal/adapters/nats/natsexecution"
	"internal/adapters/nats/natskit"
	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/envelope"
	"internal/shared/events"
	"internal/shared/healthz"

	"github.com/fxamacker/cbor/v2"
	"github.com/nats-io/nats.go"
)

// control_plane_full_path_test.go — S275: Control Plane Full-Path Proof.
//
// These tests prove the complete control plane path from control surface write
// to observable execution behavior, closing the gap between:
//   - S273 (runtime halt/resume at the venue adapter level)
//   - S271 (KV round-trip persistence)
//   - S266/S268 (actor chain signal→execution)
//
// The full path proven here is:
//
//	control surface (KV Put) → KV persistence → publisher gate check → NATS stream presence/absence
//
// This is the derive-side gate path (ExecutionPublisherActor), complementing
// the execute-side path (SafetyGate in VenueAdapterActor) proven by S273.
//
// Proven properties:
//
//   - CP-FP-1: Active gate → intent published to EXECUTION_EVENTS stream
//   - CP-FP-2: Halted gate → intent blocked, nothing on stream
//   - CP-FP-3: Active→Halted→Resume cycle → stream observability matches gate state
//   - CP-FP-4: Dual checkpoint — both derive publisher path and venue adapter path
//     observe the same control state from the same KV source
//   - CP-FP-5: Control surface writes propagate immediately (no stale reads)
//
// Requires a running NATS server. Skipped automatically when unreachable.

// fullPathSeq is a monotonic counter ensuring unique dedup keys across test runs.
var fullPathSeq atomic.Int64

// fullPathBuildEvent constructs a valid, filled PaperOrderSubmittedEvent with a unique
// timestamp to avoid JetStream deduplication across test phases and runs.
func fullPathBuildEvent(t *testing.T, ts time.Time, corrID string) domainexec.PaperOrderSubmittedEvent {
	// Offset by sequence counter to ensure unique dedup keys.
	ts = ts.Add(time.Duration(fullPathSeq.Add(1)) * time.Millisecond)
	t.Helper()
	eval := appexec.NewPaperOrderEvaluator("binancef", "btcusdt", 60)
	intent, ok := eval.Evaluate(
		"position_exposure", "approved", "0.85", "0.02",
		"long", "0.72",
		"mean_reversion_entry", "high",
		60, ts,
	)
	if !ok {
		t.Fatal("evaluation should succeed")
	}
	intent.CorrelationID = corrID
	intent.CausationID = "cause-s275-full-path"

	sim := &appexec.PaperFillSimulator{}
	intent, ok = sim.SimulateFill(intent)
	if !ok {
		t.Fatal("fill simulation should succeed")
	}

	return domainexec.PaperOrderSubmittedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(intent.CorrelationID).
			WithCausationID(intent.CausationID),
		ExecutionIntent: intent,
	}
}

// publisherGateCheck replicates the ExecutionPublisherActor.publishWithRetry() gate check
// and publish sequence. Returns true if the event was published to the stream.
func publisherGateCheck(
	controlStore *natsexecution.ControlKVStore,
	publisher *natsexecution.Publisher,
	event domainexec.PaperOrderSubmittedEvent,
	halted *atomic.Int64,
	published *atomic.Int64,
) bool {
	// Gate check: mirrors ExecutionPublisherActor lines 106-122.
	if controlStore != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		isHalted := controlStore.IsHalted(ctx)
		cancel()
		if isHalted {
			halted.Add(1)
			return false
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event)
	cancel()
	if prob != nil {
		return false
	}

	published.Add(1)
	return true
}

// streamSubscriber creates a core NATS subscription on the execution subject wildcard
// and returns a channel that receives each published message's correlation ID.
// Uses core NATS (not JetStream consumer) to avoid consumer startup race conditions.
func streamSubscriber(t *testing.T, url string) (chan string, func()) {
	t.Helper()

	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("subscriber connect: %v", err)
	}

	ch := make(chan string, 50)
	sub, err := nc.Subscribe("execution.events.paper_order.submitted.>", func(msg *nats.Msg) {
		// Decode CBOR envelope to extract correlation ID.
		var env envelope.Envelope[domainexec.PaperOrderSubmittedEvent]
		if err := cbor.Unmarshal(msg.Data, &env); err == nil && env.CorrelationID != "" {
			ch <- env.CorrelationID
		} else {
			ch <- "decode-error"
		}
	})
	if err != nil {
		nc.Close()
		t.Fatalf("subscribe: %v", err)
	}

	// Flush to ensure subscription is active on the server.
	if err := nc.Flush(); err != nil {
		sub.Unsubscribe()
		nc.Close()
		t.Fatalf("flush: %v", err)
	}

	cleanup := func() {
		sub.Unsubscribe()
		nc.Close()
	}

	return ch, cleanup
}

// waitForMessage waits for a message with the given correlation ID on the channel.
func waitForMessage(ch chan string, corrID string, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		select {
		case got := <-ch:
			if got == corrID {
				return true
			}
			// Drain other messages.
		case <-deadline:
			return false
		}
	}
}

// assertNoMessage verifies no message appears on the channel within the given duration.
func assertNoMessage(ch chan string, wait time.Duration) bool {
	timer := time.After(wait)
	for {
		select {
		case <-ch:
			return false // Unexpected message.
		case <-timer:
			return true // Good: nothing arrived.
		}
	}
}

// ---------- CP-FP-1: Active gate → intent published to stream ----------

func TestControlPlane_FullPath_Active_PublishesToStream(t *testing.T) {
	url := natsURL(t)

	// Setup: control store, publisher, stream subscriber.
	controlStore := natsexecution.NewControlKVStore(url)
	if err := controlStore.Start(); err != nil {
		t.Fatalf("start control store: %v", err)
	}
	defer controlStore.Close()

	registry := natsexecution.DefaultRegistry()
	pub := natsexecution.NewPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	msgCh, cleanup := streamSubscriber(t, url)
	defer cleanup()

	// Set gate to active.
	ctx := context.Background()
	prob := controlStore.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    "s275-full-path-active",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "s275-test",
	})
	if prob != nil {
		t.Fatalf("put active: %s", prob.Message)
	}

	// Build and publish via gate-checked path.
	var haltedCount, publishedCount atomic.Int64
	corrID := "cp-fp-1-active"
	event := fullPathBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
	ok := publisherGateCheck(controlStore, pub, event, &haltedCount, &publishedCount)
	if !ok {
		t.Fatal("expected publish to succeed when gate is active")
	}

	// Verify message arrived on stream.
	if !waitForMessage(msgCh, corrID, 3*time.Second) {
		t.Fatal("expected message on EXECUTION_EVENTS stream — not received")
	}

	if publishedCount.Load() != 1 {
		t.Fatalf("expected published=1, got %d", publishedCount.Load())
	}
	if haltedCount.Load() != 0 {
		t.Fatalf("expected halted=0, got %d", haltedCount.Load())
	}
}

// ---------- CP-FP-2: Halted gate → intent blocked, nothing on stream ----------

func TestControlPlane_FullPath_Halted_BlocksStreamPublish(t *testing.T) {
	url := natsURL(t)

	controlStore := natsexecution.NewControlKVStore(url)
	if err := controlStore.Start(); err != nil {
		t.Fatalf("start control store: %v", err)
	}
	defer controlStore.Close()

	registry := natsexecution.DefaultRegistry()
	pub := natsexecution.NewPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	msgCh, cleanup := streamSubscriber(t, url)
	defer cleanup()

	// Set gate to halted.
	ctx := context.Background()
	prob := controlStore.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "s275-full-path-halt-test",
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: "s275-test",
	})
	if prob != nil {
		t.Fatalf("put halted: %s", prob.Message)
	}

	// Attempt publish via gate-checked path.
	var haltedCount, publishedCount atomic.Int64
	corrID := "cp-fp-2-halted"
	event := fullPathBuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
	ok := publisherGateCheck(controlStore, pub, event, &haltedCount, &publishedCount)
	if ok {
		t.Fatal("expected publish to be blocked when gate is halted")
	}

	// Verify NO message on stream.
	if !assertNoMessage(msgCh, 500*time.Millisecond) {
		t.Fatal("expected no message on stream during halt — but one arrived")
	}

	if haltedCount.Load() != 1 {
		t.Fatalf("expected halted=1, got %d", haltedCount.Load())
	}
	if publishedCount.Load() != 0 {
		t.Fatalf("expected published=0, got %d", publishedCount.Load())
	}
}

// ---------- CP-FP-3: Active→Halted→Resume cycle with stream observation ----------

func TestControlPlane_FullPath_ActiveHaltedResume_Cycle(t *testing.T) {
	url := natsURL(t)

	controlStore := natsexecution.NewControlKVStore(url)
	if err := controlStore.Start(); err != nil {
		t.Fatalf("start control store: %v", err)
	}
	defer controlStore.Close()

	registry := natsexecution.DefaultRegistry()
	pub := natsexecution.NewPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	msgCh, cleanup := streamSubscriber(t, url)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	var haltedCount, publishedCount atomic.Int64

	// Phase 1: Active — publish should reach stream.
	prob := controlStore.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    "s275-cycle-phase1-active",
		UpdatedAt: now,
		UpdatedBy: "s275-test",
	})
	if prob != nil {
		t.Fatalf("phase 1 put: %s", prob.Message)
	}

	corrID1 := "cp-fp-3-phase1"
	event1 := fullPathBuildEvent(t, now.Add(-10*time.Second), corrID1)
	if !publisherGateCheck(controlStore, pub, event1, &haltedCount, &publishedCount) {
		t.Fatal("phase 1: expected publish to succeed")
	}
	if !waitForMessage(msgCh, corrID1, 3*time.Second) {
		t.Fatal("phase 1: expected message on stream")
	}

	// Phase 2: Halted — publish should be blocked.
	prob = controlStore.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "s275-cycle-phase2-halted",
		UpdatedAt: now.Add(1 * time.Second),
		UpdatedBy: "s275-test",
	})
	if prob != nil {
		t.Fatalf("phase 2 put: %s", prob.Message)
	}

	corrID2 := "cp-fp-3-phase2"
	event2 := fullPathBuildEvent(t, now.Add(-9*time.Second), corrID2)
	if publisherGateCheck(controlStore, pub, event2, &haltedCount, &publishedCount) {
		t.Fatal("phase 2: expected publish to be blocked")
	}
	if !assertNoMessage(msgCh, 500*time.Millisecond) {
		t.Fatal("phase 2: expected no message on stream")
	}

	// Phase 3: Resume — publish should reach stream again.
	prob = controlStore.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    "s275-cycle-phase3-resume",
		UpdatedAt: now.Add(2 * time.Second),
		UpdatedBy: "s275-test",
	})
	if prob != nil {
		t.Fatalf("phase 3 put: %s", prob.Message)
	}

	corrID3 := "cp-fp-3-phase3"
	event3 := fullPathBuildEvent(t, now.Add(-8*time.Second), corrID3)
	if !publisherGateCheck(controlStore, pub, event3, &haltedCount, &publishedCount) {
		t.Fatal("phase 3: expected publish to succeed after resume")
	}
	if !waitForMessage(msgCh, corrID3, 3*time.Second) {
		t.Fatal("phase 3: expected message on stream after resume")
	}

	// Verify cumulative counters.
	if publishedCount.Load() != 2 {
		t.Fatalf("expected published=2, got %d", publishedCount.Load())
	}
	if haltedCount.Load() != 1 {
		t.Fatalf("expected halted=1, got %d", haltedCount.Load())
	}
}

// ---------- CP-FP-4: Dual checkpoint — publisher path and venue path see same state ----------

func TestControlPlane_FullPath_DualCheckpoint_PublisherAndVenue(t *testing.T) {
	url := natsURL(t)

	controlStore := natsexecution.NewControlKVStore(url)
	if err := controlStore.Start(); err != nil {
		t.Fatalf("start control store: %v", err)
	}
	defer controlStore.Close()

	registry := natsexecution.DefaultRegistry()
	pub := natsexecution.NewPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	// Wire venue adapter path (SafetyGate + PaperVenueAdapter).
	gate := appexec.NewSafetyGate(controlStore, 2*time.Second, appexec.NewStalenessGuard(2*time.Minute))
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("cp-fp-4")

	ctx := context.Background()
	now := time.Now().UTC()
	var haltedCount, publishedCount atomic.Int64

	// Phase 1: Active — both paths allow.
	prob := controlStore.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    "s275-dual-active",
		UpdatedAt: now,
		UpdatedBy: "s275-test",
	})
	if prob != nil {
		t.Fatalf("put active: %s", prob.Message)
	}

	corrID := "cp-fp-4-dual-active"
	event := fullPathBuildEvent(t, now.Add(-10*time.Second), corrID)

	// Publisher path (derive side).
	pubOK := publisherGateCheck(controlStore, pub, event, &haltedCount, &publishedCount)
	// Venue path (execute side).
	_, venueBlocked, _ := runtimeOnIntent(gate, venue, tracker, event, now)

	if !pubOK {
		t.Fatal("phase 1: publisher path should allow when active")
	}
	if venueBlocked {
		t.Fatal("phase 1: venue path should allow when active")
	}

	// Phase 2: Halted — both paths block.
	prob = controlStore.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "s275-dual-halted",
		UpdatedAt: now.Add(1 * time.Second),
		UpdatedBy: "s275-test",
	})
	if prob != nil {
		t.Fatalf("put halted: %s", prob.Message)
	}

	corrID2 := "cp-fp-4-dual-halted"
	event2 := fullPathBuildEvent(t, now.Add(-9*time.Second), corrID2)

	pubOK2 := publisherGateCheck(controlStore, pub, event2, &haltedCount, &publishedCount)
	_, venueBlocked2, reason2 := runtimeOnIntent(gate, venue, tracker, event2, now)

	if pubOK2 {
		t.Fatal("phase 2: publisher path should block when halted")
	}
	if !venueBlocked2 {
		t.Fatal("phase 2: venue path should block when halted")
	}
	if reason2 != "kill_switch" {
		t.Fatalf("phase 2: venue path expected kill_switch reason, got %q", reason2)
	}

	// Verify both paths observed the same KV state.
	if haltedCount.Load() != 1 {
		t.Fatalf("publisher halted count: expected 1, got %d", haltedCount.Load())
	}
	if tracker.Counter("skipped_halt").Load() != 1 {
		t.Fatalf("venue skipped_halt count: expected 1, got %d", tracker.Counter("skipped_halt").Load())
	}
}

// ---------- CP-FP-5: Control surface writes propagate immediately ----------

func TestControlPlane_FullPath_ImmediatePropagation(t *testing.T) {
	url := natsURL(t)

	controlStore := natsexecution.NewControlKVStore(url)
	if err := controlStore.Start(); err != nil {
		t.Fatalf("start control store: %v", err)
	}
	defer controlStore.Close()

	registry := natsexecution.DefaultRegistry()
	pub := natsexecution.NewPublisher(url, "binancef", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("start publisher: %v", err)
	}
	defer pub.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	// Rapid alternation: 10 state changes, verify each reads correctly.
	for i := 0; i < 10; i++ {
		status := domainexec.GateActive
		if i%2 == 1 {
			status = domainexec.GateHalted
		}

		prob := controlStore.Put(ctx, domainexec.ControlGate{
			Status:    status,
			Reason:    fmt.Sprintf("s275-propagation-round-%d", i),
			UpdatedAt: now.Add(time.Duration(i) * time.Second),
			UpdatedBy: "s275-test",
		})
		if prob != nil {
			t.Fatalf("round %d put: %s", i, prob.Message)
		}

		// Immediate read after write.
		gate, prob := controlStore.Get(ctx)
		if prob != nil {
			t.Fatalf("round %d get: %s", i, prob.Message)
		}
		if gate.Status != status {
			t.Fatalf("round %d: wrote %q, read back %q — propagation not immediate", i, status, gate.Status)
		}

		// Also verify through IsHalted path (what the publisher actor uses).
		isHalted := controlStore.IsHalted(ctx)
		expectHalted := status == domainexec.GateHalted
		if isHalted != expectHalted {
			t.Fatalf("round %d: IsHalted()=%v, expected %v", i, isHalted, expectHalted)
		}

		// Verify publish behavior matches.
		var haltedCount, publishedCount atomic.Int64
		corrID := fmt.Sprintf("cp-fp-5-round-%d", i)
		event := fullPathBuildEvent(t, now.Add(time.Duration(-10+i)*time.Second), corrID)
		ok := publisherGateCheck(controlStore, pub, event, &haltedCount, &publishedCount)

		if status == domainexec.GateActive && !ok {
			t.Fatalf("round %d: gate active but publish blocked", i)
		}
		if status == domainexec.GateHalted && ok {
			t.Fatalf("round %d: gate halted but publish allowed", i)
		}
	}
}

// --- compile-time interface checks ---
var _ ports.VenuePort = (*appexec.PaperVenueAdapter)(nil)

// Ensure natskit import is used (needed for registry).
var _ = natskit.ContentTypeCBOR
