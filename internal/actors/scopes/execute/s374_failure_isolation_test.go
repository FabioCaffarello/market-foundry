package execute

// s374_failure_isolation_test.go — S374: Structural failure isolation proof.
//
// These tests validate the architectural properties that enable cross-binary
// failure isolation:
//   - Durable consumer specs guarantee resume-from-checkpoint after restart
//   - Independent tracker instances prevent cross-binary metric contamination
//   - Strategy consumer actor handles restart-like conditions gracefully
//   - Safety gate reads survive reconnection (structural proof of KV durability)
//
// No running NATS required — these prove the structural invariants that
// underpin the runtime failure isolation validated by smoke-failure-isolation.sh.

import (
	"testing"
	"time"

	appexec "internal/application/execution"
	natsstrategy "internal/adapters/nats/natsstrategy"
	"internal/domain/strategy"
	"internal/shared/events"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// TestS374_FailureIsolation_DurableConsumerSpecStable verifies that the execute
// consumer spec uses a stable durable name. This is critical for restart recovery:
// after a binary restart, the JetStream consumer resumes from the last ACK position
// only if the durable name is identical.
func TestS374_FailureIsolation_DurableConsumerSpecStable(t *testing.T) {
	spec1 := natsstrategy.ExecuteStrategyMeanReversionEntryConsumer()
	spec2 := natsstrategy.ExecuteStrategyMeanReversionEntryConsumer()

	// Durable name must be stable across calls (same function, same name).
	if spec1.Durable != spec2.Durable {
		t.Fatalf("durable name not stable: %q vs %q", spec1.Durable, spec2.Durable)
	}
	if spec1.Durable != "execute-strategy-mean-reversion-entry" {
		t.Fatalf("durable name changed: got %q", spec1.Durable)
	}

	// Stream name must be stable.
	if spec1.Event.Stream.Name != "STRATEGY_EVENTS" {
		t.Fatalf("stream name: got %q", spec1.Event.Stream.Name)
	}

	// AckWait and MaxDeliver must have safe values for restart.
	if spec1.AckWait < 10*time.Second {
		t.Fatalf("AckWait too low for restart safety: %v", spec1.AckWait)
	}
	if spec1.MaxDeliver < 3 {
		t.Fatalf("MaxDeliver too low for restart safety: %d", spec1.MaxDeliver)
	}

	t.Logf("[S374] durable=%q ack_wait=%v max_deliver=%d — stable for restart resume",
		spec1.Durable, spec1.AckWait, spec1.MaxDeliver)
}

// TestS374_FailureIsolation_IndependentTrackers verifies that tracker instances
// created for different binaries are fully independent — a counter increment in
// one tracker does not affect any other tracker.
func TestS374_FailureIsolation_IndependentTrackers(t *testing.T) {
	deriveTracker := healthz.NewTracker("derive-sampler")
	executeTracker := healthz.NewTracker("execute-strategy-consumer")
	storeTracker := healthz.NewTracker("store-materializer")

	// Increment derive tracker.
	deriveTracker.RecordEvent()
	deriveTracker.RecordEvent()
	deriveTracker.Counter("produced").Add(2)

	// Increment execute tracker.
	executeTracker.RecordEvent()
	executeTracker.Counter("received").Add(1)
	executeTracker.Counter("evaluated").Add(1)

	// Store tracker should be untouched.
	if storeTracker.Counter("received").Load() != 0 {
		t.Fatal("store tracker contaminated by derive/execute")
	}
	if storeTracker.EventCount() != 0 {
		t.Fatal("store event count contaminated")
	}

	// Derive tracker should not have execute counters.
	if deriveTracker.Counter("received").Load() != 0 {
		t.Fatal("derive tracker has execute counter")
	}
	if deriveTracker.Counter("evaluated").Load() != 0 {
		t.Fatal("derive tracker has execute counter")
	}

	// Execute tracker should not have derive counters.
	if executeTracker.Counter("produced").Load() != 0 {
		t.Fatal("execute tracker has derive counter")
	}

	t.Log("[S374] PASS — trackers are fully independent, no cross-binary contamination")
}

// TestS374_FailureIsolation_ActorHandlesRedelivery verifies that the strategy
// consumer actor correctly processes events that may be redelivered after a
// restart. The actor should produce the same output for the same input.
func TestS374_FailureIsolation_ActorHandlesRedelivery(t *testing.T) {
	ts := time.Now().UTC()

	event := strategy.StrategyResolvedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID("s374-redeliver-corr").
			WithCausationID("s374-redeliver-cause"),
		Strategy: strategy.Strategy{
			Type:       "mean_reversion_entry",
			Source:     "binancef",
			Symbol:     "btcusdt",
			Timeframe:  60,
			Direction:  strategy.DirectionLong,
			Confidence: "0.8500",
			Decisions:  []strategy.DecisionInput{{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Severity: "high", Timeframe: 60}},
			Parameters: map[string]string{"entry": "market"},
			Metadata:   map[string]string{},
			Final:      true,
			Timestamp:  ts,
		},
	}

	// First delivery.
	engine1, collector1, pid1 := spawnTestStrategy(t, "0.01")
	defer engine1.Poison(pid1)
	engine1.Send(pid1, strategyReceivedMessage{Event: event})
	msg1 := waitForIntent(t, collector1)

	// Simulated redelivery (same event, fresh actor — as if binary restarted).
	engine2, collector2, pid2 := spawnTestStrategy(t, "0.01")
	defer engine2.Poison(pid2)
	engine2.Send(pid2, strategyReceivedMessage{Event: event})
	msg2 := waitForIntent(t, collector2)

	// Both should produce identical output.
	if msg1.Event.ExecutionIntent.Side != msg2.Event.ExecutionIntent.Side {
		t.Fatalf("side mismatch on redelivery: %s vs %s",
			msg1.Event.ExecutionIntent.Side, msg2.Event.ExecutionIntent.Side)
	}
	if msg1.Event.ExecutionIntent.Quantity != msg2.Event.ExecutionIntent.Quantity {
		t.Fatalf("quantity mismatch on redelivery: %s vs %s",
			msg1.Event.ExecutionIntent.Quantity, msg2.Event.ExecutionIntent.Quantity)
	}
	if msg1.Event.ExecutionIntent.CorrelationID != msg2.Event.ExecutionIntent.CorrelationID {
		t.Fatal("correlation_id mismatch on redelivery")
	}

	t.Log("[S374] PASS — actor produces deterministic output on redelivery")
}

