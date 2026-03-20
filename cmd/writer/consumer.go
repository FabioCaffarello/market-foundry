package main

import (
	"fmt"
	"io"
	"log/slog"

	actorcommon "internal/actors/common"
	natskit "internal/adapters/nats/natskit"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// writerConsumerConfig holds the configuration for a writer consumer actor.
type writerConsumerConfig struct {
	family        string
	natsURL       string
	consumerSpec  natskit.ConsumerSpec
	inserterPID   *actor.PID
	tracker       *healthz.Tracker
	startConsumer func(natsURL string, spec natskit.ConsumerSpec, inserterPID *actor.PID, tracker *healthz.Tracker, logger *slog.Logger, actorCtx *actor.Context) (io.Closer, error)
	supervisorPID *actor.PID
}

// writerConsumerActor owns a durable JetStream consumer for a single pipeline family.
// It decodes events via the existing NATS consumer types and forwards rows to the inserter actor.
type writerConsumerActor struct {
	cfg    writerConsumerConfig
	logger *slog.Logger
	closer io.Closer
}

func newWriterConsumerActor(cfg writerConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &writerConsumerActor{cfg: cfg}
	}
}

func (a *writerConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "writer-"+a.cfg.family+"-consumer")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		closer, err := a.cfg.startConsumer(
			a.cfg.natsURL,
			a.cfg.consumerSpec,
			a.cfg.inserterPID,
			a.cfg.tracker,
			a.logger,
			c,
		)
		if err != nil {
			a.logger.Error("start consumer", "family", a.cfg.family, "error", err)
			if a.cfg.supervisorPID != nil {
				c.Send(a.cfg.supervisorPID, pipelineFailedMsg{
					family: a.cfg.family,
					err:    err,
				})
			} else {
				c.Engine().Poison(c.PID())
			}
			return
		}
		a.closer = closer
		a.logger.Info("consumer started",
			"durable", a.cfg.consumerSpec.Durable,
			"filter", a.cfg.consumerSpec.Event.Subject,
			"stream", a.cfg.consumerSpec.Event.Stream.Name,
		)

	case actor.Stopped:
		if a.closer != nil {
			if err := a.closer.Close(); err != nil {
				a.logger.Error("close consumer", "error", err)
			}
		}
		a.logger.Info("consumer stopped", "family", a.cfg.family)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}
