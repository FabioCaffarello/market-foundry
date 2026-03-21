package store

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"internal/adapters/nats/natskit"
	"internal/domain/decision"
	"internal/shared/healthz"
	"internal/shared/problem"
)

type mockDecisionStore struct {
	putResult  natskit.PutResult
	putProblem *problem.Problem
	putCalls   int
}

func (m *mockDecisionStore) Put(_ context.Context, _ decision.Decision) (natskit.PutResult, *problem.Problem) {
	m.putCalls++
	return m.putResult, m.putProblem
}

func validDecision(ts time.Time) decision.Decision {
	return decision.Decision{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Outcome:    decision.OutcomeTriggered,
		Severity:   decision.SeverityLow,
		Confidence: "0.85",
		Rationale:  "RSI 28.5 below oversold threshold 30.0 (distance 5.0%); severity low",
		Signals: []decision.SignalInput{
			{Type: "rsi", Value: "28.5", Timeframe: 60},
		},
		Final:     true,
		Timestamp: ts,
	}
}

func decisionActor(store *mockDecisionStore, tracker *healthz.Tracker) *DecisionProjectionActor {
	return &DecisionProjectionActor{
		cfg:    DecisionProjectionConfig{Bucket: "DECISION_RSI_OVERSOLD_LATEST", Tracker: tracker},
		logger: slog.Default(),
		store:  store,
	}
}

func TestDecisionProjection_FinalGate_SkipsNonFinal(t *testing.T) {
	store := &mockDecisionStore{putResult: natskit.PutWritten}
	a := decisionActor(store, nil)

	dec := validDecision(time.Now())
	dec.Final = false

	a.onDecision(decisionReceivedMessage{Event: decision.DecisionEvaluatedEvent{Decision: dec}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls for non-final decision, got %d", store.putCalls)
	}
	if got := a.stats.skippedNonFinal.Load(); got != 1 {
		t.Fatalf("expected skippedNonFinal=1, got %d", got)
	}
	if got := a.stats.received.Load(); got != 1 {
		t.Fatalf("expected received=1, got %d", got)
	}
}

func TestDecisionProjection_ValidationGate_RejectsMalformed(t *testing.T) {
	store := &mockDecisionStore{putResult: natskit.PutWritten}
	a := decisionActor(store, nil)

	dec := decision.Decision{Final: true} // missing required fields

	a.onDecision(decisionReceivedMessage{Event: decision.DecisionEvaluatedEvent{Decision: dec}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls, got %d", store.putCalls)
	}
	if got := a.stats.rejected.Load(); got != 1 {
		t.Fatalf("expected rejected=1, got %d", got)
	}
}

func TestDecisionProjection_ValidationGate_RejectsInvalidOutcome(t *testing.T) {
	store := &mockDecisionStore{putResult: natskit.PutWritten}
	a := decisionActor(store, nil)

	dec := validDecision(time.Now())
	dec.Outcome = "invalid_outcome"

	a.onDecision(decisionReceivedMessage{Event: decision.DecisionEvaluatedEvent{Decision: dec}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls for invalid outcome, got %d", store.putCalls)
	}
	if got := a.stats.rejected.Load(); got != 1 {
		t.Fatalf("expected rejected=1, got %d", got)
	}
}

func TestDecisionProjection_PutWritten_Materializes(t *testing.T) {
	store := &mockDecisionStore{putResult: natskit.PutWritten}
	tracker := healthz.NewTracker("test")
	a := decisionActor(store, tracker)

	a.onDecision(decisionReceivedMessage{Event: decision.DecisionEvaluatedEvent{Decision: validDecision(time.Now())}})

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

func TestDecisionProjection_PutSkippedStale(t *testing.T) {
	store := &mockDecisionStore{putResult: natskit.PutSkippedStale}
	a := decisionActor(store, nil)

	a.onDecision(decisionReceivedMessage{Event: decision.DecisionEvaluatedEvent{Decision: validDecision(time.Now())}})

	if got := a.stats.skippedStale.Load(); got != 1 {
		t.Fatalf("expected skippedStale=1, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 0 {
		t.Fatalf("expected materialized=0, got %d", got)
	}
}

func TestDecisionProjection_PutSkippedDuplicate(t *testing.T) {
	store := &mockDecisionStore{putResult: natskit.PutSkippedDuplicate}
	a := decisionActor(store, nil)

	a.onDecision(decisionReceivedMessage{Event: decision.DecisionEvaluatedEvent{Decision: validDecision(time.Now())}})

	if got := a.stats.skippedDedup.Load(); got != 1 {
		t.Fatalf("expected skippedDedup=1, got %d", got)
	}
}

func TestDecisionProjection_PutError(t *testing.T) {
	store := &mockDecisionStore{
		putResult:  natskit.PutWritten,
		putProblem: problem.New(problem.Unavailable, "NATS down"),
	}
	a := decisionActor(store, nil)

	a.onDecision(decisionReceivedMessage{Event: decision.DecisionEvaluatedEvent{Decision: validDecision(time.Now())}})

	if got := a.stats.errors.Load(); got != 1 {
		t.Fatalf("expected errors=1, got %d", got)
	}
}

func TestDecisionProjection_NoTracker_DoesNotPanic(t *testing.T) {
	store := &mockDecisionStore{putResult: natskit.PutWritten}
	a := decisionActor(store, nil)

	a.onDecision(decisionReceivedMessage{Event: decision.DecisionEvaluatedEvent{Decision: validDecision(time.Now())}})

	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
}

func TestDecisionProjection_AllOutcomeValues_PassValidation(t *testing.T) {
	outcomes := []decision.Outcome{
		decision.OutcomeTriggered,
		decision.OutcomeNotTriggered,
		decision.OutcomeInsufficient,
	}

	for _, outcome := range outcomes {
		store := &mockDecisionStore{putResult: natskit.PutWritten}
		a := decisionActor(store, nil)

		dec := validDecision(time.Now())
		dec.Outcome = outcome

		a.onDecision(decisionReceivedMessage{Event: decision.DecisionEvaluatedEvent{Decision: dec}})

		if store.putCalls != 1 {
			t.Errorf("outcome %q: expected 1 put call, got %d", outcome, store.putCalls)
		}
	}
}

func TestDecisionProjection_MultipleEvents_StatsAccumulate(t *testing.T) {
	store := &mockDecisionStore{putResult: natskit.PutWritten}
	a := decisionActor(store, nil)

	now := time.Now()
	for i := 0; i < 4; i++ {
		a.onDecision(decisionReceivedMessage{Event: decision.DecisionEvaluatedEvent{
			Decision: validDecision(now.Add(time.Duration(i) * time.Minute)),
		}})
	}

	if got := a.stats.received.Load(); got != 4 {
		t.Fatalf("expected received=4, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 4 {
		t.Fatalf("expected materialized=4, got %d", got)
	}
}
