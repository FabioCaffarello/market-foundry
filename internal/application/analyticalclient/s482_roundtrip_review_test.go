package analyticalclient_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/execution"
	"internal/shared/problem"
)

// --- Round-Trip Review Tests (S482) ---

func TestGetRoundTripReview_Batch_PairedClean(t *testing.T) {
	now := time.Now()
	entry := filledChainWithSide("corr-rv-entry", "buy", "50000.00", "0.1", "0.50", "5000.00", now)
	exit := filledChainWithSide("corr-rv-exit", "sell", "51000.00", "0.1", "0.50", "5100.00", now.Add(time.Minute))

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry, *exit},
	}
	uc := analyticalclient.NewGetRoundTripReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.RoundTripReviewQuery{
		Source:    "binance",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(result.Reviews))
	}

	rv := result.Reviews[0]
	if !rv.Reconciliation.Clean {
		t.Errorf("expected clean round-trip, got flags=%v", rv.Reconciliation.Flags)
	}
	if !rv.Reconciliation.FeeReliable {
		t.Error("expected fee_reliable=true")
	}
	if !rv.Reconciliation.PnLReliable {
		t.Error("expected pnl_reliable=true")
	}
	if rv.Attribution == nil {
		t.Fatal("expected attribution")
	}
	if rv.Attribution.Outcome != "win" {
		t.Errorf("outcome=%s, want win", rv.Attribution.Outcome)
	}

	// Summary checks.
	if result.Summary.CleanCount != 1 {
		t.Errorf("clean_count=%d, want 1", result.Summary.CleanCount)
	}
	if result.Summary.FlaggedCount != 0 {
		t.Errorf("flagged_count=%d, want 0", result.Summary.FlaggedCount)
	}
	if result.Summary.PnLReliableCount != 1 {
		t.Errorf("pnl_reliable_count=%d, want 1", result.Summary.PnLReliableCount)
	}
}

func TestGetRoundTripReview_Batch_FeeGapFlagged(t *testing.T) {
	now := time.Now()
	// Futures-style: zero fees.
	entry := filledChainWithSide("corr-rv-fg-entry", "buy", "50000.00", "0.1", "0", "5000.00", now)
	exit := filledChainWithSide("corr-rv-fg-exit", "sell", "51000.00", "0.1", "0", "5100.00", now.Add(time.Minute))

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry, *exit},
	}
	uc := analyticalclient.NewGetRoundTripReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.RoundTripReviewQuery{
		Source:    "binance",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(result.Reviews))
	}
	rv := result.Reviews[0]
	if rv.Reconciliation.Clean {
		t.Error("expected flagged round-trip for fee gap")
	}
	if rv.Reconciliation.FeeReliable {
		t.Error("expected fee_reliable=false for zero fees")
	}
	if result.Summary.FlaggedCount != 1 {
		t.Errorf("flagged_count=%d, want 1", result.Summary.FlaggedCount)
	}
	if result.Summary.FlagCounts["fee_gap"] != 1 {
		t.Errorf("flag_counts[fee_gap]=%d, want 1", result.Summary.FlagCounts["fee_gap"])
	}
}

func TestGetRoundTripReview_Batch_UnmatchedEntryFlagged(t *testing.T) {
	now := time.Now()
	entry := filledChainWithSide("corr-rv-unm", "buy", "50000.00", "0.1", "0.50", "5000.00", now)

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry},
	}
	uc := analyticalclient.NewGetRoundTripReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.RoundTripReviewQuery{
		Source:    "binance",
		Symbol:    "btcusdt",
		Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(result.Reviews))
	}
	rv := result.Reviews[0]
	if rv.Reconciliation.Clean {
		t.Error("expected flagged for unmatched entry")
	}
	hasFlag := false
	for _, f := range rv.Reconciliation.Flags {
		if string(f) == "unmatched_open" {
			hasFlag = true
		}
	}
	if !hasFlag {
		t.Error("expected unmatched_open flag")
	}
}

func TestGetRoundTripReview_OutcomeFilter(t *testing.T) {
	now := time.Now()
	// Win pair.
	entry := filledChainWithSide("corr-rv-of-entry", "buy", "50000.00", "0.1", "0.50", "5000.00", now)
	exit := filledChainWithSide("corr-rv-of-exit", "sell", "51000.00", "0.1", "0.50", "5100.00", now.Add(time.Minute))
	// Unmatched (unresolved).
	loner := filledChainWithSide("corr-rv-of-loner", "buy", "49000.00", "0.2", "0.40", "9800.00", now.Add(2*time.Minute))

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry, *exit, *loner},
	}
	uc := analyticalclient.NewGetRoundTripReviewUseCase(reader, slog.Default())

	// Filter to win only.
	result, prob := uc.Execute(context.Background(), analyticalclient.RoundTripReviewQuery{
		Source:    "binance",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Outcome:   "win",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review with outcome=win, got %d", len(result.Reviews))
	}
	if result.Reviews[0].Attribution == nil || result.Reviews[0].Attribution.Outcome != "win" {
		t.Error("expected win attribution")
	}
}

