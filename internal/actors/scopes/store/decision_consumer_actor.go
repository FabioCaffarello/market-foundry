package store

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/domain/decision"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

type DecisionConsumerConfig struct {
	URL           string
	ConsumerSpec  adapternats.ConsumerSpec
	Registry      adapternats.DecisionRegistry
	ProjectionPID *actor.PID
	Tracker       *healthz.Tracker
}

// DecisionConsumerActor owns the durable JetStream consumer for decision events.
type DecisionConsumerActor struct {
	cfg      DecisionConsumerConfig
	logger   *slog.Logger
	consumer *adapternats.DecisionConsumer
}

func NewDecisionConsumerActor(cfg DecisionConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &DecisionConsumerActor{cfg: cfg}
	}
}

func (a *DecisionConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "decision-consumer", "family", "rsi_oversold")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		if a.consumer != nil {
			if err := a.consumer.Close(); err != nil {
				a.logger.Error("close decision consumer", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *DecisionConsumerActor) start(c *actor.Context) {
	projectionPID := a.cfg.ProjectionPID

	tracker := a.cfg.Tracker
	consumer := adapternats.NewDecisionConsumer(
		a.cfg.URL,
		a.cfg.ConsumerSpec,
		a.cfg.Registry,
		func(event decision.DecisionEvaluatedEvent) {
			if tracker != nil {
				tracker.RecordEvent()
			}
			c.Send(projectionPID, decisionReceivedMessage{Event: event})
		},
		a.logger,
	)

	if err := consumer.Start(); err != nil {
		a.logger.Error("start decision consumer", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.consumer = consumer
	a.logger.Info("decision consumer started",
		"durable", a.cfg.ConsumerSpec.Durable,
		"filter", a.cfg.ConsumerSpec.Event.Subject,
		"stream", a.cfg.ConsumerSpec.Event.Stream.Name,
	)
}
