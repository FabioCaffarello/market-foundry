package routes

import (
	"context"
	"log/slog"
	"net/http"

	"internal/application/triageclient"
	"internal/interfaces/http/handlers"
	"internal/shared/problem"
	"internal/shared/webserver"
)

// TriageFamilyDeps groups batch review and triage use cases.
// S487: Severity-ranked triage surfaces for sessions, decisions, and round-trips.
type TriageFamilyDeps struct {
	GetSessionTriage   handlersGetSessionTriageUseCase
	GetDecisionTriage  handlersGetDecisionTriageUseCase
	GetRoundTripTriage handlersGetRoundTripTriageUseCase
	GetTriageOverview  handlersGetTriageOverviewUseCase
}

// HasAny reports whether at least one triage use case is available.
func (t TriageFamilyDeps) HasAny() bool {
	return t.GetSessionTriage != nil || t.GetDecisionTriage != nil || t.GetRoundTripTriage != nil || t.GetTriageOverview != nil
}

type handlersGetSessionTriageUseCase interface {
	Execute(context.Context, triageclient.SessionTriageQuery) (triageclient.SessionTriageReply, *problem.Problem)
}

type handlersGetDecisionTriageUseCase interface {
	Execute(context.Context, triageclient.DecisionTriageQuery) (triageclient.DecisionTriageReply, *problem.Problem)
}

type handlersGetRoundTripTriageUseCase interface {
	Execute(context.Context, triageclient.RoundTripTriageQuery) (triageclient.RoundTripTriageReply, *problem.Problem)
}

type handlersGetTriageOverviewUseCase interface {
	Execute(context.Context, triageclient.TriageOverviewQuery) (triageclient.TriageOverviewReply, *problem.Problem)
}

// Triage registers HTTP routes for batch review and operational triage endpoints.
// S487: These routes are registered when any triage use case is available.
func Triage(deps TriageFamilyDeps, logger *slog.Logger) []webserver.Route {
	handler := handlers.NewTriageWebHandler(handlers.TriageHandlerDeps{
		GetSessionTriage:   deps.GetSessionTriage,
		GetDecisionTriage:  deps.GetDecisionTriage,
		GetRoundTripTriage: deps.GetRoundTripTriage,
		GetTriageOverview:  deps.GetTriageOverview,
		Logger:             logger,
	})

	var routes []webserver.Route

	if deps.GetSessionTriage != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/triage/sessions",
			Handler: handler.GetSessionTriage,
		})
	}

	if deps.GetDecisionTriage != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/triage/decisions",
			Handler: handler.GetDecisionTriage,
		})
	}

	if deps.GetRoundTripTriage != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/triage/roundtrips",
			Handler: handler.GetRoundTripTriage,
		})
	}

	if deps.GetTriageOverview != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/analytical/triage/overview",
			Handler: handler.GetTriageOverview,
		})
	}

	return routes
}
