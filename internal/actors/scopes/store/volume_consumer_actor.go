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

type VolumeConsumerConfig struct {
	URL           string
	ConsumerSpec  adapternats.ConsumerSpec
	Registry      adapternats.EvidenceRegistry
	ProjectionPID *actor.PID
	Tracker       *healthz.Tracker
}

// VolumeConsumerActor owns the durable JetStream consumer for evidence volume events.
type VolumeConsumerActor struct {
	cfg      VolumeConsumerConfig
	logger   *slog.Logger
	consumer *adapternats.VolumeConsumer
}

func NewVolumeConsumerActor(cfg VolumeConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &VolumeConsumerActor{cfg: cfg}
	}
}

func (a *VolumeConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "volume-consumer")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		if a.consumer != nil {
			if err := a.consumer.Close(); err != nil {
				a.logger.Error("close volume consumer", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *VolumeConsumerActor) start(c *actor.Context) {
	projectionPID := a.cfg.ProjectionPID

	tracker := a.cfg.Tracker
	consumer := adapternats.NewVolumeConsumer(
		a.cfg.URL,
		a.cfg.ConsumerSpec,
		a.cfg.Registry,
		func(event evidence.VolumeSampledEvent) {
			if tracker != nil {
				tracker.RecordEvent()
			}
			c.Send(projectionPID, volumeReceivedMessage{Event: event})
		},
		a.logger,
	)

	if err := consumer.Start(); err != nil {
		a.logger.Error("start volume consumer", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.consumer = consumer
	a.logger.Info("volume consumer started",
		"durable", a.cfg.ConsumerSpec.Durable,
		"filter", a.cfg.ConsumerSpec.Event.Subject,
		"stream", a.cfg.ConsumerSpec.Event.Stream.Name,
	)
}
