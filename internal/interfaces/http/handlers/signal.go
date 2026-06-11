package handlers

import (
	"context"
	"net/http"

	"internal/application/signalclient"
	"internal/domain/signal"
	"internal/shared/problem"
)

type getLatestSignalUseCase interface {
	Execute(context.Context, signalclient.SignalLatestQuery) (signalclient.SignalLatestReply, *problem.Problem)
}

// SignalWebHandler handles HTTP requests for signal queries.
type SignalWebHandler struct {
	getLatestSignal getLatestSignalUseCase
}

func NewSignalWebHandler(getLatestSignal getLatestSignalUseCase) *SignalWebHandler {
	return &SignalWebHandler{getLatestSignal: getLatestSignal}
}

type latestSignalResponse struct {
	Signal *signal.Signal `json:"signal"`
}

// GetLatestSignal handles GET /signal/:type/latest?source=...&symbol=...&timeframe=...
// The signal type is extracted from the URL path parameter.
func (h *SignalWebHandler) GetLatestSignal(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getLatestSignal == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "signal query is unavailable"))
		return
	}

	signalType := pathParam(r, "type")
	if signalType == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "signal type path parameter is required"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getLatestSignal.Execute(r.Context(), signalclient.SignalLatestQuery{
		Type:       signalType,
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, latestSignalResponse{Signal: result.Signal})
}
