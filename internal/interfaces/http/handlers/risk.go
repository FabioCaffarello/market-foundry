package handlers

import (
	"context"
	"net/http"

	"internal/application/riskclient"
	"internal/domain/risk"
	"internal/shared/problem"
)

type getLatestRiskUseCase interface {
	Execute(context.Context, riskclient.RiskLatestQuery) (riskclient.RiskLatestReply, *problem.Problem)
}

// RiskWebHandler handles HTTP requests for risk queries.
type RiskWebHandler struct {
	getLatestRisk getLatestRiskUseCase
}

func NewRiskWebHandler(getLatestRisk getLatestRiskUseCase) *RiskWebHandler {
	return &RiskWebHandler{getLatestRisk: getLatestRisk}
}

type latestRiskResponse struct {
	RiskAssessment *risk.RiskAssessment `json:"risk_assessment"`
}

// GetLatestRisk handles GET /risk/:type/latest?source=...&symbol=...&timeframe=...
func (h *RiskWebHandler) GetLatestRisk(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getLatestRisk == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "risk query is unavailable"))
		return
	}

	riskType := pathParam(r, "type")
	if riskType == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "risk type path parameter is required"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getLatestRisk.Execute(r.Context(), riskclient.RiskLatestQuery{
		Type:      riskType,
		Source:    key.Source,
		Symbol:    key.Symbol,
		Timeframe: key.Timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, latestRiskResponse{RiskAssessment: result.RiskAssessment})
}
