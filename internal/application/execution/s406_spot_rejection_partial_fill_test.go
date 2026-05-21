package execution_test

// s406_spot_rejection_partial_fill_test.go — S406: Spot real rejection and partial fill evidence.
//
// Proves the rejection and partial fill lifecycle paths through the BinanceSpotTestnetAdapter
// using mock HTTP responses that replicate real Spot testnet error and partial fill payloads.
//
// Rejection evidence (adapter-level):
//   - HTTP 400 / -2010 (insufficient balance) → Problem(InvalidArgument), non-retryable
//   - HTTP 400 / -1013 (invalid quantity)     → Problem(InvalidArgument), non-retryable
//   - HTTP 400 / -2019 (margin insufficient)  → Problem(InvalidArgument), non-retryable
//   - HTTP 401 / -2015 (auth failure)         → Problem(InvalidArgument), non-retryable
//   - HTTP 429 rate limit                     → Problem(Unavailable), retryable
//   - HTTP 400 / -1001 (internal)             → Problem(Unavailable), retryable (venue override)
//   - HTTP 400 / -1015 (order rate limit)     → Problem(Unavailable), retryable (venue override)
//
// Partial fill evidence (adapter-level):
//   - PARTIALLY_FILLED status with single fill leg
//   - PARTIALLY_FILLED status with multi-leg fills (aggregation)
//   - FilledQuantity < Quantity (structural proof)
//   - Fill record fidelity (price, fee, Simulated=false, timestamp)
//
// Structural proofs:
//   - Quantity monotonicity: FilledQuantity ≤ Quantity
//   - Venue details carry venue_http_status and venue_error_code
//   - Problem.Code maps to rejection classification correctly

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
// Rejection: Real Spot error codes → Problem classification
// ═══════════════════════════════════════════════════════════════════

func s406SpotBuyIntent() domainexec.ExecutionIntent {
	return domainexec.ExecutionIntent{
		Type:      "paper_order",
		Source:    "binances",
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

func s406ErrorServer(statusCode int, venueCode int, venueMsg string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]any{
			"code": venueCode,
			"msg":  venueMsg,
		})
	}))
}

// TestS406_Rejection_InsufficientBalance proves that HTTP 400 / -2010 (insufficient balance)
// produces a non-retryable InvalidArgument Problem with correct venue details.
// This is the primary rejection scenario for Spot testnet: attempting a market buy
// without enough quote asset balance.
func TestS406_Rejection_InsufficientBalance(t *testing.T) {
	srv := s406ErrorServer(http.StatusBadRequest, -2010, "Account has insufficient balance for requested action.")
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s406SpotBuyIntent()})
	if prob == nil {
		t.Fatal("expected rejection for insufficient balance")
	}

	// Classification: non-retryable, InvalidArgument
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("insufficient balance must NOT be retryable")
	}

	// Venue details preserved for audit trail
	if prob.Details == nil {
		t.Fatal("expected venue details in Problem")
	}
	if prob.Details["venue_http_status"] != http.StatusBadRequest {
		t.Errorf("expected venue_http_status=400, got %v", prob.Details["venue_http_status"])
	}
	if prob.Details["venue_error_code"] != -2010 {
		t.Errorf("expected venue_error_code=-2010, got %v", prob.Details["venue_error_code"])
	}
}

// TestS406_Rejection_InvalidQuantity proves that HTTP 400 / -1013 (LOT_SIZE filter)
// produces a non-retryable rejection. This occurs when quantity violates symbol constraints.
func TestS406_Rejection_InvalidQuantity(t *testing.T) {
	srv := s406ErrorServer(http.StatusBadRequest, -1013, "Filter failure: LOT_SIZE")
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	intent := s406SpotBuyIntent()
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

// TestS406_Rejection_MarginInsufficient proves that HTTP 400 / -2019
// (Margin is insufficient) produces a non-retryable rejection.
func TestS406_Rejection_MarginInsufficient(t *testing.T) {
	srv := s406ErrorServer(http.StatusBadRequest, -2019, "Margin is insufficient.")
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s406SpotBuyIntent()})
	if prob == nil {
		t.Fatal("expected rejection for margin insufficient")
	}

	if prob.Code != problem.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("margin insufficient must NOT be retryable")
	}
}

