package store

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"internal/adapters/nats/natskit"
	"internal/domain/risk"
	"internal/shared/healthz"
	"internal/shared/problem"
)

type mockRiskStore struct {
	putResult  natskit.PutResult
	putProblem *problem.Problem
	putCalls   int
}

func (m *mockRiskStore) Put(_ context.Context, _ risk.RiskAssessment) (natskit.PutResult, *problem.Problem) {
	m.putCalls++
	return m.putResult, m.putProblem
}

func validRiskAssessment(ts time.Time) risk.RiskAssessment {
	return risk.RiskAssessment{
		Type:        "position_exposure",
		Source:      "binancef",
		Symbol:      "btcusdt",
		Timeframe:   60,
		Disposition: risk.DispositionApproved,
		Confidence:  "0.85",
		Strategies: []risk.StrategyInput{
			{Type: "mean_reversion_entry", Direction: "long", Confidence: "0.85", Timeframe: 60},
		},
		Constraints: risk.Constraints{MaxPositionSize: "0.01", MaxExposure: "0.05"},
		Rationale:   "Position size within exposure limits",
		Parameters:  map[string]string{"max_position_pct": "0.02", "max_portfolio_exposure_pct": "0.10"},
		Final:       true,
		Timestamp:   ts,
	}
}

func riskActor(store *mockRiskStore, tracker *healthz.Tracker) *RiskProjectionActor {
	return &RiskProjectionActor{
		cfg:    RiskProjectionConfig{Bucket: "RISK_POSITION_EXPOSURE_LATEST", Tracker: tracker},
		logger: slog.Default(),
		store:  store,
	}
}

func TestRiskProjection_FinalGate_SkipsNonFinal(t *testing.T) {
	store := &mockRiskStore{putResult: natskit.PutWritten}
	a := riskActor(store, nil)

	assessment := validRiskAssessment(time.Now())
	assessment.Final = false

	a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{RiskAssessment: assessment}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls for non-final risk, got %d", store.putCalls)
	}
	if got := a.stats.skippedNonFinal.Load(); got != 1 {
		t.Fatalf("expected skippedNonFinal=1, got %d", got)
	}
	if got := a.stats.received.Load(); got != 1 {
		t.Fatalf("expected received=1, got %d", got)
	}
}

func TestRiskProjection_ValidationGate_RejectsMalformed(t *testing.T) {
	store := &mockRiskStore{putResult: natskit.PutWritten}
	a := riskActor(store, nil)

	assessment := risk.RiskAssessment{Final: true} // missing required fields

	a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{RiskAssessment: assessment}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls, got %d", store.putCalls)
	}
	if got := a.stats.rejected.Load(); got != 1 {
		t.Fatalf("expected rejected=1, got %d", got)
	}
}

func TestRiskProjection_ValidationGate_RejectsInvalidDisposition(t *testing.T) {
	store := &mockRiskStore{putResult: natskit.PutWritten}
	a := riskActor(store, nil)

	assessment := validRiskAssessment(time.Now())
	assessment.Disposition = "unknown" // invalid

	a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{RiskAssessment: assessment}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls for invalid disposition, got %d", store.putCalls)
	}
	if got := a.stats.rejected.Load(); got != 1 {
		t.Fatalf("expected rejected=1, got %d", got)
	}
}

func TestRiskProjection_PutWritten_Materializes(t *testing.T) {
	store := &mockRiskStore{putResult: natskit.PutWritten}
	tracker := healthz.NewTracker("test")
	a := riskActor(store, tracker)

	a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{RiskAssessment: validRiskAssessment(time.Now())}})

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

func TestRiskProjection_PutSkippedStale(t *testing.T) {
	store := &mockRiskStore{putResult: natskit.PutSkippedStale}
	a := riskActor(store, nil)

	a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{RiskAssessment: validRiskAssessment(time.Now())}})

	if got := a.stats.skippedStale.Load(); got != 1 {
		t.Fatalf("expected skippedStale=1, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 0 {
		t.Fatalf("expected materialized=0, got %d", got)
	}
}

func TestRiskProjection_PutSkippedDuplicate(t *testing.T) {
	store := &mockRiskStore{putResult: natskit.PutSkippedDuplicate}
	a := riskActor(store, nil)

	a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{RiskAssessment: validRiskAssessment(time.Now())}})

	if got := a.stats.skippedDedup.Load(); got != 1 {
		t.Fatalf("expected skippedDedup=1, got %d", got)
	}
}

func TestRiskProjection_PutError(t *testing.T) {
	store := &mockRiskStore{
		putResult:  natskit.PutWritten,
		putProblem: problem.New(problem.Unavailable, "NATS down"),
	}
	a := riskActor(store, nil)

	a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{RiskAssessment: validRiskAssessment(time.Now())}})

	if got := a.stats.errors.Load(); got != 1 {
		t.Fatalf("expected errors=1, got %d", got)
	}
}

