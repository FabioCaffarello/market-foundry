package derive

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	natsevidence "internal/adapters/nats/natsevidence"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// EvidencePublisherConfig holds the configuration for the evidence publisher actor.
type EvidencePublisherConfig struct {
	URL      string
	Source   string
	Registry natsevidence.Registry
	Tracker  *healthz.Tracker
}

// EvidencePublisherActor owns the NATS JetStream connection for publishing evidence events.
type EvidencePublisherActor struct {
	cfg       EvidencePublisherConfig
	logger    *slog.Logger
	publisher *natsevidence.Publisher
}

func NewEvidencePublisherActor(cfg EvidencePublisherConfig) actor.Producer {
	return func() actor.Receiver {
		return &EvidencePublisherActor{cfg: cfg}
	}
}

func (a *EvidencePublisherActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "evidence-publisher")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		pub := natsevidence.NewPublisher(a.cfg.URL, a.cfg.Source, a.cfg.Registry)
		if err := pub.Start(); err != nil {
			a.logger.Error("start evidence publisher", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.publisher = pub
		a.logger.Info("evidence publisher started",
			"stream", a.cfg.Registry.CandleSampled.Stream.Name,
		)

	case actor.Stopped:
		if a.publisher != nil {
			if err := a.publisher.Close(); err != nil {
				a.logger.Error("close evidence publisher", "error", err)
			}
		}

	case publishCandleMessage:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := a.publisher.PublishCandle(ctx, msg.Event)
		cancel()
		if prob != nil {
			if a.cfg.Tracker != nil {
				a.cfg.Tracker.RecordError()
			}
			a.logger.Error("publish candle failed",
				"error", prob.Message,
				"code", prob.Code,
				"source", msg.Event.Candle.Source,
				"symbol", msg.Event.Candle.Symbol,
				"timeframe", msg.Event.Candle.Timeframe,
				"open_time", msg.Event.Candle.OpenTime.Format(time.RFC3339),
			)
		} else if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordEvent()
			a.cfg.Tracker.Counter("published:" + msg.Event.Candle.Symbol).Add(1)
		}

	case publishTradeBurstMessage:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := a.publisher.PublishTradeBurst(ctx, msg.Event)
		cancel()
		if prob != nil {
			if a.cfg.Tracker != nil {
				a.cfg.Tracker.RecordError()
			}
			a.logger.Error("publish trade burst failed",
				"error", prob.Message,
				"code", prob.Code,
				"source", msg.Event.TradeBurst.Source,
				"symbol", msg.Event.TradeBurst.Symbol,
				"timeframe", msg.Event.TradeBurst.Timeframe,
				"open_time", msg.Event.TradeBurst.OpenTime.Format(time.RFC3339),
			)
		} else if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordEvent()
			a.cfg.Tracker.Counter("published:" + msg.Event.TradeBurst.Symbol).Add(1)
		}

	case publishVolumeMessage:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := a.publisher.PublishVolume(ctx, msg.Event)
		cancel()
		if prob != nil {
			if a.cfg.Tracker != nil {
				a.cfg.Tracker.RecordError()
			}
			a.logger.Error("publish volume failed",
				"error", prob.Message,
				"code", prob.Code,
				"source", msg.Event.Volume.Source,
				"symbol", msg.Event.Volume.Symbol,
				"timeframe", msg.Event.Volume.Timeframe,
				"open_time", msg.Event.Volume.OpenTime.Format(time.RFC3339),
			)
		} else if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordEvent()
			a.cfg.Tracker.Counter("published:" + msg.Event.Volume.Symbol).Add(1)
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}
