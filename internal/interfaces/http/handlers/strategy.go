package handlers

import (
	"context"
	"net/http"

	"internal/application/strategyclient"
	"internal/domain/strategy"
	"internal/shared/problem"
)

type getLatestStrategyUseCase interface {
	Execute(context.Context, strategyclient.StrategyLatestQuery) (strategyclient.StrategyLatestReply, *problem.Problem)
}

// StrategyWebHandler handles HTTP requests for strategy queries.
type StrategyWebHandler struct {
	getLatestStrategy getLatestStrategyUseCase
}

func NewStrategyWebHandler(getLatestStrategy getLatestStrategyUseCase) *StrategyWebHandler {
	return &StrategyWebHandler{getLatestStrategy: getLatestStrategy}
}

type latestStrategyResponse struct {
	Strategy *strategy.Strategy `json:"strategy"`
}

// GetLatestStrategy handles GET /strategy/:type/latest?source=...&symbol=...&timeframe=...
func (h *StrategyWebHandler) GetLatestStrategy(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getLatestStrategy == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "strategy query is unavailable"))
		return
	}

	strategyType := pathParam(r, "type")
	if strategyType == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "strategy type path parameter is required"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getLatestStrategy.Execute(r.Context(), strategyclient.StrategyLatestQuery{
		Type:       strategyType,
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, latestStrategyResponse{Strategy: result.Strategy})
}
