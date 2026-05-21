package routes

import (
	"context"
	"net/http"

	"internal/application/monitoringclient"
	"internal/interfaces/http/handlers"
	"internal/shared/problem"
	"internal/shared/webserver"
)

// MonitoringFamilyDeps groups operational monitoring use cases.
// S486: Exposes consolidated monitoring surfaces.
type MonitoringFamilyDeps struct {
	GetOperationalState handlersGetOperationalStateUseCase
}

// HasAny reports whether at least one monitoring use case is available.
func (m MonitoringFamilyDeps) HasAny() bool {
	return m.GetOperationalState != nil
}

type handlersGetOperationalStateUseCase interface {
	Execute(context.Context, monitoringclient.OperationalStateQuery) (monitoringclient.OperationalStateReply, *problem.Problem)
}

// Monitoring registers HTTP routes for operational monitoring endpoints.
// S486: These routes are always registered when the monitoring use case is wired.
func Monitoring(deps MonitoringFamilyDeps) []webserver.Route {
	handler := handlers.NewMonitoringWebHandler(deps.GetOperationalState)

	var routes []webserver.Route

	if deps.GetOperationalState != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/monitoring/state",
			Handler: handler.GetOperationalState,
		})
	}

	return routes
}
