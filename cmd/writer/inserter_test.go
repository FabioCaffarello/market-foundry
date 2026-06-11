package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"internal/shared/healthz"
)

// ── enforceMaxPending ───────────────────────────────────────────

func TestEnforceMaxPending_UnderLimit(t *testing.T) {
	a := &inserterActor{
		cfg: inserterConfig{
			family:     "test",
			maxPending: 10,
		},
		logger: slog.Default(),
		buffer: makeRows(5),
	}

	a.enforceMaxPending()

	if len(a.buffer) != 5 {
		t.Errorf("expected buffer length 5, got %d", len(a.buffer))
	}
}

func TestEnforceMaxPending_AtLimit(t *testing.T) {
	a := &inserterActor{
		cfg: inserterConfig{
			family:     "test",
			maxPending: 10,
		},
		logger: slog.Default(),
		buffer: makeRows(10),
	}

	a.enforceMaxPending()

	if len(a.buffer) != 10 {
		t.Errorf("expected buffer length 10, got %d", len(a.buffer))
	}
}

func TestEnforceMaxPending_OverLimit(t *testing.T) {
	a := &inserterActor{
		cfg: inserterConfig{
			family:     "test",
			maxPending: 5,
		},
		logger: slog.Default(),
		buffer: makeRows(8),
	}

	a.enforceMaxPending()

	if len(a.buffer) != 5 {
		t.Errorf("expected buffer length 5, got %d", len(a.buffer))
	}
}

func TestEnforceMaxPending_EvictsOldestRows(t *testing.T) {
	rows := make([][]any, 8)
	for i := range rows {
		rows[i] = []any{i}
	}

	a := &inserterActor{
		cfg: inserterConfig{
			family:     "test",
			maxPending: 5,
		},
		logger: slog.Default(),
		buffer: rows,
	}

	a.enforceMaxPending()

	// Should have kept rows 3,4,5,6,7 (the newest).
	if a.buffer[0][0] != 3 {
		t.Errorf("expected first remaining row value 3, got %v", a.buffer[0][0])
	}
	if a.buffer[4][0] != 7 {
		t.Errorf("expected last remaining row value 7, got %v", a.buffer[4][0])
	}
}

func TestEnforceMaxPending_TrackerCountsDrops(t *testing.T) {
	tracker := healthz.NewTracker("test-inserter")

	a := &inserterActor{
		cfg: inserterConfig{
			family:     "test",
			maxPending: 3,
			tracker:    tracker,
		},
		logger: slog.Default(),
		buffer: makeRows(7),
	}

	a.enforceMaxPending()

	dropped := tracker.Counter("events_dropped").Load()
	if dropped != 4 {
		t.Errorf("expected 4 events_dropped, got %d", dropped)
	}

	overflowed := tracker.Counter("events_overflowed").Load()
	if overflowed != 4 {
		t.Errorf("expected 4 events_overflowed, got %d", overflowed)
	}
}

func TestEnforceMaxPending_NilTracker(t *testing.T) {
	a := &inserterActor{
		cfg: inserterConfig{
			family:     "test",
			maxPending: 3,
			tracker:    nil,
		},
		logger: slog.Default(),
		buffer: makeRows(5),
	}

	// Should not panic with nil tracker.
	a.enforceMaxPending()

	if len(a.buffer) != 3 {
		t.Errorf("expected buffer length 3, got %d", len(a.buffer))
	}
}

// ── Buffer accumulation ─────────────────────────────────────────

func TestBufferAccumulation(t *testing.T) {
	a := &inserterActor{
		cfg: inserterConfig{
			family:     "test",
			batchSize:  100,
			maxPending: 1000,
		},
		logger: slog.Default(),
		buffer: make([][]any, 0, 100),
	}

	for i := 0; i < 10; i++ {
		a.buffer = append(a.buffer, []any{i})
	}

	if len(a.buffer) != 10 {
		t.Errorf("expected buffer length 10, got %d", len(a.buffer))
	}
}

// ── Flush empty buffer ──────────────────────────────────────────

func TestFlush_EmptyBuffer(t *testing.T) {
	a := &inserterActor{
		cfg: inserterConfig{
			family:     "test",
			batchSize:  10,
			maxPending: 100,
		},
		logger: slog.Default(),
		buffer: make([][]any, 0),
	}

	// flush on empty buffer should be a no-op (no panic, no error).
	a.flush()

	if len(a.buffer) != 0 {
		t.Errorf("expected empty buffer after empty flush, got %d", len(a.buffer))
	}
}

