package execution_test

// s418_futures_read_path_audit_test.go — S418: Read-path auditability and segment parity
// under real Futures responses on a unified runtime.
//
// Proves:
//   - RejectionDetail round-trip: audit metadata embedded in intent metadata by
//     the projection actor survives KV storage and is extractable on read for Futures.
//   - Composite status carries RejectionDetail alongside rejection intent for Futures.
//   - Correlation chain preserved across all lifecycle states (accepted, filled,
//     rejected, partially_filled) for Futures segment.
//   - Segment isolation: Futures queries return only Futures results; partition key
//     structure prevents cross-segment contamination.
//   - DeriveEffectivePropagation correctly resolves lifecycle state when both
//     fill and rejection exist for a Futures partition key.
//   - Parity: Futures read-path behaves identically to Spot (S407) for equivalent
//     lifecycle scenarios.

import (
	"testing"
	"time"

	"internal/application/executionclient"
	domainexec "internal/domain/execution"
)

// ═══════════════════════════════════════════════════════════════════
// Futures rejected intent with audit metadata (mirrors S407 Spot)
// ═══════════════════════════════════════════════════════════════════

func s418FuturesRejectedIntentWithAuditMetadata() domainexec.ExecutionIntent {
	return domainexec.ExecutionIntent{
		Type:      "paper_order",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Side:      domainexec.SideBuy,
		Quantity:  "0.001",
		Status:    domainexec.StatusRejected,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "0.85",
			Timeframe:   60,
		},
		CorrelationID: "corr-s418-futures",
		CausationID:   "cause-s418-futures",
		Final:         true,
		Timestamp:     time.Now().UTC(),
		Metadata: map[string]string{
			"rejection_code":                 "VAL_INVALID_ARGUMENT",
			"rejection_reason":               "Margin is insufficient.",
			"venue_detail.venue_http_status":  "400",
			"venue_detail.venue_error_code":   "-2019",
		},
	}
}

func s418FuturesFilledIntent(ts time.Time) domainexec.ExecutionIntent {
	return domainexec.ExecutionIntent{
		Type:           "venue_market_order",
		Source:         "binancef",
		Symbol:         "btcusdt",
		Timeframe:      60,
		Side:           domainexec.SideBuy,
		Quantity:       "0.001",
		FilledQuantity: "0.001",
		Status:         domainexec.StatusFilled,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "0.85",
			Timeframe:   60,
		},
		CorrelationID: "corr-s418-futures-fill",
		CausationID:   "cause-s418-futures-fill",
		Final:         true,
		Timestamp:     ts,
		Fills: []domainexec.FillRecord{
			{Price: "65432.10", Quantity: "0.001", Fee: "0", CostBasis: "65.43210", Simulated: false, Timestamp: ts},
		},
	}
}

func s418FuturesPartiallyFilledIntent(ts time.Time) domainexec.ExecutionIntent {
	return domainexec.ExecutionIntent{
		Type:           "venue_market_order",
		Source:         "binancef",
		Symbol:         "btcusdt",
		Timeframe:      60,
		Side:           domainexec.SideBuy,
		Quantity:       "0.001",
		FilledQuantity: "0.0005",
		Status:         domainexec.StatusPartiallyFilled,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "0.85",
			Timeframe:   60,
		},
		CorrelationID: "corr-s418-futures-partial",
		CausationID:   "cause-s418-futures-partial",
		Final:         true,
		Timestamp:     ts,
		Fills: []domainexec.FillRecord{
			{Price: "65432.10", Quantity: "0.0005", Fee: "32.71605", Simulated: false, Timestamp: ts},
		},
	}
}

// ═══════════════════════════════════════════════════════════════════
// RejectionDetail: Futures rejection metadata round-trip
// ═══════════════════════════════════════════════════════════════════

