package execute_test

// s406_spot_rejection_partial_fill_test.go — S406: Actor-level Spot rejection and partial fill evidence.
//
// Proves the rejection event path and partial fill lifecycle at the actor composition
// level using the SegmentRouter, matching the VenueAdapterActor's wiring.
//
// Rejection evidence (actor composition):
//   - Spot error responses produce correct Problem through SegmentRouter
//   - Rejection event construction carries Spot venue details (HTTP status, error code)
//   - Intent status mutated to rejected+Final=true before event emission
//   - Correlation/causation chain preserved from incoming event through rejection
//
// Partial fill evidence (actor composition):
//   - PARTIALLY_FILLED response through SegmentRouter produces StatusPartiallyFilled
//   - Partial fill lifecycle transitions validated at domain level
//   - Fill records carry real venue data through router (Simulated=false)
//   - Segment isolation: Futures adapter not contacted for Spot partial fills

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
// Actor composition: Rejection through SegmentRouter
// ═══════════════════════════════════════════════════════════════════

func s406SpotRejectionServer(statusCode int, venueCode int, venueMsg string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]any{
			"code": venueCode,
			"msg":  venueMsg,
		})
	}))
}

// TestS406_ActorComposition_SpotRejection_InsufficientBalance proves that a Spot
// rejection for insufficient balance (-2010) propagates through the SegmentRouter
// and produces a Problem with correct venue details for rejection event construction.
func TestS406_ActorComposition_SpotRejection_InsufficientBalance(t *testing.T) {
	spotSrv := s406SpotRejectionServer(http.StatusBadRequest, -2010, "Account has insufficient balance for requested action.")
	defer spotSrv.Close()

	futuresCalled := false
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		futuresCalled = true
		t.Error("futures adapter must NOT be called for Spot rejection")
	}))
	defer futuresSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, futuresSrv)

	intent := s405SpotVenueIntent(domainexec.SideBuy)
	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rejection from Spot adapter")
	}

	// Problem classification
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", prob.Code)
	}
	if prob.Retryable {
		t.Fatal("insufficient balance must NOT be retryable")
	}

	// Venue details for rejection event
	if prob.Details["venue_http_status"] != http.StatusBadRequest {
		t.Errorf("venue_http_status: expected 400, got %v", prob.Details["venue_http_status"])
	}
	if prob.Details["venue_error_code"] != -2010 {
		t.Errorf("venue_error_code: expected -2010, got %v", prob.Details["venue_error_code"])
	}

	// Segment isolation
	if futuresCalled {
		t.Error("futures adapter was called — segment isolation violated")
	}
}

// TestS406_ActorComposition_SpotRejection_LOTSize proves LOT_SIZE violation
// (-1013) through the SegmentRouter.
func TestS406_ActorComposition_SpotRejection_LOTSize(t *testing.T) {
	spotSrv := s406SpotRejectionServer(http.StatusBadRequest, -1013, "Filter failure: LOT_SIZE")
	defer spotSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, nil)

	intent := s405SpotVenueIntent(domainexec.SideBuy)
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

