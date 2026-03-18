package routes

import (
	"net/http"

	"internal/interfaces/http/handlers"
	"internal/interfaces/http/webserver"
)

// Strategy registers HTTP routes for strategy query endpoints.
// Phase 1: single route for latest strategy by type.
func Strategy(deps StrategyFamilyDeps) []webserver.Route {
	handler := handlers.NewStrategyWebHandler(deps.GetLatestStrategy)

	var routes []webserver.Route

	if deps.GetLatestStrategy != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/strategy/:type/latest",
			Handler: handler.GetLatestStrategy,
		})
	}

	return routes
}