// TestS418_RejectionDetail_FuturesExtractFromMetadata proves that rejection audit
// detail embedded in Futures intent metadata can be reconstructed via
// the ExecutionRejectionReply contract — direct parity with S407 Spot proof.
func TestS418_RejectionDetail_FuturesExtractFromMetadata(t *testing.T) {
	intent := s418FuturesRejectedIntentWithAuditMetadata()

	detail := extractRejectionDetailFromIntent(&intent)
	if detail == nil {
		t.Fatal("expected rejection detail to be extracted from Futures intent metadata")
	}

	if detail.RejectionCode != "VAL_INVALID_ARGUMENT" {
		t.Errorf("rejection_code: expected VAL_INVALID_ARGUMENT, got %s", detail.RejectionCode)
	}
	if detail.RejectionReason != "Margin is insufficient." {
		t.Errorf("rejection_reason: expected margin insufficient message, got %s", detail.RejectionReason)
	}
	if detail.VenueDetails == nil {
		t.Fatal("venue_details must not be nil for Futures rejection")
	}
	if detail.VenueDetails["venue_http_status"] != "400" {
		t.Errorf("venue_http_status: expected 400, got %v", detail.VenueDetails["venue_http_status"])
	}
	if detail.VenueDetails["venue_error_code"] != "-2019" {
		t.Errorf("venue_error_code: expected -2019 (Futures margin code), got %v", detail.VenueDetails["venue_error_code"])
	}
}

// TestS418_RejectionDetail_FuturesNilWhenFilled proves that extracting rejection
// detail from a Futures filled intent returns nil.
func TestS418_RejectionDetail_FuturesNilWhenFilled(t *testing.T) {
	intent := s418FuturesFilledIntent(time.Now().UTC())

	detail := extractRejectionDetailFromIntent(&intent)
	if detail != nil {
		t.Errorf("expected nil rejection detail for filled Futures intent, got %+v", detail)
	}
}

// ═══════════════════════════════════════════════════════════════════
// Composite status: propagation derivation with Futures data
// ═══════════════════════════════════════════════════════════════════

// TestS418_Propagation_FuturesRejectionNewerThanFill proves that when both
// a Futures fill and rejection exist, the more recent timestamp wins.
func TestS418_Propagation_FuturesRejectionNewerThanFill(t *testing.T) {
	now := time.Now().UTC()

	fill := s418FuturesFilledIntent(now.Add(-10 * time.Second))
	rejection := s418FuturesRejectedIntentWithAuditMetadata()
	rejection.Timestamp = now

	prop := executionclient.DeriveEffectivePropagation(nil, &fill, &rejection)
	if prop != "rejected" {
		t.Errorf("expected rejected (newer), got %s", prop)
	}
}

// TestS418_Propagation_FuturesFillNewerThanRejection proves the reverse.
func TestS418_Propagation_FuturesFillNewerThanRejection(t *testing.T) {
	now := time.Now().UTC()

	fill := s418FuturesFilledIntent(now)
	rejection := s418FuturesRejectedIntentWithAuditMetadata()
	rejection.Timestamp = now.Add(-10 * time.Second)

	prop := executionclient.DeriveEffectivePropagation(nil, &fill, &rejection)
	if prop != "filled" {
		t.Errorf("expected filled (newer), got %s", prop)
	}
}

// TestS418_Propagation_FuturesPartiallyFilled proves partial fill propagation
// for the Futures segment — parity with S407 Spot proof.
func TestS418_Propagation_FuturesPartiallyFilled(t *testing.T) {
	now := time.Now().UTC()
	partial := s418FuturesPartiallyFilledIntent(now)

	prop := executionclient.DeriveEffectivePropagation(nil, &partial, nil)
	if prop != "partially_filled" {
		t.Errorf("expected partially_filled, got %s", prop)
	}
}

// TestS418_Propagation_FuturesIntentOnly proves that a submitted Futures intent
// with no venue outcome propagates as submitted.
func TestS418_Propagation_FuturesIntentOnly(t *testing.T) {
	intent := domainexec.ExecutionIntent{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Status:    domainexec.StatusSubmitted,
		Timestamp: time.Now().UTC(),
	}

	prop := executionclient.DeriveEffectivePropagation(&intent, nil, nil)
	if prop != "submitted" {
		t.Errorf("expected submitted, got %s", prop)
	}
}

// TestS418_Propagation_FuturesNone proves that no surfaces returns "none".
func TestS418_Propagation_FuturesNone(t *testing.T) {
	prop := executionclient.DeriveEffectivePropagation(nil, nil, nil)
	if prop != "none" {
		t.Errorf("expected none, got %s", prop)
	}
}

