//go:build integration

package natsexecution_test

import (
	"context"
	"testing"
	"time"

	"internal/adapters/nats/natsexecution"
	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/events"
	"internal/shared/healthz"
)

// control_gate_runtime_test.go — S273: ControlGate Runtime Halt/Resume Proof.
//
// These tests prove the dynamic halt/resume cycle of the ControlGate using the
// real NATS KV store (ControlKVStore), wired through SafetyGate exactly as the
// production VenueAdapterActor does. They close the S269 open debt by proving:
//
//   - CG-RT-1: Default state (no key) → fail-open → intent flows
//   - CG-RT-2: active → halted transition blocks subsequent intents
//   - CG-RT-3: halted → active transition resumes intent flow
//   - CG-RT-4: Full cycle active→halted→active→halted is repeatable
//   - CG-RT-5: Audit fields (reason, updated_by) survive the round-trip
//   - CG-RT-6: Counters track halt/resume transitions accurately
//
// Requires a running NATS server. Skipped automatically when unreachable.

// runtimeBuildIntent constructs a valid, fresh PaperOrderSubmittedEvent for runtime tests.
func runtimeBuildIntent(t *testing.T, ts time.Time) domainexec.PaperOrderSubmittedEvent {
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
	intent.CorrelationID = "corr-s273-runtime"
	intent.CausationID = "cause-s273-runtime"

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

// runtimeOnIntent replicates VenueAdapterActor.onIntent() logic:
// gate check → venue submit → fill event.
func runtimeOnIntent(
	gate *appexec.SafetyGate,
	venue ports.VenuePort,
	tracker *healthz.Tracker,
	event domainexec.PaperOrderSubmittedEvent,
	now time.Time,
) (fillEvent *domainexec.VenueOrderFilledEvent, blocked bool, reason string) {
	intent := event.ExecutionIntent

	tracker.Counter("processed").Add(1)

	verdict := gate.Check(intent.Timestamp, now)
	if !verdict.Allowed {
		switch verdict.Reason {
		case "kill_switch":
			tracker.Counter("skipped_halt").Add(1)
		case "stale":
			tracker.Counter("skipped_stale").Add(1)
		default:
			tracker.RecordError()
		}
		return nil, true, verdict.Reason
	}

	receipt, prob := venue.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		tracker.RecordError()
		return nil, true, "venue_error"
	}

	fill := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(event.Metadata.CorrelationID).
			WithCausationID(event.Metadata.ID),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	tracker.RecordEvent()
	tracker.Counter("filled").Add(1)

	return &fill, false, ""
}

// ---------- CG-RT-1: Default state (no key) → fail-open → intent flows ----------

func TestControlGateRuntime_DefaultState_FailOpen_IntentFlows(t *testing.T) {
	url := natsURL(t)
	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("start control KV store: %v", err)
	}
	defer store.Close()

	// Do NOT write any gate state — default (key-not-found) should be active.
	now := time.Now().UTC()
	gate := appexec.NewSafetyGate(store, 2*time.Second, appexec.NewStalenessGuard(2*time.Minute))
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("cg-rt-1")

	event := runtimeBuildIntent(t, now.Add(-10*time.Second))
	fillEvent, blocked, reason := runtimeOnIntent(gate, venue, tracker, event, now)
	if blocked {
		t.Fatalf("expected allowed (fail-open default), got blocked: %q", reason)
	}
	if fillEvent == nil {
		t.Fatal("expected fill event")
	}
	if fillEvent.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %q", fillEvent.ExecutionIntent.Status)
	}
	if tracker.Counter("filled").Load() != 1 {
		t.Fatalf("expected filled=1, got %d", tracker.Counter("filled").Load())
	}
}

// ---------- CG-RT-2: active → halted transition blocks subsequent intents ----------

