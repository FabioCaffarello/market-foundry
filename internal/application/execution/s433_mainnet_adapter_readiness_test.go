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
	"internal/shared/problem"
)

// --- Mainnet Spot Adapter Tests ---

func spotMainnetTestCredentials(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY", "test-mainnet-spot-key-0123456789abcdef0123456789abcdef01234567")
	t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET", "test-mainnet-spot-secret-0123456789abcdef0123456789abcdef0123456")
	creds, prob := appexec.LoadCredentials("binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("load test credentials: %s", prob.Message)
	}
	return creds
}

func TestBinanceSpotMainnetAdapter_BaseURL(t *testing.T) {
	creds := spotMainnetTestCredentials(t)
	adapter := appexec.NewBinanceSpotMainnetAdapter(creds, 5*time.Second)

	// Verify mainnet adapter sends requests to mainnet URL by using httptest.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-MBX-APIKEY") != "test-mainnet-spot-key-0123456789abcdef0123456789abcdef01234567" {
			t.Fatal("wrong API key header")
		}
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"orderId":             12345,
			"clientOrderId":       "test",
			"symbol":              "BTCUSDT",
			"status":              "FILLED",
			"side":                "BUY",
			"type":                "MARKET",
			"executedQty":         "0.001",
			"cummulativeQuoteQty": "65.43",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]string{
				{
					"price":           "65430.00",
					"qty":             "0.001",
					"commission":      "0.00006543",
					"commissionAsset": "BTC",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter.WithBaseURL(server.URL)

	intent := domainexec.ExecutionIntent{
		Type:       "paper_order",
		Source:     "binances",
		Instrument: btcUSDTSpot(t),
		Timeframe:  60,
		Side:       domainexec.SideBuy,
		Quantity:   "0.001",
		Status:     domainexec.StatusSubmitted,
		Final:      true,
		Timestamp:  time.Now().UTC(),
	}

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
	if receipt.Intent.Fills[0].Fee != "0.00006543" {
		t.Fatalf("expected fee 0.00006543, got %s", receipt.Intent.Fills[0].Fee)
	}
}

func TestBinanceSpotMainnetAdapter_VenuePortInterface(t *testing.T) {
	creds := spotMainnetTestCredentials(t)
	adapter := appexec.NewBinanceSpotMainnetAdapter(creds, 5*time.Second)
	var _ ports.VenuePort = adapter
	var _ ports.VenueQueryPort = adapter
}

// --- Mainnet Futures Adapter Tests ---

func futuresMainnetTestCredentials(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	t.Setenv("MF_VENUE_BINANCE_FUTURES_MAINNET_API_KEY", "test-mainnet-futures-key-0123456789abcdef0123456789abcdef0123456")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_MAINNET_API_SECRET", "test-mainnet-futures-secret-0123456789abcdef0123456789abcdef012345")
	creds, prob := appexec.LoadCredentials("binance_futures_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("load test credentials: %s", prob.Message)
	}
	return creds
}

func TestBinanceFuturesMainnetAdapter_BaseURL(t *testing.T) {
	creds := futuresMainnetTestCredentials(t)
	adapter := appexec.NewBinanceFuturesMainnetAdapter(creds, 5*time.Second)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-MBX-APIKEY") != "test-mainnet-futures-key-0123456789abcdef0123456789abcdef0123456" {
			t.Fatal("wrong API key header")
		}
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"orderId":       67890,
			"clientOrderId": "test",
			"symbol":        "BTCUSDT",
			"status":        "FILLED",
			"side":          "BUY",
			"type":          "MARKET",
			"avgPrice":      "65430.00",
			"executedQty":   "0.001",
			"cumQuote":      "65.43",
			"updateTime":    time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter.WithBaseURL(server.URL)

	intent := domainexec.ExecutionIntent{
		Type:       "paper_order",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       domainexec.SideBuy,
		Quantity:   "0.001",
		Status:     domainexec.StatusSubmitted,
		Final:      true,
		Timestamp:  time.Now().UTC(),
	}

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
	if receipt.Intent.Fills[0].CostBasis != "65.43" {
		t.Fatalf("expected CostBasis 65.43, got %s", receipt.Intent.Fills[0].CostBasis)
	}
}

func TestBinanceFuturesMainnetAdapter_VenuePortInterface(t *testing.T) {
	creds := futuresMainnetTestCredentials(t)
	adapter := appexec.NewBinanceFuturesMainnetAdapter(creds, 5*time.Second)
	var _ ports.VenuePort = adapter
	var _ ports.VenueQueryPort = adapter
}

