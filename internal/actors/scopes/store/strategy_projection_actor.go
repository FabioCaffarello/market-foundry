package store

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	actorcommon "internal/actors/common"
	natskit "internal/adapters/nats/natskit"
	natsstrategy "internal/adapters/nats/natsstrategy"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

type StrategyProjectionConfig struct {
	NATSURL string
	Bucket  string
	Tracker *healthz.Tracker
}

type strategyProjectionStats struct {
	received        atomic.Int64
	materialized    atomic.Int64
	skippedStale    atomic.Int64
	skippedDedup    atomic.Int64
	skippedNonFinal atomic.Int64
	rejected        atomic.Int64
	errors          atomic.Int64
}

// StrategyProjectionActor materializes finalized strategies into a NATS KV latest bucket.
type StrategyProjectionActor struct {
	cfg    StrategyProjectionConfig
	logger *slog.Logger
	store  strategyProjectionStore
	closer func() error
	stats  strategyProjectionStats
}

func NewStrategyProjectionActor(cfg StrategyProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &StrategyProjectionActor{cfg: cfg}
	}
}

func (a *StrategyProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "strategy-projection", "family", "mean_reversion_entry", "bucket", a.cfg.Bucket)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.checkStatsInvariant()
		a.logStats()
		if a.closer != nil {
			if err := a.closer(); err != nil {
				a.logger.Error("close strategy KV store", "error", err)
			}
		}

	case strategyReceivedMessage:
		a.onStrategy(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *StrategyProjectionActor) start(c *actor.Context) {
	store := natsstrategy.NewKVStore(a.cfg.NATSURL, a.cfg.Bucket)
	if err := store.Start(); err != nil {
		a.logger.Error("start strategy KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.store = store
	a.closer = store.Close
	a.logger.Info("strategy projection started",
		"bucket_latest", a.cfg.Bucket,
	)
}

func (a *StrategyProjectionActor) onStrategy(msg strategyReceivedMessage) {
	a.stats.received.Add(1)
	strat := msg.Event.Strategy

	// Gate 1: Only materialize finalized strategies.
	if !strat.Final {
		a.stats.skippedNonFinal.Add(1)
		return
	}

	// Gate 2: Domain validation.
	if prob := strat.Validate(); prob != nil {
		a.stats.rejected.Add(1)
		a.logger.Warn("strategy rejected by validation",
			"error", prob.Message,
			"type", strat.Type,
			"source", strat.Source,
			"symbol", strat.Symbol,
			"timeframe", strat.Timeframe,
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, prob := a.store.Put(ctx, strat)
	if prob != nil {
		a.stats.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("materialize strategy latest",
			"error", prob.Message,
			"type", strat.Type,
			"source", strat.Source,
			"symbol", strat.Symbol,
			"timeframe", strat.Timeframe,
		)
		return
	}

	switch result {
	case natskit.PutSkippedStale:
		a.stats.skippedStale.Add(1)
		return
	case natskit.PutSkippedDuplicate:
		a.stats.skippedDedup.Add(1)
		return
	}

	if result == natskit.PutWritten {
		a.stats.materialized.Add(1)
	}

	if a.cfg.Tracker != nil {
		a.cfg.Tracker.RecordEvent()
		a.cfg.Tracker.Counter("materialized:" + strat.Symbol).Add(1)
	}

	if result == natskit.PutWritten {
		a.logger.Info("strategy materialized",
			"type", strat.Type,
			"source", strat.Source,
			"symbol", strat.Symbol,
			"timeframe", strat.Timeframe,
			"direction", string(strat.Direction),
			"confidence", strat.Confidence,
			"timestamp", strat.Timestamp.Format(time.RFC3339),
			"correlation_id", msg.Event.Metadata.CorrelationID,
			"causation_id", msg.Event.Metadata.CausationID,
		)
	}
}

func (a *StrategyProjectionActor) checkStatsInvariant() {
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

func (a *StrategyProjectionActor) logStats() {
	a.logger.Info("strategy projection stats",
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
