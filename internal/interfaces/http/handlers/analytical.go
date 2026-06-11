package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"internal/application/analyticalclient"
	"internal/shared/problem"
)

type getAnalyticalCandleHistoryUseCase interface {
	Execute(context.Context, analyticalclient.CandleHistoryQuery) (analyticalclient.CandleHistoryReply, *problem.Problem)
}

type getAnalyticalSignalHistoryUseCase interface {
	Execute(context.Context, analyticalclient.SignalHistoryQuery) (analyticalclient.SignalHistoryReply, *problem.Problem)
}

type getAnalyticalDecisionHistoryUseCase interface {
	Execute(context.Context, analyticalclient.DecisionHistoryQuery) (analyticalclient.DecisionHistoryReply, *problem.Problem)
}

type getAnalyticalStrategyHistoryUseCase interface {
	Execute(context.Context, analyticalclient.StrategyHistoryQuery) (analyticalclient.StrategyHistoryReply, *problem.Problem)
}

type getAnalyticalRiskHistoryUseCase interface {
	Execute(context.Context, analyticalclient.RiskHistoryQuery) (analyticalclient.RiskHistoryReply, *problem.Problem)
}

type getAnalyticalExecutionHistoryUseCase interface {
	Execute(context.Context, analyticalclient.ExecutionHistoryQuery) (analyticalclient.ExecutionHistoryReply, *problem.Problem)
}

type getAnalyticalLifecycleHistoryUseCase interface {
	Execute(context.Context, analyticalclient.LifecycleHistoryQuery) (analyticalclient.LifecycleHistoryReply, *problem.Problem)
}

type getAnalyticalExecutionListUseCase interface {
	Execute(context.Context, analyticalclient.ExecutionListQuery) (analyticalclient.ExecutionListReply, *problem.Problem)
}

type getAnalyticalExecutionSummaryUseCase interface {
	Execute(context.Context, analyticalclient.ExecutionSummaryQuery) (analyticalclient.ExecutionSummaryReply, *problem.Problem)
}

type getAnalyticalSessionExplainUseCase interface {
	Execute(context.Context, analyticalclient.SessionExplainQuery) (analyticalclient.SessionExplainReply, *problem.Problem)
}

// AnalyticalWebHandler handles HTTP requests for analytical (ClickHouse-backed) queries.
// These are additive endpoints under /analytical/ — they do not modify or overlap
// with the existing operational query surface.
type AnalyticalWebHandler struct {
	getCandleHistory    getAnalyticalCandleHistoryUseCase
	getSignalHistory    getAnalyticalSignalHistoryUseCase
	getDecisionHistory  getAnalyticalDecisionHistoryUseCase
	getStrategyHistory  getAnalyticalStrategyHistoryUseCase
	getRiskHistory      getAnalyticalRiskHistoryUseCase
	getExecutionHistory getAnalyticalExecutionHistoryUseCase
	getLifecycleHistory getAnalyticalLifecycleHistoryUseCase
	getExecutionList    getAnalyticalExecutionListUseCase
	getExecutionSummary getAnalyticalExecutionSummaryUseCase
	getSessionExplain   getAnalyticalSessionExplainUseCase
	logger              *slog.Logger
}

// AnalyticalHandlerDeps groups all dependencies for the analytical HTTP handler.
// Struct-based DI replaces the positional constructor to enable unbounded family
// addition without signature churn. Each field is optional — nil disables the
// corresponding endpoint gracefully.
type AnalyticalHandlerDeps struct {
	GetCandleHistory    getAnalyticalCandleHistoryUseCase
	GetSignalHistory    getAnalyticalSignalHistoryUseCase
	GetDecisionHistory  getAnalyticalDecisionHistoryUseCase
	GetStrategyHistory  getAnalyticalStrategyHistoryUseCase
	GetRiskHistory      getAnalyticalRiskHistoryUseCase
	GetExecutionHistory getAnalyticalExecutionHistoryUseCase
	GetLifecycleHistory getAnalyticalLifecycleHistoryUseCase
	GetExecutionList    getAnalyticalExecutionListUseCase
	GetExecutionSummary getAnalyticalExecutionSummaryUseCase
	GetSessionExplain   getAnalyticalSessionExplainUseCase
	Logger              *slog.Logger
}

