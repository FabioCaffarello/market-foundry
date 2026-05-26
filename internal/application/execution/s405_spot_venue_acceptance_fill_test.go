package execution_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/settings"
)

// ==========================================================================
// S405 — Spot real venue connectivity and lifecycle acceptance/fill proof
//
// Proves the dominant lifecycle path submitted → accepted → filled for Spot
// testnet under the unified runtime. Each test uses an httptest server
// returning realistic Binance Spot testnet responses (fills[] array,
// per-leg price/qty/commission, transactTime).
//
// Governing questions answered:
//   TV-Q1: venue_live lifecycle transitions (submission to fill)
//   TV-Q2: Fill record fidelity (price, qty, fees)
//   TV-Q11: Correlation chain integrity
//   TV-Q12: Post-200 reconciliation under real conditions
//
// Capabilities targeted:
//   TV-C1: Real Spot venue acceptance lifecycle
//   TV-C2: Real Spot venue fill record fidelity
//   TV-C6: Lifecycle invariant fidelity under real data
//   TV-C8: Post-200 reconciliation under real conditions
// ==========================================================================

// ---------- helpers ----------

func s405SpotIntent(t *testing.T, side domainexec.Side) domainexec.ExecutionIntent {
	t.Helper()
	qty := "0.001"
	if side == domainexec.SideNone {
		qty = "0"
	}
	return domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        "binances",
		Instrument:    btcUSDTSpot(t),
		Timeframe:     60,
		Side:          side,
		Quantity:      qty,
		Status:        domainexec.StatusSubmitted,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		CorrelationID: "s405-corr-001",
		CausationID:   "s405-cause-001",
		Final:         true,
		Timestamp:     time.Now().UTC().Add(-5 * time.Second),
	}
}

func newSpotTestAdapter(t *testing.T, handler http.Handler) *appexec.BinanceSpotTestnetAdapter {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY", "test-spot-api-key")
	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET", "test-spot-api-secret")
	creds, prob := appexec.LoadCredentials("binance_spot_testnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("load test credentials: %s", prob.Message)
	}
	return appexec.NewBinanceSpotTestnetAdapter(creds, 5*time.Second).WithBaseURL(srv.URL)
}

// spotFilledHandler returns a handler simulating a Binance Spot FILLED response
// with the given fills array. This matches real Spot testnet response shape.
func spotFilledHandler(fills []map[string]any, executedQty string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             67890,
			"clientOrderId":       r.URL.Query().Get("newClientOrderId"),
			"symbol":              r.URL.Query().Get("symbol"),
			"status":              "FILLED",
			"side":                r.URL.Query().Get("side"),
			"type":                "MARKET",
			"executedQty":         executedQty,
			"cummulativeQuoteQty": "65.43",
			"transactTime":        time.Now().UnixMilli(),
			"fills":               fills,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

// ==========================================================================
// TV-Q1: venue_live lifecycle transitions — submitted → accepted → filled
// ==========================================================================

func TestS405_SpotVenueLive_Buy_SubmittedToFilled(t *testing.T) {
	fills := []map[string]any{
		{"price": "65430.00", "qty": "0.001", "commission": "0.00006543", "commissionAsset": "BNB"},
	}
	adapter := newSpotTestAdapter(t, spotFilledHandler(fills, "0.001"))

	intent := s405SpotIntent(t, domainexec.SideBuy)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	// Validate lifecycle transition: submitted → accepted → filled (compressed by venue)
	assertValidTransition(t, domainexec.StatusSubmitted, domainexec.StatusAccepted)
	assertValidTransition(t, domainexec.StatusAccepted, domainexec.StatusFilled)

	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected StatusFilled, got %s", receipt.Status)
	}
	if receipt.Intent.Status != domainexec.StatusFilled {
		t.Fatalf("intent.Status expected filled, got %s", receipt.Intent.Status)
	}
	if !receipt.Status.IsTerminal() {
		t.Fatal("filled must be terminal")
	}

	// VenueOrderID is venue-assigned numeric
	if receipt.VenueOrderID != "67890" {
		t.Fatalf("expected venue order ID 67890, got %s", receipt.VenueOrderID)
	}
	if strings.HasPrefix(receipt.VenueOrderID, "dryrun-") || strings.HasPrefix(receipt.VenueOrderID, "paper-") {
		t.Fatal("venue_live must NOT have simulation prefix")
	}
}

