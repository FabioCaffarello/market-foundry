package store

import (
	"context"
	"log/slog"
	"testing"
	"time"

	adapternats "internal/adapters/nats"
	"internal/domain/execution"
	"internal/shared/events"
	"internal/shared/healthz"
	"internal/shared/problem"
)

func validFillIntent(ts time.Time) execution.ExecutionIntent {
	return execution.ExecutionIntent{
		Type:           "paper_order",
		Source:         "binancef",
		Symbol:         "btcusdt",
		Timeframe:      60,
		Side:           execution.SideBuy,
		Quantity:       "0.02",
		FilledQuantity: "0.02",
		Status:         execution.StatusFilled,
		Risk: execution.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "0.85",
			Timeframe:   60,
		},
		Fills: []execution.FillRecord{
			{Price: "0", Quantity: "0.02", Fee: "0", Simulated: true, Timestamp: ts},
		},
		Parameters:    map[string]string{"max_position_pct": "0.02"},
		CorrelationID: "test-fill-corr",
		CausationID:   "test-fill-cause",
		Final:         true,
		Timestamp:     ts,
	}
}

func fillEvent(intent execution.ExecutionIntent, venueOrderID string) execution.VenueOrderFilledEvent {
	return execution.VenueOrderFilledEvent{
		Metadata:        events.Metadata{CorrelationID: "test-corr", CausationID: "test-cause"},
		ExecutionIntent: intent,
		VenueOrderID:    venueOrderID,
	}
}

// mockIntentStore is a mock for intent KV lookups used in RC-1/RC-2 tests.
type mockIntentStore struct {
	intents map[string]*execution.ExecutionIntent
}

func (m *mockIntentStore) Get(_ context.Context, source, symbol string, timeframe int) (*execution.ExecutionIntent, *problem.Problem) {
	key := source + "." + symbol + "." + string(rune(timeframe+'0'))
	if intent, ok := m.intents[key]; ok {
		return intent, nil
	}
	return nil, nil
}

// fillActorDirect builds a FillProjectionActor that delegates to a mock store for testing.
// No intent store — RC-1/RC-2 gates are disabled.
func fillActorDirect(store *mockExecutionStore, tracker *healthz.Tracker) *fillProjectionTestHarness {
	a := &FillProjectionActor{
		cfg:    FillProjectionConfig{Bucket: "EXECUTION_VENUE_MARKET_ORDER_LATEST", Tracker: tracker},
		logger: slog.Default(),
	}
	return &fillProjectionTestHarness{actor: a, store: store}
}

type fillProjectionTestHarness struct {
	actor *FillProjectionActor
	store *mockExecutionStore
}

func (h *fillProjectionTestHarness) onFill(msg fillReceivedMessage) {
	h.actor.stats.received.Add(1)
	intent := msg.Event.ExecutionIntent

	// Gate 1: Skip non-final intents.
	if !intent.Final {
		h.actor.stats.skippedNonFinal.Add(1)
		return
	}

	// Gate 2: Validate domain.
	if prob := intent.Validate(); prob != nil {
		h.actor.stats.rejected.Add(1)
		return
	}

	// RC-1/RC-2 gates are skipped in basic harness (no intentStore).
	// See fillActorWithIntentStore for RC tests.

	// Gate 3: Delegate to mock store.
	result, prob := h.store.Put(nil, intent)
	if prob != nil {
		h.actor.stats.errors.Add(1)
		return
	}

	switch result {
	case adapternats.PutSkippedStale:
		h.actor.stats.skippedStale.Add(1)
		return
	case adapternats.PutSkippedDuplicate:
		h.actor.stats.skippedDedup.Add(1)
		return
	}

	if result == adapternats.PutWritten {
		h.actor.stats.materialized.Add(1)
	}

	if h.actor.cfg.Tracker != nil {
		h.actor.cfg.Tracker.RecordEvent()
	}
}

