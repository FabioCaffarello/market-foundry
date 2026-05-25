package execute

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	natsexecution "internal/adapters/nats/natsexecution"
	natsstrategy "internal/adapters/nats/natsstrategy"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/domain/strategy"
	"internal/shared/clock"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
)

// ExecuteSupervisor is the root actor for the execute binary.
// It consumes execution intents, passes them through the venue adapter, and publishes fills.
type ExecuteSupervisor struct {
	cfg              settings.AppConfig
	venue            ports.VenuePort
	venueQuery       ports.VenueQueryPort // nil when venue has no query capability (e.g. paper)
	trackers         map[string]*healthz.Tracker
	logger           *slog.Logger
	consumer         *natsexecution.Consumer // closed on actor stop
	strategyConsumer *natsstrategy.Consumer  // S360: closed on actor stop
	// S339: Activation surface dimensions passed from binary startup.
	adapterState    domainexec.AdapterState
	credentialState domainexec.CredentialState
	// S460: Session metadata lifecycle.
	sessionStore *natsexecution.SessionKVStore
	session      *domainexec.Session
	operator     string // injected via WithOperator option
	// S490: Publisher for session lifecycle events (verification trigger).
	lifecyclePublisher *natsexecution.Publisher
	// H-4: clk is the time port threaded through to VenueAdapterConfig
	// and (in commit 6d) to Session.Close/Halt. Defaults to
	// clock.SystemClock{} when no WithClock option is supplied.
	clk clock.Clock
}

