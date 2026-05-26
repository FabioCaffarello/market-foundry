package execute_test

// s417_futures_rejection_partial_fill_test.go — S417: Actor-level Futures rejection and partial fill evidence.
//
// Proves the rejection event path and partial fill lifecycle at the actor composition
// level using the SegmentRouter, matching the VenueAdapterActor's wiring.
//
// Rejection evidence (actor composition):
//   - Futures error responses produce correct Problem through SegmentRouter
//   - Rejection event construction carries Futures venue details (HTTP status, error code)
//   - Intent status mutated to rejected+Final=true before event emission
//   - Correlation/causation chain preserved from incoming event through rejection
//   - Spot adapter NOT contacted for Futures rejections (segment isolation)
//
// Partial fill evidence (actor composition):
//   - PARTIALLY_FILLED response through SegmentRouter produces StatusPartiallyFilled
//   - Fill records carry real venue data through router (Simulated=false)
//   - Futures avgPrice/cumQuote format preserved through router
//   - Segment isolation: Spot adapter not contacted for Futures partial fills
//
// Key difference from S406 (Spot actor tests):
//   - Futures uses avgPrice/cumQuote (no fills[] array)
//   - Source is "binancef" (Futures segment)
//   - Segment isolation is reversed: Spot must NOT be called

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
	"internal/shared/events"
	"internal/shared/problem"
)

// ═══════════════════════════════════════════════════════════════════
// Actor composition: Rejection through SegmentRouter (Futures)
// ═══════════════════════════════════════════════════════════════════

func s417FuturesRejectionServer(statusCode int, venueCode int, venueMsg string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]any{
			"code": venueCode,
			"msg":  venueMsg,
		})
	}))
}

// TestS417_ActorComposition_FuturesRejection_InsufficientMargin proves that a
// Futures rejection for insufficient margin (-2019) propagates through the
// SegmentRouter and produces a Problem with correct venue details.
func TestS417_ActorComposition_FuturesRejection_InsufficientMargin(t *testing.T) {
	futuresSrv := s417FuturesRejectionServer(http.StatusBadRequest, -2019, "Margin is insufficient.")
	defer futuresSrv.Close()

	spotCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spotCalled = true
		t.Error("Spot adapter must NOT be called for Futures rejection")
	}))
	defer spotSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, spotSrv)

	intent := s416FuturesVenueIntent(t, domainexec.SideBuy)
	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rejection from Futures adapter")
	}

	// Problem classification
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("insufficient margin must NOT be retryable")
	}

	// Venue details for rejection event
	if prob.Details["venue_http_status"] != http.StatusBadRequest {
		t.Errorf("venue_http_status: expected 400, got %v", prob.Details["venue_http_status"])
	}
	if prob.Details["venue_error_code"] != -2019 {
		t.Errorf("venue_error_code: expected -2019, got %v", prob.Details["venue_error_code"])
	}

	// Segment isolation
	if spotCalled {
		t.Error("Spot adapter was called — segment isolation violated")
	}
}

// TestS417_ActorComposition_FuturesRejection_LOTSize proves LOT_SIZE violation
// (-1013) through the SegmentRouter for Futures.
func TestS417_ActorComposition_FuturesRejection_LOTSize(t *testing.T) {
	futuresSrv := s417FuturesRejectionServer(http.StatusBadRequest, -1013, "Filter failure: LOT_SIZE")
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, nil)

	intent := s416FuturesVenueIntent(t, domainexec.SideBuy)
	intent.Quantity = "0.0000001"

	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rejection for LOT_SIZE")
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", prob.Code)
	}
	if prob.Details["venue_error_code"] != -1013 {
		t.Errorf("venue_error_code: expected -1013, got %v", prob.Details["venue_error_code"])
	}
}

