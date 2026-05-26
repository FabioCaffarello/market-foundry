package execution_test

import (
	"context"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/events"
	"internal/shared/healthz"
)

// ---------- S270: SafetyGate Actor Path Integration ----------
//
// These tests prove the SafetyGate in the exact operational flow used by
// VenueAdapterActor.onIntent(). Each test follows the same decision path:
//
//   1. Build a PaperOrderSubmittedEvent (as derive scope would produce)
//   2. SafetyGate.Check(intent.Timestamp, now) — kill switch + staleness
//   3. If allowed, submit to PaperVenueAdapter via VenuePort.SubmitOrder()
//   4. If blocked, increment the appropriate counter (skipped_halt / skipped_stale)
//
// This mirrors VenueAdapterActor.onIntent() (venue_adapter_actor.go:108-155)
// line by line, proving the safety gate at the operational integration level
// without requiring NATS infrastructure.

// buildFreshIntent constructs a valid, fresh PaperOrderSubmittedEvent
// that would normally be produced by PaperOrderEvaluatorActor.
func buildFreshIntent(t *testing.T, ts time.Time) domainexec.PaperOrderSubmittedEvent {
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
	intent.CorrelationID = "corr-s270"
	intent.CausationID = "cause-s270"

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

// venueAdapterOnIntent replicates VenueAdapterActor.onIntent() logic exactly:
// gate check → venue submit → fill event construction.
// Returns (fillEvent, blocked, blockReason).
func venueAdapterOnIntent(
	gate *appexec.SafetyGate,
	venue ports.VenuePort,
	tracker *healthz.Tracker,
	event domainexec.PaperOrderSubmittedEvent,
	now time.Time,
) (*domainexec.VenueOrderFilledEvent, bool, string) {
	intent := event.ExecutionIntent

	// Counter: processed.
	if tracker != nil {
		tracker.Counter("processed").Add(1)
		tracker.Counter("processed:" + intent.VenueSymbol()).Add(1)
	}

	// Gates 1+2: Kill switch and staleness guard.
	verdict := gate.Check(intent.Timestamp, now)
	if !verdict.Allowed {
		switch verdict.Reason {
		case "kill_switch":
			if tracker != nil {
				tracker.Counter("skipped_halt").Add(1)
			}
		case "stale":
			if tracker != nil {
				tracker.Counter("skipped_stale").Add(1)
			}
		default:
			if tracker != nil {
				tracker.RecordError()
			}
		}
		return nil, true, verdict.Reason
	}

	// Gate 3: Submit to venue adapter.
	receipt, prob := venue.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		if tracker != nil {
			tracker.RecordError()
		}
		return nil, true, "venue_error"
	}

	// Construct fill event.
	fillEvent := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(event.Metadata.CorrelationID).
			WithCausationID(event.Metadata.ID),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	if tracker != nil {
		tracker.RecordEvent()
		tracker.Counter("filled").Add(1)
		tracker.Counter("filled:" + intent.VenueSymbol()).Add(1)
	}

	return &fillEvent, false, ""
}

// ---------- Scenario: Fresh intent + active gate = allowed ----------

func TestSafetyGateIntegration_FreshIntent_ActiveGate_Allowed(t *testing.T) {
	now := time.Now().UTC()
	intentTS := now.Add(-30 * time.Second) // 30s old — well within 2min staleness window

	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("venue-adapter-test")

	event := buildFreshIntent(t, intentTS)

	fillEvent, blocked, reason := venueAdapterOnIntent(gate, venue, tracker, event, now)
	if blocked {
		t.Fatalf("expected allowed, got blocked with reason: %q", reason)
	}
	if fillEvent == nil {
		t.Fatal("expected fill event, got nil")
	}

	// Verify fill event integrity.
	if fillEvent.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled status, got %q", fillEvent.ExecutionIntent.Status)
	}
	if fillEvent.ExecutionIntent.Side != domainexec.SideBuy {
		t.Fatalf("expected buy side, got %q", fillEvent.ExecutionIntent.Side)
	}
	if fillEvent.VenueOrderID == "" {
		t.Fatal("expected venue order ID to be generated")
	}
	if !fillEvent.ExecutionIntent.Fills[0].Simulated {
		t.Fatal("paper fill must be simulated")
	}

	// Verify trace preservation.
	if fillEvent.Metadata.CorrelationID != "corr-s270" {
		t.Fatalf("trace broken: expected correlation_id corr-s270, got %q", fillEvent.Metadata.CorrelationID)
	}

	// Verify counters.
	if tracker.Counter("processed").Load() != 1 {
		t.Fatalf("expected processed=1, got %d", tracker.Counter("processed").Load())
	}
	if tracker.Counter("filled").Load() != 1 {
		t.Fatalf("expected filled=1, got %d", tracker.Counter("filled").Load())
	}
	if tracker.Counter("skipped_halt").Load() != 0 {
		t.Fatalf("expected skipped_halt=0, got %d", tracker.Counter("skipped_halt").Load())
	}
	if tracker.Counter("skipped_stale").Load() != 0 {
		t.Fatalf("expected skipped_stale=0, got %d", tracker.Counter("skipped_stale").Load())
	}
}

