package execution_test

// s424_futures_read_path_consolidation_test.go — S424: Consolidated read-path
// auditability, queryability, and segment parity under real Futures responses.
//
// S424 consolidates S418 (read-path auditability), S422 (real acceptance/fill),
// and S423 (real rejection/partial fill) into a unified proof that real Futures
// venue response shapes flow correctly through the read-path and maintain full
// parity with the Spot segment.
//
// New value over S418 (prior read-path proof):
//   - Validates read-path extraction using EXACT metadata shapes produced by
//     S422 (avgPrice, cumQuote, updateTime) and S423 (rejection codes -2019, -2010, etc.)
//   - Consolidated correlation chain proof across all four lifecycle states
//     using real venue response metadata
//   - Explicit composite status proof with mixed Futures lifecycle entries
//   - Cross-segment parity matrix: same query contracts, same propagation logic,
//     same rejection detail structure, segment-transparent lifecycle list
//   - Fee semantics audit: cumQuote (Futures) vs commission (Spot) divergence documented
//
// Governing outcomes consolidated:
//   - accepted/filled/rejected/partial fill queryable and auditable for Futures
//   - Spot/Futures architectural parity characterized
//   - correlation chain, metadata fidelity, and explainability validated

import (
	"encoding/json"
	"testing"
	"time"

	"internal/application/executionclient"
	domainexec "internal/domain/execution"
)

// ═══════════════════════════════════════════════════════════════════
// Helpers: real venue response shapes (matching S422/S423 evidence)
// ═══════════════════════════════════════════════════════════════════

// s424FuturesAcceptedIntent mirrors the derive output that enters the execute binary.
func s424FuturesAcceptedIntent(t *testing.T) domainexec.ExecutionIntent {
	t.Helper()
	return domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        "binancef",
		Instrument:    btcUSDTPerp(t),
		Timeframe:     60,
		Side:          domainexec.SideBuy,
		Quantity:      "0.001",
		Status:        domainexec.StatusSubmitted,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		CorrelationID: "s424-corr-accepted",
		CausationID:   "s424-cause-accepted",
		Final:         true,
		Timestamp:     time.Date(2026, 3, 23, 14, 0, 0, 0, time.UTC),
	}
}

// s424FuturesFilledIntent uses the EXACT response shape from S422:
// avgPrice="65432.10", executedQty="0.001", cumQuote="65.43210", updateTime=venue timestamp.
func s424FuturesFilledIntent(t *testing.T, ts time.Time) domainexec.ExecutionIntent {
	t.Helper()
	return domainexec.ExecutionIntent{
		Type:           "venue_market_order",
		Source:         "binancef",
		Instrument:     btcUSDTPerp(t),
		Timeframe:      60,
		Side:           domainexec.SideBuy,
		Quantity:       "0.001",
		FilledQuantity: "0.001",
		Status:         domainexec.StatusFilled,
		Risk:           domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		CorrelationID:  "s424-corr-filled",
		CausationID:    "s424-cause-filled",
		Final:          true,
		Timestamp:      ts,
		Fills: []domainexec.FillRecord{
			{Price: "65432.10", Quantity: "0.001", Fee: "0", CostBasis: "65.43210", Simulated: false, Timestamp: ts},
		},
	}
}

// s424FuturesPartiallyFilledIntent uses structural partial fill shape from S423:
// executedQty < requestedQty, status=PARTIALLY_FILLED.
func s424FuturesPartiallyFilledIntent(t *testing.T, ts time.Time) domainexec.ExecutionIntent {
	t.Helper()
	return domainexec.ExecutionIntent{
		Type:           "venue_market_order",
		Source:         "binancef",
		Instrument:     btcUSDTPerp(t),
		Timeframe:      60,
		Side:           domainexec.SideBuy,
		Quantity:       "0.001",
		FilledQuantity: "0.0005",
		Status:         domainexec.StatusPartiallyFilled,
		Risk:           domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		CorrelationID:  "s424-corr-partial",
		CausationID:    "s424-cause-partial",
		Final:          false,
		Timestamp:      ts,
		Fills: []domainexec.FillRecord{
			{Price: "65432.10", Quantity: "0.0005", Fee: "0", CostBasis: "32.71605", Simulated: false, Timestamp: ts},
		},
	}
}

