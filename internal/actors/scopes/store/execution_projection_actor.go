package store

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// ExecutionProjectionConfig holds the configuration for the execution projection actor.
type ExecutionProjectionConfig struct {
	NATSURL string
	Bucket  string
	Tracker *healthz.Tracker
}

// ExecutionProjectionActor materializes execution events into a NATS KV read model.
// Sole writer for the configured bucket — no other actor may write to it.
// Semantics: latest-only (no history). Monotonicity enforced by timestamp in KV adapter.
type ExecutionProjectionActor struct {
	cfg    ExecutionProjectionConfig
	logger *slog.Logger
	store  *adapternats.ExecutionKVStore
	stats  executionProjectionStats
}

type executionProjectionStats struct {
	received        atomic.Int64
	materialized    atomic.Int64
	skippedStale    atomic.Int64
	skippedDedup    atomic.Int64
	skippedNonFinal atomic.Int64
	rejected        atomic.Int64
	errors          atomic.Int64
}

func NewExecutionProjectionActor(cfg ExecutionProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &ExecutionProjectionActor{cfg: cfg}
	}
}

func (a *ExecutionProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "execution-projection", "bucket", a.cfg.Bucket)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.checkStatsInvariant()
		a.logStats()
		if a.store != nil {
			if err := a.store.Close(); err != nil {
				a.logger.Error("close execution KV store", "error", err)
			}
		}

	case executionReceivedMessage:
		a.onExecution(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *ExecutionProjectionActor) start(c *actor.Context) {
	store := adapternats.NewExecutionKVStore(a.cfg.NATSURL, a.cfg.Bucket)
	if err := store.Start(); err != nil {
		a.logger.Error("start execution KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.store = store
	a.logger.Info("execution projection started — sole writer for bucket",
		"bucket_latest", a.cfg.Bucket,
		"projection_authority", "execution-paper_order-projection",
		"semantics", "latest-only",
	)
}

func (a *ExecutionProjectionActor) onExecution(msg executionReceivedMessage) {
	a.stats.received.Add(1)
	intent := msg.Event.ExecutionIntent

	// Gate 1: Skip non-final intents.
	if !intent.Final {
		a.stats.skippedNonFinal.Add(1)
		return
	}

	// Gate 2: Validate domain.
	if prob := intent.Validate(); prob != nil {
		a.stats.rejected.Add(1)
		a.logger.Warn("execution intent rejected",
			"error", prob.Message,
			"type", intent.Type,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
		)
		return
	}

	// Gate 3: Monotonicity guard (enforced by KV adapter).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, prob := a.store.Put(ctx, intent)
	if prob != nil {
		a.stats.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("put execution to KV",
			"error", prob.Message,
			"code", prob.Code,
			"type", intent.Type,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"side", string(intent.Side),
			"status", string(intent.Status),
			"correlation_id", msg.Event.Metadata.CorrelationID,
		)
		return
	}

	switch result {
	case adapternats.PutSkippedStale:
		a.stats.skippedStale.Add(1)
		return
	case adapternats.PutSkippedDuplicate:
		a.stats.skippedDedup.Add(1)
		return
	}

	if result == adapternats.PutWritten {
		a.stats.materialized.Add(1)
	}

	if a.cfg.Tracker != nil {
		a.cfg.Tracker.RecordEvent()
	}

	if result == adapternats.PutWritten {
		a.logger.Info("execution materialized",
			"type", intent.Type,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"side", string(intent.Side),
			"quantity", intent.Quantity,
			"filled_quantity", intent.FilledQuantity,
			"status", string(intent.Status),
			"fills_count", len(intent.Fills),
			"risk_disposition", intent.Risk.Disposition,
			"timestamp", intent.Timestamp.Format(time.RFC3339),
			"correlation_id", msg.Event.Metadata.CorrelationID,
			"causation_id", msg.Event.Metadata.CausationID,
		)
	}
}

func (a *ExecutionProjectionActor) checkStatsInvariant() {
	received := a.stats.received.Load()
	sum := a.stats.materialized.Load() +
		a.stats.skippedStale.Load() +
		a.stats.skippedDedup.Load() +
		a.stats.skippedNonFinal.Load() +
		a.stats.rejected.Load() +
		a.stats.errors.Load()
	if received != sum {
		a.logger.Error("stats invariant violated: received != sum of outcomes",
			"received", received,
			"sum", sum,
			"materialized", a.stats.materialized.Load(),
			"skipped_stale", a.stats.skippedStale.Load(),
			"skipped_dedup", a.stats.skippedDedup.Load(),
			"skipped_non_final", a.stats.skippedNonFinal.Load(),
			"rejected", a.stats.rejected.Load(),
			"errors", a.stats.errors.Load(),
		)
	}
}

func (a *ExecutionProjectionActor) logStats() {
	a.logger.Info("execution projection stats",
		"bucket", a.cfg.Bucket,
		"received", a.stats.received.Load(),
		"materialized", a.stats.materialized.Load(),
		"skipped_stale", a.stats.skippedStale.Load(),
		"skipped_dedup", a.stats.skippedDedup.Load(),
		"skipped_non_final", a.stats.skippedNonFinal.Load(),
		"rejected", a.stats.rejected.Load(),
		"errors", a.stats.errors.Load(),
	)
}