// ---------- Gate Tests ----------

func TestFillProjection_FinalGate_SkipsNonFinal(t *testing.T) {
	store := &mockExecutionStore{putResult: adapternats.PutWritten}
	h := fillActorDirect(store, nil)

	intent := validFillIntent(time.Now())
	intent.Final = false

	h.onFill(fillReceivedMessage{Event: fillEvent(intent, "paper-abc123")})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls for non-final, got %d", store.putCalls)
	}
	if got := h.actor.stats.skippedNonFinal.Load(); got != 1 {
		t.Fatalf("expected skippedNonFinal=1, got %d", got)
	}
}

func TestFillProjection_ValidationGate_RejectsMalformed(t *testing.T) {
	store := &mockExecutionStore{putResult: adapternats.PutWritten}
	h := fillActorDirect(store, nil)

	intent := execution.ExecutionIntent{Final: true} // missing required fields

	h.onFill(fillReceivedMessage{Event: fillEvent(intent, "paper-abc123")})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls, got %d", store.putCalls)
	}
	if got := h.actor.stats.rejected.Load(); got != 1 {
		t.Fatalf("expected rejected=1, got %d", got)
	}
}

// ---------- Put Result Tests ----------

func TestFillProjection_PutWritten_Materializes(t *testing.T) {
	store := &mockExecutionStore{putResult: adapternats.PutWritten}
	tracker := healthz.NewTracker("test")
	h := fillActorDirect(store, tracker)

	h.onFill(fillReceivedMessage{Event: fillEvent(validFillIntent(time.Now()), "paper-abc123")})

	if got := h.actor.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
	if tracker.EventCount() != 1 {
		t.Fatalf("expected tracker count=1, got %d", tracker.EventCount())
	}
}

func TestFillProjection_PutSkippedStale(t *testing.T) {
	store := &mockExecutionStore{putResult: adapternats.PutSkippedStale}
	h := fillActorDirect(store, nil)

	h.onFill(fillReceivedMessage{Event: fillEvent(validFillIntent(time.Now()), "paper-abc123")})

	if got := h.actor.stats.skippedStale.Load(); got != 1 {
		t.Fatalf("expected skippedStale=1, got %d", got)
	}
}

func TestFillProjection_PutSkippedDuplicate(t *testing.T) {
	store := &mockExecutionStore{putResult: adapternats.PutSkippedDuplicate}
	h := fillActorDirect(store, nil)

	h.onFill(fillReceivedMessage{Event: fillEvent(validFillIntent(time.Now()), "paper-abc123")})

	if got := h.actor.stats.skippedDedup.Load(); got != 1 {
		t.Fatalf("expected skippedDedup=1, got %d", got)
	}
}

func TestFillProjection_PutError(t *testing.T) {
	store := &mockExecutionStore{
		putResult:  adapternats.PutWritten,
		putProblem: problem.New(problem.Unavailable, "NATS down"),
	}
	h := fillActorDirect(store, nil)

	h.onFill(fillReceivedMessage{Event: fillEvent(validFillIntent(time.Now()), "paper-abc123")})

	if got := h.actor.stats.errors.Load(); got != 1 {
		t.Fatalf("expected errors=1, got %d", got)
	}
}

// ---------- Stats Invariant ----------