func TestControlGateRuntime_ActiveToHalted_BlocksIntents(t *testing.T) {
	url := natsURL(t)
	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("start control KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	gate := appexec.NewSafetyGate(store, 2*time.Second, appexec.NewStalenessGuard(2*time.Minute))
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("cg-rt-2")

	// Phase 1: Set active explicitly.
	prob := store.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    "s273-runtime-test-start",
		UpdatedAt: now,
		UpdatedBy: "s273-test",
	})
	if prob != nil {
		t.Fatalf("put active: %s", prob.Message)
	}

	event1 := runtimeBuildIntent(t, now.Add(-10*time.Second))
	_, blocked1, _ := runtimeOnIntent(gate, venue, tracker, event1, now)
	if blocked1 {
		t.Fatal("phase 1: expected allowed when gate is active")
	}

	// Phase 2: Transition to halted via real KV write.
	prob = store.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "s273-runtime-test-halt",
		UpdatedAt: now.Add(1 * time.Second),
		UpdatedBy: "s273-test",
	})
	if prob != nil {
		t.Fatalf("put halted: %s", prob.Message)
	}

	event2 := runtimeBuildIntent(t, now.Add(-10*time.Second))
	_, blocked2, reason2 := runtimeOnIntent(gate, venue, tracker, event2, now)
	if !blocked2 {
		t.Fatal("phase 2: expected blocked after halt")
	}
	if reason2 != "kill_switch" {
		t.Fatalf("phase 2: expected kill_switch, got %q", reason2)
	}

	// Verify counters.
	if tracker.Counter("processed").Load() != 2 {
		t.Fatalf("expected processed=2, got %d", tracker.Counter("processed").Load())
	}
	if tracker.Counter("filled").Load() != 1 {
		t.Fatalf("expected filled=1, got %d", tracker.Counter("filled").Load())
	}
	if tracker.Counter("skipped_halt").Load() != 1 {
		t.Fatalf("expected skipped_halt=1, got %d", tracker.Counter("skipped_halt").Load())
	}
}

// ---------- CG-RT-3: halted → active transition resumes intent flow ----------

func TestControlGateRuntime_HaltedToActive_ResumesFlow(t *testing.T) {
	url := natsURL(t)
	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("start control KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	gate := appexec.NewSafetyGate(store, 2*time.Second, appexec.NewStalenessGuard(2*time.Minute))
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("cg-rt-3")

	// Start halted.
	prob := store.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "s273-start-halted",
		UpdatedAt: now,
		UpdatedBy: "s273-test",
	})
	if prob != nil {
		t.Fatalf("put halted: %s", prob.Message)
	}

	event1 := runtimeBuildIntent(t, now.Add(-10*time.Second))
	_, blocked1, reason1 := runtimeOnIntent(gate, venue, tracker, event1, now)
	if !blocked1 {
		t.Fatal("phase 1: expected blocked when starting halted")
	}
	if reason1 != "kill_switch" {
		t.Fatalf("phase 1: expected kill_switch, got %q", reason1)
	}

	// Resume: transition back to active.
	prob = store.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    "s273-resume",
		UpdatedAt: now.Add(1 * time.Second),
		UpdatedBy: "s273-test",
	})
	if prob != nil {
		t.Fatalf("put active: %s", prob.Message)
	}

	event2 := runtimeBuildIntent(t, now.Add(-10*time.Second))
	fillEvent, blocked2, _ := runtimeOnIntent(gate, venue, tracker, event2, now)
	if blocked2 {
		t.Fatal("phase 2: expected allowed after resume")
	}
	if fillEvent == nil {
		t.Fatal("phase 2: expected fill event")
	}
	if fillEvent.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %q", fillEvent.ExecutionIntent.Status)
	}

	// Counters.
	if tracker.Counter("skipped_halt").Load() != 1 {
		t.Fatalf("expected skipped_halt=1, got %d", tracker.Counter("skipped_halt").Load())
	}
	if tracker.Counter("filled").Load() != 1 {
		t.Fatalf("expected filled=1, got %d", tracker.Counter("filled").Load())
	}
}

// ---------- CG-RT-4: Full cycle active→halted→active→halted is repeatable ----------

