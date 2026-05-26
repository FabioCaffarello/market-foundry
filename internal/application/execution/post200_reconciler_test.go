package execution_test

// S322 — Post-200 Reconciliation Tests.
//
// These tests verify the Post200Reconciler's behavior when the venue accepts
// an order (HTTP 200) but the response body is lost. The reconciler must:
//   - Detect the body-read-failure-after-200 marker
//   - Query the venue using the deterministic client order ID
//   - Recover order status and fills without re-submitting
//   - Pass through all non-body-read errors unchanged

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
// RC-01: Body read failure after 200 — recovery via QueryOrder
// ---------------------------------------------------------------------------
func TestRC01_BodyReadFailure_RecoveredViaQuery(t *testing.T) {
	var submitCalls, queryCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			submitCalls.Add(1)
			// Send 200 headers, then stall to trigger body read timeout.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			time.Sleep(3 * time.Second)
			w.Write([]byte(`{"orderId":7777,"status":"FILLED"}`))
			return
		}
		if r.Method == http.MethodGet {
			queryCalls.Add(1)
			// QueryOrder succeeds with the order status.
			resp := map[string]any{
				"orderId":     7777,
				"symbol":      "BTCUSDT",
				"status":      "FILLED",
				"avgPrice":    "65000.00",
				"executedQty": "0.001",
				"cumQuote":    "65.00",
				"updateTime":  time.Now().UnixMilli(),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 300*time.Millisecond).WithBaseURL(server.URL)

	reconciler := appexec.NewPost200Reconciler(adapter, adapter, 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	receipt, prob := reconciler.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob != nil {
		t.Fatalf("RC-01: expected recovery, got error: %s", prob.Message)
	}

	if submitCalls.Load() != 1 {
		t.Fatalf("RC-01: expected exactly 1 submit call, got %d", submitCalls.Load())
	}
	if queryCalls.Load() != 1 {
		t.Fatalf("RC-01: expected exactly 1 query call, got %d", queryCalls.Load())
	}
	if receipt.VenueOrderID != "7777" {
		t.Fatalf("RC-01: expected venue order ID 7777, got %s", receipt.VenueOrderID)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("RC-01: expected status filled, got %s", receipt.Status)
	}
	if receipt.ClientOrderID == "" {
		t.Fatal("RC-01: client order ID must be populated in recovered receipt")
	}
}

// ---------------------------------------------------------------------------
// RC-02: Body read failure — query also fails → enriched original error
// ---------------------------------------------------------------------------
func TestRC02_BodyReadFailure_QueryFails_OriginalErrorReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			time.Sleep(3 * time.Second)
			return
		}
		if r.Method == http.MethodGet {
			// Query fails with 404 (order not found yet, race condition).
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"code": -2013, "msg": "Order does not exist."})
			return
		}
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 300*time.Millisecond).WithBaseURL(server.URL)

	reconciler := appexec.NewPost200Reconciler(adapter, adapter, 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, prob := reconciler.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("RC-02: expected error when both submit body read and query fail")
	}

	// Must carry reconciliation metadata.
	if v, ok := prob.Details["reconciliation_attempted"]; !ok || v != true {
		t.Fatal("RC-02: expected reconciliation_attempted=true in details")
	}
	if v, ok := prob.Details["reconciliation_failed"]; !ok || v != true {
		t.Fatal("RC-02: expected reconciliation_failed=true in details")
	}
	if _, ok := prob.Details["reconciliation_error"]; !ok {
		t.Fatal("RC-02: expected reconciliation_error in details")
	}
	// Original body_read_failure marker must still be present.
	if v, ok := prob.Details["body_read_failure_after_200"]; !ok || v != true {
		t.Fatal("RC-02: original body_read_failure_after_200 marker must be preserved")
	}
}

// ---------------------------------------------------------------------------
// RC-03: Non-body-read errors pass through unchanged
// ---------------------------------------------------------------------------
func TestRC03_NonBodyReadError_PassesThrough(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, problem.New(problem.InvalidArgument, "venue rejected order")
	}}
	queryVenue := &fakeQueryVenue{behavior: func(_, _ string) (ports.VenueOrderReceipt, *problem.Problem) {
		t.Fatal("RC-03: query must not be called for non-body-read errors")
		return ports.VenueOrderReceipt{}, nil
	}}

	reconciler := appexec.NewPost200Reconciler(venue, queryVenue, 5*time.Second)

	_, prob := reconciler.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("RC-03: expected error to pass through")
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("RC-03: expected InvalidArgument, got %s", prob.Code)
	}
}

