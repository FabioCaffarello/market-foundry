//go:build integration

package execute_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	executeactor "internal/actors/scopes/execute"
	natsexecution "internal/adapters/nats/natsexecution"
	appexec "internal/application/execution"
	domainexec "internal/domain/execution"
	"internal/shared/clock"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
)

// controlled_activation_verification_test.go — S341: Controlled Activation Verification.
//
// These tests prove the full activation lifecycle on the real actor path:
// NATS JetStream → Hollywood actor → VenueAdapterActor → safety gate → venue submit → fill publish.
//
// Unlike S340 acceptance tests (domain-only, no NATS) and S333 LF tests (single gate state),
// these tests exercise gate state transitions DURING a running supervisor and verify that
// the activation surface controls real event flow through the live actor pipeline.
//
// Requires a running NATS server at localhost:4222 (or NATS_URL env var).

// s341AppConfig builds a test-safe AppConfig for controlled activation.
func s341AppConfig(url string) settings.AppConfig {
	return settings.AppConfig{
		NATS: settings.NATSConfig{Enabled: true, URL: url},
		Venue: settings.VenueConfig{
			Type:            "paper_simulator",
			StalenessMaxAge: "300s",
			SubmitTimeout:   "10s",
		},
	}
}

// s341SpawnSupervisor creates a supervisor and registers cleanup.
func s341SpawnSupervisor(t *testing.T, cfg settings.AppConfig, venue appexec.PaperVenueAdapter, trackers map[string]*healthz.Tracker) *actor.Engine {
	t.Helper()
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	pid := engine.Spawn(
		executeactor.NewExecuteSupervisor(cfg, &venue, nil, trackers),
		fmt.Sprintf("s341-sup-%d", time.Now().UnixNano()),
	)
	t.Cleanup(func() {
		engine.Poison(pid)
		time.Sleep(300 * time.Millisecond)
	})
	time.Sleep(supervisorStartupDelay)
	return engine
}

// s341SetGate sets the execution control gate and returns the store for further use.
func s341SetGate(t *testing.T, url string, status domainexec.GateStatus, reason, updatedBy string) *natsexecution.ControlKVStore {
	t.Helper()
	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("control store: %v", err)
	}
	if prob := store.Put(context.Background(), domainexec.ControlGate{
		Status:    status,
		Reason:    reason,
		UpdatedBy: updatedBy,
		UpdatedAt: time.Now().UTC(),
	}); prob != nil {
		t.Fatalf("[s341] set gate %s: %s", status, prob.Message)
	}
	// Confirm the write is server-visible through the actor's read path
	// before any event is published (G9 hardening).
	waitGateObserved(t, store, status == domainexec.GateHalted, 5*time.Second)
	return store
}

// s341WaitCounter polls a tracker counter until it reaches the target or times out.
func s341WaitCounter(t *testing.T, tracker *healthz.Tracker, name string, target int64, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		if tracker.Counter(name).Load() >= target {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("[s341] counter %q did not reach %d within %s (current=%d)",
				name, target, timeout, tracker.Counter(name).Load())
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// ---------- CAV-1: Halted Gate Blocks Live Path (Precondition) ----------

func TestControlledActivation_HaltedGateBlocksLivePath(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s341-cav1-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s341-cav1-consumer"),
	}

	// Start with gate HALTED — safe default posture.
	controlStore := s341SetGate(t, url, domainexec.GateHalted, "s341-cav1-halted", "s341-test")
	defer controlStore.Close()
	defer func() {
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status: domainexec.GateActive, Reason: "s341-cav1-cleanup",
			UpdatedAt: time.Now().UTC(), UpdatedBy: "s341-test",
		})
	}()

	venue := *appexec.NewPaperVenueAdapter(0)
	s341SpawnSupervisor(t, s341AppConfig(url), venue, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	corrID := fmt.Sprintf("s341-cav1-%d", time.Now().UnixNano())
	event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("publish: %s", prob.Message)
	}

	// Wait for event to reach the actor (proves NATS → actor path is live).
	s341WaitCounter(t, adapterTracker, "processed", 1, 10*time.Second)

	// Verify: event was blocked, not filled.
	if adapterTracker.Counter("skipped_halt").Load() < 1 {
		t.Fatalf("[CAV-1] expected skipped_halt >= 1, got %d", adapterTracker.Counter("skipped_halt").Load())
	}
	if adapterTracker.Counter("filled").Load() != 0 {
		t.Fatalf("[CAV-1] expected filled=0 when halted, got %d", adapterTracker.Counter("filled").Load())
	}

	t.Logf("[CAV-1] processed=%d skipped_halt=%d filled=%d",
		adapterTracker.Counter("processed").Load(),
		adapterTracker.Counter("skipped_halt").Load(),
		adapterTracker.Counter("filled").Load())
	t.Log("[s341/CAV-1] PASS — halted gate blocks live actor path (precondition proven)")
}

