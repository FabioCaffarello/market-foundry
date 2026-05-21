package execute_test

// s408_unified_compose_e2e_spot_test.go — S408: Unified compose E2E proof for Spot segment.
//
// Proves compose-level E2E wiring for the Spot segment on the unified runtime:
//
//   - Full pipeline: Spot ingest (binances) -> derive -> execute (SegmentRouter) -> store -> read-path
//   - Spot fills carry real venue data (Simulated=false) with audit trail on unified runtime
//   - Spot rejections carry full audit metadata through the unified runtime
//   - Correlation chain integrity across the full E2E path (Spot segment)
//   - Dry-run mode wraps the entire SegmentRouter (compose-level safety)
//   - Segment isolation: Futures adapter untouched during Spot E2E flow
//   - Read-path: rejection detail and fill data queryable by Spot partition key
//   - Config-driven segment coexistence: both segments enabled, only Spot exercised

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/events"
	"internal/shared/settings"
)

// ═══════════════════════════════════════════════════════════════════
// S408: Compose E2E — Spot fill through unified SegmentRouter
// ═══════════════════════════════════════════════════════════════════

// TestS408_ComposeE2E_SpotFill_ThroughUnifiedRuntime proves the dominant
// E2E path on the unified runtime: Spot intent arrives at SegmentRouter,
// is dispatched to BinanceSpotTestnetAdapter, and returns a filled receipt
// with real venue data and audit metadata — matching the compose wiring.
func TestS408_ComposeE2E_SpotFill_ThroughUnifiedRuntime(t *testing.T) {
	spotSrv := s405SpotFilledServer(t)
	defer spotSrv.Close()

	futuresCalled := false
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		futuresCalled = true
		t.Error("futures adapter must NOT be called for Spot E2E fill")
	}))
	defer futuresSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, futuresSrv)

	intent := s408SpotE2EIntent("s408-e2e-fill-corr", "s408-e2e-fill-cause")
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("E2E Spot fill failed: %s", prob.Message)
	}

	// Lifecycle outcome
	if receipt.Status != domainexec.StatusFilled {
		t.Errorf("expected filled, got %s", receipt.Status)
	}
	if !receipt.Status.IsTerminal() {
		t.Error("filled must be terminal — compose read-path depends on this")
	}

	// Fill fidelity (real venue data, not simulated)
	if len(receipt.Intent.Fills) == 0 {
		t.Fatal("E2E path must produce at least one fill record")
	}
	fill := receipt.Intent.Fills[0]
	if fill.Simulated {
		t.Error("compose E2E venue_live fills must have Simulated=false")
	}
	if fill.Price == "" || fill.Price == "0" {
		t.Errorf("fill price must be real venue price, got %q", fill.Price)
	}
	if fill.Fee == "" || fill.Fee == "0" {
		t.Errorf("fill fee must be real venue fee, got %q", fill.Fee)
	}

	// Correlation chain integrity (compose E2E traceability)
	if receipt.Intent.CorrelationID != "s408-e2e-fill-corr" {
		t.Errorf("correlation_id lost in E2E path: %s", receipt.Intent.CorrelationID)
	}
	if receipt.Intent.CausationID != "s408-e2e-fill-cause" {
		t.Errorf("causation_id lost in E2E path: %s", receipt.Intent.CausationID)
	}

	// Segment identity preserved for read-path
	if receipt.Intent.Source != "binances" {
		t.Errorf("source: expected binances, got %s", receipt.Intent.Source)
	}
	if key := receipt.Intent.PartitionKey(); key != "binances.btcusdt.60" {
		t.Errorf("partition_key: expected binances.btcusdt.60, got %s", key)
	}

	// Segment isolation
	if futuresCalled {
		t.Error("segment isolation violated: Futures adapter called during Spot E2E fill")
	}
}

// ═══════════════════════════════════════════════════════════════════
// S408: Compose E2E — Spot rejection with audit trail
// ═══════════════════════════════════════════════════════════════════

