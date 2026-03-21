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
	GetCompositeChain       handlersGetCompositeChainUseCase
	GetPipelineFunnel       handlersGetPipelineFunnelUseCase
	GetDispositionBreakdown handlersGetDispositionBreakdownUseCase
}

// HasAny reports whether at least one analytical use case is available.
func (a AnalyticalFamilyDeps) HasAny() bool {
	return a.GetCandleHistory != nil || a.GetSignalHistory != nil || a.GetDecisionHistory != nil || a.GetStrategyHistory != nil || a.GetRiskHistory != nil || a.GetExecutionHistory != nil || a.GetCompositeChain != nil || a.GetPipelineFunnel != nil || a.GetDispositionBreakdown != nil
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

type handlersGetCompositeChainUseCase interface {
	Execute(context.Context, analyticalclient.CompositeChainQuery) (analyticalclient.CompositeChainReply, *problem.Problem)
}

type handlersGetPipelineFunnelUseCase interface {
	Execute(context.Context, analyticalclient.PipelineFunnelQuery) (analyticalclient.PipelineFunnelReply, *problem.Problem)
}

type handlersGetDispositionBreakdownUseCase interface {
	Execute(context.Context, analyticalclient.DispositionBreakdownQuery) (analyticalclient.DispositionBreakdownReply, *problem.Problem)
}

// Analytical registers HTTP routes for analytical (ClickHouse-backed) query endpoints.
// These routes are only registered when ClickHouse is configured and available.
func Analytical(deps AnalyticalFamilyDeps, logger *slog.Logger) []webserver.Route {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{
		GetCandleHistory:   deps.GetCandleHistory,
		GetSignalHistory:   deps.GetSignalHistory,
		GetDecisionHistory: deps.GetDecisionHistory,
		GetStrategyHistory: deps.GetStrategyHistory,
		GetRiskHistory:      deps.GetRiskHistory,
		GetExecutionHistory: deps.GetExecutionHistory,
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

	// Composite chain and aggregation endpoints (S297/S298).
	if deps.GetCompositeChain != nil || deps.GetPipelineFunnel != nil || deps.GetDispositionBreakdown != nil {
		compositeHandler := handlers.NewCompositeWebHandler(handlers.CompositeHandlerDeps{
			GetCompositeChain:      deps.GetCompositeChain,
			GetPipelineFunnel:      deps.GetPipelineFunnel,
			GetDispositionBreakdown: deps.GetDispositionBreakdown,
			Logger:                 logger,
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
	}

	return routes
}
