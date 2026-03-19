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

// PipelineDomain identifies which bounded-context domain a pipeline belongs to.
// Used to determine which registries to inject into the query responder.
type PipelineDomain string

const (
	DomainEvidence  PipelineDomain = "evidence"
	DomainSignal    PipelineDomain = "signal"
	DomainDecision  PipelineDomain = "decision"
	DomainStrategy  PipelineDomain = "strategy"
	DomainRisk      PipelineDomain = "risk"
	DomainExecution PipelineDomain = "execution"
)

// Pipeline describes one projection pipeline in the store.
// Each pipeline pairs a durable JetStream consumer with a projection actor that
// materializes events into KV buckets. Adding a new pipeline of any scope means
// adding one Pipeline entry in declarePipelines().
//
// The consumer factory captures its registry via closure, eliminating the need
// for separate pipeline types per registry kind.
type Pipeline struct {
	// Scope identifies the domain this pipeline belongs to (evidence, signal, etc.).
	Scope PipelineDomain
	// Family is the canonical type name (e.g., "candle", "rsi", "paper_order").
	Family string
	// ProjectionName is the actor child name (e.g., "candle-projection").
	ProjectionName string
	// ConsumerName is the actor child name (e.g., "candle-consumer").
	ConsumerName string
	// Buckets lists the KV bucket names owned by this pipeline's projection actor.
	Buckets []string
	// ConsumerSpec returns the durable consumer spec for this pipeline.
	ConsumerSpec adapternats.ConsumerSpec
	// IsEnabled reports whether this pipeline should be spawned given current config.
	IsEnabled func(settings.PipelineConfig) bool
	// NewProjection creates the projection actor for this pipeline.
	NewProjection func(natsURL string, tracker *healthz.Tracker) actor.Producer
	// NewConsumer creates the consumer actor for this pipeline.
	// The registry is already bound via closure at declaration time.
	NewConsumer func(natsURL string, spec adapternats.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer
}

// TrackerDef describes the health tracker pair for one projection pipeline.
// Exported so that the composition root can derive trackers from the pipeline catalog
// without maintaining a separate list.
type TrackerDef struct {
	ProjectionName string
	ConsumerName   string
	IsEnabled      func(settings.PipelineConfig) bool
}

// PipelineTrackerDefs returns tracker definitions derived from the canonical pipeline catalog.
// This is the single source of truth — adding a pipeline in declarePipelines() automatically
// makes it available for tracker creation.
func PipelineTrackerDefs() []TrackerDef {
	pipelines, _ := declarePipelines()
	defs := make([]TrackerDef, len(pipelines))
	for i, p := range pipelines {
		defs[i] = TrackerDef{
			ProjectionName: p.ProjectionName,
			ConsumerName:   p.ConsumerName,
			IsEnabled:      p.IsEnabled,
		}
	}
	return defs
}

// pipelineRegistries holds all domain registries created during pipeline declaration.
// Passed to the query responder for conditional registry injection.
type pipelineRegistries struct {
	evidence  adapternats.EvidenceRegistry
	signal    adapternats.SignalRegistry
	decision  adapternats.DecisionRegistry
	strategy  adapternats.StrategyRegistry
	risk      adapternats.RiskRegistry
	execution adapternats.ExecutionRegistry
}

// queryResponderConfig builds the QueryResponderConfig with registries for enabled scopes.
func (r pipelineRegistries) queryResponderConfig(natsURL string, activeScopes map[PipelineDomain]bool) QueryResponderConfig {
	cfg := QueryResponderConfig{
		NATSURL:  natsURL,
		Source:   "store.query-responder",
		Registry: r.evidence,
	}
	if activeScopes[DomainSignal] {
		cfg.SignalRegistry = &r.signal
	}
	if activeScopes[DomainDecision] {
		cfg.DecisionRegistry = &r.decision
	}
	if activeScopes[DomainStrategy] {
		cfg.StrategyRegistry = &r.strategy
	}
	if activeScopes[DomainRisk] {
		cfg.RiskRegistry = &r.risk
	}
	if activeScopes[DomainExecution] {
		cfg.ExecutionRegistry = &r.execution
	}
	return cfg
}

// declarePipelines returns all available projection pipelines and their registries.
// Which pipelines actually spawn is controlled by each pipeline's IsEnabled predicate.
func declarePipelines() ([]Pipeline, pipelineRegistries) {
	reg := pipelineRegistries{
		evidence:  adapternats.DefaultEvidenceRegistry(),
		signal:    adapternats.DefaultSignalRegistry(),
		decision:  adapternats.DefaultDecisionRegistry(),
		strategy:  adapternats.DefaultStrategyRegistry(),
		risk:      adapternats.DefaultRiskRegistry(),
		execution: adapternats.DefaultExecutionRegistry(),
	}

	return []Pipeline{
		// --- Evidence pipelines (backward-compatible default) ---
		{
			Scope:          DomainEvidence,
			Family:         "candle",
			ProjectionName: "candle-projection",
			ConsumerName:   "candle-consumer",
			Buckets:        []string{adapternats.CandleLatestBucket, adapternats.CandleHistoryBucket},
			ConsumerSpec:   adapternats.StoreCandleConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsFamilyEnabled("candle") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewCandleProjectionActor(CandleProjectionConfig{NATSURL: natsURL, Tracker: tracker})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewEvidenceConsumerActor(EvidenceConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: reg.evidence, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
		{
			Scope:          DomainEvidence,
			Family:         "tradeburst",
			ProjectionName: "trade-burst-projection",
			ConsumerName:   "trade-burst-consumer",
			Buckets:        []string{adapternats.TradeBurstLatestBucket},
			ConsumerSpec:   adapternats.StoreTradeBurstConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsFamilyEnabled("tradeburst") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewTradeBurstProjectionActor(TradeBurstProjectionConfig{NATSURL: natsURL, Tracker: tracker})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewTradeBurstConsumerActor(TradeBurstConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: reg.evidence, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
		{
			Scope:          DomainEvidence,
			Family:         "volume",
			ProjectionName: "volume-projection",
			ConsumerName:   "volume-consumer",
			Buckets:        []string{adapternats.VolumeLatestBucket},
			ConsumerSpec:   adapternats.StoreVolumeConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsFamilyEnabled("volume") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewVolumeProjectionActor(VolumeProjectionConfig{NATSURL: natsURL, Tracker: tracker})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewVolumeConsumerActor(VolumeConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: reg.evidence, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},

		// --- Signal pipelines (opt-in via pipeline.signal_families) ---
		{
			Scope:          DomainSignal,
			Family:         "rsi",
			ProjectionName: "signal-rsi-projection",
			ConsumerName:   "signal-rsi-consumer",
			Buckets:        []string{adapternats.SignalRSILatestBucket},
			ConsumerSpec:   adapternats.StoreRSISignalConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("rsi") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewSignalProjectionActor(SignalProjectionConfig{
					NATSURL: natsURL,
					Bucket:  adapternats.SignalRSILatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewSignalConsumerActor(SignalConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: reg.signal, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},

		{
			Scope:          DomainSignal,
			Family:         "ema_crossover",
			ProjectionName: "signal-ema-crossover-projection",
			ConsumerName:   "signal-ema-crossover-consumer",
			Buckets:        []string{adapternats.SignalEMACrossoverLatestBucket},
			ConsumerSpec:   adapternats.StoreEMACrossoverSignalConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("ema_crossover") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewSignalProjectionActor(SignalProjectionConfig{
					NATSURL: natsURL,
					Bucket:  adapternats.SignalEMACrossoverLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewSignalConsumerActor(SignalConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: reg.signal, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},

		// --- Decision pipelines (opt-in via pipeline.decision_families) ---
		{
			Scope:          DomainDecision,
			Family:         "rsi_oversold",
			ProjectionName: "decision-rsi-oversold-projection",
			ConsumerName:   "decision-rsi-oversold-consumer",
			Buckets:        []string{adapternats.DecisionRSIOversoldLatestBucket},
			ConsumerSpec:   adapternats.StoreRSIOversoldDecisionConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsDecisionFamilyEnabled("rsi_oversold") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewDecisionProjectionActor(DecisionProjectionConfig{
					NATSURL: natsURL,
					Bucket:  adapternats.DecisionRSIOversoldLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewDecisionConsumerActor(DecisionConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: reg.decision, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},

		// --- Strategy pipelines (opt-in via pipeline.strategy_families) ---
		{
			Scope:          DomainStrategy,
			Family:         "mean_reversion_entry",
			ProjectionName: "strategy-mean-reversion-entry-projection",
			ConsumerName:   "strategy-mean-reversion-entry-consumer",
			Buckets:        []string{adapternats.StrategyMeanReversionEntryLatestBucket},
			ConsumerSpec:   adapternats.StoreMeanReversionEntryStrategyConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsStrategyFamilyEnabled("mean_reversion_entry") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewStrategyProjectionActor(StrategyProjectionConfig{
					NATSURL: natsURL,
					Bucket:  adapternats.StrategyMeanReversionEntryLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewStrategyConsumerActor(StrategyConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: reg.strategy, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},

		// --- Risk pipelines (opt-in via pipeline.risk_families) ---
		{
			Scope:          DomainRisk,
			Family:         "position_exposure",
			ProjectionName: "risk-position-exposure-projection",
			ConsumerName:   "risk-position-exposure-consumer",
			Buckets:        []string{adapternats.RiskPositionExposureLatestBucket},
			ConsumerSpec:   adapternats.StorePositionExposureRiskConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsRiskFamilyEnabled("position_exposure") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewRiskProjectionActor(RiskProjectionConfig{
					NATSURL: natsURL,
					Bucket:  adapternats.RiskPositionExposureLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewRiskConsumerActor(RiskConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: reg.risk, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},

		// --- Execution pipelines (opt-in via pipeline.execution_families) ---
		// Paper Family: materializes paper_order intents from derive.
		{
			Scope:          DomainExecution,
			Family:         "paper_order",
			ProjectionName: "execution-paper-order-projection",
			ConsumerName:   "execution-paper-order-consumer",
			Buckets:        []string{adapternats.ExecutionPaperOrderLatestBucket},
			ConsumerSpec:   adapternats.StorePaperOrderExecutionConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsExecutionFamilyEnabled("paper_order") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewExecutionProjectionActor(ExecutionProjectionConfig{
					NATSURL: natsURL,
					Bucket:  adapternats.ExecutionPaperOrderLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewExecutionConsumerActor(ExecutionConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: reg.execution, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
		// Venue Family: materializes venue_market_order fills from execute.
		{
			Scope:          DomainExecution,
			Family:         "venue_market_order",
			ProjectionName: "execution-venue-market-order-projection",
			ConsumerName:   "execution-venue-market-order-consumer",
			Buckets:        []string{adapternats.ExecutionVenueMarketOrderLatestBucket},
			ConsumerSpec:   adapternats.StoreVenueMarketOrderFillConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsExecutionFamilyEnabled("venue_market_order") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewFillProjectionActor(FillProjectionConfig{
					NATSURL:      natsURL,
					Bucket:       adapternats.ExecutionVenueMarketOrderLatestBucket,
					IntentBucket: adapternats.ExecutionPaperOrderLatestBucket,
					Tracker:      tracker,
				})
			},
			NewConsumer: func(natsURL string, spec adapternats.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
				return NewFillConsumerActor(FillConsumerConfig{
					URL: natsURL, ConsumerSpec: spec, Registry: reg.execution, ProjectionPID: projPID, Tracker: tracker,
				})
			},
		},
	}, reg
}

// StoreSupervisor is the root actor for the store binary.
// It materializes domain events into a persistent read model (NATS KV)
// and serves queries from the gateway. Projection pipelines are registered
// declaratively in declarePipelines() — one entry per domain type.
type StoreSupervisor struct {
	cfg      settings.AppConfig
	logger   *slog.Logger
	trackers map[string]*healthz.Tracker
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

	allPipelines, registries := declarePipelines()

	// Filter pipelines by config and spawn enabled ones.
	activeScopes := make(map[PipelineDomain]bool)
	var allBuckets []string
	var enabledFamilies []string
	var durables []string

	for _, p := range allPipelines {
		if !p.IsEnabled(s.cfg.Pipeline) {
			s.logger.Info("pipeline skipped",
				"scope", p.Scope,
				"family", p.Family,
			)
			continue
		}

		projTracker := s.trackers[p.ProjectionName]
		consTracker := s.trackers[p.ConsumerName]

		projPID := ctx.SpawnChild(p.NewProjection(s.cfg.NATS.URL, projTracker), p.ProjectionName)
		ctx.SpawnChild(p.NewConsumer(s.cfg.NATS.URL, p.ConsumerSpec, projPID, consTracker), p.ConsumerName)

		allBuckets = append(allBuckets, p.Buckets...)
		enabledFamilies = append(enabledFamilies, string(p.Scope)+"/"+p.Family)
		durables = append(durables, p.ConsumerSpec.Durable)
		activeScopes[p.Scope] = true
	}

	if len(enabledFamilies) == 0 {
		return fmt.Errorf("no projection pipelines enabled — check pipeline config")
	}

	// Spawn query responder with registries for enabled scopes.
	qrCfg := registries.queryResponderConfig(s.cfg.NATS.URL, activeScopes)
	ctx.SpawnChild(NewQueryResponderActor(qrCfg), "query-responder")

	activationMode := "all (no pipeline.families configured)"
	if s.cfg.Pipeline.EnabledFamilies() != nil {
		activationMode = "config-driven"
	}
	s.logger.Info("store supervisor started",
		"activation", activationMode,
		"pipelines", enabledFamilies,
		"consumers", durables,
		"buckets", allBuckets,
	)
	return nil
}
