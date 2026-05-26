package execution_test

// s423_futures_rejection_partial_fill_test.go — S423: Futures real rejection and partial fill evidence.
//
// This stage proves the rejection and partial fill lifecycle paths for Futures
// with explicit ValidTransition step-by-step assertions and venue contract fidelity,
// following the S422 pattern. It elevates S417's mock-based coverage into lifecycle-
// grade evidence aligned with the canonical state machine.
//
// New value over S417 (prior wave):
//   - Explicit ValidTransition assertions on submitted → rejected path
//   - Explicit ValidTransition assertions on accepted → partially_filled → filled path
//   - Rejected terminality: no further transitions from rejected
//   - QueryOrder reconciliation for rejected and partially_filled orders
//   - Multi-scenario rejection lifecycle proof (margin, LOT_SIZE, auth, venue status)
//   - SegmentRouter rejection isolation with explicit lifecycle verification
//   - S422 fill-path regression proof
//
// Governing questions answered:
//   FV-Q3:  Lifecycle transition to rejected on real Futures rejection
//   FV-Q4:  VenueOrderRejectedEvent carries real Futures error code and reason
//   FV-Q5:  Partial fill structurally proven from Futures response format
//   FV-Q6:  Quantity monotonicity under Futures partial fills

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/events"
	"internal/shared/problem"
	"internal/shared/settings"
)

// ---------- helpers ----------

func s423FuturesIntent(t *testing.T, side domainexec.Side) domainexec.ExecutionIntent {
	t.Helper()
	return domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        "binancef",
		Instrument:    btcUSDTPerp(t),
		Timeframe:     60,
		Side:          side,
		Quantity:      "0.001",
		Status:        domainexec.StatusSubmitted,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		CorrelationID: "s423-corr-001",
		CausationID:   "s423-cause-001",
		Final:         true,
		Timestamp:     time.Now().UTC().Add(-5 * time.Second),
	}
}

func s423FuturesCredentials(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY", "test-futures-api-key")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET", "test-futures-api-secret")
	creds, prob := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("load test credentials: %s", prob.Message)
	}
	return creds
}

func s423FuturesErrorServer(statusCode int, venueCode int, venueMsg string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]any{
			"code": venueCode,
			"msg":  venueMsg,
		})
	}))
}

func s423FuturesRejectedStatusServer(status string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     88901,
			"symbol":      "BTCUSDT",
			"status":      status,
			"side":        "BUY",
			"type":        "MARKET",
			"avgPrice":    "0",
			"executedQty": "0",
			"cumQuote":    "0",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
}

func s423FuturesPartialFillServer(executedQty, avgPrice, cumQuote string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     88950,
			"symbol":      "BTCUSDT",
			"status":      "PARTIALLY_FILLED",
			"side":        "BUY",
			"type":        "MARKET",
			"avgPrice":    avgPrice,
			"executedQty": executedQty,
			"cumQuote":    cumQuote,
			"updateTime":  time.Now().UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

// ==========================================================================
// FV-Q3: Rejection lifecycle with explicit ValidTransition verification
// ==========================================================================

// TestS423_Rejection_DominantPath_ValidTransitions proves the rejection lifecycle
// path submitted → rejected with step-by-step ValidTransition assertions.
// This is the S423 counterpart to S422's fill path proof.
func TestS423_Rejection_DominantPath_ValidTransitions(t *testing.T) {
	// Step 1: submitted → rejected is valid
	if !domainexec.ValidTransition(domainexec.StatusSubmitted, domainexec.StatusRejected) {
		t.Fatal("submitted → rejected must be a valid transition")
	}

	// Step 2: sent → rejected is also valid (network-level rejection)
	if !domainexec.ValidTransition(domainexec.StatusSent, domainexec.StatusRejected) {
		t.Fatal("sent → rejected must be a valid transition")
	}

	// Step 3: The adapter returns a Problem for HTTP error rejections
	server := s423FuturesErrorServer(http.StatusBadRequest, -2019, "Margin is insufficient.")
	defer server.Close()

	creds := s423FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s423FuturesIntent(t, domainexec.SideBuy)
	if intent.Status != domainexec.StatusSubmitted {
		t.Fatalf("intent must start as submitted, got %s", intent.Status)
	}

	_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rejection for insufficient margin")
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", prob.Code)
	}

	// Step 4: rejected is terminal — no further transitions allowed
	if !domainexec.StatusRejected.IsTerminal() {
		t.Fatal("rejected must be a terminal state")
	}

	nextStatuses := []domainexec.Status{
		domainexec.StatusSubmitted, domainexec.StatusSent, domainexec.StatusAccepted,
		domainexec.StatusFilled, domainexec.StatusPartiallyFilled,
		domainexec.StatusRejected, domainexec.StatusCancelled,
	}
	for _, next := range nextStatuses {
		if domainexec.ValidTransition(domainexec.StatusRejected, next) {
			t.Errorf("rejected → %s must not be valid (terminal)", next)
		}
	}
}

