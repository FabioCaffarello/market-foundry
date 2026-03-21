package execution_test

// S320 — Venue Failure Path Verification and Containment.
//
// These tests exercise integrated failure paths through the venue adapter
// and retry submitter, verifying classification correctness, retry behavior,
// containment semantics, abort conditions, and observable metadata.
//
// Each test is tagged with a failure mode identifier (FP-xx) for traceability.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/problem"
)

// ---------------------------------------------------------------------------
// FP-01: Timeout/deadline — context expires mid-retry loop
// ---------------------------------------------------------------------------
func TestFP01_ContextDeadline_ExpiresAcrossRetryAttempts(t *testing.T) {
	// Venue always returns 503 (retryable). Global context has a tight deadline
	// that expires after ~1 attempt + backoff, verifying the retry loop aborts.
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"code": -1001, "msg": "unavailable"})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 5*time.Second).WithBaseURL(server.URL)

	policy := appexec.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   200 * time.Millisecond, // long enough that context expires during backoff
		MaxDelay:    2 * time.Second,
		Factor:      2.0,
	}
	rs := appexec.NewRetrySubmitter(adapter, policy)

	// 250ms context: enough for ~1 attempt + start of first backoff sleep, then expires.
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	_, prob := rs.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("FP-01: expected error when context expires mid-retry")
	}

	// Must NOT have exhausted all 5 attempts — context should have stopped it early.
	if calls.Load() >= 5 {
		t.Fatalf("FP-01: expected fewer than 5 calls (context should abort), got %d", calls.Load())
	}

	// Retry metadata must be present.
	if _, ok := prob.Details["retry_exhausted"]; !ok {
		t.Fatal("FP-01: expected retry_exhausted in details")
	}
}

// ---------------------------------------------------------------------------
// FP-02: Auth failure — immediate abort, no retry
// ---------------------------------------------------------------------------
func TestFP02_AuthFailure_ThroughRetrySubmitter_NoRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"code": -2015, "msg": "Invalid API-key."})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	rs := appexec.NewRetrySubmitter(adapter, appexec.DefaultRetryPolicy())

	_, prob := rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("FP-02: expected auth error")
	}

	// Auth errors are non-retryable — must be exactly 1 call.
	if calls.Load() != 1 {
		t.Fatalf("FP-02: auth error must not retry, expected 1 call, got %d", calls.Load())
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("FP-02: expected InvalidArgument, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("FP-02: auth error must not be retryable")
	}
	// Must NOT carry retry metadata (non-retryable path).
	if _, ok := prob.Details["retry_attempts"]; ok {
		t.Fatal("FP-02: non-retryable error must not have retry_attempts")
	}
}

// ---------------------------------------------------------------------------
// FP-03: Auth failure after transient — escalation from retryable to non-retryable
// ---------------------------------------------------------------------------
func TestFP03_TransientThenAuthFailure_AbortsOnEscalation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(calls.Add(1))
		if n == 1 {
			// First call: retryable 503.
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]any{"code": -1001, "msg": "unavailable"})
			return
		}
		// Second call: non-retryable 401.
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"code": -2015, "msg": "Invalid API-key."})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	rs := appexec.NewRetrySubmitter(adapter, appexec.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	})

	_, prob := rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("FP-03: expected error")
	}

	// Must stop at attempt 2 — not retry after seeing 401.
	if calls.Load() != 2 {
		t.Fatalf("FP-03: expected 2 calls (503 then 401 abort), got %d", calls.Load())
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("FP-03: final error should be InvalidArgument (auth), got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("FP-03: escalated auth error must not be retryable")
	}
}

// ---------------------------------------------------------------------------
// FP-04: Rate limit recovery — 429 → 429 → success
// ---------------------------------------------------------------------------
func TestFP04_RateLimitRecovery_ThroughRetrySubmitter(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(calls.Add(1))
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]any{"code": -1015, "msg": "Too many requests."})
			return
		}
		resp := map[string]any{
			"orderId": 8888, "symbol": "BTCUSDT", "status": "FILLED",
			"avgPrice": "65000.00", "executedQty": "0.001",
			"cumQuote": "65.00", "updateTime": time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	rs := appexec.NewRetrySubmitter(adapter, appexec.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	})

	receipt, prob := rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob != nil {
		t.Fatalf("FP-04: expected success after rate limit recovery, got: %s", prob.Message)
	}
	if calls.Load() != 3 {
		t.Fatalf("FP-04: expected 3 calls (2x 429, 1x success), got %d", calls.Load())
	}
	if receipt.VenueOrderID != "8888" {
		t.Fatalf("FP-04: expected venue order ID 8888, got %s", receipt.VenueOrderID)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("FP-04: expected filled status, got %s", receipt.Status)
	}
}

