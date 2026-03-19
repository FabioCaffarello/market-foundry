package execute

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	domainexec "internal/domain/execution"
	"internal/application/ports"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
)

// ExecuteSupervisor is the root actor for the execute binary.
// It consumes execution intents, passes them through the venue adapter, and publishes fills.
type ExecuteSupervisor struct {
	cfg      settings.AppConfig
	venue    ports.VenuePort
	trackers map[string]*healthz.Tracker
	logger   *slog.Logger
}

func NewExecuteSupervisor(config settings.AppConfig, venue ports.VenuePort, trackers map[string]*healthz.Tracker) actor.Producer {
	return func() actor.Receiver {
		return &ExecuteSupervisor{
			cfg:      config,
			venue:    venue,
			trackers: trackers,
			logger:   slog.Default().With("actor", "execute-supervisor"),
		}
	}
}

func (s *ExecuteSupervisor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		if err := s.start(c); err != nil {
			s.logger.Error("start execute supervisor", "error", err)
			c.Engine().Poison(c.PID())
		}

	case actor.Stopped:
		s.logger.Info("execute supervisor stopped")

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		s.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (s *ExecuteSupervisor) start(ctx *actor.Context) error {
	if !s.cfg.NATS.Enabled {
		return fmt.Errorf("nats must be enabled for execute")
	}

	execRegistry := adapternats.DefaultExecutionRegistry()
	adapterTracker := s.trackers["venue-adapter"]

	// Staleness and submit timeout are config-driven (PRE-A5, PRE-A6).
	stalenessMaxAge := s.cfg.Venue.StalenessMaxAgeDuration()
	submitTimeout := s.cfg.Venue.SubmitTimeoutDuration()

	// Spawn the venue adapter actor.
	adapterPID := ctx.SpawnChild(NewVenueAdapterActor(VenueAdapterConfig{
		NATSURL:         s.cfg.NATS.URL,
		Source:          "execute.venue-adapter",
		Registry:        execRegistry,
		Venue:           s.venue,
		StalenessMaxAge: stalenessMaxAge,
		SubmitTimeout:   submitTimeout,
		Tracker:         adapterTracker,
	}), "venue-adapter")

	// Spawn the consumer actor for execution intents.
	// TRANSITIONAL BRIDGE: In paper mode, the venue consumer subscribes to paper_order
	// subjects because derive only produces PaperOrderSubmittedEvent. When venue-specific
	// intent subjects are introduced, this consumer's spec will migrate accordingly.
	consumerTracker := s.trackers["venue-consumer"]
	consumerSpec := adapternats.ExecuteVenueMarketOrderIntakeConsumer()

	consumer := adapternats.NewExecutionConsumer(
		s.cfg.NATS.URL,
		consumerSpec,
		execRegistry,
		func(event domainexec.PaperOrderSubmittedEvent) {
			if consumerTracker != nil {
				consumerTracker.RecordEvent()
			}
			ctx.Send(adapterPID, intentReceivedMessage{Event: event})
		},
		slog.Default().With("actor", "venue-consumer"),
	)
	if err := consumer.Start(); err != nil {
		return fmt.Errorf("start venue consumer: %w", err)
	}

	s.logger.Info("execute supervisor started",
		"consumer_durable", consumerSpec.Durable,
		"consumer_subject", consumerSpec.Event.Subject,
		"venue_type", string(s.cfg.Venue.Type),
		"staleness_max_age", stalenessMaxAge.String(),
		"submit_timeout", submitTimeout.String(),
		"control_gate", "EXECUTION_CONTROL",
	)
	return nil
}
