package handlers

import (
	"context"
	"net/http"

	"internal/application/executionclient"
	"internal/shared/problem"
)

type getActivationSurfaceUseCase interface {
	Execute(context.Context, executionclient.ActivationSurfaceQuery) (executionclient.ActivationSurfaceReply, *problem.Problem)
}

// ActivationWebHandler handles HTTP requests for activation surface queries.
type ActivationWebHandler struct {
	getSurface getActivationSurfaceUseCase
}

func NewActivationWebHandler(getSurface getActivationSurfaceUseCase) *ActivationWebHandler {
	return &ActivationWebHandler{getSurface: getSurface}
}

// GetSurface handles GET /activation/surface
func (h *ActivationWebHandler) GetSurface(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getSurface == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "activation surface query is unavailable"))
		return
	}

	result, prob := h.getSurface.Execute(r.Context(), executionclient.ActivationSurfaceQuery{})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}
