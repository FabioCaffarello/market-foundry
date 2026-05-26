package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/execution"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
	"internal/interfaces/http/handlers"
	"internal/shared/problem"
)

type mockAnalyticalCandleHistory struct {
	reply     analyticalclient.CandleHistoryReply
	prob      *problem.Problem
	lastQuery analyticalclient.CandleHistoryQuery
}

func (m *mockAnalyticalCandleHistory) Execute(_ context.Context, q analyticalclient.CandleHistoryQuery) (analyticalclient.CandleHistoryReply, *problem.Problem) {
	m.lastQuery = q
	return m.reply, m.prob
}

func TestAnalyticalWebHandler_GetCandleHistory(t *testing.T) {
	now := time.Now().UTC().Truncate(60 * time.Second)
	candles := []evidence.EvidenceCandle{
		{
			Source:     "binancef",
			Instrument: btcUSDTPerp(t),
			Timeframe:  60,
			Open:       "100.00",
			High:       "105.00",
			Low:        "99.00",
			Close:      "102.00",
			Volume:     "1000.00",
			TradeCount: 42,
			OpenTime:   now,
			CloseTime:  now.Add(60 * time.Second),
			Final:      true,
		},
	}

	mock := &mockAnalyticalCandleHistory{
		reply: analyticalclient.CandleHistoryReply{
			Candles: candles,
			Source:  "clickhouse",
			Meta:    analyticalclient.QueryMeta{QueryMs: 5, RowCount: 1},
		},
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetCandleHistory: mock})

	req := httptest.NewRequest(http.MethodGet, "/analytical/evidence/candles?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Verify Server-Timing header is present.
	if st := rec.Header().Get("Server-Timing"); st == "" {
		t.Error("expected Server-Timing header")
	}

	var resp struct {
		Candles []evidence.EvidenceCandle  `json:"candles"`
		Source  string                     `json:"source"`
		Meta    analyticalclient.QueryMeta `json:"meta"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", resp.Source)
	}
	if len(resp.Candles) != 1 {
		t.Errorf("expected 1 candle, got %d", len(resp.Candles))
	}
	if resp.Meta.RowCount != 1 {
		t.Errorf("expected meta.row_count=1, got %d", resp.Meta.RowCount)
	}
	if resp.Meta.QueryMs != 5 {
		t.Errorf("expected meta.query_ms=5, got %d", resp.Meta.QueryMs)
	}
}

func TestAnalyticalWebHandler_MissingTimeframe(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetCandleHistory: &mockAnalyticalCandleHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/evidence/candles?source=binancef&symbol=btcusdt", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_InvalidLimit(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetCandleHistory: &mockAnalyticalCandleHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/evidence/candles?source=binancef&symbol=btcusdt&timeframe=60&limit=9999", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_NilHandler(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{})
	req := httptest.NewRequest(http.MethodGet, "/analytical/evidence/candles?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_UseCaseError(t *testing.T) {
	mock := &mockAnalyticalCandleHistory{
		prob: problem.New(problem.Unavailable, "clickhouse down"),
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetCandleHistory: mock})
	req := httptest.NewRequest(http.MethodGet, "/analytical/evidence/candles?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

// ── Signal History Tests ────────────────────────────────────────

type mockAnalyticalSignalHistory struct {
	reply     analyticalclient.SignalHistoryReply
	prob      *problem.Problem
	lastQuery analyticalclient.SignalHistoryQuery
}

func (m *mockAnalyticalSignalHistory) Execute(_ context.Context, q analyticalclient.SignalHistoryQuery) (analyticalclient.SignalHistoryReply, *problem.Problem) {
	m.lastQuery = q
	return m.reply, m.prob
}

func TestAnalyticalWebHandler_GetSignalHistory(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	signals := []signal.Signal{
		{
			Type:       "rsi",
			Source:     "binancef",
			Instrument: btcUSDTPerp(t),
			Timeframe:  60,
			Value:      "32.5",
			Metadata:   map[string]string{"period": "14"},
			Final:      true,
			Timestamp:  now,
		},
	}

	mock := &mockAnalyticalSignalHistory{
		reply: analyticalclient.SignalHistoryReply{
			Signals: signals,
			Source:  "clickhouse",
			Meta:    analyticalclient.QueryMeta{QueryMs: 3, RowCount: 1},
		},
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetSignalHistory: mock})

	req := httptest.NewRequest(http.MethodGet, "/analytical/signal/history?type=rsi&source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetSignalHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if st := rec.Header().Get("Server-Timing"); st == "" {
		t.Error("expected Server-Timing header")
	}

	var resp struct {
		Signals []signal.Signal            `json:"signals"`
		Source  string                     `json:"source"`
		Meta    analyticalclient.QueryMeta `json:"meta"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", resp.Source)
	}
	if len(resp.Signals) != 1 {
		t.Errorf("expected 1 signal, got %d", len(resp.Signals))
	}
	if resp.Meta.RowCount != 1 {
		t.Errorf("expected meta.row_count=1, got %d", resp.Meta.RowCount)
	}
}