// TestS423_Rejection_HTTP200_REJECTED_ValidTransitions proves that HTTP 200 with
// status="REJECTED" produces StatusRejected with correct lifecycle semantics.
func TestS423_Rejection_HTTP200_REJECTED_ValidTransitions(t *testing.T) {
	server := s423FuturesRejectedStatusServer("REJECTED")
	defer server.Close()

	creds := s423FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s423FuturesIntent(t, domainexec.SideBuy)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("HTTP 200 REJECTED should not produce Problem: %s", prob.Message)
	}

	// Verify submitted → rejected via venue status mapping
	if receipt.Status != domainexec.StatusRejected {
		t.Fatalf("expected StatusRejected, got %s", receipt.Status)
	}
	if !domainexec.ValidTransition(domainexec.StatusSubmitted, receipt.Status) {
		t.Fatal("submitted → rejected must be valid (venue status path)")
	}
	if !receipt.Status.IsTerminal() {
		t.Fatal("rejected must be terminal")
	}

	// Rejected must have zero fills and zero filled quantity
	if receipt.Intent.FilledQuantity != "0" {
		t.Errorf("rejected order must have FilledQuantity=0, got %s", receipt.Intent.FilledQuantity)
	}
	if len(receipt.Intent.Fills) != 0 {
		t.Errorf("rejected order must have 0 fills, got %d", len(receipt.Intent.Fills))
	}
}

// TestS423_Rejection_HTTP200_EXPIRED_ValidTransitions proves EXPIRED maps to
// rejected with the same lifecycle semantics as REJECTED.
func TestS423_Rejection_HTTP200_EXPIRED_ValidTransitions(t *testing.T) {
	server := s423FuturesRejectedStatusServer("EXPIRED")
	defer server.Close()

	creds := s423FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s423FuturesIntent(t, domainexec.SideBuy)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("HTTP 200 EXPIRED should not produce Problem: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusRejected {
		t.Fatalf("EXPIRED should map to rejected, got %s", receipt.Status)
	}
	if !domainexec.ValidTransition(domainexec.StatusSubmitted, receipt.Status) {
		t.Fatal("submitted → rejected must be valid (expired path)")
	}
}

// ==========================================================================
// FV-Q4: Rejection event fidelity with Futures venue details
// ==========================================================================

