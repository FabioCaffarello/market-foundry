package effectiveness

import (
	"testing"
	"time"

	"internal/domain/execution"
	"internal/domain/instrument"
)

func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

func TestValidOutcome(t *testing.T) {
	tests := []struct {
		outcome Outcome
		valid   bool
	}{
		{OutcomeWin, true},
		{OutcomeLoss, true},
		{OutcomeBreakeven, true},
		{OutcomeUnresolved, true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := ValidOutcome(tt.outcome); got != tt.valid {
			t.Errorf("ValidOutcome(%q) = %v, want %v", tt.outcome, got, tt.valid)
		}
	}
}

func TestClassify_RejectedReturnsNil(t *testing.T) {
	intent := makeIntent(t, execution.StatusRejected, execution.SideBuy, nil)
	attr := Classify(intent)
	if attr != nil {
		t.Fatal("expected nil for rejected order")
	}
}

func TestClassify_NonTerminalIsUnresolved(t *testing.T) {
	for _, status := range []execution.Status{
		execution.StatusSubmitted, execution.StatusSent, execution.StatusAccepted,
	} {
		intent := makeIntent(t, status, execution.SideBuy, nil)
		attr := Classify(intent)
		if attr == nil {
			t.Fatalf("status=%s: expected attribution, got nil", status)
		}
		if attr.Outcome != OutcomeUnresolved {
			t.Errorf("status=%s: outcome=%s, want unresolved", status, attr.Outcome)
		}
	}
}

func TestClassify_CancelledNoFillsIsUnresolved(t *testing.T) {
	intent := makeIntent(t, execution.StatusCancelled, execution.SideBuy, nil)
	attr := Classify(intent)
	if attr == nil {
		t.Fatal("expected attribution, got nil")
	}
	if attr.Outcome != OutcomeUnresolved {
		t.Error("cancelled with no fills should be unresolved")
	}
	if attr.FillCount != 0 {
		t.Errorf("fill_count=%d, want 0", attr.FillCount)
	}
}

func TestClassify_FilledSingleLegIsUnresolved(t *testing.T) {
	fills := []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.1", Fee: "0.50", FeeAsset: "USDT", CostBasis: "5000.00", Timestamp: time.Now()},
	}
	intent := makeIntent(t, execution.StatusFilled, execution.SideBuy, fills)
	attr := Classify(intent)
	if attr == nil {
		t.Fatal("expected attribution, got nil")
	}
	// Single-leg fills are always unresolved.
	if attr.Outcome != OutcomeUnresolved {
		t.Errorf("outcome=%s, want unresolved (single-leg)", attr.Outcome)
	}
	if attr.EntryCostBasis != 5000.0 {
		t.Errorf("entry_cost_basis=%f, want 5000.0", attr.EntryCostBasis)
	}
	if attr.TotalFees != 0.50 {
		t.Errorf("total_fees=%f, want 0.50", attr.TotalFees)
	}
	if attr.FillCount != 1 {
		t.Errorf("fill_count=%d, want 1", attr.FillCount)
	}
}

func TestClassify_ZeroCostBasisIsUnresolved(t *testing.T) {
	// Paper/dry-run fills may have zero cost basis.
	fills := []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.1", Fee: "0", CostBasis: "0", Simulated: true, Timestamp: time.Now()},
	}
	intent := makeIntent(t, execution.StatusFilled, execution.SideBuy, fills)
	attr := Classify(intent)
	if attr == nil {
		t.Fatal("expected attribution")
	}
	if attr.Outcome != OutcomeUnresolved {
		t.Errorf("outcome=%s, want unresolved (zero cost basis)", attr.Outcome)
	}
	if !attr.Simulated {
		t.Error("expected simulated=true")
	}
}

func TestClassify_PartiallyFilledIsUnresolved(t *testing.T) {
	fills := []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.05", Fee: "0.25", FeeAsset: "USDT", CostBasis: "2500.00", Timestamp: time.Now()},
	}
	intent := makeIntent(t, execution.StatusPartiallyFilled, execution.SideBuy, fills)
	attr := Classify(intent)
	if attr == nil {
		t.Fatal("expected attribution")
	}
	// Non-terminal status → unresolved.
	if attr.Outcome != OutcomeUnresolved {
		t.Errorf("outcome=%s, want unresolved (partially filled is non-terminal)", attr.Outcome)
	}
}

