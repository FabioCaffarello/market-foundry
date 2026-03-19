package store

import (
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/shared/healthz"
	"internal/shared/settings"

	"github.com/anthdm/hollywood/actor"
)

// ProjectionPipeline describes one evidence type's projection pipeline in the store.
// Each pipeline pairs a durable JetStream consumer with a projection actor that
// materializes events into KV buckets. This is the canonical unit of registration —
// adding a new evidence type means adding one ProjectionPipeline entry in start().
type ProjectionPipeline struct {
	// Family is the canonical evidence type name (e.g., "candle", "tradeburst").
	Family string
	// ProjectionName is the actor child name (e.g., "candle-projection").
	ProjectionName string
	// ConsumerName is the actor child name (e.g., "candle-consumer").
	ConsumerName string
	// Buckets lists the KV bucket names owned by this pipeline's projection actor.
	Buckets []string
	// ConsumerSpec returns the durable consumer spec for this pipeline.
	ConsumerSpec adapternats.ConsumerSpec
	// NewProjection creates the projection actor for this pipeline.
	NewProjection func(natsURL string, tracker *healthz.Tracker) actor.Producer
	// NewConsumer creates the consumer actor for this pipeline.
	NewConsumer func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.EvidenceRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer
}

// SignalPipeline describes one signal type's projection pipeline in the store.
// Structurally identical to ProjectionPipeline but uses SignalRegistry for consumers
// and IsSignalFamilyEnabled for activation (opt-in, not backward-compatible default).
type SignalPipeline struct {
	Family         string
	ProjectionName string
	ConsumerName   string
	Buckets        []string
	ConsumerSpec   adapternats.ConsumerSpec
	NewProjection  func(natsURL string, tracker *healthz.Tracker) actor.Producer
	NewConsumer    func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.SignalRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer
}

// DecisionPipeline describes one decision type's projection pipeline in the store.
// Uses DecisionRegistry for consumers and IsDecisionFamilyEnabled for activation (opt-in).
type DecisionPipeline struct {
	Family         string
	ProjectionName string
	ConsumerName   string
	Buckets        []string
	ConsumerSpec   adapternats.ConsumerSpec
	NewProjection  func(natsURL string, tracker *healthz.Tracker) actor.Producer
	NewConsumer    func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.DecisionRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer
}

// StrategyPipeline describes one strategy type's projection pipeline in the store.
// Uses StrategyRegistry for consumers and IsStrategyFamilyEnabled for activation (opt-in).
type StrategyPipeline struct {
	Family         string
	ProjectionName string
	ConsumerName   string
	Buckets        []string
	ConsumerSpec   adapternats.ConsumerSpec
	NewProjection  func(natsURL string, tracker *healthz.Tracker) actor.Producer
	NewConsumer    func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.StrategyRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer
}

// RiskPipeline describes one risk type's projection pipeline in the store.
// Uses RiskRegistry for consumers and IsRiskFamilyEnabled for activation (opt-in).
type RiskPipeline struct {
	Family         string
	ProjectionName string
	ConsumerName   string
	Buckets        []string
	ConsumerSpec   adapternats.ConsumerSpec
	NewProjection  func(natsURL string, tracker *healthz.Tracker) actor.Producer
	NewConsumer    func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.RiskRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer
}

// ExecutionPipeline describes one execution type's projection pipeline in the store.
// Uses ExecutionRegistry for consumers and IsExecutionFamilyEnabled for activation (opt-in).
type ExecutionPipeline struct {
	Family         string
	ProjectionName string
	ConsumerName   string
	Buckets        []string
	ConsumerSpec   adapternats.ConsumerSpec
	NewProjection  func(natsURL string, tracker *healthz.Tracker) actor.Producer
	NewConsumer    func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.ExecutionRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer
}

// StoreSupervisor is the root actor for the store binary.
// It materializes evidence events into a persistent read model (NATS KV)
// and serves queries from the gateway. Projection pipelines are registered
// declaratively as ProjectionPipeline entries — one per evidence type.
type StoreSupervisor struct {
	cfg       settings.AppConfig
	logger    *slog.Logger
	trackers  map[string]*healthz.Tracker // key: "candle-projection", "candle-consumer", etc.
	pipelines []ProjectionPipeline
}

func NewStoreSupervisor(config settings.AppConfig, trackers map[string]*healthz.Tracker) actor.Producer {
	return func() actor.Receiver {
		return &StoreSupervisor{
			cfg:      config,
			logger:   slog.Default().With("actor", "store-supervisor"),
			trackers: trackers,
		}
	}
}

