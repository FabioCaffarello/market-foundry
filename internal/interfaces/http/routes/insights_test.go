package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/insightsclient"
	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/shared/problem"

	"github.com/julienschmidt/httprouter"
)

type getLatestVolumeProfileStub struct {
	reply insightsclient.VolumeProfileLatestReply
	prob  *problem.Problem
}

func (s getLatestVolumeProfileStub) Execute(_ context.Context, _ insightsclient.VolumeProfileLatestQuery) (insightsclient.VolumeProfileLatestReply, *problem.Problem) {
	return s.reply, s.prob
}

type getLatestTPOProfileStub struct {
	reply insightsclient.TPOProfileLatestReply
	prob  *problem.Problem
}

func (s getLatestTPOProfileStub) Execute(_ context.Context, _ insightsclient.TPOProfileLatestQuery) (insightsclient.TPOProfileLatestReply, *problem.Problem) {
	return s.reply, s.prob
}

type getLatestCrossVenueStub struct {
	reply insightsclient.CrossVenueLatestReply
	prob  *problem.Problem
}

func (s getLatestCrossVenueStub) Execute(_ context.Context, _ insightsclient.CrossVenueLatestQuery) (insightsclient.CrossVenueLatestReply, *problem.Problem) {
	return s.reply, s.prob
}

func TestInsightsRoutesServeVolumeProfile(t *testing.T) {
	t.Parallel()

	inst, _ := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	routes := Insights(InsightsFamilyDeps{
		GetLatestVolumeProfile: getLatestVolumeProfileStub{
			reply: insightsclient.VolumeProfileLatestReply{
				VolumeProfile: &insights.VolumeProfile{
					Source:     "binancef",
					Instrument: inst,
					Timeframe:  60,
					BucketSize: "1",
					Buckets:    []insights.PriceBucket{{PriceLevel: "65000", BuyVolume: "10", SellVolume: "5"}},
					Overload:   insights.OverloadL0,
					OpenTime:   time.Now().UTC(),
					CloseTime:  time.Now().UTC().Add(time.Minute),
					Final:      true,
				},
			},
		},
		GetLatestTPOProfile: getLatestTPOProfileStub{},
		GetLatestCrossVenue: getLatestCrossVenueStub{},
	})
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/insights/volume-profile/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		VolumeProfile *insights.VolumeProfile `json:"volume_profile"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.VolumeProfile == nil || len(body.VolumeProfile.Buckets) != 1 {
		t.Errorf("expected 1-bucket profile, got %+v", body.VolumeProfile)
	}
}

func TestInsightsRoutesServeTPO(t *testing.T) {
	t.Parallel()

	inst, _ := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	routes := Insights(InsightsFamilyDeps{
		GetLatestTPOProfile: getLatestTPOProfileStub{
			reply: insightsclient.TPOProfileLatestReply{
				TPOProfile: &insights.TPOProfile{
					Source:        "binancef",
					Instrument:    inst,
					Timeframe:     3600,
					BucketSize:    "1",
					PeriodSeconds: 600,
					Periods:       []insights.TPOPeriod{{Letter: "A", HighPrice: "65010", LowPrice: "65000"}},
					Levels:        []insights.TPOLevel{{PriceLevel: "65000", Letters: "A", Count: 1}},
					POCPrice:      "65000",
					Overload:      insights.OverloadL0,
					OpenTime:      time.Now().UTC(),
					CloseTime:     time.Now().UTC().Add(time.Hour),
					Final:         true,
				},
			},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/insights/tpo/latest?source=binancef&base=btc&quote=usdt&contract=perpetual&timeframe=3600", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		TPOProfile *insights.TPOProfile `json:"tpo_profile"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.TPOProfile == nil || body.TPOProfile.POCPrice != "65000" {
		t.Errorf("expected TPO profile with POC 65000, got %+v", body.TPOProfile)
	}
}

func TestInsightsRoutesServeCrossVenue(t *testing.T) {
	t.Parallel()

	inst, _ := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	routes := Insights(InsightsFamilyDeps{
		GetLatestCrossVenue: getLatestCrossVenueStub{
			reply: insightsclient.CrossVenueLatestReply{
				CrossVenueSnapshot: &insights.CrossVenueSnapshot{
					Instrument: inst,
					Timeframe:  60,
					Venues: []insights.VenueRow{
						{Venue: "binancef", TradeCount: 1, Notional: "65000.00000000", LastPrice: "65000", HighPrice: "65000", LowPrice: "65000"},
					},
					SpreadAbs:     "0.00000000",
					MidPrice:      "65000.00000000",
					DominantVenue: "binancef",
					OpenTime:      time.Now().UTC(),
					CloseTime:     time.Now().UTC().Add(time.Minute),
					Final:         true,
				},
			},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	// No source param — cross-venue spans sources.
	req := httptest.NewRequest(http.MethodGet,
		"/insights/cross-venue/latest?base=btc&quote=usdt&contract=perpetual&timeframe=60", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		CrossVenueSnapshot *insights.CrossVenueSnapshot `json:"cross_venue_snapshot"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.CrossVenueSnapshot == nil || body.CrossVenueSnapshot.DominantVenue != "binancef" {
		t.Errorf("expected cross-venue snapshot with dominant binancef, got %+v", body.CrossVenueSnapshot)
	}
}

func TestInsightsFamilyDeps_HasAny(t *testing.T) {
	t.Parallel()
	if (InsightsFamilyDeps{}).HasAny() {
		t.Error("empty deps must report HasAny=false")
	}
	if !(InsightsFamilyDeps{GetLatestVolumeProfile: getLatestVolumeProfileStub{}}).HasAny() {
		t.Error("VP-wired deps must report HasAny=true")
	}
	if !(InsightsFamilyDeps{GetLatestTPOProfile: getLatestTPOProfileStub{}}).HasAny() {
		t.Error("TPO-wired deps must report HasAny=true")
	}
	if !(InsightsFamilyDeps{GetLatestCrossVenue: getLatestCrossVenueStub{}}).HasAny() {
		t.Error("cross-venue-wired deps must report HasAny=true")
	}
}