// ---------------------------------------------------------------------------
// FP-05: Network failure recovery — connection refused → success
// ---------------------------------------------------------------------------
func TestFP05_NetworkFailureRecovery_ThroughRetrySubmitter(t *testing.T) {
	// Start with a server that works, but the first adapter call goes to a bad address.
	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId": 7777, "symbol": "BTCUSDT", "status": "FILLED",
			"avgPrice": "65000.00", "executedQty": "0.001",
			"cumQuote": "65.00", "updateTime": time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer goodServer.Close()

	// Use a fake venue that simulates network failure then success.
	var calls atomic.Int32
	venue := &fakeVenue{behavior: func(attempt int) (ports.VenueOrderReceipt, *problem.Problem) {
		calls.Add(1)
		if attempt == 1 {
			return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "connection refused").MarkRetryable()
		}
		return ports.VenueOrderReceipt{
			VenueOrderID: "7777",
			Status:       domainexec.StatusFilled,
		}, nil
	}}

	rs := appexec.NewRetrySubmitter(venue, appexec.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	})
	rs = rs.TestWithSleepFn(noSleep)

	receipt, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob != nil {
		t.Fatalf("FP-05: expected recovery after network failure, got: %s", prob.Message)
	}
	if calls.Load() != 2 {
		t.Fatalf("FP-05: expected 2 calls, got %d", calls.Load())
	}
	if receipt.VenueOrderID != "7777" {
		t.Fatalf("FP-05: expected venue order ID 7777, got %s", receipt.VenueOrderID)
	}
}

// ---------------------------------------------------------------------------
// FP-06: Retry exhaustion with mixed retryable errors — observability metadata
// ---------------------------------------------------------------------------
func TestFP06_RetryExhaustion_MixedErrors_MetadataComplete(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(calls.Add(1))
		switch n {
		case 1:
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]any{"code": -1015, "msg": "Rate limited"})
		case 2:
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]any{"code": -1001, "msg": "Bad gateway"})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{"code": -1001, "msg": "Server error"})
		}
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	rs := appexec.NewRetrySubmitter(adapter, appexec.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	})

	_, prob := rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("FP-06: expected error after exhausting retries")
	}

	if calls.Load() != 3 {
		t.Fatalf("FP-06: expected 3 calls, got %d", calls.Load())
	}

	// Verify retry metadata.
	attempts, ok := prob.Details["retry_attempts"]
	if !ok || attempts != 3 {
		t.Fatalf("FP-06: expected retry_attempts=3, got %v", attempts)
	}
	exhausted, ok := prob.Details["retry_exhausted"]
	if !ok || exhausted != true {
		t.Fatalf("FP-06: expected retry_exhausted=true, got %v", exhausted)
	}

	// Last error should carry venue_http_status from the 500 response.
	if httpStatus, ok := prob.Details["venue_http_status"]; ok {
		if httpStatus != 500 {
			t.Fatalf("FP-06: expected last venue_http_status=500, got %v", httpStatus)
		}
	}
}

// ---------------------------------------------------------------------------
// FP-07: HTTP 504 Gateway Timeout classification and retry
// ---------------------------------------------------------------------------
func TestFP07_HTTP504_GatewayTimeout_Classified_Retryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		json.NewEncoder(w).Encode(map[string]any{"code": -1001, "msg": "Gateway Timeout"})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("FP-07: expected error for HTTP 504")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("FP-07: expected Unavailable, got %s", prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("FP-07: HTTP 504 must be retryable")
	}
	if prob.Details["venue_http_status"] != 504 {
		t.Fatalf("FP-07: expected venue_http_status=504, got %v", prob.Details["venue_http_status"])
	}
}

