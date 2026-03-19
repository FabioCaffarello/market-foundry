package routes

import (
	"net/http"

	"internal/interfaces/http/handlers"
	"internal/shared/webserver"
)

// Execution registers HTTP routes for execution query and control endpoints.
func Execution(deps ExecutionFamilyDeps) []webserver.Route {
	handler := handlers.NewExecutionWebHandler(deps.GetLatestExecution, deps.GetExecutionStatus)
	controlHandler := handlers.NewExecutionControlWebHandler(deps.GetExecutionControl, deps.SetExecutionControl)

	var routes []webserver.Route

	if deps.GetLatestExecution != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/execution/:type/latest",
			Handler: handler.GetLatestExecution,
		})
	}

	if deps.GetExecutionStatus != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/execution/status/latest",
			Handler: handler.GetExecutionStatus,
		})
	}

	if deps.GetExecutionControl != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/execution/control",
			Handler: controlHandler.GetControl,
		})
	}

	if deps.SetExecutionControl != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodPut,
			Path:    "/execution/control",
			Handler: controlHandler.SetControl,
		})
	}

	return routes
}
