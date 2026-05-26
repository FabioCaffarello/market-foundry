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
// S416 — Futures real venue connectivity and lifecycle acceptance/fill proof
//
// Proves the dominant lifecycle path submitted → accepted → filled for Futures
// testnet under the unified runtime. Each test uses an httptest server
// returning realistic Binance Futures testnet responses (avgPrice, cumQuote,
// updateTime — no fills[] array).
//
// Governing questions answered:
//   FV-Q1: venue_live lifecycle transitions (submission to fill)
//   FV-Q2: Fill record fidelity (price, qty, fees)
//   FV-Q3: Correlation chain integrity
//   FV-Q4: Post-200 reconciliation under real conditions
//
// Capabilities targeted:
//   FV-C1: Real Futures venue acceptance lifecycle
//   FV-C2: Real Futures venue fill record fidelity
//   FV-C3: Lifecycle invariant fidelity under real data
//   FV-C4: Post-200 reconciliation under real conditions
// ==========================================================================

// ---------- helpers ----------

func s416FuturesIntent(t *testing.T, side domainexec.Side) domainexec.ExecutionIntent {
	t.Helper()
	qty := "0.001"
	if side == domainexec.SideNone {
		qty = "0"
	}
	return domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        "binancef",
		Instrument:    btcUSDTPerp(t),
		Timeframe:     60,
		Side:          side,
		Quantity:      qty,
		Status:        domainexec.StatusSubmitted,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		CorrelationID: "s416-corr-001",
		CausationID:   "s416-cause-001",
		Final:         true,
		Timestamp:     time.Now().UTC().Add(-5 * time.Second),
	}
}

func s416FuturesFilledServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     67890,
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

func s416FuturesCredentials(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", "test-futures-api-key")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", "test-futures-api-secret")
	creds, prob := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("load test credentials: %s", prob.Message)
	}
	return creds
}

// ==========================================================================
// FV-Q1: Dominant lifecycle path — submitted → accepted → filled
// ==========================================================================

func TestS416_FuturesVenueLive_Buy_SubmittedToFilled(t *testing.T) {
	server := s416FuturesFilledServer(t)
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s416FuturesIntent(t, domainexec.SideBuy)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
	if receipt.VenueOrderID != "67890" {
		t.Fatalf("expected venue order ID 67890, got %s", receipt.VenueOrderID)
	}
	if receipt.Intent.FilledQuantity != "0.001" {
		t.Fatalf("expected filled qty 0.001, got %s", receipt.Intent.FilledQuantity)
	}
}

func TestS416_FuturesVenueLive_Sell_SubmittedToFilled(t *testing.T) {
	server := s416FuturesFilledServer(t)
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s416FuturesIntent(t, domainexec.SideSell)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
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

func TestS416_FuturesVenueLive_None_NoVenueContact(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		t.Fatal("no-action intent should not hit venue")
	}))
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s416FuturesIntent(t, domainexec.SideNone)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusAccepted {
		t.Fatalf("expected accepted for SideNone, got %s", receipt.Status)
	}
	if callCount != 0 {
		t.Fatal("no-action should not make HTTP request")
	}
}

// ==========================================================================
// FV-Q2: Fill record fidelity — Futures-specific response parsing
// ==========================================================================

func TestS416_FuturesVenueLive_FillRecordFidelity(t *testing.T) {
	server := s416FuturesFilledServer(t)
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s416FuturesIntent(t, domainexec.SideBuy)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
	}

	fill := receipt.Intent.Fills[0]
	if fill.Price != "65432.10" {
		t.Fatalf("expected price 65432.10 (from avgPrice), got %s", fill.Price)
	}
	if fill.Quantity != "0.001" {
		t.Fatalf("expected qty 0.001, got %s", fill.Quantity)
	}
	if fill.Fee != "0" {
		t.Fatalf("expected fee 0 (Futures RESULT has no commission), got %s", fill.Fee)
	}
	if fill.CostBasis != "65.43210" {
		t.Fatalf("expected CostBasis 65.43210 (cumQuote), got %s", fill.CostBasis)
	}
	if fill.Simulated {
		t.Fatal("real venue fills must have Simulated=false")
	}
	if fill.Timestamp.IsZero() {
		t.Fatal("fill timestamp must not be zero")
	}
}

