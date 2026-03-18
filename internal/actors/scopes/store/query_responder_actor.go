package store

import (
	"context"
	"fmt"
	"log/slog"

	actorcommon "internal/actors/common"
	adapternats "internal/adapters/nats"
	"internal/application/decisionclient"
	"internal/application/evidenceclient"
	"internal/application/signalclient"
	"internal/application/strategyclient"
	"internal/domain/evidence"
	"internal/shared/problem"

	"github.com/anthdm/hollywood/actor"
)

// QueryResponderConfig holds the configuration for the query responder actor.
type QueryResponderConfig struct {
	NATSURL          string
	Source           string
	Registry         adapternats.EvidenceRegistry
	SignalRegistry   *adapternats.SignalRegistry   // nil when no signal families are enabled
	DecisionRegistry *adapternats.DecisionRegistry // nil when no decision families are enabled
	StrategyRegistry *adapternats.StrategyRegistry // nil when no strategy families are enabled
}

// QueryResponderActor serves evidence and signal queries from the NATS KV stores.
// It reads the materialized projections — no dependency on derive actors.
type QueryResponderActor struct {
	cfg                      QueryResponderConfig
	logger                   *slog.Logger
	store                    *adapternats.CandleKVStore
	burstStore               *adapternats.TradeBurstKVStore
	volumeStore              *adapternats.VolumeKVStore
	signalRSIStore           *adapternats.SignalKVStore
	decisionRSIOversoldStore             *adapternats.DecisionKVStore
	strategyMeanReversionEntryStore      *adapternats.StrategyKVStore
	responder                            *adapternats.RequestReplyResponder
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

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *QueryResponderActor) start(c *actor.Context) {
	// Open a read-only KV store connection for candle queries.
	store := adapternats.NewCandleKVStore(a.cfg.NATSURL)
	if err := store.Start(); err != nil {
		a.logger.Error("start query KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}
	a.store = store

	// Open a read-only KV store connection for trade burst queries.
	burstStore := adapternats.NewTradeBurstKVStore(a.cfg.NATSURL)
	if err := burstStore.Start(); err != nil {
		a.logger.Error("start trade burst query KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}
	a.burstStore = burstStore

	// Open a read-only KV store connection for volume queries.
	volumeStore := adapternats.NewVolumeKVStore(a.cfg.NATSURL)
	if err := volumeStore.Start(); err != nil {
		a.logger.Error("start volume query KV store", "error", err)
		c.Engine().Poison(c.PID())
		return
	}
	a.volumeStore = volumeStore

	routes := []adapternats.ControlRoute{
		adapternats.NewTypedControlRoute(
			a.cfg.Registry.CandleLatest,
			a.cfg.Source,
			a.handleCandleLatest,
		),
		adapternats.NewTypedControlRoute(
			a.cfg.Registry.CandleHistory,
			a.cfg.Source,
			a.handleCandleHistory,
		),
		adapternats.NewTypedControlRoute(
			a.cfg.Registry.TradeBurstLatest,
			a.cfg.Source,
			a.handleTradeBurstLatest,
		),
		adapternats.NewTypedControlRoute(
			a.cfg.Registry.VolumeLatest,
			a.cfg.Source,
			a.handleVolumeLatest,
		),
	}

	// Wire signal query routes if signal families are enabled.
	if a.cfg.SignalRegistry != nil {
		sigStore := adapternats.NewSignalKVStore(a.cfg.NATSURL, adapternats.SignalRSILatestBucket)
		if err := sigStore.Start(); err != nil {
			a.logger.Error("start signal RSI query KV store", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.signalRSIStore = sigStore

		routes = append(routes, adapternats.NewTypedControlRoute(
			a.cfg.SignalRegistry.RSILatest,
			a.cfg.Source,
			a.handleSignalRSILatest,
		))
	}

	// Wire decision query routes if decision families are enabled.
	if a.cfg.DecisionRegistry != nil {
		decStore := adapternats.NewDecisionKVStore(a.cfg.NATSURL, adapternats.DecisionRSIOversoldLatestBucket)
		if err := decStore.Start(); err != nil {
			a.logger.Error("start decision RSI oversold query KV store", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.decisionRSIOversoldStore = decStore

		routes = append(routes, adapternats.NewTypedControlRoute(
			a.cfg.DecisionRegistry.RSIOversoldLatest,
			a.cfg.Source,
			a.handleDecisionRSIOversoldLatest,
		))
	}

	// Wire strategy query routes if strategy families are enabled.
	if a.cfg.StrategyRegistry != nil {
		stratStore := adapternats.NewStrategyKVStore(a.cfg.NATSURL, adapternats.StrategyMeanReversionEntryLatestBucket)
		if err := stratStore.Start(); err != nil {
			a.logger.Error("start strategy mean reversion entry query KV store", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.strategyMeanReversionEntryStore = stratStore

		routes = append(routes, adapternats.NewTypedControlRoute(
			a.cfg.StrategyRegistry.MeanReversionEntryLatest,
			a.cfg.Source,
			a.handleStrategyMeanReversionEntryLatest,
		))
	}

	responder := adapternats.NewRequestReplyResponder(a.cfg.NATSURL, routes)
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
		"bucket_latest", adapternats.CandleLatestBucket,
		"bucket_history", adapternats.CandleHistoryBucket,
		"bucket_trade_burst_latest", adapternats.TradeBurstLatestBucket,
		"bucket_volume_latest", adapternats.VolumeLatestBucket,
	}
	if a.cfg.SignalRegistry != nil {
		logFields = append(logFields,
			"subject_signal_rsi_latest", a.cfg.SignalRegistry.RSILatest.Subject,
			"bucket_signal_rsi_latest", adapternats.SignalRSILatestBucket,
		)
	}
	if a.cfg.DecisionRegistry != nil {
		logFields = append(logFields,
			"subject_decision_rsi_oversold_latest", a.cfg.DecisionRegistry.RSIOversoldLatest.Subject,
			"bucket_decision_rsi_oversold_latest", adapternats.DecisionRSIOversoldLatestBucket,
		)
	}
	if a.cfg.StrategyRegistry != nil {
		logFields = append(logFields,
			"subject_strategy_mean_reversion_entry_latest", a.cfg.StrategyRegistry.MeanReversionEntryLatest.Subject,
			"bucket_strategy_mean_reversion_entry_latest", adapternats.StrategyMeanReversionEntryLatestBucket,
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