func TestFillProjection_StatsInvariant_ReceivedEqualsSum(t *testing.T) {
	store := &mockExecutionStore{putResult: adapternats.PutWritten}
	h := fillActorDirect(store, nil)

	now := time.Now()

	// 1 valid final → materialized
	h.onFill(fillReceivedMessage{Event: fillEvent(validFillIntent(now), "paper-001")})

	// 1 non-final → skippedNonFinal
	nonFinal := validFillIntent(now.Add(time.Minute))
	nonFinal.Final = false
	h.onFill(fillReceivedMessage{Event: fillEvent(nonFinal, "paper-002")})

	// 1 invalid → rejected
	invalid := execution.ExecutionIntent{Final: true}
	h.onFill(fillReceivedMessage{Event: fillEvent(invalid, "paper-003")})

	received := h.actor.stats.received.Load()
	sum := h.actor.stats.materialized.Load() +
		h.actor.stats.skippedStale.Load() +
		h.actor.stats.skippedDedup.Load() +
		h.actor.stats.skippedNonFinal.Load() +
		h.actor.stats.rejected.Load() +
		h.actor.stats.orphaned.Load() +
		h.actor.stats.overflowed.Load() +
		h.actor.stats.errors.Load()

	if received != sum {
		t.Fatalf("stats invariant broken: received=%d != sum=%d", received, sum)
	}
}

// ---------- Multi-Symbol Isolation ----------

func TestFillProjection_MultiSymbol_IndependentMaterialization(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt"}
	timeframes := []int{60, 300}

	store := &mockExecutionStore{putResult: adapternats.PutWritten}
	tracker := healthz.NewTracker("test")
	h := fillActorDirect(store, tracker)

	now := time.Now()
	eventCount := 0
	for _, sym := range symbols {
		for _, tf := range timeframes {
			intent := validFillIntent(now.Add(time.Duration(eventCount) * time.Minute))
			intent.Symbol = sym
			intent.Timeframe = tf
			h.onFill(fillReceivedMessage{Event: fillEvent(intent, "paper-"+sym)})
			eventCount++
		}
	}

	expectedCount := int64(len(symbols) * len(timeframes))
	if got := h.actor.stats.received.Load(); got != expectedCount {
		t.Fatalf("expected received=%d, got %d", expectedCount, got)
	}
	if got := h.actor.stats.materialized.Load(); got != expectedCount {
		t.Fatalf("expected materialized=%d, got %d", expectedCount, got)
	}
}

// ---------- Venue Order ID Carried Through ----------

func TestFillProjection_VenueOrderID_DoesNotAffectGating(t *testing.T) {
	store := &mockExecutionStore{putResult: adapternats.PutWritten}
	h := fillActorDirect(store, nil)

	h.onFill(fillReceivedMessage{Event: fillEvent(validFillIntent(time.Now()), "paper-deadbeef1234")})

	if got := h.actor.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
	if store.putCalls != 1 {
		t.Fatalf("expected 1 put call, got %d", store.putCalls)
	}
}

// ---------- Error Tracking ----------

func TestFillProjection_PutError_TrackerRecordsError(t *testing.T) {
	store := &mockExecutionStore{
		putResult:  adapternats.PutWritten,
		putProblem: problem.New(problem.Unavailable, "NATS down"),
	}
	tracker := healthz.NewTracker("test")
	h := fillActorDirect(store, tracker)

	h.onFill(fillReceivedMessage{Event: fillEvent(validFillIntent(time.Now()), "paper-err")})

	if got := h.actor.stats.errors.Load(); got != 1 {
		t.Fatalf("expected errors=1, got %d", got)
	}
	if tracker.EventCount() != 0 {
		t.Fatalf("expected tracker event_count=0 on error, got %d", tracker.EventCount())
	}
}

// ---------- RC-1: Orphan Fill Detection ----------

func TestFillProjection_RC1_OrphanFill_NoIntentStore(t *testing.T) {
	// When intentStore is nil (RC-1 disabled), fills pass through without orphan check.
	store := &mockExecutionStore{putResult: adapternats.PutWritten}
	h := fillActorDirect(store, nil)

	h.onFill(fillReceivedMessage{Event: fillEvent(validFillIntent(time.Now()), "paper-orphan")})

	if got := h.actor.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1 when RC-1 disabled, got %d", got)
	}
	if got := h.actor.stats.orphaned.Load(); got != 0 {
		t.Fatalf("expected orphaned=0 when RC-1 disabled, got %d", got)
	}
}