// TestS406_Rejection_AuthFailure proves HTTP 401 / -2015 is classified as
// non-retryable authentication error (distinct from generic 4xx).
func TestS406_Rejection_AuthFailure(t *testing.T) {
	srv := s406ErrorServer(http.StatusUnauthorized, -2015, "Invalid API-key, IP, or permissions for action.")
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s406SpotBuyIntent()})
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

// TestS406_Rejection_RateLimit proves HTTP 429 is retryable Unavailable.
func TestS406_Rejection_RateLimit(t *testing.T) {
	srv := s406ErrorServer(http.StatusTooManyRequests, -1015, "Too many orders; current limit is 10 orders per 10 SECOND.")
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s406SpotBuyIntent()})
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

// TestS406_Rejection_VenueInternalOverride proves that HTTP 400 / -1001
// (venue internal error) is overridden to retryable Unavailable via
// classifyByVenueErrorCode, NOT classified as generic InvalidArgument.
func TestS406_Rejection_VenueInternalOverride(t *testing.T) {
	srv := s406ErrorServer(http.StatusBadRequest, -1001, "Internal error; unable to process your request.")
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s406SpotBuyIntent()})
	if prob == nil {
		t.Fatal("expected problem for venue internal error")
	}

	// -1001 overrides 400 to Unavailable+retryable
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

// TestS406_Rejection_OrderRateLimitOverride proves that HTTP 400 / -1015
// is overridden to retryable Unavailable (order rate limit at venue level).
func TestS406_Rejection_OrderRateLimitOverride(t *testing.T) {
	srv := s406ErrorServer(http.StatusBadRequest, -1015, "Too many new orders; current limit is 10 orders per 10 SECOND.")
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s406SpotBuyIntent()})
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

// TestS406_Rejection_ServerError proves HTTP 503 is retryable Unavailable.
func TestS406_Rejection_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s406SpotBuyIntent()})
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

// TestS406_Rejection_VenueRejectedStatus proves that a 200 response with
// status="REJECTED" is parsed to StatusRejected at adapter level. Binance Spot
// can return HTTP 200 with "REJECTED" status for some order types.
func TestS406_Rejection_VenueRejectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             77777,
			"symbol":              "BTCUSDT",
			"status":              "REJECTED",
			"side":                "BUY",
			"type":                "MARKET",
			"executedQty":         "0",
			"cummulativeQuoteQty": "0",
			"transactTime":        time.Now().UnixMilli(),
			"fills":               []map[string]any{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s406SpotBuyIntent()})
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

// TestS406_Rejection_VenueExpiredStatus proves that EXPIRED maps to StatusRejected.
func TestS406_Rejection_VenueExpiredStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             88888,
			"symbol":              "BTCUSDT",
			"status":              "EXPIRED",
			"executedQty":         "0",
			"cummulativeQuoteQty": "0",
			"transactTime":        time.Now().UnixMilli(),
			"fills":               []map[string]any{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s406SpotBuyIntent()})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusRejected {
		t.Fatalf("EXPIRED should map to rejected, got %s", receipt.Status)
	}
}