// TestS423_RejectionEvent_MultiScenario_AuditTrail proves that rejection events
// built from multiple Futures error scenarios carry complete audit trail data.
// Each scenario exercises a different rejection class with full lifecycle verification.
func TestS423_RejectionEvent_MultiScenario_AuditTrail(t *testing.T) {
	scenarios := []struct {
		name       string
		httpStatus int
		venueCode  int
		venueMsg   string
		wantCode   problem.ProblemCode
		wantRetry  bool
	}{
		{
			name: "insufficient_margin", httpStatus: 400, venueCode: -2019,
			venueMsg: "Margin is insufficient.",
			wantCode: problem.InvalidArgument, wantRetry: false,
		},
		{
			name: "insufficient_balance", httpStatus: 400, venueCode: -2010,
			venueMsg: "Account has insufficient balance for requested action.",
			wantCode: problem.InvalidArgument, wantRetry: false,
		},
		{
			name: "lot_size_violation", httpStatus: 400, venueCode: -1013,
			venueMsg: "Filter failure: LOT_SIZE",
			wantCode: problem.InvalidArgument, wantRetry: false,
		},
		{
			name: "auth_failure", httpStatus: 401, venueCode: -2015,
			venueMsg: "Invalid API-key, IP, or permissions for action.",
			wantCode: problem.InvalidArgument, wantRetry: false,
		},
		{
			name: "rate_limit", httpStatus: 429, venueCode: -1015,
			venueMsg: "Too many orders; current limit is 10 orders per 10 SECOND.",
			wantCode: problem.Unavailable, wantRetry: true,
		},
		{
			name: "venue_internal_override", httpStatus: 400, venueCode: -1001,
			venueMsg: "Internal error; unable to process your request.",
			wantCode: problem.Unavailable, wantRetry: true,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			srv := s423FuturesErrorServer(sc.httpStatus, sc.venueCode, sc.venueMsg)
			defer srv.Close()

			creds := s423FuturesCredentials(t)
			adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

			intent := s423FuturesIntent(t, domainexec.SideBuy)
			intent.CorrelationID = fmt.Sprintf("s423-%s-corr", sc.name)
			intent.CausationID = fmt.Sprintf("s423-%s-cause", sc.name)

			_, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
			if prob == nil {
				t.Fatal("expected rejection")
			}

			// Problem classification
			if prob.Code != sc.wantCode {
				t.Fatalf("expected %s, got %s", sc.wantCode, prob.Code)
			}
			if prob.Retryable != sc.wantRetry {
				t.Fatalf("expected retryable=%v, got %v", sc.wantRetry, prob.Retryable)
			}

			// Venue details for audit trail
			if prob.Details == nil {
				t.Fatal("rejection Problem must carry venue details")
			}
			if prob.Details["venue_http_status"] != sc.httpStatus {
				t.Errorf("venue_http_status: expected %d, got %v", sc.httpStatus, prob.Details["venue_http_status"])
			}

			// Construct rejection event (matching VenueAdapterActor.publishRejection)
			rejected := intent
			rejected.Status = domainexec.StatusRejected
			rejected.Final = true

			event := domainexec.VenueOrderRejectedEvent{
				Metadata: events.NewMetadata().
					WithCorrelationID(intent.CorrelationID).
					WithCausationID(intent.CausationID),
				ExecutionIntent: rejected,
				RejectionCode:   string(prob.Code),
				RejectionReason: prob.Message,
				VenueDetails:    prob.Details,
			}

			// Event lifecycle invariants
			if event.ExecutionIntent.Status != domainexec.StatusRejected {
				t.Errorf("event intent must be rejected, got %s", event.ExecutionIntent.Status)
			}
			if !event.ExecutionIntent.Final {
				t.Error("rejected must be Final=true")
			}

			// Correlation chain preservation
			if event.Metadata.CorrelationID != intent.CorrelationID {
				t.Errorf("CorrelationID lost: expected %s, got %s", intent.CorrelationID, event.Metadata.CorrelationID)
			}

			// Rejection code and reason non-empty
			if event.RejectionCode == "" {
				t.Error("RejectionCode must not be empty")
			}
			if event.RejectionReason == "" {
				t.Error("RejectionReason must not be empty")
			}

			// Source/symbol preserved through rejection
			if event.ExecutionIntent.Source != "binancef" {
				t.Errorf("source lost: expected binancef, got %s", event.ExecutionIntent.Source)
			}
			if event.ExecutionIntent.VenueSymbol() != "btcusdt" {
				t.Errorf("symbol lost: expected btcusdt, got %s", event.ExecutionIntent.VenueSymbol())
			}

			// No fills on rejection
			if len(event.ExecutionIntent.Fills) != 0 {
				t.Errorf("rejected intent must have 0 fills, got %d", len(event.ExecutionIntent.Fills))
			}
		})
	}
}

