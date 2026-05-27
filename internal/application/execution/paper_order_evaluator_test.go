package execution_test

import (
	"testing"
	"time"

	appexec "internal/application/execution"
	domainexec "internal/domain/execution"
)

func TestPaperOrderEvaluator_ApprovedLong_ProducesBuy(t *testing.T) {
	eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", btcUSDTPerp(t), 60)
	intent, ok := eval.Evaluate("position_exposure", "approved", "0.85", "0.02", "long", "0.72", "mean_reversion_entry", "high", 60, time.Now())
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if intent.Side != domainexec.SideBuy {
		t.Fatalf("expected SideBuy, got %q", intent.Side)
	}
	if intent.Quantity != "0.02" {
		t.Fatalf("expected quantity 0.02, got %q", intent.Quantity)
	}
	if intent.VenueSymbol() != "btcusdt" {
		t.Fatalf("expected symbol btcusdt, got %q", intent.VenueSymbol())
	}
}

func TestPaperOrderEvaluator_ApprovedShort_ProducesSell(t *testing.T) {
	eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", ethUSDTPerp(t), 300)
	intent, ok := eval.Evaluate("position_exposure", "approved", "0.85", "0.03", "short", "0.65", "trend_following_entry", "moderate", 300, time.Now())
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if intent.Side != domainexec.SideSell {
		t.Fatalf("expected SideSell, got %q", intent.Side)
	}
	if intent.Quantity != "0.03" {
		t.Fatalf("expected quantity 0.03, got %q", intent.Quantity)
	}
	if intent.VenueSymbol() != "ethusdt" {
		t.Fatalf("expected symbol ethusdt, got %q", intent.VenueSymbol())
	}
}

func TestPaperOrderEvaluator_Rejected_ProducesNone(t *testing.T) {
	eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", btcUSDTPerp(t), 60)
	intent, ok := eval.Evaluate("position_exposure", "rejected", "0.85", "0.02", "long", "0.72", "mean_reversion_entry", "high", 60, time.Now())
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if intent.Side != domainexec.SideNone {
		t.Fatalf("expected SideNone, got %q", intent.Side)
	}
	if intent.Quantity != "0" {
		t.Fatalf("expected quantity 0, got %q", intent.Quantity)
	}
}

func TestPaperOrderEvaluator_FlatStrategy_ProducesNone(t *testing.T) {
	eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", btcUSDTPerp(t), 60)
	intent, ok := eval.Evaluate("position_exposure", "approved", "0.85", "0.02", "flat", "0.50", "mean_reversion_entry", "low", 60, time.Now())
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if intent.Side != domainexec.SideNone {
		t.Fatalf("expected SideNone, got %q", intent.Side)
	}
	if intent.Quantity != "0" {
		t.Fatalf("expected quantity 0, got %q", intent.Quantity)
	}
}

func TestPaperOrderEvaluator_IntentIsFinalAndValid(t *testing.T) {
	eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", btcUSDTPerp(t), 60)
	intent, _ := eval.Evaluate("position_exposure", "approved", "0.85", "0.02", "long", "0.72", "mean_reversion_entry", "high", 60, time.Now())
	if !intent.Final {
		t.Fatal("expected Final=true")
	}
	if intent.Status != domainexec.StatusSubmitted {
		t.Fatalf("expected StatusSubmitted, got %q", intent.Status)
	}
	if prob := intent.Validate(); prob != nil {
		t.Fatalf("expected valid intent, got: %s", prob.Message)
	}
}

// ---------- S265: Boundary Context Preservation ----------

func TestPaperOrderEvaluator_RiskInput_PreservesStrategyTypeAndSeverity(t *testing.T) {
	eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", btcUSDTPerp(t), 60)
	intent, ok := eval.Evaluate(
		"position_exposure", "approved", "0.85", "0.02",
		"long", "0.72",
		"mean_reversion_entry", "high",
		60, time.Now(),
	)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}

	// S265: StrategyType must survive into RiskInput.
	if intent.Risk.StrategyType != "mean_reversion_entry" {
		t.Fatalf("expected RiskInput.StrategyType = mean_reversion_entry, got %q", intent.Risk.StrategyType)
	}

	// S265: DecisionSeverity must survive into RiskInput.
	if intent.Risk.DecisionSeverity != "high" {
		t.Fatalf("expected RiskInput.DecisionSeverity = high, got %q", intent.Risk.DecisionSeverity)
	}

	// S265: Parameters must also carry the context for observability.
	if intent.Parameters["strategy_type"] != "mean_reversion_entry" {
		t.Fatalf("expected Parameters[strategy_type] = mean_reversion_entry, got %q", intent.Parameters["strategy_type"])
	}
	if intent.Parameters["decision_severity"] != "high" {
		t.Fatalf("expected Parameters[decision_severity] = high, got %q", intent.Parameters["decision_severity"])
	}
}

func TestPaperOrderEvaluator_DrawdownRisk_ProducesBuyWithExposureQuantity(t *testing.T) {
	eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", btcUSDTPerp(t), 60)
	// S265: Drawdown risk should pass MaxExposure (e.g., "0.0500") as position constraint,
	// not StopDistance. This test verifies the correct semantic mapping.
	intent, ok := eval.Evaluate(
		"drawdown_limit", "approved", "0.7650", "0.0500",
		"long", "0.72",
		"mean_reversion_entry", "high",
		60, time.Now(),
	)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if intent.Side != domainexec.SideBuy {
		t.Fatalf("expected SideBuy, got %q", intent.Side)
	}
	if intent.Quantity != "0.0500" {
		t.Fatalf("expected quantity 0.0500 (drawdown MaxExposure), got %q", intent.Quantity)
	}
	if intent.Risk.Type != "drawdown_limit" {
		t.Fatalf("expected risk type drawdown_limit, got %q", intent.Risk.Type)
	}
}