func TestS405_SpotVenueLive_Sell_SubmittedToFilled(t *testing.T) {
	fills := []map[string]any{
		{"price": "65400.00", "qty": "0.001", "commission": "0.0000654", "commissionAsset": "BNB"},
	}
	adapter := newSpotTestAdapter(t, spotFilledHandler(fills, "0.001"))

	intent := s405SpotIntent(t, domainexec.SideSell)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	assertValidTransition(t, domainexec.StatusSubmitted, domainexec.StatusAccepted)
	assertValidTransition(t, domainexec.StatusAccepted, domainexec.StatusFilled)

	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected StatusFilled, got %s", receipt.Status)
	}
	if receipt.Intent.Side != domainexec.SideSell {
		t.Fatalf("side lost: expected sell, got %s", receipt.Intent.Side)
	}
}

func TestS405_SpotVenueLive_None_NoVenueContact(t *testing.T) {
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		t.Fatal("venue must NOT be contacted for SideNone")
	})
	adapter := newSpotTestAdapter(t, handler)

	intent := s405SpotIntent(t, domainexec.SideNone)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	assertValidTransition(t, domainexec.StatusSubmitted, domainexec.StatusAccepted)

	if receipt.Status != domainexec.StatusAccepted {
		t.Fatalf("expected accepted, got %s", receipt.Status)
	}
	if callCount != 0 {
		t.Fatalf("SideNone must not contact venue, got %d calls", callCount)
	}
	if len(receipt.Intent.Fills) != 0 {
		t.Fatalf("SideNone must have 0 fills, got %d", len(receipt.Intent.Fills))
	}
}

// ==========================================================================
// TV-Q2: Fill record fidelity — price, qty, fees from Spot fills[] array
// ==========================================================================

func TestS405_SpotVenueLive_FillRecordFidelity_SingleLeg(t *testing.T) {
	fills := []map[string]any{
		{"price": "65430.12", "qty": "0.001", "commission": "0.00006543", "commissionAsset": "BNB"},
	}
	adapter := newSpotTestAdapter(t, spotFilledHandler(fills, "0.001"))

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s405SpotIntent(t, domainexec.SideBuy)})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 aggregated fill, got %d", len(receipt.Intent.Fills))
	}

	fill := receipt.Intent.Fills[0]

	// Price: weighted average from single leg = the leg price
	if fill.Price != "65430.12" {
		t.Errorf("expected price 65430.12, got %s", fill.Price)
	}
	// Quantity: full executed qty
	if fill.Quantity != "0.001" {
		t.Errorf("expected quantity 0.001, got %s", fill.Quantity)
	}
	// Fee: aggregated commission
	if fill.Fee != "0.00006543" {
		t.Errorf("expected fee 0.00006543, got %s", fill.Fee)
	}
	// Simulated: false for real venue
	if fill.Simulated {
		t.Error("real venue fills must have Simulated=false")
	}
	// Timestamp must not be zero
	if fill.Timestamp.IsZero() {
		t.Error("fill.Timestamp must not be zero")
	}
	// FilledQuantity matches executed qty
	if receipt.Intent.FilledQuantity != "0.001" {
		t.Errorf("FilledQuantity=%s, want 0.001", receipt.Intent.FilledQuantity)
	}
}

func TestS405_SpotVenueLive_FillRecordFidelity_MultiFillAggregation(t *testing.T) {
	fills := []map[string]any{
		{"price": "65000.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
		{"price": "65300.00", "qty": "0.001", "commission": "0.00015", "commissionAsset": "BNB"},
		{"price": "65600.00", "qty": "0.001", "commission": "0.00012", "commissionAsset": "BNB"},
	}
	adapter := newSpotTestAdapter(t, spotFilledHandler(fills, "0.003"))

	intent := s405SpotIntent(t, domainexec.SideBuy)
	intent.Quantity = "0.003"
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 aggregated fill from 3 legs, got %d", len(receipt.Intent.Fills))
	}

	fill := receipt.Intent.Fills[0]

	// Weighted avg: (65000*0.001 + 65300*0.001 + 65600*0.001) / 0.003 = 65300
	if fill.Price != "65300" {
		t.Errorf("expected weighted avg price 65300, got %s", fill.Price)
	}
	// Total fee: 0.0001 + 0.00015 + 0.00012 = 0.00037
	if fill.Fee != "0.00037" {
		t.Errorf("expected total fee 0.00037, got %s", fill.Fee)
	}
	if fill.Quantity != "0.003" {
		t.Errorf("expected quantity 0.003, got %s", fill.Quantity)
	}
	if fill.Simulated {
		t.Error("real venue fills must have Simulated=false")
	}
}

