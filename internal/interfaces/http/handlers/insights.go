package handlers

import (
	"context"
	"net/http"
	"strconv"

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

// getLatestCrossVenueUseCase is the handler's view of the cross-venue use case.
type getLatestCrossVenueUseCase interface {
	Execute(context.Context, insightsclient.CrossVenueLatestQuery) (insightsclient.CrossVenueLatestReply, *problem.Problem)
}

// InsightsWebHandler serves the insights read surface (ADR-0027:
// decision-support, read-only).
type InsightsWebHandler struct {
	getLatestVolumeProfile getLatestVolumeProfileUseCase
	getLatestTPOProfile    getLatestTPOProfileUseCase
	getLatestCrossVenue    getLatestCrossVenueUseCase
}

func NewInsightsWebHandler(getLatestVolumeProfile getLatestVolumeProfileUseCase, getLatestTPOProfile getLatestTPOProfileUseCase, getLatestCrossVenue getLatestCrossVenueUseCase) *InsightsWebHandler {
	return &InsightsWebHandler{
		getLatestVolumeProfile: getLatestVolumeProfile,
		getLatestTPOProfile:    getLatestTPOProfile,
		getLatestCrossVenue:    getLatestCrossVenue,
	}
}

type latestVolumeProfileResponse struct {
	VolumeProfile *insights.VolumeProfile `json:"volume_profile"`
}

type latestTPOProfileResponse struct {
	TPOProfile *insights.TPOProfile `json:"tpo_profile"`
}

type latestCrossVenueResponse struct {
	CrossVenueSnapshot *insights.CrossVenueSnapshot `json:"cross_venue_snapshot"`
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

// GetLatestCrossVenue handles
// GET /insights/cross-venue/latest?base=...&quote=...&contract=...&timeframe=...
// No source param: cross-venue fusion spans sources (the canonical
// instrument is the join key).
func (h *InsightsWebHandler) GetLatestCrossVenue(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getLatestCrossVenue == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "cross venue query is unavailable"))
		return
	}

	inst, prob := parseRequiredInstrumentParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}
	timeframeStr := r.URL.Query().Get("timeframe")
	if timeframeStr == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "timeframe query parameter is required"))
		return
	}
	timeframe, err := strconv.Atoi(timeframeStr)
	if err != nil {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "timeframe must be a valid integer"))
		return
	}

	result, prob := h.getLatestCrossVenue.Execute(r.Context(), insightsclient.CrossVenueLatestQuery{
		Instrument: inst,
		Timeframe:  timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, latestCrossVenueResponse{CrossVenueSnapshot: result.CrossVenueSnapshot})
}
