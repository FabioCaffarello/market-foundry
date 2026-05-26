package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/strategyclient"
	"internal/domain/strategy"
	"internal/shared/problem"

	"github.com/julienschmidt/httprouter"
)

type getLatestStrategyUseCaseStub struct {
	strat *strategy.Strategy
	prob  *problem.Problem
}

func (s getLatestStrategyUseCaseStub) Execute(_ context.Context, _ strategyclient.StrategyLatestQuery) (strategyclient.StrategyLatestReply, *problem.Problem) {
	return strategyclient.StrategyLatestReply{Strategy: s.strat}, s.prob
}

func TestStrategyRoutesRegisterHandler(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	routes := Strategy(StrategyFamilyDeps{
		GetLatestStrategy: getLatestStrategyUseCaseStub{
			strat: &strategy.Strategy{
				Type:       "mean_reversion_entry",
				Source:     "binancef",
				Instrument: btcUSDTPerpRiskRoute(t),
				Timeframe:  60,
				Direction:  strategy.DirectionLong,
				Confidence: "0.85",
				Decisions: []strategy.DecisionInput{
					{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Timeframe: 60},
				},
				Parameters: map[string]string{"entry": "market"},
				Final:      true,
				Timestamp:  now,
			},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /strategy/mean_reversion_entry/latest: expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestDefaultRoutesIncludesStrategyWhenProvided(t *testing.T) {
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
		Strategy: StrategyFamilyDeps{
			GetLatestStrategy: getLatestStrategyUseCaseStub{},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected strategy latest route to be registered, got %d", rec.Code)
	}
}

func TestDefaultRoutesOmitsStrategyWhenNil(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/strategy/mean_reversion_entry/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected strategy route to be absent (404), got %d", rec.Code)
	}
}
