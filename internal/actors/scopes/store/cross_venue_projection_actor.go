package store

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	actorcommon "internal/actors/common"
	natsinsights "internal/adapters/nats/natsinsights"
	natskit "internal/adapters/nats/natskit"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

type CrossVenueProjectionConfig struct {
	NATSURL string
	Tracker *healthz.Tracker
}

type crossVenueProjectionStats struct {
	received        atomic.Int64
	materialized    atomic.Int64
	skippedStale    atomic.Int64
	skippedDedup    atomic.Int64
	skippedNonFinal atomic.Int64
	rejected        atomic.Int64
	errors          atomic.Int64
}

// CrossVenueProjectionActor materializes finalized cross-venue snapshots
// into NATS KV (INSIGHTS_CROSS_VENUE_LATEST). PROGRAM-0005 / H-8.c.
type CrossVenueProjectionActor struct {
	cfg    CrossVenueProjectionConfig
	logger *slog.Logger
	store  crossVenueProjectionStore
	closer func() error
	stats  crossVenueProjectionStats
}

func NewCrossVenueProjectionActor(cfg CrossVenueProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &CrossVenueProjectionActor{cfg: cfg}
	}
}

func (a *CrossVenueProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "cross-venue-projection")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.checkStatsInvariant()
		a.logStats()
		if a.closer != nil {
			if err := a.closer(); err != nil {
				a.logger.Error("close cross venue KV store", "error", err)
			}
		}

	case crossVenueReceivedMessage:
		a.onCrossVenue(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *CrossVenueProjectionActor) start(c *actor.Context) {
	store := natsinsights.NewCrossVenueKVStore(a.cfg.NATSURL)
	if err := store.Start(); err != nil {
		a.logger.Error("start cross venue KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}
	a.store = store
	a.closer = store.Close
	a.logger.Info("cross venue projection started", "bucket_latest", natsinsights.CrossVenueLatestBucket)
}

func (a *CrossVenueProjectionActor) onCrossVenue(msg crossVenueReceivedMessage) {
	a.stats.received.Add(1)
	cv := msg.Event.CrossVenueSnapshot

	if !cv.Final {
		a.stats.skippedNonFinal.Add(1)
		return
	}
	if prob := cv.Validate(); prob != nil {
		a.stats.rejected.Add(1)
		a.logger.Warn("cross venue snapshot rejected by validation",
			"error", prob.Message, "symbol", cv.VenueSymbol(), "timeframe", cv.Timeframe,
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, prob := a.store.Put(ctx, cv)
	if prob != nil {
		a.stats.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("materialize cross venue latest",
			"error", prob.Message, "symbol", cv.VenueSymbol(), "timeframe", cv.Timeframe,
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
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordEvent()
			a.cfg.Tracker.Counter("materialized:crossvenue:" + cv.VenueSymbol()).Add(1)
		}
		a.logger.Info("cross venue snapshot materialized",
			"symbol", cv.VenueSymbol(), "timeframe", cv.Timeframe,
			"open_time", cv.OpenTime.Format(time.RFC3339), "venues", len(cv.Venues),
		)
	}
}

func (a *CrossVenueProjectionActor) checkStatsInvariant() {
	received := a.stats.received.Load()
	sum := a.stats.materialized.Load() +
		a.stats.skippedStale.Load() +
		a.stats.skippedDedup.Load() +
		a.stats.skippedNonFinal.Load() +
		a.stats.rejected.Load() +
		a.stats.errors.Load()
	if received != sum {
		a.logger.Error("stats invariant violated: received != sum of outcomes",
			"received", received, "sum", sum,
		)
	}
}

func (a *CrossVenueProjectionActor) logStats() {
	a.logger.Info("cross venue projection stats",
		"received", a.stats.received.Load(),
		"materialized", a.stats.materialized.Load(),
		"skipped_stale", a.stats.skippedStale.Load(),
		"skipped_dedup", a.stats.skippedDedup.Load(),
		"skipped_non_final", a.stats.skippedNonFinal.Load(),
		"rejected", a.stats.rejected.Load(),
		"errors", a.stats.errors.Load(),
	)
}