// ---------------------------------------------------------------------------
// FP-08: Containment — non-retryable errors never leak into retry metadata
// ---------------------------------------------------------------------------
func TestFP08_Containment_NonRetryableErrors_NoRetryMetadata(t *testing.T) {
	nonRetryableCodes := []int{400, 401, 403, 422}
	for _, code := range nonRetryableCodes {
		t.Run(fmt.Sprintf("HTTP_%d", code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "error"})
			}))
			defer server.Close()

			creds := testCredentials(t)
			adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
			rs := appexec.NewRetrySubmitter(adapter, appexec.RetryPolicy{
				MaxAttempts: 5,
				BaseDelay:   time.Millisecond,
				MaxDelay:    10 * time.Millisecond,
				Factor:      2.0,
			})

			_, prob := rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
			if prob == nil {
				t.Fatalf("FP-08: expected error for HTTP %d", code)
			}
			if prob.Retryable {
				t.Fatalf("FP-08: HTTP %d must not be retryable", code)
			}
			if _, ok := prob.Details["retry_attempts"]; ok {
				t.Fatalf("FP-08: non-retryable HTTP %d must not have retry_attempts", code)
			}
			if _, ok := prob.Details["retry_exhausted"]; ok {
				t.Fatalf("FP-08: non-retryable HTTP %d must not have retry_exhausted", code)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FP-09: Containment — parse failure through retry submitter, no retry
// ---------------------------------------------------------------------------
func TestFP09_ParseFailure_ThroughRetrySubmitter_NoRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	rs := appexec.NewRetrySubmitter(adapter, appexec.DefaultRetryPolicy())

	_, prob := rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("FP-09: expected parse error")
	}
	if calls.Load() != 1 {
		t.Fatalf("FP-09: parse failure must not retry, expected 1 call, got %d", calls.Load())
	}
	if prob.Code != problem.Internal {
		t.Fatalf("FP-09: expected Internal, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("FP-09: parse failure must not be retryable")
	}
}

// ---------------------------------------------------------------------------
// FP-10: Timeout per-request — adapter timeout vs context deadline interaction
// ---------------------------------------------------------------------------
func TestFP10_AdapterTimeout_ShorterThanContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	creds := testCredentials(t)
	// Adapter timeout 200ms, context timeout 5s — adapter should fire first.
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 200*time.Millisecond).WithBaseURL(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	_, prob := adapter.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: testBuyIntent()})
	elapsed := time.Since(start)

	if prob == nil {
		t.Fatal("FP-10: expected timeout error")
	}
	if !prob.Retryable {
		t.Fatal("FP-10: timeout must be retryable")
	}
	// Should complete in ~200ms (adapter timeout), not 5s (context).
	if elapsed > 2*time.Second {
		t.Fatalf("FP-10: adapter timeout should fire before context, elapsed=%v", elapsed)
	}
}

// ---------------------------------------------------------------------------
// FP-11: Slow venue response — body read after HTTP 200 headers
// Finding: once venue returns 200 headers, a body-read failure is classified
// as Internal/non-retryable. This is correct because the venue has already
// accepted the order — retrying could cause double execution.
// ---------------------------------------------------------------------------
func TestFP11_SlowBody_BodyReadFailure_NotRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send 200 headers immediately, then stall on body.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		time.Sleep(3 * time.Second)
		w.Write([]byte(`{"orderId":1,"status":"FILLED"}`))
	}))
	defer server.Close()

	creds := testCredentials(t)
	// Short client timeout — fires during body read after 200 headers received.
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 500*time.Millisecond).WithBaseURL(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, prob := adapter.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("FP-11: expected error for slow body response")
	}
	// Key finding: body read failure after HTTP 200 is NON-retryable.
	// Venue has already accepted the order — retrying risks double execution.
	if prob.Retryable {
		t.Fatal("FP-11: body read failure after 200 must NOT be retryable (venue already accepted)")
	}
}

