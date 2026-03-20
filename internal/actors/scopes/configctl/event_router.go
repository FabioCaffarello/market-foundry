package configctl

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	natsconfigctl "internal/adapters/nats/natsconfigctl"

	"github.com/anthdm/hollywood/actor"
)

type EventRouterConfig struct {
	URL      string
	Source   string
	Registry natsconfigctl.Registry
}

type EventRouterActor struct {
	cfg       EventRouterConfig
	logger    *slog.Logger
	publisher *natsconfigctl.EventPublisher
}

func NewEventRouterActor(cfg EventRouterConfig) actor.Producer {
	return func() actor.Receiver {
		return &EventRouterActor{cfg: cfg}
	}
}

func (a *EventRouterActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default()
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		publisher := natsconfigctl.NewEventPublisher(a.cfg.URL, a.cfg.Source, a.cfg.Registry)
		if err := publisher.Start(); err != nil {
			a.logger.Error("start domain event publisher", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.publisher = publisher
	case actor.Stopped:
		if a.publisher != nil {
			if err := a.publisher.Close(); err != nil {
				a.logger.Error("close domain event publisher", "error", err)
			}
		}
	case publishDomainEventMessage:
		reply := publishDomainEventResult{}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		reply.Prob = a.publisher.Publish(ctx, msg.Event)
		cancel()
		if sender := c.Sender(); sender != nil {
			c.Send(sender, reply)
		}
	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("configctl event router: unknown message", "type", fmt.Sprintf("%T", msg))
	}
}