func (s *StoreSupervisor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		if err := s.start(c); err != nil {
			s.logger.Error("start store supervisor", "error", err)
			c.Engine().Poison(c.PID())
		}

	case actor.Stopped:
		s.logger.Info("store supervisor stopped")

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		s.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (s *StoreSupervisor) start(ctx *actor.Context) error {
	if !s.cfg.NATS.Enabled {
		return fmt.Errorf("nats must be enabled for store")
	}

	evRegistry := adapternats.DefaultEvidenceRegistry()

	// All available projection pipelines — one entry per evidence type.
	// Which pipelines actually spawn is controlled by pipeline.families in config.
	// If no families are configured, all pipelines are enabled (backward compatible).
	allPipelines := []ProjectionPipeline{
		{
			Family:         "candle",
			ProjectionName: "candle-projection",
			ConsumerName:   "candle-consumer",
			Buckets:        []string{adapternats.CandleLatestBucket, adapternats.CandleHistoryBucket},
			ConsumerSpec:   adapternats.StoreCandleConsumer(),
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewCandleProjectionActor(CandleProjectionConfig{NATSURL: natsURL, Tracker: tracker})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.EvidenceRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewEvidenceConsumerActor(EvidenceConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: registry, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
		{
			Family:         "tradeburst",
			ProjectionName: "trade-burst-projection",
			ConsumerName:   "trade-burst-consumer",
			Buckets:        []string{adapternats.TradeBurstLatestBucket},
			ConsumerSpec:   adapternats.StoreTradeBurstConsumer(),
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewTradeBurstProjectionActor(TradeBurstProjectionConfig{NATSURL: natsURL, Tracker: tracker})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.EvidenceRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewTradeBurstConsumerActor(TradeBurstConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: registry, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
		{
			Family:         "volume",
			ProjectionName: "volume-projection",
			ConsumerName:   "volume-consumer",
			Buckets:        []string{adapternats.VolumeLatestBucket},
			ConsumerSpec:   adapternats.StoreVolumeConsumer(),
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewVolumeProjectionActor(VolumeProjectionConfig{NATSURL: natsURL, Tracker: tracker})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.EvidenceRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewVolumeConsumerActor(VolumeConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: registry, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
	}

	// Filter pipelines by configured families.
	for _, p := range allPipelines {
		if s.cfg.Pipeline.IsFamilyEnabled(p.Family) {
			s.pipelines = append(s.pipelines, p)
		} else {
			s.logger.Info("pipeline skipped (not in pipeline.families)",
				"family", p.Family,
			)
		}
	}

	if len(s.pipelines) == 0 {
		return fmt.Errorf("no projection pipelines enabled — check pipeline.families in config")
	}

	// Spawn enabled projection pipelines.
	var allBuckets []string
	for _, p := range s.pipelines {
		projTracker := s.trackers[p.ProjectionName]
		consTracker := s.trackers[p.ConsumerName]

		projPID := ctx.SpawnChild(p.NewProjection(s.cfg.NATS.URL, projTracker), p.ProjectionName)
		ctx.SpawnChild(p.NewConsumer(s.cfg.NATS.URL, p.ConsumerSpec, evRegistry, projPID, consTracker), p.ConsumerName)

		allBuckets = append(allBuckets, p.Buckets...)
	}

	// --- Signal pipelines (opt-in via pipeline.signal_families) ---
	sigRegistry := adapternats.DefaultSignalRegistry()

	allSignalPipelines := []SignalPipeline{
		{
			Family:         "rsi",
			ProjectionName: "signal-rsi-projection",
			ConsumerName:   "signal-rsi-consumer",
			Buckets:        []string{adapternats.SignalRSILatestBucket},
			ConsumerSpec:   adapternats.StoreRSISignalConsumer(),
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewSignalProjectionActor(SignalProjectionConfig{
					NATSURL: natsURL,
					Bucket:  adapternats.SignalRSILatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.SignalRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewSignalConsumerActor(SignalConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: registry, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
	}

	var enabledSignalFamilies []string
	for _, sp := range allSignalPipelines {
		if s.cfg.Pipeline.IsSignalFamilyEnabled(sp.Family) {
			projTracker := s.trackers[sp.ProjectionName]
			consTracker := s.trackers[sp.ConsumerName]

			projPID := ctx.SpawnChild(sp.NewProjection(s.cfg.NATS.URL, projTracker), sp.ProjectionName)
			ctx.SpawnChild(sp.NewConsumer(s.cfg.NATS.URL, sp.ConsumerSpec, sigRegistry, projPID, consTracker), sp.ConsumerName)

			allBuckets = append(allBuckets, sp.Buckets...)
			enabledSignalFamilies = append(enabledSignalFamilies, sp.Family)
		} else {
			s.logger.Info("signal pipeline skipped (not in pipeline.signal_families)",
				"family", sp.Family,
			)
		}
	}

	// --- Decision pipelines (opt-in via pipeline.decision_families) ---
	decRegistry := adapternats.DefaultDecisionRegistry()

	allDecisionPipelines := []DecisionPipeline{
		{
			Family:         "rsi_oversold",
			ProjectionName: "decision-rsi-oversold-projection",
			ConsumerName:   "decision-rsi-oversold-consumer",
			Buckets:        []string{adapternats.DecisionRSIOversoldLatestBucket},
			ConsumerSpec:   adapternats.StoreRSIOversoldDecisionConsumer(),
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewDecisionProjectionActor(DecisionProjectionConfig{
					NATSURL: natsURL,
					Bucket:  adapternats.DecisionRSIOversoldLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.DecisionRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewDecisionConsumerActor(DecisionConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: registry, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
	}

	var enabledDecisionFamilies []string
	for _, dp := range allDecisionPipelines {
		if s.cfg.Pipeline.IsDecisionFamilyEnabled(dp.Family) {
			projTracker := s.trackers[dp.ProjectionName]
			consTracker := s.trackers[dp.ConsumerName]

			projPID := ctx.SpawnChild(dp.NewProjection(s.cfg.NATS.URL, projTracker), dp.ProjectionName)
			ctx.SpawnChild(dp.NewConsumer(s.cfg.NATS.URL, dp.ConsumerSpec, decRegistry, projPID, consTracker), dp.ConsumerName)

			allBuckets = append(allBuckets, dp.Buckets...)
			enabledDecisionFamilies = append(enabledDecisionFamilies, dp.Family)
		} else {
			s.logger.Info("decision pipeline skipped (not in pipeline.decision_families)",
				"family", dp.Family,
			)
		}
	}

	// --- Strategy pipelines (opt-in via pipeline.strategy_families) ---
	stratRegistry := adapternats.DefaultStrategyRegistry()

	allStrategyPipelines := []StrategyPipeline{
		{
			Family:         "mean_reversion_entry",
			ProjectionName: "strategy-mean-reversion-entry-projection",
			ConsumerName:   "strategy-mean-reversion-entry-consumer",
			Buckets:        []string{adapternats.StrategyMeanReversionEntryLatestBucket},
			ConsumerSpec:   adapternats.StoreMeanReversionEntryStrategyConsumer(),
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewStrategyProjectionActor(StrategyProjectionConfig{
					NATSURL: natsURL,
					Bucket:  adapternats.StrategyMeanReversionEntryLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.StrategyRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewStrategyConsumerActor(StrategyConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: registry, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
	}

	var enabledStrategyFamilies []string
	for _, sp := range allStrategyPipelines {
		if s.cfg.Pipeline.IsStrategyFamilyEnabled(sp.Family) {
			projTracker := s.trackers[sp.ProjectionName]
			consTracker := s.trackers[sp.ConsumerName]

			projPID := ctx.SpawnChild(sp.NewProjection(s.cfg.NATS.URL, projTracker), sp.ProjectionName)
			ctx.SpawnChild(sp.NewConsumer(s.cfg.NATS.URL, sp.ConsumerSpec, stratRegistry, projPID, consTracker), sp.ConsumerName)

			allBuckets = append(allBuckets, sp.Buckets...)
			enabledStrategyFamilies = append(enabledStrategyFamilies, sp.Family)
		} else {
			s.logger.Info("strategy pipeline skipped (not in pipeline.strategy_families)",
				"family", sp.Family,
			)
		}
	}

	// --- Risk pipelines (opt-in via pipeline.risk_families) ---
	riskRegistry := adapternats.DefaultRiskRegistry()

	allRiskPipelines := []RiskPipeline{
		{
			Family:         "position_exposure",
			ProjectionName: "risk-position-exposure-projection",
			ConsumerName:   "risk-position-exposure-consumer",
			Buckets:        []string{adapternats.RiskPositionExposureLatestBucket},
			ConsumerSpec:   adapternats.StorePositionExposureRiskConsumer(),
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewRiskProjectionActor(RiskProjectionConfig{
					NATSURL: natsURL,
					Bucket:  adapternats.RiskPositionExposureLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.RiskRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewRiskConsumerActor(RiskConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: registry, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
	}

	var enabledRiskFamilies []string
	for _, rp := range allRiskPipelines {
		if s.cfg.Pipeline.IsRiskFamilyEnabled(rp.Family) {
			projTracker := s.trackers[rp.ProjectionName]
			consTracker := s.trackers[rp.ConsumerName]

			projPID := ctx.SpawnChild(rp.NewProjection(s.cfg.NATS.URL, projTracker), rp.ProjectionName)
			ctx.SpawnChild(rp.NewConsumer(s.cfg.NATS.URL, rp.ConsumerSpec, riskRegistry, projPID, consTracker), rp.ConsumerName)

			allBuckets = append(allBuckets, rp.Buckets...)
			enabledRiskFamilies = append(enabledRiskFamilies, rp.Family)
		} else {
			s.logger.Info("risk pipeline skipped (not in pipeline.risk_families)",
				"family", rp.Family,
			)
		}
	}

	// --- Execution pipelines (opt-in via pipeline.execution_families) ---
	execRegistry := adapternats.DefaultExecutionRegistry()

	// Paper Family: materializes paper_order intents from derive.
	// Venue Family: materializes venue_market_order fills from execute.
	// Both families have independent consumers, projections, and KV buckets.
	allExecutionPipelines := []ExecutionPipeline{
		{
			Family:         "paper_order",
			ProjectionName: "execution-paper-order-projection",
			ConsumerName:   "execution-paper-order-consumer",
			Buckets:        []string{adapternats.ExecutionPaperOrderLatestBucket},
			ConsumerSpec:   adapternats.StorePaperOrderExecutionConsumer(),
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewExecutionProjectionActor(ExecutionProjectionConfig{
					NATSURL: natsURL,
					Bucket:  adapternats.ExecutionPaperOrderLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.ExecutionRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewExecutionConsumerActor(ExecutionConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: registry, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
		{
			Family:         "venue_market_order",
			ProjectionName: "execution-venue-market-order-projection",
			ConsumerName:   "execution-venue-market-order-consumer",
			Buckets:        []string{adapternats.ExecutionVenueMarketOrderLatestBucket},
			ConsumerSpec:   adapternats.StoreVenueMarketOrderFillConsumer(),
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewFillProjectionActor(FillProjectionConfig{
					NATSURL:      natsURL,
					Bucket:       adapternats.ExecutionVenueMarketOrderLatestBucket,
					IntentBucket: adapternats.ExecutionPaperOrderLatestBucket,
					Tracker:      tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, registry adapternats.ExecutionRegistry, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewFillConsumerActor(FillConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: registry, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
	}

	var enabledExecutionFamilies []string
	for _, ep := range allExecutionPipelines {
		if s.cfg.Pipeline.IsExecutionFamilyEnabled(ep.Family) {
			projTracker := s.trackers[ep.ProjectionName]
			consTracker := s.trackers[ep.ConsumerName]

			projPID := ctx.SpawnChild(ep.NewProjection(s.cfg.NATS.URL, projTracker), ep.ProjectionName)
			ctx.SpawnChild(ep.NewConsumer(s.cfg.NATS.URL, ep.ConsumerSpec, execRegistry, projPID, consTracker), ep.ConsumerName)

			allBuckets = append(allBuckets, ep.Buckets...)
			enabledExecutionFamilies = append(enabledExecutionFamilies, ep.Family)
		} else {
			s.logger.Info("execution pipeline skipped (not in pipeline.execution_families)",
				"family", ep.Family,
			)
		}
	}

	// Spawn query responder (serves evidence, signal, decision, strategy, risk, and execution queries).
	qrCfg := QueryResponderConfig{
		NATSURL:  s.cfg.NATS.URL,
		Source:   "store.query-responder",
		Registry: evRegistry,
	}
	if len(enabledSignalFamilies) > 0 {
		qrCfg.SignalRegistry = &sigRegistry
	}
	if len(enabledDecisionFamilies) > 0 {
		qrCfg.DecisionRegistry = &decRegistry
	}
	if len(enabledStrategyFamilies) > 0 {
		qrCfg.StrategyRegistry = &stratRegistry
	}
	if len(enabledRiskFamilies) > 0 {
		qrCfg.RiskRegistry = &riskRegistry
	}
	if len(enabledExecutionFamilies) > 0 {
		qrCfg.ExecutionRegistry = &execRegistry
	}
	ctx.SpawnChild(NewQueryResponderActor(qrCfg), "query-responder")

	families := make([]string, len(s.pipelines))
	durables := make([]string, len(s.pipelines))
	for i, p := range s.pipelines {
		families[i] = p.Family
		durables[i] = p.ConsumerSpec.Durable
	}

	activationMode := "all (no pipeline.families configured)"
	if s.cfg.Pipeline.EnabledFamilies() != nil {
		activationMode = "config-driven"
	}
	s.logger.Info("store supervisor started",
		"activation", activationMode,
		"families", families,
		"signal_families", enabledSignalFamilies,
		"decision_families", enabledDecisionFamilies,
		"strategy_families", enabledStrategyFamilies,
		"risk_families", enabledRiskFamilies,
		"execution_families", enabledExecutionFamilies,
		"consumers", durables,
		"stream", evRegistry.CandleSampled.Stream.Name,
		"buckets", allBuckets,
	)
	return nil
}