func NewAnalyticalWebHandler(deps AnalyticalHandlerDeps) *AnalyticalWebHandler {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &AnalyticalWebHandler{
		getCandleHistory:    deps.GetCandleHistory,
		getSignalHistory:    deps.GetSignalHistory,
		getDecisionHistory:  deps.GetDecisionHistory,
		getStrategyHistory:  deps.GetStrategyHistory,
		getRiskHistory:      deps.GetRiskHistory,
		getExecutionHistory: deps.GetExecutionHistory,
		getLifecycleHistory: deps.GetLifecycleHistory,
		getExecutionList:    deps.GetExecutionList,
		getExecutionSummary: deps.GetExecutionSummary,
		getSessionExplain:   deps.GetSessionExplain,
		logger:              logger.With("component", "analytical_handler"),
	}
}

// Analytical query defaults and bounds.
const (
	AnalyticalDefaultLimit = 50
	AnalyticalMinLimit     = 1
	AnalyticalMaxLimit     = 500
)

// analyticalParams holds the common pagination and time-range parameters
// shared by all analytical handler methods.
type analyticalParams struct {
	Limit int
	Since int64
	Until int64
}

// parseAnalyticalParams extracts limit, since, and until from query string.
// Defaults: limit=AnalyticalDefaultLimit, since=0, until=0 (no filter).
func parseAnalyticalParams(r *http.Request) (analyticalParams, *problem.Problem) {
	limit := AnalyticalDefaultLimit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil {
			return analyticalParams{}, problem.New(problem.InvalidArgument, "limit must be a valid integer")
		}
		if parsed < AnalyticalMinLimit || parsed > AnalyticalMaxLimit {
			return analyticalParams{}, problem.New(problem.InvalidArgument,
				fmt.Sprintf("limit must be between %d and %d", AnalyticalMinLimit, AnalyticalMaxLimit))
		}
		limit = parsed
	}

	var since, until int64
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		parsed, err := strconv.ParseInt(sinceStr, 10, 64)
		if err != nil {
			return analyticalParams{}, problem.New(problem.InvalidArgument, "since must be a valid unix timestamp")
		}
		since = parsed
	}
	if untilStr := r.URL.Query().Get("until"); untilStr != "" {
		parsed, err := strconv.ParseInt(untilStr, 10, 64)
		if err != nil {
			return analyticalParams{}, problem.New(problem.InvalidArgument, "until must be a valid unix timestamp")
		}
		until = parsed
	}

	return analyticalParams{Limit: limit, Since: since, Until: until}, nil
}

type analyticalCandleHistoryResponse struct {
	Candles any                        `json:"candles"`
	Source  string                     `json:"source"`
	Meta    analyticalclient.QueryMeta `json:"meta"`
}

