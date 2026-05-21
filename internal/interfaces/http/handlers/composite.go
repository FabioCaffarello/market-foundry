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

type getCompositeChainUseCase interface {
	Execute(context.Context, analyticalclient.CompositeChainQuery) (analyticalclient.CompositeChainReply, *problem.Problem)
}

type getPipelineFunnelUseCase interface {
	Execute(context.Context, analyticalclient.PipelineFunnelQuery) (analyticalclient.PipelineFunnelReply, *problem.Problem)
}

type getDispositionBreakdownUseCase interface {
	Execute(context.Context, analyticalclient.DispositionBreakdownQuery) (analyticalclient.DispositionBreakdownReply, *problem.Problem)
}

type getDecisionReviewUseCase interface {
	Execute(context.Context, analyticalclient.DecisionReviewQuery) (analyticalclient.DecisionReviewReply, *problem.Problem)
}

type getEffectivenessUseCase interface {
	Execute(context.Context, analyticalclient.EffectivenessQuery) (analyticalclient.EffectivenessReply, *problem.Problem)
}

type getEffectivenessSummaryUseCase interface {
	Execute(context.Context, analyticalclient.EffectivenessSummaryQuery) (analyticalclient.EffectivenessSummaryReply, *problem.Problem)
}

type getPairingUseCase interface {
	Execute(context.Context, analyticalclient.PairingQuery) (analyticalclient.PairingReply, *problem.Problem)
}

type getRoundTripReviewUseCase interface {
	Execute(context.Context, analyticalclient.RoundTripReviewQuery) (analyticalclient.RoundTripReviewReply, *problem.Problem)
}

type getCrossSessionPairingUseCase interface {
	Execute(context.Context, analyticalclient.CrossSessionPairingQuery) (analyticalclient.CrossSessionPairingReply, *problem.Problem)
}

type getContinuityReviewUseCase interface {
	Execute(context.Context, analyticalclient.ContinuityReviewQuery) (analyticalclient.ContinuityReviewReply, *problem.Problem)
}

// CompositeWebHandler handles HTTP requests for composite execution chain queries
// and aggregation endpoints. These expose the composite read model (S296) and
// aggregation layer (S298) as an HTTP query surface for Q1–Q7.
// S471: Decision review endpoint added for decision-centric review and evidence bundling.
// S476: Effectiveness evaluation endpoints added for batch measurement read surfaces.
// S477: Effectiveness summary and comparative analysis endpoints added.
// S481: Round-trip pairing read model endpoint added.
// S482: Round-trip review and outcome reconciliation endpoint added.
// S495: Cross-session pairing read model endpoint added.
type CompositeWebHandler struct {
	getCompositeChain        getCompositeChainUseCase
	getPipelineFunnel        getPipelineFunnelUseCase
	getDispositionBreakdown  getDispositionBreakdownUseCase
	getDecisionReview        getDecisionReviewUseCase
	getEffectiveness         getEffectivenessUseCase
	getEffectivenessSummary  getEffectivenessSummaryUseCase
	getPairing               getPairingUseCase
	getRoundTripReview       getRoundTripReviewUseCase
	getCrossSessionPairing   getCrossSessionPairingUseCase
	getContinuityReview      getContinuityReviewUseCase
	logger                   *slog.Logger
}

// CompositeHandlerDeps groups all dependencies for the composite HTTP handler.
type CompositeHandlerDeps struct {
	GetCompositeChain        getCompositeChainUseCase
	GetPipelineFunnel        getPipelineFunnelUseCase
	GetDispositionBreakdown  getDispositionBreakdownUseCase
	GetDecisionReview        getDecisionReviewUseCase
	GetEffectiveness         getEffectivenessUseCase
	GetEffectivenessSummary  getEffectivenessSummaryUseCase
	GetPairing               getPairingUseCase
	GetRoundTripReview       getRoundTripReviewUseCase
	GetCrossSessionPairing   getCrossSessionPairingUseCase
	GetContinuityReview      getContinuityReviewUseCase
	Logger                   *slog.Logger
}

