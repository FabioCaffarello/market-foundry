package store

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	actorcommon "internal/actors/common"
	natsdecision "internal/adapters/nats/natsdecision"
	natsevidence "internal/adapters/nats/natsevidence"
	natsexecution "internal/adapters/nats/natsexecution"
	natskit "internal/adapters/nats/natskit"
	natsrisk "internal/adapters/nats/natsrisk"
	natssignal "internal/adapters/nats/natssignal"
	natsstrategy "internal/adapters/nats/natsstrategy"
	"internal/application/decisionclient"
	"internal/application/evidenceclient"
	"internal/application/executionclient"
	"internal/application/riskclient"
	"internal/application/signalclient"
	"internal/application/strategyclient"
	"internal/domain/evidence"
	"internal/domain/execution"
	"internal/shared/clock"
	"internal/shared/problem"

	"github.com/anthdm/hollywood/actor"
)

// QueryResponderConfig holds the configuration for the query responder actor.
type QueryResponderConfig struct {
	NATSURL           string
	Source            string
	Registry          natsevidence.Registry
	SignalRegistry    *natssignal.Registry    // nil when no signal families are enabled
	DecisionRegistry  *natsdecision.Registry  // nil when no decision families are enabled
	StrategyRegistry  *natsstrategy.Registry  // nil when no strategy families are enabled
	RiskRegistry      *natsrisk.Registry      // nil when no risk families are enabled
	ExecutionRegistry *natsexecution.Registry // nil when no execution families are enabled
	// H-4: Clock is the time port for sourcing wall-clock instants
	// (e.g., the activation surface ObservedAt when composing
	// NewActivationSurface in cross-family queries). When nil, the
	// actor falls back to clock.SystemClock{}. Not consumed in
	// this commit — call sites land in commit 6c.
	Clock clock.Clock
}

// QueryResponderActor serves evidence and signal queries from the NATS KV stores.
// It reads the materialized projections — no dependency on derive actors.
type QueryResponderActor struct {
	cfg                             QueryResponderConfig
	logger                          *slog.Logger
	store                           *natsevidence.CandleKVStore
	burstStore                      *natsevidence.TradeBurstKVStore
	volumeStore                     *natsevidence.VolumeKVStore
	signalRSIStore                  *natssignal.KVStore
	decisionRSIOversoldStore        *natsdecision.KVStore
	strategyMeanReversionEntryStore *natsstrategy.KVStore
	riskPositionExposureStore       *natsrisk.KVStore
	executionPaperOrderStore        *natsexecution.KVStore
	executionVenueMarketOrderStore  *natsexecution.KVStore
	executionVenueRejectionStore    *natsexecution.KVStore // S387: rejection read model
	executionControlStore           *natsexecution.ControlKVStore
	sessionStore                    *natsexecution.SessionKVStore // S460: session metadata
	responder                       *natskit.RequestReplyResponder
}

func NewQueryResponderActor(cfg QueryResponderConfig) actor.Producer {
	return func() actor.Receiver {
		return &QueryResponderActor{cfg: cfg}
	}
}

