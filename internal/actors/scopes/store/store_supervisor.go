package store

import (
	"fmt"
	"io"
	"log/slog"

	actorcommon "internal/actors/common"
	natsdecision "internal/adapters/nats/natsdecision"
	natsevidence "internal/adapters/nats/natsevidence"
	natsexecution "internal/adapters/nats/natsexecution"
	natsinsights "internal/adapters/nats/natsinsights"
	natskit "internal/adapters/nats/natskit"
	natsrisk "internal/adapters/nats/natsrisk"
	natssignal "internal/adapters/nats/natssignal"
	natsstrategy "internal/adapters/nats/natsstrategy"
	"internal/domain/decision"
	"internal/domain/evidence"
	domainexec "internal/domain/execution"
	"internal/domain/insights"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
	"internal/shared/clock"
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
	DomainInsights  PipelineDomain = "insights"
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
	ConsumerSpec natskit.ConsumerSpec
	// IsEnabled reports whether this pipeline should be spawned given current config.
	IsEnabled func(settings.PipelineConfig) bool
	// NewProjection creates the projection actor for this pipeline.
	NewProjection func(natsURL string, tracker *healthz.Tracker) actor.Producer
	// NewConsumer creates the consumer actor for this pipeline.
	// The registry is already bound via closure at declaration time.
	NewConsumer func(natsURL string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer
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
	evidence  natsevidence.Registry
	signal    natssignal.Registry
	decision  natsdecision.Registry
	strategy  natsstrategy.Registry
	risk      natsrisk.Registry
	execution natsexecution.Registry
}

// queryResponderConfig builds the QueryResponderConfig with registries for enabled scopes.
func (r pipelineRegistries) queryResponderConfig(natsURL string, activeScopes map[PipelineDomain]bool, clk clock.Clock) QueryResponderConfig {
	cfg := QueryResponderConfig{
		NATSURL:  natsURL,
		Source:   "store.query-responder",
		Registry: r.evidence,
		Clock:    clk,
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
		evidence:  natsevidence.DefaultRegistry(),
		signal:    natssignal.DefaultRegistry(),
		decision:  natsdecision.DefaultRegistry(),
		strategy:  natsstrategy.DefaultRegistry(),
		risk:      natsrisk.DefaultRegistry(),
		execution: natsexecution.DefaultRegistry(),
	}

	// Insights registry is local: insights are read from KV directly
	// by the gateway (a free KV reader per ADR-0008), so the registry
	// does not flow into the query responder (PROGRAM-0005 / H-8.a).
	insReg := natsinsights.DefaultRegistry()

	// startConsumer wires a ConsumerStartFn closure into NewGenericConsumerActor,
	// eliminating the need for per-domain consumer actor types. The registry and
	// message routing are captured via closure at declaration time.
	startConsumer := func(family string, fn ConsumerStartFn) func(string, natskit.ConsumerSpec, *actor.PID, *healthz.Tracker) actor.Producer {
		return func(natsURL string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker) actor.Producer {
			return NewGenericConsumerActor(GenericConsumerConfig{
				URL:           natsURL,
				ConsumerSpec:  spec,
				ProjectionPID: projPID,
				Tracker:       tracker,
				Family:        family,
				StartFn:       fn,
			})
		}
	}

	return []Pipeline{
		// --- Evidence pipelines (backward-compatible default) ---
		{
			Scope:          DomainEvidence,
			Family:         "candle",
			ProjectionName: "candle-projection",
			ConsumerName:   "candle-consumer",
			Buckets:        []string{natsevidence.CandleLatestBucket, natsevidence.CandleHistoryBucket},
			ConsumerSpec:   natsevidence.StoreCandleConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsFamilyEnabled("candle") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewCandleProjectionActor(CandleProjectionConfig{NATSURL: natsURL, Tracker: tracker})
			},
			NewConsumer: startConsumer("candle", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsevidence.NewCandleConsumer(url, spec, reg.evidence, func(event evidence.CandleSampledEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, candleReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},
		{
			Scope:          DomainEvidence,
			Family:         "tradeburst",
			ProjectionName: "trade-burst-projection",
			ConsumerName:   "trade-burst-consumer",
			Buckets:        []string{natsevidence.TradeBurstLatestBucket},
			ConsumerSpec:   natsevidence.StoreTradeBurstConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsFamilyEnabled("tradeburst") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewTradeBurstProjectionActor(TradeBurstProjectionConfig{NATSURL: natsURL, Tracker: tracker})
			},
			NewConsumer: startConsumer("tradeburst", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsevidence.NewTradeBurstConsumer(url, spec, reg.evidence, func(event evidence.TradeBurstSampledEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, tradeBurstReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},
		{
			Scope:          DomainEvidence,
			Family:         "volume",
			ProjectionName: "volume-projection",
			ConsumerName:   "volume-consumer",
			Buckets:        []string{natsevidence.VolumeLatestBucket},
			ConsumerSpec:   natsevidence.StoreVolumeConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsFamilyEnabled("volume") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewVolumeProjectionActor(VolumeProjectionConfig{NATSURL: natsURL, Tracker: tracker})
			},
			NewConsumer: startConsumer("volume", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsevidence.NewVolumeConsumer(url, spec, reg.evidence, func(event evidence.VolumeSampledEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, volumeReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},

		// --- Insights pipeline (PROGRAM-0005 / H-8.a; always on — read-only descriptive overlay) ---
		{
			Scope:          DomainInsights,
			Family:         "volume_profile",
			ProjectionName: "volume-profile-projection",
			ConsumerName:   "volume-profile-consumer",
			Buckets:        []string{natsinsights.VolumeProfileLatestBucket},
			ConsumerSpec:   natsinsights.StoreVolumeProfileConsumer(),
			IsEnabled:      func(settings.PipelineConfig) bool { return true },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewVolumeProfileProjectionActor(VolumeProfileProjectionConfig{NATSURL: natsURL, Tracker: tracker})
			},
			NewConsumer: startConsumer("volume_profile", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsinsights.NewVolumeProfileConsumer(url, spec, insReg, func(event insights.VolumeProfileSampledEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, volumeProfileReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},

		// --- Signal pipelines (opt-in via pipeline.signal_families) ---
		{
			Scope:          DomainSignal,
			Family:         "rsi",
			ProjectionName: "signal-rsi-projection",
			ConsumerName:   "signal-rsi-consumer",
			Buckets:        []string{natssignal.RSILatestBucket},
			ConsumerSpec:   natssignal.StoreRSISignalConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("rsi") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewSignalProjectionActor(SignalProjectionConfig{
					NATSURL: natsURL,
					Bucket:  natssignal.RSILatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: startConsumer("rsi", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natssignal.NewConsumer(url, spec, reg.signal, func(event signal.SignalGeneratedEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, signalReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},

		{
			Scope:          DomainSignal,
			Family:         "ema_crossover",
			ProjectionName: "signal-ema-crossover-projection",
			ConsumerName:   "signal-ema-crossover-consumer",
			Buckets:        []string{natssignal.EMACrossoverLatestBucket},
			ConsumerSpec:   natssignal.StoreEMACrossoverSignalConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsSignalFamilyEnabled("ema_crossover") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewSignalProjectionActor(SignalProjectionConfig{
					NATSURL: natsURL,
					Bucket:  natssignal.EMACrossoverLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: startConsumer("ema_crossover", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natssignal.NewConsumer(url, spec, reg.signal, func(event signal.SignalGeneratedEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, signalReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},

		// --- Decision pipelines (opt-in via pipeline.decision_families) ---
		{
			Scope:          DomainDecision,
			Family:         "rsi_oversold",
			ProjectionName: "decision-rsi-oversold-projection",
			ConsumerName:   "decision-rsi-oversold-consumer",
			Buckets:        []string{natsdecision.RSIOversoldLatestBucket},
			ConsumerSpec:   natsdecision.StoreRSIOversoldDecisionConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsDecisionFamilyEnabled("rsi_oversold") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewDecisionProjectionActor(DecisionProjectionConfig{
					NATSURL: natsURL,
					Bucket:  natsdecision.RSIOversoldLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: startConsumer("rsi_oversold", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsdecision.NewConsumer(url, spec, reg.decision, func(event decision.DecisionEvaluatedEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, decisionReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},

		{
			Scope:          DomainDecision,
			Family:         "ema_crossover",
			ProjectionName: "decision-ema-crossover-projection",
			ConsumerName:   "decision-ema-crossover-consumer",
			Buckets:        []string{natsdecision.EMACrossoverLatestBucket},
			ConsumerSpec:   natsdecision.StoreEMACrossoverDecisionConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsDecisionFamilyEnabled("ema_crossover") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewDecisionProjectionActor(DecisionProjectionConfig{
					NATSURL: natsURL,
					Bucket:  natsdecision.EMACrossoverLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: startConsumer("ema_crossover", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsdecision.NewConsumer(url, spec, reg.decision, func(event decision.DecisionEvaluatedEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, decisionReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},

		// --- Strategy pipelines (opt-in via pipeline.strategy_families) ---
		{
			Scope:          DomainStrategy,
			Family:         "mean_reversion_entry",
			ProjectionName: "strategy-mean-reversion-entry-projection",
			ConsumerName:   "strategy-mean-reversion-entry-consumer",
			Buckets:        []string{natsstrategy.MeanReversionEntryLatestBucket},
			ConsumerSpec:   natsstrategy.StoreMeanReversionEntryStrategyConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsStrategyFamilyEnabled("mean_reversion_entry") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewStrategyProjectionActor(StrategyProjectionConfig{
					NATSURL: natsURL,
					Bucket:  natsstrategy.MeanReversionEntryLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: startConsumer("mean_reversion_entry", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsstrategy.NewConsumer(url, spec, reg.strategy, func(event strategy.StrategyResolvedEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, strategyReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},

		{
			Scope:          DomainStrategy,
			Family:         "trend_following_entry",
			ProjectionName: "strategy-trend-following-entry-projection",
			ConsumerName:   "strategy-trend-following-entry-consumer",
			Buckets:        []string{natsstrategy.TrendFollowingEntryLatestBucket},
			ConsumerSpec:   natsstrategy.StoreTrendFollowingEntryStrategyConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsStrategyFamilyEnabled("trend_following_entry") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewStrategyProjectionActor(StrategyProjectionConfig{
					NATSURL: natsURL,
					Bucket:  natsstrategy.TrendFollowingEntryLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: startConsumer("trend_following_entry", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsstrategy.NewConsumer(url, spec, reg.strategy, func(event strategy.StrategyResolvedEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, strategyReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},

		// --- Risk pipelines (opt-in via pipeline.risk_families) ---
		{
			Scope:          DomainRisk,
			Family:         "position_exposure",
			ProjectionName: "risk-position-exposure-projection",
			ConsumerName:   "risk-position-exposure-consumer",
			Buckets:        []string{natsrisk.PositionExposureLatestBucket},
			ConsumerSpec:   natsrisk.StorePositionExposureRiskConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsRiskFamilyEnabled("position_exposure") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewRiskProjectionActor(RiskProjectionConfig{
					NATSURL: natsURL,
					Bucket:  natsrisk.PositionExposureLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: startConsumer("position_exposure", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsrisk.NewConsumer(url, spec, reg.risk, func(event risk.RiskAssessedEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, riskReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},

		// Risk Family: drawdown_limit — materializes drawdown limit risk from derive.
		{
			Scope:          DomainRisk,
			Family:         "drawdown_limit",
			ProjectionName: "risk-drawdown-limit-projection",
			ConsumerName:   "risk-drawdown-limit-consumer",
			Buckets:        []string{natsrisk.DrawdownLimitLatestBucket},
			ConsumerSpec:   natsrisk.StoreDrawdownLimitRiskConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsRiskFamilyEnabled("drawdown_limit") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewRiskProjectionActor(RiskProjectionConfig{
					NATSURL: natsURL,
					Bucket:  natsrisk.DrawdownLimitLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: startConsumer("drawdown_limit", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsrisk.NewConsumer(url, spec, reg.risk, func(event risk.RiskAssessedEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, riskReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},

		// --- Execution pipelines (opt-in via pipeline.execution_families) ---
		// Paper Family: materializes paper_order intents from derive.
		{
			Scope:          DomainExecution,
			Family:         "paper_order",
			ProjectionName: "execution-paper-order-projection",
			ConsumerName:   "execution-paper-order-consumer",
			Buckets:        []string{natsexecution.PaperOrderLatestBucket},
			ConsumerSpec:   natsexecution.StorePaperOrderExecutionConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsExecutionFamilyEnabled("paper_order") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewExecutionProjectionActor(ExecutionProjectionConfig{
					NATSURL: natsURL,
					Bucket:  natsexecution.PaperOrderLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: startConsumer("paper_order", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsexecution.NewConsumer(url, spec, reg.execution, func(event domainexec.PaperOrderSubmittedEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, executionReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},
		// Venue Family: materializes venue_market_order fills from execute.
		{
			Scope:          DomainExecution,
			Family:         "venue_market_order",
			ProjectionName: "execution-venue-market-order-projection",
			ConsumerName:   "execution-venue-market-order-consumer",
			Buckets:        []string{natsexecution.VenueMarketOrderLatestBucket},
			ConsumerSpec:   natsexecution.StoreVenueMarketOrderFillConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsExecutionFamilyEnabled("venue_market_order") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewFillProjectionActor(FillProjectionConfig{
					NATSURL:      natsURL,
					Bucket:       natsexecution.VenueMarketOrderLatestBucket,
					IntentBucket: natsexecution.PaperOrderLatestBucket,
					Tracker:      tracker,
				})
			},
			NewConsumer: startConsumer("venue_market_order", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsexecution.NewFillConsumer(url, spec, reg.execution, func(event domainexec.VenueOrderFilledEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, fillReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
		},
		// S387: Venue rejection family — materializes venue_market_order rejections from execute.
		// Paired with venue_market_order fills to close the lifecycle persistence gap.
		// Enabled by the same "venue_market_order" family flag — rejections are part of the venue outcome.
		{
			Scope:          DomainExecution,
			Family:         "venue_rejection",
			ProjectionName: "execution-venue-rejection-projection",
			ConsumerName:   "execution-venue-rejection-consumer",
			Buckets:        []string{natsexecution.VenueRejectionLatestBucket},
			ConsumerSpec:   natsexecution.StoreVenueMarketOrderRejectionConsumer(),
			IsEnabled:      func(p settings.PipelineConfig) bool { return p.IsExecutionFamilyEnabled("venue_market_order") },
			NewProjection: func(natsURL string, tracker *healthz.Tracker) actor.Producer {
				return NewRejectionProjectionActor(RejectionProjectionConfig{
					NATSURL: natsURL,
					Bucket:  natsexecution.VenueRejectionLatestBucket,
					Tracker: tracker,
				})
			},
			NewConsumer: startConsumer("venue_rejection", func(url string, spec natskit.ConsumerSpec, projPID *actor.PID, tracker *healthz.Tracker, actorCtx *actor.Context, logger *slog.Logger) (io.Closer, error) {
				c := natsexecution.NewRejectionConsumer(url, spec, reg.execution, func(event domainexec.VenueOrderRejectedEvent) {
					if tracker != nil {
						tracker.RecordEvent()
					}
					actorCtx.Send(projPID, rejectionReceivedMessage{Event: event})
				}, logger)
				return c, c.Start()
			}),
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
	// H-4: clk is the time port threaded through to
	// QueryResponderConfig and (in commit 6c) to
	// NewActivationSurface call sites. Defaults to
	// clock.SystemClock{} when no SupervisorOption supplies one.
	clk clock.Clock
}

// SupervisorOption configures optional parameters on the StoreSupervisor.
type SupervisorOption func(*StoreSupervisor)

// WithClock sets the time port the supervisor threads through to
// child actor configs. Optional; defaults to clock.SystemClock{}.
func WithClock(clk clock.Clock) SupervisorOption {
	return func(s *StoreSupervisor) {
		if clk != nil {
			s.clk = clk
		}
	}
}

func NewStoreSupervisor(config settings.AppConfig, trackers map[string]*healthz.Tracker, opts ...SupervisorOption) actor.Producer {
	return func() actor.Receiver {
		s := &StoreSupervisor{
			cfg:      config,
			logger:   slog.Default().With("actor", "store-supervisor"),
			trackers: trackers,
			clk:      clock.SystemClock{},
		}
		for _, opt := range opts {
			opt(s)
		}
		return s
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
	qrCfg := registries.queryResponderConfig(s.cfg.NATS.URL, activeScopes, s.clk)
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