func TestS416_FuturesVenueLive_FillTimestampFromUpdateTime(t *testing.T) {
	knownTime := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     11111,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"avgPrice":    "65000.00",
			"executedQty": "0.001",
			"cumQuote":    "65.00",
			"updateTime":  knownTime.UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s416FuturesIntent(t, domainexec.SideBuy)})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	fill := receipt.Intent.Fills[0]
	if !fill.Timestamp.Equal(knownTime) {
		t.Fatalf("expected fill timestamp %v, got %v", knownTime, fill.Timestamp)
	}
}

// ==========================================================================
// FV-Q3: Correlation chain integrity
// ==========================================================================

func TestS416_FuturesVenueLive_CorrelationChainPreserved(t *testing.T) {
	server := s416FuturesFilledServer(t)
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s416FuturesIntent(t, domainexec.SideBuy)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if receipt.Intent.CorrelationID != "s416-corr-001" {
		t.Errorf("CorrelationID lost: expected s416-corr-001, got %s", receipt.Intent.CorrelationID)
	}
	if receipt.Intent.CausationID != "s416-cause-001" {
		t.Errorf("CausationID lost: expected s416-cause-001, got %s", receipt.Intent.CausationID)
	}
}

func TestS416_FuturesVenueLive_IntentFieldPreservation(t *testing.T) {
	server := s416FuturesFilledServer(t)
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s416FuturesIntent(t, domainexec.SideBuy)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	ri := receipt.Intent
	if ri.Type != "paper_order" {
		t.Errorf("Type lost: %s", ri.Type)
	}
	if ri.Source != "binancef" {
		t.Errorf("Source lost: %s", ri.Source)
	}
	if ri.VenueSymbol() != "btcusdt" {
		t.Errorf("Symbol lost: %s", ri.VenueSymbol())
	}
	if ri.Timeframe != 60 {
		t.Errorf("Timeframe lost: %d", ri.Timeframe)
	}
	if ri.Risk.Type != "position_exposure" {
		t.Errorf("Risk.Type lost: %s", ri.Risk.Type)
	}
	if ri.Risk.Disposition != "approved" {
		t.Errorf("Risk.Disposition lost: %s", ri.Risk.Disposition)
	}
}

// ==========================================================================
// FV-Q4: Post-200 reconciliation via QueryOrder
// ==========================================================================

func TestS416_FuturesVenueLive_QueryOrder_ReconcilesFill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("QueryOrder should use GET, got %s", r.Method)
		}
		q := r.URL.Query()
		if q.Get("origClientOrderId") == "" {
			t.Fatal("origClientOrderId must be present")
		}

		resp := map[string]any{
			"orderId":     67890,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"avgPrice":    "65432.10",
			"executedQty": "0.001",
			"cumQuote":    "65.43210",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.QueryOrder(context.Background(), "test-client-id", "btcusdt")
	if prob != nil {
		t.Fatalf("query failed: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
	if receipt.VenueOrderID != "67890" {
		t.Errorf("expected venue order ID 67890, got %s", receipt.VenueOrderID)
	}
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
	}
}

func TestS416_FuturesVenueLive_QueryOrder_UsesCorrectAPIPath(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		resp := map[string]any{
			"orderId":     1,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"avgPrice":    "65000.00",
			"executedQty": "0.001",
			"cumQuote":    "65.00",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	adapter.QueryOrder(context.Background(), "test-id", "btcusdt")

	if capturedPath != "/fapi/v1/order" {
		t.Fatalf("expected /fapi/v1/order, got %s", capturedPath)
	}
}

// ==========================================================================
// Futures-specific contract validation
// ==========================================================================

func TestS416_FuturesVenueLive_FuturesAPIPath(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		resp := map[string]any{
			"orderId":     1,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"avgPrice":    "65000.00",
			"executedQty": "0.001",
			"cumQuote":    "65.00",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s416FuturesIntent(t, domainexec.SideBuy)})

	if capturedPath != "/fapi/v1/order" {
		t.Fatalf("Futures must use /fapi/v1/order, got %s", capturedPath)
	}
}

func TestS416_FuturesVenueLive_RESULTResponseType(t *testing.T) {
	var capturedRespType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRespType = r.URL.Query().Get("newOrderRespType")
		resp := map[string]any{
			"orderId":     1,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"avgPrice":    "65000.00",
			"executedQty": "0.001",
			"cumQuote":    "65.00",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s416FuturesIntent(t, domainexec.SideBuy)})

	if capturedRespType != "RESULT" {
		t.Fatalf("Futures should use RESULT response type, got %s", capturedRespType)
	}
}

