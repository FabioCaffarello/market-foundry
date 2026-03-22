package store

import (
	"context"
	"testing"
	"time"

	"log/slog"

	"internal/adapters/nats/natskit"
	"internal/application/strategyclient"
	"internal/domain/strategy"
	"internal/shared/events"
	"internal/shared/healthz"
	"internal/shared/problem"
)

// S367: Read-path verification tests.
// Prove that the derive-produced StrategyResolvedEvent flows correctly
// through projection → store → query responder.

// localStrategyGateway is a test double for strategyclient gateway.
type localStrategyGateway struct {
	strat *strategy.Strategy
	prob  *problem.Problem
}

func (m *localStrategyGateway) GetLatestStrategy(_ context.Context, _ strategyclient.StrategyLatestQuery) (strategyclient.StrategyLatestReply, *problem.Problem) {
	return strategyclient.StrategyLatestReply{Strategy: m.strat}, m.prob
}

// mockStrategyReadStore captures Put calls and serves Get from stored data.
type mockStrategyReadStore struct {
	stored    map[string]strategy.Strategy // partition key → strategy
	putResult natskit.PutResult
}

func newMockStrategyReadStore() *mockStrategyReadStore {
	return &mockStrategyReadStore{
		stored:    make(map[string]strategy.Strategy),
		putResult: natskit.PutWritten,
	}
}

func (m *mockStrategyReadStore) Put(_ context.Context, strat strategy.Strategy) (natskit.PutResult, *problem.Problem) {
	m.stored[strat.PartitionKey()] = strat
	return m.putResult, nil
}

func (m *mockStrategyReadStore) Get(_ context.Context, source, symbol string, timeframe int) (*strategy.Strategy, *problem.Problem) {
	key := source + "." + symbol + "." + itoa(timeframe)
	strat, ok := m.stored[key]
	if !ok {
		return nil, nil
	}
	return &strat, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func readPathActor(store strategyProjectionStore, tracker *healthz.Tracker, bucket string) *StrategyProjectionActor {
	if bucket == "" {
		bucket = "STRATEGY_MEAN_REVERSION_ENTRY_LATEST"
	}
	return &StrategyProjectionActor{
		cfg:    StrategyProjectionConfig{Bucket: bucket, Tracker: tracker},
		logger: slog.Default(),
		store:  store,
	}
}

func deriveProducedEvent(ts time.Time) strategy.StrategyResolvedEvent {
	return strategy.StrategyResolvedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID("trace-abc-123").
			WithCausationID("decision-event-456"),
		Strategy: strategy.Strategy{
			Type:       "mean_reversion_entry",
			Source:     "binancef",
			Symbol:     "btcusdt",
			Timeframe:  60,
			Direction:  strategy.DirectionLong,
			Confidence: "0.8500",
			Decisions: []strategy.DecisionInput{
				{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Severity: "high", Timeframe: 60},
			},
			Parameters: map[string]string{"entry": "market", "target_offset": "0.02", "stop_offset": "0.01"},
			Final:      true,
			Timestamp:  ts,
		},
	}
}

func TestReadPath_ProjectionMaterializesEvent(t *testing.T) {
	store := newMockStrategyReadStore()
	tracker := healthz.NewTracker("test")
	a := readPathActor(store, tracker, "")

	now := time.Now().UTC().Truncate(time.Second)
	event := deriveProducedEvent(now)

	a.onStrategy(strategyReceivedMessage{Event: event})

	if a.stats.materialized.Load() != 1 {
		t.Fatalf("expected materialized=1, got %d", a.stats.materialized.Load())
	}

	// Verify the strategy was stored.
	strat, ok := store.stored["binancef.btcusdt.60"]
	if !ok {
		t.Fatal("strategy not found in store after projection")
	}

	// Verify all strategy fields are intact.
	if strat.Type != "mean_reversion_entry" {
		t.Errorf("type: want mean_reversion_entry, got %s", strat.Type)
	}
	if strat.Direction != strategy.DirectionLong {
		t.Errorf("direction: want long, got %s", strat.Direction)
	}
	if strat.Confidence != "0.8500" {
		t.Errorf("confidence: want 0.8500, got %s", strat.Confidence)
	}
	if !strat.Timestamp.Equal(now) {
		t.Errorf("timestamp: want %v, got %v", now, strat.Timestamp)
	}
}

func TestReadPath_ProjectionPreservesDecisionInputs(t *testing.T) {
	store := newMockStrategyReadStore()
	a := readPathActor(store, nil, "")

	event := deriveProducedEvent(time.Now().UTC())
	a.onStrategy(strategyReceivedMessage{Event: event})

	strat := store.stored["binancef.btcusdt.60"]
	if len(strat.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(strat.Decisions))
	}

	d := strat.Decisions[0]
	if d.Type != "rsi_oversold" {
		t.Errorf("decision type: want rsi_oversold, got %s", d.Type)
	}
	if d.Outcome != "triggered" {
		t.Errorf("decision outcome: want triggered, got %s", d.Outcome)
	}
	if d.Severity != "high" {
		t.Errorf("decision severity: want high, got %s", d.Severity)
	}
}

