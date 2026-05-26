package execute_test

// s418_futures_read_path_audit_test.go — S418: Futures read-path segment isolation,
// rejection audit trail, and metadata round-trip under real Futures responses
// on a unified runtime.
//
// Proves:
//   - Rejection event carries embedded audit metadata (code, reason, venue details)
//     that survives the intent metadata round-trip for queryability — Futures segment.
//   - Futures rejections through SegmentRouter produce Problem with full venue details,
//     which the VenueAdapterActor embeds into VenueOrderRejectedEvent.
//   - Segment isolation: Futures rejection events carry source=binancef; Spot adapter
//     is NOT contacted for Futures rejections.
//   - Rejection metadata embedding survives JSON round-trip (KV simulation).
//   - Filled Futures responses carry Simulated=false and avgPrice-based fill records.
//   - Unified runtime coexistence: Futures fill does not contact Spot adapter.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/events"
	"internal/shared/problem"
)

// ═══════════════════════════════════════════════════════════════════
// S418: Futures rejection audit trail through SegmentRouter
// ═══════════════════════════════════════════════════════════════════

// TestS418_RejectionAuditTrail_FuturesVenueDetails proves that a Futures
// rejection through the SegmentRouter carries full venue audit metadata.
func TestS418_RejectionAuditTrail_FuturesVenueDetails(t *testing.T) {
	spotCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spotCalled = true
		t.Error("spot adapter must NOT be called for Futures rejection")
	}))
	defer spotSrv.Close()

	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -2019,
			"msg":  "Margin is insufficient.",
		})
	}))
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, spotSrv)

	intent := s416FuturesVenueIntent(t, domainexec.SideBuy)
	intent.CorrelationID = "s418-corr-audit"
	intent.CausationID = "s418-cause-audit"

	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rejection from Futures adapter")
	}

	// Build rejection event as VenueAdapterActor would.
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

	// Verify the event carries complete audit trail.
	if event.RejectionCode != string(problem.InvalidArgument) {
		t.Errorf("rejection_code: expected %s, got %s", problem.InvalidArgument, event.RejectionCode)
	}
	if event.RejectionReason == "" {
		t.Error("rejection_reason must not be empty")
	}
	if event.VenueDetails["venue_http_status"] != http.StatusBadRequest {
		t.Errorf("venue_http_status: expected 400, got %v", event.VenueDetails["venue_http_status"])
	}
	if event.VenueDetails["venue_error_code"] != -2019 {
		t.Errorf("venue_error_code: expected -2019 (Futures margin code), got %v", event.VenueDetails["venue_error_code"])
	}

	// Correlation chain preserved.
	if event.Metadata.CorrelationID != "s418-corr-audit" {
		t.Errorf("correlation_id: expected s418-corr-audit, got %s", event.Metadata.CorrelationID)
	}
	if event.Metadata.CausationID != "s418-cause-audit" {
		t.Errorf("causation_id: expected s418-cause-audit, got %s", event.Metadata.CausationID)
	}

	// Source carries Futures segment identity.
	if event.ExecutionIntent.Source != "binancef" {
		t.Errorf("source: expected binancef (Futures), got %s", event.ExecutionIntent.Source)
	}

	// Segment isolation.
	if spotCalled {
		t.Error("spot adapter was called — segment isolation violated")
	}
}

// ═══════════════════════════════════════════════════════════════════
// S418: Futures rejection metadata embedding round-trip
// ═══════════════════════════════════════════════════════════════════