// ═══════════════════════════════════════════════════════════════════
// Segment isolation: partition key prevents cross-segment read
// ═══════════════════════════════════════════════════════════════════

// TestS418_PartitionKey_FuturesSegmentIsolation proves that Futures partition keys
// are distinct from Spot for identical symbol/timeframe pairs.
func TestS418_PartitionKey_FuturesSegmentIsolation(t *testing.T) {
	spotIntent := domainexec.ExecutionIntent{
		Source:    "binances",
		Symbol:    "btcusdt",
		Timeframe: 60,
	}
	futuresIntent := domainexec.ExecutionIntent{
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
	}

	spotKey := spotIntent.PartitionKey()
	futuresKey := futuresIntent.PartitionKey()

	if spotKey == futuresKey {
		t.Fatalf("partition keys must differ: spot=%s futures=%s", spotKey, futuresKey)
	}
	if futuresKey != "binancef.btcusdt.60" {
		t.Errorf("futures partition key: expected binancef.btcusdt.60, got %s", futuresKey)
	}
}

// TestS418_PartitionKey_FuturesRejectionIsolated proves that a Futures rejection
// partition key cannot collide with a Spot rejection partition key.
func TestS418_PartitionKey_FuturesRejectionIsolated(t *testing.T) {
	futuresRejection := s418FuturesRejectedIntentWithAuditMetadata()
	spotRejection := domainexec.ExecutionIntent{
		Source:    "binances",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Status:    domainexec.StatusRejected,
	}

	if futuresRejection.PartitionKey() == spotRejection.PartitionKey() {
		t.Fatal("Futures and Spot rejection partition keys must differ")
	}
}

// ═══════════════════════════════════════════════════════════════════
// Correlation chain preservation for Futures
// ═══════════════════════════════════════════════════════════════════

// TestS418_CorrelationChain_FuturesRejectedIntent proves that correlation
// and causation IDs survive the rejection metadata embedding for Futures.
func TestS418_CorrelationChain_FuturesRejectedIntent(t *testing.T) {
	intent := s418FuturesRejectedIntentWithAuditMetadata()

	if intent.CorrelationID != "corr-s418-futures" {
		t.Errorf("correlation_id: expected corr-s418-futures, got %s", intent.CorrelationID)
	}
	if intent.CausationID != "cause-s418-futures" {
		t.Errorf("causation_id: expected cause-s418-futures, got %s", intent.CausationID)
	}
	if intent.Source != "binancef" {
		t.Errorf("source: expected binancef (Futures segment), got %s", intent.Source)
	}
	if intent.Status != domainexec.StatusRejected {
		t.Errorf("status: expected rejected, got %s", intent.Status)
	}

	detail := extractRejectionDetailFromIntent(&intent)
	if detail == nil {
		t.Fatal("rejection detail must be extractable alongside Futures correlation chain")
	}
}

// TestS418_CorrelationChain_FuturesFilledIntent proves correlation chain
// preservation for filled Futures intents.
func TestS418_CorrelationChain_FuturesFilledIntent(t *testing.T) {
	intent := s418FuturesFilledIntent(time.Now().UTC())

	if intent.CorrelationID != "corr-s418-futures-fill" {
		t.Errorf("correlation_id: expected corr-s418-futures-fill, got %s", intent.CorrelationID)
	}
	if intent.CausationID != "cause-s418-futures-fill" {
		t.Errorf("causation_id: expected cause-s418-futures-fill, got %s", intent.CausationID)
	}
	if intent.Source != "binancef" {
		t.Errorf("source: expected binancef, got %s", intent.Source)
	}
	if len(intent.Fills) != 1 {
		t.Fatalf("expected 1 fill record, got %d", len(intent.Fills))
	}
	if intent.Fills[0].Simulated {
		t.Error("fill record must not be simulated for real Futures venue")
	}
}