// ---------- CAV-2: Gate Open Enables Live Flow ----------

func TestControlledActivation_GateOpenEnablesLiveFlow(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s341-cav2-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s341-cav2-consumer"),
	}

	// Start with gate ACTIVE — simulating operator opening gate after smoke passed.
	controlStore := s341SetGate(t, url, domainexec.GateActive, "s341-cav2-enable", "s341-test")
	defer controlStore.Close()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	venue := *appexec.NewPaperVenueAdapter(0)
	s341SpawnSupervisor(t, s341AppConfig(url), venue, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	corrID := fmt.Sprintf("s341-cav2-%d", time.Now().UnixNano())
	event := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event)
	cancel()
	if prob != nil {
		t.Fatalf("publish: %s", prob.Message)
	}

	fill := fillSub.waitForFill(corrID, 10*time.Second)
	if fill == nil {
		t.Fatal("[CAV-2] fill event not received — gate open did not enable live flow")
	}

	if fill.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("[CAV-2] expected status=filled, got %q", fill.ExecutionIntent.Status)
	}
	eventuallyAtLeast(t, adapterTracker.Counter("filled"), 1, 2*time.Second,
		"[CAV-2] expected filled >= 1")

	t.Logf("[CAV-2] fill received: venue_order_id=%s correlation_id=%s",
		fill.VenueOrderID, fill.Metadata.CorrelationID)
	t.Log("[s341/CAV-2] PASS — gate open enables live flow through real actor pipeline")
}

// ---------- CAV-3: Gate Halt Blocks After Enable ----------

func TestControlledActivation_GateHaltBlocksAfterEnable(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s341-cav3-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s341-cav3-consumer"),
	}

	// Start with gate ACTIVE.
	controlStore := s341SetGate(t, url, domainexec.GateActive, "s341-cav3-enable", "s341-test")
	defer controlStore.Close()
	defer func() {
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status: domainexec.GateActive, Reason: "s341-cav3-cleanup",
			UpdatedAt: time.Now().UTC(), UpdatedBy: "s341-test",
		})
	}()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	venue := *appexec.NewPaperVenueAdapter(0)
	s341SpawnSupervisor(t, s341AppConfig(url), venue, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	// Phase 1: Verify flow is active.
	corrID1 := fmt.Sprintf("s341-cav3-live-%d", time.Now().UnixNano())
	event1 := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event1)
	cancel()
	if prob != nil {
		t.Fatalf("publish phase 1: %s", prob.Message)
	}

	fill1 := fillSub.waitForFill(corrID1, 10*time.Second)
	if fill1 == nil {
		t.Fatal("[CAV-3/phase-1] fill not received — flow should be active")
	}
	t.Logf("[CAV-3/phase-1] fill received while active: %s", fill1.VenueOrderID)

	// The fill-stream signal can lead the adapter's "filled" counter; wait
	// for the counter to reflect phase-1's fill before snapshotting it.
	s341WaitCounter(t, adapterTracker, "filled", 1, 5*time.Second)
	filledBeforeHalt := adapterTracker.Counter("filled").Load()

	// Phase 2: Halt the gate (runtime transition).
	if prob := controlStore.Put(context.Background(), domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "s341-cav3-halt",
		UpdatedBy: "s341-test",
		UpdatedAt: time.Now().UTC(),
	}); prob != nil {
		t.Fatalf("[CAV-3/phase-2] halt gate: %s", prob.Message)
	}
	t.Log("[CAV-3/phase-2] gate halted at runtime")
	waitGateObserved(t, controlStore, true, 5*time.Second)

	// Phase 3: Publish another event — should be blocked.
	corrID2 := fmt.Sprintf("s341-cav3-halted-%d", time.Now().UnixNano())
	event2 := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID2)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	prob = publisher.PublishExecution(ctx, event2)
	cancel()
	if prob != nil {
		t.Fatalf("publish phase 3: %s", prob.Message)
	}

	// Wait for the event to be processed (not filled).
	processedBefore := adapterTracker.Counter("processed").Load()
	s341WaitCounter(t, adapterTracker, "processed", processedBefore+1, 10*time.Second)

	// Verify: filled count did not increase.
	filledAfterHalt := adapterTracker.Counter("filled").Load()
	if filledAfterHalt != filledBeforeHalt {
		t.Fatalf("[CAV-3/phase-3] expected filled unchanged after halt: before=%d after=%d",
			filledBeforeHalt, filledAfterHalt)
	}
	if adapterTracker.Counter("skipped_halt").Load() < 1 {
		t.Fatalf("[CAV-3/phase-3] expected skipped_halt >= 1, got %d",
			adapterTracker.Counter("skipped_halt").Load())
	}

	t.Logf("[CAV-3] filled_before_halt=%d filled_after_halt=%d skipped_halt=%d",
		filledBeforeHalt, filledAfterHalt, adapterTracker.Counter("skipped_halt").Load())
	t.Log("[s341/CAV-3] PASS — gate halt blocks flow after prior enable (runtime transition proven)")
}