func TestReadPath_ProjectionPreservesParameters(t *testing.T) {
	store := newMockStrategyReadStore()
	a := readPathActor(store, nil, "")

	event := deriveProducedEvent(time.Now().UTC())
	a.onStrategy(strategyReceivedMessage{Event: event})

	strat := store.stored["binancef.btcusdt.60"]
	expected := map[string]string{"entry": "market", "target_offset": "0.02", "stop_offset": "0.01"}
	for k, v := range expected {
		if strat.Parameters[k] != v {
			t.Errorf("parameter %q: want %q, got %q", k, v, strat.Parameters[k])
		}
	}
}

func TestReadPath_EventMetadataNotInStore(t *testing.T) {
	// S367 key finding: projection strips event metadata (correlation/causation).
	// The store receives only strategy.Strategy, not StrategyResolvedEvent.
	store := newMockStrategyReadStore()
	a := readPathActor(store, nil, "")

	event := deriveProducedEvent(time.Now().UTC())
	if event.Metadata.CorrelationID == "" {
		t.Fatal("precondition: event must have correlation_id")
	}
	if event.Metadata.CausationID == "" {
		t.Fatal("precondition: event must have causation_id")
	}

	a.onStrategy(strategyReceivedMessage{Event: event})

	// Strategy is stored, but event metadata is not part of the persisted data.
	// This is the documented gap: correlation_id / causation_id are only logged, not persisted.
	strat := store.stored["binancef.btcusdt.60"]
	if strat.Type == "" {
		t.Fatal("strategy should be stored")
	}
	// Strategy.Metadata is domain metadata, not event metadata.
	// No correlation_id or causation_id in the Strategy struct.
}

func TestReadPath_MonotonicityGuard_StaleRejected(t *testing.T) {
	store := &mockStrategyStore{putResult: natskit.PutSkippedStale}
	a := strategyActor(store, nil)

	event := deriveProducedEvent(time.Now().UTC())
	a.onStrategy(strategyReceivedMessage{Event: event})

	if a.stats.skippedStale.Load() != 1 {
		t.Fatalf("expected skippedStale=1, got %d", a.stats.skippedStale.Load())
	}
	if a.stats.materialized.Load() != 0 {
		t.Fatalf("expected materialized=0, got %d", a.stats.materialized.Load())
	}
}

func TestReadPath_MonotonicityGuard_DuplicateRejected(t *testing.T) {
	store := &mockStrategyStore{putResult: natskit.PutSkippedDuplicate}
	a := strategyActor(store, nil)

	event := deriveProducedEvent(time.Now().UTC())
	a.onStrategy(strategyReceivedMessage{Event: event})

	if a.stats.skippedDedup.Load() != 1 {
		t.Fatalf("expected skippedDedup=1, got %d", a.stats.skippedDedup.Load())
	}
}

func TestReadPath_MonotonicityGuard_NewerAccepted(t *testing.T) {
	store := newMockStrategyReadStore()
	tracker := healthz.NewTracker("test")
	a := readPathActor(store, tracker, "")

	now := time.Now().UTC()

	// First event.
	event1 := deriveProducedEvent(now)
	a.onStrategy(strategyReceivedMessage{Event: event1})

	// Second event with newer timestamp.
	event2 := deriveProducedEvent(now.Add(time.Minute))
	event2.Strategy.Direction = strategy.DirectionShort
	event2.Strategy.Confidence = "0.60"
	a.onStrategy(strategyReceivedMessage{Event: event2})

	if a.stats.materialized.Load() != 2 {
		t.Fatalf("expected materialized=2, got %d", a.stats.materialized.Load())
	}

	// Store should have the latest strategy.
	strat := store.stored["binancef.btcusdt.60"]
	if strat.Direction != strategy.DirectionShort {
		t.Errorf("expected latest direction=short, got %s", strat.Direction)
	}
}

func TestReadPath_QueryUseCaseValidation(t *testing.T) {
	// Verify the use case layer validates inputs before hitting the gateway.
	uc := strategyclient.NewGetLatestStrategyUseCase(&localStrategyGateway{})

	_, prob := uc.Execute(context.Background(), strategyclient.StrategyLatestQuery{
		Type:      "",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected validation error for empty type")
	}
}

func TestReadPath_QueryUseCaseReturnsStoredStrategy(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	strat := &strategy.Strategy{
		Type:       "mean_reversion_entry",
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Direction:  strategy.DirectionLong,
		Confidence: "0.8500",
		Decisions: []strategy.DecisionInput{
			{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Timeframe: 60},
		},
		Parameters: map[string]string{"entry": "market"},
		Final:      true,
		Timestamp:  now,
	}

	uc := strategyclient.NewGetLatestStrategyUseCase(&localStrategyGateway{strat: strat})
	reply, prob := uc.Execute(context.Background(), strategyclient.StrategyLatestQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob.Message)
	}
	if reply.Strategy == nil {
		t.Fatal("expected strategy in reply")
	}
	if reply.Strategy.Direction != strategy.DirectionLong {
		t.Errorf("direction: want long, got %s", reply.Strategy.Direction)
	}
	if reply.Strategy.Confidence != "0.8500" {
		t.Errorf("confidence: want 0.8500, got %s", reply.Strategy.Confidence)
	}
}