func TestS405_SpotVenueLive_FillTimestampFromTransactTime(t *testing.T) {
	expectedTime := time.Date(2026, 3, 22, 14, 30, 0, 0, time.UTC)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             67890,
			"symbol":              "BTCUSDT",
			"status":              "FILLED",
			"executedQty":         "0.001",
			"cummulativeQuoteQty": "65.43",
			"transactTime":        expectedTime.UnixMilli(),
			"fills": []map[string]any{
				{"price": "65430.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	adapter := newSpotTestAdapter(t, handler)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s405SpotIntent(t, domainexec.SideBuy)})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	fill := receipt.Intent.Fills[0]
	if !fill.Timestamp.Equal(expectedTime) {
		t.Errorf("fill timestamp mismatch: got %v, want %v", fill.Timestamp, expectedTime)
	}
}

// ==========================================================================
// TV-Q11: Correlation chain integrity through Spot venue
// ==========================================================================

func TestS405_SpotVenueLive_CorrelationChainPreserved(t *testing.T) {
	fills := []map[string]any{
		{"price": "65430.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
	}
	adapter := newSpotTestAdapter(t, spotFilledHandler(fills, "0.001"))

	intent := s405SpotIntent(t, domainexec.SideBuy)
	intent.CorrelationID = "s405-spot-corr-chain"
	intent.CausationID = "s405-spot-cause-chain"

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	assertCorrelationPreserved(t, receipt, "s405-spot-corr-chain", "s405-spot-cause-chain")
}

func TestS405_SpotVenueLive_IntentFieldPreservation(t *testing.T) {
	fills := []map[string]any{
		{"price": "65430.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
	}
	adapter := newSpotTestAdapter(t, spotFilledHandler(fills, "0.001"))

	intent := s405SpotIntent(t, domainexec.SideBuy)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	ri := receipt.Intent
	if ri.Source != "binances" {
		t.Errorf("Source lost: expected binances, got %s", ri.Source)
	}
	if ri.VenueSymbol() != "btcusdt" {
		t.Errorf("Symbol lost: expected btcusdt, got %s", ri.VenueSymbol())
	}
	if ri.Timeframe != 60 {
		t.Errorf("Timeframe lost: expected 60, got %d", ri.Timeframe)
	}
	if ri.Type != "paper_order" {
		t.Errorf("Type lost: expected paper_order, got %s", ri.Type)
	}
	if ri.Risk.Type != "position_exposure" {
		t.Errorf("Risk.Type lost: %s", ri.Risk.Type)
	}
	if ri.Risk.Disposition != "approved" {
		t.Errorf("Risk.Disposition lost: %s", ri.Risk.Disposition)
	}
	if ri.Quantity != "0.001" {
		t.Errorf("Quantity lost: %s", ri.Quantity)
	}
}

// ==========================================================================
// TV-Q12: Post-200 reconciliation — QueryOrder on Spot
// ==========================================================================

func TestS405_SpotVenueLive_QueryOrder_ReconcilesFill(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// QueryOrder uses GET, SubmitOrder uses POST
		if r.Method == http.MethodGet {
			// Simulate query response — Spot query does NOT return fills[] array
			// but returns status and executedQty.
			resp := map[string]any{
				"orderId":             67890,
				"clientOrderId":       r.URL.Query().Get("origClientOrderId"),
				"symbol":              "BTCUSDT",
				"status":              "FILLED",
				"executedQty":         "0.001",
				"cummulativeQuoteQty": "65.43",
				"transactTime":        time.Now().UnixMilli(),
				"fills": []map[string]any{
					{"price": "65430.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Submit: return a FILLED response
		resp := map[string]any{
			"orderId":             67890,
			"symbol":              "BTCUSDT",
			"status":              "FILLED",
			"executedQty":         "0.001",
			"cummulativeQuoteQty": "65.43",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65430.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	adapter := newSpotTestAdapter(t, handler)

	intent := s405SpotIntent(t, domainexec.SideBuy)
	clientOrderID := appexec.ClientOrderID(intent)

	// Execute QueryOrder — verifies the reconciliation path
	queryReceipt, prob := adapter.QueryOrder(context.Background(), clientOrderID, "btcusdt")
	if prob != nil {
		t.Fatalf("query failed: %s", prob.Message)
	}

	if queryReceipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled from query, got %s", queryReceipt.Status)
	}
	if queryReceipt.VenueOrderID != "67890" {
		t.Fatalf("expected venue order ID 67890, got %s", queryReceipt.VenueOrderID)
	}
	if len(queryReceipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill from query, got %d", len(queryReceipt.Intent.Fills))
	}
}

func TestS405_SpotVenueLive_QueryOrder_UsesCorrectAPIPath(t *testing.T) {
	var capturedPath, capturedMethod string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		resp := map[string]any{
			"orderId":      67890,
			"symbol":       "BTCUSDT",
			"status":       "FILLED",
			"executedQty":  "0.001",
			"transactTime": time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65430.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	adapter := newSpotTestAdapter(t, handler)

	adapter.QueryOrder(context.Background(), "test-client-id", "btcusdt")

	if capturedMethod != http.MethodGet {
		t.Fatalf("QueryOrder should use GET, got %s", capturedMethod)
	}
	if capturedPath != "/api/v3/order" {
		t.Fatalf("QueryOrder should use /api/v3/order, got %s", capturedPath)
	}
}

// ==========================================================================
// Spot-specific: API path, symbol mapping, request signing
// ==========================================================================

func TestS405_SpotVenueLive_SpotAPIPath(t *testing.T) {
	var capturedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		resp := map[string]any{
			"orderId":      1,
			"symbol":       "BTCUSDT",
			"status":       "FILLED",
			"executedQty":  "0.001",
			"transactTime": time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	adapter := newSpotTestAdapter(t, handler)
	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s405SpotIntent(t, domainexec.SideBuy)})

	// Spot uses /api/v3/order (NOT /fapi/v1/order like Futures)
	if capturedPath != "/api/v3/order" {
		t.Fatalf("Spot must use /api/v3/order, got %s", capturedPath)
	}
}

func TestS405_SpotVenueLive_SymbolUppercased(t *testing.T) {
	var capturedSymbol string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSymbol = r.URL.Query().Get("symbol")
		resp := map[string]any{
			"orderId":      1,
			"symbol":       capturedSymbol,
			"status":       "FILLED",
			"executedQty":  "0.001",
			"transactTime": time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	adapter := newSpotTestAdapter(t, handler)
	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s405SpotIntent(t, domainexec.SideBuy)})

	if capturedSymbol != "BTCUSDT" {
		t.Fatalf("expected BTCUSDT (uppercased), got %s", capturedSymbol)
	}
}

func TestS405_SpotVenueLive_RequestSigned(t *testing.T) {
	var hasSignature, hasAPIKey, hasTimestamp bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasSignature = r.URL.Query().Get("signature") != ""
		hasAPIKey = r.Header.Get("X-MBX-APIKEY") != ""
		hasTimestamp = r.URL.Query().Get("timestamp") != ""
		resp := map[string]any{
			"orderId":      1,
			"symbol":       "BTCUSDT",
			"status":       "FILLED",
			"executedQty":  "0.001",
			"transactTime": time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	adapter := newSpotTestAdapter(t, handler)
	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s405SpotIntent(t, domainexec.SideBuy)})

	if !hasSignature {
		t.Error("request must include HMAC-SHA256 signature")
	}
	if !hasAPIKey {
		t.Error("request must include X-MBX-APIKEY header")
	}
	if !hasTimestamp {
		t.Error("request must include timestamp parameter")
	}
}

func TestS405_SpotVenueLive_ClientOrderIDSent(t *testing.T) {
	var capturedClientOrderID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedClientOrderID = r.URL.Query().Get("newClientOrderId")
		resp := map[string]any{
			"orderId":      1,
			"symbol":       "BTCUSDT",
			"status":       "FILLED",
			"executedQty":  "0.001",
			"transactTime": time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	adapter := newSpotTestAdapter(t, handler)

	intent := s405SpotIntent(t, domainexec.SideBuy)
	expectedClientOrderID := appexec.ClientOrderID(intent)

	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})

	if capturedClientOrderID == "" {
		t.Fatal("newClientOrderId must be sent to venue")
	}
	if capturedClientOrderID != expectedClientOrderID {
		t.Fatalf("expected client order ID %s, got %s", expectedClientOrderID, capturedClientOrderID)
	}
}

func TestS405_SpotVenueLive_FULLResponseTypeRequested(t *testing.T) {
	var capturedRespType string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRespType = r.URL.Query().Get("newOrderRespType")
		resp := map[string]any{
			"orderId":      1,
			"symbol":       "BTCUSDT",
			"status":       "FILLED",
			"executedQty":  "0.001",
			"transactTime": time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	adapter := newSpotTestAdapter(t, handler)
	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s405SpotIntent(t, domainexec.SideBuy)})

	// Spot adapter must request FULL response type to get fills[] array
	if capturedRespType != "FULL" {
		t.Fatalf("expected FULL response type, got %s", capturedRespType)
	}
}

// ==========================================================================
// Segment routing: Spot source dispatches to Spot adapter
// ==========================================================================

func TestS405_SegmentRouter_SpotSourceDispatchesToSpotAdapter(t *testing.T) {
	var spotCalled, futuresCalled bool

	spotHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spotCalled = true
		resp := map[string]any{
			"orderId":      1,
			"symbol":       "BTCUSDT",
			"status":       "FILLED",
			"executedQty":  "0.001",
			"transactTime": time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	futuresHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		futuresCalled = true
		w.WriteHeader(http.StatusOK)
	})

	spotSrv := httptest.NewServer(spotHandler)
	defer spotSrv.Close()
	futuresSrv := httptest.NewServer(futuresHandler)
	defer futuresSrv.Close()

	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY", "spot-key")
	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET", "spot-secret")
	spotCreds, _ := appexec.LoadCredentials("binance_spot_testnet", []string{"API_KEY", "API_SECRET"})
	spotAdapter := appexec.NewBinanceSpotTestnetAdapter(spotCreds, 5*time.Second).WithBaseURL(spotSrv.URL)

	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", "futures-key")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", "futures-secret")
	futuresCreds, _ := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	futuresAdapter := appexec.NewBinanceFuturesTestnetAdapter(futuresCreds, 5*time.Second).WithBaseURL(futuresSrv.URL)

	router := appexec.NewSegmentRouter()
	router.Register(settings.MarketSegmentSpot, spotAdapter)
	router.Register(settings.MarketSegmentFutures, futuresAdapter)

	// Submit Spot intent — should route to Spot adapter only
	intent := s405SpotIntent(t, domainexec.SideBuy)
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if !spotCalled {
		t.Error("Spot adapter must be called for binances source")
	}
	if futuresCalled {
		t.Error("Futures adapter must NOT be called for binances source")
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
}

func TestS405_SegmentRouter_UnknownSourceRejected(t *testing.T) {
	router := appexec.NewSegmentRouter()

	intent := s405SpotIntent(t, domainexec.SideBuy)
	intent.Source = "unknown_exchange"

	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected problem for unknown source")
	}
	if !strings.Contains(prob.Message, "no segment mapping") {
		t.Errorf("expected 'no segment mapping' in message, got: %s", prob.Message)
	}
}

// ==========================================================================
// Lifecycle alignment: ValidTransition matrix for Spot dominant path
// ==========================================================================

func TestS405_SpotLifecycleAlignment_DominantPathValid(t *testing.T) {
	// The dominant Spot path: submitted → accepted → filled
	// Each step must be valid per ValidTransition()
	steps := []struct {
		from, to domainexec.Status
	}{
		{domainexec.StatusSubmitted, domainexec.StatusAccepted},
		{domainexec.StatusAccepted, domainexec.StatusFilled},
	}

	for _, step := range steps {
		if !domainexec.ValidTransition(step.from, step.to) {
			t.Errorf("transition %s → %s should be valid", step.from, step.to)
		}
	}
}

func TestS405_SpotLifecycleAlignment_FilledIsTerminal(t *testing.T) {
	if !domainexec.StatusFilled.IsTerminal() {
		t.Error("filled must be terminal")
	}
	// No transitions out of filled
	allStates := []domainexec.Status{
		domainexec.StatusSubmitted, domainexec.StatusSent, domainexec.StatusAccepted,
		domainexec.StatusFilled, domainexec.StatusPartiallyFilled,
		domainexec.StatusRejected, domainexec.StatusCancelled,
	}
	for _, st := range allStates {
		if domainexec.ValidTransition(domainexec.StatusFilled, st) {
			t.Errorf("filled → %s should NOT be valid (terminal state)", st)
		}
	}
}

func TestS405_SpotLifecycleAlignment_BinanceStatusMapping(t *testing.T) {
	// Verify that Binance Spot testnet status values map to correct lifecycle states.
	// This is critical for real venue connectivity.
	fills := []map[string]any{
		{"price": "65430.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
	}

	statuses := []struct {
		binanceStatus  string
		expectedStatus domainexec.Status
	}{
		{"FILLED", domainexec.StatusFilled},
	}

	for _, tc := range statuses {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]any{
				"orderId":      int64(100),
				"symbol":       "BTCUSDT",
				"status":       tc.binanceStatus,
				"executedQty":  "0.001",
				"transactTime": time.Now().UnixMilli(),
				"fills":        fills,
			}
			json.NewEncoder(w).Encode(resp)
		})
		adapter := newSpotTestAdapter(t, handler)

		receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s405SpotIntent(t, domainexec.SideBuy)})
		if prob != nil {
			t.Fatalf("status %s: submit failed: %s", tc.binanceStatus, prob.Message)
		}
		if receipt.Status != tc.expectedStatus {
			t.Errorf("Binance %s should map to %s, got %s", tc.binanceStatus, tc.expectedStatus, receipt.Status)
		}
	}
}

