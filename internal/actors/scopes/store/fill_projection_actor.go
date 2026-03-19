package store

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// FillProjectionConfig holds the configuration for the fill projection actor.
type FillProjectionConfig struct {
	NATSURL     string
	Bucket      string
	IntentBucket string // KV bucket for intent lookup (RC-1 fill-to-intent correlation).
	Tracker     *healthz.Tracker
}

// FillProjectionActor materializes venue order fill events into a NATS KV read model.
// Sole writer for the configured bucket — no other actor may write to it.
// Semantics: latest-only (no history). Monotonicity enforced by timestamp in KV adapter.
//
// Reconciliation invariants enforced:
//   - RC-1: Fill must correlate to an existing execution intent (fill-to-intent).
//   - RC-2: Cumulative filled quantity must not exceed requested quantity.
//   - RC-4: Orphan fills (no matching intent) are logged and counted.
type FillProjectionActor struct {
	cfg         FillProjectionConfig
	logger      *slog.Logger
	store       *adapternats.ExecutionKVStore
	intentStore *adapternats.ExecutionKVStore // Read-only access to intent bucket for RC-1.
	stats       fillProjectionStats
}

type fillProjectionStats struct {
	received        atomic.Int64
	materialized    atomic.Int64
	skippedStale    atomic.Int64
	skippedDedup    atomic.Int64
	skippedNonFinal atomic.Int64
	rejected        atomic.Int64
	orphaned        atomic.Int64 // RC-4: fills without matching intent.
	overflowed      atomic.Int64 // RC-2: fills exceeding requested quantity.
	errors          atomic.Int64
}

func NewFillProjectionActor(cfg FillProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &FillProjectionActor{cfg: cfg}
	}
}

func (a *FillProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "fill-projection", "bucket", a.cfg.Bucket)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.checkStatsInvariant()
		a.logStats()
		if a.store != nil {
			if err := a.store.Close(); err != nil {
				a.logger.Error("close fill KV store", "error", err)
			}
		}
		if a.intentStore != nil {
			if err := a.intentStore.Close(); err != nil {
				a.logger.Error("close intent KV store", "error", err)
			}
		}

	case fillReceivedMessage:
		a.onFill(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *FillProjectionActor) start(c *actor.Context) {
	store := adapternats.NewExecutionKVStore(a.cfg.NATSURL, a.cfg.Bucket)
	if err := store.Start(); err != nil {
		a.logger.Error("start fill KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}
	a.store = store

	// RC-1: Open intent bucket for fill-to-intent correlation checks.
	if a.cfg.IntentBucket != "" {
		intentStore := adapternats.NewExecutionKVStore(a.cfg.NATSURL, a.cfg.IntentBucket)
		if err := intentStore.Start(); err != nil {
			a.logger.Warn("intent KV store unavailable — RC-1 correlation check disabled", "error", err)
		} else {
			a.intentStore = intentStore
		}
	}

	a.logger.Info("fill projection started — sole writer for bucket",
		"bucket_latest", a.cfg.Bucket,
		"intent_bucket", a.cfg.IntentBucket,
		"rc1_enabled", a.intentStore != nil,
		"projection_authority", "execution-venue_market_order-projection",
		"semantics", "latest-only",
	)
}

func (a *FillProjectionActor) onFill(msg fillReceivedMessage) {
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
		a.logger.Warn("fill intent rejected",
			"error", prob.Message,
			"type", intent.Type,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"venue_order_id", msg.Event.VenueOrderID,
		)
		return
	}

	// Gate RC-1: Fill-to-intent correlation (orphan detection).
	if a.intentStore != nil {
		rcCtx, rcCancel := context.WithTimeout(context.Background(), 2*time.Second)
		matchingIntent, _ := a.intentStore.Get(rcCtx, intent.Source, intent.Symbol, intent.Timeframe)
		rcCancel()
		if matchingIntent == nil {
			a.stats.orphaned.Add(1)
			a.logger.Warn("RC-1/RC-4: orphan fill — no matching intent found",
				"source", intent.Source,
				"symbol", intent.Symbol,
				"timeframe", intent.Timeframe,
				"venue_order_id", msg.Event.VenueOrderID,
				"correlation_id", msg.Event.Metadata.CorrelationID,
			)
			return
		}

		// Gate RC-2: Quantity boundary — filled quantity must not exceed requested quantity.
		if matchingIntent.Quantity != "" && intent.FilledQuantity != "" {
			requested, reqErr := strconv.ParseFloat(matchingIntent.Quantity, 64)
			filled, filErr := strconv.ParseFloat(intent.FilledQuantity, 64)
			if reqErr == nil && filErr == nil && filled > requested && requested > 0 {
				a.stats.overflowed.Add(1)
				a.logger.Warn("RC-2: fill quantity exceeds requested quantity",
					"requested", matchingIntent.Quantity,
					"filled", intent.FilledQuantity,
					"source", intent.Source,
					"symbol", intent.Symbol,
					"timeframe", intent.Timeframe,
					"venue_order_id", msg.Event.VenueOrderID,
				)
				return
			}
		}
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
		a.logger.Error("put fill to KV",
			"error", prob.Message,
			"code", prob.Code,
			"type", intent.Type,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"side", string(intent.Side),
			"status", string(intent.Status),
			"venue_order_id", msg.Event.VenueOrderID,
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
		a.logger.Info("fill materialized",
			"type", intent.Type,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"side", string(intent.Side),
			"quantity", intent.Quantity,
			"filled_quantity", intent.FilledQuantity,
			"status", string(intent.Status),
			"fills_count", len(intent.Fills),
			"venue_order_id", msg.Event.VenueOrderID,
			"timestamp", intent.Timestamp.Format(time.RFC3339),
			"correlation_id", msg.Event.Metadata.CorrelationID,
			"causation_id", msg.Event.Metadata.CausationID,
		)
	}
}

func (a *FillProjectionActor) checkStatsInvariant() {
	received := a.stats.received.Load()
	sum := a.stats.materialized.Load() +
		a.stats.skippedStale.Load() +
		a.stats.skippedDedup.Load() +
		a.stats.skippedNonFinal.Load() +
		a.stats.rejected.Load() +
		a.stats.orphaned.Load() +
		a.stats.overflowed.Load() +
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
			"orphaned", a.stats.orphaned.Load(),
			"overflowed", a.stats.overflowed.Load(),
			"errors", a.stats.errors.Load(),
		)
	}
}

func (a *FillProjectionActor) logStats() {
	a.logger.Info("fill projection stats",
		"bucket", a.cfg.Bucket,
		"received", a.stats.received.Load(),
		"materialized", a.stats.materialized.Load(),
		"skipped_stale", a.stats.skippedStale.Load(),
		"skipped_dedup", a.stats.skippedDedup.Load(),
		"skipped_non_final", a.stats.skippedNonFinal.Load(),
		"rejected", a.stats.rejected.Load(),
		"orphaned", a.stats.orphaned.Load(),
		"overflowed", a.stats.overflowed.Load(),
		"errors", a.stats.errors.Load(),
	)
}
