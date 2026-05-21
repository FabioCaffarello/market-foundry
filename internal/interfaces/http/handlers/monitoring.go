package handlers

import (
	"context"
	"net/http"

	"internal/application/monitoringclient"
	"internal/shared/problem"
)

type getOperationalStateUseCase interface {
	Execute(context.Context, monitoringclient.OperationalStateQuery) (monitoringclient.OperationalStateReply, *problem.Problem)
}

// MonitoringWebHandler handles HTTP requests for operational monitoring endpoints.
type MonitoringWebHandler struct {
	getOperationalState getOperationalStateUseCase
}

func NewMonitoringWebHandler(getOperationalState getOperationalStateUseCase) *MonitoringWebHandler {
	return &MonitoringWebHandler{getOperationalState: getOperationalState}
}

// GetOperationalState handles GET /monitoring/state
// Returns a consolidated operational state snapshot.
func (h *MonitoringWebHandler) GetOperationalState(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getOperationalState == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "operational state monitoring is unavailable"))
		return
	}

	result, prob := h.getOperationalState.Execute(r.Context(), monitoringclient.OperationalStateQuery{})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, result.State)
}
