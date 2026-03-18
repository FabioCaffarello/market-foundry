package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/evidenceclient"
	"internal/domain/evidence"
	"internal/shared/problem"

	"github.com/julienschmidt/httprouter"
)

type getLatestCandleUseCaseStub struct {
	candle *evidence.EvidenceCandle
	prob   *problem.Problem
}

func (s getLatestCandleUseCaseStub) Execute(_ context.Context, _ evidenceclient.CandleLatestQuery) (evidenceclient.CandleLatestReply, *problem.Problem) {
	return evidenceclient.CandleLatestReply{Candle: s.candle}, s.prob
}

type getCandleHistoryUseCaseStub struct {
	candles []evidence.EvidenceCandle
	prob    *problem.Problem
}

func (s getCandleHistoryUseCaseStub) Execute(_ context.Context, _ evidenceclient.CandleHistoryQuery) (evidenceclient.CandleHistoryReply, *problem.Problem) {
	return evidenceclient.CandleHistoryReply{Candles: s.candles}, s.prob
}

type getLatestTradeBurstUseCaseStub struct {
	burst *evidence.EvidenceTradeBurst
	prob  *problem.Problem
}

func (s getLatestTradeBurstUseCaseStub) Execute(_ context.Context, _ evidenceclient.TradeBurstLatestQuery) (evidenceclient.TradeBurstLatestReply, *problem.Problem) {
	return evidenceclient.TradeBurstLatestReply{TradeBurst: s.burst}, s.prob
}

func TestEvidenceRoutesRegisterHandlers(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(60 * time.Second)
	routes := Evidence(EvidenceFamilyDeps{
		GetLatestCandle: getLatestCandleUseCaseStub{
			candle: &evidence.EvidenceCandle{
				Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
				Open: "100.00", High: "105.00", Low: "99.00", Close: "102.00",
				Volume: "1000.00", TradeCount: 42,
				OpenTime: now, CloseTime: now.Add(60 * time.Second), Final: false,
			},
		},
		GetCandleHistory: getCandleHistoryUseCaseStub{candles: []evidence.EvidenceCandle{}},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /evidence/candles/latest: expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestEvidenceRoutesRegisterHistoryHandler(t *testing.T) {
	t.Parallel()

	routes := Evidence(EvidenceFamilyDeps{
		GetCandleHistory: getCandleHistoryUseCaseStub{candles: []evidence.EvidenceCandle{}},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/history?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /evidence/candles/history: expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestEvidenceRoutesRegisterTradeBurstHandler(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(60 * time.Second)
	routes := Evidence(EvidenceFamilyDeps{
		GetLatestTradeBurst: getLatestTradeBurstUseCaseStub{
			burst: &evidence.EvidenceTradeBurst{
				Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
				TradeCount: 100, BuyVolume: "500.00", SellVolume: "300.00",
				OpenTime: now, CloseTime: now.Add(60 * time.Second),
				Burst: true, Final: true,
			},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/evidence/tradeburst/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /evidence/tradeburst/latest: expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestDefaultRoutesIncludesEvidenceWhenProvided(t *testing.T) {
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
		Evidence: EvidenceFamilyDeps{
			GetLatestCandle:     getLatestCandleUseCaseStub{},
			GetCandleHistory:    getCandleHistoryUseCaseStub{candles: []evidence.EvidenceCandle{}},
			GetLatestTradeBurst: getLatestTradeBurstUseCaseStub{},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/latest?source=binancef&symbol=btcusdt&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected evidence latest route to be registered, got %d", rec.Code)
	}
}

func TestDefaultRoutesOmitsEvidenceWhenNil(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/evidence/candles/latest", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected evidence route to be absent (404), got %d", rec.Code)
	}
}