func NewExecuteSupervisor(config settings.AppConfig, venue ports.VenuePort, venueQuery ports.VenueQueryPort, trackers map[string]*healthz.Tracker, opts ...SupervisorOption) actor.Producer {
	return func() actor.Receiver {
		s := &ExecuteSupervisor{
			cfg:        config,
			venue:      venue,
			venueQuery: venueQuery,
			trackers:   trackers,
			logger:     slog.Default().With("actor", "execute-supervisor"),
			clk:        clock.SystemClock{},
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

// WithOperator sets the operator identity for session metadata.
func WithOperator(operator string) SupervisorOption {
	return func(s *ExecuteSupervisor) {
		s.operator = operator
	}
}

// WithClock sets the time port the supervisor threads through to
// child actor configs. Optional; defaults to clock.SystemClock{}.
// Tests inject deterministic Clock implementations here when
// validating Session lifecycle timing (commit 6d).
func WithClock(clk clock.Clock) SupervisorOption {
	return func(s *ExecuteSupervisor) {
		if clk != nil {
			s.clk = clk
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
		// S460: Close session before tearing down consumers.
		s.closeSession("")
		if s.lifecyclePublisher != nil {
			_ = s.lifecyclePublisher.Close()
		}
		if s.sessionStore != nil {
			_ = s.sessionStore.Close()
		}
		if s.strategyConsumer != nil {
			_ = s.strategyConsumer.Close()
		}
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

	// S401: Build allowed source set from enabled segments for defense-in-depth.
	allowedSources := make(map[string]bool)
	for _, src := range s.cfg.Venue.EnabledSegmentSources() {
		allowedSources[src] = true
	}

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
		AllowedSources:  allowedSources,
		Clock:           s.clk,
	}), "venue-adapter")

	// Spawn the consumer actor for execution intents.
	// S401: When unified segments are configured, the consumer subscribes only to
	// subjects matching enabled segment sources. This prevents cross-segment leakage
	// at the NATS subscription level — the consumer never receives intents for segments
	// without a registered adapter.
	consumerTracker := s.trackers["venue-consumer"]
	enabledSources := s.cfg.Venue.EnabledSegmentSources()
	consumerSpec := natsexecution.ExecuteVenueIntakeConsumerForSegments(enabledSources)

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

	// ── S360: Strategy-to-execution wiring ──────────────────────────
	// Spawn strategy consumer actor that evaluates StrategyResolvedEvent
	// and forwards produced ExecutionIntents to the venue adapter actor.
	strategyTracker := s.trackers["strategy-consumer"]
	strategyActorPID := ctx.SpawnChild(NewStrategyConsumerActor(StrategyConsumerConfig{
		MaxPositionPct: DefaultMaxPositionPct,
		Tracker:        strategyTracker,
		AdapterPID:     adapterPID,
	}), "strategy-consumer")

	strategyConsumerSpec := natsstrategy.ExecuteStrategyMeanReversionEntryConsumer()
	strategyRegistry := natsstrategy.DefaultRegistry()

	stratConsumer := natsstrategy.NewConsumer(
		s.cfg.NATS.URL,
		strategyConsumerSpec,
		strategyRegistry,
		func(event strategy.StrategyResolvedEvent) {
			if strategyTracker != nil {
				strategyTracker.RecordEvent()
			}
			ctx.Send(strategyActorPID, strategyReceivedMessage{Event: event})
		},
		slog.Default().With("actor", "strategy-consumer-nats"),
	)
	if err := stratConsumer.Start(); err != nil {
		return fmt.Errorf("start strategy consumer: %w", err)
	}
	s.strategyConsumer = stratConsumer

	// S490: Start session lifecycle publisher for verification triggers.
	lifecyclePub := natsexecution.NewPublisher(s.cfg.NATS.URL, "execute.supervisor", execRegistry)
	if err := lifecyclePub.Start(); err != nil {
		s.logger.Warn("session lifecycle publisher unavailable — event-driven verification degraded", "error", err)
	} else {
		s.lifecyclePublisher = lifecyclePub
	}

	// S460: Open session metadata record.
	s.openSession()

	s.logger.Info("execute supervisor started",
		"consumer_durable", consumerSpec.Durable,
		"consumer_subject", consumerSpec.Event.Subject,
		"strategy_consumer_durable", strategyConsumerSpec.Durable,
		"strategy_consumer_subject", strategyConsumerSpec.Event.Subject,
		"venue_type", string(s.cfg.Venue.Type),
		"staleness_max_age", stalenessMaxAge.String(),
		"submit_timeout", submitTimeout.String(),
		"control_gate", "EXECUTION_CONTROL",
		"session_id", s.sessionID(),
	)
	return nil
}

func (s *ExecuteSupervisor) sessionID() string {
	if s.session != nil {
		return s.session.SessionID
	}
	return ""
}

func (s *ExecuteSupervisor) openSession() {
	now := time.Now().UTC()
	session := domainexec.Session{
		SessionID: domainexec.NewSessionID(now),
		Operator:  s.operator,
		Status:    domainexec.SessionOpen,
		StartedAt: now,
		Config: domainexec.SessionConfigSnapshot{
			VenueType: string(s.cfg.Venue.Type),
			DryRun:    s.cfg.Venue.DryRun != nil && *s.cfg.Venue.DryRun,
			Segments:  s.cfg.Venue.EnabledSegmentSources(),
		},
		Activation: domainexec.SessionActivationSnapshot{
			Adapter:     s.adapterState,
			Credentials: s.credentialState,
			GateStatus:  domainexec.GateActive, // gate is active at startup
			Effective:   domainexec.ComputeEffectiveMode(s.adapterState, domainexec.GateActive, s.credentialState),
		},
	}

	store := natsexecution.NewSessionKVStore(s.cfg.NATS.URL)
	if err := store.Start(); err != nil {
		s.logger.Warn("session KV store unavailable — session metadata degraded", "error", err)
		s.session = &session
		return
	}
	s.sessionStore = store

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if prob := store.Put(ctx, session); prob != nil {
		s.logger.Warn("failed to persist session open", "error", prob.Message, "session_id", session.SessionID)
	} else {
		s.logger.Info("session opened", "session_id", session.SessionID)
	}
	s.session = &session
}

func (s *ExecuteSupervisor) closeSession(reason string) {
	if s.session == nil {
		return
	}

	// Collect segment counters from the venue-adapter tracker.
	var counters []domainexec.SessionSegmentCounters
	if tracker := s.trackers["venue-adapter"]; tracker != nil {
		allCounters := tracker.Counters()
		for _, src := range s.cfg.Venue.EnabledSegmentSources() {
			seg := string(settings.SegmentForSource(src))
			counters = append(counters, domainexec.SessionSegmentCounters{
				Segment:   seg,
				Processed: allCounters[seg+":processed"],
				Filled:    allCounters[seg+":filled"],
				Rejected:  allCounters[seg+":rejected"],
				Errors:    allCounters[seg+":errors"],
			})
		}
	}

	if reason != "" {
		if prob := s.session.Halt(s.clk, reason, counters); prob != nil {
			s.logger.Warn("session already terminal, skipping halt", "error", prob.Message, "session_id", s.session.SessionID)
			return
		}
	} else {
		if prob := s.session.Close(s.clk, counters); prob != nil {
			s.logger.Warn("session already terminal, skipping close", "error", prob.Message, "session_id", s.session.SessionID)
			return
		}
	}

	if s.sessionStore != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if prob := s.sessionStore.Put(ctx, *s.session); prob != nil {
			s.logger.Warn("failed to persist session close", "error", prob.Message, "session_id", s.session.SessionID)
		} else {
			s.logger.Info("session closed",
				"session_id", s.session.SessionID,
				"status", string(s.session.Status),
				"duration", s.session.Duration().String(),
			)
		}
	}

	// S490: Publish session lifecycle event for event-driven verification trigger.
	s.publishSessionLifecycle()
}

func (s *ExecuteSupervisor) publishSessionLifecycle() {
	if s.lifecyclePublisher == nil || s.session == nil {
		return
	}

	event := domainexec.SessionLifecycleEvent{
		SessionID:  s.session.SessionID,
		Status:     s.session.Status,
		Operator:   s.session.Operator,
		HaltReason: s.session.HaltReason,
		VenueType:  s.session.Config.VenueType,
		DryRun:     s.session.Config.DryRun,
		Segments:   s.session.Config.Segments,
	}
	if s.session.ClosedAt != nil {
		event.ClosedAt = *s.session.ClosedAt
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if prob := s.lifecyclePublisher.PublishSessionLifecycle(ctx, event); prob != nil {
		s.logger.Warn("failed to publish session lifecycle event — manual verification still available",
			"error", prob.Message,
			"session_id", s.session.SessionID,
			"status", string(s.session.Status),
		)
	} else {
		s.logger.Info("session lifecycle event published",
			"session_id", s.session.SessionID,
			"status", string(s.session.Status),
		)
	}
}
