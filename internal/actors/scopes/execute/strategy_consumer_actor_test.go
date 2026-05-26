package execute

import (
	"testing"
	"time"

	domainexec "internal/domain/execution"
	"internal/domain/strategy"
	"internal/shared/events"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// testIntentCollector is a minimal actor that collects intentReceivedMessage for assertion.
type testIntentCollector struct {
	received []intentReceivedMessage
	done     chan struct{}
}

func (c *testIntentCollector) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case intentReceivedMessage:
		c.received = append(c.received, msg)
		if c.done != nil {
			select {
			case c.done <- struct{}{}:
			default:
			}
		}
	}
}

func newTestCollector() *testIntentCollector {
	return &testIntentCollector{done: make(chan struct{}, 10)}
}

func spawnTestStrategy(t *testing.T, maxPositionPct string) (*actor.Engine, *testIntentCollector, *actor.PID) {
	t.Helper()
	return spawnTestStrategyWithConfig(t, StrategyConsumerConfig{MaxPositionPct: maxPositionPct})
}

func spawnTestStrategyWithConfig(t *testing.T, cfg StrategyConsumerConfig) (*actor.Engine, *testIntentCollector, *actor.PID) {
	t.Helper()
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatal(err)
	}

	collector := newTestCollector()
	collectorPID := engine.Spawn(func() actor.Receiver { return collector }, "collector")

	tracker := healthz.NewTracker("test-strategy-consumer")
	cfg.Tracker = tracker
	cfg.AdapterPID = collectorPID
	pid := engine.Spawn(NewStrategyConsumerActor(cfg), "strategy-consumer")

	// Let actors start.
	time.Sleep(20 * time.Millisecond)
	return engine, collector, pid
}

func makeStrategyEvent(t *testing.T, direction strategy.Direction, confidence string, strategyType string) strategy.StrategyResolvedEvent {
	t.Helper()
	return strategy.StrategyResolvedEvent{
		Metadata: events.Metadata{
			ID:            "evt-001",
			OccurredAt:    time.Now().UTC(),
			CorrelationID: "corr-001",
			CausationID:   "cause-001",
		},
		Strategy: strategy.Strategy{
			Type:       strategyType,
			Source:     "test-source",
			Instrument: btcUSDTPerpExec(t),
			Timeframe:  60,
			Direction:  direction,
			Confidence: confidence,
			Decisions: []strategy.DecisionInput{
				{
					Type:       "rsi_oversold",
					Outcome:    "triggered",
					Confidence: "0.8500",
					Severity:   "high",
					Rationale:  "RSI below 30",
					Timeframe:  60,
				},
			},
			Parameters: map[string]string{"entry": "market"},
			Metadata:   map[string]string{"decision_type": "rsi_oversold"},
			Final:      true,
			Timestamp:  time.Now().UTC(),
		},
	}
}

func waitForIntent(t *testing.T, collector *testIntentCollector) intentReceivedMessage {
	t.Helper()
	select {
	case <-collector.done:
		if len(collector.received) == 0 {
			t.Fatal("done signaled but no messages received")
		}
		return collector.received[len(collector.received)-1]
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for intent")
		return intentReceivedMessage{}
	}
}

// ── INV-2: Direction-to-side mapping is deterministic ─────────────

func TestStrategyConsumer_LongDirection_ProducesBuySide(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionLong, "0.8500", "mean_reversion_entry")})
	msg := waitForIntent(t, collector)

	intent := msg.Event.ExecutionIntent
	if intent.Side != domainexec.SideBuy {
		t.Errorf("expected side=buy, got %s", intent.Side)
	}
	if intent.Quantity != "0.01" {
		t.Errorf("expected quantity=0.01, got %s", intent.Quantity)
	}
}

func TestStrategyConsumer_ShortDirection_ProducesSellSide(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionShort, "0.7000", "mean_reversion_entry")})
	msg := waitForIntent(t, collector)

	intent := msg.Event.ExecutionIntent
	if intent.Side != domainexec.SideSell {
		t.Errorf("expected side=sell, got %s", intent.Side)
	}
	if intent.Quantity != "0.01" {
		t.Errorf("expected quantity=0.01, got %s", intent.Quantity)
	}
}

// ── INV-7: Flat direction produces side=none, quantity=0 ──────────

func TestStrategyConsumer_FlatDirection_ProducesNoExecution(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionFlat, "0.0000", "mean_reversion_entry")})
	msg := waitForIntent(t, collector)

	intent := msg.Event.ExecutionIntent
	if intent.Side != domainexec.SideNone {
		t.Errorf("expected side=none, got %s", intent.Side)
	}
	if intent.Quantity != "0" {
		t.Errorf("expected quantity=0, got %s", intent.Quantity)
	}
}