func TestClassify_AttributionCarriesContext(t *testing.T) {
	fills := []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.1", Fee: "0.50", CostBasis: "5000.00", Timestamp: time.Now()},
	}
	intent := makeIntent(t, execution.StatusFilled, execution.SideBuy, fills)
	intent.CorrelationID = "corr-123"
	intent.Risk.Type = "ema_crossover"
	intent.Risk.DecisionSeverity = "high"
	intent.Risk.StrategyType = "trend_following"
	intent.Instrument = btcUSDTPerp(t)
	intent.Source = "binance_spot"
	intent.Timeframe = 60

	attr := Classify(intent)
	if attr.CorrelationID != "corr-123" {
		t.Errorf("correlation_id=%s, want corr-123", attr.CorrelationID)
	}
	if attr.DecisionType != "ema_crossover" {
		t.Errorf("decision_type=%s, want ema_crossover", attr.DecisionType)
	}
	if attr.DecisionSeverity != "high" {
		t.Errorf("decision_severity=%s, want high", attr.DecisionSeverity)
	}
	if attr.StrategyType != "trend_following" {
		t.Errorf("strategy_type=%s, want trend_following", attr.StrategyType)
	}
	if attr.Symbol != "btcusdt" {
		t.Errorf("symbol=%s, want btcusdt", attr.Symbol)
	}
}

func TestClassifyPair_WinRoundTrip(t *testing.T) {
	entry := makeIntent(t, execution.StatusFilled, execution.SideBuy, []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.1", Fee: "0.50", CostBasis: "5000.00", Timestamp: time.Now()},
	})
	exit := makeIntent(t, execution.StatusFilled, execution.SideSell, []execution.FillRecord{
		{Price: "51000.00", Quantity: "0.1", Fee: "0.50", CostBasis: "5100.00", Timestamp: time.Now()},
	})

	attr := ClassifyPair(entry, exit)
	if attr == nil {
		t.Fatal("expected attribution")
	}
	if attr.Outcome != OutcomeWin {
		t.Errorf("outcome=%s, want win", attr.Outcome)
	}
	// Gross = exit - entry = 5100 - 5000 = 100
	if attr.GrossPnL != 100.0 {
		t.Errorf("gross_pnl=%f, want 100.0", attr.GrossPnL)
	}
	// Net = 100 - 1.0 (0.50 + 0.50) = 99.0
	if attr.NetPnL != 99.0 {
		t.Errorf("net_pnl=%f, want 99.0", attr.NetPnL)
	}
	if attr.FillCount != 2 {
		t.Errorf("fill_count=%d, want 2", attr.FillCount)
	}
}

func TestClassifyPair_LossRoundTrip(t *testing.T) {
	entry := makeIntent(t, execution.StatusFilled, execution.SideBuy, []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.1", Fee: "0.50", CostBasis: "5000.00", Timestamp: time.Now()},
	})
	exit := makeIntent(t, execution.StatusFilled, execution.SideSell, []execution.FillRecord{
		{Price: "49000.00", Quantity: "0.1", Fee: "0.50", CostBasis: "4900.00", Timestamp: time.Now()},
	})

	attr := ClassifyPair(entry, exit)
	if attr.Outcome != OutcomeLoss {
		t.Errorf("outcome=%s, want loss", attr.Outcome)
	}
	// Gross = 4900 - 5000 = -100
	if attr.GrossPnL != -100.0 {
		t.Errorf("gross_pnl=%f, want -100.0", attr.GrossPnL)
	}
	// Net = -100 - 1.0 = -101.0
	if attr.NetPnL != -101.0 {
		t.Errorf("net_pnl=%f, want -101.0", attr.NetPnL)
	}
}

func TestClassifyPair_ShortWin(t *testing.T) {
	entry := makeIntent(t, execution.StatusFilled, execution.SideSell, []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.1", Fee: "0.50", CostBasis: "5000.00", Timestamp: time.Now()},
	})
	exit := makeIntent(t, execution.StatusFilled, execution.SideBuy, []execution.FillRecord{
		{Price: "49000.00", Quantity: "0.1", Fee: "0.50", CostBasis: "4900.00", Timestamp: time.Now()},
	})

	attr := ClassifyPair(entry, exit)
	if attr.Outcome != OutcomeWin {
		t.Errorf("outcome=%s, want win (short)", attr.Outcome)
	}
	// For short: gross = entryCost - exitCost = 5000 - 4900 = 100
	if attr.GrossPnL != 100.0 {
		t.Errorf("gross_pnl=%f, want 100.0", attr.GrossPnL)
	}
}

func TestClassifyPair_Breakeven(t *testing.T) {
	entry := makeIntent(t, execution.StatusFilled, execution.SideBuy, []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.1", Fee: "0", CostBasis: "5000.00", Timestamp: time.Now()},
	})
	exit := makeIntent(t, execution.StatusFilled, execution.SideSell, []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.1", Fee: "0", CostBasis: "5000.00", Timestamp: time.Now()},
	})

	attr := ClassifyPair(entry, exit)
	if attr.Outcome != OutcomeBreakeven {
		t.Errorf("outcome=%s, want breakeven", attr.Outcome)
	}
}

