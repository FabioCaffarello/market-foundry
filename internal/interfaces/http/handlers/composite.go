package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
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

// CompositeWebHandler handles HTTP requests for composite execution chain queries
// and aggregation endpoints. These expose the composite read model (S296) and
// aggregation layer (S298) as an HTTP query surface for Q1–Q7.
type CompositeWebHandler struct {
	getCompositeChain      getCompositeChainUseCase
	getPipelineFunnel      getPipelineFunnelUseCase
	getDispositionBreakdown getDispositionBreakdownUseCase
	logger                 *slog.Logger
}

// CompositeHandlerDeps groups all dependencies for the composite HTTP handler.
type CompositeHandlerDeps struct {
	GetCompositeChain      getCompositeChainUseCase
	GetPipelineFunnel      getPipelineFunnelUseCase
	GetDispositionBreakdown getDispositionBreakdownUseCase
	Logger                 *slog.Logger
}

func NewCompositeWebHandler(deps CompositeHandlerDeps) *CompositeWebHandler {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &CompositeWebHandler{
		getCompositeChain:      deps.GetCompositeChain,
		getPipelineFunnel:      deps.GetPipelineFunnel,
		getDispositionBreakdown: deps.GetDispositionBreakdown,
		logger:                 logger.With("component", "composite_handler"),
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
