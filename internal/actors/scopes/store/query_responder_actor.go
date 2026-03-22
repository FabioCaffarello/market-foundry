package store

import (
	"context"
	"fmt"
	"log/slog"

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
	"internal/shared/problem"

	"github.com/anthdm/hollywood/actor"
)

// QueryResponderConfig holds the configuration for the query responder actor.
type QueryResponderConfig struct {
	NATSURL          string
	Source           string
	Registry         natsevidence.Registry
	SignalRegistry   *natssignal.Registry   // nil when no signal families are enabled
	DecisionRegistry *natsdecision.Registry // nil when no decision families are enabled
	StrategyRegistry *natsstrategy.Registry // nil when no strategy families are enabled
	RiskRegistry      *natsrisk.Registry      // nil when no risk families are enabled
	ExecutionRegistry *natsexecution.Registry // nil when no execution families are enabled
}

// QueryResponderActor serves evidence and signal queries from the NATS KV stores.
// It reads the materialized projections — no dependency on derive actors.
type QueryResponderActor struct {
	cfg                      QueryResponderConfig
	logger                   *slog.Logger
	store                    *natsevidence.CandleKVStore
	burstStore               *natsevidence.TradeBurstKVStore
	volumeStore              *natsevidence.VolumeKVStore
	signalRSIStore           *natssignal.KVStore
	decisionRSIOversoldStore             *natsdecision.KVStore
	strategyMeanReversionEntryStore      *natsstrategy.KVStore
	riskPositionExposureStore             *natsrisk.KVStore
	executionPaperOrderStore             *natsexecution.KVStore
	executionVenueMarketOrderStore       *natsexecution.KVStore
	executionControlStore                *natsexecution.ControlKVStore
	responder                            *natskit.RequestReplyResponder
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
		if a.executionControlStore != nil {
			if err := a.executionControlStore.Close(); err != nil {
				a.logger.Error("close execution control query KV store", "error", err)
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

		// Wire composite status query (reads both KV stores + control).
		routes = append(routes, natskit.NewTypedControlRoute(
			a.cfg.ExecutionRegistry.StatusLatest,
			a.cfg.Source,
			a.handleExecutionStatusLatest,
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
			"subject_execution_status_latest", a.cfg.ExecutionRegistry.StatusLatest.Subject,
			"subject_execution_control_get", a.cfg.ExecutionRegistry.ControlGet.Subject,
			"subject_execution_control_set", a.cfg.ExecutionRegistry.ControlSet.Subject,
			"subject_activation_surface_get", a.cfg.ExecutionRegistry.ActivationSurfaceGet.Subject,
			"bucket_execution_control", natsexecution.ControlBucket,
		)
	}
	a.logger.Info("query responder started", logFields...)
}

func (a *QueryResponderActor) handleCandleLatest(ctx context.Context, query evidenceclient.CandleLatestQuery) (evidenceclient.CandleLatestReply, *problem.Problem) {
	candle, prob := a.store.Get(ctx, query.Source, query.Symbol, query.Timeframe)
	if prob != nil {
		return evidenceclient.CandleLatestReply{}, prob
	}

	return evidenceclient.CandleLatestReply{Candle: candle}, nil
}

func (a *QueryResponderActor) handleTradeBurstLatest(ctx context.Context, query evidenceclient.TradeBurstLatestQuery) (evidenceclient.TradeBurstLatestReply, *problem.Problem) {
	burst, prob := a.burstStore.Get(ctx, query.Source, query.Symbol, query.Timeframe)
	if prob != nil {
		return evidenceclient.TradeBurstLatestReply{}, prob
	}

	return evidenceclient.TradeBurstLatestReply{TradeBurst: burst}, nil
}

func (a *QueryResponderActor) handleCandleHistory(ctx context.Context, query evidenceclient.CandleHistoryQuery) (evidenceclient.CandleHistoryReply, *problem.Problem) {
	candles, prob := a.store.GetHistory(ctx, query.Source, query.Symbol, query.Timeframe, query.Limit, query.Since, query.Until)
	if prob != nil {
		return evidenceclient.CandleHistoryReply{}, prob
	}

	if candles == nil {
		candles = []evidence.EvidenceCandle{}
	}

	return evidenceclient.CandleHistoryReply{Candles: candles}, nil
}

func (a *QueryResponderActor) handleVolumeLatest(ctx context.Context, query evidenceclient.VolumeLatestQuery) (evidenceclient.VolumeLatestReply, *problem.Problem) {
	vol, prob := a.volumeStore.Get(ctx, query.Source, query.Symbol, query.Timeframe)
	if prob != nil {
		return evidenceclient.VolumeLatestReply{}, prob
	}

	return evidenceclient.VolumeLatestReply{Volume: vol}, nil
}

func (a *QueryResponderActor) handleSignalRSILatest(ctx context.Context, query signalclient.SignalLatestQuery) (signalclient.SignalLatestReply, *problem.Problem) {
	sig, prob := a.signalRSIStore.Get(ctx, query.Source, query.Symbol, query.Timeframe)
	if prob != nil {
		return signalclient.SignalLatestReply{}, prob
	}

	return signalclient.SignalLatestReply{Signal: sig}, nil
}

func (a *QueryResponderActor) handleDecisionRSIOversoldLatest(ctx context.Context, query decisionclient.DecisionLatestQuery) (decisionclient.DecisionLatestReply, *problem.Problem) {
	dec, prob := a.decisionRSIOversoldStore.Get(ctx, query.Source, query.Symbol, query.Timeframe)
	if prob != nil {
		return decisionclient.DecisionLatestReply{}, prob
	}

	return decisionclient.DecisionLatestReply{Decision: dec}, nil
}

func (a *QueryResponderActor) handleStrategyMeanReversionEntryLatest(ctx context.Context, query strategyclient.StrategyLatestQuery) (strategyclient.StrategyLatestReply, *problem.Problem) {
	strat, prob := a.strategyMeanReversionEntryStore.Get(ctx, query.Source, query.Symbol, query.Timeframe)
	if prob != nil {
		return strategyclient.StrategyLatestReply{}, prob
	}

	return strategyclient.StrategyLatestReply{Strategy: strat}, nil
}

func (a *QueryResponderActor) handleRiskPositionExposureLatest(ctx context.Context, query riskclient.RiskLatestQuery) (riskclient.RiskLatestReply, *problem.Problem) {
	assessment, prob := a.riskPositionExposureStore.Get(ctx, query.Source, query.Symbol, query.Timeframe)
	if prob != nil {
		return riskclient.RiskLatestReply{}, prob
	}

	return riskclient.RiskLatestReply{RiskAssessment: assessment}, nil
}

func (a *QueryResponderActor) handleExecutionPaperOrderLatest(ctx context.Context, query executionclient.ExecutionLatestQuery) (executionclient.ExecutionLatestReply, *problem.Problem) {
	intent, prob := a.executionPaperOrderStore.Get(ctx, query.Source, query.Symbol, query.Timeframe)
	if prob != nil {
		return executionclient.ExecutionLatestReply{}, prob
	}

	return executionclient.ExecutionLatestReply{ExecutionIntent: intent}, nil
}

func (a *QueryResponderActor) handleExecutionVenueMarketOrderLatest(ctx context.Context, query executionclient.ExecutionLatestQuery) (executionclient.ExecutionLatestReply, *problem.Problem) {
	intent, prob := a.executionVenueMarketOrderStore.Get(ctx, query.Source, query.Symbol, query.Timeframe)
	if prob != nil {
		return executionclient.ExecutionLatestReply{}, prob
	}

	return executionclient.ExecutionLatestReply{ExecutionIntent: intent}, nil
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

	surface := execution.NewActivationSurface(adapter, gate, creds)
	return executionclient.ActivationSurfaceReply{Surface: surface}, nil
}

func (a *QueryResponderActor) handleExecutionStatusLatest(ctx context.Context, query executionclient.ExecutionStatusQuery) (executionclient.ExecutionStatusReply, *problem.Problem) {
	intent, prob := a.executionPaperOrderStore.Get(ctx, query.Source, query.Symbol, query.Timeframe)
	if prob != nil {
		return executionclient.ExecutionStatusReply{}, prob
	}

	result, prob := a.executionVenueMarketOrderStore.Get(ctx, query.Source, query.Symbol, query.Timeframe)
	if prob != nil {
		return executionclient.ExecutionStatusReply{}, prob
	}

	gate, prob := a.executionControlStore.Get(ctx)
	if prob != nil {
		return executionclient.ExecutionStatusReply{}, prob
	}

	return executionclient.ExecutionStatusReply{
		Intent:      intent,
		Result:      result,
		Gate:        gate,
		Propagation: executionclient.DeriveEffectivePropagation(intent, result),
	}, nil
}
