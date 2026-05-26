package store

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	actorcommon "internal/actors/common"
	natsevidence "internal/adapters/nats/natsevidence"
	natskit "internal/adapters/nats/natskit"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// CandleProjectionConfig holds the configuration for the candle projection actor.
type CandleProjectionConfig struct {
	NATSURL string
	Tracker *healthz.Tracker
}

// projectionStats tracks projection outcomes for observability.
// All fields are safe for concurrent access via atomic operations.
type projectionStats struct {
	received        atomic.Int64 // total candles received
	materialized    atomic.Int64 // candles written to both latest + history
	skippedStale    atomic.Int64 // latest skipped: existing candle is newer
	skippedDedup    atomic.Int64 // latest skipped: same OpenTime already exists
	skippedNonFinal atomic.Int64 // non-final candles dropped
	rejected        atomic.Int64 // candles rejected by validation
	errors          atomic.Int64 // write errors
}

// CandleProjectionActor materializes finalized candles into NATS KV.
//
// Invariants enforced:
//   - Only candles with Final=true are materialized.
//   - Candle domain validation must pass before any write.
//   - Latest bucket has a monotonicity guard: stale/duplicate candles are skipped.
//   - History bucket is idempotent by key design (open_time in key).
//   - Both buckets are written atomically per candle (latest first, then history).
type CandleProjectionActor struct {
	cfg    CandleProjectionConfig
	logger *slog.Logger
	store  candleProjectionStore
	closer func() error // closes the underlying KV connection; nil in tests
	stats  projectionStats
}

func NewCandleProjectionActor(cfg CandleProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &CandleProjectionActor{cfg: cfg}
	}
}

func (a *CandleProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "candle-projection")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.checkStatsInvariant()
		a.logStats()
		if a.closer != nil {
			if err := a.closer(); err != nil {
				a.logger.Error("close candle KV store", "error", err)
			}
		}

	case candleReceivedMessage:
		a.onCandle(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *CandleProjectionActor) start(c *actor.Context) {
	store := natsevidence.NewCandleKVStore(a.cfg.NATSURL)
	if err := store.Start(); err != nil {
		a.logger.Error("start candle KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.store = store
	a.closer = store.Close
	a.logger.Info("candle projection started",
		"bucket_latest", natsevidence.CandleLatestBucket,
		"bucket_history", natsevidence.CandleHistoryBucket,
	)
}

func (a *CandleProjectionActor) onCandle(msg candleReceivedMessage) {
	a.stats.received.Add(1)
	candle := msg.Event.Candle

	// Gate 1: Only materialize finalized candles.
	if !candle.Final {
		a.stats.skippedNonFinal.Add(1)
		return
	}

	// Gate 2: Domain validation — reject malformed candles before any write.
	if prob := candle.Validate(); prob != nil {
		a.stats.rejected.Add(1)
		a.logger.Warn("candle rejected by validation",
			"error", prob.Message,
			"source", candle.Source,
			"symbol", candle.VenueSymbol(),
			"timeframe", candle.Timeframe,
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Write latest (with monotonicity guard).
	result, prob := a.store.Put(ctx, candle)
	if prob != nil {
		a.stats.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("materialize candle latest",
			"error", prob.Message,
			"source", candle.Source,
			"symbol", candle.VenueSymbol(),
			"timeframe", candle.Timeframe,
		)
		return
	}

	switch result {
	case natskit.PutSkippedStale:
		a.stats.skippedStale.Add(1)
		a.logger.Debug("latest skipped: existing is newer",
			"source", candle.Source,
			"symbol", candle.VenueSymbol(),
			"timeframe", candle.Timeframe,
			"open_time", candle.OpenTime.Format(time.RFC3339),
		)
		return
	case natskit.PutSkippedDuplicate:
		a.stats.skippedDedup.Add(1)
		a.logger.Debug("latest skipped: duplicate open_time",
			"source", candle.Source,
			"symbol", candle.VenueSymbol(),
			"timeframe", candle.Timeframe,
			"open_time", candle.OpenTime.Format(time.RFC3339),
		)
		// Still write to history — the key-based dedup handles it there.
	}

	// Write history (idempotent by key design).
	if prob := a.store.PutHistory(ctx, candle); prob != nil {
		a.stats.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("materialize candle history",
			"error", prob.Message,
			"source", candle.Source,
			"symbol", candle.VenueSymbol(),
			"timeframe", candle.Timeframe,
		)
	}

	if result == natskit.PutWritten {
		a.stats.materialized.Add(1)
	}

	if a.cfg.Tracker != nil {
		a.cfg.Tracker.RecordEvent()
		a.cfg.Tracker.Counter("materialized:" + candle.VenueSymbol()).Add(1)
	}

	if result == natskit.PutWritten {
		a.logger.Info("candle materialized",
			"source", candle.Source,
			"symbol", candle.VenueSymbol(),
			"timeframe", candle.Timeframe,
			"open_time", candle.OpenTime.Format(time.RFC3339),
			"trades", candle.TradeCount,
		)
	}
}

func (a *CandleProjectionActor) checkStatsInvariant() {
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

func (a *CandleProjectionActor) logStats() {
	a.logger.Info("candle projection stats",
		"received", a.stats.received.Load(),
		"materialized", a.stats.materialized.Load(),
		"skipped_stale", a.stats.skippedStale.Load(),
		"skipped_dedup", a.stats.skippedDedup.Load(),
		"skipped_non_final", a.stats.skippedNonFinal.Load(),
		"rejected", a.stats.rejected.Load(),
		"errors", a.stats.errors.Load(),
	)
}