func NewCompositeWebHandler(deps CompositeHandlerDeps) *CompositeWebHandler {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &CompositeWebHandler{
		getCompositeChain:        deps.GetCompositeChain,
		getPipelineFunnel:        deps.GetPipelineFunnel,
		getDispositionBreakdown:  deps.GetDispositionBreakdown,
		getDecisionReview:        deps.GetDecisionReview,
		getEffectiveness:         deps.GetEffectiveness,
		getEffectivenessSummary:  deps.GetEffectivenessSummary,
		getPairing:               deps.GetPairing,
		getRoundTripReview:       deps.GetRoundTripReview,
		getCrossSessionPairing:   deps.GetCrossSessionPairing,
		getContinuityReview:      deps.GetContinuityReview,
		logger:                   logger.With("component", "composite_handler"),
	}
}

type compositeChainResponse struct {
	Chains []analyticalclient.CompositeExecutionChain `json:"chains"`
	Source string                                     `json:"source"`
	Meta   analyticalclient.CompositeQueryMeta        `json:"meta"`
}

// GetChain handles GET /analytical/composite/chain?correlation_id=...&symbol=...
//
// Returns the full causal chain for a single correlation_id, scoped to the given symbol.
// The symbol parameter is mandatory to guarantee cross-symbol isolation (S301).
// This is the primary endpoint for answering Q1 (why was execution X submitted?) and Q3
// (which signals contributed to decision D?). The attribution field (S298) directly answers Q2.
func (h *CompositeWebHandler) GetChain(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getCompositeChain == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "composite chain query is unavailable"))
		return
	}

	correlationID := r.URL.Query().Get("correlation_id")
	if correlationID == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "correlation_id query parameter is required"))
		return
	}
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "symbol query parameter is required for isolation (S301)"))
		return
	}

	result, prob := h.getCompositeChain.Execute(r.Context(), analyticalclient.CompositeChainQuery{
		CorrelationID: correlationID,
		Symbol:        symbol,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("composite chain request failed",
			"correlation_id", correlationID, "symbol", symbol, "total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, compositeChainResponse{
		Chains: result.Chains,
		Source: result.Source,
		Meta:   result.Meta,
	})
}

// GetChains handles GET /analytical/composite/chains?source=...&symbol=...&timeframe=...&since=...&until=...&limit=...
//
// Returns recent composite execution chains for a given source/symbol/timeframe.
// Each chain now includes an attribution field (S298) for direct Q2 explainability.
func (h *CompositeWebHandler) GetChains(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getCompositeChain == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "composite chain query is unavailable"))
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

	result, prob := h.getCompositeChain.Execute(r.Context(), analyticalclient.CompositeChainQuery{
		Source:    key.Source,
		Symbol:    key.Symbol,
		Timeframe: key.Timeframe,
		Since:     params.Since,
		Until:     params.Until,
		Limit:     params.Limit,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("composite chains request failed",
			"source", key.Source, "symbol", key.Symbol, "timeframe", key.Timeframe,
			"total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, compositeChainResponse{
		Chains: result.Chains,
		Source: result.Source,
		Meta:   result.Meta,
	})
}

type pipelineFunnelResponse struct {
	Stages []analyticalclient.StageFunnelCount `json:"stages"`
	Source string                              `json:"source"`
	Meta   analyticalclient.CompositeQueryMeta `json:"meta"`
}

