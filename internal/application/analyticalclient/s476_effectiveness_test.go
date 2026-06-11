package analyticalclient_test

import (
	"internal/domain/instrument"

	"context"
	"log/slog"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/execution"
	"internal/shared/problem"
)

// --- Effectiveness Use Case Tests (S476) ---

func filledChain(corrID string) *analyticalclient.CompositeExecutionChain {
	chain := fullChain(corrID)
	chain.Execution.Status = "filled"
	chain.Execution.FilledQuantity = "0.1"
	chain.Execution.CorrelationID = corrID
	chain.Execution.Fills = []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.1", Fee: "0.50", FeeAsset: "USDT", CostBasis: "5000.00", Timestamp: time.Now()},
	}
	chain.Execution.Risk = execution.RiskInput{
		Type:             "rsi_oversold",
		Disposition:      "approved",
		Confidence:       "0.85",
		StrategyType:     "mean_reversion_entry",
		DecisionSeverity: "high",
	}
	return chain
}

func TestGetEffectiveness_Single_FilledChain(t *testing.T) {
	chain := filledChain("corr-eff-001")
	reader := &stubCompositeReader{chain: chain}
	uc := analyticalclient.NewGetEffectivenessUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessQuery{
		CorrelationID: "corr-eff-001",
		Instrument:    instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Evaluations) != 1 {
		t.Fatalf("expected 1 evaluation, got %d", len(result.Evaluations))
	}

	eval := result.Evaluations[0]
	if eval.CorrelationID != "corr-eff-001" {
		t.Errorf("correlation_id=%s, want corr-eff-001", eval.CorrelationID)
	}
	if eval.EntryCostBasis != 5000.0 {
		t.Errorf("entry_cost_basis=%f, want 5000.0", eval.EntryCostBasis)
	}
	if eval.TotalFees != 0.50 {
		t.Errorf("total_fees=%f, want 0.50", eval.TotalFees)
	}
	if eval.FillCount != 1 {
		t.Errorf("fill_count=%d, want 1", eval.FillCount)
	}
	if eval.StrategyType != "mean_reversion_entry" {
		t.Errorf("strategy_type=%s, want mean_reversion_entry", eval.StrategyType)
	}
	if eval.DecisionSeverity != "high" {
		t.Errorf("decision_severity=%s, want high", eval.DecisionSeverity)
	}
	if result.Meta.ChainsScanned != 1 {
		t.Errorf("chains_scanned=%d, want 1", result.Meta.ChainsScanned)
	}
}

func TestGetEffectiveness_Single_RejectedExcluded(t *testing.T) {
	chain := fullChain("corr-rej")
	chain.Execution.Status = "rejected"
	reader := &stubCompositeReader{chain: chain}
	uc := analyticalclient.NewGetEffectivenessUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessQuery{
		CorrelationID: "corr-rej",
		Instrument:    instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Evaluations) != 0 {
		t.Errorf("expected 0 evaluations for rejected, got %d", len(result.Evaluations))
	}
	if result.Meta.Excluded != 1 {
		t.Errorf("excluded=%d, want 1", result.Meta.Excluded)
	}
}

func TestGetEffectiveness_Single_NoExecution(t *testing.T) {
	chain := fullChain("corr-noexec")
	chain.Execution = nil
	reader := &stubCompositeReader{chain: chain}
	uc := analyticalclient.NewGetEffectivenessUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessQuery{
		CorrelationID: "corr-noexec",
		Instrument:    instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Evaluations) != 0 {
		t.Errorf("expected 0 evaluations for no execution, got %d", len(result.Evaluations))
	}
}

func TestGetEffectiveness_Single_MissingSymbol(t *testing.T) {
	uc := analyticalclient.NewGetEffectivenessUseCase(&stubCompositeReader{}, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessQuery{
		CorrelationID: "corr-001",
	})
	if prob == nil {
		t.Fatal("expected problem for missing symbol")
	}
	if prob.Code != problem.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %s", prob.Code)
	}
}

func TestGetEffectiveness_Batch_Success(t *testing.T) {
	c1 := filledChain("corr-batch-1")
	c2 := filledChain("corr-batch-2")
	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{*c1, *c2}}
	uc := analyticalclient.NewGetEffectivenessUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Evaluations) != 2 {
		t.Fatalf("expected 2 evaluations, got %d", len(result.Evaluations))
	}
	if result.Meta.ChainsScanned != 2 {
		t.Errorf("chains_scanned=%d, want 2", result.Meta.ChainsScanned)
	}
}

func TestGetEffectiveness_Batch_SeverityFilter(t *testing.T) {
	c1 := filledChain("corr-high")
	c1.Execution.Risk.DecisionSeverity = "high"
	c2 := filledChain("corr-low")
	c2.Execution.Risk.DecisionSeverity = "low"

	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{*c1, *c2}}
	uc := analyticalclient.NewGetEffectivenessUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		Severity:   "high",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Evaluations) != 1 {
		t.Fatalf("expected 1 evaluation with severity filter, got %d", len(result.Evaluations))
	}
	if result.Evaluations[0].DecisionSeverity != "high" {
		t.Errorf("expected high severity, got %s", result.Evaluations[0].DecisionSeverity)
	}
}

