package derive

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	appingest "internal/application/ingest"
	"internal/application/configctl/contracts"
	configdomain "internal/domain/configctl"
	"internal/application/ports"

	"github.com/anthdm/hollywood/actor"
)

// BindingWatcherConfig holds the configuration for the derive binding watcher actor.
type BindingWatcherConfig struct {
	NATSURL        string
	Gateway        ports.ConfigctlGateway
	SupervisorPID  *actor.PID
	RequestTimeout time.Duration
}

// BindingWatcherActor queries configctl on startup for active ingestion bindings,
// then subscribes to IngestionRuntimeChangedEvent for dynamic updates.
// It sends activateSamplerMessage to the derive supervisor when new bindings appear.
// This replaces the supervisor's startup-only queryAndActivateBindings approach,
// enabling runtime activation without process restart.
type BindingWatcherActor struct {
	cfg      BindingWatcherConfig
	logger   *slog.Logger
	consumer *adapternats.BindingEventConsumer
}

func NewBindingWatcherActor(cfg BindingWatcherConfig) actor.Producer {
	return func() actor.Receiver {
		return &BindingWatcherActor{cfg: cfg}
	}
}

func (a *BindingWatcherActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "derive-binding-watcher")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		if a.consumer != nil {
			if err := a.consumer.Close(); err != nil {
				a.logger.Error("close binding event consumer", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *BindingWatcherActor) start(c *actor.Context) {
	// Step 1: Query configctl for currently active bindings.
	a.queryActiveBindings(c)

	// Step 2: Subscribe to runtime change events for dynamic updates.
	a.subscribeToChanges(c)
}

func (a *BindingWatcherActor) queryActiveBindings(c *actor.Context) {
	if a.cfg.Gateway == nil {
		a.logger.Warn("configctl gateway unavailable, skipping initial binding query")
		return
	}

	timeout := a.cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	reply, prob := a.cfg.Gateway.ListActiveIngestionBindings(ctx, contracts.ListActiveIngestionBindingsQuery{})
	if prob != nil {
		a.logger.Error("query active bindings", "error", prob.Message)
		return
	}

	activated := 0
	for _, record := range reply.Bindings {
		target, prob := appingest.ParseBindingTopic(record.Binding.Topic)
		if prob != nil {
			a.logger.Warn("skip invalid binding topic",
				"topic", record.Binding.Topic,
				"name", record.Binding.Name,
				"error", prob.Message,
			)
			continue
		}

		c.Send(a.cfg.SupervisorPID, activateSamplerMessage{Target: target})
		activated++
	}

	a.logger.Info("initial binding query complete",
		"total_bindings", len(reply.Bindings),
		"activated", activated,
	)
}

func (a *BindingWatcherActor) subscribeToChanges(c *actor.Context) {
	registry := adapternats.DefaultConfigctlRegistry()
	spec := adapternats.DeriveBindingConsumer()

	consumer := adapternats.NewBindingEventConsumer(
		a.cfg.NATSURL,
		spec,
		registry,
		func(event configdomain.IngestionRuntimeChangedEvent) {
			a.onRuntimeChanged(c, event)
		},
		a.logger,
	)

	if err := consumer.Start(); err != nil {
		a.logger.Error("start binding event consumer", "error", err)
		// Non-fatal: the initial query already loaded active bindings.
		// Dynamic updates won't work, but the service can operate with the startup state.
		return
	}

	a.consumer = consumer
	a.logger.Info("binding event consumer started",
		"durable", spec.Durable,
		"subject", spec.Event.Subject,
	)
}

func (a *BindingWatcherActor) onRuntimeChanged(c *actor.Context, event configdomain.IngestionRuntimeChangedEvent) {
	switch event.ChangeType {
	case configdomain.IngestionRuntimeChangeActivated:
		if event.Runtime == nil {
			a.logger.Warn("activated event with nil runtime", "scope", event.Scope.String())
			return
		}
		for _, binding := range event.Runtime.Bindings {
			target, prob := appingest.ParseBindingTopic(binding.Topic)
			if prob != nil {
				a.logger.Warn("skip invalid binding topic in event",
					"topic", binding.Topic,
					"error", prob.Message,
				)
				continue
			}
			c.Send(a.cfg.SupervisorPID, activateSamplerMessage{Target: target})
			a.logger.Info("sampler activation triggered via event",
				"source", target.Source,
				"symbol", target.Symbol,
				"scope", event.Scope.String(),
			)
		}

	case configdomain.IngestionRuntimeChangeCleared:
		// When a scope is cleared, we'd need to know which bindings were associated.
		// For now, log the event. Full reconciliation requires tracking scope → bindings.
		a.logger.Info("binding scope cleared via event",
			"scope", event.Scope.String(),
		)

	default:
		a.logger.Warn("unknown binding change type", "type", event.ChangeType)
	}
}