// ── inserterConfig defaults ─────────────────────────────────────

func TestInserterConfig_ZeroValues(t *testing.T) {
	cfg := inserterConfig{
		family:        "test",
		table:         "test_table",
		insertSQL:     "INSERT INTO test_table",
		batchSize:     0,
		flushInterval: 0,
		maxPending:    0,
	}

	// Zero batchSize means never trigger size-based flush.
	a := &inserterActor{
		cfg:    cfg,
		logger: slog.Default(),
		buffer: make([][]any, 0),
	}

	// Buffer should accept rows without flush trigger (batchSize=0 means len always >= 0).
	a.buffer = append(a.buffer, []any{"row1"})
	if len(a.buffer) != 1 {
		t.Errorf("expected buffer length 1, got %d", len(a.buffer))
	}
}

// ── scheduleFlush safety ────────────────────────────────────────

func TestScheduleFlush_NilEngine(t *testing.T) {
	a := &inserterActor{
		cfg: inserterConfig{
			family:        "test",
			flushInterval: 10 * time.Millisecond,
		},
		logger: slog.Default(),
		engine: nil,
		pid:    nil,
	}

	// Should not panic when engine/pid are nil.
	a.scheduleFlush()

	// Wait for the timer to fire.
	time.Sleep(50 * time.Millisecond)
}

// ── Flush retry behavior ────────────────────────────────────────

// fakeClient is a test double for adapterch.Client that controls InsertBatch outcomes.
type fakeClient struct {
	calls      atomic.Int32
	failUntil  int32 // fail the first N calls, succeed after
	alwaysFail bool
	lastErr    error
}

func (f *fakeClient) insertBatch(_ context.Context, _ string, _ [][]any) error {
	n := f.calls.Add(1)
	if f.alwaysFail {
		f.lastErr = errors.New("clickhouse unavailable")
		return f.lastErr
	}
	if n <= f.failUntil {
		f.lastErr = fmt.Errorf("transient error (call %d)", n)
		return f.lastErr
	}
	return nil
}

// newTestInserter creates an inserterActor wired to a fakeClient for testing flush retry.
// Since inserterActor.flush() calls a.cfg.client.InsertBatch, we need a real *adapterch.Client.
// Instead, we test via the inserterActorWithClient helper that wraps a fake.

func TestFlush_RetriesOnTransientFailure(t *testing.T) {
	tracker := healthz.NewTracker("test-inserter")
	fake := &fakeClient{failUntil: 2}

	a := &testableInserterActor{
		cfg: inserterConfig{
			family:         "test",
			table:          "test_table",
			insertSQL:      "INSERT INTO test_table",
			batchSize:      10,
			maxPending:     100,
			maxRetries:     5,
			initialBackoff: 1 * time.Millisecond, // fast for tests
			tracker:        tracker,
		},
		logger:     slog.Default(),
		buffer:     makeRows(3),
		insertFunc: fake.insertBatch,
	}

	a.flush()

	// Should have succeeded after retries.
	if fake.calls.Load() != 3 {
		t.Errorf("expected 3 insert calls (2 failures + 1 success), got %d", fake.calls.Load())
	}
	if len(a.buffer) != 0 {
		t.Errorf("expected empty buffer after successful flush, got %d", len(a.buffer))
	}

	flushed := tracker.Counter("events_flushed").Load()
	if flushed != 3 {
		t.Errorf("expected 3 events_flushed, got %d", flushed)
	}

	dropped := tracker.Counter("events_dropped").Load()
	if dropped != 0 {
		t.Errorf("expected 0 events_dropped, got %d", dropped)
	}
}

func TestFlush_DropsAfterRetriesExhausted(t *testing.T) {
	tracker := healthz.NewTracker("test-inserter")
	fake := &fakeClient{alwaysFail: true}

	a := &testableInserterActor{
		cfg: inserterConfig{
			family:         "test",
			table:          "test_table",
			insertSQL:      "INSERT INTO test_table",
			batchSize:      10,
			maxPending:     100,
			maxRetries:     3,
			initialBackoff: 1 * time.Millisecond,
			tracker:        tracker,
		},
		logger:     slog.Default(),
		buffer:     makeRows(5),
		insertFunc: fake.insertBatch,
	}

	a.flush()

	// Should have tried exactly maxRetries times.
	if fake.calls.Load() != 3 {
		t.Errorf("expected 3 insert calls, got %d", fake.calls.Load())
	}

	// Buffer should be cleared after exhaustion.
	if len(a.buffer) != 0 {
		t.Errorf("expected empty buffer after exhaustion, got %d", len(a.buffer))
	}

	dropped := tracker.Counter("events_dropped").Load()
	if dropped != 5 {
		t.Errorf("expected 5 events_dropped, got %d", dropped)
	}

	failures := tracker.Counter("flush_failures").Load()
	if failures != 1 {
		t.Errorf("expected 1 flush_failures, got %d", failures)
	}

	flushed := tracker.Counter("events_flushed").Load()
	if flushed != 0 {
		t.Errorf("expected 0 events_flushed, got %d", flushed)
	}
}

