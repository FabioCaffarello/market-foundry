package store

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/domain/signal"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

type SignalConsumerConfig struct {
	URL           string
	ConsumerSpec  adapternats.ConsumerSpec
	Registry      adapternats.SignalRegistry
	ProjectionPID *actor.PID
	Tracker       *healthz.Tracker
}

// SignalConsumerActor owns the durable JetStream consumer for signal events.
type SignalConsumerActor struct {
	cfg      SignalConsumerConfig
	logger   *slog.Logger
	consumer *adapternats.SignalConsumer
}

func NewSignalConsumerActor(cfg SignalConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &SignalConsumerActor{cfg: cfg}
	}
}

func (a *SignalConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "signal-consumer", "family", "rsi")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		if a.consumer != nil {
			if err := a.consumer.Close(); err != nil {
				a.logger.Error("close signal consumer", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *SignalConsumerActor) start(c *actor.Context) {
	projectionPID := a.cfg.ProjectionPID

	tracker := a.cfg.Tracker
	consumer := adapternats.NewSignalConsumer(
		a.cfg.URL,
		a.cfg.ConsumerSpec,
		a.cfg.Registry,
		func(event signal.SignalGeneratedEvent) {
			if tracker != nil {
				tracker.RecordEvent()
			}
			c.Send(projectionPID, signalReceivedMessage{Event: event})
		},
		a.logger,
	)

	if err := consumer.Start(); err != nil {
		a.logger.Error("start signal consumer", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.consumer = consumer
	a.logger.Info("signal consumer started",
		"durable", a.cfg.ConsumerSpec.Durable,
		"filter", a.cfg.ConsumerSpec.Event.Subject,
		"stream", a.cfg.ConsumerSpec.Event.Stream.Name,
	)
}