func (a *QueryResponderActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "query-responder")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		if a.responder != nil {
			if err := a.responder.Close(); err != nil {
				a.logger.Error("close query responder", "error", err)
			}
		}
		if a.store != nil {
			if err := a.store.Close(); err != nil {
				a.logger.Error("close query KV store", "error", err)
			}
		}
		if a.burstStore != nil {
			if err := a.burstStore.Close(); err != nil {
				a.logger.Error("close trade burst query KV store", "error", err)
			}
		}
		if a.volumeStore != nil {
			if err := a.volumeStore.Close(); err != nil {
				a.logger.Error("close volume query KV store", "error", err)
			}
		}
		if a.signalRSIStore != nil {
			if err := a.signalRSIStore.Close(); err != nil {
				a.logger.Error("close signal RSI query KV store", "error", err)
			}
		}
		if a.decisionRSIOversoldStore != nil {
			if err := a.decisionRSIOversoldStore.Close(); err != nil {
				a.logger.Error("close decision RSI oversold query KV store", "error", err)
			}
		}
		if a.strategyMeanReversionEntryStore != nil {
			if err := a.strategyMeanReversionEntryStore.Close(); err != nil {
				a.logger.Error("close strategy mean reversion entry query KV store", "error", err)
			}
		}
		if a.riskPositionExposureStore != nil {
			if err := a.riskPositionExposureStore.Close(); err != nil {
				a.logger.Error("close risk position exposure query KV store", "error", err)
			}
		}
		if a.executionPaperOrderStore != nil {
			if err := a.executionPaperOrderStore.Close(); err != nil {
				a.logger.Error("close execution paper order query KV store", "error", err)
			}
		}
		if a.executionVenueMarketOrderStore != nil {
			if err := a.executionVenueMarketOrderStore.Close(); err != nil {
				a.logger.Error("close execution venue market order query KV store", "error", err)
			}
		}
		if a.executionVenueRejectionStore != nil {
			if err := a.executionVenueRejectionStore.Close(); err != nil {
				a.logger.Error("close execution venue rejection query KV store", "error", err)
			}
		}
		if a.executionControlStore != nil {
			if err := a.executionControlStore.Close(); err != nil {
				a.logger.Error("close execution control query KV store", "error", err)
			}
		}
		if a.sessionStore != nil {
			if err := a.sessionStore.Close(); err != nil {
				a.logger.Error("close session KV store", "error", err)
			}
		}

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *QueryResponderActor) start(c *actor.Context) {
	// Open a read-only KV store connection for candle queries.
	store := natsevidence.NewCandleKVStore(a.cfg.NATSURL)
	if err := store.Start(); err != nil {
		a.logger.Error("start query KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}
	a.store = store

	// Open a read-only KV store connection for trade burst queries.
	burstStore := natsevidence.NewTradeBurstKVStore(a.cfg.NATSURL)
	if err := burstStore.Start(); err != nil {
		a.logger.Error("start trade burst query KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}
	a.burstStore = burstStore

	// Open a read-only KV store connection for volume queries.
	volumeStore := natsevidence.NewVolumeKVStore(a.cfg.NATSURL)
	if err := volumeStore.Start(); err != nil {
		a.logger.Error("start volume query KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}
	a.volumeStore = volumeStore

	routes := []natskit.ControlRoute{
		natskit.NewTypedControlRoute(
			a.cfg.Registry.CandleLatest,
			a.cfg.Source,
			a.handleCandleLatest,
		),
		natskit.NewTypedControlRoute(
			a.cfg.Registry.CandleHistory,
			a.cfg.Source,
			a.handleCandleHistory,
		),
		natskit.NewTypedControlRoute(
			a.cfg.Registry.TradeBurstLatest,
			a.cfg.Source,
			a.handleTradeBurstLatest,
		),
		natskit.NewTypedControlRoute(
			a.cfg.Registry.VolumeLatest,
			a.cfg.Source,
			a.handleVolumeLatest,
		),
	}

	// Wire signal query routes if signal families are enabled.
	if a.cfg.SignalRegistry != nil {
		sigStore := natssignal.NewKVStore(a.cfg.NATSURL, natssignal.RSILatestBucket)
		if err := sigStore.Start(); err != nil {
			a.logger.Error("start signal RSI query KV store", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.signalRSIStore = sigStore

		routes = append(routes, natskit.NewTypedControlRoute(
			a.cfg.SignalRegistry.RSILatest,
			a.cfg.Source,
			a.handleSignalRSILatest,
		))
	}

	// Wire decision query routes if decision families are enabled.
	if a.cfg.DecisionRegistry != nil {
		decStore := natsdecision.NewKVStore(a.cfg.NATSURL, natsdecision.RSIOversoldLatestBucket)
		if err := decStore.Start(); err != nil {
			a.logger.Error("start decision RSI oversold query KV store", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.decisionRSIOversoldStore = decStore

		routes = append(routes, natskit.NewTypedControlRoute(
			a.cfg.DecisionRegistry.RSIOversoldLatest,
			a.cfg.Source,
			a.handleDecisionRSIOversoldLatest,
		))
	}

	// Wire strategy query routes if strategy families are enabled.
	if a.cfg.StrategyRegistry != nil {
		stratStore := natsstrategy.NewKVStore(a.cfg.NATSURL, natsstrategy.MeanReversionEntryLatestBucket)
		if err := stratStore.Start(); err != nil {
			a.logger.Error("start strategy mean reversion entry query KV store", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.strategyMeanReversionEntryStore = stratStore

		routes = append(routes, natskit.NewTypedControlRoute(
			a.cfg.StrategyRegistry.MeanReversionEntryLatest,
			a.cfg.Source,
			a.handleStrategyMeanReversionEntryLatest,
		))
	}

	// Wire risk query routes if risk families are enabled.
	if a.cfg.RiskRegistry != nil {
		riskStore := natsrisk.NewKVStore(a.cfg.NATSURL, natsrisk.PositionExposureLatestBucket)
		if err := riskStore.Start(); err != nil {
			a.logger.Error("start risk position exposure query KV store", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.riskPositionExposureStore = riskStore

		routes = append(routes, natskit.NewTypedControlRoute(
			a.cfg.RiskRegistry.PositionExposureLatest,
			a.cfg.Source,
			a.handleRiskPositionExposureLatest,
		))
	}

	// Wire execution query routes if execution families are enabled.
	if a.cfg.ExecutionRegistry != nil {
		execStore := natsexecution.NewKVStore(a.cfg.NATSURL, natsexecution.PaperOrderLatestBucket)
		if err := execStore.Start(); err != nil {
			a.logger.Error("start execution paper order query KV store", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.executionPaperOrderStore = execStore

		routes = append(routes, natskit.NewTypedControlRoute(
			a.cfg.ExecutionRegistry.PaperOrderLatest,
			a.cfg.Source,
			a.handleExecutionPaperOrderLatest,
		))

		// Open a read-only KV store connection for venue market order fill queries.
		venueStore := natsexecution.NewKVStore(a.cfg.NATSURL, natsexecution.VenueMarketOrderLatestBucket)
		if err := venueStore.Start(); err != nil {
			a.logger.Error("start execution venue market order query KV store", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.executionVenueMarketOrderStore = venueStore

		routes = append(routes, natskit.NewTypedControlRoute(
			a.cfg.ExecutionRegistry.VenueMarketOrderLatest,
			a.cfg.Source,
			a.handleExecutionVenueMarketOrderLatest,
		))

		// S387: Open a read-only KV store connection for venue rejection queries.
		rejectionStore := natsexecution.NewKVStore(a.cfg.NATSURL, natsexecution.VenueRejectionLatestBucket)
		if err := rejectionStore.Start(); err != nil {
			// Best-effort: rejection store unavailability does not prevent startup.
			a.logger.Warn("execution venue rejection query KV store unavailable — rejection read-path degraded", "error", err)
		} else {
			a.executionVenueRejectionStore = rejectionStore

			// S407: Dedicated rejection query route — makes rejection audit detail
			// queryable independently from the composite status endpoint.
			routes = append(routes, natskit.NewTypedControlRoute(
				a.cfg.ExecutionRegistry.VenueRejectionLatest,
				a.cfg.Source,
				a.handleExecutionVenueRejectionLatest,
			))
		}

		// Wire composite status query (reads all KV stores + control).
		routes = append(routes, natskit.NewTypedControlRoute(
			a.cfg.ExecutionRegistry.StatusLatest,
			a.cfg.Source,
			a.handleExecutionStatusLatest,
		))

		// S413: Wire lifecycle list query (enumerates all tracked partition keys).
		routes = append(routes, natskit.NewTypedControlRoute(
			a.cfg.ExecutionRegistry.LifecycleList,
			a.cfg.Source,
			a.handleExecutionLifecycleList,
		))

		// Wire execution control gate (get + set).
		controlStore := natsexecution.NewControlKVStore(a.cfg.NATSURL)
		if err := controlStore.Start(); err != nil {
			a.logger.Error("start execution control KV store", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.executionControlStore = controlStore

		routes = append(routes,
			natskit.NewTypedControlRoute(
				a.cfg.ExecutionRegistry.ControlGet,
				a.cfg.Source,
				a.handleExecutionControlGet,
			),
			natskit.NewTypedControlRoute(
				a.cfg.ExecutionRegistry.ControlSet,
				a.cfg.Source,
				a.handleExecutionControlSet,
			),
			natskit.NewTypedControlRoute(
				a.cfg.ExecutionRegistry.ActivationSurfaceGet,
				a.cfg.Source,
				a.handleActivationSurfaceGet,
			),
		)

		// S460: Wire session metadata query routes.
		sessStore := natsexecution.NewSessionKVStore(a.cfg.NATSURL)
		if err := sessStore.Start(); err != nil {
			a.logger.Warn("session KV store unavailable — session queries degraded", "error", err)
		} else {
			a.sessionStore = sessStore
			routes = append(routes,
				natskit.NewTypedControlRoute(
					a.cfg.ExecutionRegistry.SessionGet,
					a.cfg.Source,
					a.handleSessionGet,
				),
				natskit.NewTypedControlRoute(
					a.cfg.ExecutionRegistry.SessionList,
					a.cfg.Source,
					a.handleSessionList,
				),
			)
		}
	}

	responder := natskit.NewRequestReplyResponder(a.cfg.NATSURL, routes)
	if err := responder.Start(); err != nil {
		a.logger.Error("start query responder", "error", err)
		c.Engine().Poison(c.PID())
		return
	}

	a.responder = responder

	logFields := []any{
		"subject_latest", a.cfg.Registry.CandleLatest.Subject,
		"subject_history", a.cfg.Registry.CandleHistory.Subject,
		"subject_trade_burst_latest", a.cfg.Registry.TradeBurstLatest.Subject,
		"subject_volume_latest", a.cfg.Registry.VolumeLatest.Subject,
		"bucket_latest", natsevidence.CandleLatestBucket,
		"bucket_history", natsevidence.CandleHistoryBucket,
		"bucket_trade_burst_latest", natsevidence.TradeBurstLatestBucket,
		"bucket_volume_latest", natsevidence.VolumeLatestBucket,
	}
	if a.cfg.SignalRegistry != nil {
		logFields = append(logFields,
			"subject_signal_rsi_latest", a.cfg.SignalRegistry.RSILatest.Subject,
			"bucket_signal_rsi_latest", natssignal.RSILatestBucket,
		)
	}
	if a.cfg.DecisionRegistry != nil {
		logFields = append(logFields,
			"subject_decision_rsi_oversold_latest", a.cfg.DecisionRegistry.RSIOversoldLatest.Subject,
			"bucket_decision_rsi_oversold_latest", natsdecision.RSIOversoldLatestBucket,
		)
	}
	if a.cfg.StrategyRegistry != nil {
		logFields = append(logFields,
			"subject_strategy_mean_reversion_entry_latest", a.cfg.StrategyRegistry.MeanReversionEntryLatest.Subject,
			"bucket_strategy_mean_reversion_entry_latest", natsstrategy.MeanReversionEntryLatestBucket,
		)
	}
	if a.cfg.RiskRegistry != nil {
		logFields = append(logFields,
			"subject_risk_position_exposure_latest", a.cfg.RiskRegistry.PositionExposureLatest.Subject,
			"bucket_risk_position_exposure_latest", natsrisk.PositionExposureLatestBucket,
		)
	}
	if a.cfg.ExecutionRegistry != nil {
		logFields = append(logFields,
			"subject_execution_paper_order_latest", a.cfg.ExecutionRegistry.PaperOrderLatest.Subject,
			"bucket_execution_paper_order_latest", natsexecution.PaperOrderLatestBucket,
			"subject_execution_venue_market_order_latest", a.cfg.ExecutionRegistry.VenueMarketOrderLatest.Subject,
			"bucket_execution_venue_market_order_latest", natsexecution.VenueMarketOrderLatestBucket,
			"bucket_execution_venue_rejection_latest", natsexecution.VenueRejectionLatestBucket,
			"subject_execution_venue_rejection_latest", a.cfg.ExecutionRegistry.VenueRejectionLatest.Subject,
			"rejection_store_available", a.executionVenueRejectionStore != nil,
			"subject_execution_status_latest", a.cfg.ExecutionRegistry.StatusLatest.Subject,
			"subject_execution_lifecycle_list", a.cfg.ExecutionRegistry.LifecycleList.Subject,
			"subject_execution_control_get", a.cfg.ExecutionRegistry.ControlGet.Subject,
			"subject_execution_control_set", a.cfg.ExecutionRegistry.ControlSet.Subject,
			"subject_activation_surface_get", a.cfg.ExecutionRegistry.ActivationSurfaceGet.Subject,
			"bucket_execution_control", natsexecution.ControlBucket,
			"session_store_available", a.sessionStore != nil,
			"bucket_session", natsexecution.SessionBucket,
		)
	}
	a.logger.Info("query responder started", logFields...)
}

func (a *QueryResponderActor) handleCandleLatest(ctx context.Context, query evidenceclient.CandleLatestQuery) (evidenceclient.CandleLatestReply, *problem.Problem) {
	candle, prob := a.store.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return evidenceclient.CandleLatestReply{}, prob
	}

	return evidenceclient.CandleLatestReply{Candle: candle}, nil
}

func (a *QueryResponderActor) handleTradeBurstLatest(ctx context.Context, query evidenceclient.TradeBurstLatestQuery) (evidenceclient.TradeBurstLatestReply, *problem.Problem) {
	burst, prob := a.burstStore.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return evidenceclient.TradeBurstLatestReply{}, prob
	}

	return evidenceclient.TradeBurstLatestReply{TradeBurst: burst}, nil
}

func (a *QueryResponderActor) handleCandleHistory(ctx context.Context, query evidenceclient.CandleHistoryQuery) (evidenceclient.CandleHistoryReply, *problem.Problem) {
	candles, prob := a.store.GetHistory(ctx, query.Source, query.Instrument, query.Timeframe, query.Limit, query.Since, query.Until)
	if prob != nil {
		return evidenceclient.CandleHistoryReply{}, prob
	}

	if candles == nil {
		candles = []evidence.EvidenceCandle{}
	}

	return evidenceclient.CandleHistoryReply{Candles: candles}, nil
}

func (a *QueryResponderActor) handleVolumeLatest(ctx context.Context, query evidenceclient.VolumeLatestQuery) (evidenceclient.VolumeLatestReply, *problem.Problem) {
	vol, prob := a.volumeStore.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return evidenceclient.VolumeLatestReply{}, prob
	}

	return evidenceclient.VolumeLatestReply{Volume: vol}, nil
}

func (a *QueryResponderActor) handleSignalRSILatest(ctx context.Context, query signalclient.SignalLatestQuery) (signalclient.SignalLatestReply, *problem.Problem) {
	sig, prob := a.signalRSIStore.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return signalclient.SignalLatestReply{}, prob
	}

	return signalclient.SignalLatestReply{Signal: sig}, nil
}

func (a *QueryResponderActor) handleDecisionRSIOversoldLatest(ctx context.Context, query decisionclient.DecisionLatestQuery) (decisionclient.DecisionLatestReply, *problem.Problem) {
	dec, prob := a.decisionRSIOversoldStore.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return decisionclient.DecisionLatestReply{}, prob
	}

	return decisionclient.DecisionLatestReply{Decision: dec}, nil
}

func (a *QueryResponderActor) handleStrategyMeanReversionEntryLatest(ctx context.Context, query strategyclient.StrategyLatestQuery) (strategyclient.StrategyLatestReply, *problem.Problem) {
	strat, prob := a.strategyMeanReversionEntryStore.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return strategyclient.StrategyLatestReply{}, prob
	}

	return strategyclient.StrategyLatestReply{Strategy: strat}, nil
}

func (a *QueryResponderActor) handleRiskPositionExposureLatest(ctx context.Context, query riskclient.RiskLatestQuery) (riskclient.RiskLatestReply, *problem.Problem) {
	assessment, prob := a.riskPositionExposureStore.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return riskclient.RiskLatestReply{}, prob
	}

	return riskclient.RiskLatestReply{RiskAssessment: assessment}, nil
}

func (a *QueryResponderActor) handleExecutionPaperOrderLatest(ctx context.Context, query executionclient.ExecutionLatestQuery) (executionclient.ExecutionLatestReply, *problem.Problem) {
	intent, prob := a.executionPaperOrderStore.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return executionclient.ExecutionLatestReply{}, prob
	}

	return executionclient.ExecutionLatestReply{ExecutionIntent: intent}, nil
}

func (a *QueryResponderActor) handleExecutionVenueMarketOrderLatest(ctx context.Context, query executionclient.ExecutionLatestQuery) (executionclient.ExecutionLatestReply, *problem.Problem) {
	intent, prob := a.executionVenueMarketOrderStore.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return executionclient.ExecutionLatestReply{}, prob
	}

	return executionclient.ExecutionLatestReply{ExecutionIntent: intent}, nil
}

// S407: Dedicated rejection query handler — returns intent + rejection audit detail.
func (a *QueryResponderActor) handleExecutionVenueRejectionLatest(ctx context.Context, query executionclient.ExecutionLatestQuery) (executionclient.ExecutionRejectionReply, *problem.Problem) {
	intent, prob := a.executionVenueRejectionStore.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return executionclient.ExecutionRejectionReply{}, prob
	}

	var detail *executionclient.RejectionDetail
	if intent != nil {
		detail = extractRejectionDetail(intent)
	}

	return executionclient.ExecutionRejectionReply{
		ExecutionIntent: intent,
		Detail:          detail,
	}, nil
}

func (a *QueryResponderActor) handleExecutionControlGet(ctx context.Context, _ executionclient.ExecutionControlQuery) (executionclient.ExecutionControlReply, *problem.Problem) {
	gate, prob := a.executionControlStore.Get(ctx)
	if prob != nil {
		return executionclient.ExecutionControlReply{}, prob
	}
	return executionclient.ExecutionControlReply{Gate: gate}, nil
}

func (a *QueryResponderActor) handleExecutionControlSet(ctx context.Context, cmd executionclient.SetExecutionControlCommand) (executionclient.ExecutionControlReply, *problem.Problem) {
	gate := execution.ControlGate{
		Status:    execution.GateStatus(cmd.Status),
		Reason:    cmd.Reason,
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: cmd.UpdatedBy,
	}

	if prob := a.executionControlStore.Put(ctx, gate); prob != nil {
		return executionclient.ExecutionControlReply{}, prob
	}

	a.logger.Info("execution control gate updated",
		"status", cmd.Status,
		"reason", cmd.Reason,
		"updated_by", cmd.UpdatedBy,
	)

	return executionclient.ExecutionControlReply{Gate: gate}, nil
}

func (a *QueryResponderActor) handleActivationSurfaceGet(ctx context.Context, _ executionclient.ActivationSurfaceQuery) (executionclient.ActivationSurfaceReply, *problem.Problem) {
	gate, prob := a.executionControlStore.Get(ctx)
	if prob != nil {
		return executionclient.ActivationSurfaceReply{}, prob
	}

	dims, prob := a.executionControlStore.GetDimensions(ctx)
	if prob != nil {
		return executionclient.ActivationSurfaceReply{}, prob
	}

	// If dimensions were published by the execute binary, compose full surface.
	// Otherwise, return gate-only surface with unknown adapter/credentials.
	adapter := execution.AdapterState("unknown")
	creds := execution.CredentialState("unknown")
	if dims != nil {
		adapter = dims.Adapter
		creds = dims.Credentials
	}

	// H-4: source time via clock.Clock per ADR-0019 INV-D1; fall back
	// to clock.SystemClock{} when the config did not inject one.
	clk := a.cfg.Clock
	if clk == nil {
		clk = clock.SystemClock{}
	}
	surface := execution.NewActivationSurface(clk, adapter, gate, creds)
	return executionclient.ActivationSurfaceReply{Surface: surface}, nil
}

func (a *QueryResponderActor) handleExecutionStatusLatest(ctx context.Context, query executionclient.ExecutionStatusQuery) (executionclient.ExecutionStatusReply, *problem.Problem) {
	intent, prob := a.executionPaperOrderStore.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return executionclient.ExecutionStatusReply{}, prob
	}

	result, prob := a.executionVenueMarketOrderStore.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return executionclient.ExecutionStatusReply{}, prob
	}

	// S387: Read rejection projection for complete lifecycle visibility.
	var rejection *execution.ExecutionIntent
	var rejectionDetail *executionclient.RejectionDetail
	if a.executionVenueRejectionStore != nil {
		rejection, _ = a.executionVenueRejectionStore.Get(ctx, query.Source, query.Instrument, query.Timeframe)
		// S407: Extract rejection audit detail from embedded metadata.
		if rejection != nil {
			rejectionDetail = extractRejectionDetail(rejection)
		}
		// Best-effort: rejection store unavailability does not fail the query.
	}

	gate, prob := a.executionControlStore.Get(ctx)
	if prob != nil {
		return executionclient.ExecutionStatusReply{}, prob
	}

	return executionclient.ExecutionStatusReply{
		Intent:          intent,
		Result:          result,
		Rejection:       rejection,
		RejectionDetail: rejectionDetail,
		Gate:            gate,
		Propagation:     executionclient.DeriveEffectivePropagation(intent, result, rejection),
	}, nil
}

// S413: handleExecutionLifecycleList enumerates all tracked partition keys across
// the three execution KV buckets and returns a per-key lifecycle summary with
// effective propagation.
// S466: Accepts optional Source/Symbol filters — when set, only matching entries are returned.
func (a *QueryResponderActor) handleExecutionLifecycleList(ctx context.Context, query executionclient.LifecycleListQuery) (executionclient.LifecycleListReply, *problem.Problem) {
	// Collect unique keys from all execution KV buckets.
	keySet := make(map[string]struct{})
	var intentKeys, fillKeys, rejectionKeys []string

	if a.executionPaperOrderStore != nil {
		keys, _ := a.executionPaperOrderStore.Keys(ctx)
		intentKeys = keys
		for _, k := range keys {
			keySet[k] = struct{}{}
		}
	}
	if a.executionVenueMarketOrderStore != nil {
		keys, _ := a.executionVenueMarketOrderStore.Keys(ctx)
		fillKeys = keys
		for _, k := range keys {
			keySet[k] = struct{}{}
		}
	}
	if a.executionVenueRejectionStore != nil {
		keys, _ := a.executionVenueRejectionStore.Keys(ctx)
		rejectionKeys = keys
		for _, k := range keys {
			keySet[k] = struct{}{}
		}
	}

	// Build lookup sets for O(1) membership checks.
	intentSet := toSet(intentKeys)
	fillSet := toSet(fillKeys)
	rejectionSet := toSet(rejectionKeys)

	entries := make([]executionclient.LifecycleEntry, 0, len(keySet))
	for key := range keySet {
		source, symbol, timeframe := parsePartitionKey(key)

		// S466: Apply optional source/symbol filters.
		if query.Source != "" && source != query.Source {
			continue
		}
		if !query.Instrument.IsZero() && symbol != query.Instrument.SubjectToken() {
			continue
		}

		entry := executionclient.LifecycleEntry{
			Key:       key,
			Source:    source,
			Symbol:    symbol,
			Timeframe: timeframe,
		}

		// Read intent (paper_order).
		var intent, result, rejection *execution.ExecutionIntent
		if _, ok := intentSet[key]; ok {
			intent, _ = a.executionPaperOrderStore.GetByKey(ctx, key)
		}
		if intent != nil {
			entry.IntentStatus = string(intent.Status)
			ts := intent.Timestamp
			entry.IntentTimestamp = &ts
		}

		// Read fill (venue_market_order).
		if _, ok := fillSet[key]; ok {
			result, _ = a.executionVenueMarketOrderStore.GetByKey(ctx, key)
		}
		if result != nil {
			entry.FillStatus = string(result.Status)
			ts := result.Timestamp
			entry.FillTimestamp = &ts
		}

		// Read rejection (venue_rejection).
		if _, ok := rejectionSet[key]; ok {
			rejection, _ = a.executionVenueRejectionStore.GetByKey(ctx, key)
		}
		if rejection != nil {
			entry.RejectionStatus = string(rejection.Status)
			ts := rejection.Timestamp
			entry.RejectionTimestamp = &ts
		}

		entry.Propagation = executionclient.DeriveEffectivePropagation(intent, result, rejection)
		entries = append(entries, entry)
	}

	return executionclient.LifecycleListReply{
		Entries: entries,
		Total:   len(entries),
	}, nil
}

// parsePartitionKey splits a "{source}.{token}.{timeframe}" key back
// into components. The token is the canonical SubjectToken since
// H-6.e.2 (underscores, never dots), so the 3-way dot split is safe
// for both the new shape and inert pre-cutover orphan keys.
func parsePartitionKey(key string) (source, symbol string, timeframe int) {
	parts := strings.SplitN(key, ".", 3)
	if len(parts) < 3 {
		return key, "", 0
	}
	tf := 0
	_, _ = fmt.Sscanf(parts[2], "%d", &tf)
	return parts[0], parts[1], tf
}

func toSet(keys []string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return m
}

// S460: handleSessionGet retrieves a session record by ID.
func (a *QueryResponderActor) handleSessionGet(ctx context.Context, query executionclient.SessionGetQuery) (executionclient.SessionGetReply, *problem.Problem) {
	session, prob := a.sessionStore.Get(ctx, query.SessionID)
	if prob != nil {
		return executionclient.SessionGetReply{}, prob
	}
	return executionclient.SessionGetReply{Session: session}, nil
}

// S460: handleSessionList returns all session records.
func (a *QueryResponderActor) handleSessionList(ctx context.Context, _ executionclient.SessionListQuery) (executionclient.SessionListReply, *problem.Problem) {
	sessions, prob := a.sessionStore.List(ctx)
	if prob != nil {
		return executionclient.SessionListReply{}, prob
	}
	if sessions == nil {
		sessions = []execution.Session{}
	}
	return executionclient.SessionListReply{
		Sessions: sessions,
		Total:    len(sessions),
	}, nil
}

// extractRejectionDetail reconstructs RejectionDetail from metadata embedded by
// the RejectionProjectionActor (S407). Returns nil if no rejection metadata is present.
func extractRejectionDetail(intent *execution.ExecutionIntent) *executionclient.RejectionDetail {
	if intent == nil || intent.Metadata == nil {
		return nil
	}

	code := intent.Metadata["rejection_code"]
	reason := intent.Metadata["rejection_reason"]
	if code == "" && reason == "" {
		return nil
	}

	detail := &executionclient.RejectionDetail{
		RejectionCode:   code,
		RejectionReason: reason,
	}

	// Reconstruct venue details from prefixed metadata keys.
	for k, v := range intent.Metadata {
		if strings.HasPrefix(k, "venue_detail.") {
			if detail.VenueDetails == nil {
				detail.VenueDetails = make(map[string]any)
			}
			detail.VenueDetails[strings.TrimPrefix(k, "venue_detail.")] = v
		}
	}

	return detail
}
