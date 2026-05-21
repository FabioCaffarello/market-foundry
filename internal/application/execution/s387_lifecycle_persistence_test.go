package execution_test

import (
	"context"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/executionclient"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/problem"
)

// ── S387: Lifecycle Persistence, Read-Path, and PriceSource Wiring Tests ──
//
// These tests validate the S387 deliverables:
// 1. DeriveEffectivePropagation with rejection awareness
// 2. ExecutionStatusReply includes Rejection field
// 3. Propagation priority: most-recent venue outcome > intent > none

func TestS387_DeriveEffectivePropagation_WithRejection(t *testing.T) {
	ts := time.Now().UTC()
	tsEarlier := ts.Add(-time.Minute)

	cases := []struct {
		name      string
		intent    *domainexec.ExecutionIntent
		result    *domainexec.ExecutionIntent
		rejection *domainexec.ExecutionIntent
		wantProp  string
	}{
		{
			name:      "all_nil_returns_none",
			intent:    nil,
			result:    nil,
			rejection: nil,
			wantProp:  "none",
		},
		{
			name: "intent_only_returns_intent_status",
			intent: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
				Status: domainexec.StatusSubmitted, Final: true, Timestamp: ts,
				Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
			},
			result:    nil,
			rejection: nil,
			wantProp:  "submitted",
		},
		{
			name:   "result_only_returns_result_status",
			intent: nil,
			result: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
				FilledQuantity: "0.01", Status: domainexec.StatusFilled, Final: true, Timestamp: ts,
				Risk:  domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
				Fills: []domainexec.FillRecord{{Price: "50000", Quantity: "0.01", Fee: "0", Simulated: true, Timestamp: ts}},
			},
			rejection: nil,
			wantProp:  "filled",
		},
		{
			name:   "rejection_only_returns_rejected",
			intent: nil,
			result: nil,
			rejection: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
				Status: domainexec.StatusRejected, Final: true, Timestamp: ts,
				Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
			},
			wantProp: "rejected",
		},
		{
			name: "result_and_rejection_newer_rejection_wins",
			intent: nil,
			result: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
				FilledQuantity: "0.01", Status: domainexec.StatusFilled, Final: true, Timestamp: tsEarlier,
				Risk:  domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
				Fills: []domainexec.FillRecord{{Price: "50000", Quantity: "0.01", Fee: "0", Simulated: true, Timestamp: tsEarlier}},
			},
			rejection: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
				Status: domainexec.StatusRejected, Final: true, Timestamp: ts,
				Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
			},
			wantProp: "rejected",
		},
		{
			name: "result_and_rejection_newer_result_wins",
			intent: nil,
			result: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
				FilledQuantity: "0.01", Status: domainexec.StatusFilled, Final: true, Timestamp: ts,
				Risk:  domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
				Fills: []domainexec.FillRecord{{Price: "50000", Quantity: "0.01", Fee: "0", Simulated: true, Timestamp: ts}},
			},
			rejection: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
				Status: domainexec.StatusRejected, Final: true, Timestamp: tsEarlier,
				Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
			},
			wantProp: "filled",
		},
		{
			name: "all_three_present_most_recent_outcome_wins",
			intent: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
				Status: domainexec.StatusSubmitted, Final: true, Timestamp: tsEarlier,
				Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
			},
			result: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
				FilledQuantity: "0.01", Status: domainexec.StatusFilled, Final: true, Timestamp: tsEarlier,
				Risk:  domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
				Fills: []domainexec.FillRecord{{Price: "50000", Quantity: "0.01", Fee: "0", Simulated: true, Timestamp: tsEarlier}},
			},
			rejection: &domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
				Status: domainexec.StatusRejected, Final: true, Timestamp: ts,
				Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
			},
			wantProp: "rejected",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := executionclient.DeriveEffectivePropagation(tc.intent, tc.result, tc.rejection)
			if got != tc.wantProp {
				t.Fatalf("expected propagation %q, got %q", tc.wantProp, got)
			}
		})
	}
}

