package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"internal/application/triageclient"
	"internal/shared/problem"
)

type getSessionTriageUseCase interface {
	Execute(context.Context, triageclient.SessionTriageQuery) (triageclient.SessionTriageReply, *problem.Problem)
}

type getDecisionTriageUseCase interface {
	Execute(context.Context, triageclient.DecisionTriageQuery) (triageclient.DecisionTriageReply, *problem.Problem)
}

type getRoundTripTriageUseCase interface {
	Execute(context.Context, triageclient.RoundTripTriageQuery) (triageclient.RoundTripTriageReply, *problem.Problem)
}

type getTriageOverviewUseCase interface {
	Execute(context.Context, triageclient.TriageOverviewQuery) (triageclient.TriageOverviewReply, *problem.Problem)
}

// TriageWebHandler handles HTTP requests for batch review and triage endpoints.
// S487: Provides severity-ranked triage surfaces for sessions, decisions, and round-trips.
type TriageWebHandler struct {
	getSessionTriage   getSessionTriageUseCase
	getDecisionTriage  getDecisionTriageUseCase
	getRoundTripTriage getRoundTripTriageUseCase
	getTriageOverview  getTriageOverviewUseCase
	logger             *slog.Logger
}

// TriageHandlerDeps groups all dependencies for the triage HTTP handler.
type TriageHandlerDeps struct {
	GetSessionTriage   getSessionTriageUseCase
	GetDecisionTriage  getDecisionTriageUseCase
	GetRoundTripTriage getRoundTripTriageUseCase
	GetTriageOverview  getTriageOverviewUseCase
	Logger             *slog.Logger
}

func NewTriageWebHandler(deps TriageHandlerDeps) *TriageWebHandler {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &TriageWebHandler{
		getSessionTriage:   deps.GetSessionTriage,
		getDecisionTriage:  deps.GetDecisionTriage,
		getRoundTripTriage: deps.GetRoundTripTriage,
		getTriageOverview:  deps.GetTriageOverview,
		logger:             logger.With("component", "triage_handler"),
	}
}

// GetSessionTriage handles GET /analytical/triage/sessions
//
// Returns sessions ranked by anomaly severity with optional check and severity filters.
// Query parameters:
//   - status: session status filter (e.g. "closed")
//   - check: filter to sessions failing a specific PO check (e.g. "PO-1")
//   - severity: minimum severity filter ("critical", "warning")
//   - limit: max items returned (default 20, max 50)
func (h *TriageWebHandler) GetSessionTriage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getSessionTriage == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "session triage is unavailable"))
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	result, prob := h.getSessionTriage.Execute(r.Context(), triageclient.SessionTriageQuery{
		StatusFilter:   r.URL.Query().Get("status"),
		CheckFilter:    r.URL.Query().Get("check"),
		SeverityFilter: r.URL.Query().Get("severity"),
		Limit:          limit,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("session triage request failed", "total_ms", totalMs, "problem", prob.Code)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d", totalMs))

	writeJSONResponse(w, http.StatusOK, result)
}

// GetDecisionTriage handles GET /analytical/triage/decisions
//
// Returns decisions with consistency violations or incomplete chains, ranked by severity.
// Query parameters:
//   - source, symbol, timeframe: required partition key
//   - since, until: optional time range (unix seconds)
//   - severity: minimum severity filter ("critical", "warning")
//   - limit: max items returned (default 20, max 100)
func (h *TriageWebHandler) GetDecisionTriage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getDecisionTriage == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "decision triage is unavailable"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	params, prob := parseAnalyticalParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	result, dProb := h.getDecisionTriage.Execute(r.Context(), triageclient.DecisionTriageQuery{
		Source:         key.Source,
		Instrument:     key.Instrument,
		Timeframe:      key.Timeframe,
		Since:          params.Since,
		Until:          params.Until,
		Limit:          limit,
		SeverityFilter: r.URL.Query().Get("severity"),
	})
	if dProb != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("decision triage request failed",
			"source", key.Source, "instrument", key.Instrument.Symbol(), "timeframe", key.Timeframe,
			"total_ms", totalMs, "problem", dProb.Code,
		)
		writeProblemResponse(w, dProb)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d", totalMs))

	writeJSONResponse(w, http.StatusOK, result)
}

// GetRoundTripTriage handles GET /analytical/triage/roundtrips
//
// Returns flagged round-trips ranked by data quality severity.
// Query parameters:
//   - source, symbol, timeframe: required partition key
//   - since, until: optional time range (unix seconds)
//   - severity: minimum severity filter ("critical", "warning")
//   - limit: max items returned (default 50, max 200)
func (h *TriageWebHandler) GetRoundTripTriage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getRoundTripTriage == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "round-trip triage is unavailable"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	params, prob := parseAnalyticalParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	result, rtProb := h.getRoundTripTriage.Execute(r.Context(), triageclient.RoundTripTriageQuery{
		Source:         key.Source,
		Instrument:     key.Instrument,
		Timeframe:      key.Timeframe,
		Since:          params.Since,
		Until:          params.Until,
		Limit:          limit,
		SeverityFilter: r.URL.Query().Get("severity"),
	})
	if rtProb != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("round-trip triage request failed",
			"source", key.Source, "instrument", key.Instrument.Symbol(), "timeframe", key.Timeframe,
			"total_ms", totalMs, "problem", rtProb.Code,
		)
		writeProblemResponse(w, rtProb)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d", totalMs))

	writeJSONResponse(w, http.StatusOK, result)
}

// GetTriageOverview handles GET /analytical/triage/overview
//
// Returns a cross-domain "what needs attention?" summary combining session,
// decision, and round-trip triage signals.
// Query parameters:
//   - session_status: optional session status filter
//   - source, symbol, timeframe: optional (enables decision and round-trip triage)
//   - since, until: optional time range (unix seconds)
func (h *TriageWebHandler) GetTriageOverview(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getTriageOverview == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "triage overview is unavailable"))
		return
	}

	timeframe, _ := strconv.Atoi(r.URL.Query().Get("timeframe"))
	since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
	until, _ := strconv.ParseInt(r.URL.Query().Get("until"), 10, 64)

	inst, instProb := parseOptionalInstrumentParams(r)
	if instProb != nil {
		writeProblemResponse(w, instProb)
		return
	}

	result, prob := h.getTriageOverview.Execute(r.Context(), triageclient.TriageOverviewQuery{
		SessionStatus: r.URL.Query().Get("session_status"),
		Source:        r.URL.Query().Get("source"),
		Instrument:    inst,
		Timeframe:     timeframe,
		Since:         since,
		Until:         until,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("triage overview request failed", "total_ms", totalMs, "problem", prob.Code)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d", totalMs))

	writeJSONResponse(w, http.StatusOK, result)
}
