package routes

import (
	"context"
	"log/slog"
	"net/http"

	"internal/application/analyticalclient"
	"internal/interfaces/http/handlers"
	"internal/shared/problem"
	"internal/shared/webserver"
)

// AnalyticalFamilyDeps groups analytical query use cases backed by ClickHouse.
// These are additive — they never overlap with or modify existing operational routes.
type AnalyticalFamilyDeps struct {
	GetCandleHistory        handlersGetAnalyticalCandleHistoryUseCase
	GetSignalHistory        handlersGetAnalyticalSignalHistoryUseCase
	GetDecisionHistory      handlersGetAnalyticalDecisionHistoryUseCase
	GetStrategyHistory      handlersGetAnalyticalStrategyHistoryUseCase
	GetRiskHistory          handlersGetAnalyticalRiskHistoryUseCase
	GetExecutionHistory     handlersGetAnalyticalExecutionHistoryUseCase
	GetLifecycleHistory     handlersGetAnalyticalLifecycleHistoryUseCase
	GetExecutionList        handlersGetAnalyticalExecutionListUseCase
	GetExecutionSummary     handlersGetAnalyticalExecutionSummaryUseCase
	GetSessionExplain       handlersGetAnalyticalSessionExplainUseCase
	GetCompositeChain       handlersGetCompositeChainUseCase
	GetPipelineFunnel       handlersGetPipelineFunnelUseCase
	GetDispositionBreakdown handlersGetDispositionBreakdownUseCase
	GetDecisionReview       handlersGetDecisionReviewUseCase
	GetEffectiveness        handlersGetEffectivenessUseCase
	GetEffectivenessSummary handlersGetEffectivenessSummaryUseCase
	GetPairing              handlersGetPairingUseCase
	GetRoundTripReview      handlersGetRoundTripReviewUseCase
	GetCrossSessionPairing  handlersGetCrossSessionPairingUseCase
	GetContinuityReview     handlersGetContinuityReviewUseCase
}

// HasAny reports whether at least one analytical use case is available.
func (a AnalyticalFamilyDeps) HasAny() bool {
	return a.GetCandleHistory != nil || a.GetSignalHistory != nil || a.GetDecisionHistory != nil || a.GetStrategyHistory != nil || a.GetRiskHistory != nil || a.GetExecutionHistory != nil || a.GetLifecycleHistory != nil || a.GetExecutionList != nil || a.GetExecutionSummary != nil || a.GetSessionExplain != nil || a.GetCompositeChain != nil || a.GetPipelineFunnel != nil || a.GetDispositionBreakdown != nil || a.GetDecisionReview != nil || a.GetEffectiveness != nil || a.GetEffectivenessSummary != nil || a.GetPairing != nil || a.GetRoundTripReview != nil || a.GetCrossSessionPairing != nil || a.GetContinuityReview != nil
}

type handlersGetAnalyticalCandleHistoryUseCase interface {
	Execute(context.Context, analyticalclient.CandleHistoryQuery) (analyticalclient.CandleHistoryReply, *problem.Problem)
}

type handlersGetAnalyticalSignalHistoryUseCase interface {
	Execute(context.Context, analyticalclient.SignalHistoryQuery) (analyticalclient.SignalHistoryReply, *problem.Problem)
}

type handlersGetAnalyticalDecisionHistoryUseCase interface {
	Execute(context.Context, analyticalclient.DecisionHistoryQuery) (analyticalclient.DecisionHistoryReply, *problem.Problem)
}

type handlersGetAnalyticalStrategyHistoryUseCase interface {
	Execute(context.Context, analyticalclient.StrategyHistoryQuery) (analyticalclient.StrategyHistoryReply, *problem.Problem)
}

type handlersGetAnalyticalRiskHistoryUseCase interface {
	Execute(context.Context, analyticalclient.RiskHistoryQuery) (analyticalclient.RiskHistoryReply, *problem.Problem)
}

type handlersGetAnalyticalExecutionHistoryUseCase interface {
	Execute(context.Context, analyticalclient.ExecutionHistoryQuery) (analyticalclient.ExecutionHistoryReply, *problem.Problem)
}

type handlersGetAnalyticalLifecycleHistoryUseCase interface {
	Execute(context.Context, analyticalclient.LifecycleHistoryQuery) (analyticalclient.LifecycleHistoryReply, *problem.Problem)
}

type handlersGetAnalyticalExecutionListUseCase interface {
	Execute(context.Context, analyticalclient.ExecutionListQuery) (analyticalclient.ExecutionListReply, *problem.Problem)
}

type handlersGetAnalyticalExecutionSummaryUseCase interface {
	Execute(context.Context, analyticalclient.ExecutionSummaryQuery) (analyticalclient.ExecutionSummaryReply, *problem.Problem)
}