func TestClassifyPair_RejectedReturnsNil(t *testing.T) {
	entry := makeIntent(t, execution.StatusRejected, execution.SideBuy, nil)
	exit := makeIntent(t, execution.StatusFilled, execution.SideSell, nil)
	if ClassifyPair(entry, exit) != nil {
		t.Error("rejected entry should return nil")
	}

	entry2 := makeIntent(t, execution.StatusFilled, execution.SideBuy, nil)
	exit2 := makeIntent(t, execution.StatusRejected, execution.SideSell, nil)
	if ClassifyPair(entry2, exit2) != nil {
		t.Error("rejected exit should return nil")
	}
}

func TestExplain_AllOutcomes(t *testing.T) {
	tests := []struct {
		attr     *Attribution
		contains string
	}{
		{nil, "No effectiveness data"},
		{&Attribution{Outcome: OutcomeUnresolved, FillCount: 0, Side: "buy", ExecutionStatus: "cancelled"}, "unresolved"},
		{&Attribution{Outcome: OutcomeUnresolved, FillCount: 1, Side: "buy", EntryCostBasis: 5000, TotalFees: 0.5}, "no paired exit"},
		{&Attribution{Outcome: OutcomeWin, NetPnL: 99.0, GrossPnL: 100.0, TotalFees: 1.0, FillCount: 2, Side: "buy", Symbol: "BTCUSDT"}, "WIN"},
		{&Attribution{Outcome: OutcomeLoss, NetPnL: -101.0, GrossPnL: -100.0, TotalFees: 1.0, FillCount: 2, Side: "buy", Symbol: "BTCUSDT"}, "LOSS"},
		{&Attribution{Outcome: OutcomeBreakeven, NetPnL: 0.00001, FillCount: 2, Side: "buy", Symbol: "BTCUSDT"}, "BREAKEVEN"},
	}
	for _, tt := range tests {
		got := tt.attr.Explain()
		if got == "" {
			t.Error("explain returned empty string")
		}
		if !containsStr(got, tt.contains) {
			t.Errorf("explain=%q, want to contain %q", got, tt.contains)
		}
	}
}

func TestClassify_MultipleFillsAggregated(t *testing.T) {
	fills := []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.05", Fee: "0.25", CostBasis: "2500.00", Timestamp: time.Now()},
		{Price: "50100.00", Quantity: "0.05", Fee: "0.25", CostBasis: "2505.00", Timestamp: time.Now()},
	}
	intent := makeIntent(t, execution.StatusFilled, execution.SideBuy, fills)
	attr := Classify(intent)
	if attr.EntryCostBasis != 5005.0 {
		t.Errorf("entry_cost_basis=%f, want 5005.0", attr.EntryCostBasis)
	}
	if attr.TotalFees != 0.50 {
		t.Errorf("total_fees=%f, want 0.50", attr.TotalFees)
	}
	if attr.FillCount != 2 {
		t.Errorf("fill_count=%d, want 2", attr.FillCount)
	}
}

// S499: ExitCostBasis tests.

func TestClassifyPair_ExitCostBasisPopulated(t *testing.T) {
	entry := makeIntent(t, execution.StatusFilled, execution.SideBuy, []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.1", Fee: "0.50", CostBasis: "5000.00", Timestamp: time.Now()},
	})
	exit := makeIntent(t, execution.StatusFilled, execution.SideSell, []execution.FillRecord{
		{Price: "51000.00", Quantity: "0.1", Fee: "0.50", CostBasis: "5100.00", Timestamp: time.Now()},
	})

	attr := ClassifyPair(entry, exit)
	if attr == nil {
		t.Fatal("expected attribution")
	}
	if attr.EntryCostBasis != 5000.0 {
		t.Errorf("entry_cost_basis=%f, want 5000.0", attr.EntryCostBasis)
	}
	if attr.ExitCostBasis != 5100.0 {
		t.Errorf("exit_cost_basis=%f, want 5100.0", attr.ExitCostBasis)
	}
}

func TestClassify_SingleLeg_ExitCostBasisIsZero(t *testing.T) {
	fills := []execution.FillRecord{
		{Price: "50000.00", Quantity: "0.1", Fee: "0.50", CostBasis: "5000.00", Timestamp: time.Now()},
	}
	intent := makeIntent(t, execution.StatusFilled, execution.SideBuy, fills)
	attr := Classify(intent)
	if attr == nil {
		t.Fatal("expected attribution")
	}
	if attr.ExitCostBasis != 0 {
		t.Errorf("exit_cost_basis=%f, want 0 (single-leg has no exit)", attr.ExitCostBasis)
	}
}

// --- helpers ---

func makeIntent(t *testing.T, status execution.Status, side execution.Side, fills []execution.FillRecord) execution.ExecutionIntent {
	t.Helper()
	return execution.ExecutionIntent{
		Type:       "market",
		Source:     "binance_spot",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       side,
		Quantity:   "0.1",
		Status:     status,
		Risk: execution.RiskInput{
			Type:        "ema_crossover",
			Disposition: "approved",
			Confidence:  "high",
		},
		Fills:     fills,
		Timestamp: time.Now(),
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
