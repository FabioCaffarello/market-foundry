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

type SignalProjectionConfig struct {
	NATSURL string
	Bucket  string
	Tracker *healthz.Tracker
}

type signalProjectionStats struct {
	materialized    atomic.Int64
	skippedStale    atomic.Int64
	skippedDedup    atomic.Int64
	skippedNonFinal atomic.Int64
	rejected        atomic.Int64
	errors          atomic.Int64
}

// SignalProjectionActor materializes finalized signals into a NATS KV latest bucket.
type SignalProjectionActor struct {
	cfg    SignalProjectionConfig
	logger *slog.Logger
	store  signalProjectionStore
	closer func() error
	stats  signalProjectionStats
}

func NewSignalProjectionActor(cfg SignalProjectionConfig) actor.Producer {
	return func() actor.Receiver {
		return &SignalProjectionActor{cfg: cfg}
	}
}

func (a *SignalProjectionActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "signal-projection", "family", "rsi", "bucket", a.cfg.Bucket)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.logStats()
		if a.closer != nil {
			if err := a.closer(); err != nil {
				a.logger.Error("close signal KV store", "error", err)
			}
		}

	case signalReceivedMessage:
		a.onSignal(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *SignalProjectionActor) start(c *actor.Context) {
	store := adapternats.NewSignalKVStore(a.cfg.NATSURL, a.cfg.Bucket)
	if err := store.Start(); err != nil {
		a.logger.Error("start signal KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.store = store
	a.closer = store.Close
	a.logger.Info("signal projection started",
		"bucket_latest", a.cfg.Bucket,
	)
}

func (a *SignalProjectionActor) onSignal(msg signalReceivedMessage) {
	sig := msg.Event.Signal

	// Gate 1: Only materialize finalized signals.
	if !sig.Final {
		a.stats.skippedNonFinal.Add(1)
		return
	}

	// Gate 2: Domain validation.
	if prob := sig.Validate(); prob != nil {
		a.stats.rejected.Add(1)
		a.logger.Warn("signal rejected by validation",
			"error", prob.Message,
			"type", sig.Type,
			"source", sig.Source,
			"symbol", sig.Symbol,
			"timeframe", sig.Timeframe,
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, prob := a.store.Put(ctx, sig)
	if prob != nil {
		a.stats.errors.Add(1)
		a.logger.Error("materialize signal latest",
			"error", prob.Message,
			"type", sig.Type,
			"source", sig.Source,
			"symbol", sig.Symbol,
			"timeframe", sig.Timeframe,
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
		a.logger.Info("signal materialized",
			"type", sig.Type,
			"source", sig.Source,
			"symbol", sig.Symbol,
			"timeframe", sig.Timeframe,
			"value", sig.Value,
			"timestamp", sig.Timestamp.Format(time.RFC3339),
		)
	}
}

func (a *SignalProjectionActor) logStats() {
	a.logger.Info("signal projection stats",
		"bucket", a.cfg.Bucket,
		"materialized", a.stats.materialized.Load(),
		"skipped_stale", a.stats.skippedStale.Load(),
		"skipped_dedup", a.stats.skippedDedup.Load(),
		"skipped_non_final", a.stats.skippedNonFinal.Load(),
		"rejected", a.stats.rejected.Load(),
		"errors", a.stats.errors.Load(),
	)
}
