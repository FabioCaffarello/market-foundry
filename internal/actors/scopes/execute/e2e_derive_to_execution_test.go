package execute

// e2e_derive_to_execution_test.go — S368: End-to-end analytical-to-execution proof.
//
// Proves the complete connected pipeline:
//   derive (MeanReversionEntryResolver) → StrategyResolvedEvent → StrategyConsumerActor
//   → PaperOrderEvaluator → intentReceivedMessage → venue adapter path.
//
// This is the capstone proof of the Derive Integration Wave (S364–S369).
// Each test exercises the real derive resolver, feeds its output to the real
// execute consumer actor, and validates the complete correlation chain, field
// preservation, and behavioral invariants across the derive→execute boundary.
//
// Governing questions answered:
//   - DI-4 Q1: Does a derive-produced event drive correct execution?
//   - DI-4 Q2: Is the correlation chain unbroken from decision to execution?
//   - DI-4 Q3: Are all 11 S359 invariants preserved end-to-end?
//   - DI-4 Q4: Does the safety gate pipeline accept fresh derive-produced events?

import (
	"context"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	appstrategy "internal/application/strategy"
	domainexec "internal/domain/execution"
	"internal/domain/strategy"
	"internal/shared/events"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// --- E2E helper: produce a real derive-shaped StrategyResolvedEvent ---

func deriveResolvedEvent(
	decisionOutcome, decisionConfidence, decisionSeverity string,
	ts time.Time,
) (strategy.StrategyResolvedEvent, bool) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	strat, ok := resolver.Resolve(
		"rsi_oversold", decisionOutcome, decisionConfidence, decisionSeverity,
		"RSI below 30 on btcusdt", 60, ts,
	)
	if !ok {
		return strategy.StrategyResolvedEvent{}, false
	}

	meta := events.NewMetadata().
		WithCorrelationID("e2e-corr-s368").
		WithCausationID("decision-evt-001")

	return strategy.StrategyResolvedEvent{
		Metadata: meta,
		Strategy: strat,
	}, true
}

// ── E2E-1: Triggered decision → long strategy → buy execution ─────

func TestE2E_DeriveTriggered_ProducesBuyExecution(t *testing.T) {
	ts := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)

	// Step 1: Derive produces a real StrategyResolvedEvent.
	event, ok := deriveResolvedEvent("triggered", "0.8500", "high", ts)
	if !ok {
		t.Fatal("derive resolver should produce event for triggered outcome")
	}

	// Verify derive output contract (PI-1, PI-2, PI-5, PI-6).
	if event.Strategy.Type != "mean_reversion_entry" {
		t.Fatalf("PI-1: type want mean_reversion_entry, got %s", event.Strategy.Type)
	}
	if event.Strategy.Direction != strategy.DirectionLong {
		t.Fatalf("PI-2: direction want long, got %s", event.Strategy.Direction)
	}
	if !event.Strategy.Final {
		t.Fatal("PI-5: Final must be true")
	}
	if !event.Strategy.Timestamp.Equal(ts) {
		t.Fatalf("PI-6: timestamp must equal decision timestamp, got %v", event.Strategy.Timestamp)
	}

	// Step 2: Feed to execute consumer actor.
	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: event})
	msg := waitForIntent(t, collector)

	// Step 3: Validate execution output.
	intent := msg.Event.ExecutionIntent
	if intent.Side != domainexec.SideBuy {
		t.Errorf("INV-2: expected side=buy for long direction, got %s", intent.Side)
	}
	if intent.Quantity != "0.01" {
		t.Errorf("expected quantity=0.01, got %s", intent.Quantity)
	}

	// INV-3: Correlation chain preserved.
	if intent.CorrelationID != "e2e-corr-s368" {
		t.Errorf("INV-3: correlation_id want e2e-corr-s368, got %s", intent.CorrelationID)
	}
	if intent.CausationID != event.Metadata.ID {
		t.Errorf("INV-3: causation_id want %s (strategy event ID), got %s", event.Metadata.ID, intent.CausationID)
	}

	// INV-4: Pass-through risk explicit.
	if intent.Risk.Type != "pass_through" {
		t.Errorf("INV-4: risk.type want pass_through, got %s", intent.Risk.Type)
	}
	if intent.Risk.Disposition != "approved" {
		t.Errorf("INV-4: risk.disposition want approved, got %s", intent.Risk.Disposition)
	}

	// INV-5: Strategy timestamp used, not time.Now().
	if !intent.Timestamp.Equal(ts) {
		t.Errorf("INV-5: intent timestamp must equal strategy timestamp %v, got %v", ts, intent.Timestamp)
	}

	// INV-1: Strategy type preserved in risk metadata.
	if intent.Risk.StrategyType != "mean_reversion_entry" {
		t.Errorf("INV-1: risk.strategy_type want mean_reversion_entry, got %s", intent.Risk.StrategyType)
	}

	// Decision severity carried from derive to execution.
	if intent.Risk.DecisionSeverity != "high" {
		t.Errorf("decision severity want high, got %s", intent.Risk.DecisionSeverity)
	}

	// Explainability fields present.
	if intent.Parameters["source_path"] != "strategy_consumer.mean_reversion_entry" {
		t.Errorf("source_path want strategy_consumer.mean_reversion_entry, got %s", intent.Parameters["source_path"])
	}
	if intent.Parameters["evaluation_outcome"] != "actionable" {
		t.Errorf("evaluation_outcome want actionable, got %s", intent.Parameters["evaluation_outcome"])
	}

	// Event-level metadata preserves chain.
	if msg.Event.Metadata.CorrelationID != "e2e-corr-s368" {
		t.Errorf("event metadata correlation_id lost")
	}
}

