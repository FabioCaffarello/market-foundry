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

type TPOProjectionConfig struct {
	NATSURL string
	Tracker *healthz.Tracker
}

type tpoProjectionStats struct {
	received        atomic.Int64
	materialized    atomic.Int64
	skippedStale    atomic.Int64
	skippedDedup    atomic.Int64
	skippedNonFinal atomic.Int64
	rejected        atomic.Int64
	errors          atomic.Int64
}

// TPOProjectionActor materializes finalized TPO profiles into NATS KV
// (INSIGHTS_TPO_LATEST). PROGRAM-0005 / H-8.b.
type TPOProjectionActor struct {
	cfg    TPOProjectionConfig
	logger *slog.Logger
	store  tpoProjectionStore
	closer func() error
	stats  tpoProjectionStats
}

func NewTPOProjectionActor(cfg TPOProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &TPOProjectionActor{cfg: cfg}
	}
}

func (a *TPOProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "tpo-projection")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.checkStatsInvariant()
		a.logStats()
		if a.closer != nil {
			if err := a.closer(); err != nil {
				a.logger.Error("close tpo KV store", "error", err)
			}
		}

	case tpoProfileReceivedMessage:
		a.onTPOProfile(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *TPOProjectionActor) start(c *actor.Context) {
	store := natsinsights.NewTPOKVStore(a.cfg.NATSURL)
	if err := store.Start(); err != nil {
		a.logger.Error("start tpo KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}
	a.store = store
	a.closer = store.Close
	a.logger.Info("tpo projection started", "bucket_latest", natsinsights.TPOLatestBucket)
}

func (a *TPOProjectionActor) onTPOProfile(msg tpoProfileReceivedMessage) {
	a.stats.received.Add(1)
	tp := msg.Event.TPOProfile

	if !tp.Final {
		a.stats.skippedNonFinal.Add(1)
		return
	}
	if prob := tp.Validate(); prob != nil {
		a.stats.rejected.Add(1)
		a.logger.Warn("tpo profile rejected by validation",
			"error", prob.Message, "source", tp.Source, "symbol", tp.VenueSymbol(), "timeframe", tp.Timeframe,
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, prob := a.store.Put(ctx, tp)
	if prob != nil {
		a.stats.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("materialize tpo latest",
			"error", prob.Message, "source", tp.Source, "symbol", tp.VenueSymbol(), "timeframe", tp.Timeframe,
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
			a.cfg.Tracker.Counter("materialized:tpo:" + tp.VenueSymbol()).Add(1)
		}
		a.logger.Info("tpo profile materialized",
			"source", tp.Source, "symbol", tp.VenueSymbol(), "timeframe", tp.Timeframe,
			"open_time", tp.OpenTime.Format(time.RFC3339), "periods", len(tp.Periods), "levels", len(tp.Levels), "overload", int(tp.Overload),
		)
	}
}

func (a *TPOProjectionActor) checkStatsInvariant() {
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

func (a *TPOProjectionActor) logStats() {
	a.logger.Info("tpo projection stats",
		"received", a.stats.received.Load(),
		"materialized", a.stats.materialized.Load(),
		"skipped_stale", a.stats.skippedStale.Load(),
		"skipped_dedup", a.stats.skippedDedup.Load(),
		"skipped_non_final", a.stats.skippedNonFinal.Load(),
		"rejected", a.stats.rejected.Load(),
		"errors", a.stats.errors.Load(),
	)
}