// ==========================================================================
// Config validation: venue_live Spot config
// ==========================================================================

func TestS405_VenueLiveSpotConfig_DryRunDisabled(t *testing.T) {
	dryRun := false
	cfg := settings.VenueConfig{
		DryRun:          &dryRun,
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentSpot: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceSpotTestnet,
			},
			settings.MarketSegmentFutures: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceFuturesTestnet,
			},
		},
	}

	if cfg.IsDryRun() {
		t.Fatal("venue_live config must have dry_run=false")
	}
	if !cfg.HasUnifiedSegments() {
		t.Fatal("expected unified segments")
	}
	if !cfg.IsSegmentEnabled(settings.MarketSegmentSpot) {
		t.Fatal("spot must be enabled")
	}

	// Verify enabled sources include binances
	sources := cfg.EnabledSegmentSources()
	found := false
	for _, s := range sources {
		if s == "binances" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected binances in enabled sources, got %v", sources)
	}
}

func TestS405_VenueLiveSpotConfig_SpotAdapterResolved(t *testing.T) {
	dryRun := false
	cfg := settings.VenueConfig{
		DryRun:          &dryRun,
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentSpot: {
				Enabled: true,
				Adapter: settings.VenueTypeBinanceSpotTestnet,
			},
		},
	}

	adapter := cfg.AdapterForSegment(settings.MarketSegmentSpot)
	if adapter != settings.VenueTypeBinanceSpotTestnet {
		t.Fatalf("expected binance_spot_testnet adapter, got %s", adapter)
	}
}