// ── E2E-2: Not-triggered decision → flat strategy → no execution ──

func TestE2E_DeriveNotTriggered_ProducesNoExecution(t *testing.T) {
	ts := time.Date(2026, 3, 22, 10, 1, 0, 0, time.UTC)

	event, ok := deriveResolvedEvent("not_triggered", "0.9000", "", ts)
	if !ok {
		t.Fatal("derive resolver should produce event for not_triggered outcome")
	}

	// Verify derive output: flat direction, zero confidence (BI-5).
	if event.Strategy.Direction != strategy.DirectionFlat {
		t.Fatalf("BI-5: direction should be flat, got %s", event.Strategy.Direction)
	}
	if event.Strategy.Confidence != "0.0000" {
		t.Fatalf("BI-5: flat confidence should be 0.0000, got %s", event.Strategy.Confidence)
	}

	// Feed to execute consumer.
	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: event})
	msg := waitForIntent(t, collector)

	// INV-7: Flat direction produces side=none.
	intent := msg.Event.ExecutionIntent
	if intent.Side != domainexec.SideNone {
		t.Errorf("INV-7: expected side=none for flat direction, got %s", intent.Side)
	}
	if intent.Quantity != "0" {
		t.Errorf("INV-7: expected quantity=0, got %s", intent.Quantity)
	}

	// Correlation chain still preserved.
	if intent.CorrelationID != "e2e-corr-s368" {
		t.Errorf("INV-3: correlation_id lost on flat path")
	}

	if intent.Parameters["evaluation_outcome"] != "flat" {
		t.Errorf("evaluation_outcome want flat, got %s", intent.Parameters["evaluation_outcome"])
	}
}

// ── E2E-3: Insufficient data → flat strategy → no execution ───────

func TestE2E_DeriveInsufficientData_ProducesNoExecution(t *testing.T) {
	ts := time.Date(2026, 3, 22, 10, 2, 0, 0, time.UTC)

	event, ok := deriveResolvedEvent("insufficient", "0.0000", "", ts)
	if !ok {
		t.Fatal("derive resolver should produce event for insufficient outcome")
	}

	if event.Strategy.Metadata["reason"] != "insufficient_data" {
		t.Fatalf("expected metadata reason=insufficient_data, got %s", event.Strategy.Metadata["reason"])
	}

	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: event})
	msg := waitForIntent(t, collector)

	intent := msg.Event.ExecutionIntent
	if intent.Side != domainexec.SideNone {
		t.Errorf("expected side=none for insufficient data, got %s", intent.Side)
	}
}