// ---------- CAV-4: Full Activation Lifecycle ----------

func TestControlledActivation_FullLifecycle(t *testing.T) {
	url := s333NatsURL(t)

	adapterTracker := healthz.NewTracker("s341-cav4-adapter")
	trackers := map[string]*healthz.Tracker{
		"venue-adapter":  adapterTracker,
		"venue-consumer": healthz.NewTracker("s341-cav4-consumer"),
	}

	// Start with gate HALTED — canonical safe deploy posture.
	controlStore := s341SetGate(t, url, domainexec.GateHalted, "s341-cav4-initial-deploy", "s341-test")
	defer controlStore.Close()
	defer func() {
		controlStore.Put(context.Background(), domainexec.ControlGate{
			Status: domainexec.GateActive, Reason: "s341-cav4-cleanup",
			UpdatedAt: time.Now().UTC(), UpdatedBy: "s341-test",
		})
	}()

	fillSub := newS333FillSubscriber(t, url)
	defer fillSub.close()

	venue := *appexec.NewPaperVenueAdapter(0)
	s341SpawnSupervisor(t, s341AppConfig(url), venue, trackers)

	publisher := natsexecution.NewPublisher(url, "binancef", natsexecution.DefaultRegistry())
	if err := publisher.Start(); err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer publisher.Close()

	// ── Phase 1: Halted — event is blocked ──

	corrID1 := fmt.Sprintf("s341-cav4-halted-%d", time.Now().UnixNano())
	event1 := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	prob := publisher.PublishExecution(ctx, event1)
	cancel()
	if prob != nil {
		t.Fatalf("phase 1 publish: %s", prob.Message)
	}

	s341WaitCounter(t, adapterTracker, "processed", 1, 10*time.Second)

	if adapterTracker.Counter("skipped_halt").Load() < 1 {
		t.Fatal("[CAV-4/phase-1] expected skipped_halt >= 1")
	}
	if adapterTracker.Counter("filled").Load() != 0 {
		t.Fatal("[CAV-4/phase-1] expected filled=0 while halted")
	}
	t.Log("[CAV-4/phase-1] HALTED — event blocked as expected")

	// ── Phase 2: Enable — operator opens gate ──

	if prob := controlStore.Put(context.Background(), domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    "s341-cav4-smoke-passed",
		UpdatedBy: "s341-operator",
		UpdatedAt: time.Now().UTC(),
	}); prob != nil {
		t.Fatalf("[CAV-4/phase-2] enable gate: %s", prob.Message)
	}
	waitGateObserved(t, controlStore, false, 5*time.Second)

	corrID2 := fmt.Sprintf("s341-cav4-live-%d", time.Now().UnixNano())
	event2 := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID2)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	prob = publisher.PublishExecution(ctx, event2)
	cancel()
	if prob != nil {
		t.Fatalf("phase 2 publish: %s", prob.Message)
	}

	fill2 := fillSub.waitForFill(corrID2, 10*time.Second)
	if fill2 == nil {
		t.Fatal("[CAV-4/phase-2] fill not received — gate open did not enable flow")
	}
	if fill2.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("[CAV-4/phase-2] expected status=filled, got %q", fill2.ExecutionIntent.Status)
	}
	t.Logf("[CAV-4/phase-2] ENABLED — fill received: %s", fill2.VenueOrderID)

	// The fill-stream signal (waitForFill) can lead the adapter's "filled"
	// counter increment; wait for the counter to reflect phase-2's fill
	// before snapshotting it, so the phase-3 comparison is deterministic.
	s341WaitCounter(t, adapterTracker, "filled", 1, 5*time.Second)
	filledAfterEnable := adapterTracker.Counter("filled").Load()

	// ── Phase 3: Halt — operator halts gate ──

	if prob := controlStore.Put(context.Background(), domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "s341-cav4-emergency-halt",
		UpdatedBy: "s341-operator",
		UpdatedAt: time.Now().UTC(),
	}); prob != nil {
		t.Fatalf("[CAV-4/phase-3] halt gate: %s", prob.Message)
	}
	waitGateObserved(t, controlStore, true, 5*time.Second)

	corrID3 := fmt.Sprintf("s341-cav4-rehalted-%d", time.Now().UnixNano())
	event3 := s333BuildEvent(t, time.Now().UTC().Add(-10*time.Second), corrID3)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	prob = publisher.PublishExecution(ctx, event3)
	cancel()
	if prob != nil {
		t.Fatalf("phase 3 publish: %s", prob.Message)
	}

	processedBefore := adapterTracker.Counter("processed").Load()
	s341WaitCounter(t, adapterTracker, "processed", processedBefore+1, 10*time.Second)

	filledAfterHalt := adapterTracker.Counter("filled").Load()
	if filledAfterHalt != filledAfterEnable {
		t.Fatalf("[CAV-4/phase-3] filled increased after halt: before=%d after=%d",
			filledAfterEnable, filledAfterHalt)
	}
	t.Log("[CAV-4/phase-3] HALTED — event blocked after re-halt")

	// ── Summary counters ──

	t.Logf("[CAV-4] summary: processed=%d filled=%d skipped_halt=%d",
		adapterTracker.Counter("processed").Load(),
		adapterTracker.Counter("filled").Load(),
		adapterTracker.Counter("skipped_halt").Load())
	t.Log("[s341/CAV-4] PASS — full activation lifecycle: halted → enabled → halted (real actor path)")
}

