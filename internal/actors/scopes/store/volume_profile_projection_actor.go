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

type VolumeProfileProjectionConfig struct {
	NATSURL string
	Tracker *healthz.Tracker
}

type volumeProfileProjectionStats struct {
	received        atomic.Int64
	materialized    atomic.Int64
	skippedStale    atomic.Int64
	skippedDedup    atomic.Int64
	skippedNonFinal atomic.Int64
	rejected        atomic.Int64
	errors          atomic.Int64
}

// VolumeProfileProjectionActor materializes finalized volume profiles
// into NATS KV (INSIGHTS_VOLUME_PROFILE_LATEST). PROGRAM-0005 / H-8.a.
type VolumeProfileProjectionActor struct {
	cfg    VolumeProfileProjectionConfig
	logger *slog.Logger
	store  volumeProfileProjectionStore
	closer func() error
	stats  volumeProfileProjectionStats
}

func NewVolumeProfileProjectionActor(cfg VolumeProfileProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &VolumeProfileProjectionActor{cfg: cfg}
	}
}

func (a *VolumeProfileProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "volume-profile-projection")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.checkStatsInvariant()
		a.logStats()
		if a.closer != nil {
			if err := a.closer(); err != nil {
				a.logger.Error("close volume profile KV store", "error", err)
			}
		}

	case volumeProfileReceivedMessage:
		a.onVolumeProfile(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *VolumeProfileProjectionActor) start(c *actor.Context) {
	store := natsinsights.NewVolumeProfileKVStore(a.cfg.NATSURL)
	if err := store.Start(); err != nil {
		a.logger.Error("start volume profile KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}
	a.store = store
	a.closer = store.Close
	a.logger.Info("volume profile projection started",
		"bucket_latest", natsinsights.VolumeProfileLatestBucket,
	)
}

func (a *VolumeProfileProjectionActor) onVolumeProfile(msg volumeProfileReceivedMessage) {
	a.stats.received.Add(1)
	vp := msg.Event.VolumeProfile

	if !vp.Final {
		a.stats.skippedNonFinal.Add(1)
		return
	}
	if prob := vp.Validate(); prob != nil {
		a.stats.rejected.Add(1)
		a.logger.Warn("volume profile rejected by validation",
			"error", prob.Message, "source", vp.Source, "symbol", vp.VenueSymbol(), "timeframe", vp.Timeframe,
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, prob := a.store.Put(ctx, vp)
	if prob != nil {
		a.stats.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("materialize volume profile latest",
			"error", prob.Message, "source", vp.Source, "symbol", vp.VenueSymbol(), "timeframe", vp.Timeframe,
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
			a.cfg.Tracker.Counter("materialized:" + vp.VenueSymbol()).Add(1)
		}
		a.logger.Info("volume profile materialized",
			"source", vp.Source, "symbol", vp.VenueSymbol(), "timeframe", vp.Timeframe,
			"open_time", vp.OpenTime.Format(time.RFC3339), "buckets", len(vp.Buckets), "overload", int(vp.Overload),
		)
	}
}

func (a *VolumeProfileProjectionActor) checkStatsInvariant() {
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

func (a *VolumeProfileProjectionActor) logStats() {
	a.logger.Info("volume profile projection stats",
		"received", a.stats.received.Load(),
		"materialized", a.stats.materialized.Load(),
		"skipped_stale", a.stats.skippedStale.Load(),
		"skipped_dedup", a.stats.skippedDedup.Load(),
		"skipped_non_final", a.stats.skippedNonFinal.Load(),
		"rejected", a.stats.rejected.Load(),
		"errors", a.stats.errors.Load(),
	)
}
