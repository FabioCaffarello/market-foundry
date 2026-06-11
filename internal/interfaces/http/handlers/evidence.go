package handlers

import (
	"internal/domain/instrument"

	"context"
	"net/http"
	"strconv"

	"internal/application/evidenceclient"
	"internal/domain/evidence"
	"internal/shared/problem"
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

// queryKeyParams holds the common query parameters shared by all handler families
// (evidence, signal, decision, strategy, risk, execution, analytical).
// Since H-6.e.2 the instrument arrives as the canonical trio
// (base, quote, contract) — the venue-native `symbol` parameter was
// retired (zero external consumers; ADR-0021 criterion #2 erratum).
type queryKeyParams struct {
	Source     string
	Instrument instrument.CanonicalInstrument
	Timeframe  int
}

// parseQueryKeyParams extracts source, the canonical instrument trio,
// and timeframe from the query string. All are required — returns a
// descriptive problem for each missing field.
func parseQueryKeyParams(r *http.Request) (queryKeyParams, *problem.Problem) {
	source := r.URL.Query().Get("source")
	if source == "" {
		return queryKeyParams{}, problem.New(problem.InvalidArgument, "source query parameter is required")
	}
	inst, prob := parseRequiredInstrumentParams(r)
	if prob != nil {
		return queryKeyParams{}, prob
	}
	timeframeStr := r.URL.Query().Get("timeframe")
	if timeframeStr == "" {
		return queryKeyParams{}, problem.New(problem.InvalidArgument, "timeframe query parameter is required")
	}
	timeframe, err := strconv.Atoi(timeframeStr)
	if err != nil {
		return queryKeyParams{}, problem.New(problem.InvalidArgument, "timeframe must be a valid integer")
	}

	return queryKeyParams{Source: source, Instrument: inst, Timeframe: timeframe}, nil
}

// parseRequiredInstrumentParams parses the mandatory canonical trio.
func parseRequiredInstrumentParams(r *http.Request) (instrument.CanonicalInstrument, *problem.Problem) {
	base := r.URL.Query().Get("base")
	quote := r.URL.Query().Get("quote")
	contract := r.URL.Query().Get("contract")
	if base == "" || quote == "" || contract == "" {
		return instrument.CanonicalInstrument{}, problem.New(problem.InvalidArgument,
			"base, quote and contract query parameters are required (canonical instrument; the legacy symbol parameter was retired in H-6.e.2)")
	}
	return instrument.New(base, quote, instrument.ContractType(contract))
}

// parseOptionalInstrumentParams parses the trio when it is an optional
// filter: all-absent means "no instrument filter"; a partial trio is
// an error (all-or-none).
func parseOptionalInstrumentParams(r *http.Request) (instrument.CanonicalInstrument, *problem.Problem) {
	base := r.URL.Query().Get("base")
	quote := r.URL.Query().Get("quote")
	contract := r.URL.Query().Get("contract")
	if base == "" && quote == "" && contract == "" {
		return instrument.CanonicalInstrument{}, nil
	}
	if base == "" || quote == "" || contract == "" {
		return instrument.CanonicalInstrument{}, problem.New(problem.InvalidArgument,
			"base, quote and contract must be provided together (all-or-none canonical instrument filter)")
	}
	return instrument.New(base, quote, instrument.ContractType(contract))
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

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getLatestCandle.Execute(r.Context(), evidenceclient.CandleLatestQuery{
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
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

	key, prob := parseQueryKeyParams(r)
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

	result, prob := h.getCandleHistory.Execute(r.Context(), evidenceclient.CandleHistoryQuery{
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
		Limit:      limit,
		Since:      since,
		Until:      until,
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

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getLatestTradeBurst.Execute(r.Context(), evidenceclient.TradeBurstLatestQuery{
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
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

	key, prob := parseQueryKeyParams(r)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	result, prob := h.getLatestVolume.Execute(r.Context(), evidenceclient.VolumeLatestQuery{
		Source:     key.Source,
		Instrument: key.Instrument,
		Timeframe:  key.Timeframe,
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, latestVolumeResponse{Volume: result.Volume})
}
