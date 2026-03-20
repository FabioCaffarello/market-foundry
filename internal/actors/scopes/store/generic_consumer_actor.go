package store

import (
	"fmt"
	"io"
	"log/slog"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// ConsumerStartFn creates and starts a domain-specific NATS consumer.
// It returns a Closer for shutdown and an error if start fails.
// The registry, event type, and message routing are captured via closure at
// declaration time in store_supervisor.go declarePipelines().
type ConsumerStartFn func(
	url string,
	spec adapternats.ConsumerSpec,
	projPID *actor.PID,
	tracker *healthz.Tracker,
	actorCtx *actor.Context,
	logger *slog.Logger,
) (io.Closer, error)

// GenericConsumerConfig holds the configuration for a generic consumer actor.
type GenericConsumerConfig struct {
	URL           string
	ConsumerSpec  adapternats.ConsumerSpec
	ProjectionPID *actor.PID
	Tracker       *healthz.Tracker
	Family        string
	StartFn       ConsumerStartFn
}

// GenericConsumerActor is a callback-driven consumer actor that delegates
// domain-specific consumer creation to a ConsumerStartFn. This eliminates
// the need for per-family consumer actor types (signal, decision, etc.)
// since the only variance is captured in the StartFn closure.
type GenericConsumerActor struct {
	cfg      GenericConsumerConfig
	logger   *slog.Logger
	consumer io.Closer
}

// NewGenericConsumerActor creates an actor.Producer for a generic consumer.
func NewGenericConsumerActor(cfg GenericConsumerConfig) actor.Producer {
	return func() actor.Receiver {
		return &GenericConsumerActor{cfg: cfg}
	}
}

func (a *GenericConsumerActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", a.cfg.Family+"-consumer", "family", a.cfg.Family)
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		if a.consumer != nil {
			if err := a.consumer.Close(); err != nil {
				a.logger.Error("close consumer", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *GenericConsumerActor) start(c *actor.Context) {
	consumer, err := a.cfg.StartFn(
		a.cfg.URL,
		a.cfg.ConsumerSpec,
		a.cfg.ProjectionPID,
		a.cfg.Tracker,
		c,
		a.logger,
	)
	if err != nil {
		a.logger.Error("start consumer", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.consumer = consumer
	a.logger.Info("consumer started",
		"durable", a.cfg.ConsumerSpec.Durable,
		"filter", a.cfg.ConsumerSpec.Event.Subject,
		"stream", a.cfg.ConsumerSpec.Event.Stream.Name,
	)
}