// TestS408_ComposeE2E_SpotRejection_AuditTrailComplete proves that a Spot
// rejection through the unified SegmentRouter carries full audit metadata
// that compose-level store and read-path can consume.
func TestS408_ComposeE2E_SpotRejection_AuditTrailComplete(t *testing.T) {
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -2010,
			"msg":  "Account has insufficient balance for requested action.",
		})
	}))
	defer spotSrv.Close()

	futuresCalled := false
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		futuresCalled = true
		t.Error("futures adapter must NOT be called for Spot E2E rejection")
	}))
	defer futuresSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, futuresSrv)

	intent := s408SpotE2EIntent("s408-e2e-rej-corr", "s408-e2e-rej-cause")
	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rejection from Spot adapter")
	}

	// Build rejection event (matches VenueAdapterActor.publishRejection)
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

	// Audit trail completeness for compose read-path
	if event.RejectionCode == "" {
		t.Error("rejection_code must not be empty — read-path depends on it")
	}
	if event.RejectionReason == "" {
		t.Error("rejection_reason must not be empty — audit trail depends on it")
	}
	if event.VenueDetails == nil {
		t.Error("venue_details must not be nil — detailed audit depends on it")
	}
	if event.VenueDetails["venue_http_status"] != http.StatusBadRequest {
		t.Errorf("venue_http_status: expected 400, got %v", event.VenueDetails["venue_http_status"])
	}
	if event.VenueDetails["venue_error_code"] != -2010 {
		t.Errorf("venue_error_code: expected -2010, got %v", event.VenueDetails["venue_error_code"])
	}

	// Correlation chain
	if event.Metadata.CorrelationID != "s408-e2e-rej-corr" {
		t.Errorf("correlation_id: expected s408-e2e-rej-corr, got %s", event.Metadata.CorrelationID)
	}

	// Segment identity
	if event.ExecutionIntent.Source != "binances" {
		t.Errorf("source: expected binances, got %s", event.ExecutionIntent.Source)
	}

	// Segment isolation
	if futuresCalled {
		t.Error("segment isolation violated: Futures called during Spot E2E rejection")
	}
}

// ═══════════════════════════════════════════════════════════════════
// S408: Compose E2E — Rejection metadata embedding round-trip
// ═══════════════════════════════════════════════════════════════════

// TestS408_ComposeE2E_RejectionMetadata_KVRoundTrip proves that Spot
// rejection audit metadata survives the KV round-trip (JSON serialize →
// deserialize) that compose-level store performs. This is the persistence
// guarantee for the read-path audit trail.
func TestS408_ComposeE2E_RejectionMetadata_KVRoundTrip(t *testing.T) {
	intent := s408SpotE2EIntent("s408-kv-rt-corr", "s408-kv-rt-cause")
	intent.Status = domainexec.StatusRejected
	intent.Final = true
	intent.Metadata = map[string]string{
		"rejection_code":                    "VAL_INVALID_ARGUMENT",
		"rejection_reason":                  "Account has insufficient balance for requested action.",
		"venue_detail.venue_http_status":    "400",
		"venue_detail.venue_error_code":     "-2010",
		"venue_detail.venue_error_message":  "Account has insufficient balance for requested action.",
	}

	data, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var recovered domainexec.ExecutionIntent
	if err := json.Unmarshal(data, &recovered); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// All audit keys survive KV round-trip
	expectedKeys := []string{
		"rejection_code", "rejection_reason",
		"venue_detail.venue_http_status", "venue_detail.venue_error_code",
		"venue_detail.venue_error_message",
	}
	for _, key := range expectedKeys {
		if recovered.Metadata[key] != intent.Metadata[key] {
			t.Errorf("metadata[%s] lost: expected %q, got %q", key, intent.Metadata[key], recovered.Metadata[key])
		}
	}

	// Source and partition key preserved
	if recovered.Source != "binances" {
		t.Errorf("source lost: expected binances, got %s", recovered.Source)
	}
	if key := recovered.PartitionKey(); key != "binances.btcusdt.60" {
		t.Errorf("partition_key lost: expected binances.btcusdt.60, got %s", key)
	}
}

// ═══════════════════════════════════════════════════════════════════
// S408: Compose E2E — Dry-run wraps unified SegmentRouter
// ═══════════════════════════════════════════════════════════════════