// GetCandleHistory handles GET /analytical/evidence/candles?source=...&symbol=...&timeframe=...&since=...&until=...&limit=...
func (h *AnalyticalWebHandler) GetCandleHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getCandleHistory == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "analytical query is unavailable"))
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

	result, prob := h.getCandleHistory.Execute(r.Context(), analyticalclient.CandleHistoryQuery{
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
		Limit:      params.Limit,
		Since:      params.Since,
		Until:      params.Until,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("analytical request failed",
			"source", key.Source, "instrument", key.Instrument.Symbol(), "timeframe", key.Timeframe,
			"total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.QueryMs))

	writeJSONResponse(w, http.StatusOK, analyticalCandleHistoryResponse{
		Candles: result.Candles,
		Source:  result.Source,
		Meta:    result.Meta,
	})
}

type analyticalSignalHistoryResponse struct {
	Signals any                        `json:"signals"`
	Source  string                     `json:"source"`
	Meta    analyticalclient.QueryMeta `json:"meta"`
}

// GetSignalHistory handles GET /analytical/signal/history?type=...&source=...&symbol=...&timeframe=...&since=...&until=...&limit=...
func (h *AnalyticalWebHandler) GetSignalHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getSignalHistory == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "analytical signal query is unavailable"))
		return
	}

	signalType := r.URL.Query().Get("type")
	if signalType == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "type query parameter is required"))
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

	result, prob := h.getSignalHistory.Execute(r.Context(), analyticalclient.SignalHistoryQuery{
		Type:       signalType,
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
		Limit:      params.Limit,
		Since:      params.Since,
		Until:      params.Until,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("analytical signal request failed",
			"type", signalType, "source", key.Source, "instrument", key.Instrument.Symbol(), "timeframe", key.Timeframe,
			"total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.QueryMs))

	writeJSONResponse(w, http.StatusOK, analyticalSignalHistoryResponse{
		Signals: result.Signals,
		Source:  result.Source,
		Meta:    result.Meta,
	})
}

type analyticalDecisionHistoryResponse struct {
	Decisions any                        `json:"decisions"`
	Source    string                     `json:"source"`
	Meta      analyticalclient.QueryMeta `json:"meta"`
}

// GetDecisionHistory handles GET /analytical/decision/history?type=...&source=...&symbol=...&timeframe=...&outcome=...&since=...&until=...&limit=...
func (h *AnalyticalWebHandler) GetDecisionHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getDecisionHistory == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "analytical decision query is unavailable"))
		return
	}

	decisionType := r.URL.Query().Get("type")
	if decisionType == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "type query parameter is required"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	outcome := r.URL.Query().Get("outcome")

	params, prob := parseAnalyticalParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getDecisionHistory.Execute(r.Context(), analyticalclient.DecisionHistoryQuery{
		Type:       decisionType,
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
		Outcome:    outcome,
		Limit:      params.Limit,
		Since:      params.Since,
		Until:      params.Until,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("analytical decision request failed",
			"type", decisionType, "source", key.Source, "instrument", key.Instrument.Symbol(), "timeframe", key.Timeframe,
			"outcome", outcome, "total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.QueryMs))

	writeJSONResponse(w, http.StatusOK, analyticalDecisionHistoryResponse{
		Decisions: result.Decisions,
		Source:    result.Source,
		Meta:      result.Meta,
	})
}

type analyticalStrategyHistoryResponse struct {
	Strategies any                        `json:"strategies"`
	Source     string                     `json:"source"`
	Meta       analyticalclient.QueryMeta `json:"meta"`
}

// GetStrategyHistory handles GET /analytical/strategy/history?type=...&source=...&symbol=...&timeframe=...&direction=...&since=...&until=...&limit=...
func (h *AnalyticalWebHandler) GetStrategyHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getStrategyHistory == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "analytical strategy query is unavailable"))
		return
	}

	strategyType := r.URL.Query().Get("type")
	if strategyType == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "type query parameter is required"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	direction := r.URL.Query().Get("direction")

	params, prob := parseAnalyticalParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getStrategyHistory.Execute(r.Context(), analyticalclient.StrategyHistoryQuery{
		Type:       strategyType,
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
		Direction:  direction,
		Limit:      params.Limit,
		Since:      params.Since,
		Until:      params.Until,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("analytical strategy request failed",
			"type", strategyType, "source", key.Source, "instrument", key.Instrument.Symbol(), "timeframe", key.Timeframe,
			"direction", direction, "total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.QueryMs))

	writeJSONResponse(w, http.StatusOK, analyticalStrategyHistoryResponse{
		Strategies: result.Strategies,
		Source:     result.Source,
		Meta:       result.Meta,
	})
}

type analyticalRiskHistoryResponse struct {
	RiskAssessments any                        `json:"risk_assessments"`
	Source          string                     `json:"source"`
	Meta            analyticalclient.QueryMeta `json:"meta"`
}

// GetRiskHistory handles GET /analytical/risk/history?type=...&source=...&symbol=...&timeframe=...&disposition=...&since=...&until=...&limit=...
func (h *AnalyticalWebHandler) GetRiskHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getRiskHistory == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "analytical risk query is unavailable"))
		return
	}

	riskType := r.URL.Query().Get("type")
	if riskType == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "type query parameter is required"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	disposition := r.URL.Query().Get("disposition")

	params, prob := parseAnalyticalParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getRiskHistory.Execute(r.Context(), analyticalclient.RiskHistoryQuery{
		Type:        riskType,
		Source:      key.Source,
		Instrument:  key.Instrument,
		Timeframe:   key.Timeframe,
		Disposition: disposition,
		Limit:       params.Limit,
		Since:       params.Since,
		Until:       params.Until,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("analytical risk request failed",
			"type", riskType, "source", key.Source, "instrument", key.Instrument.Symbol(), "timeframe", key.Timeframe,
			"disposition", disposition, "total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.QueryMs))

	writeJSONResponse(w, http.StatusOK, analyticalRiskHistoryResponse{
		RiskAssessments: result.RiskAssessments,
		Source:          result.Source,
		Meta:            result.Meta,
	})
}

type analyticalExecutionHistoryResponse struct {
	Executions any                        `json:"executions"`
	Source     string                     `json:"source"`
	Meta       analyticalclient.QueryMeta `json:"meta"`
}

// GetExecutionHistory handles GET /analytical/execution/history?type=...&source=...&symbol=...&timeframe=...&side=...&status=...&since=...&until=...&limit=...
func (h *AnalyticalWebHandler) GetExecutionHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getExecutionHistory == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "analytical execution query is unavailable"))
		return
	}

	execType := r.URL.Query().Get("type")
	if execType == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "type query parameter is required"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	side := r.URL.Query().Get("side")
	status := r.URL.Query().Get("status")

	params, prob := parseAnalyticalParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getExecutionHistory.Execute(r.Context(), analyticalclient.ExecutionHistoryQuery{
		Type:       execType,
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
		Side:       side,
		Status:     status,
		Limit:      params.Limit,
		Since:      params.Since,
		Until:      params.Until,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("analytical execution request failed",
			"type", execType, "source", key.Source, "instrument", key.Instrument.Symbol(), "timeframe", key.Timeframe,
			"side", side, "status", status, "total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.QueryMs))

	writeJSONResponse(w, http.StatusOK, analyticalExecutionHistoryResponse{
		Executions: result.Executions,
		Source:     result.Source,
		Meta:       result.Meta,
	})
}

type analyticalLifecycleHistoryResponse struct {
	Entries any                        `json:"entries"`
	Source  string                     `json:"source"`
	Meta    analyticalclient.QueryMeta `json:"meta"`
}

// GetLifecycleHistory handles GET /analytical/execution/lifecycle?source=...&symbol=...&timeframe=...&side=...&status=...&since=...&until=...&limit=...
// S453A: Returns a unified chronological timeline of all execution event types
// for a given source/symbol/timeframe — enabling lifecycle reconstruction.
func (h *AnalyticalWebHandler) GetLifecycleHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getLifecycleHistory == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "analytical lifecycle query is unavailable"))
		return
	}

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	side := r.URL.Query().Get("side")
	status := r.URL.Query().Get("status")

	params, prob := parseAnalyticalParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getLifecycleHistory.Execute(r.Context(), analyticalclient.LifecycleHistoryQuery{
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
		Side:       side,
		Status:     status,
		Limit:      params.Limit,
		Since:      params.Since,
		Until:      params.Until,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("analytical lifecycle request failed",
			"source", key.Source, "instrument", key.Instrument.Symbol(), "timeframe", key.Timeframe,
			"side", side, "status", status, "total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.QueryMs))

	writeJSONResponse(w, http.StatusOK, analyticalLifecycleHistoryResponse{
		Entries: result.Entries,
		Source:  result.Source,
		Meta:    result.Meta,
	})
}

type analyticalExecutionListResponse struct {
	Entries any                        `json:"entries"`
	Source  string                     `json:"source"`
	Meta    analyticalclient.QueryMeta `json:"meta"`
}

// GetExecutionList handles GET /analytical/execution/list?type=...&source=...&symbol=...&timeframe=...&side=...&status=...&since=...&until=...&limit=...
// S454A: Relaxed-filter execution list — at least one filter required, but none individually mandatory.
func (h *AnalyticalWebHandler) GetExecutionList(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getExecutionList == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "analytical execution list query is unavailable"))
		return
	}

	execType := r.URL.Query().Get("type")
	source := r.URL.Query().Get("source")
	inst, instProb := parseOptionalInstrumentParams(r)
	if instProb != nil {
		writeProblemResponse(w, instProb)
		return
	}
	side := r.URL.Query().Get("side")
	status := r.URL.Query().Get("status")

	var timeframe int
	if tfStr := r.URL.Query().Get("timeframe"); tfStr != "" {
		parsed, err := strconv.Atoi(tfStr)
		if err != nil {
			writeProblemResponse(w, problem.New(problem.InvalidArgument, "timeframe must be a valid integer"))
			return
		}
		timeframe = parsed
	}

	params, prob := parseAnalyticalParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getExecutionList.Execute(r.Context(), analyticalclient.ExecutionListQuery{
		Type:       execType,
		Source:     source,
		Instrument: inst,
		Timeframe:  timeframe,
		Side:       side,
		Status:     status,
		Limit:      params.Limit,
		Since:      params.Since,
		Until:      params.Until,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("analytical execution list request failed",
			"type", execType, "source", source, "instrument", inst.Symbol(), "timeframe", timeframe,
			"side", side, "status", status, "total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.QueryMs))

	writeJSONResponse(w, http.StatusOK, analyticalExecutionListResponse{
		Entries: result.Entries,
		Source:  result.Source,
		Meta:    result.Meta,
	})
}

type analyticalExecutionSummaryResponse struct {
	Entries any                        `json:"entries"`
	Source  string                     `json:"source"`
	Meta    analyticalclient.QueryMeta `json:"meta"`
}

// GetExecutionSummary handles GET /analytical/execution/summary?source=...&symbol=...&timeframe=...&since=...&until=...
// S454A: Returns execution counts grouped by (type, status) for operational overview.
func (h *AnalyticalWebHandler) GetExecutionSummary(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getExecutionSummary == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "analytical execution summary query is unavailable"))
		return
	}

	source := r.URL.Query().Get("source")
	inst, instProb := parseOptionalInstrumentParams(r)
	if instProb != nil {
		writeProblemResponse(w, instProb)
		return
	}

	var timeframe int
	if tfStr := r.URL.Query().Get("timeframe"); tfStr != "" {
		parsed, err := strconv.Atoi(tfStr)
		if err != nil {
			writeProblemResponse(w, problem.New(problem.InvalidArgument, "timeframe must be a valid integer"))
			return
		}
		timeframe = parsed
	}

	params, prob := parseAnalyticalParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getExecutionSummary.Execute(r.Context(), analyticalclient.ExecutionSummaryQuery{
		Source:     source,
		Instrument: inst,
		Timeframe:  timeframe,
		Since:      params.Since,
		Until:      params.Until,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("analytical execution summary request failed",
			"source", source, "instrument", inst.Symbol(), "timeframe", timeframe,
			"total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.QueryMs))

	writeJSONResponse(w, http.StatusOK, analyticalExecutionSummaryResponse{
		Entries: result.Entries,
		Source:  result.Source,
		Meta:    result.Meta,
	})
}

// GetSessionExplain handles GET /analytical/execution/explain?source=...&symbol=...&timeframe=...&limit=...
// S455A: Returns a unified explanation combining KV state, ClickHouse history, and cross-surface consistency.
func (h *AnalyticalWebHandler) GetSessionExplain(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getSessionExplain == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "session explain query is unavailable"))
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

	result, prob := h.getSessionExplain.Execute(r.Context(), analyticalclient.SessionExplainQuery{
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
		Limit:      params.Limit,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("session explain request failed",
			"source", key.Source, "instrument", key.Instrument.Symbol(), "timeframe", key.Timeframe,
			"total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.QueryMs))

	writeJSONResponse(w, http.StatusOK, result)
}
