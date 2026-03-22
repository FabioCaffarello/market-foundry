package routes

import (
	"net/http"

	"internal/interfaces/http/handlers"
	"internal/shared/webserver"
)

// Activation registers HTTP routes for activation surface query endpoints.
func Activation(deps ActivationFamilyDeps) []webserver.Route {
	handler := handlers.NewActivationWebHandler(deps.GetActivationSurface)

	var routes []webserver.Route

	if deps.GetActivationSurface != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/activation/surface",
			Handler: handler.GetSurface,
		})
	}

	return routes
}
