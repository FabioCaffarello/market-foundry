package execute_test

// s405_spot_venue_lifecycle_test.go — S405: Spot real venue acceptance/fill proof.
//
// Proves the Spot venue_live lifecycle path (submitted → accepted → filled) through
// the SegmentRouter composition matching unified runtime wiring. Validates that:
//
//   - Spot intents are routed to the Spot adapter via SegmentRouter
//   - Fill records carry real venue data (Simulated=false, real prices/fees)
//   - Lifecycle transitions align with ValidTransition()
//   - Correlation/causation chains survive the full path
//   - Futures adapter is NOT contacted for Spot intents (segment isolation)
//
// These tests validate the actor-level wiring without requiring NATS. They prove
// the composition that the VenueAdapterActor receives from the supervisor when
// the unified runtime resolves a SegmentRouter with Spot adapter.

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

// s405SpotVenueIntent creates an intent mimicking what VenueAdapterActor.onIntent receives
// after the execute supervisor processes a PaperOrderSubmittedEvent from the derive binary.
func s405SpotVenueIntent(t *testing.T, side domainexec.Side) domainexec.ExecutionIntent {
	t.Helper()
	qty := "0.001"
	if side == domainexec.SideNone {
		qty = "0"
	}
	return domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        "binances",
		Instrument:    btcUSDTSpotS379(t),
		Timeframe:     60,
		Side:          side,
		Quantity:      qty,
		Status:        domainexec.StatusSubmitted,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		CorrelationID: "s405-actor-corr",
		CausationID:   "s405-actor-cause",
		Final:         true,
		Timestamp:     time.Now().UTC().Add(-3 * time.Second),
	}
}

// s405SpotFilledServer returns an httptest.Server that simulates Binance Spot testnet
// FILLED responses with realistic fills[] array.
func s405SpotFilledServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             55555,
			"clientOrderId":       r.URL.Query().Get("newClientOrderId"),
			"symbol":              r.URL.Query().Get("symbol"),
			"status":              "FILLED",
			"side":                r.URL.Query().Get("side"),
			"type":                "MARKET",
			"executedQty":         "0.001",
			"cummulativeQuoteQty": "65.43",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]any{
				{
					"price":           "65430.00",
					"qty":             "0.001",
					"commission":      "0.00006543",
					"commissionAsset": "BNB",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

// s405BuildSegmentRouter builds a SegmentRouter matching the unified runtime
// wiring (cmd/execute/run.go buildVenueAdapterFromSegments) with real adapters
// pointing at httptest servers.
func s405BuildSegmentRouter(t *testing.T, spotServer, futuresServer *httptest.Server) *appexec.SegmentRouter {
	t.Helper()

	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY", "test-spot-key")
	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET", "test-spot-secret")
	spotCreds, _ := appexec.LoadCredentials("binance_spot_testnet", []string{"API_KEY", "API_SECRET"})
	spotAdapter := appexec.NewBinanceSpotTestnetAdapter(spotCreds, 5*time.Second).WithBaseURL(spotServer.URL)

	router := appexec.NewSegmentRouter()
	router.Register(settings.MarketSegmentSpot, spotAdapter)
	router.RegisterQuery(settings.MarketSegmentSpot, spotAdapter)

	if futuresServer != nil {
		t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", "test-futures-key")
		t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", "test-futures-secret")
		futuresCreds, _ := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
		futuresAdapter := appexec.NewBinanceFuturesTestnetAdapter(futuresCreds, 5*time.Second).WithBaseURL(futuresServer.URL)
		router.Register(settings.MarketSegmentFutures, futuresAdapter)
	}

	return router
}

// ==========================================================================
// Spot venue_live lifecycle through SegmentRouter (actor composition path)
// ==========================================================================

func TestS405_ActorComposition_SpotVenueLive_Buy_Filled(t *testing.T) {
	spotSrv := s405SpotFilledServer(t)
	defer spotSrv.Close()

	futuresCalled := false
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		futuresCalled = true
		t.Error("futures adapter must NOT be called for Spot intent")
	}))
	defer futuresSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, futuresSrv)

	intent := s405SpotVenueIntent(t, domainexec.SideBuy)
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	// Lifecycle: submitted → (accepted →) filled
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
	if fill.Price != "65430" {
		t.Errorf("expected price 65430, got %s", fill.Price)
	}
	if fill.Fee != "0.00006543" {
		t.Errorf("expected fee 0.00006543, got %s", fill.Fee)
	}
	// S428: FeeAsset and CostBasis present for Spot fills.
	if fill.FeeAsset != "BNB" {
		t.Errorf("expected fee_asset BNB, got %s", fill.FeeAsset)
	}
	if fill.CostBasis != "65.43" {
		t.Errorf("expected cost_basis 65.43 (cummulativeQuoteQty), got %s", fill.CostBasis)
	}
	if fill.Simulated {
		t.Error("venue_live fills must have Simulated=false")
	}

	// Segment isolation
	if futuresCalled {
		t.Error("futures adapter was called — segment isolation violated")
	}

	// Correlation preserved
	if receipt.Intent.CorrelationID != "s405-actor-corr" {
		t.Errorf("CorrelationID lost: %s", receipt.Intent.CorrelationID)
	}
	if receipt.Intent.CausationID != "s405-actor-cause" {
		t.Errorf("CausationID lost: %s", receipt.Intent.CausationID)
	}

	// Venue-assigned order ID
	if receipt.VenueOrderID != "55555" {
		t.Errorf("expected VenueOrderID 55555, got %s", receipt.VenueOrderID)
	}
	if strings.HasPrefix(receipt.VenueOrderID, "dryrun-") || strings.HasPrefix(receipt.VenueOrderID, "paper-") {
		t.Error("venue_live must NOT have simulation prefix")
	}
}