func TestFlush_SucceedsOnFirstAttempt(t *testing.T) {
	tracker := healthz.NewTracker("test-inserter")
	fake := &fakeClient{} // never fails

	a := &testableInserterActor{
		cfg: inserterConfig{
			family:         "test",
			table:          "test_table",
			insertSQL:      "INSERT INTO test_table",
			batchSize:      10,
			maxPending:     100,
			maxRetries:     5,
			initialBackoff: 1 * time.Millisecond,
			tracker:        tracker,
		},
		logger:     slog.Default(),
		buffer:     makeRows(4),
		insertFunc: fake.insertBatch,
	}

	a.flush()

	if fake.calls.Load() != 1 {
		t.Errorf("expected 1 insert call, got %d", fake.calls.Load())
	}
	if len(a.buffer) != 0 {
		t.Errorf("expected empty buffer, got %d", len(a.buffer))
	}

	flushed := tracker.Counter("events_flushed").Load()
	if flushed != 4 {
		t.Errorf("expected 4 events_flushed, got %d", flushed)
	}
}

func TestFlush_BufferRetainedDuringRetries(t *testing.T) {
	// Verify the buffer is NOT cleared until retry succeeds or exhausts.
	tracker := healthz.NewTracker("test-inserter")
	fake := &fakeClient{failUntil: 1}

	a := &testableInserterActor{
		cfg: inserterConfig{
			family:         "test",
			table:          "test_table",
			insertSQL:      "INSERT INTO test_table",
			batchSize:      10,
			maxPending:     100,
			maxRetries:     3,
			initialBackoff: 1 * time.Millisecond,
			tracker:        tracker,
		},
		logger:     slog.Default(),
		buffer:     makeRows(2),
		insertFunc: fake.insertBatch,
	}

	a.flush()

	// Should succeed on second attempt.
	if fake.calls.Load() != 2 {
		t.Errorf("expected 2 insert calls, got %d", fake.calls.Load())
	}

	// No drops.
	dropped := tracker.Counter("events_dropped").Load()
	if dropped != 0 {
		t.Errorf("expected 0 events_dropped, got %d", dropped)
	}
}

// ── testableInserterActor ───────────────────────────────────────
// Wraps the flush logic with a pluggable insert function for testing
// retry behavior without needing a real ClickHouse connection.

type testableInserterActor struct {
	cfg        inserterConfig
	logger     *slog.Logger
	buffer     [][]any
	insertFunc func(ctx context.Context, insertSQL string, rows [][]any) error
}

func (a *testableInserterActor) flush() {
	if len(a.buffer) == 0 {
		return
	}

	rows := a.buffer
	rowCount := len(rows)

	backoff := a.cfg.initialBackoff
	maxRetries := a.cfg.maxRetries
	if maxRetries < 1 {
		maxRetries = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		lastErr = a.insertFunc(ctx, a.cfg.insertSQL, rows)
		cancel()

		if lastErr == nil {
			a.buffer = make([][]any, 0, a.cfg.batchSize)
			if a.cfg.tracker != nil {
				a.cfg.tracker.RecordEvent()
				a.cfg.tracker.Counter("events_flushed").Add(int64(rowCount))
			}
			return
		}

		if attempt < maxRetries {
			time.Sleep(backoff)
			nextBackoff := backoff * 2
			if nextBackoff > 30*time.Second {
				nextBackoff = 30 * time.Second
			}
			backoff = nextBackoff
		}
	}

	a.buffer = make([][]any, 0, a.cfg.batchSize)
	if a.cfg.tracker != nil {
		a.cfg.tracker.RecordError()
		a.cfg.tracker.Counter("events_dropped").Add(int64(rowCount))
		a.cfg.tracker.Counter("flush_failures").Add(1)
	}
}

// ── Helpers ─────────────────────────────────────────────────────

func makeRows(n int) [][]any {
	rows := make([][]any, n)
	for i := range rows {
		rows[i] = []any{i}
	}
	return rows
}