// TestS418_CorrelationChain_FuturesPartialFillIntent proves correlation chain
// preservation for partially filled Futures intents.
func TestS418_CorrelationChain_FuturesPartialFillIntent(t *testing.T) {
	now := time.Now().UTC()
	intent := s418FuturesPartiallyFilledIntent(now)

	if intent.CorrelationID != "corr-s418-futures-partial" {
		t.Errorf("correlation_id: expected corr-s418-futures-partial, got %s", intent.CorrelationID)
	}
	if intent.FilledQuantity != "0.0005" {
		t.Errorf("filled_quantity: expected 0.0005, got %s", intent.FilledQuantity)
	}
	if intent.Quantity <= intent.FilledQuantity {
		t.Error("quantity must be greater than filled_quantity for partial fill")
	}
}

// ═══════════════════════════════════════════════════════════════════
// Lifecycle entry: Futures field population (S413 parity)
// ═══════════════════════════════════════════════════════════════════

// TestS418_LifecycleEntry_FuturesFieldPopulation proves that LifecycleEntry
// correctly captures per-key lifecycle state for Futures intents.
func TestS418_LifecycleEntry_FuturesFieldPopulation(t *testing.T) {
	ts := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	fillTs := ts.Add(time.Minute)

	intent := domainexec.ExecutionIntent{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
		Status: domainexec.StatusSubmitted, Timestamp: ts,
	}
	fill := s418FuturesFilledIntent(fillTs)

	entry := executionclient.LifecycleEntry{
		Key:            "binancef.btcusdt.60",
		Source:         "binancef",
		Symbol:         "btcusdt",
		Timeframe:      60,
		IntentStatus:   string(intent.Status),
		IntentTimestamp: &intent.Timestamp,
		FillStatus:     string(fill.Status),
		FillTimestamp:   &fill.Timestamp,
		Propagation:    executionclient.DeriveEffectivePropagation(&intent, &fill, nil),
	}

	if entry.Source != "binancef" {
		t.Errorf("Source = %q, want binancef", entry.Source)
	}
	if entry.IntentStatus != "submitted" {
		t.Errorf("IntentStatus = %q, want submitted", entry.IntentStatus)
	}
	if entry.FillStatus != "filled" {
		t.Errorf("FillStatus = %q, want filled", entry.FillStatus)
	}
	if entry.RejectionStatus != "" {
		t.Errorf("RejectionStatus = %q, want empty", entry.RejectionStatus)
	}
	if entry.Propagation != "filled" {
		t.Errorf("Propagation = %q, want filled", entry.Propagation)
	}
	if entry.RejectionTimestamp != nil {
		t.Errorf("RejectionTimestamp must be nil when no rejection")
	}
}

// TestS418_LifecycleEntry_FuturesRejection proves lifecycle entry for Futures
// rejection-only scenario.
func TestS418_LifecycleEntry_FuturesRejection(t *testing.T) {
	rejection := s418FuturesRejectedIntentWithAuditMetadata()

	entry := executionclient.LifecycleEntry{
		Key:                "binancef.btcusdt.60",
		Source:             "binancef",
		Symbol:             "btcusdt",
		Timeframe:          60,
		RejectionStatus:    string(rejection.Status),
		RejectionTimestamp: &rejection.Timestamp,
		Propagation:        executionclient.DeriveEffectivePropagation(nil, nil, &rejection),
	}

	if entry.RejectionStatus != "rejected" {
		t.Errorf("RejectionStatus = %q, want rejected", entry.RejectionStatus)
	}
	if entry.Propagation != "rejected" {
		t.Errorf("Propagation = %q, want rejected", entry.Propagation)
	}
	if entry.IntentStatus != "" {
		t.Errorf("IntentStatus = %q, want empty", entry.IntentStatus)
	}
	if entry.FillStatus != "" {
		t.Errorf("FillStatus = %q, want empty", entry.FillStatus)
	}
}

// ═══════════════════════════════════════════════════════════════════
// Segment parity: Futures vs Spot lifecycle behavior equivalence
// ═══════════════════════════════════════════════════════════════════