func TestGetEffectiveness_Batch_StrategyTypeFilter(t *testing.T) {
	c1 := filledChain("corr-mr")
	c1.Execution.Risk.StrategyType = "mean_reversion_entry"
	c2 := filledChain("corr-tf")
	c2.Execution.Risk.StrategyType = "trend_following"

	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{*c1, *c2}}
	uc := analyticalclient.NewGetEffectivenessUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessQuery{
		Source:       "binance",
		Instrument:   instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:    60,
		StrategyType: "trend_following",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Evaluations) != 1 {
		t.Fatalf("expected 1 evaluation with strategy filter, got %d", len(result.Evaluations))
	}
	if result.Evaluations[0].StrategyType != "trend_following" {
		t.Errorf("expected trend_following, got %s", result.Evaluations[0].StrategyType)
	}
}

func TestGetEffectiveness_Batch_ValidationErrors(t *testing.T) {
	uc := analyticalclient.NewGetEffectivenessUseCase(&stubCompositeReader{}, slog.Default())

	tests := []struct {
		name  string
		query analyticalclient.EffectivenessQuery
	}{
		{"missing source", analyticalclient.EffectivenessQuery{Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60}},
		{"missing symbol", analyticalclient.EffectivenessQuery{Source: "binance", Timeframe: 60}},
		{"invalid timeframe", analyticalclient.EffectivenessQuery{Source: "binance", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 0}},
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

func TestGetEffectiveness_NilUseCase(t *testing.T) {
	var uc *analyticalclient.GetEffectivenessUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessQuery{
		CorrelationID: "x", Instrument: instrumentFromVenue("btcusdt"),
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}

func TestGetEffectiveness_Batch_MixedRejectedAndFilled(t *testing.T) {
	c1 := filledChain("corr-filled")
	c2 := fullChain("corr-rejected")
	c2.Execution.Status = "rejected"

	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{*c1, *c2}}
	uc := analyticalclient.NewGetEffectivenessUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Evaluations) != 1 {
		t.Fatalf("expected 1 evaluation (rejected excluded), got %d", len(result.Evaluations))
	}
	if result.Meta.Excluded != 1 {
		t.Errorf("excluded=%d, want 1", result.Meta.Excluded)
	}
}

// --- Decision Review Bundle Effectiveness Integration (S476) ---

func TestGetDecisionReview_EffectivenessSection_FilledExecution(t *testing.T) {
	chain := filledChain("corr-eff-review")
	reader := &stubCompositeReader{chain: chain}
	uc := analyticalclient.NewGetDecisionReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.DecisionReviewQuery{
		CorrelationID: "corr-eff-review",
		Instrument:    instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(result.Reviews))
	}

	r := result.Reviews[0]
	if r.Effectiveness == nil {
		t.Fatal("expected effectiveness section to be present for filled execution")
	}
	if r.Effectiveness.Outcome != "unresolved" {
		t.Errorf("outcome=%s, want unresolved (single-leg)", r.Effectiveness.Outcome)
	}
	if r.Effectiveness.EntryCostBasis != 5000.0 {
		t.Errorf("entry_cost_basis=%f, want 5000.0", r.Effectiveness.EntryCostBasis)
	}
	if r.Effectiveness.TotalFees != 0.50 {
		t.Errorf("total_fees=%f, want 0.50", r.Effectiveness.TotalFees)
	}
	if r.Effectiveness.FillCount != 1 {
		t.Errorf("fill_count=%d, want 1", r.Effectiveness.FillCount)
	}
	if r.Effectiveness.Explanation == "" {
		t.Error("expected effectiveness explanation to be non-empty")
	}
}

func TestGetDecisionReview_EffectivenessSection_NoExecution(t *testing.T) {
	chain := fullChain("corr-noexec-review")
	chain.Execution = nil
	chain.StageCount = 4
	chain.ChainComplete = false
	chain.MissingStages = []string{"execution"}

	reader := &stubCompositeReader{chain: chain}
	uc := analyticalclient.NewGetDecisionReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.DecisionReviewQuery{
		CorrelationID: "corr-noexec-review",
		Instrument:    instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(result.Reviews))
	}
	if result.Reviews[0].Effectiveness != nil {
		t.Error("expected effectiveness to be nil when execution is absent")
	}
}

func TestGetDecisionReview_EffectivenessSection_RejectedExecution(t *testing.T) {
	chain := fullChain("corr-rej-review")
	chain.Execution.Status = "rejected"

	reader := &stubCompositeReader{chain: chain}
	uc := analyticalclient.NewGetDecisionReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.DecisionReviewQuery{
		CorrelationID: "corr-rej-review",
		Instrument:    instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(result.Reviews))
	}
	if result.Reviews[0].Effectiveness != nil {
		t.Error("expected effectiveness to be nil for rejected execution")
	}
}

func TestGetDecisionReview_ExplanationIncludesEffectiveness(t *testing.T) {
	chain := filledChain("corr-expl-eff")
	reader := &stubCompositeReader{chain: chain}
	uc := analyticalclient.NewGetDecisionReviewUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.DecisionReviewQuery{
		CorrelationID: "corr-expl-eff",
		Instrument:    instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(result.Reviews))
	}

	explanation := result.Reviews[0].Explanation
	if !containsSubstr(explanation, "Effectiveness:") {
		t.Errorf("expected explanation to contain effectiveness info, got: %s", explanation)
	}
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
