package store

import (
	"context"
	"log/slog"
	"testing"
	"time"

	adapternats "internal/adapters/nats"
	"internal/domain/evidence"
	"internal/shared/healthz"
	"internal/shared/problem"
)

// --- mock store ---

type mockCandleStore struct {
	putResult  adapternats.PutResult
	putProblem *problem.Problem
	histProblem *problem.Problem

	putCalls     int
	historyCalls int
	lastCandle   evidence.EvidenceCandle
}

func (m *mockCandleStore) Put(_ context.Context, candle evidence.EvidenceCandle) (adapternats.PutResult, *problem.Problem) {
	m.putCalls++
	m.lastCandle = candle
	return m.putResult, m.putProblem
}

func (m *mockCandleStore) PutHistory(_ context.Context, candle evidence.EvidenceCandle) *problem.Problem {
	m.historyCalls++
	m.lastCandle = candle
	return m.histProblem
}

// --- helpers ---

func validCandle(openTime time.Time) evidence.EvidenceCandle {
	return evidence.EvidenceCandle{
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Open:       "100.00",
		High:       "105.00",
		Low:        "99.00",
		Close:      "102.00",
		Volume:     "1000.00",
		TradeCount: 42,
		OpenTime:   openTime,
		CloseTime:  openTime.Add(60 * time.Second),
		Final:      true,
	}
}

func candleActor(store *mockCandleStore, tracker *healthz.Tracker) *CandleProjectionActor {
	return &CandleProjectionActor{
		cfg:    CandleProjectionConfig{Tracker: tracker},
		logger: slog.Default(),
		store:  store,
	}
}

// --- tests ---

func TestCandleProjection_FinalGate_SkipsNonFinal(t *testing.T) {
	store := &mockCandleStore{putResult: adapternats.PutWritten}
	a := candleActor(store, nil)

	candle := validCandle(time.Now())
	candle.Final = false

	a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{Candle: candle}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls for non-final candle, got %d", store.putCalls)
	}
	if got := a.stats.skippedNonFinal.Load(); got != 1 {
		t.Fatalf("expected skippedNonFinal=1, got %d", got)
	}
}

func TestCandleProjection_ValidationGate_RejectsMalformed(t *testing.T) {
	store := &mockCandleStore{putResult: adapternats.PutWritten}
	a := candleActor(store, nil)

	candle := evidence.EvidenceCandle{Final: true} // missing all required fields

	a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{Candle: candle}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls for invalid candle, got %d", store.putCalls)
	}
	if got := a.stats.rejected.Load(); got != 1 {
		t.Fatalf("expected rejected=1, got %d", got)
	}
}

func TestCandleProjection_PutWritten_MaterializesLatestAndHistory(t *testing.T) {
	store := &mockCandleStore{putResult: adapternats.PutWritten}
	tracker := healthz.NewTracker("test")
	a := candleActor(store, tracker)

	a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{Candle: validCandle(time.Now())}})

	if store.putCalls != 1 {
		t.Fatalf("expected 1 put call, got %d", store.putCalls)
	}
	if store.historyCalls != 1 {
		t.Fatalf("expected 1 history call, got %d", store.historyCalls)
	}
	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
	if tracker.EventCount() != 1 {
		t.Fatalf("expected tracker event count=1, got %d", tracker.EventCount())
	}
}

func TestCandleProjection_PutSkippedStale_NoHistoryWrite(t *testing.T) {
	store := &mockCandleStore{putResult: adapternats.PutSkippedStale}
	a := candleActor(store, nil)

	a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{Candle: validCandle(time.Now())}})

	if store.historyCalls != 0 {
		t.Fatalf("expected 0 history calls on stale, got %d", store.historyCalls)
	}
	if got := a.stats.skippedStale.Load(); got != 1 {
		t.Fatalf("expected skippedStale=1, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 0 {
		t.Fatalf("expected materialized=0 on stale, got %d", got)
	}
}