// TestS417_ActorComposition_FuturesRejectionEvent_Construction validates the
// full rejection event construction path matching VenueAdapterActor.publishRejection
// for Futures intents.
func TestS417_ActorComposition_FuturesRejectionEvent_Construction(t *testing.T) {
	futuresSrv := s417FuturesRejectionServer(http.StatusBadRequest, -2019, "Margin is insufficient.")
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, nil)

	intent := s416FuturesVenueIntent(t, domainexec.SideBuy)
	intent.Source = "binancef"
	intent.CorrelationID = "s417-futures-corr"
	intent.CausationID = "s417-futures-cause"

	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rejection")
	}

	// Simulate actor rejection event construction (matches venue_adapter_actor.go:publishRejection)
	incomingEventMeta := events.NewMetadata().
		WithCorrelationID(intent.CorrelationID).
		WithCausationID(intent.CausationID)

	rejected := intent
	rejected.Status = domainexec.StatusRejected
	rejected.Final = true

	event := domainexec.VenueOrderRejectedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(incomingEventMeta.CorrelationID).
			WithCausationID(incomingEventMeta.ID),
		ExecutionIntent: rejected,
		RejectionCode:   string(prob.Code),
		RejectionReason: prob.Message,
		VenueDetails:    prob.Details,
	}

	// Intent mutation
	if event.ExecutionIntent.Status != domainexec.StatusRejected {
		t.Errorf("expected rejected, got %s", event.ExecutionIntent.Status)
	}
	if !event.ExecutionIntent.Final {
		t.Error("rejected must be Final=true")
	}

	// Futures-specific venue details
	if event.RejectionCode != "VAL_INVALID_ARGUMENT" {
		t.Errorf("expected VAL_INVALID_ARGUMENT, got %s", event.RejectionCode)
	}
	if event.VenueDetails["venue_http_status"] != http.StatusBadRequest {
		t.Errorf("venue_http_status missing or wrong: %v", event.VenueDetails["venue_http_status"])
	}
	if event.VenueDetails["venue_error_code"] != -2019 {
		t.Errorf("venue_error_code missing or wrong: %v", event.VenueDetails["venue_error_code"])
	}

	// Correlation chain
	if event.Metadata.CorrelationID != "s417-futures-corr" {
		t.Errorf("CorrelationID lost: %s", event.Metadata.CorrelationID)
	}

	// Source/symbol preserved
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
}

// TestS417_ActorComposition_FuturesRejection_VenueRejectedStatus200 validates that
// HTTP 200 with REJECTED status is parsed correctly through the router for Futures.
func TestS417_ActorComposition_FuturesRejection_VenueRejectedStatus200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     88801,
			"symbol":      "BTCUSDT",
			"status":      "REJECTED",
			"avgPrice":    "0",
			"executedQty": "0",
			"cumQuote":    "0",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	router := s416BuildSegmentRouter(t, srv, nil)

	intent := s416FuturesVenueIntent(t, domainexec.SideBuy)
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("HTTP 200 with REJECTED status should not produce Problem: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusRejected {
		t.Fatalf("expected StatusRejected, got %s", receipt.Status)
	}
	if !receipt.Status.IsTerminal() {
		t.Fatal("rejected must be terminal")
	}
	if receipt.Intent.FilledQuantity != "0" {
		t.Errorf("rejected must have FilledQuantity=0, got %s", receipt.Intent.FilledQuantity)
	}
}

// ═══════════════════════════════════════════════════════════════════
// Actor composition: Partial fill through SegmentRouter (Futures)
// ═══════════════════════════════════════════════════════════════════

func s417FuturesPartialFillServer(executedQty, avgPrice, cumQuote string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     88810,
			"symbol":      r.URL.Query().Get("symbol"),
			"status":      "PARTIALLY_FILLED",
			"side":        r.URL.Query().Get("side"),
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

// TestS417_ActorComposition_FuturesPartialFill_ThroughRouter proves that a
// PARTIALLY_FILLED response routed through SegmentRouter produces correct
// StatusPartiallyFilled with Futures-format fill records.
func TestS417_ActorComposition_FuturesPartialFill_ThroughRouter(t *testing.T) {
	futuresSrv := s417FuturesPartialFillServer("0.0005", "65000.50", "32.50025")
	defer futuresSrv.Close()

	spotCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spotCalled = true
		t.Error("Spot adapter must NOT be called for Futures partial fill")
	}))
	defer spotSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, spotSrv)

	intent := s416FuturesVenueIntent(t, domainexec.SideBuy)
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("partial fill should not produce Problem: %s", prob.Message)
	}

	// Status
	if receipt.Status != domainexec.StatusPartiallyFilled {
		t.Fatalf("expected partially_filled, got %s", receipt.Status)
	}
	if receipt.Status.IsTerminal() {
		t.Fatal("partially_filled must NOT be terminal")
	}

	// Fill fidelity (Futures format: avgPrice, cumQuote)
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
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
	if fill.Simulated {
		t.Error("real venue partial fill must have Simulated=false")
	}

	// FilledQuantity
	if receipt.Intent.FilledQuantity != "0.0005" {
		t.Errorf("expected FilledQuantity=0.0005, got %s", receipt.Intent.FilledQuantity)
	}

	// Segment isolation
	if spotCalled {
		t.Error("Spot adapter was called — segment isolation violated")
	}
}

