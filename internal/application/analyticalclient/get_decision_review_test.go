package analyticalclient_test

import (
	"context"
	"log/slog"
	"testing"

	"internal/application/analyticalclient"
	"internal/shared/problem"
)

func TestGetDecisionReview_SingleChain_FullBundle(t *testing.T) {
	chain := fullChain("corr-review-001")
	reader := &stubCompositeReader{chain: chain}
	uc := analyticalclient.NewGetDecisionReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.DecisionReviewQuery{
		CorrelationID: "corr-review-001",
		Symbol:        "btcusdt",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(result.Reviews))
	}

	r := result.Reviews[0]
	if r.CorrelationID != "corr-review-001" {
		t.Errorf("expected correlation_id corr-review-001, got %s", r.CorrelationID)
	}

	// Inputs present.
	if r.Inputs == nil {
		t.Fatal("expected inputs to be present")
	}
	if r.Inputs.EventID != "sig-001" {
		t.Errorf("expected signal event_id sig-001, got %s", r.Inputs.EventID)
	}

	// Transform present.
	if r.Transform == nil {
		t.Fatal("expected transform to be present")
	}
	if r.Transform.Outcome != "triggered" {
		t.Errorf("expected outcome triggered, got %s", r.Transform.Outcome)
	}
	if r.Transform.Severity != "high" {
		t.Errorf("expected severity high, got %s", r.Transform.Severity)
	}
	if r.Transform.Confidence != "0.85" {
		t.Errorf("expected confidence 0.85, got %s", r.Transform.Confidence)
	}

	// Resolution present.
	if r.Resolution == nil {
		t.Fatal("expected resolution to be present")
	}
	if r.Resolution.Direction != "long" {
		t.Errorf("expected direction long, got %s", r.Resolution.Direction)
	}

	// Constraints present.
	if r.Constraints == nil {
		t.Fatal("expected constraints to be present")
	}
	if r.Constraints.Disposition != "approved" {
		t.Errorf("expected disposition approved, got %s", r.Constraints.Disposition)
	}
	if r.Constraints.Limits.MaxPositionSize != "0.1" {
		t.Errorf("expected max_position_size 0.1, got %s", r.Constraints.Limits.MaxPositionSize)
	}

	// Output present.
	if r.Output == nil {
		t.Fatal("expected output to be present")
	}
	if r.Output.Side != "buy" {
		t.Errorf("expected side buy, got %s", r.Output.Side)
	}

	// Chain completeness.
	if !r.ChainComplete {
		t.Error("expected chain_complete to be true")
	}
	if r.StageCount != 5 {
		t.Errorf("expected stage_count 5, got %d", r.StageCount)
	}

	// Explanation non-empty.
	if r.Explanation == "" {
		t.Error("expected explanation to be non-empty")
	}
}

func TestGetDecisionReview_SingleChain_NoDecision_EmptyResult(t *testing.T) {
	chain := fullChain("corr-nodec-001")
	chain.Decision = nil
	chain.StageCount = 4
	chain.ChainComplete = false
	chain.MissingStages = []string{"decision"}

	reader := &stubCompositeReader{chain: chain}
	uc := analyticalclient.NewGetDecisionReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.DecisionReviewQuery{
		CorrelationID: "corr-nodec-001",
		Symbol:        "btcusdt",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Reviews) != 0 {
		t.Fatalf("expected 0 reviews for chain without decision, got %d", len(result.Reviews))
	}
}

func TestGetDecisionReview_SingleChain_MissingSymbol(t *testing.T) {
	reader := &stubCompositeReader{}
	uc := analyticalclient.NewGetDecisionReviewUseCase(reader, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.DecisionReviewQuery{
		CorrelationID: "corr-001",
	})
	if prob == nil {
		t.Fatal("expected problem for missing symbol")
	}
	if prob.Code != problem.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %s", prob.Code)
	}
}

func TestGetDecisionReview_Batch_OutcomeFilter(t *testing.T) {
	triggered := fullChain("corr-triggered")
	notTriggered := fullChain("corr-not-triggered")
	notTriggered.Decision.Outcome = "not_triggered"
	notTriggered.Decision.Severity = "none"

	reader := &stubCompositeReader{
		chains: []analyticalclient.CompositeExecutionChain{*triggered, *notTriggered},
	}
	uc := analyticalclient.NewGetDecisionReviewUseCase(reader, slog.Default())

	// Filter for triggered only.
	result, prob := uc.Execute(context.Background(), analyticalclient.DecisionReviewQuery{
		Source:    "binance",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Outcome:   "triggered",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review with outcome filter, got %d", len(result.Reviews))
	}
	if result.Reviews[0].Transform.Outcome != "triggered" {
		t.Errorf("expected filtered review to be triggered, got %s", result.Reviews[0].Transform.Outcome)
	}
}

func TestGetDecisionReview_Batch_ValidationErrors(t *testing.T) {
	reader := &stubCompositeReader{}
	uc := analyticalclient.NewGetDecisionReviewUseCase(reader, slog.Default())

	tests := []struct {
		name  string
		query analyticalclient.DecisionReviewQuery
	}{
		{"missing source", analyticalclient.DecisionReviewQuery{Symbol: "btcusdt", Timeframe: 60}},
		{"missing symbol", analyticalclient.DecisionReviewQuery{Source: "binance", Timeframe: 60}},
		{"missing timeframe", analyticalclient.DecisionReviewQuery{Source: "binance", Symbol: "btcusdt"}},
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

func TestGetDecisionReview_NilReader(t *testing.T) {
	uc := analyticalclient.NewGetDecisionReviewUseCase(nil, slog.Default())
	_, prob := uc.Execute(context.Background(), analyticalclient.DecisionReviewQuery{CorrelationID: "x", Symbol: "y"})
	if prob == nil {
		t.Fatal("expected problem for nil reader")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}

func TestGetDecisionReview_PartialChain_DecisionOnly(t *testing.T) {
	chain := fullChain("corr-partial")
	chain.Signal = nil
	chain.Strategy = nil
	chain.Risk = nil
	chain.Execution = nil
	chain.StageCount = 1
	chain.ChainComplete = false
	chain.MissingStages = []string{"signal", "strategy", "risk", "execution"}

	reader := &stubCompositeReader{chain: chain}
	uc := analyticalclient.NewGetDecisionReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.DecisionReviewQuery{
		CorrelationID: "corr-partial",
		Symbol:        "btcusdt",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(result.Reviews))
	}

	r := result.Reviews[0]
	if r.Inputs != nil {
		t.Error("expected inputs to be nil for partial chain")
	}
	if r.Transform == nil {
		t.Fatal("expected transform to be present")
	}
	if r.Resolution != nil {
		t.Error("expected resolution to be nil for partial chain")
	}
	if r.Constraints != nil {
		t.Error("expected constraints to be nil for partial chain")
	}
	if r.Output != nil {
		t.Error("expected output to be nil for partial chain")
	}
	if r.ChainComplete {
		t.Error("expected chain_complete to be false")
	}
}