// ── INV-4: Pass-through risk is explicit ──────────────────────────

func TestStrategyConsumer_PassThroughRisk(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "0.02")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionLong, "0.9000", "mean_reversion_entry")})
	msg := waitForIntent(t, collector)

	risk := msg.Event.ExecutionIntent.Risk
	if risk.Type != "pass_through" {
		t.Errorf("expected risk.type=pass_through, got %s", risk.Type)
	}
	if risk.Disposition != "approved" {
		t.Errorf("expected risk.disposition=approved, got %s", risk.Disposition)
	}
}

// ── INV-3: Correlation/causation chain preserved ──────────────────

func TestStrategyConsumer_CorrelationCausationChain(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	event := makeStrategyEvent(t, strategy.DirectionLong, "0.8500", "mean_reversion_entry")
	event.Metadata.CorrelationID = "corr-test-chain"
	event.Metadata.ID = "evt-cause-id"

	engine.Send(pid, strategyReceivedMessage{Event: event})
	msg := waitForIntent(t, collector)

	intent := msg.Event.ExecutionIntent
	if intent.CorrelationID != "corr-test-chain" {
		t.Errorf("expected correlation_id=corr-test-chain, got %s", intent.CorrelationID)
	}
	if intent.CausationID != "evt-cause-id" {
		t.Errorf("expected causation_id=evt-cause-id, got %s", intent.CausationID)
	}

	// Event-level metadata should also carry the chain.
	if msg.Event.Metadata.CorrelationID != "corr-test-chain" {
		t.Errorf("expected event metadata correlation_id=corr-test-chain, got %s", msg.Event.Metadata.CorrelationID)
	}
	if msg.Event.Metadata.CausationID != "evt-cause-id" {
		t.Errorf("expected event metadata causation_id=evt-cause-id, got %s", msg.Event.Metadata.CausationID)
	}
}

// ── INV-5: Timestamp from strategy event, not time.Now() ─────────

func TestStrategyConsumer_UsesStrategyTimestamp(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	fixedTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	event := makeStrategyEvent(t, strategy.DirectionLong, "0.8500", "mean_reversion_entry")
	event.Strategy.Timestamp = fixedTime

	engine.Send(pid, strategyReceivedMessage{Event: event})
	msg := waitForIntent(t, collector)

	if !msg.Event.ExecutionIntent.Timestamp.Equal(fixedTime) {
		t.Errorf("expected intent timestamp=%v, got %v", fixedTime, msg.Event.ExecutionIntent.Timestamp)
	}
}

// ── INV-1: Strategy type identity preserved ───────────────────────

func TestStrategyConsumer_StrategyTypeIdentity(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionLong, "0.8500", "mean_reversion_entry")})
	msg := waitForIntent(t, collector)

	if msg.Event.ExecutionIntent.Risk.StrategyType != "mean_reversion_entry" {
		t.Errorf("expected risk.strategy_type=mean_reversion_entry, got %s", msg.Event.ExecutionIntent.Risk.StrategyType)
	}
	if msg.Event.ExecutionIntent.Parameters["strategy_type"] != "mean_reversion_entry" {
		t.Errorf("expected parameters.strategy_type=mean_reversion_entry, got %s", msg.Event.ExecutionIntent.Parameters["strategy_type"])
	}
}

// ── INV-6: Wrong strategy type is skipped ─────────────────────────

func TestStrategyConsumer_WrongType_Skipped(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionLong, "0.8500", "trend_following_entry")})

	// Give time for processing.
	time.Sleep(100 * time.Millisecond)

	if len(collector.received) != 0 {
		t.Errorf("expected no intent for wrong strategy type, got %d", len(collector.received))
	}
}

// ── Configurable max position pct ────────────────────────────────

func TestStrategyConsumer_ConfigurableMaxPositionPct(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "0.05")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionLong, "0.8500", "mean_reversion_entry")})
	msg := waitForIntent(t, collector)

	if msg.Event.ExecutionIntent.Quantity != "0.05" {
		t.Errorf("expected quantity=0.05, got %s", msg.Event.ExecutionIntent.Quantity)
	}
}

// ── Default max position pct ──────────────────────────────────────

func TestStrategyConsumer_DefaultMaxPositionPct(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionLong, "0.8500", "mean_reversion_entry")})
	msg := waitForIntent(t, collector)

	if msg.Event.ExecutionIntent.Quantity != DefaultMaxPositionPct {
		t.Errorf("expected quantity=%s, got %s", DefaultMaxPositionPct, msg.Event.ExecutionIntent.Quantity)
	}
}

// ── Decision severity carried forward ─────────────────────────────

