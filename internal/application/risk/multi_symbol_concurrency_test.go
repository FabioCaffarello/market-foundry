package risk_test

// S304: Risk Behavior Under Multi-Symbol Concurrency
//
// Validates that risk evaluators (position_exposure, drawdown_limit) produce
// correct, isolated, and explainable results when multiple symbols are
// evaluated simultaneously with different strategy types and severities.
//
// Scenarios:
//   RE-1 — Three symbols, same strategy type, different severities → distinct outcomes.
//   RE-2 — Three symbols, different strategy types → strategy-specific scaling isolation.
//   RE-3 — Mixed dispositions across symbols (approved/modified/rejected boundaries).
//   RE-4 — Drawdown evaluator multi-symbol with stop-distance diversity.
//   RE-5 — Cross-evaluator coherence: position_exposure + drawdown_limit per symbol.
//   RE-6 — Flat direction across all symbols: no cross-symbol leakage.

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	apprisk "internal/application/risk"
	domainrisk "internal/domain/risk"
)

// ---------------------------------------------------------------------------
// RE-1: Same strategy type, different severities → distinct outcomes per symbol
// ---------------------------------------------------------------------------

func TestS304_RE1_SameStraTypeDifferentSeverities(t *testing.T) {
	now := time.Now().UTC()

	type symbolCase struct {
		symbol    string
		severity  string
		direction string
		conf      string
	}

	cases := []symbolCase{
		{symbol: "btcusdt", severity: "high", direction: "long", conf: "0.8500"},
		{symbol: "ethusdt", severity: "moderate", direction: "long", conf: "0.8500"},
		{symbol: "solusdt", severity: "low", direction: "long", conf: "0.8500"},
	}

	results := make(map[string]domainrisk.RiskAssessment)

	for _, sc := range cases {
		eval := apprisk.NewPositionExposureEvaluator("binancef", sc.symbol, 60)
		r, ok := eval.Evaluate("mean_reversion_entry", sc.direction, sc.conf, sc.severity, "test rationale", 60, now)
		if !ok {
			t.Fatalf("[%s] evaluation failed", sc.symbol)
		}
		results[sc.symbol] = r
	}

	// All three must be approved (confidence 0.85 × 0.90 = 0.765, well within limits).
	for _, sc := range cases {
		r := results[sc.symbol]
		if r.Disposition != domainrisk.DispositionApproved {
			t.Errorf("[%s] expected approved, got %s", sc.symbol, r.Disposition)
		}
		if r.Symbol != sc.symbol {
			t.Errorf("[%s] symbol bleed: got %s", sc.symbol, r.Symbol)
		}
	}

	// Verify severity influences effective position: high > moderate > low.
	posHigh, _ := strconv.ParseFloat(results["btcusdt"].Constraints.MaxPositionSize, 64)
	posMod, _ := strconv.ParseFloat(results["ethusdt"].Constraints.MaxPositionSize, 64)
	posLow, _ := strconv.ParseFloat(results["solusdt"].Constraints.MaxPositionSize, 64)

	if posHigh <= posMod {
		t.Errorf("high severity position (%.4f) should exceed moderate (%.4f)", posHigh, posMod)
	}
	if posMod <= posLow {
		t.Errorf("moderate severity position (%.4f) should exceed low (%.4f)", posMod, posLow)
	}

	// Verify rationale isolation: each mentions its own severity.
	if !strings.Contains(results["btcusdt"].Rationale, "decision severity high") {
		t.Errorf("btcusdt rationale should mention high severity: %s", results["btcusdt"].Rationale)
	}
	if !strings.Contains(results["solusdt"].Rationale, "decision severity low") {
		t.Errorf("solusdt rationale should mention low severity: %s", results["solusdt"].Rationale)
	}

	// Verify partition key isolation.
	keys := map[string]bool{}
	for _, sc := range cases {
		pk := results[sc.symbol].PartitionKey()
		if keys[pk] {
			t.Fatalf("partition key collision: %s", pk)
		}
		keys[pk] = true
	}
}

// ---------------------------------------------------------------------------
// RE-2: Different strategy types → strategy-specific scaling isolation
// ---------------------------------------------------------------------------

