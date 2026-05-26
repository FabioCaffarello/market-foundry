package execution_test

// S314 — Error Classification Completeness (VA-1) and Retryable Flag Completeness (RF-1).
// Each test maps to one or more exit criteria from the adapter hardening tranche.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/problem"
)

// ---------------------------------------------------------------------------
// VA-1.1 / RF-1.7: HTTP 401 → InvalidArgument, Retryable == false
// ---------------------------------------------------------------------------
func TestVA1_1_HTTP401_InvalidArgument_NotRetryable(t *testing.T) {
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

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for HTTP 401")
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("VA-1.1: expected %s, got %s", problem.InvalidArgument, prob.Code)
	}
	if prob.Retryable {
		t.Fatal("RF-1.7: HTTP 401 must not be retryable")
	}
}

// ---------------------------------------------------------------------------
// VA-1.2 / RF-1.7: HTTP 403 → InvalidArgument, Retryable == false
// ---------------------------------------------------------------------------
func TestVA1_2_HTTP403_InvalidArgument_NotRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -2015,
			"msg":  "Invalid API-key, IP, or permissions for action.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for HTTP 403")
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("VA-1.2: expected %s, got %s", problem.InvalidArgument, prob.Code)
	}
	if prob.Retryable {
		t.Fatal("RF-1.7: HTTP 403 must not be retryable")
	}
}

// ---------------------------------------------------------------------------
// VA-1.3 / RF-1.8: HTTP 400 → InvalidArgument, Retryable == false
// ---------------------------------------------------------------------------
func TestVA1_3_HTTP400_InvalidArgument_NotRetryable(t *testing.T) {
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

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for HTTP 400")
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("VA-1.3: expected %s, got %s", problem.InvalidArgument, prob.Code)
	}
	if prob.Retryable {
		t.Fatal("RF-1.8: HTTP 400 must not be retryable")
	}
}

// ---------------------------------------------------------------------------
// VA-1.4 / RF-1.8: HTTP 422 → InvalidArgument, Retryable == false
// ---------------------------------------------------------------------------
func TestVA1_4_HTTP422_InvalidArgument_NotRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1100,
			"msg":  "Illegal characters found in a parameter.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for HTTP 422")
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("VA-1.4: expected %s, got %s", problem.InvalidArgument, prob.Code)
	}
	if prob.Retryable {
		t.Fatal("RF-1.8: HTTP 422 must not be retryable")
	}
}

// ---------------------------------------------------------------------------
// VA-1.5 / RF-1.2: HTTP 429 → Unavailable, Retryable == true
// ---------------------------------------------------------------------------
func TestVA1_5_HTTP429_Unavailable_Retryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1015,
			"msg":  "Too many new orders.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for HTTP 429")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("VA-1.5: expected %s, got %s", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("RF-1.2: HTTP 429 must be retryable")
	}
}

// ---------------------------------------------------------------------------
// VA-1.6 / RF-1.3: HTTP 503 → Unavailable, Retryable == true
// ---------------------------------------------------------------------------
func TestVA1_6_HTTP503_Unavailable_Retryable(t *testing.T) {
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

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for HTTP 503")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("VA-1.6: expected %s, got %s", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("RF-1.3: HTTP 503 must be retryable")
	}
}

// ---------------------------------------------------------------------------
// VA-1.7 / RF-1.4: HTTP 500 → Unavailable, Retryable == true
// ---------------------------------------------------------------------------
func TestVA1_7_HTTP500_Unavailable_Retryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1001,
			"msg":  "Internal error.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("VA-1.7: expected %s, got %s", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("RF-1.4: HTTP 500 must be retryable")
	}
}

// ---------------------------------------------------------------------------
// VA-1.8 / RF-1.4: HTTP 502 → Unavailable, Retryable == true
// ---------------------------------------------------------------------------
func TestVA1_8_HTTP502_Unavailable_Retryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1001,
			"msg":  "Bad Gateway.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for HTTP 502")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("VA-1.8: expected %s, got %s", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("RF-1.4: HTTP 502 must be retryable")
	}
}

// ---------------------------------------------------------------------------
// VA-1.9 / RF-1.5: DNS/TCP/TLS error → Unavailable, Retryable == true
// ---------------------------------------------------------------------------
func TestVA1_9_NetworkFailure_Unavailable_Retryable(t *testing.T) {
	// Point the adapter at a non-routable address to force a connection failure.
	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 2*time.Second).
		WithBaseURL("http://192.0.2.1:1") // RFC 5737 TEST-NET — guaranteed non-routable

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, prob := adapter.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for network failure")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("VA-1.9: expected %s, got %s", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("RF-1.5: network failure must be retryable")
	}
}

