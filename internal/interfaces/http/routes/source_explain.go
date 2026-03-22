package routes

import (
	"net/http"

	"internal/interfaces/http/handlers"
	"internal/shared/webserver"
)

// SourceExplain registers HTTP routes for source-driven path explainability.
// S361: GET /execution/source-explain returns the composite explanation surface.
func SourceExplain(deps SourceExplainFamilyDeps) []webserver.Route {
	handler := handlers.NewSourceExplainWebHandler(deps.GetSourceExplanation)

	var routes []webserver.Route

	if deps.GetSourceExplanation != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/execution/source-explain",
			Handler: handler.GetExplanation,
		})
	}

	return routes
}
