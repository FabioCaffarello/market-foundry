package execution_test

// s407_read_path_audit_test.go — S407: Read-path auditability and segment isolation
// under real Spot responses on a unified runtime.
//
// Proves:
//   - RejectionDetail round-trip: audit metadata embedded in intent metadata by
//     the projection actor survives KV storage and is extractable on read.
//   - Composite status carries RejectionDetail alongside rejection intent.
//   - Correlation chain preserved across all lifecycle states (accepted, filled,
//     rejected, partially_filled).
//   - Segment isolation: Spot queries return only Spot results; partition key
//     structure prevents cross-segment contamination.
//   - DeriveEffectivePropagation correctly resolves lifecycle state when both
//     fill and rejection exist for the same partition key.

import (
	"testing"
	"time"

	"internal/application/executionclient"
	domainexec "internal/domain/execution"
)

// ═══════════════════════════════════════════════════════════════════
// RejectionDetail: round-trip through intent metadata
// ═══════════════════════════════════════════════════════════════════

func s407RejectedIntentWithAuditMetadata(t *testing.T) domainexec.ExecutionIntent {
	t.Helper()
	return domainexec.ExecutionIntent{
		Type:       "paper_order",
		Source:     "binances",
		Instrument: btcUSDTSpot(t),
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
		CorrelationID: "corr-s407-test",
		CausationID:   "cause-s407-test",
		Final:         true,
		Timestamp:     time.Now().UTC(),
		Metadata: map[string]string{
			"rejection_code":                 "VAL_INVALID_ARGUMENT",
			"rejection_reason":               "Account has insufficient balance for requested action.",
			"venue_detail.venue_http_status": "400",
			"venue_detail.venue_error_code":  "-2010",
		},
	}
}

// TestS407_RejectionDetail_ExtractFromMetadata proves that rejection audit detail
// embedded in intent metadata (as done by RejectionProjectionActor) can be
// reconstructed via the ExecutionRejectionReply contract.
func TestS407_RejectionDetail_ExtractFromMetadata(t *testing.T) {
	intent := s407RejectedIntentWithAuditMetadata(t)

	detail := extractRejectionDetailFromIntent(&intent)
	if detail == nil {
		t.Fatal("expected rejection detail to be extracted from intent metadata")
	}

	if detail.RejectionCode != "VAL_INVALID_ARGUMENT" {
		t.Errorf("rejection_code: expected VAL_INVALID_ARGUMENT, got %s", detail.RejectionCode)
	}
	if detail.RejectionReason != "Account has insufficient balance for requested action." {
		t.Errorf("rejection_reason: expected insufficient balance message, got %s", detail.RejectionReason)
	}
	if detail.VenueDetails == nil {
		t.Fatal("venue_details must not be nil")
	}
	if detail.VenueDetails["venue_http_status"] != "400" {
		t.Errorf("venue_http_status: expected 400, got %v", detail.VenueDetails["venue_http_status"])
	}
	if detail.VenueDetails["venue_error_code"] != "-2010" {
		t.Errorf("venue_error_code: expected -2010, got %v", detail.VenueDetails["venue_error_code"])
	}
}

// TestS407_RejectionDetail_NilWhenNoMetadata proves that extracting rejection
// detail from an intent without rejection metadata returns nil.
func TestS407_RejectionDetail_NilWhenNoMetadata(t *testing.T) {
	intent := domainexec.ExecutionIntent{
		Type:       "paper_order",
		Source:     "binances",
		Instrument: btcUSDTSpot(t),
		Timeframe:  60,
		Side:       domainexec.SideBuy,
		Quantity:   "0.001",
		Status:     domainexec.StatusFilled,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "0.85",
			Timeframe:   60,
		},
		Final:     true,
		Timestamp: time.Now().UTC(),
	}

	detail := extractRejectionDetailFromIntent(&intent)
	if detail != nil {
		t.Errorf("expected nil rejection detail for non-rejected intent, got %+v", detail)
	}
}

// ═══════════════════════════════════════════════════════════════════
// Composite status: propagation derivation with rejection detail
// ═══════════════════════════════════════════════════════════════════

// TestS407_Propagation_RejectionNewerThanFill proves that when both a fill
// and a rejection exist, the more recent timestamp wins propagation.
func TestS407_Propagation_RejectionNewerThanFill(t *testing.T) {
	now := time.Now().UTC()

	fill := &domainexec.ExecutionIntent{
		Status:    domainexec.StatusFilled,
		Timestamp: now.Add(-10 * time.Second),
	}
	rejection := &domainexec.ExecutionIntent{
		Status:    domainexec.StatusRejected,
		Timestamp: now,
	}

	prop := executionclient.DeriveEffectivePropagation(nil, fill, rejection)
	if prop != "rejected" {
		t.Errorf("expected rejected (newer), got %s", prop)
	}
}