func TestControlGateRuntime_FullCycle_ActiveHaltedActiveHalted(t *testing.T) {
	url := natsURL(t)
	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("start control KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	gate := appexec.NewSafetyGate(store, 2*time.Second, appexec.NewStalenessGuard(2*time.Minute))
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("cg-rt-4")

	type phase struct {
		status      domainexec.GateStatus
		reason      string
		expectBlock bool
		expectCause string
	}

	phases := []phase{
		{domainexec.GateActive, "s273-cycle-active-1", false, ""},
		{domainexec.GateHalted, "s273-cycle-halted-1", true, "kill_switch"},
		{domainexec.GateActive, "s273-cycle-active-2", false, ""},
		{domainexec.GateHalted, "s273-cycle-halted-2", true, "kill_switch"},
	}

	for i, p := range phases {
		prob := store.Put(ctx, domainexec.ControlGate{
			Status:    p.status,
			Reason:    p.reason,
			UpdatedAt: now.Add(time.Duration(i) * time.Second),
			UpdatedBy: "s273-test",
		})
		if prob != nil {
			t.Fatalf("phase %d: put %s: %s", i+1, p.status, prob.Message)
		}

		event := runtimeBuildIntent(t, now.Add(-10*time.Second))
		_, blocked, reason := runtimeOnIntent(gate, venue, tracker, event, now)

		if blocked != p.expectBlock {
			t.Fatalf("phase %d (%s): expected blocked=%v, got blocked=%v (reason=%q)",
				i+1, p.status, p.expectBlock, blocked, reason)
		}
		if p.expectBlock && reason != p.expectCause {
			t.Fatalf("phase %d (%s): expected reason %q, got %q",
				i+1, p.status, p.expectCause, reason)
		}
	}

	// Cumulative counters: 4 processed, 2 filled, 2 halted.
	if tracker.Counter("processed").Load() != 4 {
		t.Fatalf("expected processed=4, got %d", tracker.Counter("processed").Load())
	}
	if tracker.Counter("filled").Load() != 2 {
		t.Fatalf("expected filled=2, got %d", tracker.Counter("filled").Load())
	}
	if tracker.Counter("skipped_halt").Load() != 2 {
		t.Fatalf("expected skipped_halt=2, got %d", tracker.Counter("skipped_halt").Load())
	}
}

// ---------- CG-RT-5: Audit fields survive round-trip ----------

func TestControlGateRuntime_AuditFields_SurviveRoundTrip(t *testing.T) {
	url := natsURL(t)
	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("start control KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Date(2026, 3, 21, 14, 30, 0, 0, time.UTC)

	// Write gate with full audit fields.
	prob := store.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "risk-limit-breach-btcusdt",
		UpdatedAt: now,
		UpdatedBy: "oncall-operator",
	})
	if prob != nil {
		t.Fatalf("put: %s", prob.Message)
	}

	// Read back and verify.
	gate, prob := store.Get(ctx)
	if prob != nil {
		t.Fatalf("get: %s", prob.Message)
	}
	if gate.Status != domainexec.GateHalted {
		t.Errorf("status: want halted, got %q", gate.Status)
	}
	if gate.Reason != "risk-limit-breach-btcusdt" {
		t.Errorf("reason: want risk-limit-breach-btcusdt, got %q", gate.Reason)
	}
	if gate.UpdatedBy != "oncall-operator" {
		t.Errorf("updated_by: want oncall-operator, got %q", gate.UpdatedBy)
	}
	if !gate.UpdatedAt.Equal(now) {
		t.Errorf("updated_at: want %v, got %v", now, gate.UpdatedAt)
	}

	// Resume with new audit trail.
	prob = store.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateActive,
		Reason:    "limits-reset-confirmed",
		UpdatedAt: now.Add(15 * time.Minute),
		UpdatedBy: "oncall-operator",
	})
	if prob != nil {
		t.Fatalf("put resume: %s", prob.Message)
	}

	gate, prob = store.Get(ctx)
	if prob != nil {
		t.Fatalf("get after resume: %s", prob.Message)
	}
	if gate.Status != domainexec.GateActive {
		t.Errorf("after resume: status want active, got %q", gate.Status)
	}
	if gate.Reason != "limits-reset-confirmed" {
		t.Errorf("after resume: reason want limits-reset-confirmed, got %q", gate.Reason)
	}
}

// ---------- CG-RT-6: Multiple intents during halt show consistent blocking ----------

func TestControlGateRuntime_MultipleIntentsDuringHalt_AllBlocked(t *testing.T) {
	url := natsURL(t)
	store := natsexecution.NewControlKVStore(url)
	if err := store.Start(); err != nil {
		t.Fatalf("start control KV store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	gate := appexec.NewSafetyGate(store, 2*time.Second, appexec.NewStalenessGuard(2*time.Minute))
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("cg-rt-6")

	// Halt the gate.
	prob := store.Put(ctx, domainexec.ControlGate{
		Status:    domainexec.GateHalted,
		Reason:    "s273-sustained-halt-test",
		UpdatedAt: now,
		UpdatedBy: "s273-test",
	})
	if prob != nil {
		t.Fatalf("put halted: %s", prob.Message)
	}

	// Send 5 intents — all must be blocked.
	for i := 0; i < 5; i++ {
		event := runtimeBuildIntent(t, now.Add(-10*time.Second))
		_, blocked, reason := runtimeOnIntent(gate, venue, tracker, event, now)
		if !blocked {
			t.Fatalf("intent %d: expected blocked during halt", i+1)
		}
		if reason != "kill_switch" {
			t.Fatalf("intent %d: expected kill_switch, got %q", i+1, reason)
		}
	}

	if tracker.Counter("processed").Load() != 5 {
		t.Fatalf("expected processed=5, got %d", tracker.Counter("processed").Load())
	}
	if tracker.Counter("skipped_halt").Load() != 5 {
		t.Fatalf("expected skipped_halt=5, got %d", tracker.Counter("skipped_halt").Load())
	}
	if tracker.Counter("filled").Load() != 0 {
		t.Fatalf("expected filled=0, got %d", tracker.Counter("filled").Load())
	}
}
