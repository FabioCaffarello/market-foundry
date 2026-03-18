package store

import (
	"context"
	"log/slog"
	"testing"
	"time"

	adapternats "internal/adapters/nats"
	"internal/domain/strategy"
	"internal/shared/healthz"
	"internal/shared/problem"
)

type mockStrategyStore struct {
	putResult  adapternats.PutResult
	putProblem *problem.Problem
	putCalls   int
}

func (m *mockStrategyStore) Put(_ context.Context, _ strategy.Strategy) (adapternats.PutResult, *problem.Problem) {
	m.putCalls++
	return m.putResult, m.putProblem
}

func validStrategy(ts time.Time) strategy.Strategy {
	return strategy.Strategy{
		Type:       "mean_reversion_entry",
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Direction:  strategy.DirectionLong,
		Confidence: "0.85",
		Decisions: []strategy.DecisionInput{
			{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Timeframe: 60},
		},
		Parameters: map[string]string{"entry": "market", "target_offset": "0.02", "stop_offset": "0.01"},
		Final:      true,
		Timestamp:  ts,
	}
}

func strategyActor(store *mockStrategyStore, tracker *healthz.Tracker) *StrategyProjectionActor {
	return &StrategyProjectionActor{
		cfg:    StrategyProjectionConfig{Bucket: "STRATEGY_MEAN_REVERSION_ENTRY_LATEST", Tracker: tracker},
		logger: slog.Default(),
		store:  store,
	}
}

func TestStrategyProjection_FinalGate_SkipsNonFinal(t *testing.T) {
	store := &mockStrategyStore{putResult: adapternats.PutWritten}
	a := strategyActor(store, nil)

	strat := validStrategy(time.Now())
	strat.Final = false

	a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{Strategy: strat}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls for non-final strategy, got %d", store.putCalls)
	}
	if got := a.stats.skippedNonFinal.Load(); got != 1 {
		t.Fatalf("expected skippedNonFinal=1, got %d", got)
	}
	if got := a.stats.received.Load(); got != 1 {
		t.Fatalf("expected received=1, got %d", got)
	}
}

func TestStrategyProjection_ValidationGate_RejectsMalformed(t *testing.T) {
	store := &mockStrategyStore{putResult: adapternats.PutWritten}
	a := strategyActor(store, nil)

	strat := strategy.Strategy{Final: true} // missing required fields

	a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{Strategy: strat}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls, got %d", store.putCalls)
	}
	if got := a.stats.rejected.Load(); got != 1 {
		t.Fatalf("expected rejected=1, got %d", got)
	}
}

func TestStrategyProjection_ValidationGate_RejectsInvalidDirection(t *testing.T) {
	store := &mockStrategyStore{putResult: adapternats.PutWritten}
	a := strategyActor(store, nil)

	strat := validStrategy(time.Now())
	strat.Direction = "sideways" // invalid

	a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{Strategy: strat}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls for invalid direction, got %d", store.putCalls)
	}
	if got := a.stats.rejected.Load(); got != 1 {
		t.Fatalf("expected rejected=1, got %d", got)
	}
}

func TestStrategyProjection_PutWritten_Materializes(t *testing.T) {
	store := &mockStrategyStore{putResult: adapternats.PutWritten}
	tracker := healthz.NewTracker("test")
	a := strategyActor(store, tracker)

	a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{Strategy: validStrategy(time.Now())}})

	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
	if got := a.stats.received.Load(); got != 1 {
		t.Fatalf("expected received=1, got %d", got)
	}
	if tracker.EventCount() != 1 {
		t.Fatalf("expected tracker count=1, got %d", tracker.EventCount())
	}
}

func TestStrategyProjection_PutSkippedStale(t *testing.T) {
	store := &mockStrategyStore{putResult: adapternats.PutSkippedStale}
	a := strategyActor(store, nil)

	a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{Strategy: validStrategy(time.Now())}})

	if got := a.stats.skippedStale.Load(); got != 1 {
		t.Fatalf("expected skippedStale=1, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 0 {
		t.Fatalf("expected materialized=0, got %d", got)
	}
}

func TestStrategyProjection_PutSkippedDuplicate(t *testing.T) {
	store := &mockStrategyStore{putResult: adapternats.PutSkippedDuplicate}
	a := strategyActor(store, nil)

	a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{Strategy: validStrategy(time.Now())}})

	if got := a.stats.skippedDedup.Load(); got != 1 {
		t.Fatalf("expected skippedDedup=1, got %d", got)
	}
}

func TestStrategyProjection_PutError(t *testing.T) {
	store := &mockStrategyStore{
		putResult:  adapternats.PutWritten,
		putProblem: problem.New(problem.Unavailable, "NATS down"),
	}
	a := strategyActor(store, nil)

	a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{Strategy: validStrategy(time.Now())}})

	if got := a.stats.errors.Load(); got != 1 {
		t.Fatalf("expected errors=1, got %d", got)
	}
}

func TestStrategyProjection_NoTracker_DoesNotPanic(t *testing.T) {
	store := &mockStrategyStore{putResult: adapternats.PutWritten}
	a := strategyActor(store, nil)

	a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{Strategy: validStrategy(time.Now())}})

	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
}

