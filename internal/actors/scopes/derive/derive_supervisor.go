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

	// All available family processors — one entry per evidence type.
	// Which processors actually activate is controlled by pipeline.families in config.
	// If no families are configured, all processors are enabled (backward compatible).
	allProcessors := []FamilyProcessor{
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
	}

	// Filter processors by configured families.
	for _, p := range allProcessors {
		if s.cfg.Pipeline.IsFamilyEnabled(p.Family) {
			s.processors = append(s.processors, p)
		} else {
			s.logger.Info("family processor skipped (not in pipeline.families)",
				"family", p.Family,
			)
		}
	}

	if len(s.processors) == 0 {
		return fmt.Errorf("no family processors enabled — check pipeline.families in config")
	}

	// Signal family processors — opt-in via pipeline.signal_families.
	// Unlike evidence families, absent = no signal activation.
	allSignalProcessors := []SignalFamilyProcessor{
		{
			Family:      "rsi",
			ActorPrefix: "signal-rsi",
			NewActor: func(source, symbol string, tf time.Duration, sigPub, scopePID *actor.PID) actor.Producer {
				return NewRSISignalSamplerActor(SignalSamplerConfig{
					Source: source, Symbol: symbol, Timeframe: tf, SignalPublisherPID: sigPub, ScopePID: scopePID,
				})
			},
		},
	}

	for _, p := range allSignalProcessors {
		if s.cfg.Pipeline.IsSignalFamilyEnabled(p.Family) {
			s.signalProcessors = append(s.signalProcessors, p)
		} else {
			s.logger.Info("signal family processor skipped (not in pipeline.signal_families)",
				"family", p.Family,
			)
		}
	}

	// Decision family processors — opt-in via pipeline.decision_families.
	// Unlike evidence families, absent = no decision activation.
	allDecisionProcessors := []DecisionFamilyProcessor{
		{
			Family:      "rsi_oversold",
			ActorPrefix: "decision-rsi-oversold",
			NewActor: func(source, symbol string, tf time.Duration, decPub, scopePID *actor.PID) actor.Producer {
				return NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
					Source: source, Symbol: symbol, Timeframe: tf, DecisionPublisherPID: decPub, ScopePID: scopePID,
				})
			},
		},
	}

	for _, p := range allDecisionProcessors {
		if s.cfg.Pipeline.IsDecisionFamilyEnabled(p.Family) {
			s.decisionProcessors = append(s.decisionProcessors, p)
		} else {
			s.logger.Info("decision family processor skipped (not in pipeline.decision_families)",
				"family", p.Family,
			)
		}
	}

	// Strategy family processors — opt-in via pipeline.strategy_families.
	// Unlike evidence families, absent = no strategy activation.
	allStrategyProcessors := []StrategyFamilyProcessor{
		{
			Family:      "mean_reversion_entry",
			ActorPrefix: "strategy-mean-reversion-entry",
			NewActor: func(source, symbol string, tf time.Duration, stratPub, scopePID *actor.PID) actor.Producer {
				return NewMeanReversionEntryResolverActor(StrategyResolverConfig{
					Source: source, Symbol: symbol, Timeframe: tf, StrategyPublisherPID: stratPub, ScopePID: scopePID,
				})
			},
		},
	}

	for _, p := range allStrategyProcessors {
		if s.cfg.Pipeline.IsStrategyFamilyEnabled(p.Family) {
			s.strategyProcessors = append(s.strategyProcessors, p)
		} else {
			s.logger.Info("strategy family processor skipped (not in pipeline.strategy_families)",
				"family", p.Family,
			)
		}
	}

	// Risk family processors — opt-in via pipeline.risk_families.
	// Unlike evidence families, absent = no risk activation.
	allRiskProcessors := []RiskFamilyProcessor{
		{
			Family:      "position_exposure",
			ActorPrefix: "risk-position-exposure",
			NewActor: func(source, symbol string, tf time.Duration, riskPub, scopePID *actor.PID) actor.Producer {
				return NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
					Source: source, Symbol: symbol, Timeframe: tf, RiskPublisherPID: riskPub, ScopePID: scopePID,
				})
			},
		},
	}

	for _, p := range allRiskProcessors {
		if s.cfg.Pipeline.IsRiskFamilyEnabled(p.Family) {
			s.riskProcessors = append(s.riskProcessors, p)
		} else {
			s.logger.Info("risk family processor skipped (not in pipeline.risk_families)",
				"family", p.Family,
			)
		}
	}

	// Execution family processors — opt-in via pipeline.execution_families.
	// Unlike evidence families, absent = no execution activation.
	allExecutionProcessors := []ExecutionFamilyProcessor{
		{
			Family:      "paper_order",
			ActorPrefix: "execution-paper-order",
			NewActor: func(source, symbol string, tf time.Duration, execPub *actor.PID) actor.Producer {
				return NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
					Source: source, Symbol: symbol, Timeframe: tf, ExecutionPublisherPID: execPub,
				})
			},
		},
	}

	for _, p := range allExecutionProcessors {
		if s.cfg.Pipeline.IsExecutionFamilyEnabled(p.Family) {
			s.executionProcessors = append(s.executionProcessors, p)
		} else {
			s.logger.Info("execution family processor skipped (not in pipeline.execution_families)",
				"family", p.Family,
			)
		}
	}

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
	families := make([]string, len(s.processors))
	for i, p := range s.processors {
		families[i] = p.Family
	}
	signalFamilies := make([]string, len(s.signalProcessors))
	for i, p := range s.signalProcessors {
		signalFamilies[i] = p.Family
	}
	decisionFamilies := make([]string, len(s.decisionProcessors))
	for i, p := range s.decisionProcessors {
		decisionFamilies[i] = p.Family
	}
	strategyFamilies := make([]string, len(s.strategyProcessors))
	for i, p := range s.strategyProcessors {
		strategyFamilies[i] = p.Family
	}
	riskFamilies := make([]string, len(s.riskProcessors))
	for i, p := range s.riskProcessors {
		riskFamilies[i] = p.Family
	}
	executionFamilies := make([]string, len(s.executionProcessors))
	for i, p := range s.executionProcessors {
		executionFamilies[i] = p.Family
	}

	activationMode := "all (no pipeline.families configured)"
	if s.cfg.Pipeline.EnabledFamilies() != nil {
		activationMode = "config-driven"
	}
	s.logger.Info("derive supervisor started",
		"activation", activationMode,
		"families", families,
		"signal_families", signalFamilies,
		"decision_families", decisionFamilies,
		"strategy_families", strategyFamilies,
		"risk_families", riskFamilies,
		"execution_families", executionFamilies,
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
