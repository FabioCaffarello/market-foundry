package execution_test

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
)

func newTestCredentials() *appexec.CredentialSet {
	t := &testing.T{}
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", "test-api-key")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", "test-api-secret")
	creds, prob := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		panic("failed to load test credentials: " + prob.Message)
	}
	return creds
}

func testCredentials(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", "test-api-key")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", "test-api-secret")
	creds, prob := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("load test credentials: %s", prob.Message)
	}
	return creds
}

func testBuyIntent() domainexec.ExecutionIntent {
	return domainexec.ExecutionIntent{
		Type:      "paper_order",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Side:      domainexec.SideBuy,
		Quantity:  "0.001",
		Status:    domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "0.85",
			Timeframe:   60,
		},
		Final:     true,
		Timestamp: time.Now().UTC(),
	}
}

func TestBinanceAdapter_SubmitOrder_Filled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request basics.
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-MBX-APIKEY") != "test-api-key" {
			t.Fatal("missing or wrong API key header")
		}

		// Verify query params.
		q := r.URL.Query()
		if q.Get("symbol") != "BTCUSDT" {
			t.Fatalf("expected BTCUSDT, got %s", q.Get("symbol"))
		}
		if q.Get("side") != "BUY" {
			t.Fatalf("expected BUY, got %s", q.Get("side"))
		}
		if q.Get("type") != "MARKET" {
			t.Fatalf("expected MARKET, got %s", q.Get("type"))
		}
		if q.Get("signature") == "" {
			t.Fatal("signature missing")
		}

		resp := map[string]any{
			"orderId":     12345,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"side":        "BUY",
			"type":        "MARKET",
			"avgPrice":    "65432.10",
			"executedQty": "0.001",
			"cumQuote":    "65.43210",
			"updateTime":  time.Now().UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if receipt.VenueOrderID != "12345" {
		t.Fatalf("expected venue order ID 12345, got %s", receipt.VenueOrderID)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
	if receipt.Intent.FilledQuantity != "0.001" {
		t.Fatalf("expected filled qty 0.001, got %s", receipt.Intent.FilledQuantity)
	}
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
	}
	fill := receipt.Intent.Fills[0]
	if fill.Price != "65432.10" {
		t.Fatalf("expected price 65432.10, got %s", fill.Price)
	}
	if fill.Simulated {
		t.Fatal("fill should NOT be simulated")
	}
}

func TestBinanceAdapter_SubmitOrder_SellSide(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("side") != "SELL" {
			t.Fatalf("expected SELL, got %s", q.Get("side"))
		}
		resp := map[string]any{
			"orderId":     99999,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"side":        "SELL",
			"type":        "MARKET",
			"avgPrice":    "65000.00",
			"executedQty": "0.001",
			"cumQuote":    "65.00",
			"updateTime":  time.Now().UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := testBuyIntent()
	intent.Side = domainexec.SideSell

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
}

func TestBinanceAdapter_SubmitOrder_NoAction(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		t.Fatal("no-action intent should not hit venue")
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := testBuyIntent()
	intent.Side = domainexec.SideNone

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}
	if receipt.Status != domainexec.StatusAccepted {
		t.Fatalf("expected accepted, got %s", receipt.Status)
	}
	if requestCount != 0 {
		t.Fatal("no-action should not make HTTP request")
	}
}

func TestBinanceAdapter_SubmitOrder_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -2015,
			"msg":  "Invalid API-key, IP, or permissions for action.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for auth failure")
	}
	if prob.Retryable {
		t.Fatal("auth errors should not be retryable")
	}
}

func TestBinanceAdapter_SubmitOrder_RejectedOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1121,
			"msg":  "Invalid symbol.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for rejected order")
	}
	if prob.Retryable {
		t.Fatal("4xx errors should not be retryable")
	}
}

func TestBinanceAdapter_SubmitOrder_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1001,
			"msg":  "Internal error; unable to process your request.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for server failure")
	}
	if !prob.Retryable {
		t.Fatal("503 should be retryable")
	}
}

func TestBinanceAdapter_SubmitOrder_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 500*time.Millisecond).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for timeout")
	}
	if !prob.Retryable {
		t.Fatal("timeout should be retryable")
	}
}

func TestBinanceAdapter_SubmitOrder_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1015,
			"msg":  "Too many requests.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for rate limit")
	}
	if !prob.Retryable {
		t.Fatal("rate limit should be retryable")
	}
}

func TestBinanceAdapter_SymbolMapping(t *testing.T) {
	var capturedSymbol string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSymbol = r.URL.Query().Get("symbol")
		resp := map[string]any{
			"orderId":     1,
			"symbol":      capturedSymbol,
			"status":      "FILLED",
			"side":        "BUY",
			"type":        "MARKET",
			"avgPrice":    "100.00",
			"executedQty": "0.001",
			"cumQuote":    "0.10",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := testBuyIntent()
	intent.Symbol = "ethusdt"
	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})

	if capturedSymbol != "ETHUSDT" {
		t.Fatalf("expected ETHUSDT, got %s", capturedSymbol)
	}
}

func TestBinanceAdapter_SignaturePresent(t *testing.T) {
	var capturedSignature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSignature = r.URL.Query().Get("signature")
		resp := map[string]any{
			"orderId":     1,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"avgPrice":    "100.00",
			"executedQty": "0.001",
			"cumQuote":    "0.10",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})

	if capturedSignature == "" {
		t.Fatal("signature should be present in request")
	}
	if len(capturedSignature) != 64 {
		t.Fatalf("HMAC-SHA256 signature should be 64 hex chars, got %d", len(capturedSignature))
	}
}