func TestCandleProjection_PutSkippedDuplicate_StillWritesHistory(t *testing.T) {
	store := &mockCandleStore{putResult: adapternats.PutSkippedDuplicate}
	a := candleActor(store, nil)

	a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{Candle: validCandle(time.Now())}})

	if store.historyCalls != 1 {
		t.Fatalf("expected 1 history call on duplicate (history dedup by key), got %d", store.historyCalls)
	}
	if got := a.stats.skippedDedup.Load(); got != 1 {
		t.Fatalf("expected skippedDedup=1, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 0 {
		t.Fatalf("expected materialized=0 on duplicate, got %d", got)
	}
}

func TestCandleProjection_PutError_IncrementsErrorStat(t *testing.T) {
	store := &mockCandleStore{
		putResult:  adapternats.PutWritten,
		putProblem: problem.New(problem.Unavailable, "NATS down"),
	}
	a := candleActor(store, nil)

	a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{Candle: validCandle(time.Now())}})

	if got := a.stats.errors.Load(); got != 1 {
		t.Fatalf("expected errors=1, got %d", got)
	}
	if store.historyCalls != 0 {
		t.Fatalf("expected 0 history calls on put error, got %d", store.historyCalls)
	}
}

func TestCandleProjection_HistoryError_StillCountsMaterialized(t *testing.T) {
	store := &mockCandleStore{
		putResult:   adapternats.PutWritten,
		histProblem: problem.New(problem.Unavailable, "history write fail"),
	}
	tracker := healthz.NewTracker("test")
	a := candleActor(store, tracker)

	a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{Candle: validCandle(time.Now())}})

	if got := a.stats.errors.Load(); got != 1 {
		t.Fatalf("expected errors=1 from history failure, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1 even with history error, got %d", got)
	}
	if tracker.EventCount() != 1 {
		t.Fatalf("expected tracker event count=1, got %d", tracker.EventCount())
	}
}

func TestCandleProjection_NoTracker_DoesNotPanic(t *testing.T) {
	store := &mockCandleStore{putResult: adapternats.PutWritten}
	a := candleActor(store, nil)

	// Should not panic with nil tracker.
	a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{Candle: validCandle(time.Now())}})

	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
}

func TestCandleProjection_MultipleEvents_StatsAccumulate(t *testing.T) {
	store := &mockCandleStore{putResult: adapternats.PutWritten}
	a := candleActor(store, nil)

	now := time.Now()
	for i := 0; i < 5; i++ {
		a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{
			Candle: validCandle(now.Add(time.Duration(i) * time.Minute)),
		}})
	}

	if got := a.stats.materialized.Load(); got != 5 {
		t.Fatalf("expected materialized=5, got %d", got)
	}
	if store.putCalls != 5 {
		t.Fatalf("expected 5 put calls, got %d", store.putCalls)
	}
	if store.historyCalls != 5 {
		t.Fatalf("expected 5 history calls, got %d", store.historyCalls)
	}
}

func TestCandleProjection_MixedOutcomes(t *testing.T) {
	store := &mockCandleStore{}
	a := candleActor(store, nil)

	now := time.Now()

	// 1. valid + written
	store.putResult = adapternats.PutWritten
	a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{Candle: validCandle(now)}})

	// 2. non-final
	nonFinal := validCandle(now.Add(time.Minute))
	nonFinal.Final = false
	a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{Candle: nonFinal}})

	// 3. stale
	store.putResult = adapternats.PutSkippedStale
	a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{Candle: validCandle(now.Add(2 * time.Minute))}})

	// 4. duplicate
	store.putResult = adapternats.PutSkippedDuplicate
	a.onCandle(candleReceivedMessage{Event: evidence.CandleSampledEvent{Candle: validCandle(now.Add(3 * time.Minute))}})

	if got := a.stats.materialized.Load(); got != 1 {
		t.Errorf("materialized: want 1, got %d", got)
	}
	if got := a.stats.skippedNonFinal.Load(); got != 1 {
		t.Errorf("skippedNonFinal: want 1, got %d", got)
	}
	if got := a.stats.skippedStale.Load(); got != 1 {
		t.Errorf("skippedStale: want 1, got %d", got)
	}
	if got := a.stats.skippedDedup.Load(); got != 1 {
		t.Errorf("skippedDedup: want 1, got %d", got)
	}
}