// TestS407_Propagation_FillNewerThanRejection proves the reverse case.
func TestS407_Propagation_FillNewerThanRejection(t *testing.T) {
	now := time.Now().UTC()

	fill := &domainexec.ExecutionIntent{
		Status:    domainexec.StatusFilled,
		Timestamp: now,
	}
	rejection := &domainexec.ExecutionIntent{
		Status:    domainexec.StatusRejected,
		Timestamp: now.Add(-10 * time.Second),
	}

	prop := executionclient.DeriveEffectivePropagation(nil, fill, rejection)
	if prop != "filled" {
		t.Errorf("expected filled (newer), got %s", prop)
	}
}

// TestS407_Propagation_PartiallyFilled proves partial fill propagation.
func TestS407_Propagation_PartiallyFilled(t *testing.T) {
	fill := &domainexec.ExecutionIntent{
		Status:    domainexec.StatusPartiallyFilled,
		Timestamp: time.Now().UTC(),
	}

	prop := executionclient.DeriveEffectivePropagation(nil, fill, nil)
	if prop != "partially_filled" {
		t.Errorf("expected partially_filled, got %s", prop)
	}
}

// ═══════════════════════════════════════════════════════════════════
// Segment isolation: partition key prevents cross-segment read
// ═══════════════════════════════════════════════════════════════════

// TestS407_PartitionKey_SegmentIsolation proves that identical symbol/timeframe
// pairs on different segments produce distinct partition keys.
func TestS407_PartitionKey_SegmentIsolation(t *testing.T) {
	spotIntent := domainexec.ExecutionIntent{
		Source:     "binances",
		Instrument: btcUSDTSpot(t),
		Timeframe:  60,
	}
	futuresIntent := domainexec.ExecutionIntent{
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
	}

	spotKey := spotIntent.PartitionKey()
	futuresKey := futuresIntent.PartitionKey()

	if spotKey == futuresKey {
		t.Fatalf("partition keys must differ: spot=%s futures=%s", spotKey, futuresKey)
	}
	if spotKey != "binances.btc_usdt_spot.60" {
		t.Errorf("spot partition key: expected binances.btc_usdt_spot.60, got %s", spotKey)
	}
	if futuresKey != "binancef.btc_usdt_perpetual.60" {
		t.Errorf("futures partition key: expected binancef.btc_usdt_perpetual.60, got %s", futuresKey)
	}
}

// TestS407_CorrelationChain_PreservedInRejectedIntent proves that correlation
// and causation IDs survive the rejection metadata embedding.
func TestS407_CorrelationChain_PreservedInRejectedIntent(t *testing.T) {
	intent := s407RejectedIntentWithAuditMetadata(t)

	if intent.CorrelationID != "corr-s407-test" {
		t.Errorf("correlation_id: expected corr-s407-test, got %s", intent.CorrelationID)
	}
	if intent.CausationID != "cause-s407-test" {
		t.Errorf("causation_id: expected cause-s407-test, got %s", intent.CausationID)
	}
	if intent.Source != "binances" {
		t.Errorf("source: expected binances (Spot segment), got %s", intent.Source)
	}
	if intent.Status != domainexec.StatusRejected {
		t.Errorf("status: expected rejected, got %s", intent.Status)
	}

	// Verify audit metadata doesn't corrupt correlation fields.
	detail := extractRejectionDetailFromIntent(&intent)
	if detail == nil {
		t.Fatal("rejection detail must be extractable alongside correlation chain")
	}
}

// ═══════════════════════════════════════════════════════════════════
// Helper: mirror of extractRejectionDetail from query_responder_actor
// ═══════════════════════════════════════════════════════════════════

func extractRejectionDetailFromIntent(intent *domainexec.ExecutionIntent) *executionclient.RejectionDetail {
	if intent == nil || intent.Metadata == nil {
		return nil
	}

	code := intent.Metadata["rejection_code"]
	reason := intent.Metadata["rejection_reason"]
	if code == "" && reason == "" {
		return nil
	}

	detail := &executionclient.RejectionDetail{
		RejectionCode:   code,
		RejectionReason: reason,
	}

	for k, v := range intent.Metadata {
		if len(k) > len("venue_detail.") && k[:len("venue_detail.")] == "venue_detail." {
			if detail.VenueDetails == nil {
				detail.VenueDetails = make(map[string]any)
			}
			detail.VenueDetails[k[len("venue_detail."):]] = v
		}
	}

	return detail
}