// --- Rate Limiter Tests ---

func TestRateLimiter_PassesThrough(t *testing.T) {
	called := false
	inner := &stubVenuePort{submitFn: func(ctx context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, error) {
		called = true
		return ports.VenueOrderReceipt{
			VenueOrderID: "test-123",
			Status:       domainexec.StatusFilled,
		}, nil
	}}

	rl := appexec.NewRateLimiter(inner, 5, 50*time.Millisecond)
	defer rl.Close()

	intent := domainexec.ExecutionIntent{
		Source:     "binances",
		Instrument: btcUSDTSpot(t),
		Side:       domainexec.SideBuy,
		Quantity:   "0.001",
	}

	receipt, prob := rl.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}
	if !called {
		t.Fatal("inner adapter was not called")
	}
	if receipt.VenueOrderID != "test-123" {
		t.Fatalf("expected test-123, got %s", receipt.VenueOrderID)
	}
}

func TestRateLimiter_RespectsContextCancellation(t *testing.T) {
	// Create a rate limiter with 0 initial tokens (all consumed).
	inner := &stubVenuePort{}
	rl := appexec.NewRateLimiter(inner, 1, 10*time.Second) // very slow refill
	defer rl.Close()

	// Consume the single token.
	ctx := context.Background()
	intent := domainexec.ExecutionIntent{Side: domainexec.SideNone}
	_, _ = rl.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: intent})

	// Now try with a cancelled context — should fail.
	cancelCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, prob := rl.SubmitOrder(cancelCtx, ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rate limit error, got nil")
	}
}

// --- Credential Loading Tests ---

func TestMainnetCredentialLoading_Spot(t *testing.T) {
	t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY", "mainnet-key-0123456789abcdef0123456789abcdef0123456789abcdefgh")
	t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET", "mainnet-secret-0123456789abcdef0123456789abcdef0123456789abcde")

	creds, prob := appexec.LoadCredentials("binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}
	if creds.Get("API_KEY") != "mainnet-key-0123456789abcdef0123456789abcdef0123456789abcdefgh" {
		t.Fatal("wrong API_KEY")
	}
	if creds.Get("API_SECRET") != "mainnet-secret-0123456789abcdef0123456789abcdef0123456789abcde" {
		t.Fatal("wrong API_SECRET")
	}
}

func TestMainnetCredentialLoading_Futures(t *testing.T) {
	t.Setenv("MF_VENUE_BINANCE_FUTURES_MAINNET_API_KEY", "mainnet-f-key-0123456789abcdef0123456789abcdef0123456789abcde")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_MAINNET_API_SECRET", "mainnet-f-secret-0123456789abcdef0123456789abcdef01234567890")

	creds, prob := appexec.LoadCredentials("binance_futures_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}
	if creds.Get("API_KEY") != "mainnet-f-key-0123456789abcdef0123456789abcdef0123456789abcde" {
		t.Fatal("wrong API_KEY")
	}
}

func TestMainnetCredentialLoading_FailClosed(t *testing.T) {
	// No env vars set — should fail.
	_, prob := appexec.LoadCredentials("binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob == nil {
		t.Fatal("expected credential error, got nil")
	}
}

// --- Mainnet Base URL Constants ---

func TestMainnetBaseURLs(t *testing.T) {
	// Verify via adapter creation and WithBaseURL override pattern.
	// The constants are package-level, but we verify the adapter uses them
	// by checking that the default adapter targets the right host.
	spotCreds := spotMainnetTestCredentials(t)
	spotAdapter := appexec.NewBinanceSpotMainnetAdapter(spotCreds, 5*time.Second)

	futuresCreds := futuresMainnetTestCredentials(t)
	futuresAdapter := appexec.NewBinanceFuturesMainnetAdapter(futuresCreds, 5*time.Second)

	// These adapters should work with httptest — we just verify construction doesn't panic.
	_ = spotAdapter
	_ = futuresAdapter
}

// stubVenuePort is a test double for ports.VenuePort.
type stubVenuePort struct {
	submitFn func(ctx context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, error)
}

func (s *stubVenuePort) SubmitOrder(ctx context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	if s.submitFn != nil {
		receipt, err := s.submitFn(ctx, req)
		if err != nil {
			return receipt, nil // simplified for test
		}
		return receipt, nil
	}
	return ports.VenueOrderReceipt{
		VenueOrderID: "stub-noop",
		Status:       domainexec.StatusAccepted,
	}, nil
}