// GetFunnel handles GET /analytical/composite/funnel?type=...&source=...&symbol=...&timeframe=...&since=...&until=...
//
// Returns event counts per pipeline stage for Q7 (conversion rate per stage per family)
// and Q5 (where did the pipeline break for symbol S?).
func (h *CompositeWebHandler) GetFunnel(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getPipelineFunnel == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "pipeline funnel query is unavailable"))
		return
	}

	typ := r.URL.Query().Get("type")
	if typ == "" {
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

	result, prob := h.getPipelineFunnel.Execute(r.Context(), analyticalclient.PipelineFunnelQuery{
		Type:      typ,
		Source:    key.Source,
		Symbol:    key.Symbol,
		Timeframe: key.Timeframe,
		Since:     params.Since,
		Until:     params.Until,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("pipeline funnel request failed",
			"type", typ, "source", key.Source, "symbol", key.Symbol, "timeframe", key.Timeframe,
			"total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, pipelineFunnelResponse{
		Stages: result.Stages,
		Source: result.Source,
		Meta:   result.Meta,
	})
}

type dispositionBreakdownResponse struct {
	Dispositions []analyticalclient.DispositionCount `json:"dispositions"`
	Total        int64                               `json:"total"`
	Source       string                              `json:"source"`
	Meta         analyticalclient.CompositeQueryMeta `json:"meta"`
}

// GetDispositions handles GET /analytical/composite/dispositions?type=...&source=...&symbol=...&timeframe=...&since=...&until=...
//
// Returns risk disposition counts (approved/modified/rejected) with percentages for Q6.
func (h *CompositeWebHandler) GetDispositions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getDispositionBreakdown == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "disposition breakdown query is unavailable"))
		return
	}

	typ := r.URL.Query().Get("type")
	if typ == "" {
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

	result, prob := h.getDispositionBreakdown.Execute(r.Context(), analyticalclient.DispositionBreakdownQuery{
		Type:      typ,
		Source:    key.Source,
		Symbol:    key.Symbol,
		Timeframe: key.Timeframe,
		Since:     params.Since,
		Until:     params.Until,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("disposition breakdown request failed",
			"type", typ, "source", key.Source, "symbol", key.Symbol, "timeframe", key.Timeframe,
			"total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, dispositionBreakdownResponse{
		Dispositions: result.Dispositions,
		Total:        result.Total,
		Source:       result.Source,
		Meta:         result.Meta,
	})
}

type decisionReviewResponse struct {
	Reviews []analyticalclient.DecisionReviewBundle `json:"reviews"`
	Source  string                                  `json:"source"`
	Meta    analyticalclient.CompositeQueryMeta     `json:"meta"`
}

// GetDecisionReview handles GET /analytical/composite/decision/review?correlation_id=...&symbol=...
//
// Returns a decision-centric evidence bundle for a single correlation_id.
// The bundle assembles signal inputs, decision transform, strategy resolution,
// risk constraints, and execution output into a single auditable review.
//
// S471: Primary endpoint for decision review and evidence bundling.
func (h *CompositeWebHandler) GetDecisionReview(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getDecisionReview == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "decision review query is unavailable"))
		return
	}

	correlationID := r.URL.Query().Get("correlation_id")
	if correlationID == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "correlation_id query parameter is required"))
		return
	}
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "symbol query parameter is required for isolation (S301)"))
		return
	}

	outcome := r.URL.Query().Get("outcome")

	result, prob := h.getDecisionReview.Execute(r.Context(), analyticalclient.DecisionReviewQuery{
		CorrelationID: correlationID,
		Symbol:        symbol,
		Outcome:       outcome,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("decision review request failed",
			"correlation_id", correlationID, "symbol", symbol, "total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, decisionReviewResponse{
		Reviews: result.Reviews,
		Source:  result.Source,
		Meta:    result.Meta,
	})
}

// GetDecisionReviews handles GET /analytical/composite/decision/reviews?source=...&symbol=...&timeframe=...&outcome=...&since=...&until=...&limit=...
//
// Returns recent decision-centric evidence bundles for a given source/symbol/timeframe.
// Optional outcome filter narrows to triggered/not_triggered/insufficient decisions.
//
// S471: Batch endpoint for decision review and comparison.
func (h *CompositeWebHandler) GetDecisionReviews(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getDecisionReview == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "decision review query is unavailable"))
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

	result, prob := h.getDecisionReview.Execute(r.Context(), analyticalclient.DecisionReviewQuery{
		Source:    key.Source,
		Symbol:    key.Symbol,
		Timeframe: key.Timeframe,
		Outcome:   outcome,
		Since:     params.Since,
		Until:     params.Until,
		Limit:     params.Limit,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("decision reviews request failed",
			"source", key.Source, "symbol", key.Symbol, "timeframe", key.Timeframe,
			"outcome", outcome, "total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, decisionReviewResponse{
		Reviews: result.Reviews,
		Source:  result.Source,
		Meta:    result.Meta,
	})
}

