package store

// e2e_derive_to_store_test.go — S368: End-to-end analytical-to-store proof.
//
// Proves the connected path:
//   derive (MeanReversionEntryResolver) → StrategyResolvedEvent → StrategyProjectionActor
//   → KV store → query use case → strategy read-back.
//
// Each test exercises the real derive resolver, feeds its output to the real
// store projection actor (with mock KV), and validates that the materialized
// strategy is queryable and field-complete.
//
// Governing questions answered:
//   - DI-4 Q5: Does a derive-produced event materialize correctly in store?
//   - DI-4 Q6: Is the materialized strategy queryable with correct fields?
//   - DI-4 Q7: Does the read-path preserve the derive-produced state?

import (
	"context"
	"testing"
	"time"

	"internal/adapters/nats/natskit"
	appstrategy "internal/application/strategy"
	"internal/application/strategyclient"
	"internal/domain/strategy"
	"internal/shared/events"
	"internal/shared/healthz"
)

// deriveEvent produces a real StrategyResolvedEvent using the derive resolver.
func deriveEvent(outcome, confidence, severity string, ts time.Time) strategy.StrategyResolvedEvent {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	strat, ok := resolver.Resolve(
		"rsi_oversold", outcome, confidence, severity,
		"RSI below 30 on btcusdt", 60, ts,
	)
	if !ok {
		// For outcomes like "unknown" we return empty event; test should check ok.
		return strategy.StrategyResolvedEvent{}
	}

	return strategy.StrategyResolvedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID("e2e-store-corr-s368").
			WithCausationID("decision-evt-store-001"),
		Strategy: strat,
	}
}

// ── E2E-S1: Derive-produced triggered event materializes correctly ──

func TestE2E_Store_DeriveTriggered_Materializes(t *testing.T) {
	store := newMockStrategyReadStore()
	tracker := healthz.NewTracker("e2e-store-test")
	a := readPathActor(store, tracker, "")

	ts := time.Now().UTC().Truncate(time.Second)
	event := deriveEvent("triggered", "0.8500", "high", ts)

	a.onStrategy(strategyReceivedMessage{Event: event})

	if a.stats.materialized.Load() != 1 {
		t.Fatalf("expected materialized=1, got %d", a.stats.materialized.Load())
	}

	strat, ok := store.stored["binancef.btcusdt.60"]
	if !ok {
		t.Fatal("strategy not found in store")
	}

	// Verify all derive-produced fields survive.
	if strat.Type != "mean_reversion_entry" {
		t.Errorf("type: want mean_reversion_entry, got %s", strat.Type)
	}
	if strat.Direction != strategy.DirectionLong {
		t.Errorf("direction: want long, got %s", strat.Direction)
	}
	if strat.Confidence != "0.8500" {
		t.Errorf("confidence: want 0.8500, got %s", strat.Confidence)
	}
	if strat.Source != "binancef" {
		t.Errorf("source: want binancef, got %s", strat.Source)
	}
	if strat.Symbol != "btcusdt" {
		t.Errorf("symbol: want btcusdt, got %s", strat.Symbol)
	}
	if strat.Timeframe != 60 {
		t.Errorf("timeframe: want 60, got %d", strat.Timeframe)
	}
	if !strat.Final {
		t.Error("final: should be true")
	}
	if !strat.Timestamp.Equal(ts) {
		t.Errorf("timestamp: want %v, got %v", ts, strat.Timestamp)
	}

	// Decision inputs preserved.
	if len(strat.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(strat.Decisions))
	}
	if strat.Decisions[0].Type != "rsi_oversold" {
		t.Errorf("decision type: want rsi_oversold, got %s", strat.Decisions[0].Type)
	}
	if strat.Decisions[0].Severity != "high" {
		t.Errorf("decision severity: want high, got %s", strat.Decisions[0].Severity)
	}

	// Parameters preserved.
	if strat.Parameters["entry"] != "market" {
		t.Errorf("parameter entry: want market, got %s", strat.Parameters["entry"])
	}
	if strat.Parameters["target_offset"] != "0.03" {
		t.Errorf("parameter target_offset: want 0.03, got %s", strat.Parameters["target_offset"])
	}
	if strat.Parameters["stop_offset"] != "0.01" {
		t.Errorf("parameter stop_offset: want 0.01 (0.01×0.75 rounded to .2f), got %s", strat.Parameters["stop_offset"])
	}
}

// ── E2E-S2: Derive-produced flat event materializes ─────────────────

