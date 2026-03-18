package routes

import (
	"net/http"

	"internal/interfaces/http/handlers"
	"internal/interfaces/http/webserver"
)

// Decision registers HTTP routes for decision query endpoints.
// Phase 1: single route for latest decision by type.
func Decision(deps DecisionFamilyDeps) []webserver.Route {
	handler := handlers.NewDecisionWebHandler(deps.GetLatestDecision)

	var routes []webserver.Route

	if deps.GetLatestDecision != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/decision/:type/latest",
			Handler: handler.GetLatestDecision,
		})
	}

	return routes
}
