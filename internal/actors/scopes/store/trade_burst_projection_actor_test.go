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

type mockTradeBurstStore struct {
	putResult  adapternats.PutResult
	putProblem *problem.Problem
	putCalls   int
}

func (m *mockTradeBurstStore) Put(_ context.Context, _ evidence.EvidenceTradeBurst) (adapternats.PutResult, *problem.Problem) {
	m.putCalls++
	return m.putResult, m.putProblem
}

func validTradeBurst(openTime time.Time) evidence.EvidenceTradeBurst {
	return evidence.EvidenceTradeBurst{
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		TradeCount: 150,
		BuyVolume:  "500.00",
		SellVolume: "300.00",
		OpenTime:   openTime,
		CloseTime:  openTime.Add(60 * time.Second),
		Burst:      true,
		Final:      true,
	}
}

func tradeBurstActor(store *mockTradeBurstStore, tracker *healthz.Tracker) *TradeBurstProjectionActor {
	return &TradeBurstProjectionActor{
		cfg:    TradeBurstProjectionConfig{Tracker: tracker},
		logger: slog.Default(),
		store:  store,
	}
}

func TestTradeBurstProjection_FinalGate_SkipsNonFinal(t *testing.T) {
	store := &mockTradeBurstStore{putResult: adapternats.PutWritten}
	a := tradeBurstActor(store, nil)

	burst := validTradeBurst(time.Now())
	burst.Final = false

	a.onTradeBurst(tradeBurstReceivedMessage{Event: evidence.TradeBurstSampledEvent{TradeBurst: burst}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls for non-final burst, got %d", store.putCalls)
	}
	if got := a.stats.skippedNonFinal.Load(); got != 1 {
		t.Fatalf("expected skippedNonFinal=1, got %d", got)
	}
}

func TestTradeBurstProjection_ValidationGate_RejectsMalformed(t *testing.T) {
	store := &mockTradeBurstStore{putResult: adapternats.PutWritten}
	a := tradeBurstActor(store, nil)

	burst := evidence.EvidenceTradeBurst{Final: true}

	a.onTradeBurst(tradeBurstReceivedMessage{Event: evidence.TradeBurstSampledEvent{TradeBurst: burst}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls, got %d", store.putCalls)
	}
	if got := a.stats.rejected.Load(); got != 1 {
		t.Fatalf("expected rejected=1, got %d", got)
	}
}

func TestTradeBurstProjection_PutWritten_Materializes(t *testing.T) {
	store := &mockTradeBurstStore{putResult: adapternats.PutWritten}
	tracker := healthz.NewTracker("test")
	a := tradeBurstActor(store, tracker)

	a.onTradeBurst(tradeBurstReceivedMessage{Event: evidence.TradeBurstSampledEvent{TradeBurst: validTradeBurst(time.Now())}})

	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
	if tracker.EventCount() != 1 {
		t.Fatalf("expected tracker count=1, got %d", tracker.EventCount())
	}
}

func TestTradeBurstProjection_PutSkippedStale(t *testing.T) {
	store := &mockTradeBurstStore{putResult: adapternats.PutSkippedStale}
	a := tradeBurstActor(store, nil)

	a.onTradeBurst(tradeBurstReceivedMessage{Event: evidence.TradeBurstSampledEvent{TradeBurst: validTradeBurst(time.Now())}})

	if got := a.stats.skippedStale.Load(); got != 1 {
		t.Fatalf("expected skippedStale=1, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 0 {
		t.Fatalf("expected materialized=0, got %d", got)
	}
}

func TestTradeBurstProjection_PutSkippedDuplicate(t *testing.T) {
	store := &mockTradeBurstStore{putResult: adapternats.PutSkippedDuplicate}
	a := tradeBurstActor(store, nil)

	a.onTradeBurst(tradeBurstReceivedMessage{Event: evidence.TradeBurstSampledEvent{TradeBurst: validTradeBurst(time.Now())}})

	if got := a.stats.skippedDedup.Load(); got != 1 {
		t.Fatalf("expected skippedDedup=1, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 0 {
		t.Fatalf("expected materialized=0, got %d", got)
	}
}

func TestTradeBurstProjection_PutError(t *testing.T) {
	store := &mockTradeBurstStore{
		putResult:  adapternats.PutWritten,
		putProblem: problem.New(problem.Unavailable, "NATS down"),
	}
	a := tradeBurstActor(store, nil)

	a.onTradeBurst(tradeBurstReceivedMessage{Event: evidence.TradeBurstSampledEvent{TradeBurst: validTradeBurst(time.Now())}})

	if got := a.stats.errors.Load(); got != 1 {
		t.Fatalf("expected errors=1, got %d", got)
	}
}

func TestTradeBurstProjection_NoTracker_DoesNotPanic(t *testing.T) {
	store := &mockTradeBurstStore{putResult: adapternats.PutWritten}
	a := tradeBurstActor(store, nil)

	a.onTradeBurst(tradeBurstReceivedMessage{Event: evidence.TradeBurstSampledEvent{TradeBurst: validTradeBurst(time.Now())}})

	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
}