// ---------- Scenario: Kill switch halted = blocked ----------

func TestSafetyGateIntegration_KillSwitch_Halted_BlocksExecution(t *testing.T) {
	now := time.Now().UTC()
	intentTS := now.Add(-30 * time.Second) // Fresh intent, but kill switch is on.

	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: true},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("venue-adapter-test")

	event := buildFreshIntent(t, intentTS)

	fillEvent, blocked, reason := venueAdapterOnIntent(gate, venue, tracker, event, now)
	if !blocked {
		t.Fatal("expected blocked by kill switch")
	}
	if reason != "kill_switch" {
		t.Fatalf("expected reason kill_switch, got %q", reason)
	}
	if fillEvent != nil {
		t.Fatal("expected no fill event when kill switch is halted")
	}

	// Verify counters.
	if tracker.Counter("processed").Load() != 1 {
		t.Fatalf("expected processed=1, got %d", tracker.Counter("processed").Load())
	}
	if tracker.Counter("skipped_halt").Load() != 1 {
		t.Fatalf("expected skipped_halt=1, got %d", tracker.Counter("skipped_halt").Load())
	}
	if tracker.Counter("filled").Load() != 0 {
		t.Fatalf("expected filled=0, got %d", tracker.Counter("filled").Load())
	}
}

// ---------- Scenario: Stale intent = blocked ----------

func TestSafetyGateIntegration_StaleIntent_BlocksExecution(t *testing.T) {
	now := time.Now().UTC()
	intentTS := now.Add(-5 * time.Minute) // 5min old — exceeds 2min staleness window.

	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false}, // Kill switch is NOT halted.
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("venue-adapter-test")

	event := buildFreshIntent(t, intentTS)

	fillEvent, blocked, reason := venueAdapterOnIntent(gate, venue, tracker, event, now)
	if !blocked {
		t.Fatal("expected blocked by staleness guard")
	}
	if reason != "stale" {
		t.Fatalf("expected reason stale, got %q", reason)
	}
	if fillEvent != nil {
		t.Fatal("expected no fill event when intent is stale")
	}

	// Verify counters.
	if tracker.Counter("processed").Load() != 1 {
		t.Fatalf("expected processed=1, got %d", tracker.Counter("processed").Load())
	}
	if tracker.Counter("skipped_stale").Load() != 1 {
		t.Fatalf("expected skipped_stale=1, got %d", tracker.Counter("skipped_stale").Load())
	}
	if tracker.Counter("filled").Load() != 0 {
		t.Fatalf("expected filled=0, got %d", tracker.Counter("filled").Load())
	}
	if tracker.Counter("skipped_halt").Load() != 0 {
		t.Fatalf("expected skipped_halt=0 (staleness, not halt), got %d", tracker.Counter("skipped_halt").Load())
	}
}

// ---------- Scenario: Kill switch priority over staleness ----------

func TestSafetyGateIntegration_KillSwitch_Priority_OverStaleness(t *testing.T) {
	now := time.Now().UTC()
	intentTS := now.Add(-10 * time.Minute) // Very stale AND kill switch halted.

	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: true},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("venue-adapter-test")

	event := buildFreshIntent(t, intentTS)

	fillEvent, blocked, reason := venueAdapterOnIntent(gate, venue, tracker, event, now)
	if !blocked {
		t.Fatal("expected blocked")
	}
	if reason != "kill_switch" {
		t.Fatalf("expected kill_switch to take priority over stale, got %q", reason)
	}
	if fillEvent != nil {
		t.Fatal("expected no fill event")
	}

	// Kill switch incremented, NOT staleness.
	if tracker.Counter("skipped_halt").Load() != 1 {
		t.Fatalf("expected skipped_halt=1, got %d", tracker.Counter("skipped_halt").Load())
	}
	if tracker.Counter("skipped_stale").Load() != 0 {
		t.Fatalf("expected skipped_stale=0 (kill switch has priority), got %d", tracker.Counter("skipped_stale").Load())
	}
}