func TestS416_FuturesVenueLive_SymbolUppercased(t *testing.T) {
	var capturedSymbol string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSymbol = r.URL.Query().Get("symbol")
		resp := map[string]any{
			"orderId":     1,
			"symbol":      capturedSymbol,
			"status":      "FILLED",
			"avgPrice":    "3000.00",
			"executedQty": "0.01",
			"cumQuote":    "30.00",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s416FuturesIntent(t, domainexec.SideBuy)
	intent.Instrument = ethUSDTPerp(t)
	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})

	if capturedSymbol != "ETHUSDT" {
		t.Fatalf("expected ETHUSDT, got %s", capturedSymbol)
	}
}

func TestS416_FuturesVenueLive_RequestSigned(t *testing.T) {
	var capturedSig string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSig = r.URL.Query().Get("signature")
		resp := map[string]any{
			"orderId":     1,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"avgPrice":    "65000.00",
			"executedQty": "0.001",
			"cumQuote":    "65.00",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s416FuturesIntent(t, domainexec.SideBuy)})

	if capturedSig == "" {
		t.Fatal("signature must be present in request")
	}
	if len(capturedSig) != 64 {
		t.Fatalf("HMAC-SHA256 signature should be 64 hex chars, got %d", len(capturedSig))
	}
}

func TestS416_FuturesVenueLive_ClientOrderIDSent(t *testing.T) {
	var capturedCOID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCOID = r.URL.Query().Get("newClientOrderId")
		resp := map[string]any{
			"orderId":     1,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"avgPrice":    "65000.00",
			"executedQty": "0.001",
			"cumQuote":    "65.00",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s416FuturesIntent(t, domainexec.SideBuy)
	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})

	if capturedCOID == "" {
		t.Fatal("newClientOrderId must be present in HTTP request")
	}
	expected := appexec.ClientOrderID(intent)
	if capturedCOID != expected {
		t.Fatalf("expected newClientOrderId %q, got %q", expected, capturedCOID)
	}
}

func TestS416_FuturesVenueLive_ClientOrderIDDeterministic(t *testing.T) {
	intent := s416FuturesIntent(t, domainexec.SideBuy)
	id1 := appexec.ClientOrderID(intent)
	id2 := appexec.ClientOrderID(intent)
	if id1 != id2 {
		t.Fatalf("ClientOrderID must be deterministic: %q != %q", id1, id2)
	}
}

// ==========================================================================
// Lifecycle alignment with ValidTransition() (S383)
// ==========================================================================

func TestS416_FuturesLifecycleAlignment_DominantPathValid(t *testing.T) {
	// The dominant Futures path is: submitted → accepted → filled.
	// Both transitions must be valid per the canonical lifecycle.
	if !domainexec.ValidTransition(domainexec.StatusSubmitted, domainexec.StatusAccepted) {
		t.Fatal("submitted → accepted must be a valid transition")
	}
	if !domainexec.ValidTransition(domainexec.StatusAccepted, domainexec.StatusFilled) {
		t.Fatal("accepted → filled must be a valid transition")
	}
}

func TestS416_FuturesLifecycleAlignment_FilledIsTerminal(t *testing.T) {
	if !domainexec.StatusFilled.IsTerminal() {
		t.Fatal("filled must be a terminal state")
	}
	allStatuses := []domainexec.Status{
		domainexec.StatusSubmitted, domainexec.StatusSent, domainexec.StatusAccepted,
		domainexec.StatusFilled, domainexec.StatusPartiallyFilled,
		domainexec.StatusRejected, domainexec.StatusCancelled,
	}
	for _, to := range allStatuses {
		if domainexec.ValidTransition(domainexec.StatusFilled, to) {
			t.Fatalf("filled → %s should not be valid (terminal state)", to)
		}
	}
}

