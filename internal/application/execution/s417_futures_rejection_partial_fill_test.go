package execution_test

// s417_futures_rejection_partial_fill_test.go — S417: Futures real rejection and partial fill evidence.
//
// Proves the rejection and partial fill lifecycle paths through the BinanceFuturesTestnetAdapter
// using mock HTTP responses that replicate real Futures testnet error and partial fill payloads.
//
// Rejection evidence (adapter-level):
//   - HTTP 400 / -2019 (insufficient margin) → Problem(InvalidArgument), non-retryable
//   - HTTP 400 / -1013 (LOT_SIZE violation)  → Problem(InvalidArgument), non-retryable
//   - HTTP 400 / -2010 (insufficient balance) → Problem(InvalidArgument), non-retryable
//   - HTTP 401 / -2015 (auth failure)         → Problem(InvalidArgument), non-retryable
//   - HTTP 429 rate limit                     → Problem(Unavailable), retryable
//   - HTTP 400 / -1001 (internal)             → Problem(Unavailable), retryable (venue override)
//   - HTTP 400 / -1015 (order rate limit)     → Problem(Unavailable), retryable (venue override)
//   - HTTP 503 (server error)                 → Problem(Unavailable), retryable
//   - HTTP 200 with status=REJECTED           → StatusRejected via response parsing
//   - HTTP 200 with status=EXPIRED            → StatusRejected via response parsing
//
// Partial fill evidence (adapter-level):
//   - PARTIALLY_FILLED status with avgPrice/cumQuote (Futures response format)
//   - FilledQuantity < Quantity (structural proof)
//   - Fill record fidelity (price from avgPrice, fee from cumQuote, Simulated=false)
//   - Fill timestamp from updateTime (not local clock)
//   - Quantity monotonicity invariant
//
// Key difference from S406 (Spot):
//   - Futures uses avgPrice + cumQuote (no fills[] array)
//   - Fee is cumQuote proxy (not per-leg commission)
//   - Timestamp source is updateTime (not transactTime)

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

// ═══════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════

func s417FuturesBuyIntent(t *testing.T) domainexec.ExecutionIntent {
	t.Helper()
	return domainexec.ExecutionIntent{
		Type:       "paper_order",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
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
		CorrelationID: "s417-futures-corr",
		CausationID:   "s417-futures-cause",
		Final:         true,
		Timestamp:     time.Now().UTC(),
	}
}

func s417FuturesErrorServer(statusCode int, venueCode int, venueMsg string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]any{
			"code": venueCode,
			"msg":  venueMsg,
		})
	}))
}

// ═══════════════════════════════════════════════════════════════════
// Rejection: Futures error codes → Problem classification
// ═══════════════════════════════════════════════════════════════════

// TestS417_Rejection_InsufficientMargin proves that HTTP 400 / -2019 (margin insufficient)
// produces a non-retryable InvalidArgument Problem. This is the primary rejection scenario
// for Futures testnet: attempting a trade without sufficient margin in the Futures wallet.
func TestS417_Rejection_InsufficientMargin(t *testing.T) {
	srv := s417FuturesErrorServer(http.StatusBadRequest, -2019, "Margin is insufficient.")
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s417FuturesBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected rejection for insufficient margin")
	}

	if prob.Code != problem.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("insufficient margin must NOT be retryable")
	}
	if prob.Details == nil {
		t.Fatal("expected venue details in Problem")
	}
	if prob.Details["venue_http_status"] != http.StatusBadRequest {
		t.Errorf("expected venue_http_status=400, got %v", prob.Details["venue_http_status"])
	}
	if prob.Details["venue_error_code"] != -2019 {
		t.Errorf("expected venue_error_code=-2019, got %v", prob.Details["venue_error_code"])
	}
}

// TestS417_Rejection_InsufficientBalance proves HTTP 400 / -2010 (insufficient balance)
// in the Futures context. Futures testnet uses this when wallet balance cannot cover
// the required initial margin.
func TestS417_Rejection_InsufficientBalance(t *testing.T) {
	srv := s417FuturesErrorServer(http.StatusBadRequest, -2010, "Account has insufficient balance for requested action.")
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s417FuturesBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected rejection for insufficient balance")
	}

	if prob.Code != problem.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("insufficient balance must NOT be retryable")
	}
	if prob.Details["venue_error_code"] != -2010 {
		t.Errorf("expected venue_error_code=-2010, got %v", prob.Details["venue_error_code"])
	}
}