// ---------------------------------------------------------------------------
// RC-04: Successful submit — no reconciliation triggered
// ---------------------------------------------------------------------------
func TestRC04_SuccessfulSubmit_NoReconciliation(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{
			VenueOrderID: "8888",
			Status:       domainexec.StatusFilled,
		}, nil
	}}
	queryVenue := &fakeQueryVenue{behavior: func(_, _ string) (ports.VenueOrderReceipt, *problem.Problem) {
		t.Fatal("RC-04: query must not be called on successful submit")
		return ports.VenueOrderReceipt{}, nil
	}}

	reconciler := appexec.NewPost200Reconciler(venue, queryVenue, 5*time.Second)

	receipt, prob := reconciler.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob != nil {
		t.Fatalf("RC-04: expected success, got: %s", prob.Message)
	}
	if receipt.VenueOrderID != "8888" {
		t.Fatalf("RC-04: expected order ID 8888, got %s", receipt.VenueOrderID)
	}
}

// ---------------------------------------------------------------------------
// RC-05: No duplicate submit — only 1 POST, then GET for recovery
// ---------------------------------------------------------------------------
func TestRC05_NoDuplicateSubmit_OnlyOnePostThenGet(t *testing.T) {
	var postCalls, getCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			postCalls.Add(1)
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			time.Sleep(2 * time.Second)
		case http.MethodGet:
			getCalls.Add(1)
			resp := map[string]any{
				"orderId": 9999, "symbol": "BTCUSDT", "status": "FILLED",
				"avgPrice": "65000.00", "executedQty": "0.001",
				"cumQuote": "65.00", "updateTime": time.Now().UnixMilli(),
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 300*time.Millisecond).WithBaseURL(server.URL)

	reconciler := appexec.NewPost200Reconciler(adapter, adapter, 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, prob := reconciler.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob != nil {
		t.Fatalf("RC-05: expected recovery, got: %s", prob.Message)
	}

	if postCalls.Load() != 1 {
		t.Fatalf("RC-05: CRITICAL — expected exactly 1 POST (no duplicate submit), got %d", postCalls.Load())
	}
	if getCalls.Load() != 1 {
		t.Fatalf("RC-05: expected exactly 1 GET (recovery query), got %d", getCalls.Load())
	}
}

// ---------------------------------------------------------------------------
// RC-06: Client order ID in recovered receipt matches original intent
// ---------------------------------------------------------------------------
func TestRC06_RecoveredReceipt_HasCorrectClientOrderID(t *testing.T) {
	intent := testBuyIntent(t)
	expectedID := appexec.ClientOrderID(intent)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			time.Sleep(2 * time.Second)
			return
		}
		if r.Method == http.MethodGet {
			// Verify the query uses the correct client order ID.
			gotID := r.URL.Query().Get("origClientOrderId")
			if gotID != expectedID {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{
					"code": -1, "msg": fmt.Sprintf("wrong client order ID: got %s, want %s", gotID, expectedID),
				})
				return
			}
			resp := map[string]any{
				"orderId": 6666, "clientOrderId": expectedID,
				"symbol": "BTCUSDT", "status": "FILLED",
				"avgPrice": "65000.00", "executedQty": "0.001",
				"cumQuote": "65.00", "updateTime": time.Now().UnixMilli(),
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 300*time.Millisecond).WithBaseURL(server.URL)

	reconciler := appexec.NewPost200Reconciler(adapter, adapter, 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	receipt, prob := reconciler.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("RC-06: expected recovery, got: %s", prob.Message)
	}
	if receipt.ClientOrderID != expectedID {
		t.Fatalf("RC-06: client order ID mismatch: got %s, want %s", receipt.ClientOrderID, expectedID)
	}
}

