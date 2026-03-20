package handlers

import (
	"context"
	"net/http"

	"internal/application/executionclient"
	"internal/shared/problem"
)

type getExecutionControlUseCase interface {
	Execute(context.Context, executionclient.ExecutionControlQuery) (executionclient.ExecutionControlReply, *problem.Problem)
}

type setExecutionControlUseCase interface {
	Execute(context.Context, executionclient.SetExecutionControlCommand) (executionclient.ExecutionControlReply, *problem.Problem)
}

// ExecutionControlWebHandler handles HTTP requests for execution control gate operations.
type ExecutionControlWebHandler struct {
	getControl getExecutionControlUseCase
	setControl setExecutionControlUseCase
}

func NewExecutionControlWebHandler(getControl getExecutionControlUseCase, setControl setExecutionControlUseCase) *ExecutionControlWebHandler {
	return &ExecutionControlWebHandler{getControl: getControl, setControl: setControl}
}

// GetControl handles GET /execution/control
func (h *ExecutionControlWebHandler) GetControl(w http.ResponseWriter, r *http.Request) {
	if pathParam(r, "type") != "control" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "execution control path parameter must be control"))
		return
	}

	if h == nil || h.getControl == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "execution control query is unavailable"))
		return
	}

	result, prob := h.getControl.Execute(r.Context(), executionclient.ExecutionControlQuery{})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// SetControl handles PUT /execution/control
func (h *ExecutionControlWebHandler) SetControl(w http.ResponseWriter, r *http.Request) {
	if pathParam(r, "type") != "control" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "execution control path parameter must be control"))
		return
	}

	if h == nil || h.setControl == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "execution control command is unavailable"))
		return
	}

	cmd, prob := decodeJSONBody[executionclient.SetExecutionControlCommand](r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.setControl.Execute(r.Context(), cmd)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}