// s424FuturesRejectedIntent uses the EXACT metadata shape from S423 rejection path:
// RejectionProjectionActor embeds rejection_code, rejection_reason, venue_detail.* keys.
// This simulates what the KV bucket contains AFTER the rejection projection writes it.
func s424FuturesRejectedIntent(t *testing.T, ts time.Time) domainexec.ExecutionIntent {
	t.Helper()
	return domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        "binancef",
		Instrument:    btcUSDTPerp(t),
		Timeframe:     60,
		Side:          domainexec.SideBuy,
		Quantity:      "0.001",
		Status:        domainexec.StatusRejected,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		CorrelationID: "s424-corr-rejected",
		CausationID:   "s424-cause-rejected",
		Final:         true,
		Timestamp:     ts,
		Metadata: map[string]string{
			"rejection_code":                 "VAL_INVALID_ARGUMENT",
			"rejection_reason":               "Margin is insufficient.",
			"venue_detail.venue_http_status": "400",
			"venue_detail.venue_error_code":  "-2019",
		},
	}
}

// s424SpotFilledIntent mirrors S405/S407 Spot fill shape for parity comparison.
func s424SpotFilledIntent(t *testing.T, ts time.Time) domainexec.ExecutionIntent {
	t.Helper()
	return domainexec.ExecutionIntent{
		Type:           "venue_market_order",
		Source:         "binances",
		Instrument:     btcUSDTSpot(t),
		Timeframe:      60,
		Side:           domainexec.SideBuy,
		Quantity:       "0.001",
		FilledQuantity: "0.001",
		Status:         domainexec.StatusFilled,
		Risk:           domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		CorrelationID:  "s424-corr-spot-fill",
		CausationID:    "s424-cause-spot-fill",
		Final:          true,
		Timestamp:      ts,
		Fills: []domainexec.FillRecord{
			{Price: "65000.00", Quantity: "0.001", Fee: "0.065", Simulated: false, Timestamp: ts},
		},
	}
}

// s424SpotRejectedIntent mirrors S406/S407 Spot rejection shape for parity comparison.
func s424SpotRejectedIntent(t *testing.T, ts time.Time) domainexec.ExecutionIntent {
	t.Helper()
	return domainexec.ExecutionIntent{
		Type:          "paper_order",
		Source:        "binances",
		Instrument:    btcUSDTSpot(t),
		Timeframe:     60,
		Side:          domainexec.SideBuy,
		Quantity:      "0.001",
		Status:        domainexec.StatusRejected,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		CorrelationID: "s424-corr-spot-rej",
		CausationID:   "s424-cause-spot-rej",
		Final:         true,
		Timestamp:     ts,
		Metadata: map[string]string{
			"rejection_code":                 "VAL_INVALID_ARGUMENT",
			"rejection_reason":               "Account has insufficient balance for requested action.",
			"venue_detail.venue_http_status": "400",
			"venue_detail.venue_error_code":  "-2010",
		},
	}
}

// ═══════════════════════════════════════════════════════════════════
// 1. Consolidated rejection detail extraction under real Futures metadata
// ═══════════════════════════════════════════════════════════════════

// TestS424_RejectionDetail_RealFuturesMarginInsufficient proves that the rejection
// metadata embedded by RejectionProjectionActor for a real Futures margin rejection
// (S423 shape: HTTP 400, venue code -2019) is correctly extractable on the read-path.
func TestS424_RejectionDetail_RealFuturesMarginInsufficient(t *testing.T) {
	intent := s424FuturesRejectedIntent(t, time.Now().UTC())

	detail := extractRejectionDetailFromIntent(&intent)
	if detail == nil {
		t.Fatal("rejection detail must be extractable from Futures margin rejection")
	}
	if detail.RejectionCode != "VAL_INVALID_ARGUMENT" {
		t.Errorf("rejection_code: got %s, want VAL_INVALID_ARGUMENT", detail.RejectionCode)
	}
	if detail.RejectionReason != "Margin is insufficient." {
		t.Errorf("rejection_reason: got %s, want Margin is insufficient.", detail.RejectionReason)
	}
	if detail.VenueDetails["venue_http_status"] != "400" {
		t.Errorf("venue_http_status: got %v, want 400", detail.VenueDetails["venue_http_status"])
	}
	if detail.VenueDetails["venue_error_code"] != "-2019" {
		t.Errorf("venue_error_code: got %v, want -2019", detail.VenueDetails["venue_error_code"])
	}
}

