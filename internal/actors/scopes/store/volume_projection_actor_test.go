package store

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"internal/adapters/nats/natskit"
	"internal/domain/evidence"
	"internal/shared/healthz"
	"internal/shared/problem"
)

type mockVolumeStore struct {
	putResult  natskit.PutResult
	putProblem *problem.Problem
	putCalls   int
}

func (m *mockVolumeStore) Put(_ context.Context, _ evidence.EvidenceVolume) (natskit.PutResult, *problem.Problem) {
	m.putCalls++
	return m.putResult, m.putProblem
}

func validVolume(openTime time.Time) evidence.EvidenceVolume {
	return evidence.EvidenceVolume{
		Source:      "binancef",
		Symbol:      "btcusdt",
		Timeframe:   60,
		BuyVolume:   "500000.00",
		SellVolume:  "300000.00",
		TotalVolume: "800000.00",
		VWAP:        "50123.45",
		TradeCount:  200,
		OpenTime:    openTime,
		CloseTime:   openTime.Add(60 * time.Second),
		Final:       true,
	}
}

func volumeActor(store *mockVolumeStore, tracker *healthz.Tracker) *VolumeProjectionActor {
	return &VolumeProjectionActor{
		cfg:    VolumeProjectionConfig{Tracker: tracker},
		logger: slog.Default(),
		store:  store,
	}
}

func TestVolumeProjection_FinalGate_SkipsNonFinal(t *testing.T) {
	store := &mockVolumeStore{putResult: natskit.PutWritten}
	a := volumeActor(store, nil)

	vol := validVolume(time.Now())
	vol.Final = false

	a.onVolume(volumeReceivedMessage{Event: evidence.VolumeSampledEvent{Volume: vol}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls for non-final volume, got %d", store.putCalls)
	}
	if got := a.stats.skippedNonFinal.Load(); got != 1 {
		t.Fatalf("expected skippedNonFinal=1, got %d", got)
	}
}

func TestVolumeProjection_ValidationGate_RejectsMalformed(t *testing.T) {
	store := &mockVolumeStore{putResult: natskit.PutWritten}
	a := volumeActor(store, nil)

	vol := evidence.EvidenceVolume{Final: true}

	a.onVolume(volumeReceivedMessage{Event: evidence.VolumeSampledEvent{Volume: vol}})

	if store.putCalls != 0 {
		t.Fatalf("expected 0 put calls, got %d", store.putCalls)
	}
	if got := a.stats.rejected.Load(); got != 1 {
		t.Fatalf("expected rejected=1, got %d", got)
	}
}

func TestVolumeProjection_PutWritten_Materializes(t *testing.T) {
	store := &mockVolumeStore{putResult: natskit.PutWritten}
	tracker := healthz.NewTracker("test")
	a := volumeActor(store, tracker)

	a.onVolume(volumeReceivedMessage{Event: evidence.VolumeSampledEvent{Volume: validVolume(time.Now())}})

	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
	if tracker.EventCount() != 1 {
		t.Fatalf("expected tracker count=1, got %d", tracker.EventCount())
	}
}

func TestVolumeProjection_PutSkippedStale(t *testing.T) {
	store := &mockVolumeStore{putResult: natskit.PutSkippedStale}
	a := volumeActor(store, nil)

	a.onVolume(volumeReceivedMessage{Event: evidence.VolumeSampledEvent{Volume: validVolume(time.Now())}})

	if got := a.stats.skippedStale.Load(); got != 1 {
		t.Fatalf("expected skippedStale=1, got %d", got)
	}
	if got := a.stats.materialized.Load(); got != 0 {
		t.Fatalf("expected materialized=0, got %d", got)
	}
}

func TestVolumeProjection_PutSkippedDuplicate(t *testing.T) {
	store := &mockVolumeStore{putResult: natskit.PutSkippedDuplicate}
	a := volumeActor(store, nil)

	a.onVolume(volumeReceivedMessage{Event: evidence.VolumeSampledEvent{Volume: validVolume(time.Now())}})

	if got := a.stats.skippedDedup.Load(); got != 1 {
		t.Fatalf("expected skippedDedup=1, got %d", got)
	}
}

func TestVolumeProjection_PutError(t *testing.T) {
	store := &mockVolumeStore{
		putResult:  natskit.PutWritten,
		putProblem: problem.New(problem.Unavailable, "NATS down"),
	}
	a := volumeActor(store, nil)

	a.onVolume(volumeReceivedMessage{Event: evidence.VolumeSampledEvent{Volume: validVolume(time.Now())}})

	if got := a.stats.errors.Load(); got != 1 {
		t.Fatalf("expected errors=1, got %d", got)
	}
}

func TestVolumeProjection_NoTracker_DoesNotPanic(t *testing.T) {
	store := &mockVolumeStore{putResult: natskit.PutWritten}
	a := volumeActor(store, nil)

	a.onVolume(volumeReceivedMessage{Event: evidence.VolumeSampledEvent{Volume: validVolume(time.Now())}})

	if got := a.stats.materialized.Load(); got != 1 {
		t.Fatalf("expected materialized=1, got %d", got)
	}
}