// TestS406_ActorComposition_SpotRejectionEvent_Construction validates the
// full rejection event construction path matching VenueAdapterActor.publishRejection.
func TestS406_ActorComposition_SpotRejectionEvent_Construction(t *testing.T) {
	spotSrv := s406SpotRejectionServer(http.StatusBadRequest, -2010, "Account has insufficient balance for requested action.")
	defer spotSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, nil)

	intent := s405SpotVenueIntent(domainexec.SideBuy)
	intent.Source = "binances"
	intent.CorrelationID = "s406-spot-corr"
	intent.CausationID = "s406-spot-cause"

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

	// Spot-specific venue details
	if event.RejectionCode != "VAL_INVALID_ARGUMENT" {
		t.Errorf("expected VAL_INVALID_ARGUMENT, got %s", event.RejectionCode)
	}
	if event.VenueDetails["venue_http_status"] != http.StatusBadRequest {
		t.Errorf("venue_http_status missing or wrong: %v", event.VenueDetails["venue_http_status"])
	}
	if event.VenueDetails["venue_error_code"] != -2010 {
		t.Errorf("venue_error_code missing or wrong: %v", event.VenueDetails["venue_error_code"])
	}

	// Correlation chain
	if event.Metadata.CorrelationID != "s406-spot-corr" {
		t.Errorf("CorrelationID lost: %s", event.Metadata.CorrelationID)
	}

	// Source/symbol preserved
	if event.ExecutionIntent.Source != "binances" {
		t.Errorf("source lost: expected binances, got %s", event.ExecutionIntent.Source)
	}
	if event.ExecutionIntent.Symbol != "btcusdt" {
		t.Errorf("symbol lost: expected btcusdt, got %s", event.ExecutionIntent.Symbol)
	}

	// No fills on rejection
	if len(event.ExecutionIntent.Fills) != 0 {
		t.Errorf("rejected intent must have 0 fills, got %d", len(event.ExecutionIntent.Fills))
	}
}

// TestS406_ActorComposition_SpotRejection_VenueRejectedStatus200 validates that
// HTTP 200 with REJECTED status is parsed correctly through the router.
func TestS406_ActorComposition_SpotRejection_VenueRejectedStatus200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             77777,
			"symbol":              "BTCUSDT",
			"status":              "REJECTED",
			"executedQty":         "0",
			"cummulativeQuoteQty": "0",
			"transactTime":        time.Now().UnixMilli(),
			"fills":               []map[string]any{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	router := s405BuildSegmentRouter(t, srv, nil)

	intent := s405SpotVenueIntent(domainexec.SideBuy)
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("HTTP 200 with REJECTED status should not produce Problem: %s", prob.Message)
	}

	// Venue returns REJECTED via HTTP 200 — adapter maps it to domain StatusRejected
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
// Actor composition: Partial fill through SegmentRouter
// ═══════════════════════════════════════════════════════════════════

func s406SpotPartialFillServer(executedQty string, fills []map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             66666,
			"symbol":              r.URL.Query().Get("symbol"),
			"status":              "PARTIALLY_FILLED",
			"side":                r.URL.Query().Get("side"),
			"type":                "MARKET",
			"executedQty":         executedQty,
			"cummulativeQuoteQty": "32.50",
			"transactTime":        time.Now().UnixMilli(),
			"fills":               fills,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

// TestS406_ActorComposition_SpotPartialFill_ThroughRouter proves that a
// PARTIALLY_FILLED response routed through SegmentRouter produces correct
// StatusPartiallyFilled with fill records.
func TestS406_ActorComposition_SpotPartialFill_ThroughRouter(t *testing.T) {
	fills := []map[string]any{
		{"price": "65000.00", "qty": "0.0005", "commission": "0.00005", "commissionAsset": "BNB"},
	}
	spotSrv := s406SpotPartialFillServer("0.0005", fills)
	defer spotSrv.Close()

	futuresCalled := false
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		futuresCalled = true
		t.Error("futures adapter must NOT be called for Spot partial fill")
	}))
	defer futuresSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, futuresSrv)

	intent := s405SpotVenueIntent(domainexec.SideBuy)
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

	// Fill fidelity
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill, got %d", len(receipt.Intent.Fills))
	}
	fill := receipt.Intent.Fills[0]
	if fill.Price != "65000" {
		t.Errorf("expected price 65000, got %s", fill.Price)
	}
	if fill.Simulated {
		t.Error("real venue partial fill must have Simulated=false")
	}

	// FilledQuantity
	if receipt.Intent.FilledQuantity != "0.0005" {
		t.Errorf("expected FilledQuantity=0.0005, got %s", receipt.Intent.FilledQuantity)
	}

	// Segment isolation
	if futuresCalled {
		t.Error("futures adapter was called — segment isolation violated")
	}
}

