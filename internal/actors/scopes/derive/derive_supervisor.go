package derive

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/application/ports"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
)

// DeriveSupervisor is the root actor for the derive binary.
// It delegates per-source lifecycle to SourceScopeActor children.
// Each source scope owns its own evidence publisher and family processor actors.
// Derive is write-only: it publishes evidence events but does not serve queries.
// Queries are served by the store binary from a materialized read model.
type DeriveSupervisor struct {
	cfg                settings.AppConfig
	gateway            ports.ConfigctlGateway
	logger             *slog.Logger
	evRegistry         adapternats.EvidenceRegistry
	sigRegistry        adapternats.SignalRegistry
	decRegistry        adapternats.DecisionRegistry
	stratRegistry      adapternats.StrategyRegistry
	riskRegistry       adapternats.RiskRegistry
	execRegistry       adapternats.ExecutionRegistry
	processors         []FamilyProcessor
	signalProcessors   []SignalFamilyProcessor
	decisionProcessors []DecisionFamilyProcessor
	strategyProcessors []StrategyFamilyProcessor
	riskProcessors     []RiskFamilyProcessor
	executionProcessors []ExecutionFamilyProcessor
	sources            map[string]*actor.PID // key: source → SourceScopeActor PID
	timeframes         []time.Duration
	publisherTracker   *healthz.Tracker
}

func NewDeriveSupervisor(config settings.AppConfig, gateway ports.ConfigctlGateway, publisherTracker *healthz.Tracker) actor.Producer {
	return func() actor.Receiver {
		return &DeriveSupervisor{
			cfg:              config,
			gateway:          gateway,
			logger:           slog.Default().With("actor", "derive-supervisor"),
			sources:          make(map[string]*actor.PID),
			timeframes:       config.Pipeline.TimeframeDurations(),
			publisherTracker: publisherTracker,
		}
	}
}