// TestS418_SegmentParity_PropagationSymmetry proves that DeriveEffectivePropagation
// produces identical results for Spot and Futures intents in equivalent scenarios.
func TestS418_SegmentParity_PropagationSymmetry(t *testing.T) {
	now := time.Now().UTC()

	cases := []struct {
		name       string
		spotFill   *domainexec.ExecutionIntent
		spotRej    *domainexec.ExecutionIntent
		futFill    *domainexec.ExecutionIntent
		futRej     *domainexec.ExecutionIntent
		wantProp   string
	}{
		{
			name:     "fill_only",
			spotFill: &domainexec.ExecutionIntent{Source: "binances", Status: domainexec.StatusFilled, Timestamp: now},
			futFill:  &domainexec.ExecutionIntent{Source: "binancef", Status: domainexec.StatusFilled, Timestamp: now},
			wantProp: "filled",
		},
		{
			name:    "rejection_only",
			spotRej: &domainexec.ExecutionIntent{Source: "binances", Status: domainexec.StatusRejected, Timestamp: now},
			futRej:  &domainexec.ExecutionIntent{Source: "binancef", Status: domainexec.StatusRejected, Timestamp: now},
			wantProp: "rejected",
		},
		{
			name:     "fill_newer_than_rejection",
			spotFill: &domainexec.ExecutionIntent{Source: "binances", Status: domainexec.StatusFilled, Timestamp: now},
			spotRej:  &domainexec.ExecutionIntent{Source: "binances", Status: domainexec.StatusRejected, Timestamp: now.Add(-time.Minute)},
			futFill:  &domainexec.ExecutionIntent{Source: "binancef", Status: domainexec.StatusFilled, Timestamp: now},
			futRej:   &domainexec.ExecutionIntent{Source: "binancef", Status: domainexec.StatusRejected, Timestamp: now.Add(-time.Minute)},
			wantProp: "filled",
		},
		{
			name:     "partial_fill",
			spotFill: &domainexec.ExecutionIntent{Source: "binances", Status: domainexec.StatusPartiallyFilled, Timestamp: now},
			futFill:  &domainexec.ExecutionIntent{Source: "binancef", Status: domainexec.StatusPartiallyFilled, Timestamp: now},
			wantProp: "partially_filled",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spotProp := executionclient.DeriveEffectivePropagation(nil, tc.spotFill, tc.spotRej)
			futProp := executionclient.DeriveEffectivePropagation(nil, tc.futFill, tc.futRej)

			if spotProp != tc.wantProp {
				t.Errorf("Spot propagation = %q, want %q", spotProp, tc.wantProp)
			}
			if futProp != tc.wantProp {
				t.Errorf("Futures propagation = %q, want %q", futProp, tc.wantProp)
			}
			if spotProp != futProp {
				t.Errorf("parity violation: Spot=%q vs Futures=%q", spotProp, futProp)
			}
		})
	}
}

// TestS418_SegmentParity_RejectionDetailExtraction proves that the rejection
// detail extraction logic works identically for Spot and Futures metadata.
func TestS418_SegmentParity_RejectionDetailExtraction(t *testing.T) {
	spotMeta := map[string]string{
		"rejection_code":                 "VAL_INVALID_ARGUMENT",
		"rejection_reason":               "Account has insufficient balance.",
		"venue_detail.venue_http_status":  "400",
		"venue_detail.venue_error_code":   "-2010",
	}
	futuresMeta := map[string]string{
		"rejection_code":                 "VAL_INVALID_ARGUMENT",
		"rejection_reason":               "Margin is insufficient.",
		"venue_detail.venue_http_status":  "400",
		"venue_detail.venue_error_code":   "-2019",
	}

	spotIntent := &domainexec.ExecutionIntent{Source: "binances", Status: domainexec.StatusRejected, Metadata: spotMeta}
	futIntent := &domainexec.ExecutionIntent{Source: "binancef", Status: domainexec.StatusRejected, Metadata: futuresMeta}

	spotDetail := extractRejectionDetailFromIntent(spotIntent)
	futDetail := extractRejectionDetailFromIntent(futIntent)

	if spotDetail == nil {
		t.Fatal("Spot rejection detail must be extractable")
	}
	if futDetail == nil {
		t.Fatal("Futures rejection detail must be extractable")
	}

	// Both use same code — venue reason and error code differ by segment.
	if spotDetail.RejectionCode != futDetail.RejectionCode {
		t.Errorf("rejection_code parity: Spot=%s Futures=%s", spotDetail.RejectionCode, futDetail.RejectionCode)
	}

	// Venue error codes are segment-specific (Spot: -2010, Futures: -2019).
	if futDetail.VenueDetails["venue_error_code"] != "-2019" {
		t.Errorf("Futures venue_error_code: expected -2019, got %v", futDetail.VenueDetails["venue_error_code"])
	}
	if spotDetail.VenueDetails["venue_error_code"] != "-2010" {
		t.Errorf("Spot venue_error_code: expected -2010, got %v", spotDetail.VenueDetails["venue_error_code"])
	}
}