// ---------------------------------------------------------------------------
// FP-12: Intent immutability — intent is never mutated across retry attempts
// ---------------------------------------------------------------------------
func TestFP12_IntentImmutability_AcrossRetries(t *testing.T) {
	var capturedIntents []domainexec.ExecutionIntent
	venue := &fakeVenue{behavior: func(attempt int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, retryableProblem("transient")
	}}

	// Wrap to capture intents.
	intent := testBuyIntent()
	originalSymbol := intent.Symbol
	originalQty := intent.Quantity
	originalSide := intent.Side
	originalSource := intent.Source

	rs := appexec.NewRetrySubmitter(venue, appexec.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	})
	rs = rs.TestWithSleepFn(noSleep)

	rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})

	// Verify intent was not mutated.
	if intent.Symbol != originalSymbol {
		t.Fatalf("FP-12: symbol mutated: %s → %s", originalSymbol, intent.Symbol)
	}
	if intent.Quantity != originalQty {
		t.Fatalf("FP-12: quantity mutated: %s → %s", originalQty, intent.Quantity)
	}
	if intent.Side != originalSide {
		t.Fatalf("FP-12: side mutated: %s → %s", originalSide, intent.Side)
	}
	if intent.Source != originalSource {
		t.Fatalf("FP-12: source mutated: %s → %s", originalSource, intent.Source)
	}
	_ = capturedIntents
}

// ---------------------------------------------------------------------------
// FP-13: Client order ID determinism — same across retry + recovery
// ---------------------------------------------------------------------------
func TestFP13_ClientOrderID_StableAcrossRetryAndRecovery(t *testing.T) {
	var capturedIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIDs = append(capturedIDs, r.URL.Query().Get("newClientOrderId"))
		if len(capturedIDs) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]any{"code": -1001, "msg": "unavailable"})
			return
		}
		resp := map[string]any{
			"orderId": 9999, "symbol": "BTCUSDT", "status": "FILLED",
			"avgPrice": "65000.00", "executedQty": "0.001",
			"cumQuote": "65.00", "updateTime": time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	rs := appexec.NewRetrySubmitter(adapter, appexec.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	})

	intent := testBuyIntent()
	receipt, prob := rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("FP-13: expected success on third attempt, got: %s", prob.Message)
	}
	if receipt.VenueOrderID != "9999" {
		t.Fatalf("FP-13: expected order ID 9999, got %s", receipt.VenueOrderID)
	}

	// All 3 HTTP requests must carry the same client order ID.
	if len(capturedIDs) != 3 {
		t.Fatalf("FP-13: expected 3 captured IDs, got %d", len(capturedIDs))
	}
	expectedID := appexec.ClientOrderID(intent)
	for i, id := range capturedIDs {
		if id != expectedID {
			t.Fatalf("FP-13: attempt %d client order ID %q != expected %q", i+1, id, expectedID)
		}
	}
}

// ---------------------------------------------------------------------------
// FP-14: Error details propagation — venue_http_status and venue_error_code
// ---------------------------------------------------------------------------
func TestFP14_ErrorDetails_PropagateThroughRetrySubmitter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{"code": -1015, "msg": "Too many orders."})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	rs := appexec.NewRetrySubmitter(adapter, appexec.RetryPolicy{
		MaxAttempts: 2,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	})

	_, prob := rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("FP-14: expected error after retry exhaustion")
	}

	// Venue details must survive through the retry submitter.
	if prob.Details["venue_http_status"] != 429 {
		t.Fatalf("FP-14: expected venue_http_status=429, got %v", prob.Details["venue_http_status"])
	}
	if prob.Details["venue_error_code"] != -1015 {
		t.Fatalf("FP-14: expected venue_error_code=-1015, got %v", prob.Details["venue_error_code"])
	}
	// Retry metadata also present.
	if prob.Details["retry_attempts"] != 2 {
		t.Fatalf("FP-14: expected retry_attempts=2, got %v", prob.Details["retry_attempts"])
	}
	if prob.Details["retry_exhausted"] != true {
		t.Fatalf("FP-14: expected retry_exhausted=true, got %v", prob.Details["retry_exhausted"])
	}
}

// ---------------------------------------------------------------------------
// FP-15: HTTP 403 Forbidden — different from 401 but same classification
// ---------------------------------------------------------------------------
func TestFP15_HTTP403_ThroughRetrySubmitter_NoRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{"code": -2015, "msg": "Forbidden."})
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	rs := appexec.NewRetrySubmitter(adapter, appexec.DefaultRetryPolicy())

	_, prob := rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("FP-15: expected 403 error")
	}
	if calls.Load() != 1 {
		t.Fatalf("FP-15: 403 must not retry, expected 1 call, got %d", calls.Load())
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("FP-15: expected InvalidArgument, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("FP-15: 403 must not be retryable")
	}
}