// ---------- CAV-5: Audit Fields Observable Through Live Path ----------

func TestControlledActivation_AuditFieldsObservable(t *testing.T) {
	url := s333NatsURL(t)

	// Set gate with explicit audit fields.
	auditReason := "s341-cav5-audit-verification"
	auditOperator := "s341-audit-operator"
	auditTime := time.Now().UTC()

	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("control store: %v", err)
	}
	defer store.Close()

	store.Put(context.Background(), domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    auditReason,
		UpdatedBy: auditOperator,
		UpdatedAt: auditTime,
	})

	// Read back and verify audit fields round-trip through NATS KV.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	gate, prob := store.Get(ctx)
	cancel()
	if prob != nil {
		t.Fatalf("gate read: %s", prob.Message)
	}

	if gate.Status != domainexec.GateHalted {
		t.Fatalf("[CAV-5] expected status=halted, got %q", gate.Status)
	}
	if gate.Reason != auditReason {
		t.Fatalf("[CAV-5] expected reason=%q, got %q", auditReason, gate.Reason)
	}
	if gate.UpdatedBy != auditOperator {
		t.Fatalf("[CAV-5] expected updated_by=%q, got %q", auditOperator, gate.UpdatedBy)
	}
	// Time comparison within 1 second tolerance (NATS KV stores as RFC3339).
	if gate.UpdatedAt.Sub(auditTime).Abs() > time.Second {
		t.Fatalf("[CAV-5] expected updated_at ~%v, got %v (delta=%v)",
			auditTime, gate.UpdatedAt, gate.UpdatedAt.Sub(auditTime))
	}

	// Construct activation surface with the retrieved gate — verify it composes correctly.
	surface := domainexec.NewActivationSurface(clock.SystemClock{}, domainexec.AdapterVenue, gate, domainexec.CredentialPresent)
	if surface.Effective != domainexec.ModeVenueHalted {
		t.Fatalf("[CAV-5] expected effective=venue_halted, got %s", surface.Effective)
	}
	if surface.IsLive() {
		t.Fatal("[CAV-5] halted surface should not be live")
	}

	t.Logf("[CAV-5] audit fields: reason=%q updated_by=%q updated_at=%v effective=%s",
		gate.Reason, gate.UpdatedBy, gate.UpdatedAt, surface.Effective)

	// Restore gate for other tests.
	store.Put(context.Background(), domainexec.ControlGate{
		Status: domainexec.GateActive, Reason: "s341-cav5-cleanup",
		UpdatedAt: time.Now().UTC(), UpdatedBy: "s341-test",
	})

	t.Log("[s341/CAV-5] PASS — audit fields round-trip through NATS KV and compose into activation surface")
}