// GetEffectiveness handles GET /analytical/composite/decision/effectiveness?correlation_id=...&symbol=...
//
// Returns effectiveness attribution for a single decision chain.
//
// S476: Single-chain effectiveness lookup (C-SE4).
func (h *CompositeWebHandler) GetEffectiveness(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getEffectiveness == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "effectiveness evaluation is unavailable"))
		return
	}

	correlationID := r.URL.Query().Get("correlation_id")
	if correlationID == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "correlation_id query parameter is required"))
		return
	}
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "symbol query parameter is required for isolation (S301)"))
		return
	}

	result, prob := h.getEffectiveness.Execute(r.Context(), analyticalclient.EffectivenessQuery{
		CorrelationID: correlationID,
		Symbol:        symbol,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("effectiveness request failed",
			"correlation_id", correlationID, "symbol", symbol, "total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, result)
}

// GetEffectivenessBatch handles GET /analytical/composite/decision/effectiveness/batch?source=...&symbol=...&timeframe=...
//
// Returns batch effectiveness evaluations for a cohort of decision chains.
// Supports filtering by decision_type, strategy_type, severity, and effectiveness outcome.
//
// S476: Batch evaluation endpoint (C-SE4, Q-SE4).
func (h *CompositeWebHandler) GetEffectivenessBatch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getEffectiveness == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "effectiveness evaluation is unavailable"))
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

	decisionType := r.URL.Query().Get("decision_type")
	strategyType := r.URL.Query().Get("strategy_type")
	severity := r.URL.Query().Get("severity")
	effectivenessFilter := r.URL.Query().Get("effectiveness")

	result, prob := h.getEffectiveness.Execute(r.Context(), analyticalclient.EffectivenessQuery{
		Source:        key.Source,
		Symbol:        key.Symbol,
		Timeframe:     key.Timeframe,
		DecisionType:  decisionType,
		StrategyType:  strategyType,
		Severity:      severity,
		Effectiveness: effectivenessFilter,
		Since:         params.Since,
		Until:         params.Until,
		Limit:         params.Limit,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("effectiveness batch request failed",
			"source", key.Source, "symbol", key.Symbol, "timeframe", key.Timeframe,
			"total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, result)
}

// GetEffectivenessSummary handles GET /analytical/composite/decision/effectiveness/summary?source=...&symbol=...&timeframe=...
//
// Returns cohort-level aggregation of effectiveness evaluations: win/loss/breakeven
// counts, total and average P&L, win rate. When group_by is set, returns one cohort
// per distinct value of the grouping dimension for side-by-side comparison.
//
// S477: Cohort aggregation and comparative analysis endpoint (Q-SE5).
func (h *CompositeWebHandler) GetEffectivenessSummary(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getEffectivenessSummary == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "effectiveness summary is unavailable"))
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

	groupBy := r.URL.Query().Get("group_by")
	decisionType := r.URL.Query().Get("decision_type")
	strategyType := r.URL.Query().Get("strategy_type")
	severity := r.URL.Query().Get("severity")

	result, prob := h.getEffectivenessSummary.Execute(r.Context(), analyticalclient.EffectivenessSummaryQuery{
		Source:       key.Source,
		Symbol:       key.Symbol,
		Timeframe:    key.Timeframe,
		DecisionType: decisionType,
		StrategyType: strategyType,
		Severity:     severity,
		Since:        params.Since,
		Until:        params.Until,
		Limit:        params.Limit,
		GroupBy:      groupBy,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("effectiveness summary request failed",
			"source", key.Source, "symbol", key.Symbol, "timeframe", key.Timeframe,
			"group_by", groupBy, "total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, result)
}

// GetPairing handles GET /analytical/composite/pairing?source=...&symbol=...&timeframe=...
//
// Returns round-trip pairing results with FIFO matching and effectiveness attribution.
// Paired round-trips include realized P&L; unmatched legs include explicit reason codes.
//
// S481: Round-trip pairing read model endpoint (Q-RT1, Q-RT4).
func (h *CompositeWebHandler) GetPairing(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getPairing == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "pairing read model is unavailable"))
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

	state := r.URL.Query().Get("state")
	side := r.URL.Query().Get("side")

	pairingResult, prob := h.getPairing.Execute(r.Context(), analyticalclient.PairingQuery{
		Source:    key.Source,
		Symbol:    key.Symbol,
		Timeframe: key.Timeframe,
		Since:     params.Since,
		Until:     params.Until,
		Limit:     params.Limit,
		State:     state,
		Side:      side,
	})
	if prob != nil {
		pairingTotalMs := time.Since(start).Milliseconds()
		h.logger.Warn("pairing request failed",
			"source", key.Source, "symbol", key.Symbol, "timeframe", key.Timeframe,
			"total_ms", pairingTotalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	pairingTotalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", pairingTotalMs, pairingResult.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, pairingResult)
}

// GetPairingSingle handles GET /analytical/composite/pairing/chain?correlation_id=...&symbol=...
//
// Returns pairing for a single correlation chain.
//
// S481: Single-chain pairing lookup.
func (h *CompositeWebHandler) GetPairingSingle(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getPairing == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "pairing read model is unavailable"))
		return
	}

	correlationID := r.URL.Query().Get("correlation_id")
	if correlationID == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "correlation_id query parameter is required"))
		return
	}
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "symbol query parameter is required for isolation (S301)"))
		return
	}

	singleResult, prob := h.getPairing.Execute(r.Context(), analyticalclient.PairingQuery{
		CorrelationID: correlationID,
		Symbol:        symbol,
	})
	if prob != nil {
		singleTotalMs := time.Since(start).Milliseconds()
		h.logger.Warn("pairing single request failed",
			"correlation_id", correlationID, "symbol", symbol,
			"total_ms", singleTotalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	singleTotalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", singleTotalMs, singleResult.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, singleResult)
}

// GetRoundTripReview handles GET /analytical/composite/pairing/review?source=...&symbol=...&timeframe=...
//
// Returns round-trip review with reconciliation flags, data-quality signals,
// and effectiveness attribution. Supports outcome and flagged filters.
//
// S482: Round-trip review and outcome reconciliation endpoint.
func (h *CompositeWebHandler) GetRoundTripReview(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getRoundTripReview == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "round-trip review is unavailable"))
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

	state := r.URL.Query().Get("state")
	side := r.URL.Query().Get("side")
	outcome := r.URL.Query().Get("outcome")
	flagged := r.URL.Query().Get("flagged") == "true"

	reviewResult, prob := h.getRoundTripReview.Execute(r.Context(), analyticalclient.RoundTripReviewQuery{
		Source:    key.Source,
		Symbol:    key.Symbol,
		Timeframe: key.Timeframe,
		Since:     params.Since,
		Until:     params.Until,
		Limit:     params.Limit,
		State:     state,
		Side:      side,
		Outcome:   outcome,
		Flagged:   flagged,
	})
	if prob != nil {
		reviewTotalMs := time.Since(start).Milliseconds()
		h.logger.Warn("round-trip review request failed",
			"source", key.Source, "symbol", key.Symbol, "timeframe", key.Timeframe,
			"total_ms", reviewTotalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	reviewTotalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", reviewTotalMs, reviewResult.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, reviewResult)
}

// GetRoundTripReviewSingle handles GET /analytical/composite/pairing/review/chain?correlation_id=...&symbol=...
//
// Returns review for a single correlation chain.
//
// S482: Single-chain round-trip review.
func (h *CompositeWebHandler) GetRoundTripReviewSingle(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getRoundTripReview == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "round-trip review is unavailable"))
		return
	}

	correlationID := r.URL.Query().Get("correlation_id")
	if correlationID == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "correlation_id query parameter is required"))
		return
	}
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "symbol query parameter is required for isolation (S301)"))
		return
	}

	reviewSingleResult, prob := h.getRoundTripReview.Execute(r.Context(), analyticalclient.RoundTripReviewQuery{
		CorrelationID: correlationID,
		Symbol:        symbol,
	})
	if prob != nil {
		reviewSingleTotalMs := time.Since(start).Milliseconds()
		h.logger.Warn("round-trip review single request failed",
			"correlation_id", correlationID, "symbol", symbol,
			"total_ms", reviewSingleTotalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	reviewSingleTotalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", reviewSingleTotalMs, reviewSingleResult.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, reviewSingleResult)
}

// GetCrossSessionPairing handles GET /analytical/composite/pairing/cross-session?source=...&symbol=...&timeframe=...&since=...
//
// Returns cross-session round-trip pairing with continuity classification,
// session provenance, and effectiveness attribution.
//
// S495: Cross-session pairing read model — extends S481/S494 with multi-session
// leg discovery and continuity attribution.
func (h *CompositeWebHandler) GetCrossSessionPairing(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getCrossSessionPairing == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "cross-session pairing read model is unavailable"))
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

	if params.Since == 0 {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "since query parameter is required for cross-session pairing"))
		return
	}

	var maxSessions int
	if msStr := r.URL.Query().Get("max_sessions"); msStr != "" {
		var msErr error
		maxSessions, msErr = strconv.Atoi(msStr)
		if msErr != nil {
			writeProblemResponse(w, problem.New(problem.InvalidArgument, "max_sessions must be a valid integer"))
			return
		}
	}

	continuity := r.URL.Query().Get("continuity")
	crossOnly := r.URL.Query().Get("cross_only") == "true"

	result, prob := h.getCrossSessionPairing.Execute(r.Context(), analyticalclient.CrossSessionPairingQuery{
		Source:      key.Source,
		Symbol:      key.Symbol,
		Timeframe:   key.Timeframe,
		Since:       params.Since,
		Until:       params.Until,
		MaxSessions: maxSessions,
		Continuity:  continuity,
		CrossOnly:   crossOnly,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("cross-session pairing request failed",
			"source", key.Source, "symbol", key.Symbol, "timeframe", key.Timeframe,
			"total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, result)
}

// GetContinuityReview handles GET /analytical/composite/pairing/continuity-review?source=...&symbol=...&timeframe=...&since=...
//
// Returns a unified continuity review combining cross-session pairing,
// reconciliation flags, effectiveness attribution, and boundary carryover analysis.
//
// S496: Continuity review surface — extends S495 (cross-session pairing) with
// S482 (reconciliation) and S476 (effectiveness) into a single operator view.
func (h *CompositeWebHandler) GetContinuityReview(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if h == nil || h.getContinuityReview == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "continuity review is unavailable"))
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

	if params.Since == 0 {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "since query parameter is required for continuity review"))
		return
	}

	var maxSessions int
	if msStr := r.URL.Query().Get("max_sessions"); msStr != "" {
		var msErr error
		maxSessions, msErr = strconv.Atoi(msStr)
		if msErr != nil {
			writeProblemResponse(w, problem.New(problem.InvalidArgument, "max_sessions must be a valid integer"))
			return
		}
	}

	continuity := r.URL.Query().Get("continuity")
	crossOnly := r.URL.Query().Get("cross_only") == "true"
	flagged := r.URL.Query().Get("flagged") == "true"
	outcome := r.URL.Query().Get("outcome")

	result, prob := h.getContinuityReview.Execute(r.Context(), analyticalclient.ContinuityReviewQuery{
		Source:      key.Source,
		Symbol:      key.Symbol,
		Timeframe:   key.Timeframe,
		Since:       params.Since,
		Until:       params.Until,
		MaxSessions: maxSessions,
		Continuity:  continuity,
		CrossOnly:   crossOnly,
		Flagged:     flagged,
		Outcome:     outcome,
	})
	if prob != nil {
		totalMs := time.Since(start).Milliseconds()
		h.logger.Warn("continuity review request failed",
			"source", key.Source, "symbol", key.Symbol, "timeframe", key.Timeframe,
			"total_ms", totalMs, "problem", prob.Code,
		)
		writeProblemResponse(w, prob)
		return
	}

	totalMs := time.Since(start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%d, query;dur=%d", totalMs, result.Meta.TotalMs))

	writeJSONResponse(w, http.StatusOK, result)
}