func TestS416_FuturesLifecycleAlignment_BinanceStatusMapping(t *testing.T) {
	// Futures-specific: Binance returns these statuses. Verify our mapping covers them.
	cases := []struct {
		binanceStatus string
		expected      domainexec.Status
	}{
		{"NEW", domainexec.StatusAccepted},
		{"FILLED", domainexec.StatusFilled},
		{"PARTIALLY_FILLED", domainexec.StatusPartiallyFilled},
		{"CANCELED", domainexec.StatusCancelled},
		{"REJECTED", domainexec.StatusRejected},
		{"EXPIRED", domainexec.StatusRejected},
	}

	// Verify by submitting mock responses with each status.
	for _, tc := range cases {
		t.Run(tc.binanceStatus, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := map[string]any{
					"orderId":     1,
					"symbol":      "BTCUSDT",
					"status":      tc.binanceStatus,
					"avgPrice":    "65000.00",
					"executedQty": "0.001",
					"cumQuote":    "65.00",
					"updateTime":  time.Now().UnixMilli(),
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			creds := s416FuturesCredentials(t)
			adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

			receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s416FuturesIntent(t, domainexec.SideBuy)})
			if prob != nil {
				t.Fatalf("submit failed: %s", prob.Message)
			}
			if receipt.Status != tc.expected {
				t.Fatalf("expected %s for Binance status %s, got %s", tc.expected, tc.binanceStatus, receipt.Status)
			}
		})
	}
}

// ==========================================================================
// SegmentRouter: Futures-specific routing
// ==========================================================================

func TestS416_SegmentRouter_FuturesSourceDispatchesToFuturesAdapter(t *testing.T) {
	futuresCalled := false
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		futuresCalled = true
		resp := map[string]any{
			"orderId":     67890,
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

	spotCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spotCalled = true
		t.Error("Spot adapter must NOT be called for Futures intent")
	}))
	defer spotSrv.Close()

	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", "test-key")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", "test-secret")
	futuresCreds, _ := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	futuresAdapter := appexec.NewBinanceFuturesTestnetAdapter(futuresCreds, 5*time.Second).WithBaseURL(futuresSrv.URL)

	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY", "test-key")
	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET", "test-secret")
	spotCreds, _ := appexec.LoadCredentials("binance_spot_testnet", []string{"API_KEY", "API_SECRET"})
	spotAdapter := appexec.NewBinanceSpotTestnetAdapter(spotCreds, 5*time.Second).WithBaseURL(spotSrv.URL)

	router := appexec.NewSegmentRouter()
	router.Register(settings.MarketSegmentFutures, futuresAdapter)
	router.Register(settings.MarketSegmentSpot, spotAdapter)

	intent := s416FuturesIntent(t, domainexec.SideBuy)
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if !futuresCalled {
		t.Error("Futures adapter must be called for source=binancef")
	}
	if spotCalled {
		t.Error("Spot adapter must NOT be called for Futures intent — segment isolation violated")
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
}

// ==========================================================================
// Config validation: venue_live Futures config
// ==========================================================================

func TestS416_VenueLiveFuturesConfig_DryRunDisabled(t *testing.T) {
	// The venue_live Futures config must have dry_run=false.
	// This validates the config pattern, not the actual file I/O.
	cfg := settings.VenueConfig{
		DryRun:          boolPtr(false),
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentFutures: {Enabled: true, Adapter: settings.VenueTypeBinanceFuturesTestnet},
			settings.MarketSegmentSpot:    {Enabled: true, Adapter: settings.VenueTypeBinanceSpotTestnet},
		},
	}

	if cfg.IsDryRun() {
		t.Fatal("venue_live config must have dry_run=false")
	}

	enabled := cfg.EnabledSegments()
	hasFutures := false
	for _, seg := range enabled {
		if seg == settings.MarketSegmentFutures {
			hasFutures = true
		}
	}
	if !hasFutures {
		t.Fatal("Futures segment must be enabled")
	}
}

func TestS416_VenueLiveFuturesConfig_FuturesAdapterResolved(t *testing.T) {
	cfg := settings.VenueConfig{
		DryRun:          boolPtr(false),
		StalenessMaxAge: "120s",
		SubmitTimeout:   "10s",
		Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
			settings.MarketSegmentFutures: {Enabled: true, Adapter: settings.VenueTypeBinanceFuturesTestnet},
			settings.MarketSegmentSpot:    {Enabled: true, Adapter: settings.VenueTypeBinanceSpotTestnet},
		},
	}

	adapter := cfg.AdapterForSegment(settings.MarketSegmentFutures)
	if adapter != settings.VenueTypeBinanceFuturesTestnet {
		t.Fatalf("expected binance_futures_testnet, got %s", adapter)
	}
}