// ── E2E-4: Severity scaling flows from derive to execution ────────

func TestE2E_DeriveSeverityScaling_FlowsToExecution(t *testing.T) {
	ts := time.Date(2026, 3, 22, 10, 3, 0, 0, time.UTC)

	severityCases := []struct {
		name            string
		severity        string
		wantConfidence  string
		wantDecSeverity string
	}{
		{"high", "high", "0.8500", "high"},
		{"moderate", "moderate", "0.7650", "moderate"},
		{"low", "low", "0.6800", "low"},
	}

	for _, tc := range severityCases {
		t.Run(tc.name, func(t *testing.T) {
			event, ok := deriveResolvedEvent("triggered", "0.8500", tc.severity, ts)
			if !ok {
				t.Fatal("derive resolver should produce event")
			}

			// Verify severity-scaled confidence from derive (PI-3).
			if event.Strategy.Confidence != tc.wantConfidence {
				t.Fatalf("PI-3: confidence for %s severity want %s, got %s",
					tc.severity, tc.wantConfidence, event.Strategy.Confidence)
			}

			engine, collector, pid := spawnTestStrategy(t, "0.01")
			defer engine.Poison(pid)

			engine.Send(pid, strategyReceivedMessage{Event: event})
			msg := waitForIntent(t, collector)

			// Severity carried to execution risk metadata.
			if msg.Event.ExecutionIntent.Risk.DecisionSeverity != tc.wantDecSeverity {
				t.Errorf("decision severity want %s, got %s",
					tc.wantDecSeverity, msg.Event.ExecutionIntent.Risk.DecisionSeverity)
			}

			// Strategy confidence carried to risk confidence.
			if msg.Event.ExecutionIntent.Risk.Confidence != tc.wantConfidence {
				t.Errorf("risk confidence want %s, got %s",
					tc.wantConfidence, msg.Event.ExecutionIntent.Risk.Confidence)
			}
		})
	}
}

// ── E2E-5: Confidence threshold gate with derive-produced events ──

func TestE2E_ConfidenceThreshold_FiltersDeriveLowConfidence(t *testing.T) {
	ts := time.Date(2026, 3, 22, 10, 4, 0, 0, time.UTC)

	// Low severity produces confidence 0.6800 (0.85 × 0.80).
	event, ok := deriveResolvedEvent("triggered", "0.8500", "low", ts)
	if !ok {
		t.Fatal("derive resolver should produce event")
	}

	// Set threshold at 0.70 — event at 0.6800 should be skipped.
	engine, collector, pid := spawnTestStrategyWithConfig(t, StrategyConsumerConfig{
		MaxPositionPct: "0.01",
		MinConfidence:  "0.70",
	})
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: event})

	time.Sleep(100 * time.Millisecond)
	if len(collector.received) != 0 {
		t.Errorf("expected event skipped below 0.70 threshold, got %d intents", len(collector.received))
	}
}

func TestE2E_ConfidenceThreshold_PassesDeriveHighConfidence(t *testing.T) {
	ts := time.Date(2026, 3, 22, 10, 5, 0, 0, time.UTC)

	// High severity produces confidence 0.8500 (0.85 × 1.00).
	event, ok := deriveResolvedEvent("triggered", "0.8500", "high", ts)
	if !ok {
		t.Fatal("derive resolver should produce event")
	}

	// Set threshold at 0.70 — event at 0.8500 should pass.
	engine, collector, pid := spawnTestStrategyWithConfig(t, StrategyConsumerConfig{
		MaxPositionPct: "0.01",
		MinConfidence:  "0.70",
	})
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: event})
	msg := waitForIntent(t, collector)

	if msg.Event.ExecutionIntent.Side != domainexec.SideBuy {
		t.Errorf("expected buy side for high-confidence event, got %s", msg.Event.ExecutionIntent.Side)
	}
}

// ── E2E-6: Safety gate accepts fresh derive-produced events ───────

