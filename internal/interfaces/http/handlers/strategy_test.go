package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/strategyclient"
	"internal/domain/strategy"
	"internal/interfaces/http/handlers"
	"internal/shared/problem"

	"github.com/julienschmidt/httprouter"
)

type mockGetLatestStrategy struct {
	reply strategyclient.StrategyLatestReply
	prob  *problem.Problem
}

func (m *mockGetLatestStrategy) Execute(_ context.Context, _ strategyclient.StrategyLatestQuery) (strategyclient.StrategyLatestReply, *problem.Problem) {
	return m.reply, m.prob
}

func strategyRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

func TestStrategyWebHandler_GetLatestStrategy(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	strat := &strategy.Strategy{
		Type:       "mean_reversion_entry",
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Direction:  strategy.DirectionLong,
		Confidence: "0.85",
		Decisions: []strategy.DecisionInput{
			{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Timeframe: 60},
		},
		Parameters: map[string]string{"entry": "market", "target_offset": "0.02", "stop_offset": "0.01"},
		Final:      true,
		Timestamp:  now,
	}

	handler := handlers.NewStrategyWebHandler(
		&mockGetLatestStrategy{reply: strategyclient.StrategyLatestReply{Strategy: strat}},
	)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/strategy/:type/latest", handler.GetLatestStrategy)

	req := strategyRequest(http.MethodGet, "/strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["strategy"] == nil {
		t.Fatal("expected strategy in response")
	}
}

func TestStrategyWebHandler_GetLatestStrategy_Unavailable(t *testing.T) {
	handler := handlers.NewStrategyWebHandler(nil)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/strategy/:type/latest", handler.GetLatestStrategy)

	req := strategyRequest(http.MethodGet, "/strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestStrategyWebHandler_GetLatestStrategy_MissingTimeframe(t *testing.T) {
	handler := handlers.NewStrategyWebHandler(
		&mockGetLatestStrategy{},
	)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/strategy/:type/latest", handler.GetLatestStrategy)

	req := strategyRequest(http.MethodGet, "/strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing timeframe, got %d", rec.Code)
	}
}

func TestStrategyWebHandler_GetLatestStrategy_MultiSymbol_NoBleed(t *testing.T) {
	// Verify that querying for different symbols returns the correct symbol in the response.
	symbols := []struct {
		symbol    string
		direction strategy.Direction
	}{
		{"btcusdt", strategy.DirectionLong},
		{"ethusdt", strategy.DirectionShort},
	}

	for _, tc := range symbols {
		now := time.Now().UTC().Truncate(time.Second)
		strat := &strategy.Strategy{
			Type:       "mean_reversion_entry",
			Source:     "binancef",
			Symbol:     tc.symbol,
			Timeframe:  60,
			Direction:  tc.direction,
			Confidence: "0.85",
			Decisions: []strategy.DecisionInput{
				{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Timeframe: 60},
			},
			Parameters: map[string]string{"entry": "market"},
			Final:      true,
			Timestamp:  now,
		}

		handler := handlers.NewStrategyWebHandler(
			&mockGetLatestStrategy{reply: strategyclient.StrategyLatestReply{Strategy: strat}},
		)

		router := httprouter.New()
		router.HandlerFunc(http.MethodGet, "/strategy/:type/latest", handler.GetLatestStrategy)

		req := strategyRequest(http.MethodGet, "/strategy/mean_reversion_entry/latest?source=binancef&symbol="+tc.symbol+"&timeframe=60")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("symbol=%s: expected 200, got %d", tc.symbol, rec.Code)
		}

		var body map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("symbol=%s: decode body: %v", tc.symbol, err)
		}
		stratMap, ok := body["strategy"].(map[string]any)
		if !ok {
			t.Fatalf("symbol=%s: expected strategy object in response", tc.symbol)
		}
		if stratMap["symbol"] != tc.symbol {
			t.Fatalf("symbol=%s: response has symbol=%v — CROSS-SYMBOL BLEED", tc.symbol, stratMap["symbol"])
		}
		if stratMap["direction"] != string(tc.direction) {
			t.Fatalf("symbol=%s: expected direction=%s, got %v", tc.symbol, tc.direction, stratMap["direction"])
		}
	}
}

func TestStrategyWebHandler_GetLatestStrategy_NullStrategy(t *testing.T) {
	handler := handlers.NewStrategyWebHandler(
		&mockGetLatestStrategy{reply: strategyclient.StrategyLatestReply{Strategy: nil}},
	)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/strategy/:type/latest", handler.GetLatestStrategy)

	req := strategyRequest(http.MethodGet, "/strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for null strategy, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, exists := body["strategy"]; !exists {
		t.Fatal("expected strategy key in response even when null")
	}
}