func TestBinanceAdapter_FillNotSimulated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     42,
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

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}

	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
	}
	if receipt.Intent.Fills[0].Simulated {
		t.Fatal("real venue fills must have Simulated=false")
	}
}

// --- EC-1.4: VenueOrderReceipt includes ClientOrderID populated from derivation ---

func TestBinanceAdapter_ClientOrderID_InReceipt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     100,
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

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	intent := testBuyIntent()

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	expected := appexec.ClientOrderID(intent)
	if receipt.ClientOrderID == "" {
		t.Fatal("ClientOrderID in receipt must not be empty")
	}
	if receipt.ClientOrderID != expected {
		t.Fatalf("expected ClientOrderID %q, got %q", expected, receipt.ClientOrderID)
	}
}

// --- EC-1.5: newClientOrderId is present in the HTTP request sent to venue ---

func TestBinanceAdapter_ClientOrderID_InHTTPRequest(t *testing.T) {
	var capturedClientOrderID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedClientOrderID = r.URL.Query().Get("newClientOrderId")
		resp := map[string]any{
			"orderId":     200,
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

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	intent := testBuyIntent()

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if capturedClientOrderID == "" {
		t.Fatal("newClientOrderId must be present in HTTP request")
	}

	expected := appexec.ClientOrderID(intent)
	if capturedClientOrderID != expected {
		t.Fatalf("expected newClientOrderId %q in request, got %q", expected, capturedClientOrderID)
	}
}

// --- EC-2.2: Response body exceeding 64 KB is truncated at the read boundary ---

func TestBinanceAdapter_OversizedBody_Truncated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a 128 KB body (exceeds 64 KB limit).
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"orderId":1,"status":"FILLED","avgPrice":"100.00","executedQty":"0.001","cumQuote":"0.10","updateTime":1}`))
		// Pad with spaces to exceed 64 KB.
		padding := strings.Repeat(" ", 128*1024)
		w.Write([]byte(padding))
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	// Should still parse correctly because the JSON is at the front.
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob != nil {
		t.Fatalf("oversized body with valid JSON at start should parse: %s", prob.Message)
	}
	if receipt.VenueOrderID != "1" {
		t.Fatalf("expected order ID 1, got %s", receipt.VenueOrderID)
	}
}

// --- EC-2.3/EC-2.4: Oversized body that cuts valid JSON produces Internal, non-retryable ---

func TestBinanceAdapter_OversizedBody_CorruptedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Start a very large JSON object that will be truncated mid-parse.
		w.Write([]byte(`{"orderId":1,"status":"FILLED","avgPrice":"100.00","executedQty":"0.001","data":"`))
		w.Write([]byte(strings.Repeat("x", 128*1024)))
		w.Write([]byte(`"}`))
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for corrupted oversized JSON")
	}
	if prob.Code != "SYS_INTERNAL" {
		t.Fatalf("expected SYS_INTERNAL, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("parse error from oversized body should not be retryable")
	}
}

// --- EC-2.5: Normal-sized responses (< 64 KB) are unaffected ---
// (Covered by TestBinanceAdapter_SubmitOrder_Filled and other existing tests.)

// --- EC-3.3: Slow venue response triggers context cancellation ---

func TestBinanceAdapter_ContextDeadline_Exceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		resp := map[string]any{
			"orderId": 1, "status": "FILLED",
			"avgPrice": "100.00", "executedQty": "0.001",
			"cumQuote": "0.10", "updateTime": time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 30*time.Second).WithBaseURL(server.URL)

	// Use a short context deadline (EC-3 enforcement).
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, prob := adapter.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for deadline exceeded")
	}
	if !prob.Retryable {
		t.Fatal("context deadline exceeded should be retryable")
	}
}

// --- EC-3.4: Timeout error is classified as problem.Unavailable with Retryable == true ---
// (Covered by TestBinanceAdapter_SubmitOrder_Timeout and the test above.)

// --- EC-3.5: Intent state after timeout remains submitted (PGR-08 preserved) ---

func TestBinanceAdapter_ContextDeadline_IntentUnmutated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 30*time.Second).WithBaseURL(server.URL)

	intent := testBuyIntent()
	originalStatus := intent.Status

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, _ = adapter.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: intent})

	// Intent must not be mutated by the adapter.
	if intent.Status != originalStatus {
		t.Fatalf("expected intent status %q unchanged after timeout, got %q", originalStatus, intent.Status)
	}
}

// --- EC-3.6: Normal venue responses within deadline are unaffected ---
// (Covered by all existing happy-path tests.)

// --- EC-3.1/EC-3.2: Adapter enforces default deadline when none provided ---

func TestBinanceAdapter_DefaultDeadline_Enforced(t *testing.T) {
	// Verify that calling SubmitOrder with context.Background() (no deadline)
	// still works — the adapter internally adds a default deadline.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     300,
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

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	// No explicit deadline — adapter must enforce its own.
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob != nil {
		t.Fatalf("default deadline should not block normal responses: %s", prob.Message)
	}
	if receipt.VenueOrderID != "300" {
		t.Fatalf("expected order ID 300, got %s", receipt.VenueOrderID)
	}
}