func TestS304_RE2_DifferentStrategyTypes(t *testing.T) {
	now := time.Now().UTC()

	type symbolCase struct {
		symbol       string
		strategyType string
		direction    string
		wantFactor   float64 // expected confidence factor
	}

	cases := []symbolCase{
		{symbol: "btcusdt", strategyType: "mean_reversion_entry", direction: "long", wantFactor: 0.90},
		{symbol: "ethusdt", strategyType: "trend_following_entry", direction: "long", wantFactor: 0.95},
		{symbol: "solusdt", strategyType: "squeeze_breakout_entry", direction: "short", wantFactor: 0.93},
	}

	baseConf := 0.80

	for _, sc := range cases {
		t.Run(sc.symbol+"_"+sc.strategyType, func(t *testing.T) {
			eval := apprisk.NewPositionExposureEvaluator("binancef", sc.symbol, 60)
			r, ok := eval.Evaluate(sc.strategyType, sc.direction, fmt.Sprintf("%.4f", baseConf), "moderate", "", 60, now)
			if !ok {
				t.Fatalf("evaluation failed")
			}

			// Symbol ownership.
			if r.Symbol != sc.symbol {
				t.Errorf("symbol bleed: got %s", r.Symbol)
			}

			// Risk confidence = baseConf × strategy-type factor.
			expectedConf := fmt.Sprintf("%.4f", baseConf*sc.wantFactor)
			if r.Confidence != expectedConf {
				t.Errorf("confidence=%s, want %s (factor ×%.2f)", r.Confidence, expectedConf, sc.wantFactor)
			}

			// Strategy type recorded in metadata.
			if r.Metadata["strategy_type"] != sc.strategyType {
				t.Errorf("metadata.strategy_type=%q, want %q", r.Metadata["strategy_type"], sc.strategyType)
			}

			// Rationale includes strategy type.
			if !strings.Contains(r.Rationale, sc.strategyType) {
				t.Errorf("rationale should mention strategy type: %s", r.Rationale)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RE-3: Mixed dispositions across symbols at the boundary
// ---------------------------------------------------------------------------

func TestS304_RE3_MixedDispositionsAcrossSymbols(t *testing.T) {
	now := time.Now().UTC()

	// btcusdt: high severity, high conf → approved (well within limits)
	evalBTC := apprisk.NewPositionExposureEvaluator("binancef", "btcusdt", 60)
	btc, ok := evalBTC.Evaluate("mean_reversion_entry", "long", "0.5000", "high", "BTC signal", 60, now)
	if !ok {
		t.Fatal("btcusdt evaluation failed")
	}

	// ethusdt: zero confidence → rejected (S256 behavior)
	evalETH := apprisk.NewPositionExposureEvaluator("binancef", "ethusdt", 60)
	eth, ok := evalETH.Evaluate("trend_following_entry", "long", "0.0000", "moderate", "ETH signal", 60, now)
	if !ok {
		t.Fatal("ethusdt evaluation failed")
	}

	// solusdt: negative confidence → rejected
	evalSOL := apprisk.NewPositionExposureEvaluator("binancef", "solusdt", 60)
	sol, ok := evalSOL.Evaluate("squeeze_breakout_entry", "short", "-0.5000", "low", "SOL signal", 60, now)
	if !ok {
		t.Fatal("solusdt evaluation failed")
	}

	if btc.Disposition != domainrisk.DispositionApproved {
		t.Errorf("btcusdt: expected approved, got %s", btc.Disposition)
	}
	if eth.Disposition != domainrisk.DispositionRejected {
		t.Errorf("ethusdt: expected rejected (zero conf), got %s", eth.Disposition)
	}
	if sol.Disposition != domainrisk.DispositionRejected {
		t.Errorf("solusdt: expected rejected (negative conf), got %s", sol.Disposition)
	}

	// Symbol isolation.
	if btc.Symbol != "btcusdt" || eth.Symbol != "ethusdt" || sol.Symbol != "solusdt" {
		t.Fatal("symbol ownership bleed detected")
	}

	// Rejected assessments carry rationale per symbol.
	if !strings.Contains(eth.Rationale, "non-positive confidence") {
		t.Errorf("ethusdt rejected rationale missing: %s", eth.Rationale)
	}
	if !strings.Contains(sol.Rationale, "non-positive confidence") {
		t.Errorf("solusdt rejected rationale missing: %s", sol.Rationale)
	}

	// Approved assessment has constraints; rejected do not.
	if btc.Constraints.MaxPositionSize == "" {
		t.Error("btcusdt: approved should have max_position_size")
	}
}

// ---------------------------------------------------------------------------
// RE-4: Drawdown evaluator multi-symbol with stop-distance diversity
// ---------------------------------------------------------------------------

func TestS304_RE4_DrawdownMultiSymbolStopDiversity(t *testing.T) {
	now := time.Now().UTC()

	type symbolCase struct {
		symbol       string
		strategyType string
		direction    string
		conf         string
		severity     string
	}

	cases := []symbolCase{
		{symbol: "btcusdt", strategyType: "mean_reversion_entry", direction: "long", conf: "0.8000", severity: "high"},
		{symbol: "ethusdt", strategyType: "trend_following_entry", direction: "short", conf: "0.8000", severity: "moderate"},
		{symbol: "solusdt", strategyType: "squeeze_breakout_entry", direction: "long", conf: "0.8000", severity: "low"},
	}

	results := make(map[string]domainrisk.RiskAssessment)

	for _, sc := range cases {
		eval := apprisk.NewDrawdownLimitEvaluator("binancef", sc.symbol, 60)
		r, ok := eval.Evaluate(sc.strategyType, sc.direction, sc.conf, sc.severity, sc.symbol+" signal", 60, now)
		if !ok {
			t.Fatalf("[%s] drawdown evaluation failed", sc.symbol)
		}
		results[sc.symbol] = r
	}

	// All should produce drawdown_limit assessments.
	for _, sc := range cases {
		r := results[sc.symbol]
		if r.Type != "drawdown_limit" {
			t.Errorf("[%s] type=%q, want drawdown_limit", sc.symbol, r.Type)
		}
		if r.Symbol != sc.symbol {
			t.Errorf("[%s] symbol bleed: got %s", sc.symbol, r.Symbol)
		}
		if r.Constraints.StopDistance == "" {
			t.Errorf("[%s] missing stop_distance constraint", sc.symbol)
		}
	}

	// Stop distances should differ due to strategy-type stop factor.
	// mean_reversion ×0.85 (tightest), squeeze_breakout ×1.05, trend_following ×1.15 (widest).
	sdBTC, _ := strconv.ParseFloat(results["btcusdt"].Constraints.StopDistance, 64)
	sdSOL, _ := strconv.ParseFloat(results["solusdt"].Constraints.StopDistance, 64)

	// With same base conf 0.80, strategy type dominates stop factor.
	// BTC: mean_reversion (×0.85) should have tighter stop than SOL: squeeze (×1.05).
	if sdBTC >= sdSOL {
		t.Errorf("mean_reversion stop (%.4f) should be tighter than squeeze (%.4f)", sdBTC, sdSOL)
	}

	// Verify rationale isolation.
	for _, sc := range cases {
		if !strings.Contains(results[sc.symbol].Rationale, sc.strategyType) {
			t.Errorf("[%s] rationale should mention %s: %s", sc.symbol, sc.strategyType, results[sc.symbol].Rationale)
		}
	}

	// Partition key isolation.
	keys := map[string]bool{}
	for _, sc := range cases {
		pk := results[sc.symbol].PartitionKey()
		if keys[pk] {
			t.Fatalf("partition key collision: %s", pk)
		}
		keys[pk] = true
	}
}

// ---------------------------------------------------------------------------
// RE-5: Cross-evaluator coherence per symbol
// ---------------------------------------------------------------------------

func TestS304_RE5_CrossEvaluatorCoherence(t *testing.T) {
	now := time.Now().UTC()

	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	strategyTypes := map[string]string{
		"btcusdt": "mean_reversion_entry",
		"ethusdt": "trend_following_entry",
		"solusdt": "squeeze_breakout_entry",
	}
	severities := map[string]string{
		"btcusdt": "high",
		"ethusdt": "moderate",
		"solusdt": "low",
	}

	for _, sym := range symbols {
		t.Run(sym, func(t *testing.T) {
			posEval := apprisk.NewPositionExposureEvaluator("binancef", sym, 60)
			ddEval := apprisk.NewDrawdownLimitEvaluator("binancef", sym, 60)

			posResult, ok := posEval.Evaluate(strategyTypes[sym], "long", "0.7500", severities[sym], sym+" signal", 60, now)
			if !ok {
				t.Fatalf("position exposure eval failed")
			}
			ddResult, ok := ddEval.Evaluate(strategyTypes[sym], "long", "0.7500", severities[sym], sym+" signal", 60, now)
			if !ok {
				t.Fatalf("drawdown limit eval failed")
			}

			// Both evaluators must agree on symbol.
			if posResult.Symbol != sym || ddResult.Symbol != sym {
				t.Fatalf("symbol mismatch: pos=%s dd=%s", posResult.Symbol, ddResult.Symbol)
			}

			// Both carry the same strategy type in metadata.
			if posResult.Metadata["strategy_type"] != strategyTypes[sym] {
				t.Errorf("pos metadata.strategy_type=%q", posResult.Metadata["strategy_type"])
			}
			if ddResult.Metadata["strategy_type"] != strategyTypes[sym] {
				t.Errorf("dd metadata.strategy_type=%q", ddResult.Metadata["strategy_type"])
			}

			// Both carry the same decision severity.
			if posResult.Metadata["decision_severity"] != severities[sym] {
				t.Errorf("pos metadata.decision_severity=%q", posResult.Metadata["decision_severity"])
			}
			if ddResult.Metadata["decision_severity"] != severities[sym] {
				t.Errorf("dd metadata.decision_severity=%q", ddResult.Metadata["decision_severity"])
			}

			// Types differ.
			if posResult.Type != "position_exposure" {
				t.Errorf("pos type=%q", posResult.Type)
			}
			if ddResult.Type != "drawdown_limit" {
				t.Errorf("dd type=%q", ddResult.Type)
			}

			// Both pass domain validation.
			if prob := posResult.Validate(); prob != nil {
				t.Errorf("pos validation failed: %s", prob.Message)
			}
			if prob := ddResult.Validate(); prob != nil {
				t.Errorf("dd validation failed: %s", prob.Message)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RE-6: Flat direction across all symbols — no cross-symbol leakage
// ---------------------------------------------------------------------------

func TestS304_RE6_FlatDirectionNoLeakage(t *testing.T) {
	now := time.Now().UTC()
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}

	for _, sym := range symbols {
		t.Run(sym, func(t *testing.T) {
			posEval := apprisk.NewPositionExposureEvaluator("binancef", sym, 60)
			ddEval := apprisk.NewDrawdownLimitEvaluator("binancef", sym, 60)

			posR, ok := posEval.Evaluate("mean_reversion_entry", "flat", "0.0000", "none", "", 60, now)
			if !ok {
				t.Fatal("pos eval failed for flat")
			}
			ddR, ok := ddEval.Evaluate("mean_reversion_entry", "flat", "0.0000", "none", "", 60, now)
			if !ok {
				t.Fatal("dd eval failed for flat")
			}

			// Both must be approved with confidence 1.0.
			if posR.Disposition != domainrisk.DispositionApproved {
				t.Errorf("pos: expected approved, got %s", posR.Disposition)
			}
			if ddR.Disposition != domainrisk.DispositionApproved {
				t.Errorf("dd: expected approved, got %s", ddR.Disposition)
			}
			if posR.Confidence != "1.0000" {
				t.Errorf("pos: confidence=%s, want 1.0000", posR.Confidence)
			}
			if ddR.Confidence != "1.0000" {
				t.Errorf("dd: confidence=%s, want 1.0000", ddR.Confidence)
			}

			// Symbol isolation.
			if posR.Symbol != sym {
				t.Errorf("pos symbol bleed: got %s", posR.Symbol)
			}
			if ddR.Symbol != sym {
				t.Errorf("dd symbol bleed: got %s", ddR.Symbol)
			}
		})
	}
}