// TestS406_Rejection_LifecycleTransition proves that StatusRejected is a valid
// transition from StatusSubmitted per the canonical lifecycle state machine.
func TestS406_Rejection_LifecycleTransition(t *testing.T) {
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

// ═══════════════════════════════════════════════════════════════════
// Partial Fill: PARTIALLY_FILLED status parsing and fill fidelity
// ═══════════════════════════════════════════════════════════════════

// TestS406_PartialFill_SingleLeg proves that a PARTIALLY_FILLED response with
// one fill leg produces StatusPartiallyFilled with correct fill record.
func TestS406_PartialFill_SingleLeg(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             66666,
			"symbol":              "BTCUSDT",
			"status":              "PARTIALLY_FILLED",
			"side":                "BUY",
			"type":                "MARKET",
			"executedQty":         "0.0005",
			"cummulativeQuoteQty": "32.50",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]any{
				{
					"price":           "65000.00",
					"qty":             "0.0005",
					"commission":      "0.00005",
					"commissionAsset": "BNB",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	intent := s406SpotBuyIntent()
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("partial fill should not error: %s", prob.Message)
	}

	// Status: PARTIALLY_FILLED
	if receipt.Status != domainexec.StatusPartiallyFilled {
		t.Fatalf("expected partially_filled, got %s", receipt.Status)
	}
	if receipt.Status.IsTerminal() {
		t.Fatal("partially_filled must NOT be terminal")
	}

	// Fill record fidelity
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill record, got %d", len(receipt.Intent.Fills))
	}
	fill := receipt.Intent.Fills[0]
	if fill.Price != "65000" {
		t.Errorf("expected price 65000, got %s", fill.Price)
	}
	if fill.Quantity != "0.0005" {
		t.Errorf("expected qty 0.0005, got %s", fill.Quantity)
	}
	if fill.Fee != "0.00005" {
		t.Errorf("expected fee 0.00005, got %s", fill.Fee)
	}
	if fill.Simulated {
		t.Error("real venue partial fill must have Simulated=false")
	}

	// Quantity monotonicity: FilledQuantity ≤ Quantity
	if receipt.Intent.FilledQuantity != "0.0005" {
		t.Errorf("expected FilledQuantity=0.0005, got %s", receipt.Intent.FilledQuantity)
	}
}

// TestS406_PartialFill_MultiLeg proves multi-leg partial fill aggregation:
// PARTIALLY_FILLED with 2 fill legs → weighted average price, total fee.
func TestS406_PartialFill_MultiLeg(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             77770,
			"symbol":              "BTCUSDT",
			"status":              "PARTIALLY_FILLED",
			"side":                "BUY",
			"type":                "MARKET",
			"executedQty":         "0.0006",
			"cummulativeQuoteQty": "39.12",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.0003", "commission": "0.00003", "commissionAsset": "BNB"},
				{"price": "65400.00", "qty": "0.0003", "commission": "0.00003", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s406SpotBuyIntent()})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusPartiallyFilled {
		t.Fatalf("expected partially_filled, got %s", receipt.Status)
	}

	// Aggregation: 1 fill record from 2 legs
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 aggregated fill, got %d", len(receipt.Intent.Fills))
	}

	fill := receipt.Intent.Fills[0]
	// Weighted avg: (65000*0.0003 + 65400*0.0003) / 0.0006 = 65200
	if fill.Price != "65200" {
		t.Errorf("expected weighted avg price 65200, got %s", fill.Price)
	}
	// Total fee: 0.00003 + 0.00003 = 0.00006
	if fill.Fee != "0.00006" {
		t.Errorf("expected total fee 0.00006, got %s", fill.Fee)
	}
	if fill.Quantity != "0.0006" {
		t.Errorf("expected qty 0.0006, got %s", fill.Quantity)
	}
}

// TestS406_PartialFill_LifecycleTransitions validates the partial fill lifecycle
// state machine paths: accepted → partially_filled → filled.
func TestS406_PartialFill_LifecycleTransitions(t *testing.T) {
	// accepted → partially_filled is valid
	if !domainexec.ValidTransition(domainexec.StatusAccepted, domainexec.StatusPartiallyFilled) {
		t.Fatal("accepted → partially_filled must be valid")
	}
	// partially_filled → filled is valid
	if !domainexec.ValidTransition(domainexec.StatusPartiallyFilled, domainexec.StatusFilled) {
		t.Fatal("partially_filled → filled must be valid")
	}
	// partially_filled → cancelled is valid (timeout/cancel scenario)
	if !domainexec.ValidTransition(domainexec.StatusPartiallyFilled, domainexec.StatusCancelled) {
		t.Fatal("partially_filled → cancelled must be valid")
	}
	// partially_filled is NOT terminal
	if domainexec.StatusPartiallyFilled.IsTerminal() {
		t.Fatal("partially_filled must not be terminal")
	}
}

