package routes

import (
	"net/http"

	"internal/interfaces/http/handlers"
	"internal/shared/webserver"
)

// Signal registers HTTP routes for signal query endpoints.
// Phase 1: single route for latest signal by type. The :type path param
// dispatches to the correct NATS subject via the signal gateway.
func Signal(deps SignalFamilyDeps) []webserver.Route {
	handler := handlers.NewSignalWebHandler(deps.GetLatestSignal)

	var routes []webserver.Route

	if deps.GetLatestSignal != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/signal/:type/latest",
			Handler: handler.GetLatestSignal,
		})
	}

	return routes
}
