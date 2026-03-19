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

// ExecutionConsumerConfig holds the configuration for the execution consumer actor.
type ExecutionConsumerConfig struct {
	URL           string
	ConsumerSpec  adapternats.ConsumerSpec
	Registry      adapternats.ExecutionRegistry
	ProjectionPID *actor.PID
	Tracker       *healthz.Tracker
}

// ExecutionConsumerActor owns a durable JetStream consumer for execution events.
type ExecutionConsumerActor struct {
	cfg      ExecutionConsumerConfig
	logger   *slog.Logger
	consumer *adapternats.ExecutionConsumer
}

func NewExecutionConsumerActor(cfg ExecutionConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &ExecutionConsumerActor{cfg: cfg}
	}
}

func (a *ExecutionConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "execution-consumer", "durable", a.cfg.ConsumerSpec.Durable)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		projPID := a.cfg.ProjectionPID
		tracker := a.cfg.Tracker
		consumer := adapternats.NewExecutionConsumer(
			a.cfg.URL,
			a.cfg.ConsumerSpec,
			a.cfg.Registry,
			func(event domainexec.PaperOrderSubmittedEvent) {
				if tracker != nil {
					tracker.RecordEvent()
				}
				c.Send(projPID, executionReceivedMessage{Event: event})
			},
			a.logger,
		)
		if err := consumer.Start(); err != nil {
			a.logger.Error("start execution consumer", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.consumer = consumer
		a.logger.Info("execution consumer started",
			"durable", a.cfg.ConsumerSpec.Durable,
			"subject", a.cfg.ConsumerSpec.Event.Subject,
		)

	case actor.Stopped:
		if a.consumer != nil {
			if err := a.consumer.Close(); err != nil {
				a.logger.Error("close execution consumer", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}