func TestStrategyConsumer_DecisionSeverityPreserved(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	event := makeStrategyEvent(t, strategy.DirectionLong, "0.8500", "mean_reversion_entry")
	event.Strategy.Decisions[0].Severity = "moderate"

	engine.Send(pid, strategyReceivedMessage{Event: event})
	msg := waitForIntent(t, collector)

	if msg.Event.ExecutionIntent.Risk.DecisionSeverity != "moderate" {
		t.Errorf("expected decision_severity=moderate, got %s", msg.Event.ExecutionIntent.Risk.DecisionSeverity)
	}
	if msg.Event.ExecutionIntent.Parameters["decision_severity"] != "moderate" {
		t.Errorf("expected parameters.decision_severity=moderate, got %s", msg.Event.ExecutionIntent.Parameters["decision_severity"])
	}
}

// ── S361: Confidence threshold ────────────────────────────────────

func TestStrategyConsumer_ConfidenceThreshold_AboveThreshold_Evaluated(t *testing.T) {
	engine, collector, pid := spawnTestStrategyWithConfig(t, StrategyConsumerConfig{
		MaxPositionPct: "0.01",
		MinConfidence:  "0.50",
	})
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionLong, "0.8500", "mean_reversion_entry")})
	msg := waitForIntent(t, collector)

	if msg.Event.ExecutionIntent.Side != domainexec.SideBuy {
		t.Errorf("expected side=buy, got %s", msg.Event.ExecutionIntent.Side)
	}
}

func TestStrategyConsumer_ConfidenceThreshold_BelowThreshold_Skipped(t *testing.T) {
	engine, collector, pid := spawnTestStrategyWithConfig(t, StrategyConsumerConfig{
		MaxPositionPct: "0.01",
		MinConfidence:  "0.90",
	})
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionLong, "0.5000", "mean_reversion_entry")})

	time.Sleep(100 * time.Millisecond)

	if len(collector.received) != 0 {
		t.Errorf("expected no intent for low confidence, got %d", len(collector.received))
	}
}

func TestStrategyConsumer_ConfidenceThreshold_EmptyString_DisablesFilter(t *testing.T) {
	engine, collector, pid := spawnTestStrategyWithConfig(t, StrategyConsumerConfig{
		MaxPositionPct: "0.01",
		MinConfidence:  "",
	})
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionLong, "0.0100", "mean_reversion_entry")})
	msg := waitForIntent(t, collector)

	if msg.Event.ExecutionIntent.Side != domainexec.SideBuy {
		t.Errorf("expected side=buy with empty threshold, got %s", msg.Event.ExecutionIntent.Side)
	}
}

func TestStrategyConsumer_ConfidenceThreshold_EqualToThreshold_Evaluated(t *testing.T) {
	engine, collector, pid := spawnTestStrategyWithConfig(t, StrategyConsumerConfig{
		MaxPositionPct: "0.01",
		MinConfidence:  "0.50",
	})
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionLong, "0.5000", "mean_reversion_entry")})
	msg := waitForIntent(t, collector)

	if msg.Event.ExecutionIntent.Side != domainexec.SideBuy {
		t.Errorf("expected side=buy at exact threshold, got %s", msg.Event.ExecutionIntent.Side)
	}
}

// ── S361: Source path explainability fields ────────────────────────

func TestStrategyConsumer_ExplainabilityFields_Present(t *testing.T) {
	engine, collector, pid := spawnTestStrategyWithConfig(t, StrategyConsumerConfig{
		MaxPositionPct: "0.01",
		MinConfidence:  "0.30",
	})
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionLong, "0.8500", "mean_reversion_entry")})
	msg := waitForIntent(t, collector)

	params := msg.Event.ExecutionIntent.Parameters
	if params["source_path"] != "strategy_consumer.mean_reversion_entry" {
		t.Errorf("expected source_path=strategy_consumer.mean_reversion_entry, got %s", params["source_path"])
	}
	if params["evaluation_outcome"] != "actionable" {
		t.Errorf("expected evaluation_outcome=actionable, got %s", params["evaluation_outcome"])
	}
	if params["confidence_threshold"] != "0.30" {
		t.Errorf("expected confidence_threshold=0.30, got %s", params["confidence_threshold"])
	}
}

func TestStrategyConsumer_ExplainabilityFields_FlatOutcome(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	engine.Send(pid, strategyReceivedMessage{Event: makeStrategyEvent(t, strategy.DirectionFlat, "0.0000", "mean_reversion_entry")})
	msg := waitForIntent(t, collector)

	if msg.Event.ExecutionIntent.Parameters["evaluation_outcome"] != "flat" {
		t.Errorf("expected evaluation_outcome=flat, got %s", msg.Event.ExecutionIntent.Parameters["evaluation_outcome"])
	}
}
