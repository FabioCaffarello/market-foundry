package store

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"internal/adapters/nats/natskit"
	"internal/domain/signal"
	"internal/shared/healthz"
	"internal/shared/problem"
)

type mockSignalStore struct {
	putResult  natskit.PutResult
	putProblem *problem.Problem
	putCalls   int
}

func (m *mockSignalStore) Put(_ context.Context, _ signal.Signal) (natskit.PutResult, *problem.Problem) {
	m.putCalls++
	return m.putResult, m.putProblem
}

func validSignal(ts time.Time) signal.Signal {
	return signal.Signal{
		Type:       "rsi",
		Source:     "binancef",
		Instrument: btcUSDTPerpForCandleTest(),
		Timeframe:  60,
		Value:      "35.50",
		Final:      true,
		Timestamp:  ts,
	}
}

func signalActor(store *mockSignalStore, tracker *healthz.Tracker) *SignalProjectionActor {
	return &SignalProjectionActor{
		cfg:    SignalProjectionConfig{Bucket: "SIGNAL_RSI_LATEST", Tracker: tracker},
		logger: slog.Default(),
		store:  store,
	}
}

func TestSignalProjection_FinalGate_SkipsNonFinal(t *testing.T) {
	store := &mockSignalStore{putResult: natskit.PutWritten}
	a := signalActor(store, nil)

	sig := validSignal(time.Now())
	sig.Final = false

	a.onSignal(signalReceivedMessage{Event: signal.SignalGeneratedEvent{Signal: sig}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls for non-final signal, got %d", store.putCalls)
	}
	if got := a.stats.skippedNonFinal.Load(); got != 1 {
		t.Fatalf("expected skippedNonFinal=1, got %d", got)
	}
}

func TestSignalProjection_ValidationGate_RejectsMalformed(t *testing.T) {
	store := &mockSignalStore{putResult: natskit.PutWritten}
	a := signalActor(store, nil)

	sig := signal.Signal{Final: true} // missing required fields

	a.onSignal(signalReceivedMessage{Event: signal.SignalGeneratedEvent{Signal: sig}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls, got %d", store.putCalls)
	}
	if got := a.stats.rejected.Load(); got != 1 {
		t.Fatalf("expected rejected=1, got %d", got)
	}
}

func TestSignalProjection_PutWritten_Materializes(t *testing.T) {
	store := &mockSignalStore{putResult: natskit.PutWritten}
	tracker := healthz.NewTracker("test")
	a := signalActor(store, tracker)

	a.onSignal(signalReceivedMessage{Event: signal.SignalGeneratedEvent{Signal: validSignal(time.Now())}})

	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
	if tracker.EventCount() != 1 {
		t.Fatalf("expected tracker count=1, got %d", tracker.EventCount())
	}
}

func TestSignalProjection_PutSkippedStale(t *testing.T) {
	store := &mockSignalStore{putResult: natskit.PutSkippedStale}
	a := signalActor(store, nil)

	a.onSignal(signalReceivedMessage{Event: signal.SignalGeneratedEvent{Signal: validSignal(time.Now())}})

	if got := a.stats.skippedStale.Load(); got != 1 {
		t.Fatalf("expected skippedStale=1, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 0 {
		t.Fatalf("expected materialized=0, got %d", got)
	}
}

func TestSignalProjection_PutSkippedDuplicate(t *testing.T) {
	store := &mockSignalStore{putResult: natskit.PutSkippedDuplicate}
	a := signalActor(store, nil)

	a.onSignal(signalReceivedMessage{Event: signal.SignalGeneratedEvent{Signal: validSignal(time.Now())}})

	if got := a.stats.skippedDedup.Load(); got != 1 {
		t.Fatalf("expected skippedDedup=1, got %d", got)
	}
}

func TestSignalProjection_PutError(t *testing.T) {
	store := &mockSignalStore{
		putResult:  natskit.PutWritten,
		putProblem: problem.New(problem.Unavailable, "NATS down"),
	}
	a := signalActor(store, nil)

	a.onSignal(signalReceivedMessage{Event: signal.SignalGeneratedEvent{Signal: validSignal(time.Now())}})

	if got := a.stats.errors.Load(); got != 1 {
		t.Fatalf("expected errors=1, got %d", got)
	}
}

func TestSignalProjection_NoTracker_DoesNotPanic(t *testing.T) {
	store := &mockSignalStore{putResult: natskit.PutWritten}
	a := signalActor(store, nil)

	a.onSignal(signalReceivedMessage{Event: signal.SignalGeneratedEvent{Signal: validSignal(time.Now())}})

	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
}

func TestSignalProjection_MultipleEvents_StatsAccumulate(t *testing.T) {
	store := &mockSignalStore{putResult: natskit.PutWritten}
	a := signalActor(store, nil)

	now := time.Now()
	for i := 0; i < 3; i++ {
		a.onSignal(signalReceivedMessage{Event: signal.SignalGeneratedEvent{
			Signal: validSignal(now.Add(time.Duration(i) * time.Minute)),
		}})
	}

	if got := a.stats.materialized.Load(); got != 3 {
		t.Fatalf("expected materialized=3, got %d", got)
	}
}
