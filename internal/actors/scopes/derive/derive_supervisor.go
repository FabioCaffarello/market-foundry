package derive

import (
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	natsdecision "internal/adapters/nats/natsdecision"
	natsevidence "internal/adapters/nats/natsevidence"
	natsexecution "internal/adapters/nats/natsexecution"
	natsinsights "internal/adapters/nats/natsinsights"
	natsobservation "internal/adapters/nats/natsobservation"
	natsrisk "internal/adapters/nats/natsrisk"
	natssignal "internal/adapters/nats/natssignal"
	natsstrategy "internal/adapters/nats/natsstrategy"
	"internal/application/ports"
	"internal/domain/instrument"
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
	cfg                 settings.AppConfig
	gateway             ports.ConfigctlGateway
	logger              *slog.Logger
	evRegistry          natsevidence.Registry
	sigRegistry         natssignal.Registry
	decRegistry         natsdecision.Registry
	stratRegistry       natsstrategy.Registry
	riskRegistry        natsrisk.Registry
	execRegistry        natsexecution.Registry
	insRegistry         natsinsights.Registry
	processors          []FamilyProcessor
	signalProcessors    []SignalFamilyProcessor
	decisionProcessors  []DecisionFamilyProcessor
	strategyProcessors  []StrategyFamilyProcessor
	riskProcessors      []RiskFamilyProcessor
	executionProcessors []ExecutionFamilyProcessor
	insightsProcessors  []FamilyProcessor
	sources             map[string]*actor.PID // key: source → SourceScopeActor PID
	crossVenuePID       *actor.PID            // single cross-venue fusion actor (H-8.c)
	timeframes          []time.Duration
	publisherTracker    *healthz.Tracker
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

	obsRegistry := natsobservation.DefaultRegistry()
	s.evRegistry = natsevidence.DefaultRegistry()
	s.sigRegistry = natssignal.DefaultRegistry()
	s.decRegistry = natsdecision.DefaultRegistry()
	s.stratRegistry = natsstrategy.DefaultRegistry()
	s.riskRegistry = natsrisk.DefaultRegistry()
	s.execRegistry = natsexecution.DefaultRegistry()
	s.insRegistry = natsinsights.DefaultRegistry()
	consumerSpec := natsobservation.DeriveObservationConsumer()

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
			NewActor: func(source, symbol string, inst instrument.CanonicalInstrument, tf time.Duration, sigPub, scopePID *actor.PID) actor.Producer {
				return NewRSISignalSamplerActor(SignalSamplerConfig{
					Source: source, Symbol: symbol, Instrument: inst, Timeframe: tf, SignalPublisherPID: sigPub, ScopePID: scopePID,
				})
			},
		},
		{
			Family:      "ema_crossover",
			ActorPrefix: "signal-ema-crossover",
			NewActor: func(source, symbol string, inst instrument.CanonicalInstrument, tf time.Duration, sigPub, scopePID *actor.PID) actor.Producer {
				return NewEMACrossoverSignalSamplerActor(SignalSamplerConfig{
					Source: source, Symbol: symbol, Instrument: inst, Timeframe: tf, SignalPublisherPID: sigPub, ScopePID: scopePID,
				})
			},
		},
		{
			Family:      "bollinger",
			ActorPrefix: "signal-bollinger",
			NewActor: func(source, symbol string, inst instrument.CanonicalInstrument, tf time.Duration, sigPub, scopePID *actor.PID) actor.Producer {
				return NewBollingerSignalSamplerActor(SignalSamplerConfig{
					Source: source, Symbol: symbol, Instrument: inst, Timeframe: tf, SignalPublisherPID: sigPub, ScopePID: scopePID,
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
			NewActor: func(source, symbol string, inst instrument.CanonicalInstrument, tf time.Duration, decPub, scopePID *actor.PID) actor.Producer {
				return NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
					Source: source, Symbol: symbol, Instrument: inst, Timeframe: tf, DecisionPublisherPID: decPub, ScopePID: scopePID,
				})
			},
		},
		{
			Family:      "ema_crossover",
			ActorPrefix: "decision-ema-crossover",
			NewActor: func(source, symbol string, inst instrument.CanonicalInstrument, tf time.Duration, decPub, scopePID *actor.PID) actor.Producer {
				return NewEMACrossoverEvaluatorActor(DecisionEvaluatorConfig{
					Source: source, Symbol: symbol, Instrument: inst, Timeframe: tf, DecisionPublisherPID: decPub, ScopePID: scopePID,
				})
			},
		},
		{
			Family:      "bollinger_squeeze",
			ActorPrefix: "decision-bollinger-squeeze",
			NewActor: func(source, symbol string, inst instrument.CanonicalInstrument, tf time.Duration, decPub, scopePID *actor.PID) actor.Producer {
				return NewBollingerSqueezeEvaluatorActor(DecisionEvaluatorConfig{
					Source: source, Symbol: symbol, Instrument: inst, Timeframe: tf, DecisionPublisherPID: decPub, ScopePID: scopePID,
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
			NewActor: func(source, symbol string, inst instrument.CanonicalInstrument, tf time.Duration, stratPub, scopePID *actor.PID) actor.Producer {
				return NewMeanReversionEntryResolverActor(StrategyResolverConfig{
					Source: source, Symbol: symbol, Instrument: inst, Timeframe: tf, StrategyPublisherPID: stratPub, ScopePID: scopePID,
				})
			},
		},
		{
			Family:      "trend_following_entry",
			ActorPrefix: "strategy-trend-following-entry",
			NewActor: func(source, symbol string, inst instrument.CanonicalInstrument, tf time.Duration, stratPub, scopePID *actor.PID) actor.Producer {
				return NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
					Source: source, Symbol: symbol, Instrument: inst, Timeframe: tf, StrategyPublisherPID: stratPub, ScopePID: scopePID,
				})
			},
		},
		{
			Family:      "squeeze_breakout_entry",
			ActorPrefix: "strategy-squeeze-breakout-entry",
			NewActor: func(source, symbol string, inst instrument.CanonicalInstrument, tf time.Duration, stratPub, scopePID *actor.PID) actor.Producer {
				return NewSqueezeBreakoutEntryResolverActor(StrategyResolverConfig{
					Source: source, Symbol: symbol, Instrument: inst, Timeframe: tf, StrategyPublisherPID: stratPub, ScopePID: scopePID,
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
			NewActor: func(source, symbol string, inst instrument.CanonicalInstrument, tf time.Duration, riskPub, scopePID *actor.PID) actor.Producer {
				return NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
					Source: source, Symbol: symbol, Instrument: inst, Timeframe: tf, RiskPublisherPID: riskPub, ScopePID: scopePID,
				})
			},
		},
		{
			Family:      "drawdown_limit",
			ActorPrefix: "risk-drawdown-limit",
			NewActor: func(source, symbol string, inst instrument.CanonicalInstrument, tf time.Duration, riskPub, scopePID *actor.PID) actor.Producer {
				return NewDrawdownLimitEvaluatorActor(RiskEvaluatorConfig{
					Source: source, Symbol: symbol, Instrument: inst, Timeframe: tf, RiskPublisherPID: riskPub, ScopePID: scopePID,
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
			NewActor: func(source, symbol string, inst instrument.CanonicalInstrument, tf time.Duration, execPub *actor.PID) actor.Producer {
				return NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
					Source: source, Symbol: symbol, Instrument: inst, Timeframe: tf, ExecutionPublisherPID: execPub,
				})
			},
		},
	}, func(p ExecutionFamilyProcessor) string { return p.Family },
		s.cfg.Pipeline.IsExecutionFamilyEnabled, s.logger, "execution")

	// Insights family processors (PROGRAM-0005 / H-8.a). Consume
	// trades like evidence; publish via the insights publisher.
	// Decision-support only (ADR-0027). Always enabled — insights is
	// a read-only descriptive overlay, not part of the directive
	// pipeline that pipeline.*_families gates. BucketSize default "1"
	// (price units of the quote asset); tunable per-config is a
	// future refinement.
	s.insightsProcessors = []FamilyProcessor{
		{
			Family:      "volume_profile",
			ActorPrefix: "volume-profile-sampler",
			NewActor: func(source, symbol string, tf time.Duration, pub, _ *actor.PID) actor.Producer {
				return NewVolumeProfileSamplerActor(VolumeProfileSamplerConfig{
					Source: source, Symbol: symbol, Timeframe: tf,
					BucketSize: "1", MaxBuckets: 0, PublisherPID: pub,
				})
			},
		},
		{
			Family:      "tpo",
			ActorPrefix: "tpo-sampler",
			NewActor: func(source, symbol string, tf time.Duration, pub, _ *actor.PID) actor.Producer {
				// PeriodSeconds 0 → sampler derives ~12 periods/window
				// (capped to the A..X range). BucketSize "1" like VPVR.
				return NewTPOSamplerActor(TPOSamplerConfig{
					Source: source, Symbol: symbol, Timeframe: tf,
					BucketSize: "1", PeriodSeconds: 0, MaxLevels: 0, PublisherPID: pub,
				})
			},
		},
	}

	// Spawn the observation consumer — routes trades back to this supervisor.
	supervisorPID := ctx.PID()
	ctx.SpawnChild(NewConsumerActor(ConsumerConfig{
		URL:          s.cfg.NATS.URL,
		ConsumerSpec: consumerSpec,
		ObsRegistry:  obsRegistry,
		SamplerPID:   supervisorPID, // trades come back here for routing
	}), "observation-consumer")

	// Spawn the SINGLE cross-venue fusion actor (H-8.c, Decisão C1). It
	// lives at the supervisor level — not per-source — because fusion is
	// cross-source by definition. The supervisor fans every trade to it
	// (routeTrade). Always-on, like the per-source insights samplers.
	s.crossVenuePID = ctx.SpawnChild(NewCrossVenueFusionActor(CrossVenueFusionConfig{
		NATSURL:    s.cfg.NATS.URL,
		Registry:   s.insRegistry,
		Timeframes: s.timeframes,
		Tracker:    s.publisherTracker,
	}), "cross-venue-fusion")

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
		Source:              source,
		NATSURL:             s.cfg.NATS.URL,
		Registry:            s.evRegistry,
		SignalRegistry:      s.sigRegistry,
		DecisionRegistry:    s.decRegistry,
		StrategyRegistry:    s.stratRegistry,
		RiskRegistry:        s.riskRegistry,
		ExecutionRegistry:   s.execRegistry,
		InsightsRegistry:    s.insRegistry,
		Timeframes:          s.timeframes,
		Processors:          s.processors,
		SignalProcessors:    s.signalProcessors,
		DecisionProcessors:  s.decisionProcessors,
		StrategyProcessors:  s.strategyProcessors,
		RiskProcessors:      s.riskProcessors,
		ExecutionProcessors: s.executionProcessors,
		InsightsProcessors:  s.insightsProcessors,
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
	// Cross-venue fusion sees EVERY trade across all sources (H-8.c),
	// independent of per-source scope routing.
	if s.crossVenuePID != nil {
		c.Send(s.crossVenuePID, msg)
	}

	source := msg.Event.Trade.Source
	pid, exists := s.sources[source]
	if !exists {
		return
	}
	c.Send(pid, msg)
}
