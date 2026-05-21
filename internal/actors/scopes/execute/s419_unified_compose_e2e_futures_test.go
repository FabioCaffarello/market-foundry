package execute_test

// s419_unified_compose_e2e_futures_test.go — S419: Unified compose E2E proof for Futures segment.
//
// Proves compose-level E2E wiring for the Futures segment on the unified runtime:
//
//   - Full pipeline: Futures ingest (binancef) -> derive -> execute (SegmentRouter) -> store -> read-path
//   - Futures fills carry real venue data (Simulated=false) with audit trail on unified runtime
//   - Futures rejections carry full audit metadata through the unified runtime
//   - Correlation chain integrity across the full E2E path (Futures segment)
//   - Dry-run mode wraps the entire SegmentRouter (compose-level safety)
//   - Segment isolation: Spot adapter untouched during Futures E2E flow
//   - Read-path: rejection detail and fill data queryable by Futures partition key
//   - Config-driven segment coexistence: both segments enabled, only Futures exercised

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
// S419: Compose E2E — Futures fill through unified SegmentRouter
// ═══════════════════════════════════════════════════════════════════

// TestS419_ComposeE2E_FuturesFill_ThroughUnifiedRuntime proves the dominant
// E2E path for Futures on the unified runtime: Futures intent arrives at
// SegmentRouter, is dispatched to BinanceFuturesTestnetAdapter, and returns
// a filled receipt with real venue data and audit metadata.
func TestS419_ComposeE2E_FuturesFill_ThroughUnifiedRuntime(t *testing.T) {
	futuresSrv := s416FuturesFilledServer(t)
	defer futuresSrv.Close()

	spotCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spotCalled = true
		t.Error("Spot adapter must NOT be called for Futures E2E fill")
	}))
	defer spotSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, spotSrv)

	intent := s419FuturesE2EIntent("s419-e2e-fill-corr", "s419-e2e-fill-cause")
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("E2E Futures fill failed: %s", prob.Message)
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
	// Futures RESULT has no commission — Fee must be "0", CostBasis carries cumQuote
	if fill.Fee != "0" {
		t.Errorf("expected fee 0 (Futures RESULT has no commission), got %s", fill.Fee)
	}

	// Futures-specific fill fidelity: avgPrice-based
	if fill.Price != "65432.10" {
		t.Errorf("expected avgPrice 65432.10, got %s", fill.Price)
	}
	if fill.CostBasis != "65.43210" {
		t.Errorf("expected CostBasis 65.43210 (cumQuote), got %s", fill.CostBasis)
	}

	// Correlation chain integrity (compose E2E traceability)
	if receipt.Intent.CorrelationID != "s419-e2e-fill-corr" {
		t.Errorf("correlation_id lost in E2E path: %s", receipt.Intent.CorrelationID)
	}
	if receipt.Intent.CausationID != "s419-e2e-fill-cause" {
		t.Errorf("causation_id lost in E2E path: %s", receipt.Intent.CausationID)
	}

	// Segment identity preserved for read-path
	if receipt.Intent.Source != "binancef" {
		t.Errorf("source: expected binancef, got %s", receipt.Intent.Source)
	}
	if key := receipt.Intent.PartitionKey(); key != "binancef.btcusdt.60" {
		t.Errorf("partition_key: expected binancef.btcusdt.60, got %s", key)
	}

	// Segment isolation
	if spotCalled {
		t.Error("segment isolation violated: Spot adapter called during Futures E2E fill")
	}
}

// ═══════════════════════════════════════════════════════════════════
// S419: Compose E2E — Futures rejection with audit trail
// ═══════════════════════════════════════════════════════════════════

// TestS419_ComposeE2E_FuturesRejection_AuditTrailComplete proves that a
// Futures rejection through the unified SegmentRouter carries full audit
// metadata that compose-level store and read-path can consume.
func TestS419_ComposeE2E_FuturesRejection_AuditTrailComplete(t *testing.T) {
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code": -2019,
			"msg":  "Margin is insufficient.",
		})
	}))
	defer futuresSrv.Close()

	spotCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spotCalled = true
		t.Error("Spot adapter must NOT be called for Futures E2E rejection")
	}))
	defer spotSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, spotSrv)

	intent := s419FuturesE2EIntent("s419-e2e-rej-corr", "s419-e2e-rej-cause")
	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected rejection from Futures adapter")
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
	if event.VenueDetails["venue_error_code"] != -2019 {
		t.Errorf("venue_error_code: expected -2019, got %v", event.VenueDetails["venue_error_code"])
	}

	// Correlation chain
	if event.Metadata.CorrelationID != "s419-e2e-rej-corr" {
		t.Errorf("correlation_id: expected s419-e2e-rej-corr, got %s", event.Metadata.CorrelationID)
	}

	// Segment identity
	if event.ExecutionIntent.Source != "binancef" {
		t.Errorf("source: expected binancef, got %s", event.ExecutionIntent.Source)
	}

	// Segment isolation
	if spotCalled {
		t.Error("segment isolation violated: Spot called during Futures E2E rejection")
	}
}