func TestGetRoundTripReview_FlaggedFilter(t *testing.T) {
	now := time.Now()
	// Clean pair.
	entry := filledChainWithSide("corr-rv-ff-entry", "buy", "50000.00", "0.1", "0.50", "5000.00", now)
	exit := filledChainWithSide("corr-rv-ff-exit", "sell", "51000.00", "0.1", "0.50", "5100.00", now.Add(time.Minute))
	// Fee gap pair (flagged).
	entry2 := filledChainWithSide("corr-rv-ff-entry2", "buy", "50000.00", "0.1", "0", "5000.00", now.Add(3*time.Minute))
	exit2 := filledChainWithSide("corr-rv-ff-exit2", "sell", "51000.00", "0.1", "0", "5100.00", now.Add(4*time.Minute))

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry, *exit, *entry2, *exit2},
	}
	uc := analyticalclient.NewGetRoundTripReviewUseCase(reader, slog.Default())

	// Filter to flagged only.
	result, prob := uc.Execute(context.Background(), analyticalclient.RoundTripReviewQuery{
		Source:    "binance",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Flagged:   true,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 flagged review, got %d", len(result.Reviews))
	}
	if result.Reviews[0].Reconciliation.Clean {
		t.Error("expected flagged review")
	}
}

func TestGetRoundTripReview_ValidationErrors(t *testing.T) {
	uc := analyticalclient.NewGetRoundTripReviewUseCase(&stubCompositeReader{}, slog.Default())

	tests := []struct {
		name  string
		query analyticalclient.RoundTripReviewQuery
	}{
		{"missing source", analyticalclient.RoundTripReviewQuery{Symbol: "btcusdt", Timeframe: 60}},
		{"missing symbol", analyticalclient.RoundTripReviewQuery{Source: "binance", Timeframe: 60}},
		{"invalid timeframe", analyticalclient.RoundTripReviewQuery{Source: "binance", Symbol: "btcusdt", Timeframe: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, prob := uc.Execute(context.Background(), tt.query)
			if prob == nil {
				t.Fatal("expected validation problem")
			}
			if prob.Code != problem.InvalidArgument {
				t.Errorf("expected InvalidArgument, got %s", prob.Code)
			}
		})
	}
}

func TestGetRoundTripReview_NilUseCase(t *testing.T) {
	var uc *analyticalclient.GetRoundTripReviewUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.RoundTripReviewQuery{
		CorrelationID: "x", Symbol: "y",
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}

func TestGetRoundTripReview_SimulatedFlagged(t *testing.T) {
	now := time.Now()
	entry := filledChainWithSide("corr-rv-sim-entry", "buy", "50000.00", "0.1", "0", "0", now)
	entry.Execution.ExecutionIntent.Fills[0].Simulated = true
	exit := filledChainWithSide("corr-rv-sim-exit", "sell", "51000.00", "0.1", "0", "0", now.Add(time.Minute))
	exit.Execution.ExecutionIntent.Fills[0].Simulated = true

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*entry, *exit},
	}
	uc := analyticalclient.NewGetRoundTripReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.RoundTripReviewQuery{
		Source: "binance", Symbol: "btcusdt", Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(result.Reviews))
	}
	rv := result.Reviews[0]
	hasSimulated := false
	for _, f := range rv.Reconciliation.Flags {
		if string(f) == "simulated" {
			hasSimulated = true
		}
	}
	if !hasSimulated {
		t.Error("expected simulated flag")
	}
}

// Verify the review surface produces correct summary reconciliation counts
// with a mix of clean and flagged round-trips.
func TestGetRoundTripReview_SummaryReconciliationCounts(t *testing.T) {
	now := time.Now()
	// Clean pair.
	e1 := filledChainWithSide("corr-rv-sc-e1", "buy", "50000.00", "0.1", "0.50", "5000.00", now)
	x1 := filledChainWithSide("corr-rv-sc-x1", "sell", "51000.00", "0.1", "0.50", "5100.00", now.Add(time.Minute))
	// Fee gap pair (flagged).
	e2 := filledChainWithSide("corr-rv-sc-e2", "buy", "50000.00", "0.1", "0", "5000.00", now.Add(3*time.Minute))
	x2 := filledChainWithSide("corr-rv-sc-x2", "sell", "49000.00", "0.1", "0", "4900.00", now.Add(4*time.Minute))
	// Unmatched.
	lone := filledChainWithSide("corr-rv-sc-lone", "buy", "48000.00", "0.2", "0.40", "9600.00", now.Add(5*time.Minute))

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*e1, *x1, *e2, *x2, *lone},
	}
	uc := analyticalclient.NewGetRoundTripReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.RoundTripReviewQuery{
		Source: "binance", Symbol: "btcusdt", Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if result.Summary.PairedCount != 2 {
		t.Errorf("paired_count=%d, want 2", result.Summary.PairedCount)
	}
	if result.Summary.UnmatchedEntries != 1 {
		t.Errorf("unmatched_entries=%d, want 1", result.Summary.UnmatchedEntries)
	}
	// 1 clean pair.
	if result.Summary.CleanCount != 1 {
		t.Errorf("clean_count=%d, want 1", result.Summary.CleanCount)
	}
	// 1 fee-gap pair + 1 unmatched = 2 flagged.
	if result.Summary.FlaggedCount != 2 {
		t.Errorf("flagged_count=%d, want 2", result.Summary.FlaggedCount)
	}
	if result.Summary.FeeReliableCount != 1 {
		t.Errorf("fee_reliable_count=%d, want 1", result.Summary.FeeReliableCount)
	}
}

// Verify rejected chains are excluded from review.
func TestGetRoundTripReview_RejectedExcluded(t *testing.T) {
	rejected := filledChainWithSide("corr-rv-rej", "buy", "50000.00", "0.1", "0.50", "5000.00", time.Now())
	rejected.Execution.ExecutionIntent.Status = execution.Status("rejected")
	rejected.Execution.ExecutionIntent.Fills = nil

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*rejected},
	}
	uc := analyticalclient.NewGetRoundTripReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.RoundTripReviewQuery{
		Source: "binance", Symbol: "btcusdt", Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Reviews) != 0 {
		t.Errorf("expected 0 reviews for rejected, got %d", len(result.Reviews))
	}
}