// TestVA1_9_DNSFailure_Unavailable_Retryable verifies DNS resolution failures.
func TestVA1_9_DNSFailure_Unavailable_Retryable(t *testing.T) {
	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 2*time.Second).
		WithBaseURL("http://this-host-does-not-exist.invalid:443")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, prob := adapter.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for DNS failure")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("VA-1.9: expected %s, got %s", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("RF-1.5: DNS failure must be retryable")
	}
}

// TestVA1_9_ConnectionRefused_Unavailable_Retryable verifies TCP connection refused.
func TestVA1_9_ConnectionRefused_Unavailable_Retryable(t *testing.T) {
	// Bind and immediately close a port to get a guaranteed-refused port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close() // Port is now closed → connection refused.

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 2*time.Second).
		WithBaseURL("http://" + addr)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for connection refused")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("VA-1.9: expected %s, got %s", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("RF-1.5: connection refused must be retryable")
	}
}

// ---------------------------------------------------------------------------
// VA-1.10 / RF-1.9: Malformed JSON response → Internal, Retryable == false
// ---------------------------------------------------------------------------
func TestVA1_10_MalformedJSON_Internal_NotRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{this is not valid json`))
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if prob.Code != problem.Internal {
		t.Fatalf("VA-1.10: expected %s, got %s", problem.Internal, prob.Code)
	}
	if prob.Retryable {
		t.Fatal("RF-1.9: parse failure must not be retryable")
	}
}

// TestVA1_10_EmptyBody_Internal_NotRetryable verifies empty 200 response.
func TestVA1_10_EmptyBody_Internal_NotRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Empty body — JSON unmarshal will fail.
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for empty response body")
	}
	if prob.Code != problem.Internal {
		t.Fatalf("VA-1.10: expected %s, got %s", problem.Internal, prob.Code)
	}
	if prob.Retryable {
		t.Fatal("RF-1.9: parse failure from empty body must not be retryable")
	}
}

// ---------------------------------------------------------------------------
// VA-1.11 / RF-1.10: Unknown venue status → Internal, Retryable == false
// ---------------------------------------------------------------------------
func TestVA1_11_UnknownVenueStatus_Internal_NotRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     555,
			"symbol":      "BTCUSDT",
			"status":      "PENDING_CANCEL", // Not in our mapping.
			"side":        "BUY",
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

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for unknown venue status")
	}
	if prob.Code != problem.Internal {
		t.Fatalf("VA-1.11: expected %s, got %s", problem.Internal, prob.Code)
	}
	if prob.Retryable {
		t.Fatal("RF-1.10: unknown status must not be retryable")
	}
}

// ---------------------------------------------------------------------------
// VA-1.13: Error messages never contain credentials or API keys
// ---------------------------------------------------------------------------
func TestVA1_13_NoCredentialsInErrorMessages(t *testing.T) {
	apiKey := "test-api-key-secret-value"
	apiSecret := "test-api-secret-value-hidden"

	scenarios := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"auth_401", 401, `{"code":-2015,"msg":"Invalid API-key."}`},
		{"auth_403", 403, `{"code":-2015,"msg":"Forbidden."}`},
		{"rejected_400", 400, `{"code":-1121,"msg":"Invalid symbol."}`},
		{"rate_429", 429, `{"code":-1015,"msg":"Too many requests."}`},
		{"server_500", 500, `{"code":-1001,"msg":"Internal error."}`},
		{"server_502", 502, `{"code":-1001,"msg":"Bad Gateway."}`},
		{"server_503", 503, `{"code":-1001,"msg":"Service unavailable."}`},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(sc.statusCode)
				w.Write([]byte(sc.body))
			}))
			defer server.Close()

			t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", apiKey)
			t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", apiSecret)
			creds, cProb := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
			if cProb != nil {
				t.Fatalf("load creds: %s", cProb.Message)
			}

			adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
			_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
			if prob == nil {
				t.Fatal("expected error")
			}

			errMsg := prob.Error()
			if contains(errMsg, apiKey) {
				t.Fatalf("VA-1.13: error message contains API key: %s", errMsg)
			}
			if contains(errMsg, apiSecret) {
				t.Fatalf("VA-1.13: error message contains API secret: %s", errMsg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RF-1.6: Context deadline exceeded → Retryable == true
// ---------------------------------------------------------------------------
func TestRF1_6_ContextDeadline_Retryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 30*time.Second).WithBaseURL(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, prob := adapter.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error for context deadline")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("RF-1.6: expected %s, got %s", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("RF-1.6: context deadline must be retryable")
	}
}

// ---------------------------------------------------------------------------
// RF-1.1: Verify every error path carries correct Retryable value via table test
// ---------------------------------------------------------------------------
func TestRF1_1_AllErrorPaths_RetryableConsistency(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		body       string
		wantCode   problem.ProblemCode
		wantRetry  bool
	}{
		{"401_auth", 401, `{"code":-2015,"msg":"Invalid API-key."}`, problem.InvalidArgument, false},
		{"403_auth", 403, `{"code":-2015,"msg":"Forbidden."}`, problem.InvalidArgument, false},
		{"400_rejected", 400, `{"code":-1121,"msg":"Invalid symbol."}`, problem.InvalidArgument, false},
		{"422_rejected", 422, `{"code":-1100,"msg":"Illegal param."}`, problem.InvalidArgument, false},
		{"429_rate", 429, `{"code":-1015,"msg":"Too many."}`, problem.Unavailable, true},
		{"500_server", 500, `{"code":-1001,"msg":"Internal."}`, problem.Unavailable, true},
		{"502_gateway", 502, `{"code":-1001,"msg":"Gateway."}`, problem.Unavailable, true},
		{"503_unavail", 503, `{"code":-1001,"msg":"Unavailable."}`, problem.Unavailable, true},
		{"504_timeout", 504, `{"code":-1001,"msg":"Timeout."}`, problem.Unavailable, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()

			creds := testCredentials(t)
			adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

			_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
			if prob == nil {
				t.Fatalf("expected error for HTTP %d", tc.statusCode)
			}
			if prob.Code != tc.wantCode {
				t.Fatalf("RF-1.1: HTTP %d → expected code %s, got %s", tc.statusCode, tc.wantCode, prob.Code)
			}
			if prob.Retryable != tc.wantRetry {
				t.Fatalf("RF-1.1: HTTP %d → expected retryable=%v, got %v", tc.statusCode, tc.wantRetry, prob.Retryable)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// VA-1 supplemental: Error details carry venue_http_status for observability
// ---------------------------------------------------------------------------
func TestVA1_ErrorDetails_VenueHTTPStatus(t *testing.T) {
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

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected error")
	}
	if prob.Details == nil {
		t.Fatal("expected details with venue_http_status")
	}
	httpStatus, ok := prob.Details["venue_http_status"]
	if !ok {
		t.Fatal("missing venue_http_status in problem details")
	}
	if httpStatus != 400 {
		t.Fatalf("expected venue_http_status=400, got %v", httpStatus)
	}
	venueCode, ok := prob.Details["venue_error_code"]
	if !ok {
		t.Fatal("missing venue_error_code in problem details")
	}
	if venueCode != -1121 {
		t.Fatalf("expected venue_error_code=-1121, got %v", venueCode)
	}
}

// ---------------------------------------------------------------------------
// VA-1 supplemental: Binance status mapping completeness (all known statuses)
// ---------------------------------------------------------------------------
func TestVA1_StatusMapping_AllKnownStatuses(t *testing.T) {
	cases := []struct {
		binanceStatus string
		wantStatus    domainexec.Status
	}{
		{"NEW", domainexec.StatusAccepted},
		{"FILLED", domainexec.StatusFilled},
		{"PARTIALLY_FILLED", domainexec.StatusPartiallyFilled},
		{"CANCELED", domainexec.StatusCancelled},
		{"CANCELLED", domainexec.StatusCancelled},
		{"REJECTED", domainexec.StatusRejected},
		{"EXPIRED", domainexec.StatusRejected},
	}

	for _, tc := range cases {
		t.Run(tc.binanceStatus, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := map[string]any{
					"orderId":     1,
					"symbol":      "BTCUSDT",
					"status":      tc.binanceStatus,
					"side":        "BUY",
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

			receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
			if prob != nil {
				t.Fatalf("unexpected error for status %s: %s", tc.binanceStatus, prob.Message)
			}
			if receipt.Status != tc.wantStatus {
				t.Fatalf("status %s: expected %s, got %s", tc.binanceStatus, tc.wantStatus, receipt.Status)
			}
		})
	}
}

// contains checks if s contains substr (simple string search).
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