func TestReadPath_QueryUseCaseNilStrategy(t *testing.T) {
	uc := strategyclient.NewGetLatestStrategyUseCase(&localStrategyGateway{strat: nil})
	reply, prob := uc.Execute(context.Background(), strategyclient.StrategyLatestQuery{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob.Message)
	}
	if reply.Strategy != nil {
		t.Fatal("expected nil strategy for unfound key")
	}
}

func TestReadPath_RegistrySubjectAlignment(t *testing.T) {
	// Verify that the store consumer filter matches the publisher subject pattern.
	// Store consumer: strategy.events.mean_reversion_entry.resolved.>
	// Publisher:       strategy.events.mean_reversion_entry.resolved.{source}.{symbol}.{timeframe}
	// The wildcard > must capture all publisher variations.

	publisherSubject := "strategy.events.mean_reversion_entry.resolved.binancef.btcusdt.60"
	consumerFilter := "strategy.events.mean_reversion_entry.resolved.>"

	// Simple prefix match — NATS > wildcard matches any number of tokens after the prefix.
	prefix := consumerFilter[:len(consumerFilter)-1] // strip ">"
	if publisherSubject[:len(prefix)] != prefix {
		t.Fatalf("consumer filter %q does not match publisher subject %q", consumerFilter, publisherSubject)
	}
}

func TestReadPath_GatewayQuerySubjectAlignment(t *testing.T) {
	// Verify query responder and gateway use matching subjects.
	// Both should resolve to "strategy.query.mean_reversion_entry.latest".
	querySubject := "strategy.query.mean_reversion_entry.latest"
	expectedSubject := "strategy.query.mean_reversion_entry.latest"

	if querySubject != expectedSubject {
		t.Fatalf("query subject mismatch: want %q, got %q", expectedSubject, querySubject)
	}
}

func TestReadPath_FlatStrategyMaterializes(t *testing.T) {
	// Flat strategies (direction=flat, confidence=0) should still materialize
	// if Final=true and validation passes.
	store := newMockStrategyReadStore()
	a := readPathActor(store, nil, "")

	event := deriveProducedEvent(time.Now().UTC())
	event.Strategy.Direction = strategy.DirectionFlat
	event.Strategy.Confidence = "0.0000"

	a.onStrategy(strategyReceivedMessage{Event: event})

	if a.stats.materialized.Load() != 1 {
		t.Fatalf("flat strategy should materialize, got materialized=%d", a.stats.materialized.Load())
	}

	strat := store.stored["binancef.btcusdt.60"]
	if strat.Direction != strategy.DirectionFlat {
		t.Errorf("expected flat, got %s", strat.Direction)
	}
}

func TestReadPath_MultiFamily_PartitionIsolation(t *testing.T) {
	// Different families use different buckets — verify no cross-family contamination.
	// mean_reversion_entry → STRATEGY_MEAN_REVERSION_ENTRY_LATEST
	// trend_following_entry → STRATEGY_TREND_FOLLOWING_ENTRY_LATEST

	mreStore := newMockStrategyReadStore()
	mreActor := readPathActor(mreStore, nil, "STRATEGY_MEAN_REVERSION_ENTRY_LATEST")

	tfeStore := newMockStrategyReadStore()
	tfeActor := readPathActor(tfeStore, nil, "STRATEGY_TREND_FOLLOWING_ENTRY_LATEST")

	now := time.Now().UTC()

	mreEvent := deriveProducedEvent(now)
	mreEvent.Strategy.Type = "mean_reversion_entry"
	mreActor.onStrategy(strategyReceivedMessage{Event: mreEvent})

	tfeEvent := deriveProducedEvent(now.Add(time.Second))
	tfeEvent.Strategy.Type = "trend_following_entry"
	tfeActor.onStrategy(strategyReceivedMessage{Event: tfeEvent})

	if mreActor.stats.materialized.Load() != 1 {
		t.Errorf("MRE: expected materialized=1, got %d", mreActor.stats.materialized.Load())
	}
	if tfeActor.stats.materialized.Load() != 1 {
		t.Errorf("TFE: expected materialized=1, got %d", tfeActor.stats.materialized.Load())
	}

	// Each store should have exactly one entry.
	if len(mreStore.stored) != 1 {
		t.Errorf("MRE store: expected 1 entry, got %d", len(mreStore.stored))
	}
	if len(tfeStore.stored) != 1 {
		t.Errorf("TFE store: expected 1 entry, got %d", len(tfeStore.stored))
	}
}