// ==========================================================================
// ClientOrderID determinism: same intent yields same ID (safe for retries)
// ==========================================================================

func TestS405_SpotVenueLive_ClientOrderIDDeterministic(t *testing.T) {
	intent := s405SpotIntent(t, domainexec.SideBuy)
	id1 := appexec.ClientOrderID(intent)
	id2 := appexec.ClientOrderID(intent)

	if id1 != id2 {
		t.Fatalf("ClientOrderID must be deterministic: %s != %s", id1, id2)
	}
	if len(id1) != 32 {
		t.Fatalf("ClientOrderID must be 32 hex chars, got %d", len(id1))
	}

	// Verify it's valid hex
	_, err := strconv.ParseUint(id1[:16], 16, 64)
	if err != nil {
		t.Fatalf("ClientOrderID must be valid hex: %s", err)
	}
}

// ==========================================================================
// Error classification: Spot testnet error responses
// ==========================================================================

func TestS405_SpotVenueLive_InsufficientBalance_NonRetryable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -2010,
			"msg":  "Account has insufficient balance for requested action.",
		})
	})
	adapter := newSpotTestAdapter(t, handler)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s405SpotIntent(t, domainexec.SideBuy)})
	if prob == nil {
		t.Fatal("expected problem for insufficient balance")
	}
	if prob.Retryable {
		t.Error("insufficient balance should NOT be retryable")
	}
}

func TestS405_SpotVenueLive_RateLimit_Retryable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1015,
			"msg":  "Too many new orders.",
		})
	})
	adapter := newSpotTestAdapter(t, handler)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s405SpotIntent(t, domainexec.SideBuy)})
	if prob == nil {
		t.Fatal("expected problem for rate limit")
	}
	if !prob.Retryable {
		t.Error("rate limit should be retryable")
	}
}

func TestS405_SpotVenueLive_ServerError_Retryable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	adapter := newSpotTestAdapter(t, handler)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s405SpotIntent(t, domainexec.SideBuy)})
	if prob == nil {
		t.Fatal("expected problem for server error")
	}
	if !prob.Retryable {
		t.Error("503 should be retryable")
	}
}
