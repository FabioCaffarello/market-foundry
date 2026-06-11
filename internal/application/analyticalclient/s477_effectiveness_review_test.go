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

// --- Effectiveness Summary Use Case Tests (S477) ---

// filledChainWithContext builds a chain with specified decision/strategy context.
func filledChainWithContext(corrID, decisionType, strategyType, severity string) *analyticalclient.CompositeExecutionChain {
	chain := filledChain(corrID)
	chain.Execution.Risk.Type = decisionType
	chain.Execution.Risk.StrategyType = strategyType
	chain.Execution.Risk.DecisionSeverity = severity
	return chain
}

func TestGetEffectivenessSummary_Ungrouped_SingleCohort(t *testing.T) {
	c1 := filledChain("corr-s1")
	c2 := filledChain("corr-s2")

	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{*c1, *c2}}
	uc := analyticalclient.NewGetEffectivenessSummaryUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessSummaryQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Cohorts) != 1 {
		t.Fatalf("expected 1 cohort (ungrouped), got %d", len(result.Cohorts))
	}

	cs := result.Cohorts[0]
	if cs.Key != "all" {
		t.Errorf("key=%s, want all", cs.Key)
	}
	if cs.Evaluated != 2 {
		t.Errorf("evaluated=%d, want 2", cs.Evaluated)
	}
	if result.Meta.ChainsScanned != 2 {
		t.Errorf("chains_scanned=%d, want 2", result.Meta.ChainsScanned)
	}
}

func TestGetEffectivenessSummary_GroupByDecisionType(t *testing.T) {
	c1 := filledChainWithContext("corr-g1", "rsi_oversold", "mean_reversion", "high")
	c2 := filledChainWithContext("corr-g2", "rsi_oversold", "mean_reversion", "high")
	c3 := filledChainWithContext("corr-g3", "ema_crossover", "trend_following", "moderate")

	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{*c1, *c2, *c3}}
	uc := analyticalclient.NewGetEffectivenessSummaryUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessSummaryQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		GroupBy:    "decision_type",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Cohorts) != 2 {
		t.Fatalf("expected 2 cohorts by decision_type, got %d", len(result.Cohorts))
	}

	// Sorted by evaluated count descending.
	if result.Cohorts[0].Key != "rsi_oversold" {
		t.Errorf("first cohort key=%s, want rsi_oversold (highest count)", result.Cohorts[0].Key)
	}
	if result.Cohorts[0].Evaluated != 2 {
		t.Errorf("rsi_oversold evaluated=%d, want 2", result.Cohorts[0].Evaluated)
	}
	if result.Cohorts[1].Key != "ema_crossover" {
		t.Errorf("second cohort key=%s, want ema_crossover", result.Cohorts[1].Key)
	}
	if result.Cohorts[1].Evaluated != 1 {
		t.Errorf("ema_crossover evaluated=%d, want 1", result.Cohorts[1].Evaluated)
	}
}

func TestGetEffectivenessSummary_GroupByStrategyType(t *testing.T) {
	c1 := filledChainWithContext("corr-st1", "rsi", "mean_reversion", "high")
	c2 := filledChainWithContext("corr-st2", "ema", "trend_following", "moderate")

	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{*c1, *c2}}
	uc := analyticalclient.NewGetEffectivenessSummaryUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessSummaryQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		GroupBy:    "strategy_type",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Cohorts) != 2 {
		t.Fatalf("expected 2 cohorts by strategy_type, got %d", len(result.Cohorts))
	}
}

func TestGetEffectivenessSummary_GroupBySeverity(t *testing.T) {
	c1 := filledChainWithContext("corr-sv1", "rsi", "mr", "high")
	c2 := filledChainWithContext("corr-sv2", "rsi", "mr", "moderate")
	c3 := filledChainWithContext("corr-sv3", "rsi", "mr", "high")

	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{*c1, *c2, *c3}}
	uc := analyticalclient.NewGetEffectivenessSummaryUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessSummaryQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		GroupBy:    "severity",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Cohorts) != 2 {
		t.Fatalf("expected 2 cohorts by severity, got %d", len(result.Cohorts))
	}
	if result.Cohorts[0].Key != "high" {
		t.Errorf("first cohort key=%s, want high (2 evaluated)", result.Cohorts[0].Key)
	}
	if result.Cohorts[0].Evaluated != 2 {
		t.Errorf("high evaluated=%d, want 2", result.Cohorts[0].Evaluated)
	}
}

func TestGetEffectivenessSummary_InvalidGroupBy(t *testing.T) {
	uc := analyticalclient.NewGetEffectivenessSummaryUseCase(&stubCompositeReader{}, slog.Default())

	_, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessSummaryQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
		GroupBy:    "invalid_dimension",
	})
	if prob == nil {
		t.Fatal("expected validation problem for invalid group_by")
	}
	if prob.Code != problem.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %s", prob.Code)
	}
}

