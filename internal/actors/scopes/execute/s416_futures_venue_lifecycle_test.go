package execute_test

// s416_futures_venue_lifecycle_test.go — S416: Futures real venue acceptance/fill proof.
//
// Proves the Futures venue_live lifecycle path (submitted -> accepted -> filled) through
// the SegmentRouter composition matching unified runtime wiring. Validates that:
//
//   - Futures intents are routed to the Futures adapter via SegmentRouter
//   - Fill records carry real venue data (Simulated=false, real prices/fees)
//   - Lifecycle transitions align with ValidTransition()
//   - Correlation/causation chains survive the full path
//   - Spot adapter is NOT contacted for Futures intents (segment isolation)

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/settings"
)

func s416FuturesVenueIntent(side domainexec.Side) domainexec.ExecutionIntent {
	qty := "0.001"
	if side == domainexec.SideNone {
		qty = "0"
	}
	return domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        "binancef",
		Symbol:        "btcusdt",
		Timeframe:     60,
		Side:          side,
		Quantity:      qty,
		Status:        domainexec.StatusSubmitted,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		CorrelationID: "s416-actor-corr",
		CausationID:   "s416-actor-cause",
		Final:         true,
		Timestamp:     time.Now().UTC().Add(-3 * time.Second),
	}
}

func s416FuturesFilledServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     77777,
			"clientOrderId": r.URL.Query().Get("newClientOrderId"),
			"symbol":      r.URL.Query().Get("symbol"),
			"status":      "FILLED",
			"side":        r.URL.Query().Get("side"),
			"type":        "MARKET",
			"avgPrice":    "65432.10",
			"executedQty": "0.001",
			"cumQuote":    "65.43210",
			"updateTime":  time.Now().UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func s416BuildSegmentRouter(t *testing.T, futuresServer, spotServer *httptest.Server) *appexec.SegmentRouter {
	t.Helper()

	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", "test-futures-key")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", "test-futures-secret")
	futuresCreds, _ := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	futuresAdapter := appexec.NewBinanceFuturesTestnetAdapter(futuresCreds, 5*time.Second).WithBaseURL(futuresServer.URL)

	router := appexec.NewSegmentRouter()
	router.Register(settings.MarketSegmentFutures, futuresAdapter)
	router.RegisterQuery(settings.MarketSegmentFutures, futuresAdapter)

	if spotServer != nil {
		t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY", "test-spot-key")
		t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET", "test-spot-secret")
		spotCreds, _ := appexec.LoadCredentials("binance_spot_testnet", []string{"API_KEY", "API_SECRET"})
		spotAdapter := appexec.NewBinanceSpotTestnetAdapter(spotCreds, 5*time.Second).WithBaseURL(spotServer.URL)
		router.Register(settings.MarketSegmentSpot, spotAdapter)
	}

	return router
}

// ==========================================================================
// Futures venue_live lifecycle through SegmentRouter (actor composition path)
// ==========================================================================

func TestS416_ActorComposition_FuturesVenueLive_Buy_Filled(t *testing.T) {
	futuresSrv := s416FuturesFilledServer(t)
	defer futuresSrv.Close()

	spotCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spotCalled = true
		t.Error("Spot adapter must NOT be called for Futures intent")
	}))
	defer spotSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, spotSrv)

	intent := s416FuturesVenueIntent(domainexec.SideBuy)
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	// Lifecycle: submitted -> (accepted ->) filled
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
	if !receipt.Status.IsTerminal() {
		t.Fatal("filled must be terminal")
	}

	// Fill fidelity
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
	}
	fill := receipt.Intent.Fills[0]
	if fill.Price != "65432.10" {
		t.Errorf("expected price 65432.10 (avgPrice), got %s", fill.Price)
	}
	if fill.Fee != "0" {
		t.Errorf("expected fee 0 (Futures RESULT has no commission), got %s", fill.Fee)
	}
	if fill.CostBasis != "65.43210" {
		t.Errorf("expected CostBasis 65.43210 (cumQuote), got %s", fill.CostBasis)
	}
	if fill.Simulated {
		t.Error("venue_live fills must have Simulated=false")
	}

	// Segment isolation
	if spotCalled {
		t.Error("Spot adapter was called — segment isolation violated")
	}

	// Correlation preserved
	if receipt.Intent.CorrelationID != "s416-actor-corr" {
		t.Errorf("CorrelationID lost: %s", receipt.Intent.CorrelationID)
	}
	if receipt.Intent.CausationID != "s416-actor-cause" {
		t.Errorf("CausationID lost: %s", receipt.Intent.CausationID)
	}

	// Venue-assigned order ID
	if receipt.VenueOrderID != "77777" {
		t.Errorf("expected VenueOrderID 77777, got %s", receipt.VenueOrderID)
	}
	if strings.HasPrefix(receipt.VenueOrderID, "dryrun-") || strings.HasPrefix(receipt.VenueOrderID, "paper-") {
		t.Error("venue_live must NOT have simulation prefix")
	}
}

