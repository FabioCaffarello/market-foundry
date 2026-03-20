package ingest

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	natsobservation "internal/adapters/nats/natsobservation"
	"internal/application/ports"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
)

// IngestSupervisor is the root actor for the ingest binary.
// It delegates per-exchange lifecycle to ExchangeScopeActor children.
// Each exchange scope owns its own publisher and WebSocket adapters.
type IngestSupervisor struct {
	cfg              settings.AppConfig
	gateway          ports.ConfigctlGateway
	logger           *slog.Logger
	registry         natsobservation.Registry
	exchanges        map[string]*actor.PID // key: source → ExchangeScopeActor PID
	publisherTracker *healthz.Tracker
}

func NewIngestSupervisor(config settings.AppConfig, gateway ports.ConfigctlGateway, publisherTracker *healthz.Tracker) actor.Producer {
	return func() actor.Receiver {
		return &IngestSupervisor{
			cfg:              config,
			gateway:          gateway,
			logger:           slog.Default().With("actor", "ingest-supervisor"),
			exchanges:        make(map[string]*actor.PID),
			publisherTracker: publisherTracker,
		}
	}
}

func (s *IngestSupervisor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		if err := s.start(c); err != nil {
			s.logger.Error("start ingest supervisor", "error", err)
			c.Engine().Poison(c.PID())
		}

	case actor.Stopped:
		s.logger.Info("ingest supervisor stopped", "active_exchanges", len(s.exchanges))

	case activateBindingMessage:
		s.onActivateBinding(c, msg)

	case clearBindingMessage:
		s.onClearBinding(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		s.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (s *IngestSupervisor) start(ctx *actor.Context) error {
	if !s.cfg.NATS.Enabled {
		return fmt.Errorf("nats must be enabled for ingest")
	}

	s.registry = natsobservation.DefaultRegistry()

	// Spawn the binding watcher to discover and react to active bindings.
	ctx.SpawnChild(NewBindingWatcherActor(BindingWatcherConfig{
		NATSURL:        s.cfg.NATS.URL,
		Gateway:        s.gateway,
		SupervisorPID:  ctx.PID(),
		RequestTimeout: s.cfg.NATS.RequestTimeoutDuration(),
	}), "binding-watcher")

	s.logger.Info("ingest supervisor started",
		"stream", s.registry.TradeReceived.Stream.Name,
	)
	return nil
}

func (s *IngestSupervisor) ensureExchangeScope(c *actor.Context, source string) *actor.PID {
	pid, exists := s.exchanges[source]
	if exists {
		return pid
	}

	childName := "source-" + source
	pid = c.SpawnChild(NewExchangeScopeActor(ExchangeScopeConfig{
		Source:           source,
		NATSURL:          s.cfg.NATS.URL,
		Registry:         s.registry,
		PublisherTracker: s.publisherTracker,
	}), childName)

	s.exchanges[source] = pid
	s.logger.Info("exchange scope created",
		"source", source,
		"active_exchanges", len(s.exchanges),
	)
	return pid
}

func (s *IngestSupervisor) onActivateBinding(c *actor.Context, msg activateBindingMessage) {
	scope := s.ensureExchangeScope(c, msg.Target.Source)
	c.Send(scope, msg)
}

func (s *IngestSupervisor) onClearBinding(c *actor.Context, msg clearBindingMessage) {
	pid, exists := s.exchanges[msg.Target.Source]
	if !exists {
		s.logger.Info("no exchange scope for source, nothing to clear",
			"source", msg.Target.Source,
		)
		return
	}
	c.Send(pid, msg)
}