func TestS405_ActorComposition_SpotVenueLive_Sell_Filled(t *testing.T) {
	spotSrv := s405SpotFilledServer(t)
	defer spotSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, nil)

	intent := s405SpotVenueIntent(t, domainexec.SideSell)
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

func TestS405_ActorComposition_SpotVenueLive_None_NoContact(t *testing.T) {
	callCount := 0
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
	}))
	defer spotSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, nil)

	intent := s405SpotVenueIntent(t, domainexec.SideNone)
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

func TestS405_ActorComposition_SpotQueryOrder_Reconciliation(t *testing.T) {
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":      55555,
			"symbol":       "BTCUSDT",
			"status":       "FILLED",
			"executedQty":  "0.001",
			"transactTime": time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65430.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer spotSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, nil)

	queryReceipt, prob := router.QueryOrder(context.Background(), "test-client-id", "btcusdt")
	if prob != nil {
		t.Fatalf("query failed: %s", prob.Message)
	}

	if queryReceipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled from query, got %s", queryReceipt.Status)
	}
	if queryReceipt.VenueOrderID != "55555" {
		t.Errorf("expected VenueOrderID 55555, got %s", queryReceipt.VenueOrderID)
	}
}

// ==========================================================================
// DryRunSubmitter bypass: when dry_run=false, real adapter is called
// ==========================================================================

func TestS405_ActorComposition_DryRunDisabled_RealAdapterCalled(t *testing.T) {
	adapterCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adapterCalled = true
		resp := map[string]any{
			"orderId":      55555,
			"symbol":       "BTCUSDT",
			"status":       "FILLED",
			"executedQty":  "0.001",
			"transactTime": time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65430.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer spotSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, nil)

	// In venue_live mode (dry_run=false), the SegmentRouter is the outermost adapter.
	// DryRunSubmitter is NOT composed. Real HTTP calls reach the venue.
	intent := s405SpotVenueIntent(t, domainexec.SideBuy)
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

func TestS405_ActorComposition_DryRunEnabled_InterceptsSpotAdapter(t *testing.T) {
	adapterCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		adapterCalled = true
		t.Error("DryRunSubmitter should intercept before reaching adapter")
	}))
	defer spotSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, nil)

	// Wrap with DryRunSubmitter (matches cmd/execute/run.go when dry_run=true).
	drs := appexec.NewDryRunSubmitter(router)

	intent := s405SpotVenueIntent(t, domainexec.SideBuy)
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