type handlersGetAnalyticalSessionExplainUseCase interface {
	Execute(context.Context, analyticalclient.SessionExplainQuery) (analyticalclient.SessionExplainReply, *problem.Problem)
}

type handlersGetCompositeChainUseCase interface {
	Execute(context.Context, analyticalclient.CompositeChainQuery) (analyticalclient.CompositeChainReply, *problem.Problem)
}

type handlersGetPipelineFunnelUseCase interface {
	Execute(context.Context, analyticalclient.PipelineFunnelQuery) (analyticalclient.PipelineFunnelReply, *problem.Problem)
}

type handlersGetDispositionBreakdownUseCase interface {
	Execute(context.Context, analyticalclient.DispositionBreakdownQuery) (analyticalclient.DispositionBreakdownReply, *problem.Problem)
}

type handlersGetDecisionReviewUseCase interface {
	Execute(context.Context, analyticalclient.DecisionReviewQuery) (analyticalclient.DecisionReviewReply, *problem.Problem)
}

type handlersGetEffectivenessUseCase interface {
	Execute(context.Context, analyticalclient.EffectivenessQuery) (analyticalclient.EffectivenessReply, *problem.Problem)
}

type handlersGetEffectivenessSummaryUseCase interface {
	Execute(context.Context, analyticalclient.EffectivenessSummaryQuery) (analyticalclient.EffectivenessSummaryReply, *problem.Problem)
}

type handlersGetPairingUseCase interface {
	Execute(context.Context, analyticalclient.PairingQuery) (analyticalclient.PairingReply, *problem.Problem)
}

type handlersGetRoundTripReviewUseCase interface {
	Execute(context.Context, analyticalclient.RoundTripReviewQuery) (analyticalclient.RoundTripReviewReply, *problem.Problem)
}

type handlersGetCrossSessionPairingUseCase interface {
	Execute(context.Context, analyticalclient.CrossSessionPairingQuery) (analyticalclient.CrossSessionPairingReply, *problem.Problem)
}

type handlersGetContinuityReviewUseCase interface {
	Execute(context.Context, analyticalclient.ContinuityReviewQuery) (analyticalclient.ContinuityReviewReply, *problem.Problem)
}