// ==========================================================================
// Error classification (Futures-specific)
// ==========================================================================

func TestS416_FuturesVenueLive_InsufficientMargin_NonRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -2019,
			"msg":  "Margin is insufficient.",
		})
	}))
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s416FuturesIntent(t, domainexec.SideBuy)})
	if prob == nil {
		t.Fatal("expected error for insufficient margin")
	}
	if prob.Retryable {
		t.Fatal("insufficient margin should not be retryable")
	}
}

func TestS416_FuturesVenueLive_RateLimit_Retryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1015,
			"msg":  "Too many new orders.",
		})
	}))
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s416FuturesIntent(t, domainexec.SideBuy)})
	if prob == nil {
		t.Fatal("expected error for rate limit")
	}
	if !prob.Retryable {
		t.Fatal("rate limit should be retryable")
	}
}

func TestS416_FuturesVenueLive_ServerError_Retryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1001,
			"msg":  "Internal error.",
		})
	}))
	defer server.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s416FuturesIntent(t, domainexec.SideBuy)})
	if prob == nil {
		t.Fatal("expected error for server failure")
	}
	if !prob.Retryable {
		t.Fatal("503 should be retryable")
	}
}

// ---------- helpers ----------

func boolPtr(b bool) *bool { return &b }

// s416SegmentForSource validates the segment routing table.
func TestS416_SegmentForSource_Futures(t *testing.T) {
	seg := settings.SegmentForSource("binancef")
	if seg != settings.MarketSegmentFutures {
		t.Fatalf("expected futures, got %s", seg)
	}
}

// s416SpotDiffFromFutures validates that Spot and Futures use different API paths and response types.
func TestS416_SpotFuturesDifference_APIPathAndResponseType(t *testing.T) {
	var futuresPath, spotPath string
	var futuresRespType, spotRespType string

	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		futuresPath = r.URL.Path
		futuresRespType = r.URL.Query().Get("newOrderRespType")
		resp := map[string]any{
			"orderId": 1, "symbol": "BTCUSDT", "status": "FILLED",
			"avgPrice": "65000.00", "executedQty": "0.001", "cumQuote": "65.00",
			"updateTime": time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer futuresSrv.Close()

	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spotPath = r.URL.Path
		spotRespType = r.URL.Query().Get("newOrderRespType")
		resp := map[string]any{
			"orderId": 1, "symbol": "BTCUSDT", "status": "FILLED",
			"executedQty": "0.001", "cummulativeQuoteQty": "65.00",
			"transactTime": time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer spotSrv.Close()

	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", "test-key")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", "test-secret")
	fCreds, _ := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	futuresAdapter := appexec.NewBinanceFuturesTestnetAdapter(fCreds, 5*time.Second).WithBaseURL(futuresSrv.URL)

	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY", "test-key")
	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET", "test-secret")
	sCreds, _ := appexec.LoadCredentials("binance_spot_testnet", []string{"API_KEY", "API_SECRET"})
	spotAdapter := appexec.NewBinanceSpotTestnetAdapter(sCreds, 5*time.Second).WithBaseURL(spotSrv.URL)

	futuresAdapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s416FuturesIntent(t, domainexec.SideBuy)})
	spotIntent := s416FuturesIntent(t, domainexec.SideBuy)
	spotIntent.Source = "binances"
	spotAdapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: spotIntent})

	if futuresPath != "/fapi/v1/order" {
		t.Fatalf("Futures path: expected /fapi/v1/order, got %s", futuresPath)
	}
	if spotPath != "/api/v3/order" {
		t.Fatalf("Spot path: expected /api/v3/order, got %s", spotPath)
	}
	if futuresRespType != "RESULT" {
		t.Fatalf("Futures respType: expected RESULT, got %s", futuresRespType)
	}
	if spotRespType != "FULL" {
		t.Fatalf("Spot respType: expected FULL, got %s", spotRespType)
	}
}

// Ensure unused imports don't cause build failure.
var _ = strconv.Itoa
var _ = strings.HasPrefix
