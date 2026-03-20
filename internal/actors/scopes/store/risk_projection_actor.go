package store

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	actorcommon "internal/actors/common"
	natskit "internal/adapters/nats/natskit"
	natsrisk "internal/adapters/nats/natsrisk"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

type RiskProjectionConfig struct {
	NATSURL string
	Bucket  string
	Tracker *healthz.Tracker
}

type riskProjectionStats struct {
	received        atomic.Int64
	materialized    atomic.Int64
	skippedStale    atomic.Int64
	skippedDedup    atomic.Int64
	skippedNonFinal atomic.Int64
	rejected        atomic.Int64
	errors          atomic.Int64
}

// RiskProjectionActor materializes finalized risk assessments into a NATS KV latest bucket.
type RiskProjectionActor struct {
	cfg    RiskProjectionConfig
	logger *slog.Logger
	store  riskProjectionStore
	closer func() error
	stats  riskProjectionStats
}

func NewRiskProjectionActor(cfg RiskProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &RiskProjectionActor{cfg: cfg}
	}
}

func (a *RiskProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "risk-projection", "family", "position_exposure", "bucket", a.cfg.Bucket)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.checkStatsInvariant()
		a.logStats()
		if a.closer != nil {
			if err := a.closer(); err != nil {
				a.logger.Error("close risk KV store", "error", err)
			}
		}

	case riskReceivedMessage:
		a.onRisk(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *RiskProjectionActor) start(c *actor.Context) {
	store := natsrisk.NewKVStore(a.cfg.NATSURL, a.cfg.Bucket)
	if err := store.Start(); err != nil {
		a.logger.Error("start risk KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.store = store
	a.closer = store.Close
	a.logger.Info("risk projection started — sole writer for bucket",
		"bucket_latest", a.cfg.Bucket,
		"projection_authority", "risk-position_exposure-projection",
		"semantics", "latest-only",
	)
}

func (a *RiskProjectionActor) onRisk(msg riskReceivedMessage) {
	a.stats.received.Add(1)
	assessment := msg.Event.RiskAssessment

	// Gate 1: Only materialize finalized assessments.
	if !assessment.Final {
		a.stats.skippedNonFinal.Add(1)
		return
	}

	// Gate 2: Domain validation.
	if prob := assessment.Validate(); prob != nil {
		a.stats.rejected.Add(1)
		a.logger.Warn("risk rejected by validation",
			"error", prob.Message,
			"type", assessment.Type,
			"source", assessment.Source,
			"symbol", assessment.Symbol,
			"timeframe", assessment.Timeframe,
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, prob := a.store.Put(ctx, assessment)
	if prob != nil {
		a.stats.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("materialize risk latest",
			"error", prob.Message,
			"type", assessment.Type,
			"source", assessment.Source,
			"symbol", assessment.Symbol,
			"timeframe", assessment.Timeframe,
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
		a.cfg.Tracker.Counter("materialized:" + assessment.Symbol).Add(1)
	}

	if result == natskit.PutWritten {
		a.logger.Info("risk materialized",
			"type", assessment.Type,
			"source", assessment.Source,
			"symbol", assessment.Symbol,
			"timeframe", assessment.Timeframe,
			"disposition", string(assessment.Disposition),
			"confidence", assessment.Confidence,
			"timestamp", assessment.Timestamp.Format(time.RFC3339),
			"correlation_id", msg.Event.Metadata.CorrelationID,
			"causation_id", msg.Event.Metadata.CausationID,
		)
	}
}

func (a *RiskProjectionActor) checkStatsInvariant() {
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

func (a *RiskProjectionActor) logStats() {
	a.logger.Info("risk projection stats",
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