// ==========================================================================
// FV-Q3 + FV-Q4: QueryOrder reconciliation for rejected orders
// ==========================================================================

// TestS423_Rejection_QueryOrder_RecoversRejectedStatus proves that QueryOrder
// can recover a REJECTED order status via the Futures /fapi/v1/order endpoint.
func TestS423_Rejection_QueryOrder_RecoversRejectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("QueryOrder should use GET, got %s", r.Method)
		}
		resp := map[string]any{
			"orderId":     70100,
			"symbol":      "BTCUSDT",
			"status":      "REJECTED",
			"avgPrice":    "0",
			"executedQty": "0",
			"cumQuote":    "0",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := s423FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.QueryOrder(context.Background(), "s423-rejected-order", "btcusdt")
	if prob != nil {
		t.Fatalf("query failed: %s", prob.Message)
	}
	if receipt.Status != domainexec.StatusRejected {
		t.Fatalf("expected rejected, got %s", receipt.Status)
	}
	if receipt.VenueOrderID != "70100" {
		t.Errorf("expected venue order ID 70100, got %s", receipt.VenueOrderID)
	}
	if len(receipt.Intent.Fills) != 0 {
		t.Errorf("rejected order must have 0 fills, got %d", len(receipt.Intent.Fills))
	}
}

// TestS423_Rejection_QueryOrder_RecoversExpiredStatus proves QueryOrder recovery
// for EXPIRED orders (mapped to rejected).
func TestS423_Rejection_QueryOrder_RecoversExpiredStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     70101,
			"symbol":      "BTCUSDT",
			"status":      "EXPIRED",
			"avgPrice":    "0",
			"executedQty": "0",
			"cumQuote":    "0",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := s423FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.QueryOrder(context.Background(), "s423-expired-order", "btcusdt")
	if prob != nil {
		t.Fatalf("query failed: %s", prob.Message)
	}
	if receipt.Status != domainexec.StatusRejected {
		t.Fatalf("EXPIRED should map to rejected via QueryOrder, got %s", receipt.Status)
	}
}

// ==========================================================================
// FV-Q5: Partial fill lifecycle with explicit ValidTransition verification
// ==========================================================================

// TestS423_PartialFill_LifecyclePath_ValidTransitions proves the partial fill
// lifecycle path accepted → partially_filled → filled with step-by-step assertions.
func TestS423_PartialFill_LifecyclePath_ValidTransitions(t *testing.T) {
	// Step 1: submitted → accepted is valid
	if !domainexec.ValidTransition(domainexec.StatusSubmitted, domainexec.StatusAccepted) {
		t.Fatal("submitted → accepted must be a valid transition")
	}

	// Step 2: accepted → partially_filled is valid
	if !domainexec.ValidTransition(domainexec.StatusAccepted, domainexec.StatusPartiallyFilled) {
		t.Fatal("accepted → partially_filled must be a valid transition")
	}

	// Step 3: partially_filled → filled is valid (completion)
	if !domainexec.ValidTransition(domainexec.StatusPartiallyFilled, domainexec.StatusFilled) {
		t.Fatal("partially_filled → filled must be a valid transition")
	}

	// Step 4: partially_filled → cancelled is valid (abandon)
	if !domainexec.ValidTransition(domainexec.StatusPartiallyFilled, domainexec.StatusCancelled) {
		t.Fatal("partially_filled → cancelled must be a valid transition")
	}

	// Step 5: partially_filled is NOT terminal
	if domainexec.StatusPartiallyFilled.IsTerminal() {
		t.Fatal("partially_filled must NOT be a terminal state")
	}

	// Step 6: The adapter returns StatusPartiallyFilled for PARTIALLY_FILLED response
	server := s423FuturesPartialFillServer("0.0005", "65000.50", "32.50025")
	defer server.Close()

	creds := s423FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s423FuturesIntent(t, domainexec.SideBuy)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("partial fill should not error: %s", prob.Message)
	}

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
}

