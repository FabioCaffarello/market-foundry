package routes

import (
	"net/http"

	"internal/interfaces/http/handlers"
	"internal/shared/webserver"
)

// Execution registers HTTP routes for execution query and control endpoints.
func Execution(deps ExecutionFamilyDeps) []webserver.Route {
	handler := handlers.NewExecutionWebHandler(deps.GetLatestExecution, deps.GetExecutionStatus, deps.GetLifecycleList)
	controlHandler := handlers.NewExecutionControlWebHandler(deps.GetExecutionControl, deps.SetExecutionControl)

	var routes []webserver.Route

	if deps.GetLatestExecution != nil || deps.GetExecutionStatus != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/execution/:type/latest",
			Handler: handler.GetLatestExecution,
		})
	}

	// S454A: Lifecycle list — enumerates all tracked execution lifecycle entries.
	if deps.GetLifecycleList != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/execution/lifecycle/list",
			Handler: handler.GetLifecycleList,
		})
	}

	if deps.GetExecutionControl != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/execution/:type",
			Handler: controlHandler.GetControl,
		})
	}

	if deps.SetExecutionControl != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodPut,
			Path:    "/execution/:type",
			Handler: controlHandler.SetControl,
		})
	}

	return routes
}
