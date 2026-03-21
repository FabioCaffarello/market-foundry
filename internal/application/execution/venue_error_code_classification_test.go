package execution_test

// S325 — Venue Error Code Aware Classification Enrichment.
// Tests verify that specific Binance error codes override the default HTTP-based
// classification when they carry stronger semantic signal about retryability.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	"internal/shared/problem"
)

// ---------------------------------------------------------------------------
// EC-S325-1: HTTP 400 + code -1001 → Unavailable, retryable (venue internal)
// Binance returns -1001 "Internal error" even on HTTP 400 when the issue is
// server-side. Without error code enrichment, this would be InvalidArgument.
// ---------------------------------------------------------------------------
func TestEC_S325_1_HTTP400_Code1001_VenueInternal_Retryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1001,
			"msg":  "Internal error; unable to process your request. Please try again.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for HTTP 400 with code -1001")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("EC-S325-1: expected %s, got %s (venue internal should override InvalidArgument)", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("EC-S325-1: venue internal error (-1001) must be retryable")
	}
	assertDetail(t, prob, "venue_error_class", "venue_internal")
	assertDetail(t, prob, "venue_error_code", -1001)
	assertDetail(t, prob, "venue_http_status", 400)
}

// ---------------------------------------------------------------------------
// EC-S325-2: HTTP 418 + code -1003 → Unavailable, retryable (IP rate limit)
// Binance uses HTTP 418 for IP bans with code -1003. Without enrichment,
// this falls into the 4xx catch-all as InvalidArgument, non-retryable.
// ---------------------------------------------------------------------------
func TestEC_S325_2_HTTP418_Code1003_IPRateLimit_Retryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(418) // Binance IP ban status
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1003,
			"msg":  "Too many requests; IP has been auto-banned.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for HTTP 418 with code -1003")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("EC-S325-2: expected %s, got %s (IP rate limit should override InvalidArgument)", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("EC-S325-2: IP rate limit (-1003) must be retryable")
	}
	assertDetail(t, prob, "venue_error_class", "ip_rate_limit")
	assertDetail(t, prob, "venue_error_code", -1003)
	assertDetail(t, prob, "venue_http_status", 418)
}

// ---------------------------------------------------------------------------
// EC-S325-3: HTTP 400 + code -1015 → Unavailable, retryable (order rate limit)
// Binance returns -1015 "Too many new orders" as HTTP 400 when the order
// submission rate is exceeded. Without enrichment, classified as InvalidArgument.
// ---------------------------------------------------------------------------
func TestEC_S325_3_HTTP400_Code1015_OrderRateLimit_Retryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1015,
			"msg":  "Too many new orders; please use the websocket for live updates.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for HTTP 400 with code -1015")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("EC-S325-3: expected %s, got %s (order rate limit should override InvalidArgument)", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("EC-S325-3: order rate limit (-1015) must be retryable")
	}
	assertDetail(t, prob, "venue_error_class", "order_rate_limit")
	assertDetail(t, prob, "venue_error_code", -1015)
	assertDetail(t, prob, "venue_http_status", 400)
}

// ---------------------------------------------------------------------------
// EC-S325-4: HTTP 400 + code -1121 → InvalidArgument, non-retryable (NO override)
// Validates that unmapped error codes still fall through to HTTP-based classification.
// Code -1121 "Invalid symbol" is a genuine client error, NOT overridden.
// ---------------------------------------------------------------------------
func TestEC_S325_4_HTTP400_Code1121_NoOverride_InvalidArgument(t *testing.T) {
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
		t.Fatal("expected error for HTTP 400 with code -1121")
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("EC-S325-4: expected %s, got %s (unmapped code should not override)", problem.InvalidArgument, prob.Code)
	}
	if prob.Retryable {
		t.Fatal("EC-S325-4: unmapped error code must remain non-retryable")
	}
	// Unmapped codes should NOT have venue_error_class detail
	if prob.Details != nil {
		if _, hasClass := prob.Details["venue_error_class"]; hasClass {
			t.Fatal("EC-S325-4: unmapped error code should not have venue_error_class detail")
		}
	}
}

// ---------------------------------------------------------------------------
// EC-S325-5: HTTP 401 + code -1001 → InvalidArgument, non-retryable (NO override)
// Validates that auth errors are NEVER overridden by venue error codes.
// Even if code -1001 is mapped, HTTP 401/403 classification takes precedence.
// ---------------------------------------------------------------------------
func TestEC_S325_5_HTTP401_Code1001_AuthNotOverridden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -1001,
			"msg":  "Internal error.",
		})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for HTTP 401")
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("EC-S325-5: expected %s, got %s (auth must not be overridden)", problem.InvalidArgument, prob.Code)
	}
	if prob.Retryable {
		t.Fatal("EC-S325-5: auth errors must remain non-retryable regardless of venue code")
	}
}

// ---------------------------------------------------------------------------
// EC-S325-6: HTTP 429 + code -1015 → Unavailable, retryable (HTTP-based, NO override)
// Validates that HTTP 429 is already correctly classified and venue code does
// not interfere. The code override only applies where HTTP classification is wrong.
// ---------------------------------------------------------------------------
func TestEC_S325_6_HTTP429_Code1015_AlreadyCorrect(t *testing.T) {
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

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for HTTP 429")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("EC-S325-6: expected %s, got %s", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("EC-S325-6: HTTP 429 must remain retryable")
	}
}

