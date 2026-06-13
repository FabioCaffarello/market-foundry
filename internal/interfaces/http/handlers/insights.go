package handlers

import (
	"context"
	"net/http"

	"internal/application/insightsclient"
	"internal/domain/insights"
	"internal/shared/problem"
)

// getLatestVolumeProfileUseCase is the handler's view of the use case
// (avoids importing the concrete type).
type getLatestVolumeProfileUseCase interface {
	Execute(context.Context, insightsclient.VolumeProfileLatestQuery) (insightsclient.VolumeProfileLatestReply, *problem.Problem)
}

// getLatestTPOProfileUseCase is the handler's view of the TPO use case.
type getLatestTPOProfileUseCase interface {
	Execute(context.Context, insightsclient.TPOProfileLatestQuery) (insightsclient.TPOProfileLatestReply, *problem.Problem)
}

// InsightsWebHandler serves the insights read surface (ADR-0027:
// decision-support, read-only).
type InsightsWebHandler struct {
	getLatestVolumeProfile getLatestVolumeProfileUseCase
	getLatestTPOProfile    getLatestTPOProfileUseCase
}

func NewInsightsWebHandler(getLatestVolumeProfile getLatestVolumeProfileUseCase, getLatestTPOProfile getLatestTPOProfileUseCase) *InsightsWebHandler {
	return &InsightsWebHandler{
		getLatestVolumeProfile: getLatestVolumeProfile,
		getLatestTPOProfile:    getLatestTPOProfile,
	}
}

type latestVolumeProfileResponse struct {
	VolumeProfile *insights.VolumeProfile `json:"volume_profile"`
}

type latestTPOProfileResponse struct {
	TPOProfile *insights.TPOProfile `json:"tpo_profile"`
}

// GetLatestVolumeProfile handles
// GET /insights/volume-profile/latest?source=...&base=...&quote=...&contract=...&timeframe=...
func (h *InsightsWebHandler) GetLatestVolumeProfile(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getLatestVolumeProfile == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "volume profile query is unavailable"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getLatestVolumeProfile.Execute(r.Context(), insightsclient.VolumeProfileLatestQuery{
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, latestVolumeProfileResponse{VolumeProfile: result.VolumeProfile})
}

// GetLatestTPOProfile handles
// GET /insights/tpo/latest?source=...&base=...&quote=...&contract=...&timeframe=...
func (h *InsightsWebHandler) GetLatestTPOProfile(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getLatestTPOProfile == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "tpo query is unavailable"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getLatestTPOProfile.Execute(r.Context(), insightsclient.TPOProfileLatestQuery{
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, latestTPOProfileResponse{TPOProfile: result.TPOProfile})
}