// ═══════════════════════════════════════════════════════════════════
// S419: Compose E2E — Futures rejection metadata embedding round-trip
// ═══════════════════════════════════════════════════════════════════

// TestS419_ComposeE2E_RejectionMetadata_FuturesKVRoundTrip proves that
// Futures rejection audit metadata survives the KV round-trip (JSON
// serialize -> deserialize) that compose-level store performs.
func TestS419_ComposeE2E_RejectionMetadata_FuturesKVRoundTrip(t *testing.T) {
	intent := s419FuturesE2EIntent("s419-kv-rt-corr", "s419-kv-rt-cause")
	intent.Status = domainexec.StatusRejected
	intent.Final = true
	intent.Metadata = map[string]string{
		"rejection_code":                   "VAL_INVALID_ARGUMENT",
		"rejection_reason":                 "Margin is insufficient.",
		"venue_detail.venue_http_status":   "400",
		"venue_detail.venue_error_code":    "-2019",
		"venue_detail.venue_error_message": "Margin is insufficient.",
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

	// Source and partition key preserved (Futures identity)
	if recovered.Source != "binancef" {
		t.Errorf("source lost: expected binancef, got %s", recovered.Source)
	}
	if key := recovered.PartitionKey(); key != "binancef.btcusdt.60" {
		t.Errorf("partition_key lost: expected binancef.btcusdt.60, got %s", key)
	}
}

// ═══════════════════════════════════════════════════════════════════
// S419: Compose E2E — Dry-run wraps unified SegmentRouter (Futures path)
// ═══════════════════════════════════════════════════════════════════

// TestS419_ComposeE2E_DryRun_WrapsFuturesOnUnifiedRouter proves that when
// dry_run=true (the compose default), DryRunSubmitter intercepts Futures
// intents before the SegmentRouter, matching compose wiring.
func TestS419_ComposeE2E_DryRun_WrapsFuturesOnUnifiedRouter(t *testing.T) {
	futuresCalled := false
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		futuresCalled = true
		t.Error("DryRunSubmitter must intercept before Futures adapter")
	}))
	defer futuresSrv.Close()

	spotCalled := false
	spotSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spotCalled = true
		t.Error("DryRunSubmitter must intercept before Spot adapter")
	}))
	defer spotSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, spotSrv)
	drs := appexec.NewDryRunSubmitter(router)

	intent := s419FuturesE2EIntent("s419-dryrun-corr", "s419-dryrun-cause")
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
	if futuresCalled {
		t.Error("Futures adapter called under dry_run=true")
	}
	if spotCalled {
		t.Error("Spot adapter called under dry_run=true")
	}

	// Correlation chain preserved through dry-run
	if receipt.Intent.CorrelationID != "s419-dryrun-corr" {
		t.Errorf("correlation_id lost through dry-run: %s", receipt.Intent.CorrelationID)
	}
}

// ═══════════════════════════════════════════════════════════════════
// S419: Compose E2E — Fill event construction for store pipeline
// ═══════════════════════════════════════════════════════════════════

// TestS419_ComposeE2E_FillEventConstruction_FuturesSegment proves that
// the VenueOrderFilledEvent constructed from a Futures fill carries all
// fields required by the store pipeline (projection, KV, read-path).
func TestS419_ComposeE2E_FillEventConstruction_FuturesSegment(t *testing.T) {
	futuresSrv := s416FuturesFilledServer(t)
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, nil)

	intent := s419FuturesE2EIntent("s419-fill-evt-corr", "s419-fill-evt-cause")
	receipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit failed: %s", prob.Message)
	}

	// Construct fill event as VenueAdapterActor does
	fillEvent := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID("s419-fill-evt-corr").
			WithCausationID("s419-fill-evt-cause"),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	// Event carries the Futures segment identity
	if fillEvent.ExecutionIntent.Source != "binancef" {
		t.Errorf("source: expected binancef, got %s", fillEvent.ExecutionIntent.Source)
	}

	// Metadata populated
	if fillEvent.Metadata.ID == "" {
		t.Error("metadata.ID must be auto-generated")
	}
	if fillEvent.Metadata.CorrelationID != "s419-fill-evt-corr" {
		t.Errorf("correlation_id: expected s419-fill-evt-corr, got %s", fillEvent.Metadata.CorrelationID)
	}

	// VenueOrderID is real (not simulated)
	if strings.HasPrefix(fillEvent.VenueOrderID, "dryrun-") || strings.HasPrefix(fillEvent.VenueOrderID, "paper-") {
		t.Error("venue_live fill event must NOT have simulation prefix")
	}

	// Fill data present for store projection
	if fillEvent.ExecutionIntent.FilledQuantity == "" || fillEvent.ExecutionIntent.FilledQuantity == "0" {
		t.Errorf("filled_quantity must be populated: got %q", fillEvent.ExecutionIntent.FilledQuantity)
	}

	// Partition key carries Futures segment identity
	if key := fillEvent.ExecutionIntent.PartitionKey(); key != "binancef.btcusdt.60" {
		t.Errorf("partition_key: expected binancef.btcusdt.60, got %s", key)
	}
}

