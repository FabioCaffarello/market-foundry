package store

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	domainexec "internal/domain/execution"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// FillConsumerConfig holds the configuration for the fill consumer actor.
type FillConsumerConfig struct {
	URL           string
	ConsumerSpec  adapternats.ConsumerSpec
	Registry      adapternats.ExecutionRegistry
	ProjectionPID *actor.PID
	Tracker       *healthz.Tracker
}

// FillConsumerActor owns a durable JetStream consumer for venue order fill events.
type FillConsumerActor struct {
	cfg      FillConsumerConfig
	logger   *slog.Logger
	consumer *adapternats.FillConsumer
}

func NewFillConsumerActor(cfg FillConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &FillConsumerActor{cfg: cfg}
	}
}

func (a *FillConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "fill-consumer", "durable", a.cfg.ConsumerSpec.Durable)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		projPID := a.cfg.ProjectionPID
		tracker := a.cfg.Tracker
		consumer := adapternats.NewFillConsumer(
			a.cfg.URL,
			a.cfg.ConsumerSpec,
			a.cfg.Registry,
			func(event domainexec.VenueOrderFilledEvent) {
				if tracker != nil {
					tracker.RecordEvent()
				}
				c.Send(projPID, fillReceivedMessage{Event: event})
			},
			a.logger,
		)
		if err := consumer.Start(); err != nil {
			a.logger.Error("start fill consumer", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.consumer = consumer
		a.logger.Info("fill consumer started",
			"durable", a.cfg.ConsumerSpec.Durable,
			"subject", a.cfg.ConsumerSpec.Event.Subject,
		)

	case actor.Stopped:
		if a.consumer != nil {
			if err := a.consumer.Close(); err != nil {
				a.logger.Error("close fill consumer", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}
