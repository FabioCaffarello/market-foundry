package store

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/domain/evidence"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// EvidenceConsumerConfig holds the configuration for the evidence consumer actor.
type EvidenceConsumerConfig struct {
	URL           string
	ConsumerSpec  adapternats.ConsumerSpec
	Registry      adapternats.EvidenceRegistry
	ProjectionPID *actor.PID
	Tracker       *healthz.Tracker
}

// EvidenceConsumerActor owns the durable JetStream consumer for evidence candle events.
// It decodes each event and forwards it to the projection actor.
type EvidenceConsumerActor struct {
	cfg      EvidenceConsumerConfig
	logger   *slog.Logger
	consumer *adapternats.EvidenceConsumer
}

func NewEvidenceConsumerActor(cfg EvidenceConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &EvidenceConsumerActor{cfg: cfg}
	}
}

func (a *EvidenceConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "candle-consumer")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		if a.consumer != nil {
			if err := a.consumer.Close(); err != nil {
				a.logger.Error("close evidence consumer", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *EvidenceConsumerActor) start(c *actor.Context) {
	projectionPID := a.cfg.ProjectionPID

	tracker := a.cfg.Tracker
	consumer := adapternats.NewEvidenceConsumer(
		a.cfg.URL,
		a.cfg.ConsumerSpec,
		a.cfg.Registry,
		func(event evidence.CandleSampledEvent) {
			if tracker != nil {
				tracker.RecordEvent()
			}
			c.Send(projectionPID, candleReceivedMessage{Event: event})
		},
		a.logger,
	)

	if err := consumer.Start(); err != nil {
		a.logger.Error("start evidence consumer", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.consumer = consumer
	a.logger.Info("evidence consumer started",
		"durable", a.cfg.ConsumerSpec.Durable,
		"filter", a.cfg.ConsumerSpec.Event.Subject,
		"stream", a.cfg.ConsumerSpec.Event.Stream.Name,
	)
}