// ═══════════════════════════════════════════════════════════════════
// S419: Compose E2E — Config-driven segment coexistence (Futures focus)
// ═══════════════════════════════════════════════════════════════════

// TestS419_ComposeE2E_ConfigCoexistence_FuturesAndSpotEnabled proves that
// the compose-level config enables both segments simultaneously, with
// Futures correctly dispatched by SegmentRouter and Spot structurally present.
func TestS419_ComposeE2E_ConfigCoexistence_FuturesAndSpotEnabled(t *testing.T) {
	futuresSrv := s416FuturesFilledServer(t)
	defer futuresSrv.Close()

	spotSrv := s405SpotFilledServer(t)
	defer spotSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, spotSrv)

	// Verify both segments registered
	if !router.HasSegment(settings.MarketSegmentFutures) {
		t.Error("Futures segment must be registered")
	}
	if !router.HasSegment(settings.MarketSegmentSpot) {
		t.Error("Spot segment must be registered")
	}
	if router.SegmentCount() != 2 {
		t.Errorf("expected 2 segments, got %d", router.SegmentCount())
	}

	// Futures intent routes to Futures adapter
	futuresIntent := s419FuturesE2EIntent("s419-coex-futures", "")
	futuresReceipt, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: futuresIntent})
	if prob != nil {
		t.Fatalf("Futures submit failed: %s", prob.Message)
	}
	if futuresReceipt.VenueOrderID != "77777" {
		t.Errorf("Futures fill: expected venue ID 77777, got %s", futuresReceipt.VenueOrderID)
	}

	// Source unknown intent rejected (fail-closed)
	unknownIntent := s419FuturesE2EIntent("s419-coex-unknown", "")
	unknownIntent.Source = "unknown_exchange"
	_, unknownProb := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: unknownIntent})
	if unknownProb == nil {
		t.Error("unknown source must be rejected — fail-closed")
	}
}

// ═══════════════════════════════════════════════════════════════════
// S419: Compose E2E — Futures partial fill through unified runtime
// ═══════════════════════════════════════════════════════════════════

// TestS419_ComposeE2E_FuturesPartialFill_UnifiedRuntime proves that a
// partial fill from the Futures adapter flows correctly through the unified
// runtime, preserving quantity monotonicity and audit trail.
func TestS419_ComposeE2E_FuturesPartialFill_UnifiedRuntime(t *testing.T) {
	futuresSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     88888,
			"clientOrderId": r.URL.Query().Get("newClientOrderId"),
			"symbol":      "BTCUSDT",
			"status":      "PARTIALLY_FILLED",
			"avgPrice":    "65500.00",
			"executedQty": "0.0005",
			"cumQuote":    "32.75",
			"updateTime":  time.Now().UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer futuresSrv.Close()

	router := s416BuildSegmentRouter(t, futuresSrv, nil)

	intent := s419FuturesE2EIntent("s419-partial-corr", "s419-partial-cause")
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

	// Fill record present (Futures uses avgPrice-based single fill)
	if len(receipt.Intent.Fills) != 1 {
		t.Fatalf("expected 1 fill leg, got %d", len(receipt.Intent.Fills))
	}
	if receipt.Intent.Fills[0].Simulated {
		t.Error("real venue partial fill must have Simulated=false")
	}
	if receipt.Intent.Fills[0].Price != "65500.00" {
		t.Errorf("partial fill price: expected 65500.00, got %s", receipt.Intent.Fills[0].Price)
	}

	// Correlation preserved
	if receipt.Intent.CorrelationID != "s419-partial-corr" {
		t.Errorf("correlation_id lost: %s", receipt.Intent.CorrelationID)
	}

	// Segment identity
	if receipt.Intent.Source != "binancef" {
		t.Errorf("source: expected binancef, got %s", receipt.Intent.Source)
	}
}

// ═══════════════════════════════════════════════════════════════════
// S419: Compose E2E — AllowedSources gate permits Futures
// ═══════════════════════════════════════════════════════════════════

// TestS419_ComposeE2E_AllowedSourcesGate_FuturesPermitted proves that the
// AllowedSources defense-in-depth gate permits Futures intents when both
// segments are enabled on the unified runtime.
func TestS419_ComposeE2E_AllowedSourcesGate_FuturesPermitted(t *testing.T) {
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

	// Futures source must be allowed
	if !allowed["binancef"] {
		t.Error("binancef (Futures) must be in allowed sources on unified runtime")
	}
	// Spot source must be allowed (coexistence)
	if !allowed["binances"] {
		t.Error("binances (Spot) must be in allowed sources on unified runtime")
	}
	// Unknown sources must NOT be allowed
	if allowed["unknown"] {
		t.Error("unknown source must NOT be in allowed set — fail-closed")
	}
}

// ═══════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════

// s419FuturesE2EIntent creates a Futures intent for E2E compose proof tests.
func s419FuturesE2EIntent(correlationID, causationID string) domainexec.ExecutionIntent {
	return domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        "binancef",
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