func TestStrategyProjection_AllDirectionValues_PassValidation(t *testing.T) {
	directions := []strategy.Direction{
		strategy.DirectionLong,
		strategy.DirectionShort,
		strategy.DirectionFlat,
	}

	for _, dir := range directions {
		store := &mockStrategyStore{putResult: adapternats.PutWritten}
		a := strategyActor(store, nil)

		strat := validStrategy(time.Now())
		strat.Direction = dir

		a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{Strategy: strat}})

		if store.putCalls != 1 {
			t.Errorf("direction %q: expected 1 put call, got %d", dir, store.putCalls)
		}
	}
}

func TestStrategyProjection_MultipleEvents_StatsAccumulate(t *testing.T) {
	store := &mockStrategyStore{putResult: adapternats.PutWritten}
	a := strategyActor(store, nil)

	now := time.Now()
	for i := 0; i < 4; i++ {
		a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{
			Strategy: validStrategy(now.Add(time.Duration(i) * time.Minute)),
		}})
	}

	if got := a.stats.received.Load(); got != 4 {
		t.Fatalf("expected received=4, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 4 {
		t.Fatalf("expected materialized=4, got %d", got)
	}
}

func TestStrategyProjection_MultiSymbol_IndependentMaterialization(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt"}
	timeframes := []int{60, 300}

	store := &mockStrategyStore{putResult: adapternats.PutWritten}
	tracker := healthz.NewTracker("test")
	a := strategyActor(store, tracker)

	now := time.Now()
	eventCount := 0
	for _, sym := range symbols {
		for _, tf := range timeframes {
			strat := validStrategy(now.Add(time.Duration(eventCount) * time.Minute))
			strat.Symbol = sym
			strat.Timeframe = tf
			a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{Strategy: strat}})
			eventCount++
		}
	}

	expectedCount := int64(len(symbols) * len(timeframes))
	if got := a.stats.received.Load(); got != expectedCount {
		t.Fatalf("expected received=%d, got %d", expectedCount, got)
	}
	if got := a.stats.materialized.Load(); got != expectedCount {
		t.Fatalf("expected materialized=%d, got %d", expectedCount, got)
	}
	if store.putCalls != int(expectedCount) {
		t.Fatalf("expected %d put calls, got %d", expectedCount, store.putCalls)
	}
	if got := int64(tracker.EventCount()); got != expectedCount {
		t.Fatalf("expected tracker count=%d, got %d", expectedCount, got)
	}
}

func TestStrategyProjection_MultiSymbol_NoBleed_PartitionKeys(t *testing.T) {
	// Verify that strategies for different symbols produce distinct partition keys,
	// ensuring KV store isolation.
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	timeframes := []int{60, 300}
	keys := make(map[string]string) // partition key → symbol

	now := time.Now()
	for _, sym := range symbols {
		for _, tf := range timeframes {
			strat := validStrategy(now)
			strat.Symbol = sym
			strat.Timeframe = tf
			key := strat.PartitionKey()
			if existing, collision := keys[key]; collision {
				t.Fatalf("partition key collision: %q used by both %q and %q", key, existing, sym)
			}
			keys[key] = sym
		}
	}

	expectedCount := len(symbols) * len(timeframes)
	if len(keys) != expectedCount {
		t.Fatalf("expected %d unique partition keys, got %d", expectedCount, len(keys))
	}
}

func TestStrategyProjection_MultiSymbol_DeduplicationKeys(t *testing.T) {
	// Verify that deduplication keys are unique per symbol even at the same timestamp.
	symbols := []string{"btcusdt", "ethusdt"}
	ts := time.Now()
	dedupKeys := make(map[string]string)

	for _, sym := range symbols {
		strat := validStrategy(ts)
		strat.Symbol = sym
		key := strat.DeduplicationKey()
		if existing, collision := dedupKeys[key]; collision {
			t.Fatalf("dedup key collision: %q used by both %q and %q", key, existing, sym)
		}
		dedupKeys[key] = sym
	}

	if len(dedupKeys) != len(symbols) {
		t.Fatalf("expected %d unique dedup keys, got %d", len(symbols), len(dedupKeys))
	}
}

func TestStrategyProjection_StatsInvariant_ReceivedEqualsSum(t *testing.T) {
	store := &mockStrategyStore{putResult: adapternats.PutWritten}
	a := strategyActor(store, nil)

	now := time.Now()

	// 1 valid final → materialized
	a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{
		Strategy: validStrategy(now),
	}})

	// 1 non-final → skippedNonFinal
	nonFinal := validStrategy(now.Add(time.Minute))
	nonFinal.Final = false
	a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{Strategy: nonFinal}})

	// 1 invalid → rejected
	invalid := strategy.Strategy{Final: true}
	a.onStrategy(strategyReceivedMessage{Event: strategy.StrategyResolvedEvent{Strategy: invalid}})

	received := a.stats.received.Load()
	sum := a.stats.materialized.Load() +
		a.stats.skippedStale.Load() +
		a.stats.skippedDedup.Load() +
		a.stats.skippedNonFinal.Load() +
		a.stats.rejected.Load() +
		a.stats.errors.Load()

	if received != sum {
		t.Fatalf("stats invariant broken: received=%d != sum=%d (materialized=%d, skippedStale=%d, skippedDedup=%d, skippedNonFinal=%d, rejected=%d, errors=%d)",
			received, sum,
			a.stats.materialized.Load(),
			a.stats.skippedStale.Load(),
			a.stats.skippedDedup.Load(),
			a.stats.skippedNonFinal.Load(),
			a.stats.rejected.Load(),
			a.stats.errors.Load(),
		)
	}
}