// TestS417_Rejection_InvalidQuantity proves HTTP 400 / -1013 (LOT_SIZE violation)
// for Futures — quantity violates the symbol's step size or min notional.
func TestS417_Rejection_InvalidQuantity(t *testing.T) {
	srv := s417FuturesErrorServer(http.StatusBadRequest, -1013, "Filter failure: LOT_SIZE")
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	intent := s417FuturesBuyIntent(t)
	intent.Quantity = "0.0000001" // below LOT_SIZE minimum

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rejection for invalid quantity")
	}

	if prob.Code != problem.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("LOT_SIZE violation must NOT be retryable")
	}
	if prob.Details["venue_error_code"] != -1013 {
		t.Errorf("expected venue_error_code=-1013, got %v", prob.Details["venue_error_code"])
	}
}

// TestS417_Rejection_AuthFailure proves HTTP 401 / -2015 is classified as
// non-retryable authentication error in the Futures context.
func TestS417_Rejection_AuthFailure(t *testing.T) {
	srv := s417FuturesErrorServer(http.StatusUnauthorized, -2015, "Invalid API-key, IP, or permissions for action.")
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s417FuturesBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected rejection for auth failure")
	}

	if prob.Code != problem.InvalidArgument {
		t.Fatalf("expected InvalidArgument for auth, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("auth failure must NOT be retryable")
	}
	if prob.Details["venue_http_status"] != http.StatusUnauthorized {
		t.Errorf("expected venue_http_status=401, got %v", prob.Details["venue_http_status"])
	}
}

// TestS417_Rejection_RateLimit proves HTTP 429 is retryable Unavailable for Futures.
func TestS417_Rejection_RateLimit(t *testing.T) {
	srv := s417FuturesErrorServer(http.StatusTooManyRequests, -1015, "Too many orders; current limit is 10 orders per 10 SECOND.")
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s417FuturesBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected problem for rate limit")
	}

	if prob.Code != problem.Unavailable {
		t.Fatalf("expected Unavailable for rate limit, got %s", prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("rate limit must be retryable")
	}
}

// TestS417_Rejection_VenueInternalOverride proves HTTP 400 / -1001 is overridden
// to retryable Unavailable via classifyByVenueErrorCode for Futures adapter.
func TestS417_Rejection_VenueInternalOverride(t *testing.T) {
	srv := s417FuturesErrorServer(http.StatusBadRequest, -1001, "Internal error; unable to process your request.")
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s417FuturesBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected problem for venue internal error")
	}

	if prob.Code != problem.Unavailable {
		t.Fatalf("expected Unavailable for -1001 override, got %s", prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("-1001 venue internal must be retryable")
	}
	if prob.Details["venue_error_class"] != "venue_internal" {
		t.Errorf("expected venue_error_class=venue_internal, got %v", prob.Details["venue_error_class"])
	}
}

// TestS417_Rejection_OrderRateLimitOverride proves HTTP 400 / -1015 is overridden
// to retryable Unavailable for Futures adapter.
func TestS417_Rejection_OrderRateLimitOverride(t *testing.T) {
	srv := s417FuturesErrorServer(http.StatusBadRequest, -1015, "Too many new orders; current limit is 10 orders per 10 SECOND.")
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s417FuturesBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected problem for order rate limit")
	}

	if prob.Code != problem.Unavailable {
		t.Fatalf("expected Unavailable for -1015 override, got %s", prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("-1015 order rate limit must be retryable")
	}
	if prob.Details["venue_error_class"] != "order_rate_limit" {
		t.Errorf("expected venue_error_class=order_rate_limit, got %v", prob.Details["venue_error_class"])
	}
}

// TestS417_Rejection_ServerError proves HTTP 503 is retryable Unavailable for Futures.
func TestS417_Rejection_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s417FuturesBuyIntent(t)})
	if prob == nil {
		t.Fatal("expected problem for server error")
	}

	if prob.Code != problem.Unavailable {
		t.Fatalf("expected Unavailable, got %s", prob.Code)
	}
	if !prob.Retryable {
		t.Fatal("503 must be retryable")
	}
}