// ---------------------------------------------------------------------------
// RC-07: Recovered intent preserves original fields (symbol, source, etc.)
// ---------------------------------------------------------------------------
func TestRC07_RecoveredIntent_PreservesOriginalFields(t *testing.T) {
	intent := testBuyIntent(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			time.Sleep(2 * time.Second)
			return
		}
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

	reconciler := appexec.NewPost200Reconciler(adapter, adapter, 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	receipt, prob := reconciler.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("RC-07: expected recovery, got: %s", prob.Message)
	}

	// Original intent fields must be preserved.
	if receipt.Intent.VenueSymbol() != intent.VenueSymbol() {
		t.Fatalf("RC-07: symbol changed: %s → %s", intent.VenueSymbol(), receipt.Intent.VenueSymbol())
	}
	if receipt.Intent.Source != intent.Source {
		t.Fatalf("RC-07: source changed: %s → %s", intent.Source, receipt.Intent.Source)
	}
	if receipt.Intent.Side != intent.Side {
		t.Fatalf("RC-07: side changed: %s → %s", intent.Side, receipt.Intent.Side)
	}
	if receipt.Intent.Quantity != intent.Quantity {
		t.Fatalf("RC-07: quantity changed: %s → %s", intent.Quantity, receipt.Intent.Quantity)
	}
	// Status should be updated to recovered status.
	if receipt.Intent.Status != domainexec.StatusFilled {
		t.Fatalf("RC-07: status should be filled, got %s", receipt.Intent.Status)
	}
}

// ---------------------------------------------------------------------------
// RC-08: Retryable errors (non-200) pass through without reconciliation
// ---------------------------------------------------------------------------
func TestRC08_RetryableError_PassesThrough(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "venue unavailable").MarkRetryable()
	}}
	queryVenue := &fakeQueryVenue{behavior: func(_, _ string) (ports.VenueOrderReceipt, *problem.Problem) {
		t.Fatal("RC-08: query must not be called for retryable errors")
		return ports.VenueOrderReceipt{}, nil
	}}

	reconciler := appexec.NewPost200Reconciler(venue, queryVenue, 5*time.Second)

	_, prob := reconciler.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob == nil {
		t.Fatal("RC-08: expected error to pass through")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("RC-08: expected Unavailable, got %s", prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("RC-08: retryable flag must be preserved")
	}
}

// ---------------------------------------------------------------------------
// RC-09: Reconciler composes with RetrySubmitter
// ---------------------------------------------------------------------------
func TestRC09_Reconciler_ComposesWithRetrySubmitter(t *testing.T) {
	var submitCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			n := submitCalls.Add(1)
			if n == 1 {
				// First attempt: 503 retryable.
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]any{"code": -1001, "msg": "unavailable"})
				return
			}
			// Second attempt: 200 but body read fails.
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			time.Sleep(2 * time.Second)
			return
		}
		if r.Method == http.MethodGet {
			resp := map[string]any{
				"orderId": 3333, "symbol": "BTCUSDT", "status": "FILLED",
				"avgPrice": "65000.00", "executedQty": "0.001",
				"cumQuote": "65.00", "updateTime": time.Now().UnixMilli(),
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 300*time.Millisecond).WithBaseURL(server.URL)

	// Stack: RetrySubmitter → Post200Reconciler → adapter
	retrier := appexec.NewRetrySubmitter(adapter, appexec.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	})
	retrier = retrier.TestWithSleepFn(noSleep)

	reconciler := appexec.NewPost200Reconciler(retrier, adapter, 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	receipt, prob := reconciler.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob != nil {
		t.Fatalf("RC-09: expected recovery after retry+reconciliation, got: %s", prob.Message)
	}
	if receipt.VenueOrderID != "3333" {
		t.Fatalf("RC-09: expected order ID 3333, got %s", receipt.VenueOrderID)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// fakeQueryVenue implements ports.VenueQueryPort for testing.
type fakeQueryVenue struct {
	behavior func(clientOrderID, symbol string) (ports.VenueOrderReceipt, *problem.Problem)
}

func (f *fakeQueryVenue) QueryOrder(_ context.Context, clientOrderID, symbol string) (ports.VenueOrderReceipt, *problem.Problem) {
	return f.behavior(clientOrderID, symbol)
}