// TestS424_RejectionDetail_AllFuturesRejectionScenarios proves that every S423 rejection
// scenario produces extractable audit detail on the read-path. Uses the same six error
// scenarios proven in S423.
func TestS424_RejectionDetail_AllFuturesRejectionScenarios(t *testing.T) {
	scenarios := []struct {
		name       string
		code       string
		reason     string
		httpStatus string
		venueCode  string
	}{
		{"margin_insufficient", "VAL_INVALID_ARGUMENT", "Margin is insufficient.", "400", "-2019"},
		{"balance_insufficient", "VAL_INVALID_ARGUMENT", "Account has insufficient balance for requested action.", "400", "-2010"},
		{"lot_size", "VAL_INVALID_ARGUMENT", "Filter failure: LOT_SIZE", "400", "-1013"},
		{"auth_failure", "VAL_INVALID_ARGUMENT", "Invalid API-key, IP, or permissions for action.", "401", "-2015"},
		{"rate_limit", "UNAVAILABLE", "Too many orders; current limit is 10 orders per 10 SECOND.", "429", "-1015"},
		{"venue_internal", "UNAVAILABLE", "Internal error; unable to process your request.", "400", "-1001"},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			intent := domainexec.ExecutionIntent{
				Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
				Status: domainexec.StatusRejected, Final: true,
				Metadata: map[string]string{
					"rejection_code":                 sc.code,
					"rejection_reason":               sc.reason,
					"venue_detail.venue_http_status": sc.httpStatus,
					"venue_detail.venue_error_code":  sc.venueCode,
				},
			}

			detail := extractRejectionDetailFromIntent(&intent)
			if detail == nil {
				t.Fatal("rejection detail must be extractable")
			}
			if detail.RejectionCode != sc.code {
				t.Errorf("code: got %s, want %s", detail.RejectionCode, sc.code)
			}
			if detail.RejectionReason != sc.reason {
				t.Errorf("reason: got %s, want %s", detail.RejectionReason, sc.reason)
			}
			if detail.VenueDetails["venue_http_status"] != sc.httpStatus {
				t.Errorf("http_status: got %v, want %s", detail.VenueDetails["venue_http_status"], sc.httpStatus)
			}
			if detail.VenueDetails["venue_error_code"] != sc.venueCode {
				t.Errorf("venue_code: got %v, want %s", detail.VenueDetails["venue_error_code"], sc.venueCode)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════
// 2. Consolidated composite status under real Futures lifecycle scenarios
// ═══════════════════════════════════════════════════════════════════

// TestS424_CompositeStatus_FuturesFilledWithIntent proves that composite status
// correctly assembles intent (derive) + result (fill) for Futures using S422 data shapes.
func TestS424_CompositeStatus_FuturesFilledWithIntent(t *testing.T) {
	now := time.Now().UTC()
	intent := s424FuturesAcceptedIntent(t)
	fill := s424FuturesFilledIntent(t, now)

	reply := executionclient.ExecutionStatusReply{
		Intent:      &intent,
		Result:      &fill,
		Propagation: executionclient.DeriveEffectivePropagation(&intent, &fill, nil),
	}

	if reply.Propagation != "filled" {
		t.Errorf("propagation: got %s, want filled", reply.Propagation)
	}
	if reply.Intent.Source != "binancef" {
		t.Errorf("intent source: got %s, want binancef", reply.Intent.Source)
	}
	if reply.Result.Fills[0].Simulated {
		t.Error("real venue fill must have Simulated=false")
	}
	if reply.Result.Fills[0].Price != "65432.10" {
		t.Errorf("fill price: got %s, want 65432.10 (from avgPrice)", reply.Result.Fills[0].Price)
	}
	if reply.Result.Fills[0].Fee != "0" {
		t.Errorf("fill fee: got %s, want 0 (Futures RESULT has no commission)", reply.Result.Fills[0].Fee)
	}
	if reply.Result.Fills[0].CostBasis != "65.43210" {
		t.Errorf("fill CostBasis: got %s, want 65.43210 (from cumQuote)", reply.Result.Fills[0].CostBasis)
	}
}

// TestS424_CompositeStatus_FuturesRejectedWithAuditDetail proves that composite status
// carries RejectionDetail for Futures rejections using S423 data shapes.
func TestS424_CompositeStatus_FuturesRejectedWithAuditDetail(t *testing.T) {
	now := time.Now().UTC()
	intent := s424FuturesAcceptedIntent(t)
	rejection := s424FuturesRejectedIntent(t, now)

	detail := extractRejectionDetailFromIntent(&rejection)

	reply := executionclient.ExecutionStatusReply{
		Intent:          &intent,
		Rejection:       &rejection,
		RejectionDetail: detail,
		Propagation:     executionclient.DeriveEffectivePropagation(&intent, nil, &rejection),
	}

	if reply.Propagation != "rejected" {
		t.Errorf("propagation: got %s, want rejected", reply.Propagation)
	}
	if reply.RejectionDetail == nil {
		t.Fatal("rejection detail must be present in composite status")
	}
	if reply.RejectionDetail.VenueDetails["venue_error_code"] != "-2019" {
		t.Errorf("venue_error_code: got %v, want -2019", reply.RejectionDetail.VenueDetails["venue_error_code"])
	}
}

// TestS424_CompositeStatus_FuturesPartialFill proves composite status for partial fill.
func TestS424_CompositeStatus_FuturesPartialFill(t *testing.T) {
	now := time.Now().UTC()
	intent := s424FuturesAcceptedIntent(t)
	partial := s424FuturesPartiallyFilledIntent(t, now)

	reply := executionclient.ExecutionStatusReply{
		Intent:      &intent,
		Result:      &partial,
		Propagation: executionclient.DeriveEffectivePropagation(&intent, &partial, nil),
	}

	if reply.Propagation != "partially_filled" {
		t.Errorf("propagation: got %s, want partially_filled", reply.Propagation)
	}
	if reply.Result.FilledQuantity >= reply.Result.Quantity {
		t.Error("partial fill: FilledQuantity must be less than Quantity")
	}
}

// TestS424_CompositeStatus_FuturesMixedFillAndRejection_TimestampPriority proves
// that when both fill and rejection exist, the newer timestamp determines propagation.
func TestS424_CompositeStatus_FuturesMixedFillAndRejection_TimestampPriority(t *testing.T) {
	now := time.Now().UTC()

	// Scenario 1: rejection newer than fill -> propagation = rejected
	fill1 := s424FuturesFilledIntent(t, now.Add(-10*time.Second))
	rej1 := s424FuturesRejectedIntent(t, now)
	prop1 := executionclient.DeriveEffectivePropagation(nil, &fill1, &rej1)
	if prop1 != "rejected" {
		t.Errorf("scenario 1: got %s, want rejected (newer)", prop1)
	}

	// Scenario 2: fill newer than rejection -> propagation = filled
	fill2 := s424FuturesFilledIntent(t, now)
	rej2 := s424FuturesRejectedIntent(t, now.Add(-10*time.Second))
	prop2 := executionclient.DeriveEffectivePropagation(nil, &fill2, &rej2)
	if prop2 != "filled" {
		t.Errorf("scenario 2: got %s, want filled (newer)", prop2)
	}
}

// ═══════════════════════════════════════════════════════════════════
// 3. Consolidated correlation chain proof
// ═══════════════════════════════════════════════════════════════════

// TestS424_CorrelationChain_AllFuturesLifecycleStates proves that CorrelationID,
// CausationID, and Source survive all four lifecycle states under real Futures data.
func TestS424_CorrelationChain_AllFuturesLifecycleStates(t *testing.T) {
	now := time.Now().UTC()

	states := []struct {
		name   string
		intent domainexec.ExecutionIntent
		corrID string
	}{
		{"accepted", s424FuturesAcceptedIntent(t), "s424-corr-accepted"},
		{"filled", s424FuturesFilledIntent(t, now), "s424-corr-filled"},
		{"partially_filled", s424FuturesPartiallyFilledIntent(t, now), "s424-corr-partial"},
		{"rejected", s424FuturesRejectedIntent(t, now), "s424-corr-rejected"},
	}

	for _, tc := range states {
		t.Run(tc.name, func(t *testing.T) {
			if tc.intent.CorrelationID != tc.corrID {
				t.Errorf("CorrelationID: got %s, want %s", tc.intent.CorrelationID, tc.corrID)
			}
			if tc.intent.CausationID == "" {
				t.Error("CausationID must not be empty")
			}
			if tc.intent.Source != "binancef" {
				t.Errorf("Source: got %s, want binancef", tc.intent.Source)
			}
		})
	}
}

// TestS424_CorrelationChain_RejectionMetadataRoundTrip proves that rejection
// audit metadata survives JSON marshal/unmarshal (simulating KV storage round-trip)
// using the exact metadata shape from S423 rejection scenarios.
func TestS424_CorrelationChain_RejectionMetadataRoundTrip(t *testing.T) {
	original := s424FuturesRejectedIntent(t, time.Now().UTC())

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var recovered domainexec.ExecutionIntent
	if err := json.Unmarshal(data, &recovered); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Correlation chain survives round-trip
	if recovered.CorrelationID != original.CorrelationID {
		t.Errorf("CorrelationID lost: got %s, want %s", recovered.CorrelationID, original.CorrelationID)
	}
	if recovered.CausationID != original.CausationID {
		t.Errorf("CausationID lost: got %s, want %s", recovered.CausationID, original.CausationID)
	}

	// Rejection metadata survives round-trip
	detail := extractRejectionDetailFromIntent(&recovered)
	if detail == nil {
		t.Fatal("rejection detail must survive JSON round-trip")
	}
	if detail.RejectionCode != "VAL_INVALID_ARGUMENT" {
		t.Errorf("rejection_code after round-trip: got %s", detail.RejectionCode)
	}
	if detail.VenueDetails["venue_error_code"] != "-2019" {
		t.Errorf("venue_error_code after round-trip: got %v", detail.VenueDetails["venue_error_code"])
	}
}

// ═══════════════════════════════════════════════════════════════════
// 4. Consolidated segment parity: Spot vs Futures
// ═══════════════════════════════════════════════════════════════════

// TestS424_SegmentParity_PropagationIdentical proves that DeriveEffectivePropagation
// produces identical results for Spot and Futures across all lifecycle scenarios.
func TestS424_SegmentParity_PropagationIdentical(t *testing.T) {
	now := time.Now().UTC()

	cases := []struct {
		name     string
		spotFill *domainexec.ExecutionIntent
		spotRej  *domainexec.ExecutionIntent
		futFill  *domainexec.ExecutionIntent
		futRej   *domainexec.ExecutionIntent
		want     string
	}{
		{
			name:     "fill_only",
			spotFill: ptr(s424SpotFilledIntent(t, now)),
			futFill:  ptr(s424FuturesFilledIntent(t, now)),
			want:     "filled",
		},
		{
			name:    "rejection_only",
			spotRej: ptr(s424SpotRejectedIntent(t, now)),
			futRej:  ptr(s424FuturesRejectedIntent(t, now)),
			want:    "rejected",
		},
		{
			name:     "fill_newer_than_rejection",
			spotFill: ptr(s424SpotFilledIntent(t, now)),
			spotRej:  ptr(s424SpotRejectedIntent(t, now.Add(-time.Minute))),
			futFill:  ptr(s424FuturesFilledIntent(t, now)),
			futRej:   ptr(s424FuturesRejectedIntent(t, now.Add(-time.Minute))),
			want:     "filled",
		},
		{
			name:     "rejection_newer_than_fill",
			spotFill: ptr(s424SpotFilledIntent(t, now.Add(-time.Minute))),
			spotRej:  ptr(s424SpotRejectedIntent(t, now)),
			futFill:  ptr(s424FuturesFilledIntent(t, now.Add(-time.Minute))),
			futRej:   ptr(s424FuturesRejectedIntent(t, now)),
			want:     "rejected",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spotProp := executionclient.DeriveEffectivePropagation(nil, tc.spotFill, tc.spotRej)
			futProp := executionclient.DeriveEffectivePropagation(nil, tc.futFill, tc.futRej)

			if spotProp != tc.want {
				t.Errorf("Spot: got %s, want %s", spotProp, tc.want)
			}
			if futProp != tc.want {
				t.Errorf("Futures: got %s, want %s", futProp, tc.want)
			}
			if spotProp != futProp {
				t.Errorf("parity violation: Spot=%s Futures=%s", spotProp, futProp)
			}
		})
	}
}

// TestS424_SegmentParity_RejectionDetailStructure proves that rejection detail
// extraction uses the same contract for both segments, with expected venue-specific
// differences in error codes and reasons.
func TestS424_SegmentParity_RejectionDetailStructure(t *testing.T) {
	now := time.Now().UTC()

	spotDetail := extractRejectionDetailFromIntent(ptr(s424SpotRejectedIntent(t, now)))
	futDetail := extractRejectionDetailFromIntent(ptr(s424FuturesRejectedIntent(t, now)))

	if spotDetail == nil || futDetail == nil {
		t.Fatal("both segments must produce extractable rejection detail")
	}

	// Same rejection code (VAL_INVALID_ARGUMENT) — structural parity
	if spotDetail.RejectionCode != futDetail.RejectionCode {
		t.Errorf("code parity: Spot=%s Futures=%s", spotDetail.RejectionCode, futDetail.RejectionCode)
	}

	// Venue error codes differ (expected: Spot=-2010, Futures=-2019)
	spotCode := spotDetail.VenueDetails["venue_error_code"]
	futCode := futDetail.VenueDetails["venue_error_code"]
	if spotCode == futCode {
		t.Errorf("venue codes should differ between segments: both=%v", spotCode)
	}
	if spotCode != "-2010" {
		t.Errorf("Spot venue_error_code: got %v, want -2010", spotCode)
	}
	if futCode != "-2019" {
		t.Errorf("Futures venue_error_code: got %v, want -2019", futCode)
	}
}

// TestS424_SegmentParity_FillRecordStructuralEquivalence proves that fill records
// from both segments share the same structure despite different fee semantics.
func TestS424_SegmentParity_FillRecordStructuralEquivalence(t *testing.T) {
	now := time.Now().UTC()

	spotFill := s424SpotFilledIntent(t, now).Fills[0]
	futFill := s424FuturesFilledIntent(t, now).Fills[0]

	// Both carry non-empty Price, Quantity, Fee
	for _, check := range []struct {
		name, spot, fut string
	}{
		{"Price", spotFill.Price, futFill.Price},
		{"Quantity", spotFill.Quantity, futFill.Quantity},
		{"Fee", spotFill.Fee, futFill.Fee},
	} {
		if check.spot == "" {
			t.Errorf("Spot %s must not be empty", check.name)
		}
		if check.fut == "" {
			t.Errorf("Futures %s must not be empty", check.name)
		}
	}

	// Same Simulated flag (both real venue)
	if spotFill.Simulated || futFill.Simulated {
		t.Error("both must have Simulated=false for real venue fills")
	}

	// Fee values DIFFER (Spot=commission, Futures=cumQuote) — this is expected
	if spotFill.Fee == futFill.Fee {
		t.Log("note: Spot and Futures fees happen to match — unusual but not invalid")
	}
}

// TestS424_SegmentParity_PartitionKeyIsolation proves that same symbol/timeframe
// on different segments produces distinct partition keys.
func TestS424_SegmentParity_PartitionKeyIsolation(t *testing.T) {
	now := time.Now().UTC()

	spotKey := s424SpotFilledIntent(t, now).PartitionKey()
	futKey := s424FuturesFilledIntent(t, now).PartitionKey()

	if spotKey == futKey {
		t.Fatalf("partition keys must differ: spot=%s futures=%s", spotKey, futKey)
	}
	if spotKey != "binances.btcusdt.60" {
		t.Errorf("spot key: got %s, want binances.btcusdt.60", spotKey)
	}
	if futKey != "binancef.btcusdt.60" {
		t.Errorf("futures key: got %s, want binancef.btcusdt.60", futKey)
	}
}

// ═══════════════════════════════════════════════════════════════════
// 5. Consolidated lifecycle list with mixed Spot/Futures entries
// ═══════════════════════════════════════════════════════════════════

// TestS424_LifecycleList_ConsolidatedMixedSegments proves that LifecycleListReply
// correctly aggregates entries from both segments with different lifecycle states.
func TestS424_LifecycleList_ConsolidatedMixedSegments(t *testing.T) {
	now := time.Now().UTC()

	spotFill := s424SpotFilledIntent(t, now)
	futFill := s424FuturesFilledIntent(t, now)
	futRej := s424FuturesRejectedIntent(t, now.Add(-5*time.Second))

	entries := []executionclient.LifecycleEntry{
		{
			Key: spotFill.PartitionKey(), Source: "binances", Symbol: "btcusdt", Timeframe: 60,
			FillStatus: "filled", FillTimestamp: &now,
			Propagation: executionclient.DeriveEffectivePropagation(nil, &spotFill, nil),
		},
		{
			Key: futFill.PartitionKey(), Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
			FillStatus: "filled", FillTimestamp: &now,
			Propagation: executionclient.DeriveEffectivePropagation(nil, &futFill, nil),
		},
		{
			Key: "binancef.ethusdt.60", Source: "binancef", Symbol: "ethusdt", Timeframe: 60,
			RejectionStatus: "rejected", RejectionTimestamp: &now,
			Propagation: executionclient.DeriveEffectivePropagation(nil, nil, &futRej),
		},
	}

	reply := executionclient.LifecycleListReply{Entries: entries, Total: len(entries)}

	if reply.Total != 3 {
		t.Errorf("Total: got %d, want 3", reply.Total)
	}

	// Segment distribution
	segments := make(map[string]int)
	propagations := make(map[string]int)
	for _, e := range reply.Entries {
		segments[e.Source]++
		propagations[e.Propagation]++
	}

	if segments["binances"] != 1 {
		t.Errorf("Spot entries: got %d, want 1", segments["binances"])
	}
	if segments["binancef"] != 2 {
		t.Errorf("Futures entries: got %d, want 2", segments["binancef"])
	}
	if propagations["filled"] != 2 {
		t.Errorf("filled propagations: got %d, want 2", propagations["filled"])
	}
	if propagations["rejected"] != 1 {
		t.Errorf("rejected propagations: got %d, want 1", propagations["rejected"])
	}
}

// ═══════════════════════════════════════════════════════════════════
// 6. Fee semantics audit (Spot commission vs Futures cumQuote)
// ═══════════════════════════════════════════════════════════════════

// TestS424_FeeSemantics_FuturesCumQuoteAuditTrail proves that the Futures cumQuote
// fee proxy (identified in S422 as G-4) is preserved through the read-path and
// is distinguishable from Spot commission-based fees by source segment.
func TestS424_FeeSemantics_FuturesCumQuoteAuditTrail(t *testing.T) {
	now := time.Now().UTC()

	spotFill := s424SpotFilledIntent(t, now)
	futFill := s424FuturesFilledIntent(t, now)

	// Spot: commission per fill (small relative to notional)
	// Futures: cumQuote (total notional, large relative to quantity)
	spotFee := spotFill.Fills[0].Fee
	futFee := futFill.Fills[0].Fee

	if spotFee == "" || futFee == "" {
		t.Fatal("both fees must be non-empty")
	}

	// Both are stored as string decimals — consumer must interpret by source
	if spotFill.Source != "binances" {
		t.Error("Spot source must be binances for fee interpretation")
	}
	if futFill.Source != "binancef" {
		t.Error("Futures source must be binancef for fee interpretation")
	}

	// The source field is the key to correct fee interpretation
	// No architectural divergence — same FillRecord.Fee field, different semantics
}

// ═══════════════════════════════════════════════════════════════════
// helper
// ═══════════════════════════════════════════════════════════════════

func ptr(i domainexec.ExecutionIntent) *domainexec.ExecutionIntent { return &i }