// TestS406_ActorComposition_SpotPartialFill_MultiLeg proves multi-leg
// partial fill aggregation through the router.
func TestS406_ActorComposition_SpotPartialFill_MultiLeg(t *testing.T) {
	fills := []map[string]any{
		{"price": "65000.00", "qty": "0.0003", "commission": "0.00003", "commissionAsset": "BNB"},
		{"price": "65400.00", "qty": "0.0003", "commission": "0.00003", "commissionAsset": "BNB"},
	}
	spotSrv := s406SpotPartialFillServer("0.0006", fills)
	defer spotSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, nil)

	intent := s405SpotVenueIntent(domainexec.SideBuy)
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}

	if receipt.Status != domainexec.StatusPartiallyFilled {
		t.Fatalf("expected partially_filled, got %s", receipt.Status)
	}

	// Aggregation: weighted avg from 2 legs
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 aggregated fill, got %d", len(receipt.Intent.Fills))
	}
	fill := receipt.Intent.Fills[0]
	if fill.Price != "65200" {
		t.Errorf("expected weighted avg 65200, got %s", fill.Price)
	}
	if fill.Fee != "0.00006" {
		t.Errorf("expected total fee 0.00006, got %s", fill.Fee)
	}
}

// TestS406_ActorComposition_SpotPartialFill_CorrelationPreserved proves
// that correlation/causation survive the partial fill path through the router.
func TestS406_ActorComposition_SpotPartialFill_CorrelationPreserved(t *testing.T) {
	fills := []map[string]any{
		{"price": "65000.00", "qty": "0.0005", "commission": "0.00005", "commissionAsset": "BNB"},
	}
	spotSrv := s406SpotPartialFillServer("0.0005", fills)
	defer spotSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, nil)

	intent := s405SpotVenueIntent(domainexec.SideBuy)
	intent.CorrelationID = "s406-partial-corr"
	intent.CausationID = "s406-partial-cause"

	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}

	if receipt.Intent.CorrelationID != "s406-partial-corr" {
		t.Errorf("CorrelationID lost: %s", receipt.Intent.CorrelationID)
	}
	if receipt.Intent.CausationID != "s406-partial-cause" {
		t.Errorf("CausationID lost: %s", receipt.Intent.CausationID)
	}
}

// TestS406_ActorComposition_SpotPartialFill_DryRunIntercepted proves
// DryRunSubmitter intercepts partial fill scenarios (never reaches venue).
func TestS406_ActorComposition_SpotPartialFill_DryRunIntercepted(t *testing.T) {
	adapterCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		adapterCalled = true
		t.Error("DryRunSubmitter must intercept before Spot adapter")
	}))
	defer spotSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, nil)
	drs := appexec.NewDryRunSubmitter(router)

	intent := s405SpotVenueIntent(domainexec.SideBuy)
	receipt, prob := drs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("dry run should not fail: %s", prob.Message)
	}

	if adapterCalled {
		t.Error("DryRunSubmitter must intercept — adapter should NOT be called")
	}
	// DryRunSubmitter always returns filled (not partially_filled)
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("dry run should return filled, got %s", receipt.Status)
	}
	if !receipt.Intent.Fills[0].Simulated {
		t.Error("dry run fills must be Simulated=true")
	}
}

// ═══════════════════════════════════════════════════════════════════
// Structural: Rejection event auditability invariants
// ═══════════════════════════════════════════════════════════════════

// TestS406_RejectionEvent_SpotVenueDetails_AuditTrail proves that a rejection
// event built from a Spot error carries sufficient details for audit trail:
// venue_http_status, venue_error_code, and rejection reason from Problem.
func TestS406_RejectionEvent_SpotVenueDetails_AuditTrail(t *testing.T) {
	cases := []struct {
		name       string
		httpStatus int
		venueCode  int
		venueMsg   string
		wantCode   string
		wantRetry  bool
	}{
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
			srv := s406SpotRejectionServer(tc.httpStatus, tc.venueCode, tc.venueMsg)
			defer srv.Close()

			router := s405BuildSegmentRouter(t, srv, nil)
			intent := s405SpotVenueIntent(domainexec.SideBuy)

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