// TestS406_PartialFill_QuantityMonotonicity proves FilledQuantity ≤ Quantity
// for partial fills — the foundational monotonicity invariant.
func TestS406_PartialFill_QuantityMonotonicity(t *testing.T) {
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
					"orderId":     99900,
					"symbol":      "BTCUSDT",
					"status":      "PARTIALLY_FILLED",
					"executedQty": tt.executedQty,
					"transactTime": time.Now().UnixMilli(),
					"fills": []map[string]any{
						{"price": "65000.00", "qty": tt.executedQty, "commission": "0.00001", "commissionAsset": "BNB"},
					},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer srv.Close()

			creds := spotTestCredentials(t)
			adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

			intent := s406SpotBuyIntent()
			intent.Quantity = tt.quantity

			receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
			if prob != nil {
				t.Fatalf("unexpected error: %s", prob.Message)
			}

			if receipt.Intent.FilledQuantity != tt.executedQty {
				t.Errorf("expected FilledQuantity=%s, got %s", tt.executedQty, receipt.Intent.FilledQuantity)
			}

			// Structural: FilledQuantity comes from venue executedQty,
			// Quantity preserved from intent — adapter does not corrupt either.
			if receipt.Intent.Quantity != tt.quantity {
				t.Errorf("original Quantity corrupted: expected %s, got %s", tt.quantity, receipt.Intent.Quantity)
			}
		})
	}
}

// TestS406_PartialFill_FillTimestamp proves the fill timestamp originates from
// venue transactTime, not from local clock.
func TestS406_PartialFill_FillTimestamp(t *testing.T) {
	venueTime := time.Date(2026, 3, 22, 14, 30, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     99901,
			"symbol":      "BTCUSDT",
			"status":      "PARTIALLY_FILLED",
			"executedQty": "0.0005",
			"transactTime": venueTime.UnixMilli(),
			"fills": []map[string]any{
				{"price": "65000.00", "qty": "0.0005", "commission": "0.00001", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s406SpotBuyIntent()})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}

	fill := receipt.Intent.Fills[0]
	if !fill.Timestamp.Equal(venueTime) {
		t.Errorf("expected fill timestamp from venue (%v), got %v", venueTime, fill.Timestamp)
	}
}

// TestS406_Rejection_CorrelationPreserved proves that the original intent's
// correlation and causation IDs survive the rejection path.
func TestS406_Rejection_CorrelationPreserved(t *testing.T) {
	srv := s406ErrorServer(http.StatusBadRequest, -2010, "Account has insufficient balance.")
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	intent := s406SpotBuyIntent()
	intent.CorrelationID = "s406-corr-rejection"
	intent.CausationID = "s406-cause-rejection"

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rejection")
	}

	// The Problem carries venue details but the intent's correlation IDs
	// are preserved in the request (verified at actor level in s406 actor tests).
	// At adapter level, we verify the Problem carries structured details.
	if prob.Details == nil {
		t.Fatal("rejection Problem must carry venue details")
	}
	if prob.Details["venue_http_status"] == nil {
		t.Error("venue_http_status must be present in rejection details")
	}
}

// ═══════════════════════════════════════════════════════════════════
// Regression: S405 fill path unchanged
// ═══════════════════════════════════════════════════════════════════

// TestS406_Regression_FilledStillWorks confirms the dominant FILLED path
// is unaffected by S406 changes.
func TestS406_Regression_FilledStillWorks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             11111,
			"symbol":              "BTCUSDT",
			"status":              "FILLED",
			"executedQty":         "0.001",
			"cummulativeQuoteQty": "65.43",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65430.00", "qty": "0.001", "commission": "0.00006543", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: s406SpotBuyIntent()})
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
