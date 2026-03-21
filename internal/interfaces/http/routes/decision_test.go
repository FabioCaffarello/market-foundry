package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/decisionclient"
	"internal/domain/decision"
	"internal/shared/problem"

	"github.com/julienschmidt/httprouter"
)

type getLatestDecisionUseCaseStub struct {
	dec  *decision.Decision
	prob *problem.Problem
}

func (s getLatestDecisionUseCaseStub) Execute(_ context.Context, _ decisionclient.DecisionLatestQuery) (decisionclient.DecisionLatestReply, *problem.Problem) {
	return decisionclient.DecisionLatestReply{Decision: s.dec}, s.prob
}

func TestDecisionRoutesRegisterHandler(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	routes := Decision(DecisionFamilyDeps{
		GetLatestDecision: getLatestDecisionUseCaseStub{
			dec: &decision.Decision{
				Type:       "rsi_oversold",
				Source:     "binancef",
				Symbol:     "btcusdt",
				Timeframe:  60,
				Outcome:    decision.OutcomeTriggered,
				Severity:   decision.SeverityLow,
				Confidence: "0.85",
				Rationale:  "RSI 25.00 below oversold threshold 30.0 (distance 16.7%); severity low",
				Signals: []decision.SignalInput{
					{Type: "rsi", Value: "25.00", Timeframe: 60},
				},
				Metadata:  map[string]string{"threshold": "30.0"},
				Final:     true,
				Timestamp: now,
			},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/decision/rsi_oversold/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /decision/rsi_oversold/latest: expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestDefaultRoutesIncludesDecisionWhenProvided(t *testing.T) {
	t.Parallel()

	routes := DefaultRoutes(Dependencies{
		CreateDraft:    createDraftUseCaseStub{},
		GetConfig:      getConfigUseCaseStub{},
		GetActive:      getActiveUseCaseStub{},
		ListConfigs:    listConfigsUseCaseStub{},
		ValidateDraft:  validateDraftUseCaseStub{},
		ValidateConfig: validateConfigUseCaseStub{},
		CompileConfig:  compileConfigUseCaseStub{},
		ActivateConfig: activateConfigUseCaseStub{},
		Decision: DecisionFamilyDeps{
			GetLatestDecision: getLatestDecisionUseCaseStub{},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/decision/rsi_oversold/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected decision latest route to be registered, got %d", rec.Code)
	}
}

func TestDefaultRoutesOmitsDecisionWhenNil(t *testing.T) {
	t.Parallel()

	routes := DefaultRoutes(Dependencies{
		CreateDraft:    createDraftUseCaseStub{},
		GetConfig:      getConfigUseCaseStub{},
		GetActive:      getActiveUseCaseStub{},
		ListConfigs:    listConfigsUseCaseStub{},
		ValidateDraft:  validateDraftUseCaseStub{},
		ValidateConfig: validateConfigUseCaseStub{},
		CompileConfig:  compileConfigUseCaseStub{},
		ActivateConfig: activateConfigUseCaseStub{},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/decision/rsi_oversold/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected decision route to be absent (404), got %d", rec.Code)
	}
}