func (s *DeriveSupervisor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		if err := s.start(c); err != nil {
			s.logger.Error("start derive supervisor", "error", err)
			c.Engine().Poison(c.PID())
		}

	case actor.Stopped:
		s.logger.Info("derive supervisor stopped",
			"active_sources", len(s.sources),
		)

	case tradeReceivedMessage:
		s.routeTrade(c, msg)

	case activateSamplerMessage:
		s.onActivateSampler(c, msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		s.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (s *DeriveSupervisor) start(ctx *actor.Context) error {
	if !s.cfg.NATS.Enabled {
		return fmt.Errorf("nats must be enabled for derive")
	}

	obsRegistry := adapternats.DefaultObservationRegistry()
	s.evRegistry = adapternats.DefaultEvidenceRegistry()
	s.sigRegistry = adapternats.DefaultSignalRegistry()
	s.decRegistry = adapternats.DefaultDecisionRegistry()
	s.stratRegistry = adapternats.DefaultStrategyRegistry()
	s.riskRegistry = adapternats.DefaultRiskRegistry()
	s.execRegistry = adapternats.DefaultExecutionRegistry()
	consumerSpec := adapternats.DeriveObservationConsumer()

	// Evidence family processors — backward-compatible default: enabled when no families configured.
	s.processors = filterEnabled([]FamilyProcessor{
		{
			Family:      "candle",
			ActorPrefix: "sampler",
			NewActor: func(source, symbol string, tf time.Duration, pub, scope *actor.PID) actor.Producer {
				return NewSamplerActor(SamplerConfig{
					Source: source, Symbol: symbol, Timeframe: tf, PublisherPID: pub, ScopePID: scope,
				})
			},
		},
		{
			Family:      "tradeburst",
			ActorPrefix: "burst-sampler",
			NewActor: func(source, symbol string, tf time.Duration, pub, _ *actor.PID) actor.Producer {
				return NewTradeBurstSamplerActor(TradeBurstSamplerConfig{
					Source: source, Symbol: symbol, Timeframe: tf, PublisherPID: pub,
				})
			},
		},
		{
			Family:      "volume",
			ActorPrefix: "volume-sampler",
			NewActor: func(source, symbol string, tf time.Duration, pub, _ *actor.PID) actor.Producer {
				return NewVolumeSamplerActor(VolumeSamplerConfig{
					Source: source, Symbol: symbol, Timeframe: tf, PublisherPID: pub,
				})
			},
		},
	}, func(p FamilyProcessor) string { return p.Family },
		s.cfg.Pipeline.IsFamilyEnabled, s.logger, "evidence")

	if len(s.processors) == 0 {
		return fmt.Errorf("no family processors enabled — check pipeline.families in config")
	}

	// Signal family processors — opt-in via pipeline.signal_families.
	s.signalProcessors = filterEnabled([]SignalFamilyProcessor{
		{
			Family:      "rsi",
			ActorPrefix: "signal-rsi",
			NewActor: func(source, symbol string, tf time.Duration, sigPub, scopePID *actor.PID) actor.Producer {
				return NewRSISignalSamplerActor(SignalSamplerConfig{
					Source: source, Symbol: symbol, Timeframe: tf, SignalPublisherPID: sigPub, ScopePID: scopePID,
				})
			},
		},
		{
			Family:      "ema_crossover",
			ActorPrefix: "signal-ema-crossover",
			NewActor: func(source, symbol string, tf time.Duration, sigPub, scopePID *actor.PID) actor.Producer {
				return NewEMACrossoverSignalSamplerActor(SignalSamplerConfig{
					Source: source, Symbol: symbol, Timeframe: tf, SignalPublisherPID: sigPub, ScopePID: scopePID,
				})
			},
		},
	}, func(p SignalFamilyProcessor) string { return p.Family },
		s.cfg.Pipeline.IsSignalFamilyEnabled, s.logger, "signal")

	// Decision family processors — opt-in via pipeline.decision_families.
	s.decisionProcessors = filterEnabled([]DecisionFamilyProcessor{
		{
			Family:      "rsi_oversold",
			ActorPrefix: "decision-rsi-oversold",
			NewActor: func(source, symbol string, tf time.Duration, decPub, scopePID *actor.PID) actor.Producer {
				return NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
					Source: source, Symbol: symbol, Timeframe: tf, DecisionPublisherPID: decPub, ScopePID: scopePID,
				})
			},
		},
	}, func(p DecisionFamilyProcessor) string { return p.Family },
		s.cfg.Pipeline.IsDecisionFamilyEnabled, s.logger, "decision")

	// Strategy family processors — opt-in via pipeline.strategy_families.
	s.strategyProcessors = filterEnabled([]StrategyFamilyProcessor{
		{
			Family:      "mean_reversion_entry",
			ActorPrefix: "strategy-mean-reversion-entry",
			NewActor: func(source, symbol string, tf time.Duration, stratPub, scopePID *actor.PID) actor.Producer {
				return NewMeanReversionEntryResolverActor(StrategyResolverConfig{
					Source: source, Symbol: symbol, Timeframe: tf, StrategyPublisherPID: stratPub, ScopePID: scopePID,
				})
			},
		},
	}, func(p StrategyFamilyProcessor) string { return p.Family },
		s.cfg.Pipeline.IsStrategyFamilyEnabled, s.logger, "strategy")

	// Risk family processors — opt-in via pipeline.risk_families.
	s.riskProcessors = filterEnabled([]RiskFamilyProcessor{
		{
			Family:      "position_exposure",
			ActorPrefix: "risk-position-exposure",
			NewActor: func(source, symbol string, tf time.Duration, riskPub, scopePID *actor.PID) actor.Producer {
				return NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
					Source: source, Symbol: symbol, Timeframe: tf, RiskPublisherPID: riskPub, ScopePID: scopePID,
				})
			},
		},
	}, func(p RiskFamilyProcessor) string { return p.Family },
		s.cfg.Pipeline.IsRiskFamilyEnabled, s.logger, "risk")

	// Execution family processors — opt-in via pipeline.execution_families.
	s.executionProcessors = filterEnabled([]ExecutionFamilyProcessor{
		{
			Family:      "paper_order",
			ActorPrefix: "execution-paper-order",
			NewActor: func(source, symbol string, tf time.Duration, execPub *actor.PID) actor.Producer {
				return NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
					Source: source, Symbol: symbol, Timeframe: tf, ExecutionPublisherPID: execPub,
				})
			},
		},
	}, func(p ExecutionFamilyProcessor) string { return p.Family },
		s.cfg.Pipeline.IsExecutionFamilyEnabled, s.logger, "execution")

	// Spawn the observation consumer — routes trades back to this supervisor.
	supervisorPID := ctx.PID()
	ctx.SpawnChild(NewConsumerActor(ConsumerConfig{
		URL:          s.cfg.NATS.URL,
		ConsumerSpec: consumerSpec,
		ObsRegistry:  obsRegistry,
		SamplerPID:   supervisorPID, // trades come back here for routing
	}), "observation-consumer")

	// Spawn the binding watcher — queries configctl on startup and subscribes
	// to IngestionRuntimeChangedEvent for dynamic activation without restart.
	ctx.SpawnChild(NewBindingWatcherActor(BindingWatcherConfig{
		NATSURL:        s.cfg.NATS.URL,
		Gateway:        s.gateway,
		SupervisorPID:  supervisorPID,
		RequestTimeout: s.cfg.NATS.RequestTimeoutDuration(),
	}), "binding-watcher")

	tfSeconds := make([]int, len(s.timeframes))
	for i, tf := range s.timeframes {
		tfSeconds[i] = int(tf.Seconds())
	}

	activationMode := "all (no pipeline.families configured)"
	if s.cfg.Pipeline.EnabledFamilies() != nil {
		activationMode = "config-driven"
	}
	s.logger.Info("derive supervisor started",
		"activation", activationMode,
		"families", familyNames(s.processors, func(p FamilyProcessor) string { return p.Family }),
		"signal_families", familyNames(s.signalProcessors, func(p SignalFamilyProcessor) string { return p.Family }),
		"decision_families", familyNames(s.decisionProcessors, func(p DecisionFamilyProcessor) string { return p.Family }),
		"strategy_families", familyNames(s.strategyProcessors, func(p StrategyFamilyProcessor) string { return p.Family }),
		"risk_families", familyNames(s.riskProcessors, func(p RiskFamilyProcessor) string { return p.Family }),
		"execution_families", familyNames(s.executionProcessors, func(p ExecutionFamilyProcessor) string { return p.Family }),
		"timeframes_s", tfSeconds,
		"consumer", consumerSpec.Durable,
		"input_stream", consumerSpec.Event.Stream.Name,
		"output_stream", s.evRegistry.CandleSampled.Stream.Name,
	)
	return nil
}

