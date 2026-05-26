package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/signalclient"
	"internal/domain/signal"
	"internal/shared/problem"

	"github.com/julienschmidt/httprouter"
)

type getLatestSignalUseCaseStub struct {
	sig  *signal.Signal
	prob *problem.Problem
}

func (s getLatestSignalUseCaseStub) Execute(_ context.Context, _ signalclient.SignalLatestQuery) (signalclient.SignalLatestReply, *problem.Problem) {
	return signalclient.SignalLatestReply{Signal: s.sig}, s.prob
}

func TestSignalRoutesRegisterHandler(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	routes := Signal(SignalFamilyDeps{
		GetLatestSignal: getLatestSignalUseCaseStub{
			sig: &signal.Signal{
				Type:       "rsi",
				Source:     "binancef",
				Instrument: btcUSDTPerp(t),
				Timeframe:  60,
				Value:      "65.32",
				Metadata:   map[string]string{"period": "14"},
				Final:      true,
				Timestamp:  now,
			},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /signal/rsi/latest: expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestDefaultRoutesIncludesSignalWhenProvided(t *testing.T) {
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
		Signal: SignalFamilyDeps{
			GetLatestSignal: getLatestSignalUseCaseStub{},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected signal latest route to be registered, got %d", rec.Code)
	}
}

func TestDefaultRoutesOmitsSignalWhenNil(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/signal/rsi/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected signal route to be absent (404), got %d", rec.Code)
	}
}
