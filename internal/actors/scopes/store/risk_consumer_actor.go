package store

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/domain/risk"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

type RiskConsumerConfig struct {
	URL           string
	ConsumerSpec  adapternats.ConsumerSpec
	Registry      adapternats.RiskRegistry
	ProjectionPID *actor.PID
	Tracker       *healthz.Tracker
}

// RiskConsumerActor owns the durable JetStream consumer for risk events.
type RiskConsumerActor struct {
	cfg      RiskConsumerConfig
	logger   *slog.Logger
	consumer *adapternats.RiskConsumer
}

func NewRiskConsumerActor(cfg RiskConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &RiskConsumerActor{cfg: cfg}
	}
}

func (a *RiskConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "risk-consumer", "family", "position_exposure")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		if a.consumer != nil {
			if err := a.consumer.Close(); err != nil {
				a.logger.Error("close risk consumer", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *RiskConsumerActor) start(c *actor.Context) {
	projectionPID := a.cfg.ProjectionPID

	tracker := a.cfg.Tracker
	consumer := adapternats.NewRiskConsumer(
		a.cfg.URL,
		a.cfg.ConsumerSpec,
		a.cfg.Registry,
		func(event risk.RiskAssessedEvent) {
			if tracker != nil {
				tracker.RecordEvent()
			}
			c.Send(projectionPID, riskReceivedMessage{Event: event})
		},
		a.logger,
	)

	if err := consumer.Start(); err != nil {
		a.logger.Error("start risk consumer", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.consumer = consumer
	a.logger.Info("risk consumer started",
		"durable", a.cfg.ConsumerSpec.Durable,
		"filter", a.cfg.ConsumerSpec.Event.Subject,
		"stream", a.cfg.ConsumerSpec.Event.Stream.Name,
	)
}
