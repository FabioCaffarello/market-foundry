package execute

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	natsexecution "internal/adapters/nats/natsexecution"
	domainexec "internal/domain/execution"
	"internal/application/ports"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
)

// ExecuteSupervisor is the root actor for the execute binary.
// It consumes execution intents, passes them through the venue adapter, and publishes fills.
type ExecuteSupervisor struct {
	cfg             settings.AppConfig
	venue           ports.VenuePort
	venueQuery      ports.VenueQueryPort // nil when venue has no query capability (e.g. paper)
	trackers        map[string]*healthz.Tracker
	logger          *slog.Logger
	consumer        *natsexecution.Consumer // closed on actor stop
	// S339: Activation surface dimensions passed from binary startup.
	adapterState    domainexec.AdapterState
	credentialState domainexec.CredentialState
}

func NewExecuteSupervisor(config settings.AppConfig, venue ports.VenuePort, venueQuery ports.VenueQueryPort, trackers map[string]*healthz.Tracker, opts ...SupervisorOption) actor.Producer {
	return func() actor.Receiver {
		s := &ExecuteSupervisor{
			cfg:        config,
			venue:      venue,
			venueQuery: venueQuery,
			trackers:   trackers,
			logger:     slog.Default().With("actor", "execute-supervisor"),
		}
		for _, opt := range opts {
			opt(s)
		}
		return s
	}
}

// SupervisorOption configures optional parameters on the ExecuteSupervisor.
type SupervisorOption func(*ExecuteSupervisor)

// WithActivationState sets the activation surface dimensions for canonical state reporting.
func WithActivationState(adapter domainexec.AdapterState, creds domainexec.CredentialState) SupervisorOption {
	return func(s *ExecuteSupervisor) {
		s.adapterState = adapter
		s.credentialState = creds
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
		if s.consumer != nil {
			_ = s.consumer.Close()
		}
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

	execRegistry := natsexecution.DefaultRegistry()
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
		VenueQuery:      s.venueQuery,
		StalenessMaxAge: stalenessMaxAge,
		SubmitTimeout:   submitTimeout,
		Tracker:         adapterTracker,
		AdapterState:    s.adapterState,
		CredentialState: s.credentialState,
	}), "venue-adapter")

	// Spawn the consumer actor for execution intents.
	// TRANSITIONAL BRIDGE: In paper mode, the venue consumer subscribes to paper_order
	// subjects because derive only produces PaperOrderSubmittedEvent. When venue-specific
	// intent subjects are introduced, this consumer's spec will migrate accordingly.
	consumerTracker := s.trackers["venue-consumer"]
	consumerSpec := natsexecution.ExecuteVenueMarketOrderIntakeConsumer()

	consumer := natsexecution.NewConsumer(
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
	s.consumer = consumer

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