// TestS418_RejectionMetadataEmbedding_FuturesRoundTrip proves that Futures
// rejection audit metadata embedded in the intent metadata map survives JSON
// serialization and is reconstructable on the read-path.
func TestS418_RejectionMetadataEmbedding_FuturesRoundTrip(t *testing.T) {
	intent := domainexec.ExecutionIntent{
		Type:       "paper_order",
		Source:     "binancef",
		Instrument: btcUSDTPerpS379(t),
		Timeframe:  60,
		Side:       domainexec.SideBuy,
		Quantity:   "0.001",
		Status:     domainexec.StatusRejected,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "0.85",
			Timeframe:   60,
		},
		Final:     true,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]string),
	}

	// Simulate what RejectionProjectionActor does.
	rejCode := "VAL_INVALID_ARGUMENT"
	rejReason := "Margin is insufficient."
	venueDetails := map[string]any{
		"venue_http_status": 400,
		"venue_error_code":  -2019,
	}

	intent.Metadata["rejection_code"] = rejCode
	intent.Metadata["rejection_reason"] = rejReason
	for k, v := range venueDetails {
		intent.Metadata["venue_detail."+k] = fmt.Sprintf("%v", v)
	}

	// Serialize/deserialize to simulate KV round-trip.
	data, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var recovered domainexec.ExecutionIntent
	if err := json.Unmarshal(data, &recovered); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify metadata survived.
	if recovered.Metadata["rejection_code"] != rejCode {
		t.Errorf("rejection_code lost: expected %s, got %s", rejCode, recovered.Metadata["rejection_code"])
	}
	if recovered.Metadata["rejection_reason"] != rejReason {
		t.Errorf("rejection_reason lost: expected %s, got %s", rejReason, recovered.Metadata["rejection_reason"])
	}
	if recovered.Metadata["venue_detail.venue_http_status"] != "400" {
		t.Errorf("venue_detail.venue_http_status lost: got %s", recovered.Metadata["venue_detail.venue_http_status"])
	}
	if recovered.Metadata["venue_detail.venue_error_code"] != "-2019" {
		t.Errorf("venue_detail.venue_error_code lost: got %s", recovered.Metadata["venue_detail.venue_error_code"])
	}

	// Source (Futures segment) preserved.
	if recovered.Source != "binancef" {
		t.Errorf("source: expected binancef, got %s", recovered.Source)
	}
}

// ═══════════════════════════════════════════════════════════════════
// S418: Futures fill read-path — real fill carries segment and audit
// ═══════════════════════════════════════════════════════════════════

// TestS418_FillReadPath_FuturesRealFillCarriesSegmentAndAudit proves that a
// filled Futures intent preserves source (segment identity), correlation chain,
// and fill record fidelity (Simulated=false, avgPrice-based) for the read-path.
func TestS418_FillReadPath_FuturesRealFillCarriesSegmentAndAudit(t *testing.T) {
	futuresSrv := s416FuturesFilledServer(t)
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, nil)

	intent := s416FuturesVenueIntent(t, domainexec.SideBuy)
	intent.CorrelationID = "s418-fill-corr"
	intent.CausationID = "s418-fill-cause"

	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected rejection: %s", prob.Message)
	}

	// Status
	if receipt.Status != domainexec.StatusFilled {
		t.Errorf("status: expected filled, got %s", receipt.Status)
	}

	// Segment identity
	if receipt.Intent.Source != "binancef" {
		t.Errorf("source: expected binancef, got %s", receipt.Intent.Source)
	}

	// Fill record fidelity (avgPrice-based for Futures)
	if len(receipt.Intent.Fills) == 0 {
		t.Fatal("expected at least one fill record")
	}
	for i, fill := range receipt.Intent.Fills {
		if fill.Simulated {
			t.Errorf("fill[%d].Simulated must be false for real Futures venue", i)
		}
		if fill.Price == "" {
			t.Errorf("fill[%d].Price must not be empty", i)
		}
	}

	// Correlation chain
	if receipt.Intent.CorrelationID != "s418-fill-corr" {
		t.Errorf("correlation_id: expected s418-fill-corr, got %s", receipt.Intent.CorrelationID)
	}

	// Partition key carries Futures segment
	key := receipt.Intent.PartitionKey()
	if key != "binancef.btcusdt.60" {
		t.Errorf("partition_key: expected binancef.btcusdt.60, got %s", key)
	}
}

// ═══════════════════════════════════════════════════════════════════
// S418: Unified runtime — Futures fill does not contact Spot
// ═══════════════════════════════════════════════════════════════════

// TestS418_UnifiedRuntime_FuturesFillDoesNotContactSpot proves that on a unified
// runtime with both adapters, a Futures fill only contacts the Futures adapter.
func TestS418_UnifiedRuntime_FuturesFillDoesNotContactSpot(t *testing.T) {
	spotCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spotCalled = true
		t.Error("spot adapter must NOT be called for Futures fill")
	}))
	defer spotSrv.Close()

	futuresSrv := s416FuturesFilledServer(t)
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, spotSrv)

	intent := s416FuturesVenueIntent(t, domainexec.SideBuy)
	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected rejection: %s", prob.Message)
	}

	if spotCalled {
		t.Error("spot adapter was called — segment isolation violated on unified runtime")
	}
}