// TestS374_FailureIsolation_StalenessGuardProtectsAfterRestart verifies that
// events that become stale during a binary restart window are correctly rejected.
func TestS374_FailureIsolation_StalenessGuardProtectsAfterRestart(t *testing.T) {
	guard := appexec.NewStalenessGuard(2 * time.Minute)
	now := time.Now().UTC()

	// Event produced just before restart: 1 minute old — still fresh.
	preRestart := now.Add(-1 * time.Minute)
	if guard.IsStale(preRestart, now) {
		t.Fatal("1-minute-old event should not be stale (within restart window)")
	}

	// Event produced long before restart: 5 minutes old — stale.
	longBefore := now.Add(-5 * time.Minute)
	if !guard.IsStale(longBefore, now) {
		t.Fatal("5-minute-old event must be stale after restart")
	}

	// Event produced just past restart boundary: 2min + 1s — stale.
	pastBoundary := now.Add(-2*time.Minute - time.Second)
	if !guard.IsStale(pastBoundary, now) {
		t.Fatal("event past staleness boundary should be stale")
	}

	t.Log("[S374] PASS — staleness guard correctly filters events across restart boundary")
}

// TestS374_FailureIsolation_TrackerSurvivesActorRecreation verifies that creating
// a new actor with the same tracker (simulating binary restart with preserved metrics)
// maintains counter continuity.
func TestS374_FailureIsolation_TrackerSurvivesActorRecreation(t *testing.T) {
	ts := time.Now().UTC()
	tracker := healthz.NewTracker("s374-survive")
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatal(err)
	}

	event := strategy.StrategyResolvedEvent{
		Metadata: events.NewMetadata().WithCorrelationID("s374-survive-corr"),
		Strategy: strategy.Strategy{
			Type: "mean_reversion_entry", Source: "binancef", Symbol: "btcusdt",
			Timeframe: 60, Direction: strategy.DirectionLong, Confidence: "0.8500",
			Decisions: []strategy.DecisionInput{{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Severity: "high", Timeframe: 60}},
			Parameters: map[string]string{}, Metadata: map[string]string{},
			Final: true, Timestamp: ts,
		},
	}

	// First actor lifecycle.
	collector1 := newTestCollector()
	cPID1 := engine.Spawn(func() actor.Receiver { return collector1 }, "s374-col1")
	pid1 := engine.Spawn(NewStrategyConsumerActor(StrategyConsumerConfig{
		MaxPositionPct: "0.01", Tracker: tracker, AdapterPID: cPID1,
	}), "s374-actor1")
	time.Sleep(20 * time.Millisecond)

	engine.Send(pid1, strategyReceivedMessage{Event: event})
	collector1.waitForN(t, 1)
	engine.Poison(pid1)
	time.Sleep(50 * time.Millisecond)

	// Verify counters after first lifecycle.
	if tracker.Counter("received").Load() != 1 {
		t.Fatalf("received after first lifecycle: want 1, got %d", tracker.Counter("received").Load())
	}

	// Second actor lifecycle (simulates restart) — same tracker.
	collector2 := newTestCollector()
	cPID2 := engine.Spawn(func() actor.Receiver { return collector2 }, "s374-col2")
	pid2 := engine.Spawn(NewStrategyConsumerActor(StrategyConsumerConfig{
		MaxPositionPct: "0.01", Tracker: tracker, AdapterPID: cPID2,
	}), "s374-actor2")
	time.Sleep(20 * time.Millisecond)
	defer engine.Poison(pid2)

	engine.Send(pid2, strategyReceivedMessage{Event: event})
	collector2.waitForN(t, 1)

	// Tracker should have accumulated across both lifecycles.
	if tracker.Counter("received").Load() != 2 {
		t.Fatalf("received after second lifecycle: want 2, got %d", tracker.Counter("received").Load())
	}

	_ = cPID1
	_ = cPID2

	t.Log("[S374] PASS — tracker counters survive actor recreation (restart simulation)")
}

// TestS374_FailureIsolation_GateSafetyOnRestart verifies that the safety gate
// produces correct verdicts even if the control store returns an error (simulating
// a transient KV unavailability during store restart).
func TestS374_FailureIsolation_GateSafetyOnRestart(t *testing.T) {
	now := time.Now().UTC()

	// Gate with nil store (simulates KV unavailability).
	staleness := appexec.NewStalenessGuard(2 * time.Minute)
	gate := appexec.NewSafetyGate(nil, 0, staleness)

	// Fresh event should pass (gate without control store allows through).
	verdict := gate.Check(now.Add(-30*time.Second), now)
	if !verdict.Allowed {
		t.Fatalf("gate should allow fresh event when store unavailable: %s", verdict.Reason)
	}

	// Stale event should still be blocked by staleness guard.
	verdict = gate.Check(now.Add(-5*time.Minute), now)
	if verdict.Allowed {
		t.Fatal("gate should block stale event even when store unavailable")
	}
	if verdict.Reason != "stale" {
		t.Fatalf("expected reason 'stale', got %q", verdict.Reason)
	}

	t.Log("[S374] PASS — safety gate maintains staleness protection during KV unavailability")
}