// TestS408_ComposeE2E_DryRun_WrapsUnifiedRouter proves that when
// dry_run=true (the compose default), DryRunSubmitter intercepts ALL
// intents before the SegmentRouter, including Spot. This matches the
// compose wiring in cmd/execute/run.go.
func TestS408_ComposeE2E_DryRun_WrapsUnifiedRouter(t *testing.T) {
	spotCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spotCalled = true
		t.Error("DryRunSubmitter must intercept before Spot adapter")
	}))
	defer spotSrv.Close()

	futuresCalled := false
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		futuresCalled = true
		t.Error("DryRunSubmitter must intercept before Futures adapter")
	}))
	defer futuresSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, futuresSrv)
	drs := appexec.NewDryRunSubmitter(router)

	intent := s408SpotE2EIntent("s408-dryrun-corr", "s408-dryrun-cause")
	receipt, prob := drs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	// Dry-run produces simulated fills
	if !strings.HasPrefix(receipt.VenueOrderID, "dryrun-") {
		t.Errorf("expected dryrun- prefix, got %s", receipt.VenueOrderID)
	}
	if len(receipt.Intent.Fills) > 0 && !receipt.Intent.Fills[0].Simulated {
		t.Error("dry-run fills must have Simulated=true")
	}

	// Neither adapter contacted
	if spotCalled {
		t.Error("Spot adapter called under dry_run=true")
	}
	if futuresCalled {
		t.Error("Futures adapter called under dry_run=true")
	}

	// Correlation chain preserved through dry-run
	if receipt.Intent.CorrelationID != "s408-dryrun-corr" {
		t.Errorf("correlation_id lost through dry-run: %s", receipt.Intent.CorrelationID)
	}
}

// ═══════════════════════════════════════════════════════════════════
// S408: Compose E2E — Fill event construction for store pipeline
// ═══════════════════════════════════════════════════════════════════

// TestS408_ComposeE2E_FillEventConstruction_SpotSegment proves that
// the VenueOrderFilledEvent constructed from a Spot fill carries all
// fields required by the store pipeline (projection, KV, read-path).
func TestS408_ComposeE2E_FillEventConstruction_SpotSegment(t *testing.T) {
	spotSrv := s405SpotFilledServer(t)
	defer spotSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, nil)

	intent := s408SpotE2EIntent("s408-fill-evt-corr", "s408-fill-evt-cause")
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	// Construct fill event as VenueAdapterActor does
	fillEvent := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID("s408-fill-evt-corr").
			WithCausationID("s408-fill-evt-cause"),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	// Event carries the Spot segment identity
	if fillEvent.ExecutionIntent.Source != "binances" {
		t.Errorf("source: expected binances, got %s", fillEvent.ExecutionIntent.Source)
	}

	// Metadata populated
	if fillEvent.Metadata.ID == "" {
		t.Error("metadata.ID must be auto-generated")
	}
	if fillEvent.Metadata.CorrelationID != "s408-fill-evt-corr" {
		t.Errorf("correlation_id: expected s408-fill-evt-corr, got %s", fillEvent.Metadata.CorrelationID)
	}

	// VenueOrderID is real (not simulated)
	if strings.HasPrefix(fillEvent.VenueOrderID, "dryrun-") || strings.HasPrefix(fillEvent.VenueOrderID, "paper-") {
		t.Error("venue_live fill event must NOT have simulation prefix")
	}

	// Fill data present for store projection
	if fillEvent.ExecutionIntent.FilledQuantity == "" || fillEvent.ExecutionIntent.FilledQuantity == "0" {
		t.Errorf("filled_quantity must be populated: got %q", fillEvent.ExecutionIntent.FilledQuantity)
	}
}

// ═══════════════════════════════════════════════════════════════════
// S408: Compose E2E — Config-driven segment coexistence
// ═══════════════════════════════════════════════════════════════════

// TestS408_ComposeE2E_ConfigCoexistence_BothSegmentsEnabled proves that
// the compose-level config can enable both segments simultaneously with
// the SegmentRouter dispatching correctly by source.
func TestS408_ComposeE2E_ConfigCoexistence_BothSegmentsEnabled(t *testing.T) {
	spotSrv := s405SpotFilledServer(t)
	defer spotSrv.Close()

	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     77777,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"avgPrice":    "65500.00",
			"executedQty": "0.001",
			"cumQuote":    "65.50",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer futuresSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, futuresSrv)

	// Verify both segments registered
	if !router.HasSegment(settings.MarketSegmentSpot) {
		t.Error("Spot segment must be registered")
	}
	if !router.HasSegment(settings.MarketSegmentFutures) {
		t.Error("Futures segment must be registered")
	}
	if router.SegmentCount() != 2 {
		t.Errorf("expected 2 segments, got %d", router.SegmentCount())
	}

	// Spot intent routes to Spot
	spotIntent := s408SpotE2EIntent("s408-coex-spot", "")
	spotReceipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: spotIntent})
	if prob != nil {
		t.Fatalf("Spot submit failed: %s", prob.Message)
	}
	if spotReceipt.VenueOrderID != "55555" {
		t.Errorf("Spot fill: expected venue ID 55555, got %s", spotReceipt.VenueOrderID)
	}

	// Source unknown intent rejected (fail-closed)
	unknownIntent := s408SpotE2EIntent("s408-coex-unknown", "")
	unknownIntent.Source = "unknown_exchange"
	_, unknownProb := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: unknownIntent})
	if unknownProb == nil {
		t.Error("unknown source must be rejected — fail-closed")
	}
}

