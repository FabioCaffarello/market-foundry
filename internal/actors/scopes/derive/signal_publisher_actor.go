package derive

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// SignalPublisherConfig holds the configuration for the signal publisher actor.
type SignalPublisherConfig struct {
	URL      string
	Source   string
	Registry adapternats.SignalRegistry
	Tracker  *healthz.Tracker
}

// SignalPublisherActor owns the NATS JetStream connection for publishing signal events.
type SignalPublisherActor struct {
	cfg       SignalPublisherConfig
	logger    *slog.Logger
	publisher *adapternats.SignalPublisher
}

func NewSignalPublisherActor(cfg SignalPublisherConfig) actor.Producer {
	return func() actor.Receiver {
		return &SignalPublisherActor{cfg: cfg}
	}
}

func (a *SignalPublisherActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "signal-publisher")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		pub := adapternats.NewSignalPublisher(a.cfg.URL, a.cfg.Source, a.cfg.Registry)
		if err := pub.Start(); err != nil {
			a.logger.Error("start signal publisher", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.publisher = pub
		a.logger.Info("signal publisher started",
			"stream", a.cfg.Registry.RSIGenerated.Stream.Name,
		)

	case actor.Stopped:
		if a.publisher != nil {
			if err := a.publisher.Close(); err != nil {
				a.logger.Error("close signal publisher", "error", err)
			}
		}

	case publishSignalMessage:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := a.publisher.PublishSignal(ctx, msg.Event)
		cancel()
		if prob != nil {
			a.logger.Error("publish signal failed",
				"error", prob.Message,
				"code", prob.Code,
				"type", msg.Event.Signal.Type,
				"source", msg.Event.Signal.Source,
				"symbol", msg.Event.Signal.Symbol,
				"timeframe", msg.Event.Signal.Timeframe,
			)
		} else if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordEvent()
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}