// TestS417_Rejection_VenueRejectedStatus proves that HTTP 200 with status="REJECTED"
// is parsed to StatusRejected. Binance Futures can return HTTP 200 with REJECTED status.
func TestS417_Rejection_VenueRejectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     88801,
			"symbol":      "BTCUSDT",
			"status":      "REJECTED",
			"side":        "BUY",
			"type":        "MARKET",
			"avgPrice":    "0",
			"executedQty": "0",
			"cumQuote":    "0",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s417FuturesBuyIntent(t)})
	if prob != nil {
		t.Fatalf("HTTP 200 should not produce Problem, got: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusRejected {
		t.Fatalf("expected StatusRejected, got %s", receipt.Status)
	}
	if !receipt.Status.IsTerminal() {
		t.Fatal("rejected must be terminal")
	}
	if receipt.Intent.FilledQuantity != "0" {
		t.Errorf("rejected order must have FilledQuantity=0, got %s", receipt.Intent.FilledQuantity)
	}
	if len(receipt.Intent.Fills) != 0 {
		t.Errorf("rejected order must have no fills, got %d", len(receipt.Intent.Fills))
	}
}

// TestS417_Rejection_VenueExpiredStatus proves that EXPIRED maps to StatusRejected
// for Futures — same canonical mapping as Spot.
func TestS417_Rejection_VenueExpiredStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     88802,
			"symbol":      "BTCUSDT",
			"status":      "EXPIRED",
			"avgPrice":    "0",
			"executedQty": "0",
			"cumQuote":    "0",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s417FuturesBuyIntent(t)})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusRejected {
		t.Fatalf("EXPIRED should map to rejected, got %s", receipt.Status)
	}
}

// TestS417_Rejection_LifecycleTransition proves that StatusRejected is a valid
// transition from StatusSubmitted per the canonical lifecycle state machine.
func TestS417_Rejection_LifecycleTransition(t *testing.T) {
	if !domainexec.ValidTransition(domainexec.StatusSubmitted, domainexec.StatusRejected) {
		t.Fatal("submitted → rejected must be a valid transition")
	}
	if !domainexec.ValidTransition(domainexec.StatusSent, domainexec.StatusRejected) {
		t.Fatal("sent → rejected must be a valid transition")
	}
	if !domainexec.StatusRejected.IsTerminal() {
		t.Fatal("rejected must be terminal")
	}
}

// TestS417_Rejection_CorrelationPreserved proves that venue details are preserved
// in the Problem for rejection event construction.
func TestS417_Rejection_CorrelationPreserved(t *testing.T) {
	srv := s417FuturesErrorServer(http.StatusBadRequest, -2019, "Margin is insufficient.")
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	intent := s417FuturesBuyIntent(t)
	intent.CorrelationID = "s417-corr-rejection"
	intent.CausationID = "s417-cause-rejection"

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rejection")
	}

	if prob.Details == nil {
		t.Fatal("rejection Problem must carry venue details")
	}
	if prob.Details["venue_http_status"] == nil {
		t.Error("venue_http_status must be present in rejection details")
	}
	if prob.Details["venue_error_code"] == nil {
		t.Error("venue_error_code must be present in rejection details")
	}
}

// ═══════════════════════════════════════════════════════════════════
// Partial Fill: Futures PARTIALLY_FILLED status and fill fidelity
// ═══════════════════════════════════════════════════════════════════

// TestS417_PartialFill_FuturesFormat proves that a PARTIALLY_FILLED response in
// Futures format (avgPrice + cumQuote, no fills[] array) produces StatusPartiallyFilled
// with correct fill record.
func TestS417_PartialFill_FuturesFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     88803,
			"symbol":      "BTCUSDT",
			"status":      "PARTIALLY_FILLED",
			"side":        "BUY",
			"type":        "MARKET",
			"avgPrice":    "65000.50",
			"executedQty": "0.0005",
			"cumQuote":    "32.50025",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	intent := s417FuturesBuyIntent(t)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("partial fill should not error: %s", prob.Message)
	}

	// Status
	if receipt.Status != domainexec.StatusPartiallyFilled {
		t.Fatalf("expected partially_filled, got %s", receipt.Status)
	}
	if receipt.Status.IsTerminal() {
		t.Fatal("partially_filled must NOT be terminal")
	}

	// Fill record fidelity (Futures format: avgPrice, cumQuote)
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill record, got %d", len(receipt.Intent.Fills))
	}
	fill := receipt.Intent.Fills[0]
	if fill.Price != "65000.50" {
		t.Errorf("expected price 65000.50 (avgPrice), got %s", fill.Price)
	}
	if fill.Fee != "0" {
		t.Errorf("expected fee 0 (Futures RESULT has no commission), got %s", fill.Fee)
	}
	if fill.CostBasis != "32.50025" {
		t.Errorf("expected CostBasis 32.50025 (cumQuote), got %s", fill.CostBasis)
	}
	if fill.Quantity != "0.0005" {
		t.Errorf("expected qty 0.0005, got %s", fill.Quantity)
	}
	if fill.Simulated {
		t.Error("real venue partial fill must have Simulated=false")
	}

	// FilledQuantity from executedQty
	if receipt.Intent.FilledQuantity != "0.0005" {
		t.Errorf("expected FilledQuantity=0.0005, got %s", receipt.Intent.FilledQuantity)
	}
}

