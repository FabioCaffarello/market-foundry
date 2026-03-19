package handlers

import (
	"context"
	"net/http"

	"internal/application/decisionclient"
	"internal/domain/decision"
	"internal/shared/problem"
)

type getLatestDecisionUseCase interface {
	Execute(context.Context, decisionclient.DecisionLatestQuery) (decisionclient.DecisionLatestReply, *problem.Problem)
}

// DecisionWebHandler handles HTTP requests for decision queries.
type DecisionWebHandler struct {
	getLatestDecision getLatestDecisionUseCase
}

func NewDecisionWebHandler(getLatestDecision getLatestDecisionUseCase) *DecisionWebHandler {
	return &DecisionWebHandler{getLatestDecision: getLatestDecision}
}

type latestDecisionResponse struct {
	Decision *decision.Decision `json:"decision"`
}

// GetLatestDecision handles GET /decision/:type/latest?source=...&symbol=...&timeframe=...
func (h *DecisionWebHandler) GetLatestDecision(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getLatestDecision == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "decision query is unavailable"))
		return
	}

	decisionType := pathParam(r, "type")
	if decisionType == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "decision type path parameter is required"))
		return
	}

	key, prob := parseEvidenceKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getLatestDecision.Execute(r.Context(), decisionclient.DecisionLatestQuery{
		Type:      decisionType,
		Source:    key.Source,
		Symbol:    key.Symbol,
		Timeframe: key.Timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, latestDecisionResponse{Decision: result.Decision})
}
