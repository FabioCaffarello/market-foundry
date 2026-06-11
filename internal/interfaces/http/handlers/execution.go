package handlers

import (
	"context"
	"net/http"

	"internal/application/executionclient"
	"internal/domain/execution"
	"internal/shared/problem"
)

type getLatestExecutionUseCase interface {
	Execute(context.Context, executionclient.ExecutionLatestQuery) (executionclient.ExecutionLatestReply, *problem.Problem)
}

type getExecutionStatusUseCase interface {
	Execute(context.Context, executionclient.ExecutionStatusQuery) (executionclient.ExecutionStatusReply, *problem.Problem)
}

type getLifecycleListUseCase interface {
	Execute(context.Context, executionclient.LifecycleListQuery) (executionclient.LifecycleListReply, *problem.Problem)
}

// ExecutionWebHandler handles HTTP requests for execution queries.
type ExecutionWebHandler struct {
	getLatestExecution getLatestExecutionUseCase
	getExecutionStatus getExecutionStatusUseCase
	getLifecycleList   getLifecycleListUseCase
}

func NewExecutionWebHandler(getLatestExecution getLatestExecutionUseCase, getExecutionStatus getExecutionStatusUseCase, getLifecycleList getLifecycleListUseCase) *ExecutionWebHandler {
	return &ExecutionWebHandler{getLatestExecution: getLatestExecution, getExecutionStatus: getExecutionStatus, getLifecycleList: getLifecycleList}
}

type latestExecutionResponse struct {
	ExecutionIntent *execution.ExecutionIntent `json:"execution_intent"`
}

// GetLatestExecution handles GET /execution/:type/latest?source=...&symbol=...&timeframe=...
// The composite status path is served through the same route with type=status
// to avoid a wildcard/static conflict in httprouter.
func (h *ExecutionWebHandler) GetLatestExecution(w http.ResponseWriter, r *http.Request) {
	execType := pathParam(r, "type")
	if execType == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "execution type path parameter is required"))
		return
	}

	if execType == "status" {
		h.GetExecutionStatus(w, r)
		return
	}

	if h == nil || h.getLatestExecution == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "execution query is unavailable"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getLatestExecution.Execute(r.Context(), executionclient.ExecutionLatestQuery{
		Type:       execType,
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, latestExecutionResponse{ExecutionIntent: result.ExecutionIntent})
}

// GetExecutionStatus handles GET /execution/status/latest?source=...&symbol=...&timeframe=...
func (h *ExecutionWebHandler) GetExecutionStatus(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getExecutionStatus == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "execution status query is unavailable"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getExecutionStatus.Execute(r.Context(), executionclient.ExecutionStatusQuery{
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}