func TestAnalyticalWebHandler_SignalHistory_MissingType(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetSignalHistory: &mockAnalyticalSignalHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/signal/history?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetSignalHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_SignalHistory_MissingTimeframe(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetSignalHistory: &mockAnalyticalSignalHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/signal/history?type=rsi&source=binancef&symbol=btcusdt", nil)
	rec := httptest.NewRecorder()
	handler.GetSignalHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_SignalHistory_InvalidLimit(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetSignalHistory: &mockAnalyticalSignalHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/signal/history?type=rsi&source=binancef&symbol=btcusdt&timeframe=60&limit=9999", nil)
	rec := httptest.NewRecorder()
	handler.GetSignalHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_SignalHistory_NilHandler(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{})
	req := httptest.NewRequest(http.MethodGet, "/analytical/signal/history?type=rsi&source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetSignalHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_SignalHistory_UseCaseError(t *testing.T) {
	mock := &mockAnalyticalSignalHistory{
		prob: problem.New(problem.Unavailable, "clickhouse down"),
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetSignalHistory: mock})
	req := httptest.NewRequest(http.MethodGet, "/analytical/signal/history?type=rsi&source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetSignalHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

// -- Decision History Tests ---------------------------------------------------

type mockAnalyticalDecisionHistory struct {
	reply     analyticalclient.DecisionHistoryReply
	prob      *problem.Problem
	lastQuery analyticalclient.DecisionHistoryQuery
}

func (m *mockAnalyticalDecisionHistory) Execute(_ context.Context, q analyticalclient.DecisionHistoryQuery) (analyticalclient.DecisionHistoryReply, *problem.Problem) {
	m.lastQuery = q
	return m.reply, m.prob
}

func TestAnalyticalWebHandler_GetDecisionHistory(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	decisions := []decision.Decision{
		{
			Type:       "rsi_oversold",
			Source:     "binancef",
			Instrument: btcUSDTPerp(t),
			Timeframe:  60,
			Outcome:    decision.OutcomeTriggered,
			Severity:   decision.SeverityLow,
			Confidence: "0.85",
			Rationale:  "RSI 28.5 below oversold threshold 30.0 (distance 5.0%); severity low",
			Signals:    []decision.SignalInput{{Type: "rsi", Value: "28.5", Timeframe: 60}},
			Metadata:   map[string]string{"threshold": "30"},
			Final:      true,
			Timestamp:  now,
		},
	}

	mock := &mockAnalyticalDecisionHistory{
		reply: analyticalclient.DecisionHistoryReply{
			Decisions: decisions,
			Source:    "clickhouse",
			Meta:      analyticalclient.QueryMeta{QueryMs: 4, RowCount: 1},
		},
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetDecisionHistory: mock})

	req := httptest.NewRequest(http.MethodGet, "/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetDecisionHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if st := rec.Header().Get("Server-Timing"); st == "" {
		t.Error("expected Server-Timing header")
	}

	var resp struct {
		Decisions []decision.Decision        `json:"decisions"`
		Source    string                     `json:"source"`
		Meta      analyticalclient.QueryMeta `json:"meta"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", resp.Source)
	}
	if len(resp.Decisions) != 1 {
		t.Errorf("expected 1 decision, got %d", len(resp.Decisions))
	}
	if resp.Meta.RowCount != 1 {
		t.Errorf("expected meta.row_count=1, got %d", resp.Meta.RowCount)
	}
}

func TestAnalyticalWebHandler_DecisionHistory_MissingType(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetDecisionHistory: &mockAnalyticalDecisionHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/decision/history?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetDecisionHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_DecisionHistory_MissingTimeframe(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetDecisionHistory: &mockAnalyticalDecisionHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt", nil)
	rec := httptest.NewRecorder()
	handler.GetDecisionHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_DecisionHistory_InvalidLimit(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetDecisionHistory: &mockAnalyticalDecisionHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60&limit=9999", nil)
	rec := httptest.NewRecorder()
	handler.GetDecisionHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_DecisionHistory_WithOutcome(t *testing.T) {
	mock := &mockAnalyticalDecisionHistory{
		reply: analyticalclient.DecisionHistoryReply{
			Decisions: []decision.Decision{},
			Source:    "clickhouse",
			Meta:      analyticalclient.QueryMeta{QueryMs: 1, RowCount: 0},
		},
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetDecisionHistory: mock})

	req := httptest.NewRequest(http.MethodGet, "/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60&outcome=triggered", nil)
	rec := httptest.NewRecorder()
	handler.GetDecisionHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if mock.lastQuery.Outcome != "triggered" {
		t.Errorf("expected outcome=triggered, got %q", mock.lastQuery.Outcome)
	}
}

func TestAnalyticalWebHandler_DecisionHistory_NilHandler(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{})
	req := httptest.NewRequest(http.MethodGet, "/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetDecisionHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_DecisionHistory_UseCaseError(t *testing.T) {
	mock := &mockAnalyticalDecisionHistory{
		prob: problem.New(problem.Unavailable, "clickhouse down"),
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetDecisionHistory: mock})
	req := httptest.NewRequest(http.MethodGet, "/analytical/decision/history?type=rsi_oversold&source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetDecisionHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

// -- Strategy History Tests ---------------------------------------------------

type mockAnalyticalStrategyHistory struct {
	reply     analyticalclient.StrategyHistoryReply
	prob      *problem.Problem
	lastQuery analyticalclient.StrategyHistoryQuery
}

func (m *mockAnalyticalStrategyHistory) Execute(_ context.Context, q analyticalclient.StrategyHistoryQuery) (analyticalclient.StrategyHistoryReply, *problem.Problem) {
	m.lastQuery = q
	return m.reply, m.prob
}

func TestAnalyticalWebHandler_GetStrategyHistory(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	strategies := []strategy.Strategy{
		{
			Type:       "mean_reversion_entry",
			Source:     "binancef",
			Symbol:     "btcusdt",
			Timeframe:  60,
			Direction:  strategy.DirectionLong,
			Confidence: "0.85",
			Decisions:  []strategy.DecisionInput{{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Timeframe: 60}},
			Parameters: map[string]string{"entry_threshold": "30"},
			Metadata:   map[string]string{"version": "1"},
			Final:      true,
			Timestamp:  now,
		},
	}

	mock := &mockAnalyticalStrategyHistory{
		reply: analyticalclient.StrategyHistoryReply{
			Strategies: strategies,
			Source:     "clickhouse",
			Meta:       analyticalclient.QueryMeta{QueryMs: 6, RowCount: 1},
		},
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetStrategyHistory: mock})

	req := httptest.NewRequest(http.MethodGet, "/analytical/strategy/history?type=mean_reversion_entry&source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetStrategyHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if st := rec.Header().Get("Server-Timing"); st == "" {
		t.Error("expected Server-Timing header")
	}

	var resp struct {
		Strategies []strategy.Strategy        `json:"strategies"`
		Source     string                     `json:"source"`
		Meta       analyticalclient.QueryMeta `json:"meta"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", resp.Source)
	}
	if len(resp.Strategies) != 1 {
		t.Errorf("expected 1 strategy, got %d", len(resp.Strategies))
	}
	if resp.Meta.RowCount != 1 {
		t.Errorf("expected meta.row_count=1, got %d", resp.Meta.RowCount)
	}
}

func TestAnalyticalWebHandler_StrategyHistory_MissingType(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetStrategyHistory: &mockAnalyticalStrategyHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/strategy/history?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetStrategyHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_StrategyHistory_MissingTimeframe(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetStrategyHistory: &mockAnalyticalStrategyHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/strategy/history?type=mean_reversion_entry&source=binancef&symbol=btcusdt", nil)
	rec := httptest.NewRecorder()
	handler.GetStrategyHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_StrategyHistory_InvalidLimit(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetStrategyHistory: &mockAnalyticalStrategyHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/strategy/history?type=mean_reversion_entry&source=binancef&symbol=btcusdt&timeframe=60&limit=9999", nil)
	rec := httptest.NewRecorder()
	handler.GetStrategyHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_StrategyHistory_WithDirection(t *testing.T) {
	mock := &mockAnalyticalStrategyHistory{
		reply: analyticalclient.StrategyHistoryReply{
			Strategies: []strategy.Strategy{},
			Source:     "clickhouse",
			Meta:       analyticalclient.QueryMeta{QueryMs: 1, RowCount: 0},
		},
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetStrategyHistory: mock})

	req := httptest.NewRequest(http.MethodGet, "/analytical/strategy/history?type=mean_reversion_entry&source=binancef&symbol=btcusdt&timeframe=60&direction=long", nil)
	rec := httptest.NewRecorder()
	handler.GetStrategyHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if mock.lastQuery.Direction != "long" {
		t.Errorf("expected direction=long, got %q", mock.lastQuery.Direction)
	}
}

func TestAnalyticalWebHandler_StrategyHistory_NilHandler(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{})
	req := httptest.NewRequest(http.MethodGet, "/analytical/strategy/history?type=mean_reversion_entry&source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetStrategyHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_StrategyHistory_UseCaseError(t *testing.T) {
	mock := &mockAnalyticalStrategyHistory{
		prob: problem.New(problem.Unavailable, "clickhouse down"),
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetStrategyHistory: mock})
	req := httptest.NewRequest(http.MethodGet, "/analytical/strategy/history?type=mean_reversion_entry&source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetStrategyHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

// -- Risk History Tests -------------------------------------------------------

type mockAnalyticalRiskHistory struct {
	reply     analyticalclient.RiskHistoryReply
	prob      *problem.Problem
	lastQuery analyticalclient.RiskHistoryQuery
}

func (m *mockAnalyticalRiskHistory) Execute(_ context.Context, q analyticalclient.RiskHistoryQuery) (analyticalclient.RiskHistoryReply, *problem.Problem) {
	m.lastQuery = q
	return m.reply, m.prob
}

func TestAnalyticalWebHandler_GetRiskHistory(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	assessments := []risk.RiskAssessment{
		{
			Type:        "position_exposure",
			Source:      "binancef",
			Symbol:      "btcusdt",
			Timeframe:   60,
			Disposition: risk.DispositionApproved,
			Confidence:  "0.82",
			Strategies:  []risk.StrategyInput{{Type: "mean_reversion_entry", Direction: "long", Confidence: "0.85", Timeframe: 60}},
			Constraints: risk.Constraints{MaxPositionSize: "0.1", MaxExposure: "1000.00"},
			Rationale:   "Position within exposure limits",
			Parameters:  map[string]string{"risk_model": "basic"},
			Metadata:    map[string]string{},
			Final:       true,
			Timestamp:   now,
		},
	}

	mock := &mockAnalyticalRiskHistory{
		reply: analyticalclient.RiskHistoryReply{
			RiskAssessments: assessments,
			Source:          "clickhouse",
			Meta:            analyticalclient.QueryMeta{QueryMs: 8, RowCount: 1},
		},
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetRiskHistory: mock})

	req := httptest.NewRequest(http.MethodGet, "/analytical/risk/history?type=position_exposure&source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetRiskHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if st := rec.Header().Get("Server-Timing"); st == "" {
		t.Error("expected Server-Timing header")
	}

	var resp struct {
		RiskAssessments []risk.RiskAssessment      `json:"risk_assessments"`
		Source          string                     `json:"source"`
		Meta            analyticalclient.QueryMeta `json:"meta"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", resp.Source)
	}
	if len(resp.RiskAssessments) != 1 {
		t.Errorf("expected 1 risk assessment, got %d", len(resp.RiskAssessments))
	}
	if resp.Meta.RowCount != 1 {
		t.Errorf("expected meta.row_count=1, got %d", resp.Meta.RowCount)
	}
}

func TestAnalyticalWebHandler_RiskHistory_MissingType(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetRiskHistory: &mockAnalyticalRiskHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/risk/history?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetRiskHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_RiskHistory_MissingTimeframe(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetRiskHistory: &mockAnalyticalRiskHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/risk/history?type=position_exposure&source=binancef&symbol=btcusdt", nil)
	rec := httptest.NewRecorder()
	handler.GetRiskHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_RiskHistory_InvalidLimit(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetRiskHistory: &mockAnalyticalRiskHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/risk/history?type=position_exposure&source=binancef&symbol=btcusdt&timeframe=60&limit=9999", nil)
	rec := httptest.NewRecorder()
	handler.GetRiskHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_RiskHistory_WithDisposition(t *testing.T) {
	mock := &mockAnalyticalRiskHistory{
		reply: analyticalclient.RiskHistoryReply{
			RiskAssessments: []risk.RiskAssessment{},
			Source:          "clickhouse",
			Meta:            analyticalclient.QueryMeta{QueryMs: 1, RowCount: 0},
		},
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetRiskHistory: mock})

	req := httptest.NewRequest(http.MethodGet, "/analytical/risk/history?type=position_exposure&source=binancef&symbol=btcusdt&timeframe=60&disposition=approved", nil)
	rec := httptest.NewRecorder()
	handler.GetRiskHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if mock.lastQuery.Disposition != "approved" {
		t.Errorf("expected disposition=approved, got %q", mock.lastQuery.Disposition)
	}
}

func TestAnalyticalWebHandler_RiskHistory_NilHandler(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{})
	req := httptest.NewRequest(http.MethodGet, "/analytical/risk/history?type=position_exposure&source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetRiskHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_RiskHistory_UseCaseError(t *testing.T) {
	mock := &mockAnalyticalRiskHistory{
		prob: problem.New(problem.Unavailable, "clickhouse down"),
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetRiskHistory: mock})
	req := httptest.NewRequest(http.MethodGet, "/analytical/risk/history?type=position_exposure&source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetRiskHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

// -- Execution History Tests --------------------------------------------------

type mockAnalyticalExecutionHistory struct {
	reply     analyticalclient.ExecutionHistoryReply
	prob      *problem.Problem
	lastQuery analyticalclient.ExecutionHistoryQuery
}

func (m *mockAnalyticalExecutionHistory) Execute(_ context.Context, q analyticalclient.ExecutionHistoryQuery) (analyticalclient.ExecutionHistoryReply, *problem.Problem) {
	m.lastQuery = q
	return m.reply, m.prob
}

func TestAnalyticalWebHandler_GetExecutionHistory(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	executions := []execution.ExecutionIntent{
		{
			Type:           "paper_order",
			Source:         "derive",
			Symbol:         "btcusdt",
			Timeframe:      60,
			Side:           execution.SideBuy,
			Quantity:       "0.001",
			FilledQuantity: "0.001",
			Status:         execution.StatusFilled,
			Risk:           execution.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
			Fills:          []execution.FillRecord{{Price: "67500.00", Quantity: "0.001", Fee: "0.00", Simulated: true, Timestamp: now}},
			Parameters:     map[string]string{"strategy": "mean_reversion_entry"},
			Metadata:       map[string]string{"version": "1"},
			CorrelationID:  "corr-123",
			CausationID:    "cause-456",
			Final:          true,
			Timestamp:      now,
		},
	}

	mock := &mockAnalyticalExecutionHistory{
		reply: analyticalclient.ExecutionHistoryReply{
			Executions: executions,
			Source:     "clickhouse",
			Meta:       analyticalclient.QueryMeta{QueryMs: 15, RowCount: 1},
		},
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetExecutionHistory: mock})

	req := httptest.NewRequest(http.MethodGet, "/analytical/execution/history?type=paper_order&source=derive&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetExecutionHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if st := rec.Header().Get("Server-Timing"); st == "" {
		t.Error("expected Server-Timing header")
	}

	var resp struct {
		Executions []execution.ExecutionIntent `json:"executions"`
		Source     string                      `json:"source"`
		Meta       analyticalclient.QueryMeta  `json:"meta"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Source != "clickhouse" {
		t.Errorf("expected source=clickhouse, got %q", resp.Source)
	}
	if len(resp.Executions) != 1 {
		t.Errorf("expected 1 execution, got %d", len(resp.Executions))
	}
	if resp.Meta.RowCount != 1 {
		t.Errorf("expected meta.row_count=1, got %d", resp.Meta.RowCount)
	}
}

func TestAnalyticalWebHandler_ExecutionHistory_MissingType(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetExecutionHistory: &mockAnalyticalExecutionHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/execution/history?source=derive&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetExecutionHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_ExecutionHistory_MissingTimeframe(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetExecutionHistory: &mockAnalyticalExecutionHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/execution/history?type=paper_order&source=derive&symbol=btcusdt", nil)
	rec := httptest.NewRecorder()
	handler.GetExecutionHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_ExecutionHistory_InvalidLimit(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetExecutionHistory: &mockAnalyticalExecutionHistory{}})
	req := httptest.NewRequest(http.MethodGet, "/analytical/execution/history?type=paper_order&source=derive&symbol=btcusdt&timeframe=60&limit=9999", nil)
	rec := httptest.NewRecorder()
	handler.GetExecutionHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_ExecutionHistory_WithSide(t *testing.T) {
	mock := &mockAnalyticalExecutionHistory{
		reply: analyticalclient.ExecutionHistoryReply{
			Executions: []execution.ExecutionIntent{},
			Source:     "clickhouse",
			Meta:       analyticalclient.QueryMeta{QueryMs: 1, RowCount: 0},
		},
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetExecutionHistory: mock})

	req := httptest.NewRequest(http.MethodGet, "/analytical/execution/history?type=paper_order&source=derive&symbol=btcusdt&timeframe=60&side=buy", nil)
	rec := httptest.NewRecorder()
	handler.GetExecutionHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if mock.lastQuery.Side != "buy" {
		t.Errorf("expected side=buy, got %q", mock.lastQuery.Side)
	}
}

func TestAnalyticalWebHandler_ExecutionHistory_WithStatus(t *testing.T) {
	mock := &mockAnalyticalExecutionHistory{
		reply: analyticalclient.ExecutionHistoryReply{
			Executions: []execution.ExecutionIntent{},
			Source:     "clickhouse",
			Meta:       analyticalclient.QueryMeta{QueryMs: 1, RowCount: 0},
		},
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetExecutionHistory: mock})

	req := httptest.NewRequest(http.MethodGet, "/analytical/execution/history?type=paper_order&source=derive&symbol=btcusdt&timeframe=60&status=filled", nil)
	rec := httptest.NewRecorder()
	handler.GetExecutionHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if mock.lastQuery.Status != "filled" {
		t.Errorf("expected status=filled, got %q", mock.lastQuery.Status)
	}
}

func TestAnalyticalWebHandler_ExecutionHistory_WithBothFilters(t *testing.T) {
	mock := &mockAnalyticalExecutionHistory{
		reply: analyticalclient.ExecutionHistoryReply{
			Executions: []execution.ExecutionIntent{},
			Source:     "clickhouse",
			Meta:       analyticalclient.QueryMeta{QueryMs: 1, RowCount: 0},
		},
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetExecutionHistory: mock})

	req := httptest.NewRequest(http.MethodGet, "/analytical/execution/history?type=paper_order&source=derive&symbol=btcusdt&timeframe=60&side=sell&status=rejected", nil)
	rec := httptest.NewRecorder()
	handler.GetExecutionHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if mock.lastQuery.Side != "sell" {
		t.Errorf("expected side=sell, got %q", mock.lastQuery.Side)
	}
	if mock.lastQuery.Status != "rejected" {
		t.Errorf("expected status=rejected, got %q", mock.lastQuery.Status)
	}
}

func TestAnalyticalWebHandler_ExecutionHistory_NilHandler(t *testing.T) {
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{})
	req := httptest.NewRequest(http.MethodGet, "/analytical/execution/history?type=paper_order&source=derive&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetExecutionHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestAnalyticalWebHandler_ExecutionHistory_UseCaseError(t *testing.T) {
	mock := &mockAnalyticalExecutionHistory{
		prob: problem.New(problem.Unavailable, "clickhouse down"),
	}
	handler := handlers.NewAnalyticalWebHandler(handlers.AnalyticalHandlerDeps{GetExecutionHistory: mock})
	req := httptest.NewRequest(http.MethodGet, "/analytical/execution/history?type=paper_order&source=derive&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetExecutionHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
