package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/evidenceclient"
	"internal/domain/evidence"
	"internal/domain/instrument"
	"internal/interfaces/http/handlers"
	"internal/shared/problem"
)

func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("test setup: failed to build canonical BTC/USDT-perpetual: %v", prob)
	}
	return inst
}

type mockGetLatestCandle struct {
	reply evidenceclient.CandleLatestReply
	prob  *problem.Problem
}

func (m *mockGetLatestCandle) Execute(_ context.Context, _ evidenceclient.CandleLatestQuery) (evidenceclient.CandleLatestReply, *problem.Problem) {
	return m.reply, m.prob
}

type mockGetCandleHistory struct {
	reply     evidenceclient.CandleHistoryReply
	prob      *problem.Problem
	lastQuery evidenceclient.CandleHistoryQuery
}

func (m *mockGetCandleHistory) Execute(_ context.Context, q evidenceclient.CandleHistoryQuery) (evidenceclient.CandleHistoryReply, *problem.Problem) {
	m.lastQuery = q
	return m.reply, m.prob
}

func TestEvidenceWebHandler_GetLatestCandle(t *testing.T) {
	now := time.Now().UTC().Truncate(60 * time.Second)
	candle := &evidence.EvidenceCandle{
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
		Final:      false,
	}

	handler := handlers.NewEvidenceWebHandler(
		&mockGetLatestCandle{reply: evidenceclient.CandleLatestReply{Candle: candle}},
		nil, nil, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestCandle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["candle"] == nil {
		t.Fatal("expected candle in response")
	}
}

func TestEvidenceWebHandler_GetLatestCandle_Unavailable(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestCandle(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetLatestCandle_MissingTimeframe(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(&mockGetLatestCandle{}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/latest?source=binancef&symbol=btcusdt", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestCandle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing timeframe, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetLatestCandle_InvalidTimeframe(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(&mockGetLatestCandle{}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=abc", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestCandle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid timeframe, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetLatestCandle_NullCandle(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(
		&mockGetLatestCandle{reply: evidenceclient.CandleLatestReply{Candle: nil}},
		nil, nil, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestCandle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for null candle, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, exists := body["candle"]; !exists {
		t.Fatal("expected candle key in response even when null")
	}
}

// --- History handler tests ---

func TestEvidenceWebHandler_GetCandleHistory(t *testing.T) {
	now := time.Now().UTC().Truncate(60 * time.Second)
	candles := []evidence.EvidenceCandle{
		{
			Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Open: "102.00", High: "106.00", Low: "101.00", Close: "104.00",
			Volume: "500.00", TradeCount: 20,
			OpenTime: now, CloseTime: now.Add(60 * time.Second), Final: true,
		},
	}

	handler := handlers.NewEvidenceWebHandler(
		nil,
		&mockGetCandleHistory{reply: evidenceclient.CandleHistoryReply{Candles: candles}},
		nil, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60&limit=5", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	arr, ok := body["candles"].([]any)
	if !ok || len(arr) != 1 {
		t.Fatal("expected candles array with 1 element")
	}
}

func TestEvidenceWebHandler_GetCandleHistory_WithRange(t *testing.T) {
	mock := &mockGetCandleHistory{
		reply: evidenceclient.CandleHistoryReply{Candles: []evidence.EvidenceCandle{}},
	}
	handler := handlers.NewEvidenceWebHandler(nil, mock, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60&since=1710000000&until=1710003600&limit=20", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if mock.lastQuery.Since != 1710000000 {
		t.Fatalf("expected since=1710000000, got %d", mock.lastQuery.Since)
	}
	if mock.lastQuery.Until != 1710003600 {
		t.Fatalf("expected until=1710003600, got %d", mock.lastQuery.Until)
	}
	if mock.lastQuery.Limit != 20 {
		t.Fatalf("expected limit=20, got %d", mock.lastQuery.Limit)
	}
}

func TestEvidenceWebHandler_GetCandleHistory_InvalidSince(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, &mockGetCandleHistory{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60&since=abc", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid since, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetCandleHistory_InvalidUntil(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, &mockGetCandleHistory{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60&until=abc", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid until, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetCandleHistory_Unavailable(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetCandleHistory_MissingTimeframe(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, &mockGetCandleHistory{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/history?source=binancef&symbol=btcusdt", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing timeframe, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetCandleHistory_InvalidLimit(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, &mockGetCandleHistory{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60&limit=999", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for limit > 100, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetCandleHistory_LimitZeroNotParsed(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, &mockGetCandleHistory{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60&limit=0", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for limit=0, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetCandleHistory_EmptyResult(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(
		nil,
		&mockGetCandleHistory{reply: evidenceclient.CandleHistoryReply{Candles: nil}},
		nil, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetCandleHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	arr, ok := body["candles"].([]any)
	if !ok {
		t.Fatal("expected candles array in response")
	}
	if len(arr) != 0 {
		t.Fatalf("expected empty candles array, got %d", len(arr))
	}
}

// --- Trade Burst handler mock and tests ---

type mockGetLatestTradeBurst struct {
	reply evidenceclient.TradeBurstLatestReply
	prob  *problem.Problem
}

func (m *mockGetLatestTradeBurst) Execute(_ context.Context, _ evidenceclient.TradeBurstLatestQuery) (evidenceclient.TradeBurstLatestReply, *problem.Problem) {
	return m.reply, m.prob
}

func TestEvidenceWebHandler_GetLatestTradeBurst(t *testing.T) {
	now := time.Now().UTC().Truncate(60 * time.Second)
	burst := &evidence.EvidenceTradeBurst{
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		TradeCount: 150,
		BuyVolume:  "500000.00",
		SellVolume: "300000.00",
		OpenTime:   now,
		CloseTime:  now.Add(60 * time.Second),
		Burst:      true,
		Final:      true,
	}

	handler := handlers.NewEvidenceWebHandler(
		nil, nil,
		&mockGetLatestTradeBurst{reply: evidenceclient.TradeBurstLatestReply{TradeBurst: burst}},
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/tradeburst/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestTradeBurst(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["trade_burst"] == nil {
		t.Fatal("expected trade_burst in response")
	}
}

func TestEvidenceWebHandler_GetLatestTradeBurst_Unavailable(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/tradeburst/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestTradeBurst(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetLatestTradeBurst_MissingTimeframe(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, nil, &mockGetLatestTradeBurst{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/tradeburst/latest?source=binancef&symbol=btcusdt", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestTradeBurst(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing timeframe, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetLatestTradeBurst_InvalidTimeframe(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, nil, &mockGetLatestTradeBurst{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/tradeburst/latest?source=binancef&symbol=btcusdt&timeframe=abc", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestTradeBurst(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid timeframe, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetLatestTradeBurst_NullResult(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(
		nil, nil,
		&mockGetLatestTradeBurst{reply: evidenceclient.TradeBurstLatestReply{TradeBurst: nil}},
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/tradeburst/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestTradeBurst(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for null result, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, exists := body["trade_burst"]; !exists {
		t.Fatal("expected trade_burst key in response even when null")
	}
}

func TestEvidenceWebHandler_GetLatestTradeBurst_UseCaseError(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(
		nil, nil,
		&mockGetLatestTradeBurst{prob: problem.New(problem.Unavailable, "store down")},
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/tradeburst/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestTradeBurst(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for use case error, got %d", rec.Code)
	}
}

// --- Volume handler mock and tests ---

type mockGetLatestVolume struct {
	reply evidenceclient.VolumeLatestReply
	prob  *problem.Problem
}

func (m *mockGetLatestVolume) Execute(_ context.Context, _ evidenceclient.VolumeLatestQuery) (evidenceclient.VolumeLatestReply, *problem.Problem) {
	return m.reply, m.prob
}

func TestEvidenceWebHandler_GetLatestVolume(t *testing.T) {
	now := time.Now().UTC().Truncate(60 * time.Second)
	vol := &evidence.EvidenceVolume{
		Source:      "binancef",
		Symbol:      "btcusdt",
		Timeframe:   60,
		BuyVolume:   "500000.00",
		SellVolume:  "300000.00",
		TotalVolume: "800000.00",
		VWAP:        "50000.12",
		TradeCount:  200,
		OpenTime:    now,
		CloseTime:   now.Add(60 * time.Second),
		Final:       true,
	}

	handler := handlers.NewEvidenceWebHandler(
		nil, nil, nil,
		&mockGetLatestVolume{reply: evidenceclient.VolumeLatestReply{Volume: vol}},
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/volume/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestVolume(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["volume"] == nil {
		t.Fatal("expected volume in response")
	}
}

func TestEvidenceWebHandler_GetLatestVolume_Unavailable(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/evidence/volume/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestVolume(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetLatestVolume_MissingTimeframe(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, nil, nil, &mockGetLatestVolume{})

	req := httptest.NewRequest(http.MethodGet, "/evidence/volume/latest?source=binancef&symbol=btcusdt", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestVolume(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing timeframe, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetLatestVolume_InvalidTimeframe(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(nil, nil, nil, &mockGetLatestVolume{})

	req := httptest.NewRequest(http.MethodGet, "/evidence/volume/latest?source=binancef&symbol=btcusdt&timeframe=abc", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestVolume(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid timeframe, got %d", rec.Code)
	}
}

func TestEvidenceWebHandler_GetLatestVolume_NullResult(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(
		nil, nil, nil,
		&mockGetLatestVolume{reply: evidenceclient.VolumeLatestReply{Volume: nil}},
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/volume/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestVolume(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for null result, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, exists := body["volume"]; !exists {
		t.Fatal("expected volume key in response even when null")
	}
}

func TestEvidenceWebHandler_GetLatestVolume_UseCaseError(t *testing.T) {
	handler := handlers.NewEvidenceWebHandler(
		nil, nil, nil,
		&mockGetLatestVolume{prob: problem.New(problem.Unavailable, "store down")},
	)

	req := httptest.NewRequest(http.MethodGet, "/evidence/volume/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	handler.GetLatestVolume(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for use case error, got %d", rec.Code)
	}
}