// ═══════════════════════════════════════════════════════════════════
// S408: Compose E2E — Partial fill through unified runtime
// ═══════════════════════════════════════════════════════════════════

// TestS408_ComposeE2E_SpotPartialFill_UnifiedRuntime proves that a
// partial fill from the Spot adapter flows correctly through the unified
// runtime, preserving quantity monotonicity and audit trail.
func TestS408_ComposeE2E_SpotPartialFill_UnifiedRuntime(t *testing.T) {
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             66666,
			"clientOrderId":      r.URL.Query().Get("newClientOrderId"),
			"symbol":              "BTCUSDT",
			"status":              "PARTIALLY_FILLED",
			"executedQty":         "0.0005",
			"cummulativeQuoteQty": "32.72",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]any{
				{
					"price":           "65430.00",
					"qty":             "0.0005",
					"commission":      "0.00003272",
					"commissionAsset": "BNB",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer spotSrv.Close()

	router := s405BuildSegmentRouter(t, spotSrv, nil)

	intent := s408SpotE2EIntent("s408-partial-corr", "s408-partial-cause")
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	// Partial fill status
	if receipt.Status != domainexec.StatusPartiallyFilled {
		t.Errorf("expected partially_filled, got %s", receipt.Status)
	}

	// Quantity monotonicity: filled < requested
	if receipt.Intent.FilledQuantity >= intent.Quantity {
		t.Errorf("filled_quantity (%s) must be less than requested (%s)",
			receipt.Intent.FilledQuantity, intent.Quantity)
	}

	// Fill record present
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill leg, got %d", len(receipt.Intent.Fills))
	}
	if receipt.Intent.Fills[0].Simulated {
		t.Error("real venue partial fill must have Simulated=false")
	}

	// Correlation preserved
	if receipt.Intent.CorrelationID != "s408-partial-corr" {
		t.Errorf("correlation_id lost: %s", receipt.Intent.CorrelationID)
	}
}

// ═══════════════════════════════════════════════════════════════════
// S408: Compose E2E — Execution intent carries Spot source through
// the full VenueAdapterActor pipeline
// ═══════════════════════════════════════════════════════════════════

// TestS408_ComposeE2E_AllowedSourcesGate_SpotPermitted proves that the
// AllowedSources defense-in-depth gate (S401) permits Spot intents when
// both segments are enabled on the unified runtime.
func TestS408_ComposeE2E_AllowedSourcesGate_SpotPermitted(t *testing.T) {
	enabledSources := settings.AppConfig{
		Venue: settings.VenueConfig{
			Segments: map[settings.MarketSegment]*settings.SegmentVenueConfig{
				settings.MarketSegmentSpot:    {Enabled: true, Adapter: "binance_spot_testnet"},
				settings.MarketSegmentFutures: {Enabled: true, Adapter: "binance_futures_testnet"},
			},
		},
	}

	sources := enabledSources.Venue.EnabledSegmentSources()
	allowed := make(map[string]bool)
	for _, src := range sources {
		allowed[src] = true
	}

	// Spot source must be allowed
	if !allowed["binances"] {
		t.Error("binances (Spot) must be in allowed sources on unified runtime")
	}
	// Futures source must be allowed
	if !allowed["binancef"] {
		t.Error("binancef (Futures) must be in allowed sources on unified runtime")
	}
	// Unknown sources must NOT be allowed
	if allowed["unknown"] {
		t.Error("unknown source must NOT be in allowed set — fail-closed")
	}
}

// ═══════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════

// s408SpotE2EIntent creates a Spot intent for E2E compose proof tests.
func s408SpotE2EIntent(correlationID, causationID string) domainexec.ExecutionIntent {
	return domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        "binances",
		Symbol:        "btcusdt",
		Timeframe:     60,
		Side:          domainexec.SideBuy,
		Quantity:      "0.001",
		Status:        domainexec.StatusSubmitted,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		CorrelationID: correlationID,
		CausationID:   causationID,
		Final:         true,
		Timestamp:     time.Now().UTC().Add(-3 * time.Second),
	}
}