func TestE2E_SafetyGate_AcceptsFreshDeriveEvent(t *testing.T) {
	now := time.Now().UTC()

	event, ok := deriveResolvedEvent("triggered", "0.8500", "high", now)
	if !ok {
		t.Fatal("derive resolver should produce event")
	}

	// Verify event timestamp is fresh for safety gate.
	staleness := appexec.NewStalenessGuard(2 * time.Minute)
	if staleness.IsStale(event.Strategy.Timestamp, now) {
		t.Fatal("derive-produced event with current timestamp should not be stale")
	}

	// Full gate check.
	gate := appexec.NewSafetyGate(nil, 0, staleness)
	verdict := gate.Check(event.Strategy.Timestamp, now)
	if !verdict.Allowed {
		t.Fatalf("safety gate must allow fresh derive event, blocked: %s", verdict.Reason)
	}
}

func TestE2E_SafetyGate_RejectsStaleReplayedDeriveEvent(t *testing.T) {
	staleTS := time.Now().UTC().Add(-10 * time.Minute)
	now := time.Now().UTC()

	event, ok := deriveResolvedEvent("triggered", "0.8500", "high", staleTS)
	if !ok {
		t.Fatal("derive resolver should produce event")
	}

	staleness := appexec.NewStalenessGuard(2 * time.Minute)
	gate := appexec.NewSafetyGate(nil, 0, staleness)

	verdict := gate.Check(event.Strategy.Timestamp, now)
	if verdict.Allowed {
		t.Fatal("safety gate must block stale replayed derive event")
	}
	if verdict.Reason != "stale" {
		t.Fatalf("expected reason 'stale', got %q", verdict.Reason)
	}
}

// ── E2E-7: Full pipeline — derive → evaluate → venue submit → fill ─

func TestE2E_FullPipeline_DeriveToVenueFill(t *testing.T) {
	ts := time.Date(2026, 3, 22, 10, 6, 0, 0, time.UTC)

	// Step 1: Derive resolver produces strategy event.
	event, ok := deriveResolvedEvent("triggered", "0.8500", "high", ts)
	if !ok {
		t.Fatal("derive resolver should produce event")
	}

	// Step 2: Execute consumer evaluates.
	engine, collector, pid := spawnTestStrategy(t, "0.02")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: event})
	msg := waitForIntent(t, collector)

	submitEvent := msg.Event
	intent := submitEvent.ExecutionIntent

	// Step 3: Venue adapter submits (paper).
	venue := appexec.NewPaperVenueAdapter(0)
	receipt, prob := venue.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("venue submit failed: %s", prob.Message)
	}

	// Step 4: Construct fill event (mirrors VenueAdapterActor).
	fillEvent := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(submitEvent.Metadata.CorrelationID).
			WithCausationID(submitEvent.Metadata.ID),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	// ── Assertions: complete chain ──

	// Correlation chain: derive → execute → venue.
	if fillEvent.Metadata.CorrelationID != "e2e-corr-s368" {
		t.Fatalf("correlation chain broken: expected e2e-corr-s368, got %s", fillEvent.Metadata.CorrelationID)
	}

	// Causation chain: fill.CausationID = submit event ID.
	if fillEvent.Metadata.CausationID != submitEvent.Metadata.ID {
		t.Fatalf("causation chain broken: expected %s, got %s", submitEvent.Metadata.ID, fillEvent.Metadata.CausationID)
	}

	// Fill event has its own unique ID.
	if fillEvent.Metadata.ID == submitEvent.Metadata.ID {
		t.Fatal("fill event ID must differ from submit event ID")
	}

	// Venue order ID assigned.
	if receipt.VenueOrderID == "" {
		t.Fatal("venue order ID should be assigned")
	}

	// Receipt status.
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected status filled, got %s", receipt.Status)
	}

	// Fill record present and simulated.
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
	}
	if !receipt.Intent.Fills[0].Simulated {
		t.Fatal("paper fill must be simulated")
	}

	// Symbol preserved through entire chain.
	if receipt.Intent.VenueSymbol() != "btcusdt" {
		t.Fatalf("symbol bleed: expected btcusdt, got %s", receipt.Intent.VenueSymbol())
	}

	// Trace preserved through venue.
	if receipt.Intent.CorrelationID != "e2e-corr-s368" {
		t.Fatalf("correlation lost through venue: got %s", receipt.Intent.CorrelationID)
	}
}

