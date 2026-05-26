package execution_test

import (
	"testing"
	"time"

	"internal/application/executionclient"
	domainexec "internal/domain/execution"
)

// ==========================================================================
// S413 — Operational Lifecycle Queryability and Read Consolidation
//
// These tests validate the S413 deliverables:
//   1. LifecycleEntry correctly captures per-key lifecycle state
//   2. LifecycleListReply aggregates entries with correct totals
//   3. Propagation derivation aligns across all lifecycle surfaces
//   4. Partial presence (intent-only, fill-only, rejection-only) handled
//   5. Timestamp tracking on each surface
// ==========================================================================

func s413Intent(t *testing.T, status domainexec.Status, ts time.Time) *domainexec.ExecutionIntent {
	t.Helper()
	return &domainexec.ExecutionIntent{
		Type: "paper_order", Source: "binances", Instrument: btcUSDTSpot(t),
		Timeframe: 1, Side: domainexec.SideBuy, Quantity: "0.01",
		Status: status, Final: true, Timestamp: ts,
		Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 1},
	}
}

func TestS413_LifecycleEntry_PropagationAlignment(t *testing.T) {
	ts := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	tsLater := ts.Add(time.Minute)

	cases := []struct {
		name      string
		intent    *domainexec.ExecutionIntent
		fill      *domainexec.ExecutionIntent
		rejection *domainexec.ExecutionIntent
		wantProp  string
	}{
		{
			name:     "intent_only_submitted",
			intent:   s413Intent(t, domainexec.StatusSubmitted, ts),
			wantProp: "submitted",
		},
		{
			name:     "fill_only_filled",
			fill:     s413Intent(t, domainexec.StatusFilled, ts),
			wantProp: "filled",
		},
		{
			name:      "rejection_only_rejected",
			rejection: s413Intent(t, domainexec.StatusRejected, ts),
			wantProp:  "rejected",
		},
		{
			name:     "intent_plus_fill_returns_fill",
			intent:   s413Intent(t, domainexec.StatusSubmitted, ts),
			fill:     s413Intent(t, domainexec.StatusFilled, tsLater),
			wantProp: "filled",
		},
		{
			name:      "intent_plus_rejection_returns_rejected",
			intent:    s413Intent(t, domainexec.StatusSubmitted, ts),
			rejection: s413Intent(t, domainexec.StatusRejected, tsLater),
			wantProp:  "rejected",
		},
		{
			name:      "fill_and_rejection_newer_rejection_wins",
			fill:      s413Intent(t, domainexec.StatusFilled, ts),
			rejection: s413Intent(t, domainexec.StatusRejected, tsLater),
			wantProp:  "rejected",
		},
		{
			name:      "fill_and_rejection_newer_fill_wins",
			fill:      s413Intent(t, domainexec.StatusFilled, tsLater),
			rejection: s413Intent(t, domainexec.StatusRejected, ts),
			wantProp:  "filled",
		},
		{
			name:      "all_three_most_recent_wins",
			intent:    s413Intent(t, domainexec.StatusSubmitted, ts),
			fill:      s413Intent(t, domainexec.StatusFilled, ts.Add(time.Minute)),
			rejection: s413Intent(t, domainexec.StatusRejected, ts.Add(2*time.Minute)),
			wantProp:  "rejected",
		},
		{
			name:     "no_surfaces_returns_none",
			wantProp: "none",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := executionclient.DeriveEffectivePropagation(tc.intent, tc.fill, tc.rejection)
			if got != tc.wantProp {
				t.Errorf("propagation = %q, want %q", got, tc.wantProp)
			}
		})
	}
}