func TestGetEffectivenessSummary_ValidationErrors(t *testing.T) {
	uc := analyticalclient.NewGetEffectivenessSummaryUseCase(&stubCompositeReader{}, slog.Default())

	tests := []struct {
		name  string
		query analyticalclient.EffectivenessSummaryQuery
	}{
		{"missing source", analyticalclient.EffectivenessSummaryQuery{Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60}},
		{"missing symbol", analyticalclient.EffectivenessSummaryQuery{Source: "binance", Timeframe: 60}},
		{"invalid timeframe", analyticalclient.EffectivenessSummaryQuery{Source: "binance", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 0}},
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

func TestGetEffectivenessSummary_NilUseCase(t *testing.T) {
	var uc *analyticalclient.GetEffectivenessSummaryUseCase
	_, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessSummaryQuery{
		Source: "binance", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected problem for nil use case")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}

func TestGetEffectivenessSummary_EmptyResult(t *testing.T) {
	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{}}
	uc := analyticalclient.NewGetEffectivenessSummaryUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessSummaryQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(result.Cohorts) != 1 {
		t.Fatalf("expected 1 cohort (empty ungrouped), got %d", len(result.Cohorts))
	}
	cs := result.Cohorts[0]
	if cs.Evaluated != 0 {
		t.Errorf("evaluated=%d, want 0", cs.Evaluated)
	}
	if cs.WinRate != 0 {
		t.Errorf("win_rate=%f, want 0 for empty cohort", cs.WinRate)
	}
	if cs.AvgPnL != 0 {
		t.Errorf("avg_pnl=%f, want 0 for empty cohort", cs.AvgPnL)
	}
}

func TestGetEffectivenessSummary_RejectedExcluded(t *testing.T) {
	c1 := filledChain("corr-ok")
	c2 := fullChain("corr-rej")
	c2.Execution.Status = "rejected"

	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{*c1, *c2}}
	uc := analyticalclient.NewGetEffectivenessSummaryUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessSummaryQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Meta.Excluded != 1 {
		t.Errorf("excluded=%d, want 1", result.Meta.Excluded)
	}
	if result.Cohorts[0].Evaluated != 1 {
		t.Errorf("evaluated=%d, want 1", result.Cohorts[0].Evaluated)
	}
}

func TestGetEffectivenessSummary_PreAggregationFilter(t *testing.T) {
	c1 := filledChainWithContext("corr-f1", "rsi", "mr", "high")
	c2 := filledChainWithContext("corr-f2", "ema", "tf", "moderate")

	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{*c1, *c2}}
	uc := analyticalclient.NewGetEffectivenessSummaryUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessSummaryQuery{
		Source:       "binance",
		Instrument:   instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:    60,
		DecisionType: "rsi",
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if result.Cohorts[0].Evaluated != 1 {
		t.Errorf("evaluated=%d, want 1 (filtered to rsi only)", result.Cohorts[0].Evaluated)
	}
}

func TestGetEffectivenessSummary_WinRateComputation(t *testing.T) {
	// Create chains with round-trip pairs for actual win/loss classification.
	entry := filledChain("corr-win-entry")
	entry.Execution.Status = "filled"
	entry.Execution.Side = execution.SideBuy
	entry.Execution.Fills = []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.1", Fee: "0.00", FeeAsset: "USDT", CostBasis: "5000.00", Timestamp: time.Now()},
	}

	// Single-leg fills are unresolved, so win_rate for these should be 0.
	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{*entry}}
	uc := analyticalclient.NewGetEffectivenessSummaryUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessSummaryQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	cs := result.Cohorts[0]
	if cs.UnresolvedCount != 1 {
		t.Errorf("unresolved_count=%d, want 1 (single-leg fills)", cs.UnresolvedCount)
	}
	if cs.Resolved != 0 {
		t.Errorf("resolved=%d, want 0 (all unresolved)", cs.Resolved)
	}
	if cs.WinRate != 0 {
		t.Errorf("win_rate=%f, want 0 when all unresolved", cs.WinRate)
	}
}

func TestGetEffectivenessSummary_TotalFeesAccumulated(t *testing.T) {
	c1 := filledChain("corr-fee1")
	c1.Execution.Fills = []execution.FillRecord{
		{Price: "50000", Quantity: "0.1", Fee: "1.50", FeeAsset: "USDT", CostBasis: "5000", Timestamp: time.Now()},
	}
	c2 := filledChain("corr-fee2")
	c2.Execution.Fills = []execution.FillRecord{
		{Price: "50000", Quantity: "0.1", Fee: "2.00", FeeAsset: "USDT", CostBasis: "5000", Timestamp: time.Now()},
	}

	reader := &stubCompositeReader{chains: []analyticalclient.CompositeExecutionChain{*c1, *c2}}
	uc := analyticalclient.NewGetEffectivenessSummaryUseCase(reader, slog.Default())

	result, prob := uc.Execute(context.Background(), analyticalclient.EffectivenessSummaryQuery{
		Source:     "binance",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	cs := result.Cohorts[0]
	if cs.TotalFees != 3.50 {
		t.Errorf("total_fees=%f, want 3.50", cs.TotalFees)
	}
}

// --- ValidGroupBy Tests ---

func TestValidGroupBy(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", true},
		{"decision_type", true},
		{"strategy_type", true},
		{"severity", true},
		{"source", true},
		{"invalid", false},
		{"timeframe", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := analyticalclient.ValidGroupBy(tt.input); got != tt.want {
				t.Errorf("ValidGroupBy(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
