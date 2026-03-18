package handlers

import (
	"context"
	"net/http"
	"strconv"

	"internal/application/evidenceclient"
	"internal/domain/evidence"
	"internal/shared/problem"
	"internal/shared/requestctx"
)

type getLatestCandleUseCase interface {
	Execute(context.Context, evidenceclient.CandleLatestQuery) (evidenceclient.CandleLatestReply, *problem.Problem)
}

type getCandleHistoryUseCase interface {
	Execute(context.Context, evidenceclient.CandleHistoryQuery) (evidenceclient.CandleHistoryReply, *problem.Problem)
}

type getLatestTradeBurstUseCase interface {
	Execute(context.Context, evidenceclient.TradeBurstLatestQuery) (evidenceclient.TradeBurstLatestReply, *problem.Problem)
}

type getLatestVolumeUseCase interface {
	Execute(context.Context, evidenceclient.VolumeLatestQuery) (evidenceclient.VolumeLatestReply, *problem.Problem)
}

// EvidenceWebHandler handles HTTP requests for evidence queries.
type EvidenceWebHandler struct {
	getLatestCandle     getLatestCandleUseCase
	getCandleHistory    getCandleHistoryUseCase
	getLatestTradeBurst getLatestTradeBurstUseCase
	getLatestVolume     getLatestVolumeUseCase
}

func NewEvidenceWebHandler(getLatestCandle getLatestCandleUseCase, getCandleHistory getCandleHistoryUseCase, getLatestTradeBurst getLatestTradeBurstUseCase, getLatestVolume getLatestVolumeUseCase) *EvidenceWebHandler {
	return &EvidenceWebHandler{
		getLatestCandle:     getLatestCandle,
		getCandleHistory:    getCandleHistory,
		getLatestTradeBurst: getLatestTradeBurst,
		getLatestVolume:     getLatestVolume,
	}
}

// evidenceKeyParams holds the common query parameters shared by all evidence queries.
type evidenceKeyParams struct {
	Source    string
	Symbol    string
	Timeframe int
}

// parseEvidenceKeyParams extracts source, symbol, timeframe from query string.
// Returns a problem if timeframe is missing or invalid.
func parseEvidenceKeyParams(r *http.Request) (evidenceKeyParams, *problem.Problem) {
	source := r.URL.Query().Get("source")
	symbol := r.URL.Query().Get("symbol")
	timeframeStr := r.URL.Query().Get("timeframe")

	if timeframeStr == "" {
		return evidenceKeyParams{}, problem.New(problem.InvalidArgument, "timeframe query parameter is required")
	}
	timeframe, err := strconv.Atoi(timeframeStr)
	if err != nil {
		return evidenceKeyParams{}, problem.New(problem.InvalidArgument, "timeframe must be a valid integer")
	}

	return evidenceKeyParams{Source: source, Symbol: symbol, Timeframe: timeframe}, nil
}

type latestCandleResponse struct {
	Candle *evidence.EvidenceCandle `json:"candle"`
}

// GetLatestCandle handles GET /evidence/candles/latest?source=...&symbol=...&timeframe=...
func (h *EvidenceWebHandler) GetLatestCandle(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getLatestCandle == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "evidence query is unavailable"))
		return
	}

	key, prob := parseEvidenceKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	ctx := requestctx.WithCorrelationID(r.Context(), r.Header.Get("X-Correlation-ID"))
	result, prob := h.getLatestCandle.Execute(ctx, evidenceclient.CandleLatestQuery{
		Source:    key.Source,
		Symbol:    key.Symbol,
		Timeframe: key.Timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, latestCandleResponse{Candle: result.Candle})
}

type candleHistoryResponse struct {
	Candles []evidence.EvidenceCandle `json:"candles"`
}

// GetCandleHistory handles GET /evidence/candles/history?source=...&symbol=...&timeframe=...&limit=...
func (h *EvidenceWebHandler) GetCandleHistory(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getCandleHistory == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "evidence history query is unavailable"))
		return
	}

	key, prob := parseEvidenceKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil {
			writeProblemResponse(w, problem.New(problem.InvalidArgument, "limit must be a valid integer"))
			return
		}
		if parsed < 1 || parsed > 100 {
			writeProblemResponse(w, problem.New(problem.InvalidArgument, "limit must be between 1 and 100"))
			return
		}
		limit = parsed
	}

	var since, until int64
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		parsed, err := strconv.ParseInt(sinceStr, 10, 64)
		if err != nil {
			writeProblemResponse(w, problem.New(problem.InvalidArgument, "since must be a valid unix timestamp"))
			return
		}
		since = parsed
	}
	if untilStr := r.URL.Query().Get("until"); untilStr != "" {
		parsed, err := strconv.ParseInt(untilStr, 10, 64)
		if err != nil {
			writeProblemResponse(w, problem.New(problem.InvalidArgument, "until must be a valid unix timestamp"))
			return
		}
		until = parsed
	}

	ctx := requestctx.WithCorrelationID(r.Context(), r.Header.Get("X-Correlation-ID"))
	result, prob := h.getCandleHistory.Execute(ctx, evidenceclient.CandleHistoryQuery{
		Source:    key.Source,
		Symbol:    key.Symbol,
		Timeframe: key.Timeframe,
		Limit:     limit,
		Since:     since,
		Until:     until,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	candles := result.Candles
	if candles == nil {
		candles = []evidence.EvidenceCandle{}
	}

	writeJSONResponse(w, http.StatusOK, candleHistoryResponse{Candles: candles})
}

type latestTradeBurstResponse struct {
	TradeBurst *evidence.EvidenceTradeBurst `json:"trade_burst"`
}

// GetLatestTradeBurst handles GET /evidence/tradeburst/latest?source=...&symbol=...&timeframe=...
func (h *EvidenceWebHandler) GetLatestTradeBurst(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getLatestTradeBurst == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "trade burst query is unavailable"))
		return
	}

	key, prob := parseEvidenceKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	ctx := requestctx.WithCorrelationID(r.Context(), r.Header.Get("X-Correlation-ID"))
	result, prob := h.getLatestTradeBurst.Execute(ctx, evidenceclient.TradeBurstLatestQuery{
		Source:    key.Source,
		Symbol:    key.Symbol,
		Timeframe: key.Timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, latestTradeBurstResponse{TradeBurst: result.TradeBurst})
}

type latestVolumeResponse struct {
	Volume *evidence.EvidenceVolume `json:"volume"`
}

// GetLatestVolume handles GET /evidence/volume/latest?source=...&symbol=...&timeframe=...
func (h *EvidenceWebHandler) GetLatestVolume(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getLatestVolume == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "volume query is unavailable"))
		return
	}

	key, prob := parseEvidenceKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	ctx := requestctx.WithCorrelationID(r.Context(), r.Header.Get("X-Correlation-ID"))
	result, prob := h.getLatestVolume.Execute(ctx, evidenceclient.VolumeLatestQuery{
		Source:    key.Source,
		Symbol:    key.Symbol,
		Timeframe: key.Timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, latestVolumeResponse{Volume: result.Volume})
}