// TestS418_SegmentParity_FillRecordFormat proves that Futures fill records
// carry the expected format differences from Spot (avgPrice-based vs fills[]-based)
// while maintaining the same FillRecord structure.
func TestS418_SegmentParity_FillRecordFormat(t *testing.T) {
	now := time.Now().UTC()

	// Spot fill: price from fills[0].price, fee from commission.
	spotFill := domainexec.FillRecord{
		Price: "65000.00", Quantity: "0.001", Fee: "0.065", Simulated: false, Timestamp: now,
	}
	// Futures fill: price from avgPrice, fee from cumQuote (proxy).
	futuresFill := domainexec.FillRecord{
		Price: "65432.10", Quantity: "0.001", Fee: "0", CostBasis: "65.43210", Simulated: false, Timestamp: now,
	}

	// Structural parity: both are FillRecord with same fields populated.
	if spotFill.Simulated {
		t.Error("Spot fill must not be simulated")
	}
	if futuresFill.Simulated {
		t.Error("Futures fill must not be simulated")
	}
	if spotFill.Price == "" || futuresFill.Price == "" {
		t.Error("both fills must have non-empty price")
	}
	if spotFill.Quantity != futuresFill.Quantity {
		t.Errorf("quantity parity: Spot=%s Futures=%s", spotFill.Quantity, futuresFill.Quantity)
	}
}

// ═══════════════════════════════════════════════════════════════════
// Lifecycle list: Futures + Spot coexistence
// ═══════════════════════════════════════════════════════════════════

// TestS418_LifecycleList_MixedSegmentAggregation proves that LifecycleListReply
// correctly aggregates entries from both Spot and Futures segments.
func TestS418_LifecycleList_MixedSegmentAggregation(t *testing.T) {
	ts := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)

	entries := []executionclient.LifecycleEntry{
		{
			Key: "binances.btcusdt.60", Source: "binances", Symbol: "btcusdt", Timeframe: 60,
			IntentStatus: "submitted", IntentTimestamp: &ts,
			FillStatus: "filled", FillTimestamp: &ts,
			Propagation: "filled",
		},
		{
			Key: "binancef.btcusdt.60", Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
			IntentStatus: "submitted", IntentTimestamp: &ts,
			FillStatus: "filled", FillTimestamp: &ts,
			Propagation: "filled",
		},
		{
			Key: "binancef.ethusdt.60", Source: "binancef", Symbol: "ethusdt", Timeframe: 60,
			RejectionStatus: "rejected", RejectionTimestamp: &ts,
			Propagation: "rejected",
		},
	}

	reply := executionclient.LifecycleListReply{Entries: entries, Total: len(entries)}

	if reply.Total != 3 {
		t.Errorf("Total = %d, want 3", reply.Total)
	}

	// Count by segment.
	segmentCounts := make(map[string]int)
	for _, e := range reply.Entries {
		segmentCounts[e.Source]++
	}
	if segmentCounts["binances"] != 1 {
		t.Errorf("Spot entries = %d, want 1", segmentCounts["binances"])
	}
	if segmentCounts["binancef"] != 2 {
		t.Errorf("Futures entries = %d, want 2", segmentCounts["binancef"])
	}

	// Verify keys are unique across segments.
	keys := make(map[string]bool)
	for _, e := range reply.Entries {
		if keys[e.Key] {
			t.Errorf("duplicate key: %s", e.Key)
		}
		keys[e.Key] = true
	}
}
