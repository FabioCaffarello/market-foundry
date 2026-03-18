package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/riskclient"
	"internal/domain/risk"
	"internal/interfaces/http/handlers"
	"internal/shared/problem"

	"github.com/julienschmidt/httprouter"
)

type mockGetLatestRisk struct {
	reply riskclient.RiskLatestReply
	prob  *problem.Problem
}

func (m *mockGetLatestRisk) Execute(_ context.Context, _ riskclient.RiskLatestQuery) (riskclient.RiskLatestReply, *problem.Problem) {
	return m.reply, m.prob
}

func riskRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

func TestRiskWebHandler_GetLatestRisk(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	assessment := &risk.RiskAssessment{
		Type:        "position_exposure",
		Source:      "binancef",
		Symbol:      "btcusdt",
		Timeframe:   60,
		Disposition: risk.DispositionApproved,
		Confidence:  "0.85",
		Strategies: []risk.StrategyInput{
			{Type: "mean_reversion_entry", Direction: "long", Confidence: "0.85", Timeframe: 60},
		},
		Constraints: risk.Constraints{MaxPositionSize: "0.01", MaxExposure: "0.05"},
		Rationale:   "Position size within exposure limits",
		Final:       true,
		Timestamp:   now,
	}

	handler := handlers.NewRiskWebHandler(
		&mockGetLatestRisk{reply: riskclient.RiskLatestReply{RiskAssessment: assessment}},
	)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/risk/:type/latest", handler.GetLatestRisk)

	req := riskRequest(http.MethodGet, "/risk/position_exposure/latest?source=binancef&symbol=btcusdt&timeframe=60")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["risk_assessment"] == nil {
		t.Fatal("expected risk_assessment in response")
	}
}

func TestRiskWebHandler_GetLatestRisk_Unavailable(t *testing.T) {
	handler := handlers.NewRiskWebHandler(nil)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/risk/:type/latest", handler.GetLatestRisk)

	req := riskRequest(http.MethodGet, "/risk/position_exposure/latest?source=binancef&symbol=btcusdt&timeframe=60")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestRiskWebHandler_GetLatestRisk_MissingTimeframe(t *testing.T) {
	handler := handlers.NewRiskWebHandler(
		&mockGetLatestRisk{},
	)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/risk/:type/latest", handler.GetLatestRisk)

	req := riskRequest(http.MethodGet, "/risk/position_exposure/latest?source=binancef&symbol=btcusdt")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing timeframe, got %d", rec.Code)
	}
}

func TestRiskWebHandler_GetLatestRisk_NullRisk(t *testing.T) {
	handler := handlers.NewRiskWebHandler(
		&mockGetLatestRisk{reply: riskclient.RiskLatestReply{RiskAssessment: nil}},
	)

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/risk/:type/latest", handler.GetLatestRisk)

	req := riskRequest(http.MethodGet, "/risk/position_exposure/latest?source=binancef&symbol=btcusdt&timeframe=60")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for null risk, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, exists := body["risk_assessment"]; !exists {
		t.Fatal("expected risk_assessment key in response even when null")
	}
}