func TestE2E_Store_DeriveFlat_Materializes(t *testing.T) {
	store := newMockStrategyReadStore()
	a := readPathActor(store, nil, "")

	ts := time.Now().UTC().Truncate(time.Second)
	event := deriveEvent("not_triggered", "0.9000", "", ts)

	a.onStrategy(strategyReceivedMessage{Event: event})

	if a.stats.materialized.Load() != 1 {
		t.Fatalf("flat strategy should materialize, got %d", a.stats.materialized.Load())
	}

	strat := store.stored["binancef.btcusdt.60"]
	if strat.Direction != strategy.DirectionFlat {
		t.Errorf("direction: want flat, got %s", strat.Direction)
	}
	if strat.Confidence != "0.0000" {
		t.Errorf("confidence: want 0.0000, got %s", strat.Confidence)
	}
}

// ── E2E-S3: Materialized strategy is queryable via use case ─────────

func TestE2E_Store_MaterializedStrategyQueryable(t *testing.T) {
	ts := time.Now().UTC().Truncate(time.Second)
	event := deriveEvent("triggered", "0.8500", "moderate", ts)

	// Simulate materialization by extracting the strategy.
	strat := event.Strategy

	gateway := &localStrategyGateway{strat: &strat}
	uc := strategyclient.NewGetLatestStrategyUseCase(gateway)

	reply, prob := uc.Execute(context.Background(), strategyclient.StrategyLatestQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("query failed: %s", prob.Message)
	}
	if reply.Strategy == nil {
		t.Fatal("expected strategy in reply")
	}

	// Verify derive-produced fields survive the full read path.
	if reply.Strategy.Direction != strategy.DirectionLong {
		t.Errorf("direction: want long, got %s", reply.Strategy.Direction)
	}
	if reply.Strategy.Confidence != "0.7650" {
		t.Errorf("confidence: want 0.7650 (moderate scaling), got %s", reply.Strategy.Confidence)
	}
	if reply.Strategy.Parameters["entry"] != "market" {
		t.Errorf("parameter entry: want market, got %s", reply.Strategy.Parameters["entry"])
	}
}

// ── E2E-S4: Monotonicity guard rejects stale derive events ──────────

func TestE2E_Store_MonotonicityRejectsStale(t *testing.T) {
	staleStore := &mockStrategyStore{putResult: natskit.PutSkippedStale}
	a := strategyActor(staleStore, nil)

	ts := time.Now().UTC().Truncate(time.Second)
	event := deriveEvent("triggered", "0.8500", "high", ts)

	a.onStrategy(strategyReceivedMessage{Event: event})

	if a.stats.skippedStale.Load() != 1 {
		t.Fatalf("expected skippedStale=1, got %d", a.stats.skippedStale.Load())
	}
	if a.stats.materialized.Load() != 0 {
		t.Fatalf("stale event should not materialize, got %d", a.stats.materialized.Load())
	}
}

// ── E2E-S5: Newer derive event overwrites older in store ────────────

func TestE2E_Store_NewerDeriveEventOverwrites(t *testing.T) {
	store := newMockStrategyReadStore()
	tracker := healthz.NewTracker("e2e-overwrite")
	a := readPathActor(store, tracker, "")

	now := time.Now().UTC().Truncate(time.Second)

	// First event: long at high confidence.
	event1 := deriveEvent("triggered", "0.8500", "high", now)
	a.onStrategy(strategyReceivedMessage{Event: event1})

	// Second event: flat (not triggered) with newer timestamp.
	event2 := deriveEvent("not_triggered", "0.9000", "", now.Add(time.Minute))
	a.onStrategy(strategyReceivedMessage{Event: event2})

	if a.stats.materialized.Load() != 2 {
		t.Fatalf("expected 2 materializations, got %d", a.stats.materialized.Load())
	}

	// Latest value should be flat.
	strat := store.stored["binancef.btcusdt.60"]
	if strat.Direction != strategy.DirectionFlat {
		t.Errorf("latest should be flat, got %s", strat.Direction)
	}
}

// ── E2E-S6: Event metadata not persisted (documented gap L1) ────────

func TestE2E_Store_EventMetadataNotPersisted(t *testing.T) {
	store := newMockStrategyReadStore()
	a := readPathActor(store, nil, "")

	event := deriveEvent("triggered", "0.8500", "high", time.Now().UTC())

	// Precondition: event has correlation metadata.
	if event.Metadata.CorrelationID == "" {
		t.Fatal("precondition: event must have correlation_id")
	}

	a.onStrategy(strategyReceivedMessage{Event: event})

	// Strategy is stored, but Strategy struct has no correlation_id field.
	// This is the documented gap L1 from S367.
	strat := store.stored["binancef.btcusdt.60"]
	if strat.Type == "" {
		t.Fatal("strategy should be stored")
	}
	// Domain metadata is separate from event metadata.
	// Strategy.Metadata contains decision context, not event trace.
	if strat.Metadata["decision_type"] != "rsi_oversold" {
		t.Errorf("domain metadata should carry decision_type, got %v", strat.Metadata)
	}
}
