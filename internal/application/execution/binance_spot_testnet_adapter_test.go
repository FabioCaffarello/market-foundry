package execution_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
)

func spotTestCredentials(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_KEY", "test-spot-api-key")
	t.Setenv("MF_VENUE_BINANCE_SPOT_TESTNET_API_SECRET", "test-spot-api-secret")
	creds, prob := appexec.LoadCredentials("binance_spot_testnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("load test credentials: %s", prob.Message)
	}
	return creds
}

func testSpotBuyIntent(t *testing.T) domainexec.ExecutionIntent {
	t.Helper()
	return domainexec.ExecutionIntent{
		Type:       "paper_order",
		Source:     "binances",
		Instrument: btcUSDTSpot(t),
		Timeframe:  60,
		Side:       domainexec.SideBuy,
		Quantity:   "0.001",
		Status:     domainexec.StatusSubmitted,
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

func TestBinanceSpotAdapter_SubmitOrder_Filled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-MBX-APIKEY") != "test-spot-api-key" {
			t.Fatal("missing or wrong API key header")
		}

		q := r.URL.Query()
		if q.Get("symbol") != "BTCUSDT" {
			t.Fatalf("expected BTCUSDT, got %s", q.Get("symbol"))
		}
		if q.Get("side") != "BUY" {
			t.Fatalf("expected BUY, got %s", q.Get("side"))
		}
		if q.Get("newOrderRespType") != "FULL" {
			t.Fatalf("spot should use FULL response type, got %s", q.Get("newOrderRespType"))
		}

		resp := map[string]any{
			"orderId":             12345,
			"symbol":              "BTCUSDT",
			"status":              "FILLED",
			"side":                "BUY",
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
	defer server.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testSpotBuyIntent(t)})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if receipt.VenueOrderID != "12345" {
		t.Fatalf("expected venue order ID 12345, got %s", receipt.VenueOrderID)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 aggregated fill, got %d", len(receipt.Intent.Fills))
	}
	fill := receipt.Intent.Fills[0]
	if fill.Price != "65430" {
		t.Fatalf("expected price 65430, got %s", fill.Price)
	}
	if fill.Simulated {
		t.Fatal("fill should NOT be simulated")
	}
	// S428: Spot fills carry real commission, fee asset, and cost basis.
	if fill.Fee != "0.00006543" {
		t.Fatalf("expected fee 0.00006543 (Spot commission), got %s", fill.Fee)
	}
	if fill.FeeAsset != "BNB" {
		t.Fatalf("expected fee_asset BNB, got %s", fill.FeeAsset)
	}
	if fill.CostBasis != "65.43" {
		t.Fatalf("expected cost_basis 65.43 (cummulativeQuoteQty), got %s", fill.CostBasis)
	}
}

func TestBinanceSpotAdapter_SubmitOrder_MultiFill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             99999,
			"symbol":              "BTCUSDT",
			"status":              "FILLED",
			"side":                "BUY",
			"type":                "MARKET",
			"executedQty":         "0.003",
			"cummulativeQuoteQty": "195.90",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
				{"price": "65300.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
				{"price": "65600.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testSpotBuyIntent(t)})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 aggregated fill from 3 legs, got %d", len(receipt.Intent.Fills))
	}

	fill := receipt.Intent.Fills[0]
	// Weighted avg: (65000*0.001 + 65300*0.001 + 65600*0.001) / 0.003 = 65300
	if fill.Price != "65300" {
		t.Fatalf("expected weighted avg price 65300, got %s", fill.Price)
	}
	// Total fee: 0.0001 * 3 = 0.0003
	if fill.Fee != "0.0003" {
		t.Fatalf("expected total fee 0.0003, got %s", fill.Fee)
	}
	// S428: FeeAsset and CostBasis carried through multi-fill aggregation.
	if fill.FeeAsset != "BNB" {
		t.Fatalf("expected fee_asset BNB, got %s", fill.FeeAsset)
	}
	if fill.CostBasis != "195.90" {
		t.Fatalf("expected cost_basis 195.90 (cummulativeQuoteQty), got %s", fill.CostBasis)
	}
}

func TestBinanceSpotAdapter_SubmitOrder_NoAction(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		t.Fatal("no-action intent should not hit venue")
	}))
	defer server.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := testSpotBuyIntent(t)
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

func TestBinanceSpotAdapter_SubmitOrder_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -2015,
			"msg":  "Invalid API-key, IP, or permissions for action.",
		})
	}))
	defer server.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testSpotBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for auth failure")
	}
	if prob.Retryable {
		t.Fatal("auth errors should not be retryable")
	}
}

func TestBinanceSpotAdapter_APIPath(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		resp := map[string]any{
			"orderId":             1,
			"symbol":              "BTCUSDT",
			"status":              "FILLED",
			"executedQty":         "0.001",
			"cummulativeQuoteQty": "65.00",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testSpotBuyIntent(t)})

	if capturedPath != "/api/v3/order" {
		t.Fatalf("expected /api/v3/order, got %s", capturedPath)
	}
}

func TestBinanceSpotAdapter_FillNotSimulated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             42,
			"symbol":              "BTCUSDT",
			"status":              "FILLED",
			"executedQty":         "0.001",
			"cummulativeQuoteQty": "65.00",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testSpotBuyIntent(t)})
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

func TestBinanceSpotAdapter_ClientOrderID_InReceipt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             100,
			"symbol":              "BTCUSDT",
			"status":              "FILLED",
			"executedQty":         "0.001",
			"cummulativeQuoteQty": "65.00",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.001", "commission": "0.0001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	intent := testSpotBuyIntent(t)

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
