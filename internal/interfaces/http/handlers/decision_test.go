package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/decisionclient"
	"internal/domain/decision"
	"internal/interfaces/http/handlers"
	"internal/shared/problem"

	"github.com/julienschmidt/httprouter"
)

type mockGetLatestDecision struct {
	reply decisionclient.DecisionLatestReply
	prob  *problem.Problem
}

func (m *mockGetLatestDecision) Execute(_ context.Context, _ decisionclient.DecisionLatestQuery) (decisionclient.DecisionLatestReply, *problem.Problem) {
	return m.reply, m.prob
}

func decisionRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

func TestDecisionWebHandler_GetLatestDecision(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	dec := &decision.Decision{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Outcome:    decision.OutcomeTriggered,
		Confidence: "0.85",
		Signals: []decision.SignalInput{
			{Type: "rsi", Value: "25.00", Timeframe: 60},
		},
		Metadata:  map[string]string{"threshold": "30.0"},
		Final:     true,
		Timestamp: now,
	}

	handler := handlers.NewDecisionWebHandler(
		&mockGetLatestDecision{reply: decisionclient.DecisionLatestReply{Decision: dec}},
	)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/decision/:type/latest", handler.GetLatestDecision)

	req := decisionRequest(http.MethodGet, "/decision/rsi_oversold/latest?source=binancef&symbol=btcusdt&timeframe=60")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["decision"] == nil {
		t.Fatal("expected decision in response")
	}
}

func TestDecisionWebHandler_GetLatestDecision_Unavailable(t *testing.T) {
	handler := handlers.NewDecisionWebHandler(nil)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/decision/:type/latest", handler.GetLatestDecision)

	req := decisionRequest(http.MethodGet, "/decision/rsi_oversold/latest?source=binancef&symbol=btcusdt&timeframe=60")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestDecisionWebHandler_GetLatestDecision_MissingTimeframe(t *testing.T) {
	handler := handlers.NewDecisionWebHandler(
		&mockGetLatestDecision{},
	)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/decision/:type/latest", handler.GetLatestDecision)

	req := decisionRequest(http.MethodGet, "/decision/rsi_oversold/latest?source=binancef&symbol=btcusdt")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing timeframe, got %d", rec.Code)
	}
}

func TestDecisionWebHandler_GetLatestDecision_NullDecision(t *testing.T) {
	handler := handlers.NewDecisionWebHandler(
		&mockGetLatestDecision{reply: decisionclient.DecisionLatestReply{Decision: nil}},
	)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/decision/:type/latest", handler.GetLatestDecision)

	req := decisionRequest(http.MethodGet, "/decision/rsi_oversold/latest?source=binancef&symbol=btcusdt&timeframe=60")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for null decision, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, exists := body["decision"]; !exists {
		t.Fatal("expected decision key in response even when null")
	}
}