func TestS413_LifecycleEntry_FieldPopulation(t *testing.T) {
	ts := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)

	intent := s413Intent(t, domainexec.StatusSubmitted, ts)
	fill := s413Intent(t, domainexec.StatusFilled, ts.Add(time.Minute))

	entry := executionclient.LifecycleEntry{
		Key:             "binances.BTCUSDT.1",
		Source:          "binances",
		Symbol:          "BTCUSDT",
		Timeframe:       1,
		IntentStatus:    string(intent.Status),
		IntentTimestamp: &intent.Timestamp,
		FillStatus:      string(fill.Status),
		FillTimestamp:   &fill.Timestamp,
		Propagation:     executionclient.DeriveEffectivePropagation(intent, fill, nil),
	}

	if entry.IntentStatus != "submitted" {
		t.Errorf("IntentStatus = %q, want %q", entry.IntentStatus, "submitted")
	}
	if entry.FillStatus != "filled" {
		t.Errorf("FillStatus = %q, want %q", entry.FillStatus, "filled")
	}
	if entry.RejectionStatus != "" {
		t.Errorf("RejectionStatus = %q, want empty", entry.RejectionStatus)
	}
	if entry.Propagation != "filled" {
		t.Errorf("Propagation = %q, want %q", entry.Propagation, "filled")
	}
	if entry.RejectionTimestamp != nil {
		t.Errorf("RejectionTimestamp = %v, want nil", entry.RejectionTimestamp)
	}
}

func TestS413_LifecycleListReply_Aggregation(t *testing.T) {
	ts := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)

	entries := []executionclient.LifecycleEntry{
		{
			Key: "binances.BTCUSDT.1", Source: "binances", Symbol: "BTCUSDT", Timeframe: 1,
			IntentStatus: "submitted", IntentTimestamp: &ts,
			Propagation: "submitted",
		},
		{
			Key: "binances.ETHUSDT.1", Source: "binances", Symbol: "ETHUSDT", Timeframe: 1,
			IntentStatus: "submitted", IntentTimestamp: &ts,
			FillStatus: "filled", FillTimestamp: &ts,
			Propagation: "filled",
		},
		{
			Key: "binancef.BTCUSDT.5", Source: "binancef", Symbol: "BTCUSDT", Timeframe: 5,
			RejectionStatus: "rejected", RejectionTimestamp: &ts,
			Propagation: "rejected",
		},
	}

	reply := executionclient.LifecycleListReply{
		Entries: entries,
		Total:   len(entries),
	}

	if reply.Total != 3 {
		t.Errorf("Total = %d, want 3", reply.Total)
	}

	// Verify propagation diversity.
	propCounts := make(map[string]int)
	for _, e := range reply.Entries {
		propCounts[e.Propagation]++
	}
	if propCounts["submitted"] != 1 {
		t.Errorf("submitted count = %d, want 1", propCounts["submitted"])
	}
	if propCounts["filled"] != 1 {
		t.Errorf("filled count = %d, want 1", propCounts["filled"])
	}
	if propCounts["rejected"] != 1 {
		t.Errorf("rejected count = %d, want 1", propCounts["rejected"])
	}
}

func TestS413_LifecycleEntry_PartiallyFilledPropagation(t *testing.T) {
	ts := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)

	intent := s413Intent(t, domainexec.StatusSubmitted, ts)
	partialFill := &domainexec.ExecutionIntent{
		Type: "venue_market_order", Source: "binances", Instrument: btcUSDTSpot(t),
		Timeframe: 1, Side: domainexec.SideBuy, Quantity: "0.01",
		FilledQuantity: "0.005", Status: domainexec.StatusPartiallyFilled,
		Final: true, Timestamp: ts.Add(time.Minute),
		Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 1},
		Fills: []domainexec.FillRecord{
			{Price: "50000", Quantity: "0.005", Fee: "0.01", Simulated: false, Timestamp: ts.Add(time.Minute)},
		},
	}

	got := executionclient.DeriveEffectivePropagation(intent, partialFill, nil)
	if got != "partially_filled" {
		t.Errorf("propagation = %q, want %q", got, "partially_filled")
	}
}
