package execute

import (
	"testing"
	"time"

	appexec "internal/application/execution"
	domainexec "internal/domain/execution"
	"internal/domain/strategy"
	"internal/shared/events"
)

// execute_venue_adapter_canary_test.go (H-6.c.2 commit 6)
//
// Locks the 37f8ddd regression contract at the execute-scope
// boundary: when a Strategy event with the synthetic Source
// "execute.venue-adapter" (produced in production by
// execute_supervisor.go:148 as the VenueAdapterActor's NATS
// publisher identity) flows through the PaperOrderEvaluator
// pipeline, the resulting ExecutionIntent MUST preserve the
// canonical Instrument carried by the Strategy — NOT silent-
// zero from source-string reconstruction.
//
// The H-6.b' commit 37f8ddd was the original regression: the
// legacy NewPaperOrderEvaluator(source, symbol, timeframe)
// delegated to instrumentFromBinding(source, symbol), which
// returned a zero CanonicalInstrument for any source outside
// the {"binances", "binancef"} hardcoded mapping. With
// "execute.venue-adapter" as the source, every ExecutionIntent
// produced by the evaluator had a zero Instrument, leaking
// downstream until 6 integration tests caught it in CI.
//
// H-6.b' commit 37f8ddd added NewPaperOrderEvaluatorForInstrument
// as the fix; H-6.c.2 commit 5 deleted the legacy ctor + helper
// entirely. This canary asserts that the structural prevention
// holds: even if a future change accidentally reintroduces
// source-string reconstruction, the canary would surface as
// a zero-Instrument assertion failure here.
//
// Coverage scope (pre-flight 5 + 7):
// - Test 1: unit-shape. Direct evaluator construction → Evaluate
//   → assert intent.Instrument == input AND !intent.Instrument.IsZero().
// - Test 2: actor-shape. Spawn strategy_consumer_actor →
//   send strategyReceivedMessage with synthetic Source → wait
//   for intentReceivedMessage → assert intent.Instrument
//   matches. Mirrors H-6.c.1 commit 8 derive canary
//   architectural shape (stand-in for the production wiring).

// TestPaperOrderEvaluator_PreservesInstrument_WithSyntheticSource
// is the unit-shape canary: a direct evaluator construction with
// the production synthetic source "execute.venue-adapter" must
// preserve the canonical Instrument unchanged in the resulting
// ExecutionIntent. Pre-37f8ddd, the legacy ctor would have
// silent-zero'd here (Source not in {binances, binancef}); post
// H-6.c.2 commit 5, this is structurally impossible because the
// legacy ctor is deleted — but the canary remains to prevent
// regression if a future change reintroduces source-string
// reconstruction in any form.
func TestPaperOrderEvaluator_PreservesInstrument_WithSyntheticSource(t *testing.T) {
	inst := btcUSDTPerpExec(t)

	evaluator := appexec.NewPaperOrderEvaluatorForInstrument(
		"execute.venue-adapter", // synthetic source — the 37f8ddd trigger
		inst,
		60,
	)

	ts := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	intent, ok := evaluator.Evaluate(
		"pass_through", "approved", "0.8500", "0.0100",
		"long", "0.8500",
		"mean_reversion_entry", "high",
		60, ts,
	)
	if !ok {
		t.Fatal("Evaluate should succeed with valid input")
	}

	if intent.Instrument.IsZero() {
		t.Fatalf("intent.Instrument is zero — 37f8ddd regression. Source=%q, evaluator constructed via ForInstrument should pass-through inst", "execute.venue-adapter")
	}
	if intent.Instrument != inst {
		t.Errorf("intent.Instrument = %+v, want %+v (pass-through broken)", intent.Instrument, inst)
	}
	if intent.Source != "execute.venue-adapter" {
		t.Errorf("intent.Source = %q, want \"execute.venue-adapter\"", intent.Source)
	}
}

// TestStrategyConsumerActor_PreservesInstrument_WithSyntheticSource
// is the actor-shape canary: drives a Strategy event with the
// synthetic Source through the actual strategy_consumer_actor
// (which calls NewPaperOrderEvaluatorForInstrument at line 138
// in production), captures the published intentReceivedMessage,
// and asserts the resulting Intent.Instrument matches the input.
// This locks the wiring contract at the same layer where 37f8ddd
// originally failed.
func TestStrategyConsumerActor_PreservesInstrument_WithSyntheticSource(t *testing.T) {
	engine, collector, pid := spawnTestStrategy(t, "0.01")
	defer engine.Poison(pid)

	inst := btcUSDTPerpExec(t)
	event := strategy.StrategyResolvedEvent{
		Metadata: events.Metadata{
			ID:            "evt-canary-37f8ddd",
			OccurredAt:    time.Now().UTC(),
			CorrelationID: "corr-canary",
			CausationID:   "cause-canary",
		},
		Strategy: strategy.Strategy{
			Type:       "mean_reversion_entry",
			Source:     "execute.venue-adapter", // synthetic — the 37f8ddd trigger
			Instrument: inst,
			Timeframe:  60,
			Direction:  strategy.DirectionLong,
			Confidence: "0.8500",
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

	engine.Send(pid, strategyReceivedMessage{Event: event})
	msg := waitForIntent(t, collector)

	intent := msg.Event.ExecutionIntent
	if intent.Instrument.IsZero() {
		t.Fatalf("intent.Instrument is zero — 37f8ddd regression. Strategy.Source=%q must NOT silent-zero through the consumer", event.Strategy.Source)
	}
	if intent.Instrument != inst {
		t.Errorf("intent.Instrument = %+v, want %+v (consumer broke pass-through)", intent.Instrument, inst)
	}
	if intent.Source != "execute.venue-adapter" {
		t.Errorf("intent.Source = %q, want \"execute.venue-adapter\"", intent.Source)
	}
	if intent.Side != domainexec.SideBuy {
		t.Errorf("intent.Side = %s, want buy", intent.Side)
	}
}