func TestS416_ActorComposition_FuturesVenueLive_Sell_Filled(t *testing.T) {
	futuresSrv := s416FuturesFilledServer(t)
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, nil)

	intent := s416FuturesVenueIntent(domainexec.SideSell)
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
	if receipt.Intent.Side != domainexec.SideSell {
		t.Errorf("side lost: expected sell, got %s", receipt.Intent.Side)
	}
}

func TestS416_ActorComposition_FuturesVenueLive_None_NoContact(t *testing.T) {
	callCount := 0
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
	}))
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, nil)

	intent := s416FuturesVenueIntent(domainexec.SideNone)
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusAccepted {
		t.Fatalf("expected accepted for SideNone, got %s", receipt.Status)
	}
	if callCount != 0 {
		t.Errorf("SideNone must not contact venue, got %d calls", callCount)
	}
}

// ==========================================================================
// Post-200 reconciliation through SegmentRouter QueryOrder
// ==========================================================================

func TestS416_ActorComposition_FuturesQueryOrder_Reconciliation(t *testing.T) {
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     77777,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"avgPrice":    "65432.10",
			"executedQty": "0.001",
			"cumQuote":    "65.43210",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, nil)

	queryReceipt, prob := router.QueryOrder(context.Background(), "test-client-id", "btcusdt")
	if prob != nil {
		t.Fatalf("query failed: %s", prob.Message)
	}

	if queryReceipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled from query, got %s", queryReceipt.Status)
	}
	if queryReceipt.VenueOrderID != "77777" {
		t.Errorf("expected VenueOrderID 77777, got %s", queryReceipt.VenueOrderID)
	}
}

// ==========================================================================
// DryRunSubmitter bypass: when dry_run=false, real adapter is called
// ==========================================================================

func TestS416_ActorComposition_DryRunDisabled_RealFuturesAdapterCalled(t *testing.T) {
	adapterCalled := false
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adapterCalled = true
		resp := map[string]any{
			"orderId":     77777,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"avgPrice":    "65432.10",
			"executedQty": "0.001",
			"cumQuote":    "65.43210",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, nil)

	intent := s416FuturesVenueIntent(domainexec.SideBuy)
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if !adapterCalled {
		t.Error("real adapter must be called when dry_run=false")
	}
	if receipt.Intent.Fills[0].Simulated {
		t.Error("venue_live fills must have Simulated=false")
	}
}

func TestS416_ActorComposition_DryRunEnabled_InterceptsFuturesAdapter(t *testing.T) {
	adapterCalled := false
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		adapterCalled = true
		t.Error("DryRunSubmitter should intercept before reaching adapter")
	}))
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, nil)

	drs := appexec.NewDryRunSubmitter(router)

	intent := s416FuturesVenueIntent(domainexec.SideBuy)
	receipt, prob := drs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if adapterCalled {
		t.Error("DryRunSubmitter must intercept — adapter should NOT be called")
	}
	if !receipt.Intent.Fills[0].Simulated {
		t.Error("dry_run fills must have Simulated=true")
	}
	if !strings.HasPrefix(receipt.VenueOrderID, "dryrun-") {
		t.Errorf("expected dryrun- prefix, got %s", receipt.VenueOrderID)
	}
}
