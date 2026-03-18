package store

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/domain/strategy"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

type StrategyConsumerConfig struct {
	URL           string
	ConsumerSpec  adapternats.ConsumerSpec
	Registry      adapternats.StrategyRegistry
	ProjectionPID *actor.PID
	Tracker       *healthz.Tracker
}

// StrategyConsumerActor owns the durable JetStream consumer for strategy events.
type StrategyConsumerActor struct {
	cfg      StrategyConsumerConfig
	logger   *slog.Logger
	consumer *adapternats.StrategyConsumer
}

func NewStrategyConsumerActor(cfg StrategyConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &StrategyConsumerActor{cfg: cfg}
	}
}

func (a *StrategyConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "strategy-consumer", "family", "mean_reversion_entry")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		if a.consumer != nil {
			if err := a.consumer.Close(); err != nil {
				a.logger.Error("close strategy consumer", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *StrategyConsumerActor) start(c *actor.Context) {
	projectionPID := a.cfg.ProjectionPID

	tracker := a.cfg.Tracker
	consumer := adapternats.NewStrategyConsumer(
		a.cfg.URL,
		a.cfg.ConsumerSpec,
		a.cfg.Registry,
		func(event strategy.StrategyResolvedEvent) {
			if tracker != nil {
				tracker.RecordEvent()
			}
			c.Send(projectionPID, strategyReceivedMessage{Event: event})
		},
		a.logger,
	)

	if err := consumer.Start(); err != nil {
		a.logger.Error("start strategy consumer", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.consumer = consumer
	a.logger.Info("strategy consumer started",
		"durable", a.cfg.ConsumerSpec.Durable,
		"filter", a.cfg.ConsumerSpec.Event.Subject,
		"stream", a.cfg.ConsumerSpec.Event.Stream.Name,
	)
}