// ---------------------------------------------------------------------------
// FP-16: Full adapter-level timeout recovery through retry
// ---------------------------------------------------------------------------
func TestFP16_AdapterTimeout_RecoveryOnRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(calls.Add(1))
		if n == 1 {
			// First call: stall to cause timeout.
			time.Sleep(2 * time.Second)
			return
		}
		// Second call: respond immediately.
		resp := map[string]any{
			"orderId": 5555, "symbol": "BTCUSDT", "status": "FILLED",
			"avgPrice": "65000.00", "executedQty": "0.001",
			"cumQuote": "65.00", "updateTime": time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 300*time.Millisecond).WithBaseURL(server.URL)
	rs := appexec.NewRetrySubmitter(adapter, appexec.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	})

	// Use a wide context so only the adapter-level timeout fires.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	receipt, prob := rs.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob != nil {
		t.Fatalf("FP-16: expected recovery on retry after timeout, got: %s", prob.Message)
	}
	if receipt.VenueOrderID != "5555" {
		t.Fatalf("FP-16: expected order ID 5555, got %s", receipt.VenueOrderID)
	}
	if calls.Load() != 2 {
		t.Fatalf("FP-16: expected 2 calls (timeout + success), got %d", calls.Load())
	}
}

// ---------------------------------------------------------------------------
// FP-17: No-action intent — bypasses venue entirely, no failure path
// ---------------------------------------------------------------------------
func TestFP17_NoActionIntent_BypassesVenue_ThroughRetrySubmitter(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Fatal("FP-17: no-action intent must not hit venue")
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	rs := appexec.NewRetrySubmitter(adapter, appexec.DefaultRetryPolicy())

	intent := testBuyIntent()
	intent.Side = domainexec.SideNone

	receipt, prob := rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("FP-17: no-action should succeed, got: %s", prob.Message)
	}
	if receipt.Status != domainexec.StatusAccepted {
		t.Fatalf("FP-17: expected accepted, got %s", receipt.Status)
	}
	if calls.Load() != 0 {
		t.Fatal("FP-17: no HTTP request should be made for no-action intent")
	}
}

// ---------------------------------------------------------------------------
// FP-18: Unknown status through retry submitter — contained as non-retryable
// ---------------------------------------------------------------------------
func TestFP18_UnknownVenueStatus_ThroughRetrySubmitter_NoRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		resp := map[string]any{
			"orderId": 4444, "symbol": "BTCUSDT", "status": "PENDING_CANCEL",
			"side": "BUY", "type": "MARKET", "avgPrice": "65000.00",
			"executedQty": "0.001", "cumQuote": "65.00",
			"updateTime": time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	rs := appexec.NewRetrySubmitter(adapter, appexec.DefaultRetryPolicy())

	_, prob := rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("FP-18: expected error for unknown status")
	}
	if calls.Load() != 1 {
		t.Fatalf("FP-18: unknown status must not retry, expected 1 call, got %d", calls.Load())
	}
	if prob.Code != problem.Internal {
		t.Fatalf("FP-18: expected Internal, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("FP-18: unknown status must not be retryable")
	}
}

// ---------------------------------------------------------------------------
// FP-19: Credential redaction — errors from retry submitter never leak secrets
// ---------------------------------------------------------------------------
func TestFP19_CredentialRedaction_ThroughRetrySubmitter(t *testing.T) {
	apiKey := "super-secret-api-key-12345"
	apiSecret := "ultra-secret-api-secret-67890"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"code": -2015, "msg": "Invalid API-key."})
	}))
	defer server.Close()

	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", apiKey)
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", apiSecret)
	creds, cProb := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	if cProb != nil {
		t.Fatalf("load creds: %s", cProb.Message)
	}

	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)
	rs := appexec.NewRetrySubmitter(adapter, appexec.RetryPolicy{
		MaxAttempts: 2,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	})

	_, prob := rs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent()})
	if prob == nil {
		t.Fatal("FP-19: expected error")
	}

	errMsg := prob.Error()
	if contains(errMsg, apiKey) {
		t.Fatalf("FP-19: error message leaks API key: %s", errMsg)
	}
	if contains(errMsg, apiSecret) {
		t.Fatalf("FP-19: error message leaks API secret: %s", errMsg)
	}
}