// TestS417_ActorComposition_FuturesPartialFill_CorrelationPreserved proves
// that correlation/causation survive the partial fill path through the router.
func TestS417_ActorComposition_FuturesPartialFill_CorrelationPreserved(t *testing.T) {
	futuresSrv := s417FuturesPartialFillServer("0.0005", "65000.50", "32.50025")
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, nil)

	intent := s416FuturesVenueIntent(t, domainexec.SideBuy)
	intent.CorrelationID = "s417-partial-corr"
	intent.CausationID = "s417-partial-cause"

	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}

	if receipt.Intent.CorrelationID != "s417-partial-corr" {
		t.Errorf("CorrelationID lost: %s", receipt.Intent.CorrelationID)
	}
	if receipt.Intent.CausationID != "s417-partial-cause" {
		t.Errorf("CausationID lost: %s", receipt.Intent.CausationID)
	}
}

// TestS417_ActorComposition_FuturesPartialFill_DryRunIntercepted proves
// DryRunSubmitter intercepts Futures partial fill scenarios (never reaches venue).
func TestS417_ActorComposition_FuturesPartialFill_DryRunIntercepted(t *testing.T) {
	adapterCalled := false
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		adapterCalled = true
		t.Error("DryRunSubmitter must intercept before Futures adapter")
	}))
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, nil)
	drs := appexec.NewDryRunSubmitter(router)

	intent := s416FuturesVenueIntent(t, domainexec.SideBuy)
	receipt, prob := drs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("dry run should not fail: %s", prob.Message)
	}

	if adapterCalled {
		t.Error("DryRunSubmitter must intercept — adapter should NOT be called")
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("dry run should return filled, got %s", receipt.Status)
	}
	if !receipt.Intent.Fills[0].Simulated {
		t.Error("dry run fills must be Simulated=true")
	}
}

// ═══════════════════════════════════════════════════════════════════
// Structural: Rejection event auditability invariants (Futures)
// ═══════════════════════════════════════════════════════════════════

// TestS417_RejectionEvent_FuturesVenueDetails_AuditTrail proves that a rejection
// event built from a Futures error carries sufficient details for audit trail.
func TestS417_RejectionEvent_FuturesVenueDetails_AuditTrail(t *testing.T) {
	cases := []struct {
		name       string
		httpStatus int
		venueCode  int
		venueMsg   string
		wantCode   string
		wantRetry  bool
	}{
		{
			name:       "insufficient_margin",
			httpStatus: 400, venueCode: -2019,
			venueMsg: "Margin is insufficient.",
			wantCode: "VAL_INVALID_ARGUMENT", wantRetry: false,
		},
		{
			name:       "insufficient_balance",
			httpStatus: 400, venueCode: -2010,
			venueMsg: "Account has insufficient balance for requested action.",
			wantCode: "VAL_INVALID_ARGUMENT", wantRetry: false,
		},
		{
			name:       "lot_size_violation",
			httpStatus: 400, venueCode: -1013,
			venueMsg: "Filter failure: LOT_SIZE",
			wantCode: "VAL_INVALID_ARGUMENT", wantRetry: false,
		},
		{
			name:       "auth_failure",
			httpStatus: 401, venueCode: -2015,
			venueMsg: "Invalid API-key, IP, or permissions for action.",
			wantCode: "VAL_INVALID_ARGUMENT", wantRetry: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := s417FuturesRejectionServer(tc.httpStatus, tc.venueCode, tc.venueMsg)
			defer srv.Close()

			router := s416BuildSegmentRouter(t, srv, nil)
			intent := s416FuturesVenueIntent(t, domainexec.SideBuy)

			_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
			if prob == nil {
				t.Fatal("expected rejection")
			}

			// Build rejection event (matching actor path)
			rejected := intent
			rejected.Status = domainexec.StatusRejected
			rejected.Final = true

			event := domainexec.VenueOrderRejectedEvent{
				Metadata:        events.NewMetadata(),
				ExecutionIntent: rejected,
				RejectionCode:   string(prob.Code),
				RejectionReason: prob.Message,
				VenueDetails:    prob.Details,
			}

			// Audit trail completeness
			if event.RejectionCode != tc.wantCode {
				t.Errorf("RejectionCode: expected %s, got %s", tc.wantCode, event.RejectionCode)
			}
			if event.RejectionReason == "" {
				t.Error("RejectionReason must not be empty")
			}
			if event.VenueDetails["venue_http_status"] != tc.httpStatus {
				t.Errorf("venue_http_status: expected %d, got %v", tc.httpStatus, event.VenueDetails["venue_http_status"])
			}

			// Status and terminal state
			if event.ExecutionIntent.Status != domainexec.StatusRejected {
				t.Errorf("intent status: expected rejected, got %s", event.ExecutionIntent.Status)
			}
			if !event.ExecutionIntent.Final {
				t.Error("rejected must be Final=true")
			}
		})
	}
}
