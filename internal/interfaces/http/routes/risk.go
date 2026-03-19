package routes

import (
	"net/http"

	"internal/interfaces/http/handlers"
	"internal/shared/webserver"
)

// Risk registers HTTP routes for risk query endpoints.
// Phase 1: single route for latest risk assessment by type.
func Risk(deps RiskFamilyDeps) []webserver.Route {
	handler := handlers.NewRiskWebHandler(deps.GetLatestRisk)

	var routes []webserver.Route

	if deps.GetLatestRisk != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/risk/:type/latest",
			Handler: handler.GetLatestRisk,
		})
	}

	return routes
}