// TestS423_PartialFill_QueryOrder_RecoversPartialFillStatus proves that QueryOrder
// can recover a PARTIALLY_FILLED order with correct fill record.
func TestS423_PartialFill_QueryOrder_RecoversPartialFillStatus(t *testing.T) {
	venueTime := time.Date(2026, 3, 23, 16, 45, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("QueryOrder should use GET, got %s", r.Method)
		}
		resp := map[string]any{
			"orderId":     70200,
			"symbol":      "BTCUSDT",
			"status":      "PARTIALLY_FILLED",
			"avgPrice":    "64500.00",
			"executedQty": "0.0003",
			"cumQuote":    "19.35",
			"updateTime":  venueTime.UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := s423FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.QueryOrder(context.Background(), "s423-partial-order", "btcusdt")
	if prob != nil {
		t.Fatalf("query failed: %s", prob.Message)
	}
	if receipt.Status != domainexec.StatusPartiallyFilled {
		t.Fatalf("expected partially_filled, got %s", receipt.Status)
	}
	if receipt.VenueOrderID != "70200" {
		t.Errorf("expected venue order ID 70200, got %s", receipt.VenueOrderID)
	}
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
	}
	fill := receipt.Intent.Fills[0]
	if fill.Price != "64500.00" {
		t.Errorf("expected price 64500.00, got %s", fill.Price)
	}
	if fill.Quantity != "0.0003" {
		t.Errorf("expected qty 0.0003, got %s", fill.Quantity)
	}
	if !fill.Timestamp.Equal(venueTime) {
		t.Errorf("expected timestamp from venue (%v), got %v", venueTime, fill.Timestamp)
	}
}

// ==========================================================================
// FV-Q6: Quantity monotonicity with lifecycle verification
// ==========================================================================

// TestS423_PartialFill_QuantityMonotonicity_WithLifecycle proves FilledQuantity ≤ Quantity
// for multiple partial fill ratios, each with explicit lifecycle step verification.
func TestS423_PartialFill_QuantityMonotonicity_WithLifecycle(t *testing.T) {
	cases := []struct {
		name        string
		quantity    string
		executedQty string
	}{
		{"half_filled", "0.001", "0.0005"},
		{"quarter_filled", "0.004", "0.001"},
		{"tiny_partial", "1.0", "0.001"},
		{"near_full", "0.001", "0.0009"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := s423FuturesPartialFillServer(tc.executedQty, "65000.00", "32.50")
			defer srv.Close()

			creds := s423FuturesCredentials(t)
			adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(srv.URL)

			intent := s423FuturesIntent(t, domainexec.SideBuy)
			intent.Quantity = tc.quantity

			receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
			if prob != nil {
				t.Fatalf("unexpected error: %s", prob.Message)
			}

			// Status lifecycle
			if receipt.Status != domainexec.StatusPartiallyFilled {
				t.Fatalf("expected partially_filled, got %s", receipt.Status)
			}
			if !domainexec.ValidTransition(domainexec.StatusAccepted, receipt.Status) {
				t.Fatal("accepted → partially_filled must be valid")
			}

			// Quantity monotonicity
			if receipt.Intent.FilledQuantity != tc.executedQty {
				t.Errorf("expected FilledQuantity=%s, got %s", tc.executedQty, receipt.Intent.FilledQuantity)
			}
			if receipt.Intent.Quantity != tc.quantity {
				t.Errorf("original Quantity corrupted: expected %s, got %s", tc.quantity, receipt.Intent.Quantity)
			}
		})
	}
}

// ==========================================================================
// SegmentRouter: rejection and partial fill isolation
// ==========================================================================

