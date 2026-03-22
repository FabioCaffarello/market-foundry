package handlers

import (
	"context"
	"net/http"

	"internal/application/executionclient"
	"internal/shared/problem"
)

type getSourceExplanationUseCase interface {
	Execute(context.Context, executionclient.SourceExplainQuery) (executionclient.SourceExplainReply, *problem.Problem)
}

// SourceExplainWebHandler handles HTTP requests for source-driven path explainability.
type SourceExplainWebHandler struct {
	getExplanation getSourceExplanationUseCase
}

func NewSourceExplainWebHandler(getExplanation getSourceExplanationUseCase) *SourceExplainWebHandler {
	return &SourceExplainWebHandler{getExplanation: getExplanation}
}

// GetExplanation handles GET /execution/source-explain
// Optional query params: source, symbol, timeframe (when provided, includes last intent/result).
func (h *SourceExplainWebHandler) GetExplanation(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getExplanation == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "source explanation query is unavailable"))
		return
	}

	query := executionclient.SourceExplainQuery{}
	if source := r.URL.Query().Get("source"); source != "" {
		key, prob := parseQueryKeyParams(r)
		if prob != nil {
			writeProblemResponse(w, prob)
			return
		}
		query.Source = key.Source
		query.Symbol = key.Symbol
		query.Timeframe = key.Timeframe
	}

	result, prob := h.getExplanation.Execute(r.Context(), query)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}
