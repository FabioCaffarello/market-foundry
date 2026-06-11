package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/riskclient"
	"internal/domain/instrument"
	"internal/domain/risk"
	"internal/shared/problem"

	"github.com/julienschmidt/httprouter"
)

func btcUSDTPerpRiskRoute(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

type getLatestRiskUseCaseStub struct {
	assessment *risk.RiskAssessment
	prob       *problem.Problem
}

func (s getLatestRiskUseCaseStub) Execute(_ context.Context, _ riskclient.RiskLatestQuery) (riskclient.RiskLatestReply, *problem.Problem) {
	return riskclient.RiskLatestReply{RiskAssessment: s.assessment}, s.prob
}

func TestRiskRoutesRegisterHandler(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	routes := Risk(RiskFamilyDeps{
		GetLatestRisk: getLatestRiskUseCaseStub{
			assessment: &risk.RiskAssessment{
				Type:        "position_exposure",
				Source:      "binancef",
				Instrument:  btcUSDTPerpRiskRoute(t),
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
			},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/risk/position_exposure/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /risk/position_exposure/latest: expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestDefaultRoutesIncludesRiskWhenProvided(t *testing.T) {
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
		Risk: RiskFamilyDeps{
			GetLatestRisk: getLatestRiskUseCaseStub{},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/risk/position_exposure/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected risk latest route to be registered, got %d", rec.Code)
	}
}

func TestDefaultRoutesOmitsRiskWhenNil(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/risk/position_exposure/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected risk route to be absent (404), got %d", rec.Code)
	}
}
