package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/signalclient"
	"internal/domain/signal"
	"internal/interfaces/http/handlers"
	"internal/shared/problem"

	"github.com/julienschmidt/httprouter"
)

type mockGetLatestSignal struct {
	reply signalclient.SignalLatestReply
	prob  *problem.Problem
}

func (m *mockGetLatestSignal) Execute(_ context.Context, _ signalclient.SignalLatestQuery) (signalclient.SignalLatestReply, *problem.Problem) {
	return m.reply, m.prob
}

func signalRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

func TestSignalWebHandler_GetLatestSignal(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	sig := &signal.Signal{
		Type:      "rsi",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Value:     "65.32",
		Metadata:  map[string]string{"period": "14", "avg_gain": "1.20", "avg_loss": "0.64"},
		Final:     true,
		Timestamp: now,
	}

	handler := handlers.NewSignalWebHandler(
		&mockGetLatestSignal{reply: signalclient.SignalLatestReply{Signal: sig}},
	)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/signal/:type/latest", handler.GetLatestSignal)

	req := signalRequest(http.MethodGet, "/signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["signal"] == nil {
		t.Fatal("expected signal in response")
	}
}

func TestSignalWebHandler_GetLatestSignal_Unavailable(t *testing.T) {
	handler := handlers.NewSignalWebHandler(nil)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/signal/:type/latest", handler.GetLatestSignal)

	req := signalRequest(http.MethodGet, "/signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestSignalWebHandler_GetLatestSignal_MissingTimeframe(t *testing.T) {
	handler := handlers.NewSignalWebHandler(
		&mockGetLatestSignal{},
	)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/signal/:type/latest", handler.GetLatestSignal)

	req := signalRequest(http.MethodGet, "/signal/rsi/latest?source=binancef&symbol=btcusdt")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing timeframe, got %d", rec.Code)
	}
}

func TestSignalWebHandler_GetLatestSignal_NullSignal(t *testing.T) {
	handler := handlers.NewSignalWebHandler(
		&mockGetLatestSignal{reply: signalclient.SignalLatestReply{Signal: nil}},
	)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/signal/:type/latest", handler.GetLatestSignal)

	req := signalRequest(http.MethodGet, "/signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for null signal, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, exists := body["signal"]; !exists {
		t.Fatal("expected signal key in response even when null")
	}
}