// ---------------------------------------------------------------------------
// EC-S325-7: HTTP 500 + code -1001 → Unavailable, retryable (5xx, NO override)
// Validates that 5xx errors bypass the venue code override (already retryable).
// ---------------------------------------------------------------------------
func TestEC_S325_7_HTTP500_Code1001_5xxNotOverridden(t *testing.T) {
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

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("EC-S325-7: expected %s, got %s", problem.Unavailable, prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("EC-S325-7: HTTP 500 must remain retryable")
	}
}

// ---------------------------------------------------------------------------
// EC-S325-8: HTTP 400 + no error code → InvalidArgument, non-retryable
// Validates that when the venue returns no error code (code=0), the default
// HTTP-based classification applies without venue code interference.
// ---------------------------------------------------------------------------
func TestEC_S325_8_HTTP400_NoCode_FallsThrough(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"msg":"Bad request"}`))
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("expected error for HTTP 400 with no code")
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("EC-S325-8: expected %s, got %s", problem.InvalidArgument, prob.Code)
	}
	if prob.Retryable {
		t.Fatal("EC-S325-8: HTTP 400 with no venue code must be non-retryable")
	}
}

// ---------------------------------------------------------------------------
// EC-S325-9: Credential redaction preserved with venue code override
// Validates that venue code enrichment does not introduce credential leakage.
// ---------------------------------------------------------------------------
func TestEC_S325_9_CredentialRedaction_WithOverride(t *testing.T) {
	apiKey := "test-api-key-secret-s325"
	apiSecret := "test-api-secret-hidden-s325"

	codes := []struct {
		name       string
		statusCode int
		venueCode  int
	}{
		{"venue_internal", 400, -1001},
		{"ip_rate_limit", 418, -1003},
		{"order_rate_limit", 400, -1015},
	}

	for _, tc := range codes {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				json.NewEncoder(w).Encode(map[string]any{
					"code": tc.venueCode,
					"msg":  "Error message.",
				})
			}))
			defer server.Close()

			t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", apiKey)
			t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", apiSecret)
			creds, cProb := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
			if cProb != nil {
				t.Fatalf("load creds: %s", cProb.Message)
			}

			adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
			_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
			if prob == nil {
				t.Fatal("expected error")
			}

			errMsg := prob.Error()
			if contains(errMsg, apiKey) {
				t.Fatalf("EC-S325-9: error message contains API key: %s", errMsg)
			}
			if contains(errMsg, apiSecret) {
				t.Fatalf("EC-S325-9: error message contains API secret: %s", errMsg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EC-S325-10: Existing S314 tests remain unaffected (regression guard)
// The matrix test validates that all previously classified HTTP status codes
// retain their existing classification when paired with unmapped venue codes.
// ---------------------------------------------------------------------------
func TestEC_S325_10_ExistingClassification_Unchanged(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		venueCode  int
		wantCode   problem.ProblemCode
		wantRetry  bool
	}{
		// Auth errors: never overridden
		{"401_auth_2015", 401, -2015, problem.InvalidArgument, false},
		{"403_auth_2015", 403, -2015, problem.InvalidArgument, false},
		// Client errors with unmapped codes: unchanged
		{"400_invalid_1121", 400, -1121, problem.InvalidArgument, false},
		{"422_invalid_1100", 422, -1100, problem.InvalidArgument, false},
		// Rate limit by HTTP: unchanged
		{"429_rate_1015", 429, -1015, problem.Unavailable, true},
		// Server errors: unchanged
		{"500_server_1001", 500, -1001, problem.Unavailable, true},
		{"502_gateway_1001", 502, -1001, problem.Unavailable, true},
		{"503_unavail_1001", 503, -1001, problem.Unavailable, true},
		{"504_timeout_1001", 504, -1001, problem.Unavailable, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				json.NewEncoder(w).Encode(map[string]any{
					"code": tc.venueCode,
					"msg":  "Error.",
				})
			}))
			defer server.Close()

			creds := testCredentials(t)
			adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

			_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
			if prob == nil {
				t.Fatalf("expected error for HTTP %d", tc.statusCode)
			}
			if prob.Code != tc.wantCode {
				t.Fatalf("expected code %s, got %s", tc.wantCode, prob.Code)
			}
			if prob.Retryable != tc.wantRetry {
				t.Fatalf("expected retryable=%v, got %v", tc.wantRetry, prob.Retryable)
			}
		})
	}
}

// assertDetail verifies a specific key-value pair in problem.Details.
func assertDetail(t *testing.T, prob *problem.Problem, key string, want any) {
	t.Helper()
	if prob.Details == nil {
		t.Fatalf("expected detail %q but Details is nil", key)
	}
	got, ok := prob.Details[key]
	if !ok {
		t.Fatalf("expected detail %q but key not found in Details", key)
	}
	if got != want {
		t.Fatalf("detail %q: expected %v (%T), got %v (%T)", key, want, want, got, got)
	}
}