// ---------- Multi-Symbol Isolation ----------

func TestPaperOrderEvaluator_MultiSymbol_IndependentEvaluation(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	timeframes := []int{60, 300}
	ts := time.Now()

	partitionKeys := make(map[string]string) // key → symbol
	dedupKeys := make(map[string]string)     // key → symbol

	for _, sym := range symbols {
		for _, tf := range timeframes {
			eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", instrumentFromVenueSymbol(t, "binancef", sym), tf)
			intent, ok := eval.Evaluate("position_exposure", "approved", "0.85", "0.02", "long", "0.72", "mean_reversion_entry", "high", tf, ts)
			if !ok {
				t.Fatalf("evaluation failed for %s/%d", sym, tf)
			}

			// Verify symbol/timeframe ownership.
			if intent.VenueSymbol() != sym {
				t.Fatalf("expected symbol %q, got %q", sym, intent.VenueSymbol())
			}
			if intent.Timeframe != tf {
				t.Fatalf("expected timeframe %d, got %d", tf, intent.Timeframe)
			}

			// Verify intent passes domain validation.
			if prob := intent.Validate(); prob != nil {
				t.Fatalf("intent for %s/%d invalid: %s", sym, tf, prob.Message)
			}

			// Check partition key isolation.
			pk := intent.PartitionKey()
			if existing, collision := partitionKeys[pk]; collision {
				t.Fatalf("partition key collision: %q used by both %q and %q", pk, existing, sym)
			}
			partitionKeys[pk] = sym

			// Check dedup key isolation.
			dk := intent.DeduplicationKey()
			if existing, collision := dedupKeys[dk]; collision {
				t.Fatalf("dedup key collision: %q used by both %q and %q", dk, existing, sym)
			}
			dedupKeys[dk] = sym
		}
	}

	expectedCount := len(symbols) * len(timeframes)
	if len(partitionKeys) != expectedCount {
		t.Fatalf("expected %d unique partition keys, got %d", expectedCount, len(partitionKeys))
	}
	if len(dedupKeys) != expectedCount {
		t.Fatalf("expected %d unique dedup keys, got %d", expectedCount, len(dedupKeys))
	}
}

func TestPaperOrderEvaluator_MultiSymbol_DifferentDispositions(t *testing.T) {
	// Verify that per-symbol evaluators with different risk dispositions produce
	// correct, independent results.
	ts := time.Now()

	// btcusdt: approved long → buy
	evalBTC := appexec.NewPaperOrderEvaluatorForInstrument("binancef", btcUSDTPerp(t), 60)
	btc, _ := evalBTC.Evaluate("position_exposure", "approved", "0.85", "0.02", "long", "0.72", "mean_reversion_entry", "high", 60, ts)

	// ethusdt: rejected → none
	evalETH := appexec.NewPaperOrderEvaluatorForInstrument("binancef", ethUSDTPerp(t), 60)
	eth, _ := evalETH.Evaluate("position_exposure", "rejected", "0.30", "0.02", "long", "0.72", "mean_reversion_entry", "low", 60, ts)

	// solusdt: approved short → sell
	evalSOL := appexec.NewPaperOrderEvaluatorForInstrument("binancef", solUSDTPerp(t), 60)
	sol, _ := evalSOL.Evaluate("position_exposure", "approved", "0.90", "0.05", "short", "0.80", "trend_following_entry", "high", 60, ts)

	if btc.Side != domainexec.SideBuy {
		t.Fatalf("btcusdt: expected SideBuy, got %q", btc.Side)
	}
	if eth.Side != domainexec.SideNone {
		t.Fatalf("ethusdt: expected SideNone, got %q", eth.Side)
	}
	if sol.Side != domainexec.SideSell {
		t.Fatalf("solusdt: expected SideSell, got %q", sol.Side)
	}

	// No bleed: each intent owns its correct symbol.
	if btc.VenueSymbol() != "btcusdt" || eth.VenueSymbol() != "ethusdt" || sol.VenueSymbol() != "solusdt" {
		t.Fatal("symbol ownership bleed detected")
	}

	// All three must have distinct partition keys.
	keys := map[string]bool{btc.PartitionKey(): true, eth.PartitionKey(): true, sol.PartitionKey(): true}
	if len(keys) != 3 {
		t.Fatalf("expected 3 unique partition keys, got %d", len(keys))
	}
}

func TestPaperOrderEvaluator_MultiSymbol_ModifiedDisposition(t *testing.T) {
	ts := time.Now()
	eval := appexec.NewPaperOrderEvaluatorForInstrument("binancef", btcUSDTPerp(t), 60)
	intent, ok := eval.Evaluate("position_exposure", "modified", "0.60", "0.01", "long", "0.72", "mean_reversion_entry", "moderate", 60, ts)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if intent.Side != domainexec.SideBuy {
		t.Fatalf("expected SideBuy for modified+long, got %q", intent.Side)
	}
	if intent.Quantity != "0.01" {
		t.Fatalf("expected quantity 0.01 (risk-adjusted), got %q", intent.Quantity)
	}
}
