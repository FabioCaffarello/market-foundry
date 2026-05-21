package store

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	actorcommon "internal/actors/common"
	natsexecution "internal/adapters/nats/natsexecution"
	natskit "internal/adapters/nats/natskit"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// RejectionProjectionConfig holds the configuration for the rejection projection actor.
type RejectionProjectionConfig struct {
	NATSURL string
	Bucket  string
	Tracker *healthz.Tracker
}

// RejectionProjectionActor materializes venue order rejection events into a NATS KV read model.
// S387: Closes the persistence gap from S386 — rejection events are now queryable via KV.
// Sole writer for the configured bucket — no other actor may write to it.
// Semantics: latest-only (no history). Monotonicity enforced by timestamp in KV adapter.
type RejectionProjectionActor struct {
	cfg    RejectionProjectionConfig
	logger *slog.Logger
	store  *natsexecution.KVStore
	stats  rejectionProjectionStats
}

type rejectionProjectionStats struct {
	received     atomic.Int64
	materialized atomic.Int64
	skippedStale atomic.Int64
	skippedDedup atomic.Int64
	rejected     atomic.Int64
	errors       atomic.Int64
}

func NewRejectionProjectionActor(cfg RejectionProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &RejectionProjectionActor{cfg: cfg}
	}
}

func (a *RejectionProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "rejection-projection", "bucket", a.cfg.Bucket)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.checkStatsInvariant()
		a.logStats()
		if a.store != nil {
			if err := a.store.Close(); err != nil {
				a.logger.Error("close rejection KV store", "error", err)
			}
		}

	case rejectionReceivedMessage:
		a.onRejection(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *RejectionProjectionActor) start(c *actor.Context) {
	store := natsexecution.NewKVStore(a.cfg.NATSURL, a.cfg.Bucket)
	if err := store.Start(); err != nil {
		a.logger.Error("start rejection KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.store = store
	a.logger.Info("rejection projection started — sole writer for bucket",
		"bucket_latest", a.cfg.Bucket,
		"projection_authority", "execution-venue_rejection-projection",
		"semantics", "latest-only",
	)
}

func (a *RejectionProjectionActor) onRejection(msg rejectionReceivedMessage) {
	a.stats.received.Add(1)
	intent := msg.Event.ExecutionIntent

	// S407: Embed rejection audit detail into intent metadata so it survives KV round-trip.
	// This makes rejection_code, rejection_reason, and venue details queryable from the
	// read-path without changing the KV schema (which stores ExecutionIntent).
	if intent.Metadata == nil {
		intent.Metadata = make(map[string]string)
	}
	if msg.Event.RejectionCode != "" {
		intent.Metadata["rejection_code"] = msg.Event.RejectionCode
	}
	if msg.Event.RejectionReason != "" {
		intent.Metadata["rejection_reason"] = msg.Event.RejectionReason
	}
	for k, v := range msg.Event.VenueDetails {
		intent.Metadata["venue_detail."+k] = fmt.Sprintf("%v", v)
	}

	// Gate 1: Validate domain.
	if prob := intent.Validate(); prob != nil {
		a.stats.rejected.Add(1)
		a.logger.Warn("rejection intent rejected",
			"error", prob.Message,
			"type", intent.Type,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
		)
		return
	}

	// Gate 2: Monotonicity guard (enforced by KV adapter).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, prob := a.store.Put(ctx, intent)
	if prob != nil {
		a.stats.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("put rejection to KV",
			"error", prob.Message,
			"code", prob.Code,
			"type", intent.Type,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"side", string(intent.Side),
			"status", string(intent.Status),
			"rejection_code", msg.Event.RejectionCode,
			"rejection_reason", msg.Event.RejectionReason,
			"correlation_id", msg.Event.Metadata.CorrelationID,
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
	}

	if result == natskit.PutWritten {
		a.logger.Info("rejection materialized",
			"type", intent.Type,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"side", string(intent.Side),
			"status", string(intent.Status),
			"rejection_code", msg.Event.RejectionCode,
			"rejection_reason", msg.Event.RejectionReason,
			"timestamp", intent.Timestamp.Format(time.RFC3339),
			"correlation_id", msg.Event.Metadata.CorrelationID,
			"causation_id", msg.Event.Metadata.CausationID,
		)
	}
}

func (a *RejectionProjectionActor) checkStatsInvariant() {
	received := a.stats.received.Load()
	sum := a.stats.materialized.Load() +
		a.stats.skippedStale.Load() +
		a.stats.skippedDedup.Load() +
		a.stats.rejected.Load() +
		a.stats.errors.Load()
	if received != sum {
		a.logger.Error("stats invariant violated: received != sum of outcomes",
			"received", received,
			"sum", sum,
			"materialized", a.stats.materialized.Load(),
			"skipped_stale", a.stats.skippedStale.Load(),
			"skipped_dedup", a.stats.skippedDedup.Load(),
			"rejected", a.stats.rejected.Load(),
			"errors", a.stats.errors.Load(),
		)
	}
}

func (a *RejectionProjectionActor) logStats() {
	a.logger.Info("rejection projection stats",
		"bucket", a.cfg.Bucket,
		"received", a.stats.received.Load(),
		"materialized", a.stats.materialized.Load(),
		"skipped_stale", a.stats.skippedStale.Load(),
		"skipped_dedup", a.stats.skippedDedup.Load(),
		"rejected", a.stats.rejected.Load(),
		"errors", a.stats.errors.Load(),
	)
}