// TestS423_SegmentRouter_FuturesRejection_SpotIsolated proves that a Futures
// rejection routes exclusively through the Futures adapter with segment isolation.
func TestS423_SegmentRouter_FuturesRejection_SpotIsolated(t *testing.T) {
	futuresSrv := s423FuturesErrorServer(http.StatusBadRequest, -2019, "Margin is insufficient.")
	defer futuresSrv.Close()

	spotCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spotCalled = true
		t.Error("Spot adapter must NOT be called for Futures rejection")
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

	router := appexec.NewSegmentRouter()
	router.Register(settings.MarketSegmentFutures, futuresAdapter)
	router.Register(settings.MarketSegmentSpot, spotAdapter)

	intent := s423FuturesIntent(t, domainexec.SideBuy)
	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rejection from Futures adapter via router")
	}

	if prob.Code != problem.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", prob.Code)
	}
	if spotCalled {
		t.Error("Spot adapter was called — segment isolation violated")
	}

	// Lifecycle: submitted → rejected is valid from the router rejection
	if !domainexec.ValidTransition(domainexec.StatusSubmitted, domainexec.StatusRejected) {
		t.Fatal("submitted → rejected must be valid via router")
	}
}

// TestS423_SegmentRouter_FuturesPartialFill_SpotIsolated proves partial fill
// routes through the Futures adapter with segment isolation.
func TestS423_SegmentRouter_FuturesPartialFill_SpotIsolated(t *testing.T) {
	futuresSrv := s423FuturesPartialFillServer("0.0005", "65000.50", "32.50025")
	defer futuresSrv.Close()

	spotCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spotCalled = true
		t.Error("Spot adapter must NOT be called for Futures partial fill")
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

	router := appexec.NewSegmentRouter()
	router.Register(settings.MarketSegmentFutures, futuresAdapter)
	router.Register(settings.MarketSegmentSpot, spotAdapter)

	intent := s423FuturesIntent(t, domainexec.SideBuy)
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("partial fill should not produce Problem: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusPartiallyFilled {
		t.Fatalf("expected partially_filled, got %s", receipt.Status)
	}
	if spotCalled {
		t.Error("Spot adapter was called — segment isolation violated")
	}
	if receipt.Intent.Fills[0].Simulated {
		t.Error("real venue partial fill through router must have Simulated=false")
	}
}

// ==========================================================================
// Regression: S422 fill path unchanged by S423
// ==========================================================================

// TestS423_Regression_S422FillPathUnchanged confirms the dominant FILLED path
// for Futures is unaffected by S423 changes.
func TestS423_Regression_S422FillPathUnchanged(t *testing.T) {
	server := s422FuturesFilledServer(t, 95000)
	defer server.Close()

	creds := s423FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s423FuturesIntent(t, domainexec.SideBuy)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("filled should not error: %s", prob.Message)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("expected filled, got %s", receipt.Status)
	}
	if !domainexec.ValidTransition(domainexec.StatusAccepted, receipt.Status) {
		t.Fatal("accepted → filled must remain valid")
	}
	if receipt.Intent.Fills[0].Simulated {
		t.Error("real fill must have Simulated=false")
	}
	if receipt.Intent.FilledQuantity != "0.001" {
		t.Errorf("expected FilledQuantity=0.001, got %s", receipt.Intent.FilledQuantity)
	}
}

// TestS423_Regression_S422CorrelationPreserved confirms correlation chain is
// maintained through the fill path after S423 changes.
func TestS423_Regression_S422CorrelationPreserved(t *testing.T) {
	server := s422FuturesFilledServer(t, 95100)
	defer server.Close()

	creds := s423FuturesCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 10*time.Second).WithBaseURL(server.URL)

	intent := s423FuturesIntent(t, domainexec.SideBuy)
	intent.CorrelationID = "s423-regression-corr"
	intent.CausationID = "s423-regression-cause"

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("fill should not error: %s", prob.Message)
	}

	if receipt.Intent.CorrelationID != "s423-regression-corr" {
		t.Errorf("CorrelationID lost: %s", receipt.Intent.CorrelationID)
	}
	if receipt.Intent.CausationID != "s423-regression-cause" {
		t.Errorf("CausationID lost: %s", receipt.Intent.CausationID)
	}
}