// ---------- Scenario: Kill switch nil (fail-open) + fresh = allowed ----------

func TestSafetyGateIntegration_KillSwitchNil_FailOpen_FreshAllowed(t *testing.T) {
	now := time.Now().UTC()
	intentTS := now.Add(-30 * time.Second)

	// No gate checker — simulates NATS KV store unavailable at startup.
	gate := appexec.NewSafetyGate(
		nil,
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("venue-adapter-test")

	event := buildFreshIntent(t, intentTS)

	fillEvent, blocked, reason := venueAdapterOnIntent(gate, venue, tracker, event, now)
	if blocked {
		t.Fatalf("expected allowed (fail-open + fresh), got blocked with reason: %q", reason)
	}
	if fillEvent == nil {
		t.Fatal("expected fill event")
	}
	if fillEvent.ExecutionIntent.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %q", fillEvent.ExecutionIntent.Status)
	}
}

// ---------- Scenario: Kill switch nil (fail-open) + stale = blocked ----------

func TestSafetyGateIntegration_KillSwitchNil_StaleStillBlocked(t *testing.T) {
	now := time.Now().UTC()
	intentTS := now.Add(-5 * time.Minute)

	// No gate checker (fail-open), but intent is stale.
	gate := appexec.NewSafetyGate(
		nil,
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("venue-adapter-test")

	event := buildFreshIntent(t, intentTS)

	fillEvent, blocked, reason := venueAdapterOnIntent(gate, venue, tracker, event, now)
	if !blocked {
		t.Fatal("expected blocked by staleness even when kill switch is unavailable")
	}
	if reason != "stale" {
		t.Fatalf("expected reason stale, got %q", reason)
	}
	if fillEvent != nil {
		t.Fatal("expected no fill event")
	}
}

// ---------- Scenario: Multi-intent sequence with gate state change ----------

func TestSafetyGateIntegration_SequentialIntents_GateStateChange(t *testing.T) {
	now := time.Now().UTC()
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("venue-adapter-test")

	// Phase 1: gate active — intent is allowed.
	gateChecker := &mockGateChecker{halted: false}
	gate := appexec.NewSafetyGate(gateChecker, 2*time.Second, appexec.NewStalenessGuard(2*time.Minute))

	event1 := buildFreshIntent(t, now.Add(-10*time.Second))
	_, blocked1, _ := venueAdapterOnIntent(gate, venue, tracker, event1, now)
	if blocked1 {
		t.Fatal("phase 1: expected allowed")
	}

	// Phase 2: kill switch flips to halted — next intent is blocked.
	gateChecker.halted = true

	event2 := buildFreshIntent(t, now.Add(-10*time.Second))
	_, blocked2, reason2 := venueAdapterOnIntent(gate, venue, tracker, event2, now)
	if !blocked2 {
		t.Fatal("phase 2: expected blocked after kill switch activation")
	}
	if reason2 != "kill_switch" {
		t.Fatalf("phase 2: expected kill_switch, got %q", reason2)
	}

	// Phase 3: kill switch deactivated — next intent is allowed again.
	gateChecker.halted = false

	event3 := buildFreshIntent(t, now.Add(-10*time.Second))
	_, blocked3, _ := venueAdapterOnIntent(gate, venue, tracker, event3, now)
	if blocked3 {
		t.Fatal("phase 3: expected allowed after kill switch deactivation")
	}

	// Verify cumulative counters.
	if tracker.Counter("processed").Load() != 3 {
		t.Fatalf("expected processed=3, got %d", tracker.Counter("processed").Load())
	}
	if tracker.Counter("filled").Load() != 2 {
		t.Fatalf("expected filled=2, got %d", tracker.Counter("filled").Load())
	}
	if tracker.Counter("skipped_halt").Load() != 1 {
		t.Fatalf("expected skipped_halt=1, got %d", tracker.Counter("skipped_halt").Load())
	}
}

// ---------- Scenario: Exact staleness boundary (2min) = allowed ----------

func TestSafetyGateIntegration_ExactBoundary_Allowed(t *testing.T) {
	now := time.Now().UTC()
	intentTS := now.Add(-2 * time.Minute) // Exactly at boundary — NOT stale (uses >).

	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("venue-adapter-test")

	event := buildFreshIntent(t, intentTS)

	fillEvent, blocked, reason := venueAdapterOnIntent(gate, venue, tracker, event, now)
	if blocked {
		t.Fatalf("expected allowed at exact boundary, got blocked with reason: %q", reason)
	}
	if fillEvent == nil {
		t.Fatal("expected fill event at exact boundary")
	}
}

// ---------- Scenario: 1ns past boundary = stale ----------

func TestSafetyGateIntegration_OnePastBoundary_Stale(t *testing.T) {
	now := time.Now().UTC()
	intentTS := now.Add(-2*time.Minute - 1*time.Nanosecond) // 1ns past boundary — stale.

	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("venue-adapter-test")

	event := buildFreshIntent(t, intentTS)

	_, blocked, reason := venueAdapterOnIntent(gate, venue, tracker, event, now)
	if !blocked {
		t.Fatal("expected blocked at 1ns past boundary")
	}
	if reason != "stale" {
		t.Fatalf("expected reason stale, got %q", reason)
	}
}

// ---------- Scenario: No-action intent (side=none) with gate active ----------

func TestSafetyGateIntegration_NoActionIntent_GateActive_AllowedThrough(t *testing.T) {
	now := time.Now().UTC()
	intentTS := now.Add(-30 * time.Second)

	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("venue-adapter-test")

	// Build a no-action intent (rejected risk → side=none).
	eval := appexec.NewPaperOrderEvaluator("binancef", "btcusdt", 60)
	intent, ok := eval.Evaluate(
		"position_exposure", "rejected", "0.30", "0.02",
		"long", "0.72",
		"mean_reversion_entry", "high",
		60, intentTS,
	)
	if !ok {
		t.Fatal("evaluation should succeed")
	}
	intent.CorrelationID = "corr-noaction"
	intent.CausationID = "cause-noaction"

	sim := &appexec.PaperFillSimulator{}
	intent, _ = sim.SimulateFill(intent)

	event := domainexec.PaperOrderSubmittedEvent{
		Metadata:        events.NewMetadata().WithCorrelationID("corr-noaction").WithCausationID("cause-noaction"),
		ExecutionIntent: intent,
	}

	fillEvent, blocked, reason := venueAdapterOnIntent(gate, venue, tracker, event, now)
	if blocked {
		t.Fatalf("expected no-action intent to pass gate, got blocked with reason: %q", reason)
	}
	if fillEvent == nil {
		t.Fatal("expected fill event for no-action intent")
	}
	// No-action: accepted but not filled (side=none).
	if fillEvent.ExecutionIntent.Side != domainexec.SideNone {
		t.Fatalf("expected side none, got %q", fillEvent.ExecutionIntent.Side)
	}
}

// ---------- Scenario: No-action intent with kill switch = still blocked ----------

func TestSafetyGateIntegration_NoActionIntent_KillSwitch_Blocked(t *testing.T) {
	now := time.Now().UTC()
	intentTS := now.Add(-30 * time.Second)

	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: true},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	venue := appexec.NewPaperVenueAdapter(0)
	tracker := healthz.NewTracker("venue-adapter-test")

	// Build no-action intent.
	eval := appexec.NewPaperOrderEvaluator("binancef", "btcusdt", 60)
	intent, _ := eval.Evaluate(
		"position_exposure", "rejected", "0.30", "0.02",
		"long", "0.72",
		"mean_reversion_entry", "high",
		60, intentTS,
	)
	intent.CorrelationID = "corr-noaction-halt"
	intent.CausationID = "cause-noaction-halt"

	sim := &appexec.PaperFillSimulator{}
	intent, _ = sim.SimulateFill(intent)

	event := domainexec.PaperOrderSubmittedEvent{
		Metadata:        events.NewMetadata().WithCorrelationID("corr-noaction-halt").WithCausationID("cause-noaction-halt"),
		ExecutionIntent: intent,
	}

	_, blocked, reason := venueAdapterOnIntent(gate, venue, tracker, event, now)
	if !blocked {
		t.Fatal("expected even no-action intents blocked by kill switch")
	}
	if reason != "kill_switch" {
		t.Fatalf("expected kill_switch, got %q", reason)
	}
}