func (s *DeriveSupervisor) ensureSourceScope(c *actor.Context, source string) *actor.PID {
	pid, exists := s.sources[source]
	if exists {
		return pid
	}

	childName := "source-" + source
	pid = c.SpawnChild(NewSourceScopeActor(SourceScopeConfig{
		Source:             source,
		NATSURL:            s.cfg.NATS.URL,
		Registry:           s.evRegistry,
		SignalRegistry:     s.sigRegistry,
		DecisionRegistry:   s.decRegistry,
		StrategyRegistry:   s.stratRegistry,
		RiskRegistry:        s.riskRegistry,
		ExecutionRegistry:   s.execRegistry,
		Timeframes:          s.timeframes,
		Processors:          s.processors,
		SignalProcessors:    s.signalProcessors,
		DecisionProcessors:  s.decisionProcessors,
		StrategyProcessors:  s.strategyProcessors,
		RiskProcessors:      s.riskProcessors,
		ExecutionProcessors: s.executionProcessors,
		PublisherTracker:    s.publisherTracker,
	}), childName)

	s.sources[source] = pid
	s.logger.Info("source scope created",
		"source", source,
		"active_sources", len(s.sources),
	)
	return pid
}

func (s *DeriveSupervisor) onActivateSampler(c *actor.Context, msg activateSamplerMessage) {
	scope := s.ensureSourceScope(c, msg.Target.Source)
	c.Send(scope, msg)
}

func (s *DeriveSupervisor) routeTrade(c *actor.Context, msg tradeReceivedMessage) {
	source := msg.Event.Trade.Source
	pid, exists := s.sources[source]
	if !exists {
		return
	}
	c.Send(pid, msg)
}
