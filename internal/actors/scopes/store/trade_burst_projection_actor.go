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

// TradeBurstProjectionConfig holds the configuration for the trade burst projection actor.
type TradeBurstProjectionConfig struct {
	NATSURL string
	Tracker *healthz.Tracker
}

// tradeBurstProjectionStats tracks projection outcomes for observability.
// All fields are safe for concurrent access via atomic operations.
type tradeBurstProjectionStats struct {
	received        atomic.Int64 // total bursts received
	materialized    atomic.Int64 // bursts written to latest
	skippedStale    atomic.Int64 // latest skipped: existing burst is newer
	skippedDedup    atomic.Int64 // latest skipped: same OpenTime already exists
	skippedNonFinal atomic.Int64 // non-final bursts dropped
	rejected        atomic.Int64 // bursts rejected by validation
	errors          atomic.Int64 // write errors
}

// TradeBurstProjectionActor materializes finalized trade bursts into NATS KV.
//
// Invariants enforced:
//   - Only bursts with Final=true are materialized.
//   - TradeBurst domain validation must pass before any write.
//   - Latest bucket has a monotonicity guard: stale/duplicate bursts are skipped.
type TradeBurstProjectionActor struct {
	cfg    TradeBurstProjectionConfig
	logger *slog.Logger
	store  tradeBurstProjectionStore
	closer func() error
	stats  tradeBurstProjectionStats
}

func NewTradeBurstProjectionActor(cfg TradeBurstProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &TradeBurstProjectionActor{cfg: cfg}
	}
}

func (a *TradeBurstProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "trade-burst-projection")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.checkStatsInvariant()
		a.logStats()
		if a.closer != nil {
			if err := a.closer(); err != nil {
				a.logger.Error("close trade burst KV store", "error", err)
			}
		}

	case tradeBurstReceivedMessage:
		a.onTradeBurst(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *TradeBurstProjectionActor) start(c *actor.Context) {
	store := adapternats.NewTradeBurstKVStore(a.cfg.NATSURL)
	if err := store.Start(); err != nil {
		a.logger.Error("start trade burst KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.store = store
	a.closer = store.Close
	a.logger.Info("trade burst projection started",
		"bucket_latest", adapternats.TradeBurstLatestBucket,
	)
}

func (a *TradeBurstProjectionActor) onTradeBurst(msg tradeBurstReceivedMessage) {
	a.stats.received.Add(1)
	burst := msg.Event.TradeBurst

	// Gate 1: Only materialize finalized bursts.
	if !burst.Final {
		a.stats.skippedNonFinal.Add(1)
		return
	}

	// Gate 2: Domain validation — reject malformed bursts before any write.
	if prob := burst.Validate(); prob != nil {
		a.stats.rejected.Add(1)
		a.logger.Warn("trade burst rejected by validation",
			"error", prob.Message,
			"source", burst.Source,
			"symbol", burst.Symbol,
			"timeframe", burst.Timeframe,
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Write latest (with monotonicity guard).
	result, prob := a.store.Put(ctx, burst)
	if prob != nil {
		a.stats.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("materialize trade burst latest",
			"error", prob.Message,
			"source", burst.Source,
			"symbol", burst.Symbol,
			"timeframe", burst.Timeframe,
		)
		return
	}

	switch result {
	case adapternats.PutSkippedStale:
		a.stats.skippedStale.Add(1)
		a.logger.Debug("latest skipped: existing is newer",
			"source", burst.Source,
			"symbol", burst.Symbol,
			"timeframe", burst.Timeframe,
			"open_time", burst.OpenTime.Format(time.RFC3339),
		)
		return
	case adapternats.PutSkippedDuplicate:
		a.stats.skippedDedup.Add(1)
		a.logger.Debug("latest skipped: duplicate open_time",
			"source", burst.Source,
			"symbol", burst.Symbol,
			"timeframe", burst.Timeframe,
			"open_time", burst.OpenTime.Format(time.RFC3339),
		)
		return
	}

	if result == adapternats.PutWritten {
		a.stats.materialized.Add(1)
	}

	if a.cfg.Tracker != nil {
		a.cfg.Tracker.RecordEvent()
		a.cfg.Tracker.Counter("materialized:" + burst.Symbol).Add(1)
	}

	if result == adapternats.PutWritten {
		a.logger.Info("trade burst materialized",
			"source", burst.Source,
			"symbol", burst.Symbol,
			"timeframe", burst.Timeframe,
			"open_time", burst.OpenTime.Format(time.RFC3339),
			"trades", burst.TradeCount,
			"burst", burst.Burst,
		)
	}
}

func (a *TradeBurstProjectionActor) checkStatsInvariant() {
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

func (a *TradeBurstProjectionActor) logStats() {
	a.logger.Info("trade burst projection stats",
		"received", a.stats.received.Load(),
		"materialized", a.stats.materialized.Load(),
		"skipped_stale", a.stats.skippedStale.Load(),
		"skipped_dedup", a.stats.skippedDedup.Load(),
		"skipped_non_final", a.stats.skippedNonFinal.Load(),
		"rejected", a.stats.rejected.Load(),
		"errors", a.stats.errors.Load(),
	)
}
