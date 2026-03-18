package derive

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/domain/observation"

	"github.com/anthdm/hollywood/actor"
)

// ConsumerConfig holds the configuration for the observation consumer actor.
type ConsumerConfig struct {
	URL          string
	ConsumerSpec adapternats.ConsumerSpec
	ObsRegistry  adapternats.ObservationRegistry
	SamplerPID   *actor.PID
}

// ConsumerActor owns the JetStream durable consumer for observation events.
// It decodes each event and forwards it to the sampler actor.
type ConsumerActor struct {
	cfg      ConsumerConfig
	logger   *slog.Logger
	consumer *adapternats.ObservationConsumer
}

func NewConsumerActor(cfg ConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &ConsumerActor{cfg: cfg}
	}
}

func (a *ConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "observation-consumer")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		if a.consumer != nil {
			if err := a.consumer.Close(); err != nil {
				a.logger.Error("close observation consumer", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *ConsumerActor) start(c *actor.Context) {
	samplerPID := a.cfg.SamplerPID

	consumer := adapternats.NewObservationConsumer(
		a.cfg.URL,
		a.cfg.ConsumerSpec,
		a.cfg.ObsRegistry,
		func(event observation.TradeReceivedEvent) {
			c.Send(samplerPID, tradeReceivedMessage{Event: event})
		},
		a.logger,
	)

	if err := consumer.Start(); err != nil {
		a.logger.Error("start observation consumer", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.consumer = consumer
	a.logger.Info("observation consumer started",
		"durable", a.cfg.ConsumerSpec.Durable,
		"filter", a.cfg.ConsumerSpec.Event.Subject,
		"stream", a.cfg.ConsumerSpec.Event.Stream.Name,
	)
}