// ── E2E-8: Unknown decision outcome — no event, no execution ──────

func TestE2E_UnknownOutcome_NeverReachesExecution(t *testing.T) {
	ts := time.Date(2026, 3, 22, 10, 7, 0, 0, time.UTC)

	// BI-3: Unknown outcomes produce no event.
	_, ok := deriveResolvedEvent("unknown", "0.8500", "high", ts)
	if ok {
		t.Fatal("BI-3: unknown outcome should not produce an event")
	}
}

// ── E2E-9: Deduplication key determinism across the pipeline ──────

func TestE2E_DeduplicationKey_Deterministic(t *testing.T) {
	ts := time.Date(2026, 3, 22, 10, 8, 0, 0, time.UTC)

	event1, ok := deriveResolvedEvent("triggered", "0.8500", "high", ts)
	if !ok {
		t.Fatal("event 1 should be produced")
	}

	event2, ok := deriveResolvedEvent("triggered", "0.8500", "high", ts)
	if !ok {
		t.Fatal("event 2 should be produced")
	}

	// Same inputs → same dedup key (INV-11, BI-1).
	if event1.Strategy.DeduplicationKey() != event2.Strategy.DeduplicationKey() {
		t.Fatalf("INV-11: dedup keys must be identical for same inputs, got %s vs %s",
			event1.Strategy.DeduplicationKey(), event2.Strategy.DeduplicationKey())
	}

	// Different timestamp → different dedup key.
	event3, ok := deriveResolvedEvent("triggered", "0.8500", "high", ts.Add(time.Minute))
	if !ok {
		t.Fatal("event 3 should be produced")
	}
	if event1.Strategy.DeduplicationKey() == event3.Strategy.DeduplicationKey() {
		t.Fatal("INV-11: different timestamps must produce different dedup keys")
	}
}

// ── E2E-10: Derive → Execute with tracker metrics ─────────────────

func TestE2E_TrackerMetrics_CrossScope(t *testing.T) {
	ts := time.Date(2026, 3, 22, 10, 9, 0, 0, time.UTC)

	event, ok := deriveResolvedEvent("triggered", "0.8500", "high", ts)
	if !ok {
		t.Fatal("derive resolver should produce event")
	}

	tracker := healthz.NewTracker("e2e-test")
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := newTestCollector()
	collectorPID := engine.Spawn(func() actor.Receiver { return collector }, "e2e-collector")

	pid := engine.Spawn(NewStrategyConsumerActor(StrategyConsumerConfig{
		MaxPositionPct: "0.01",
		Tracker:        tracker,
		AdapterPID:     collectorPID,
	}), "e2e-strategy-consumer")
	time.Sleep(20 * time.Millisecond)
	defer engine.Poison(pid)

	// Send long + flat events.
	engine.Send(pid, strategyReceivedMessage{Event: event})

	flatEvent, _ := deriveResolvedEvent("not_triggered", "0.9000", "", ts.Add(time.Second))
	engine.Send(pid, strategyReceivedMessage{Event: flatEvent})

	// Wait for both.
	collector.waitForN(t, 2)

	// Verify tracker counters.
	if tracker.Counter("received").Load() != 2 {
		t.Errorf("expected received=2, got %d", tracker.Counter("received").Load())
	}
	if tracker.Counter("evaluated").Load() != 2 {
		t.Errorf("expected evaluated=2, got %d", tracker.Counter("evaluated").Load())
	}
	if tracker.Counter("evaluated_actionable").Load() != 1 {
		t.Errorf("expected evaluated_actionable=1, got %d", tracker.Counter("evaluated_actionable").Load())
	}
	if tracker.Counter("evaluated_flat").Load() != 1 {
		t.Errorf("expected evaluated_flat=1, got %d", tracker.Counter("evaluated_flat").Load())
	}
}

// waitForN waits for n messages on the test collector.
func (c *testIntentCollector) waitForN(t *testing.T, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-c.done:
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for message %d/%d (got %d)", i+1, n, len(c.received))
		}
	}
}