// Analytical registers HTTP routes for analytical (ClickHouse-backed) query endpoints.
// These routes are only registered when ClickHouse is configured and available.
func Analytical(deps AnalyticalFamilyDeps, logger *slog.Logger) []webserver.Route {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{
		GetCandleHistory:    deps.GetCandleHistory,
		GetSignalHistory:    deps.GetSignalHistory,
		GetDecisionHistory:  deps.GetDecisionHistory,
		GetStrategyHistory:  deps.GetStrategyHistory,
		GetRiskHistory:      deps.GetRiskHistory,
		GetExecutionHistory: deps.GetExecutionHistory,
		GetLifecycleHistory: deps.GetLifecycleHistory,
		GetExecutionList:    deps.GetExecutionList,
		GetExecutionSummary: deps.GetExecutionSummary,
		GetSessionExplain:   deps.GetSessionExplain,
		Logger:              logger,
	})

	var routes []webserver.Route

	if deps.GetCandleHistory != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/evidence/candles",
			Handler: handler.GetCandleHistory,
		})
	}

	if deps.GetSignalHistory != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/signal/history",
			Handler: handler.GetSignalHistory,
		})
	}

	if deps.GetDecisionHistory != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/decision/history",
			Handler: handler.GetDecisionHistory,
		})
	}

	if deps.GetStrategyHistory != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/strategy/history",
			Handler: handler.GetStrategyHistory,
		})
	}

	if deps.GetRiskHistory != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/risk/history",
			Handler: handler.GetRiskHistory,
		})
	}

	if deps.GetExecutionHistory != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/execution/history",
			Handler: handler.GetExecutionHistory,
		})
	}

	// S453A: Lifecycle history endpoint — unified cross-type timeline.
	if deps.GetLifecycleHistory != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/execution/lifecycle",
			Handler: handler.GetLifecycleHistory,
		})
	}

	// S454A: Execution list — relaxed-filter operational list query.
	if deps.GetExecutionList != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/execution/list",
			Handler: handler.GetExecutionList,
		})
	}

	// S454A: Execution summary — counts grouped by (type, status).
	if deps.GetExecutionSummary != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/execution/summary",
			Handler: handler.GetExecutionSummary,
		})
	}

	// S455A: Session explain — unified explainability with cross-surface consistency.
	if deps.GetSessionExplain != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/execution/explain",
			Handler: handler.GetSessionExplain,
		})
	}

	// Composite chain, aggregation, decision review, effectiveness, and pairing endpoints (S297/S298/S471/S476/S481).
	if deps.GetCompositeChain != nil || deps.GetPipelineFunnel != nil || deps.GetDispositionBreakdown != nil || deps.GetDecisionReview != nil || deps.GetEffectiveness != nil || deps.GetPairing != nil || deps.GetRoundTripReview != nil || deps.GetCrossSessionPairing != nil || deps.GetContinuityReview != nil {
		compositeHandler := handlers.NewCompositeWebHandler(handlers.CompositeHandlerDeps{
			GetCompositeChain:       deps.GetCompositeChain,
			GetPipelineFunnel:       deps.GetPipelineFunnel,
			GetDispositionBreakdown: deps.GetDispositionBreakdown,
			GetDecisionReview:       deps.GetDecisionReview,
			GetEffectiveness:        deps.GetEffectiveness,
			GetEffectivenessSummary: deps.GetEffectivenessSummary,
			GetPairing:              deps.GetPairing,
			GetRoundTripReview:      deps.GetRoundTripReview,
			GetCrossSessionPairing:  deps.GetCrossSessionPairing,
			GetContinuityReview:     deps.GetContinuityReview,
			Logger:                  logger,
		})

		if deps.GetCompositeChain != nil {
			routes = append(routes,
				webserver.Route{
					Method:  http.MethodGet,
					Path:    "/analytical/composite/chain",
					Handler: compositeHandler.GetChain,
				},
				webserver.Route{
					Method:  http.MethodGet,
					Path:    "/analytical/composite/chains",
					Handler: compositeHandler.GetChains,
				},
			)
		}

		if deps.GetPipelineFunnel != nil {
			routes = append(routes, webserver.Route{
				Method:  http.MethodGet,
				Path:    "/analytical/composite/funnel",
				Handler: compositeHandler.GetFunnel,
			})
		}

		if deps.GetDispositionBreakdown != nil {
			routes = append(routes, webserver.Route{
				Method:  http.MethodGet,
				Path:    "/analytical/composite/dispositions",
				Handler: compositeHandler.GetDispositions,
			})
		}

		// S471: Decision review — decision-centric evidence bundle surface.
		if deps.GetDecisionReview != nil {
			routes = append(routes,
				webserver.Route{
					Method:  http.MethodGet,
					Path:    "/analytical/composite/decision/review",
					Handler: compositeHandler.GetDecisionReview,
				},
				webserver.Route{
					Method:  http.MethodGet,
					Path:    "/analytical/composite/decision/reviews",
					Handler: compositeHandler.GetDecisionReviews,
				},
			)
		}

		// S476: Effectiveness evaluation — measurement read surfaces and batch evaluation.
		if deps.GetEffectiveness != nil {
			routes = append(routes,
				webserver.Route{
					Method:  http.MethodGet,
					Path:    "/analytical/composite/decision/effectiveness",
					Handler: compositeHandler.GetEffectiveness,
				},
				webserver.Route{
					Method:  http.MethodGet,
					Path:    "/analytical/composite/decision/effectiveness/batch",
					Handler: compositeHandler.GetEffectivenessBatch,
				},
			)
		}

		// S477: Effectiveness summary and comparative analysis.
		if deps.GetEffectivenessSummary != nil {
			routes = append(routes, webserver.Route{
				Method:  http.MethodGet,
				Path:    "/analytical/composite/decision/effectiveness/summary",
				Handler: compositeHandler.GetEffectivenessSummary,
			})
		}

		// S481: Round-trip pairing read model.
		if deps.GetPairing != nil {
			routes = append(routes,
				webserver.Route{
					Method:  http.MethodGet,
					Path:    "/analytical/composite/pairing",
					Handler: compositeHandler.GetPairing,
				},
				webserver.Route{
					Method:  http.MethodGet,
					Path:    "/analytical/composite/pairing/chain",
					Handler: compositeHandler.GetPairingSingle,
				},
			)
		}

		// S482: Round-trip review and outcome reconciliation.
		if deps.GetRoundTripReview != nil {
			routes = append(routes,
				webserver.Route{
					Method:  http.MethodGet,
					Path:    "/analytical/composite/pairing/review",
					Handler: compositeHandler.GetRoundTripReview,
				},
				webserver.Route{
					Method:  http.MethodGet,
					Path:    "/analytical/composite/pairing/review/chain",
					Handler: compositeHandler.GetRoundTripReviewSingle,
				},
			)
		}

		// S495: Cross-session pairing read model.
		if deps.GetCrossSessionPairing != nil {
			routes = append(routes, webserver.Route{
				Method:  http.MethodGet,
				Path:    "/analytical/composite/pairing/cross-session",
				Handler: compositeHandler.GetCrossSessionPairing,
			})
		}

		// S496: Continuity review — cross-session reconciliation and effectiveness surface.
		if deps.GetContinuityReview != nil {
			routes = append(routes, webserver.Route{
				Method:  http.MethodGet,
				Path:    "/analytical/composite/pairing/continuity-review",
				Handler: compositeHandler.GetContinuityReview,
			})
		}
	}

	return routes
}
