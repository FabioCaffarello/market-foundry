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

type DecisionProjectionConfig struct {
	NATSURL string
	Bucket  string
	Tracker *healthz.Tracker
}

type decisionProjectionStats struct {
	received        atomic.Int64
	materialized    atomic.Int64
	skippedStale    atomic.Int64
	skippedDedup    atomic.Int64
	skippedNonFinal atomic.Int64
	rejected        atomic.Int64
	errors          atomic.Int64
}

// DecisionProjectionActor materializes finalized decisions into a NATS KV latest bucket.
type DecisionProjectionActor struct {
	cfg    DecisionProjectionConfig
	logger *slog.Logger
	store  decisionProjectionStore
	closer func() error
	stats  decisionProjectionStats
}

func NewDecisionProjectionActor(cfg DecisionProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &DecisionProjectionActor{cfg: cfg}
	}
}

func (a *DecisionProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "decision-projection", "family", "rsi_oversold", "bucket", a.cfg.Bucket)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.checkStatsInvariant()
		a.logStats()
		if a.closer != nil {
			if err := a.closer(); err != nil {
				a.logger.Error("close decision KV store", "error", err)
			}
		}

	case decisionReceivedMessage:
		a.onDecision(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *DecisionProjectionActor) start(c *actor.Context) {
	store := adapternats.NewDecisionKVStore(a.cfg.NATSURL, a.cfg.Bucket)
	if err := store.Start(); err != nil {
		a.logger.Error("start decision KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.store = store
	a.closer = store.Close
	a.logger.Info("decision projection started",
		"bucket_latest", a.cfg.Bucket,
	)
}

func (a *DecisionProjectionActor) onDecision(msg decisionReceivedMessage) {
	a.stats.received.Add(1)
	dec := msg.Event.Decision

	// Gate 1: Only materialize finalized decisions.
	if !dec.Final {
		a.stats.skippedNonFinal.Add(1)
		return
	}

	// Gate 2: Domain validation.
	if prob := dec.Validate(); prob != nil {
		a.stats.rejected.Add(1)
		a.logger.Warn("decision rejected by validation",
			"error", prob.Message,
			"type", dec.Type,
			"source", dec.Source,
			"symbol", dec.Symbol,
			"timeframe", dec.Timeframe,
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, prob := a.store.Put(ctx, dec)
	if prob != nil {
		a.stats.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("materialize decision latest",
			"error", prob.Message,
			"type", dec.Type,
			"source", dec.Source,
			"symbol", dec.Symbol,
			"timeframe", dec.Timeframe,
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
		a.cfg.Tracker.Counter("materialized:" + dec.Symbol).Add(1)
	}

	if result == adapternats.PutWritten {
		a.logger.Info("decision materialized",
			"type", dec.Type,
			"source", dec.Source,
			"symbol", dec.Symbol,
			"timeframe", dec.Timeframe,
			"outcome", string(dec.Outcome),
			"confidence", dec.Confidence,
			"timestamp", dec.Timestamp.Format(time.RFC3339),
			"correlation_id", msg.Event.Metadata.CorrelationID,
			"causation_id", msg.Event.Metadata.CausationID,
		)
	}
}

func (a *DecisionProjectionActor) checkStatsInvariant() {
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

func (a *DecisionProjectionActor) logStats() {
	a.logger.Info("decision projection stats",
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
