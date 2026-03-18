package ingest

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

// PublisherConfig holds the configuration for the observation publisher actor.
type PublisherConfig struct {
	URL      string
	Source   string
	Registry adapternats.ObservationRegistry
	Tracker  *healthz.Tracker
}

// PublisherActor owns the NATS JetStream connection for publishing observation events.
type PublisherActor struct {
	cfg       PublisherConfig
	logger    *slog.Logger
	publisher *adapternats.ObservationPublisher
}

func NewPublisherActor(cfg PublisherConfig) actor.Producer {
	return func() actor.Receiver {
		return &PublisherActor{cfg: cfg}
	}
}

func (a *PublisherActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "observation-publisher")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		pub := adapternats.NewObservationPublisher(a.cfg.URL, a.cfg.Source, a.cfg.Registry)
		if err := pub.Start(); err != nil {
			a.logger.Error("start observation publisher", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.publisher = pub
		a.logger.Info("observation publisher started",
			"stream", a.cfg.Registry.TradeReceived.Stream.Name,
		)

	case actor.Stopped:
		if a.publisher != nil {
			if err := a.publisher.Close(); err != nil {
				a.logger.Error("close observation publisher", "error", err)
			}
		}

	case publishTradeMessage:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := a.publisher.PublishTrade(ctx, msg.Event)
		cancel()
		if prob != nil {
			a.logger.Error("publish trade",
				"error", prob.Message,
				"trade_id", msg.Event.Trade.TradeID,
				"source", msg.Event.Trade.Source,
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