func TestRiskProjection_NoTracker_DoesNotPanic(t *testing.T) {
	store := &mockRiskStore{putResult: natskit.PutWritten}
	a := riskActor(store, nil)

	a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{RiskAssessment: validRiskAssessment(time.Now())}})

	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
}

func TestRiskProjection_AllDispositionValues_PassValidation(t *testing.T) {
	dispositions := []risk.Disposition{
		risk.DispositionApproved,
		risk.DispositionModified,
		risk.DispositionRejected,
	}

	for _, disp := range dispositions {
		store := &mockRiskStore{putResult: natskit.PutWritten}
		a := riskActor(store, nil)

		assessment := validRiskAssessment(time.Now())
		assessment.Disposition = disp

		a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{RiskAssessment: assessment}})

		if store.putCalls != 1 {
			t.Errorf("disposition %q: expected 1 put call, got %d", disp, store.putCalls)
		}
	}
}

func TestRiskProjection_MultipleEvents_StatsAccumulate(t *testing.T) {
	store := &mockRiskStore{putResult: natskit.PutWritten}
	a := riskActor(store, nil)

	now := time.Now()
	for i := 0; i < 4; i++ {
		a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{
			RiskAssessment: validRiskAssessment(now.Add(time.Duration(i) * time.Minute)),
		}})
	}

	if got := a.stats.received.Load(); got != 4 {
		t.Fatalf("expected received=4, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 4 {
		t.Fatalf("expected materialized=4, got %d", got)
	}
}

func TestRiskProjection_MultiSymbol_IndependentMaterialization(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt"}
	timeframes := []int{60, 300}

	store := &mockRiskStore{putResult: natskit.PutWritten}
	tracker := healthz.NewTracker("test")
	a := riskActor(store, tracker)

	now := time.Now()
	eventCount := 0
	for _, sym := range symbols {
		for _, tf := range timeframes {
			assessment := validRiskAssessment(now.Add(time.Duration(eventCount) * time.Minute))
			assessment.Symbol = sym
			assessment.Timeframe = tf
			a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{RiskAssessment: assessment}})
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

func TestRiskProjection_MultiSymbol_NoBleed_PartitionKeys(t *testing.T) {
	// Verify that risk assessments for different symbols produce distinct partition keys,
	// ensuring KV store isolation.
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	timeframes := []int{60, 300}
	keys := make(map[string]string) // partition key → symbol

	now := time.Now()
	for _, sym := range symbols {
		for _, tf := range timeframes {
			assessment := validRiskAssessment(now)
			assessment.Symbol = sym
			assessment.Timeframe = tf
			key := assessment.PartitionKey()
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

func TestRiskProjection_MultiSymbol_DeduplicationKeys(t *testing.T) {
	// Verify that deduplication keys are unique per symbol even at the same timestamp.
	symbols := []string{"btcusdt", "ethusdt"}
	ts := time.Now()
	dedupKeys := make(map[string]string)

	for _, sym := range symbols {
		assessment := validRiskAssessment(ts)
		assessment.Symbol = sym
		key := assessment.DeduplicationKey()
		if existing, collision := dedupKeys[key]; collision {
			t.Fatalf("dedup key collision: %q used by both %q and %q", key, existing, sym)
		}
		dedupKeys[key] = sym
	}

	if len(dedupKeys) != len(symbols) {
		t.Fatalf("expected %d unique dedup keys, got %d", len(symbols), len(dedupKeys))
	}
}

func TestRiskProjection_StatsInvariant_ReceivedEqualsSum(t *testing.T) {
	store := &mockRiskStore{putResult: natskit.PutWritten}
	a := riskActor(store, nil)

	now := time.Now()

	// 1 valid final → materialized
	a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{
		RiskAssessment: validRiskAssessment(now),
	}})

	// 1 non-final → skippedNonFinal
	nonFinal := validRiskAssessment(now.Add(time.Minute))
	nonFinal.Final = false
	a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{RiskAssessment: nonFinal}})

	// 1 invalid → rejected
	invalid := risk.RiskAssessment{Final: true}
	a.onRisk(riskReceivedMessage{Event: risk.RiskAssessedEvent{RiskAssessment: invalid}})

	received := a.stats.received.Load()
	sum := a.stats.materialized.Load() +
		a.stats.skippedStale.Load() +
		a.stats.skippedDedup.Load() +
		a.stats.skippedNonFinal.Load() +
		a.stats.rejected.Load() +
		a.stats.errors.Load()

	if received != sum {
		t.Fatalf("stats invariant broken: received=%d != sum=%d", received, sum)
	}
}