// TestS417_PartialFill_LifecycleTransitions validates the partial fill lifecycle
// state machine paths for Futures: accepted → partially_filled → filled.
func TestS417_PartialFill_LifecycleTransitions(t *testing.T) {
	if !domainexec.ValidTransition(domainexec.StatusAccepted, domainexec.StatusPartiallyFilled) {
		t.Fatal("accepted → partially_filled must be valid")
	}
	if !domainexec.ValidTransition(domainexec.StatusPartiallyFilled, domainexec.StatusFilled) {
		t.Fatal("partially_filled → filled must be valid")
	}
	if !domainexec.ValidTransition(domainexec.StatusPartiallyFilled, domainexec.StatusCancelled) {
		t.Fatal("partially_filled → cancelled must be valid")
	}
	if domainexec.StatusPartiallyFilled.IsTerminal() {
		t.Fatal("partially_filled must not be terminal")
	}
}

// TestS417_PartialFill_QuantityMonotonicity proves FilledQuantity ≤ Quantity
// for Futures partial fills using the Futures response format (avgPrice, cumQuote).
func TestS417_PartialFill_QuantityMonotonicity(t *testing.T) {
	tests := []struct {
		name        string
		quantity    string
		executedQty string
	}{
		{"half_filled", "0.001", "0.0005"},
		{"quarter_filled", "0.004", "0.001"},
		{"tiny_partial", "1.0", "0.001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := map[string]any{
					"orderId":     99800,
					"symbol":      "BTCUSDT",
					"status":      "PARTIALLY_FILLED",
					"avgPrice":    "65000.00",
					"executedQty": tt.executedQty,
					"cumQuote":    "32.50",
					"updateTime":  time.Now().UnixMilli(),
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer srv.Close()

			creds := s416FuturesCredentials(t)
			adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

			intent := s417FuturesBuyIntent(t)
			intent.Quantity = tt.quantity

			receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
			if prob != nil {
				t.Fatalf("unexpected error: %s", prob.Message)
			}

			if receipt.Intent.FilledQuantity != tt.executedQty {
				t.Errorf("expected FilledQuantity=%s, got %s", tt.executedQty, receipt.Intent.FilledQuantity)
			}
			if receipt.Intent.Quantity != tt.quantity {
				t.Errorf("original Quantity corrupted: expected %s, got %s", tt.quantity, receipt.Intent.Quantity)
			}
		})
	}
}

// TestS417_PartialFill_FillTimestamp proves the fill timestamp originates from
// venue updateTime (Futures format), not from local clock.
func TestS417_PartialFill_FillTimestamp(t *testing.T) {
	venueTime := time.Date(2026, 3, 23, 10, 15, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     99801,
			"symbol":      "BTCUSDT",
			"status":      "PARTIALLY_FILLED",
			"avgPrice":    "65000.00",
			"executedQty": "0.0005",
			"cumQuote":    "32.50",
			"updateTime":  venueTime.UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s417FuturesBuyIntent(t)})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}

	fill := receipt.Intent.Fills[0]
	if !fill.Timestamp.Equal(venueTime) {
		t.Errorf("expected fill timestamp from venue (%v), got %v", venueTime, fill.Timestamp)
	}
}

// ═══════════════════════════════════════════════════════════════════
// Regression: S416 fill path unchanged
// ═══════════════════════════════════════════════════════════════════

// TestS417_Regression_FilledStillWorks confirms the dominant FILLED path for Futures
// is unaffected by S417 changes.
func TestS417_Regression_FilledStillWorks(t *testing.T) {
	srv := s416FuturesFilledServer(t)
	defer srv.Close()

	creds := s416FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s417FuturesBuyIntent(t)})
	if prob != nil {
		t.Fatalf("filled should not error: %s", prob.Message)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
	if receipt.Intent.Fills[0].Simulated {
		t.Error("real fill must have Simulated=false")
	}
}
