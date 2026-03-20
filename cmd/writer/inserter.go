package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	adapterch "internal/adapters/clickhouse"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// inserterConfig holds the configuration for a batch inserter actor.
type inserterConfig struct {
	client         *adapterch.Client
	family         string
	table          string
	insertSQL      string
	batchSize      int
	flushInterval  time.Duration
	maxPending     int
	maxRetries     int
	initialBackoff time.Duration
	tracker        *healthz.Tracker
}

// insertRowMsg carries a single row to buffer for batch insertion.
type insertRowMsg struct {
	row []any
}

// flushTickMsg triggers a periodic batch flush.
type flushTickMsg struct{}

// inserterActor buffers rows from consumer actors and batch-inserts them into ClickHouse.
//
// Flush triggers:
//   - Buffer reaches batchSize
//   - Flush interval timer fires
//   - Actor stopped (drain remaining buffer)
//
// Overflow behavior: when buffer exceeds maxPending, oldest rows are evicted (FIFO).
//
// Retry behavior: failed INSERT batches are retried with exponential backoff
// up to maxRetries. Only after all retries are exhausted is the batch dropped.
type inserterActor struct {
	cfg    inserterConfig
	logger *slog.Logger
	buffer [][]any
	engine *actor.Engine
	pid    *actor.PID
}

func newInserterActor(cfg inserterConfig) actor.Producer {
	return func() actor.Receiver {
		return &inserterActor{
			cfg:    cfg,
			buffer: make([][]any, 0, cfg.batchSize),
		}
	}
}

func (a *inserterActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "writer-"+a.cfg.family+"-inserter", "table", a.cfg.table)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.engine = c.Engine()
		a.pid = c.PID()
		a.scheduleFlush()
		a.logger.Info("inserter started",
			"batch_size", a.cfg.batchSize,
			"flush_interval", a.cfg.flushInterval.String(),
			"max_pending", a.cfg.maxPending,
			"max_retries", a.cfg.maxRetries,
			"initial_backoff", a.cfg.initialBackoff.String(),
		)

	case insertRowMsg:
		a.buffer = append(a.buffer, msg.row)
		a.enforceMaxPending()
		a.updateBufferDepth()
		if len(a.buffer) >= a.cfg.batchSize {
			a.flush()
		}

	case flushTickMsg:
		a.flush()
		a.scheduleFlush()

	case actor.Stopped:
		// Drain remaining buffer on shutdown.
		if len(a.buffer) > 0 {
			a.logger.Info("draining buffer on shutdown", "rows", len(a.buffer))
			a.flush()
		}
		a.logger.Info("inserter stopped", "family", a.cfg.family)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

// flush inserts all buffered rows into ClickHouse with retry and exponential backoff.
// The buffer is only cleared on success or after all retries are exhausted.
func (a *inserterActor) flush() {
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
	flushStart := time.Now()
	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		lastErr = a.cfg.client.InsertBatch(ctx, a.cfg.insertSQL, rows)
		cancel()

		if lastErr == nil {
			flushMs := time.Since(flushStart).Milliseconds()
			// Success — clear buffer and record.
			a.buffer = make([][]any, 0, a.cfg.batchSize)
			if a.cfg.tracker != nil {
				a.cfg.tracker.RecordEvent()
				a.cfg.tracker.Counter("events_flushed").Add(int64(rowCount))
				a.cfg.tracker.Counter("flush_total").Add(1)
				a.cfg.tracker.Counter("flush_duration_ms").Store(flushMs)
			}
			a.updateBufferDepth()
			a.logger.Debug("batch flushed",
				"family", a.cfg.family,
				"table", a.cfg.table,
				"rows", rowCount,
				"flush_ms", flushMs,
			)
			return
		}

		a.logger.Warn("flush attempt failed",
			"error", lastErr,
			"family", a.cfg.family,
			"table", a.cfg.table,
			"rows", rowCount,
			"attempt", attempt,
			"max_retries", maxRetries,
		)

		if attempt < maxRetries {
			time.Sleep(backoff)
			nextBackoff := backoff * 2
			if nextBackoff > 30*time.Second {
				nextBackoff = 30 * time.Second
			}
			backoff = nextBackoff
		}
	}

	// All retries exhausted — drop batch.
	flushMs := time.Since(flushStart).Milliseconds()
	a.buffer = make([][]any, 0, a.cfg.batchSize)
	a.logger.Error("flush failed — retries exhausted, batch dropped",
		"error", lastErr,
		"family", a.cfg.family,
		"table", a.cfg.table,
		"rows_dropped", rowCount,
		"attempts", maxRetries,
		"flush_ms", flushMs,
	)
	if a.cfg.tracker != nil {
		a.cfg.tracker.RecordError()
		a.cfg.tracker.Counter("events_dropped").Add(int64(rowCount))
		a.cfg.tracker.Counter("flush_failures").Add(1)
		a.cfg.tracker.Counter("flush_duration_ms").Store(flushMs)
	}
	a.updateBufferDepth()
}

// enforceMaxPending drops oldest rows when the buffer exceeds maxPending.
// Overflow is logged at ERROR level because it means permanent data loss
// from the analytical projection.
func (a *inserterActor) enforceMaxPending() {
	if len(a.buffer) <= a.cfg.maxPending {
		return
	}
	overflow := len(a.buffer) - a.cfg.maxPending
	a.buffer = a.buffer[overflow:]
	a.logger.Error("buffer overflow — evicted oldest rows",
		"family", a.cfg.family,
		"evicted", overflow,
		"buffer_depth", len(a.buffer),
	)
	if a.cfg.tracker != nil {
		a.cfg.tracker.Counter("events_dropped").Add(int64(overflow))
		a.cfg.tracker.Counter("events_overflowed").Add(int64(overflow))
	}
}

// updateBufferDepth records the current buffer size as a gauge-style counter.
func (a *inserterActor) updateBufferDepth() {
	if a.cfg.tracker != nil {
		a.cfg.tracker.Counter("buffer_depth").Store(int64(len(a.buffer)))
	}
}

// scheduleFlush sends a flushTickMsg after the configured interval.
func (a *inserterActor) scheduleFlush() {
	time.AfterFunc(a.cfg.flushInterval, func() {
		if a.engine != nil && a.pid != nil {
			a.engine.Send(a.pid, flushTickMsg{})
		}
	})
}
