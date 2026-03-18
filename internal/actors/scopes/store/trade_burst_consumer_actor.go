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

// TradeBurstConsumerConfig holds the configuration for the trade burst consumer actor.
type TradeBurstConsumerConfig struct {
	URL           string
	ConsumerSpec  adapternats.ConsumerSpec
	Registry      adapternats.EvidenceRegistry
	ProjectionPID *actor.PID
	Tracker       *healthz.Tracker
}

// TradeBurstConsumerActor owns the durable JetStream consumer for evidence trade burst events.
// It decodes each event and forwards it to the projection actor.
type TradeBurstConsumerActor struct {
	cfg      TradeBurstConsumerConfig
	logger   *slog.Logger
	consumer *adapternats.TradeBurstConsumer
}

func NewTradeBurstConsumerActor(cfg TradeBurstConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &TradeBurstConsumerActor{cfg: cfg}
	}
}

func (a *TradeBurstConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "trade-burst-consumer")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		if a.consumer != nil {
			if err := a.consumer.Close(); err != nil {
				a.logger.Error("close trade burst consumer", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *TradeBurstConsumerActor) start(c *actor.Context) {
	projectionPID := a.cfg.ProjectionPID

	tracker := a.cfg.Tracker
	consumer := adapternats.NewTradeBurstConsumer(
		a.cfg.URL,
		a.cfg.ConsumerSpec,
		a.cfg.Registry,
		func(event evidence.TradeBurstSampledEvent) {
			if tracker != nil {
				tracker.RecordEvent()
			}
			c.Send(projectionPID, tradeBurstReceivedMessage{Event: event})
		},
		a.logger,
	)

	if err := consumer.Start(); err != nil {
		a.logger.Error("start trade burst consumer", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.consumer = consumer
	a.logger.Info("trade burst consumer started",
		"durable", a.cfg.ConsumerSpec.Durable,
		"filter", a.cfg.ConsumerSpec.Event.Subject,
		"stream", a.cfg.ConsumerSpec.Event.Stream.Name,
	)
}