func TestS387_ExecutionStatusReply_RejectionField(t *testing.T) {
	ts := time.Now().UTC()

	t.Run("rejection_field_nil_when_no_rejection", func(t *testing.T) {
		reply := executionclient.ExecutionStatusReply{
			Intent:    nil,
			Result:    nil,
			Rejection: nil,
		}
		if reply.Rejection != nil {
			t.Fatal("expected Rejection to be nil")
		}
	})

	t.Run("rejection_field_present_when_rejected", func(t *testing.T) {
		rejection := &domainexec.ExecutionIntent{
			Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
			Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
			Status: domainexec.StatusRejected, Final: true, Timestamp: ts,
			Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
		}
		reply := executionclient.ExecutionStatusReply{
			Intent:    nil,
			Result:    nil,
			Rejection: rejection,
		}
		if reply.Rejection == nil {
			t.Fatal("expected Rejection to be non-nil")
		}
		if reply.Rejection.Status != domainexec.StatusRejected {
			t.Fatalf("expected Status rejected, got %q", reply.Rejection.Status)
		}
	})
}

// s387MockPriceSource is a test double for ports.PriceSource.
type s387MockPriceSource struct {
	price string
	prob  *problem.Problem
}

var _ ports.PriceSource = (*s387MockPriceSource)(nil)

func (m *s387MockPriceSource) LastPrice(_ context.Context, _, _ string, _ int) (string, *problem.Problem) {
	return m.price, m.prob
}

func TestS387_PriceSource_WiringValidation(t *testing.T) {
	// S387 closes the G1 gap: PriceSource is now wired in production via CandleKVPriceSource.
	// These tests validate the builder chain that is used in cmd/execute/run.go.

	t.Run("dry_run_submitter_uses_price_source_for_fills", func(t *testing.T) {
		ps := &s387MockPriceSource{price: "50123.45"}
		inner := appexec.NewPaperVenueAdapter(0)
		sub := appexec.NewDryRunSubmitter(inner).WithPriceSource(ps)

		receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{
			Intent: domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
				Status: domainexec.StatusSubmitted, Final: true,
				Timestamp: time.Now().UTC(),
				Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
			},
		})
		if prob != nil {
			t.Fatalf("unexpected problem: %s", prob.Message)
		}
		if len(receipt.Intent.Fills) == 0 {
			t.Fatal("expected at least one fill")
		}
		if receipt.Intent.Fills[0].Price != "50123.45" {
			t.Fatalf("expected fill price '50123.45', got %q", receipt.Intent.Fills[0].Price)
		}
		if !receipt.Intent.Fills[0].Simulated {
			t.Fatal("expected Simulated=true on dry-run fill")
		}
	})

	t.Run("paper_adapter_uses_price_source_for_fills", func(t *testing.T) {
		ps := &s387MockPriceSource{price: "49500.00"}
		adapter := appexec.NewPaperVenueAdapter(0).WithPriceSource(ps)

		receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{
			Intent: domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideSell, Quantity: "0.02",
				Status: domainexec.StatusSubmitted, Final: true,
				Timestamp: time.Now().UTC(),
				Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
			},
		})
		if prob != nil {
			t.Fatalf("unexpected problem: %s", prob.Message)
		}
		if len(receipt.Intent.Fills) == 0 {
			t.Fatal("expected at least one fill")
		}
		if receipt.Intent.Fills[0].Price != "49500.00" {
			t.Fatalf("expected fill price '49500.00', got %q", receipt.Intent.Fills[0].Price)
		}
	})

	t.Run("price_source_fallback_to_0_on_error", func(t *testing.T) {
		ps := &s387MockPriceSource{price: "0", prob: problem.New(problem.Unavailable, "candle KV unavailable")}
		inner := appexec.NewPaperVenueAdapter(0)
		sub := appexec.NewDryRunSubmitter(inner).WithPriceSource(ps)

		receipt, prob := sub.SubmitOrder(context.Background(), ports.VenueOrderRequest{
			Intent: domainexec.ExecutionIntent{
				Type: "paper_order", Source: "binancef", Symbol: "btcusdt",
				Timeframe: 60, Side: domainexec.SideBuy, Quantity: "0.01",
				Status: domainexec.StatusSubmitted, Final: true,
				Timestamp: time.Now().UTC(),
				Risk: domainexec.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60},
			},
		})
		if prob != nil {
			t.Fatalf("unexpected problem: %s", prob.Message)
		}
		if len(receipt.Intent.Fills) == 0 {
			t.Fatal("expected at least one fill")
		}
		// Fallback price should be "0" when PriceSource returns error.
		if receipt.Intent.Fills[0].Price != "0" {
			t.Fatalf("expected fallback fill price '0', got %q", receipt.Intent.Fills[0].Price)
		}
	})
}
