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

type VolumeProjectionConfig struct {
	NATSURL string
	Tracker *healthz.Tracker
}

type volumeProjectionStats struct {
	received        atomic.Int64
	materialized    atomic.Int64
	skippedStale    atomic.Int64
	skippedDedup    atomic.Int64
	skippedNonFinal atomic.Int64
	rejected        atomic.Int64
	errors          atomic.Int64
}

// VolumeProjectionActor materializes finalized volume profiles into NATS KV.
type VolumeProjectionActor struct {
	cfg    VolumeProjectionConfig
	logger *slog.Logger
	store  volumeProjectionStore
	closer func() error
	stats  volumeProjectionStats
}

func NewVolumeProjectionActor(cfg VolumeProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &VolumeProjectionActor{cfg: cfg}
	}
}

func (a *VolumeProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "volume-projection")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.checkStatsInvariant()
		a.logStats()
		if a.closer != nil {
			if err := a.closer(); err != nil {
				a.logger.Error("close volume KV store", "error", err)
			}
		}

	case volumeReceivedMessage:
		a.onVolume(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *VolumeProjectionActor) start(c *actor.Context) {
	store := natsevidence.NewVolumeKVStore(a.cfg.NATSURL)
	if err := store.Start(); err != nil {
		a.logger.Error("start volume KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.store = store
	a.closer = store.Close
	a.logger.Info("volume projection started",
		"bucket_latest", natsevidence.VolumeLatestBucket,
	)
}

func (a *VolumeProjectionActor) onVolume(msg volumeReceivedMessage) {
	a.stats.received.Add(1)
	vol := msg.Event.Volume

	if !vol.Final {
		a.stats.skippedNonFinal.Add(1)
		return
	}

	if prob := vol.Validate(); prob != nil {
		a.stats.rejected.Add(1)
		a.logger.Warn("volume rejected by validation",
			"error", prob.Message,
			"source", vol.Source,
			"symbol", vol.Symbol,
			"timeframe", vol.Timeframe,
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, prob := a.store.Put(ctx, vol)
	if prob != nil {
		a.stats.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("materialize volume latest",
			"error", prob.Message,
			"source", vol.Source,
			"symbol", vol.Symbol,
			"timeframe", vol.Timeframe,
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
		a.cfg.Tracker.Counter("materialized:" + vol.Symbol).Add(1)
	}

	if result == natskit.PutWritten {
		a.logger.Info("volume materialized",
			"source", vol.Source,
			"symbol", vol.Symbol,
			"timeframe", vol.Timeframe,
			"open_time", vol.OpenTime.Format(time.RFC3339),
			"trades", vol.TradeCount,
			"vwap", vol.VWAP,
		)
	}
}

func (a *VolumeProjectionActor) checkStatsInvariant() {
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

func (a *VolumeProjectionActor) logStats() {
	a.logger.Info("volume projection stats",
		"received", a.stats.received.Load(),
		"materialized", a.stats.materialized.Load(),
		"skipped_stale", a.stats.skippedStale.Load(),
		"skipped_dedup", a.stats.skippedDedup.Load(),
		"skipped_non_final", a.stats.skippedNonFinal.Load(),
		"rejected", a.stats.rejected.Load(),
		"errors", a.stats.errors.Load(),
	)
}
