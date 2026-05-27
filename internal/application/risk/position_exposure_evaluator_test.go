package risk_test

import (
	"strings"
	"testing"
	"time"

	apprisk "internal/application/risk"
	domainrisk "internal/domain/risk"
)

func TestPositionExposureEvaluator_LongApproved(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", "low", "RSI 28.50 below threshold", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Disposition != domainrisk.DispositionApproved {
		t.Fatalf("expected approved, got %s", r.Disposition)
	}
	if r.Type != "position_exposure" {
		t.Fatalf("expected type position_exposure, got %s", r.Type)
	}
	if r.Constraints.MaxPositionSize == "" {
		t.Fatal("expected max_position_size constraint")
	}
	if !r.Final {
		t.Fatal("expected final=true")
	}
	if len(r.Strategies) != 1 {
		t.Fatalf("expected 1 strategy input, got %d", len(r.Strategies))
	}
	if r.Strategies[0].Type != "mean_reversion_entry" || r.Strategies[0].Direction != "long" {
		t.Fatalf("unexpected strategy input: %+v", r.Strategies[0])
	}
	// S251: confidence = 0.85 × 0.90 (mean_reversion factor)
	if r.Confidence != "0.7650" {
		t.Errorf("expected risk confidence 0.7650 (mean_reversion ×0.90), got %s", r.Confidence)
	}
}

func TestPositionExposureEvaluator_ShortApproved(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "short", "0.7500", "moderate", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Disposition != domainrisk.DispositionApproved {
		t.Fatalf("expected approved, got %s", r.Disposition)
	}
	if r.Strategies[0].Direction != "short" {
		t.Fatalf("expected short direction, got %s", r.Strategies[0].Direction)
	}
	// S251: confidence = 0.75 × 0.90 (mean_reversion factor)
	if r.Confidence != "0.6750" {
		t.Errorf("expected risk confidence 0.6750, got %s", r.Confidence)
	}
}

func TestPositionExposureEvaluator_FlatApproved(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "flat", "0.0000", "none", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Disposition != domainrisk.DispositionApproved {
		t.Fatalf("expected approved, got %s", r.Disposition)
	}
	if r.Confidence != "1.0000" {
		t.Fatalf("expected confidence 1.0000 for flat, got %s", r.Confidence)
	}
	if r.Rationale != "Flat strategy requires no position" {
		t.Fatalf("expected flat rationale, got %s", r.Rationale)
	}
}

func TestPositionExposureEvaluator_UnknownDirection(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	_, ok := eval.Evaluate("mean_reversion_entry", "sideways", "0.5000", "", "", 60, now)
	if ok {
		t.Fatal("expected evaluation to fail for unknown direction")
	}
}

func TestPositionExposureEvaluator_InvalidConfidence(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	_, ok := eval.Evaluate("mean_reversion_entry", "long", "not-a-number", "", "", 60, now)
	if ok {
		t.Fatal("expected evaluation to fail for invalid confidence")
	}
}

func TestPositionExposureEvaluator_TimestampPreserved(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", "", "", 60, ts)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if !r.Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, r.Timestamp)
	}
}

func TestPositionExposureEvaluator_Validation(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", "low", "RSI below threshold", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if prob := r.Validate(); prob != nil {
		t.Fatalf("risk should be valid, got: %s", prob.Message)
	}
}

func TestPositionExposureEvaluator_PartitionKey(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", "", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if got := r.PartitionKey(); got != "binancef.btcusdt.60" {
		t.Fatalf("expected binancef.btcusdt.60, got %s", got)
	}
}

func TestPositionExposureEvaluator_StrategyInputPreserved(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 300)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.9000", "high", "RSI 10.00 below threshold", 300, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if len(r.Strategies) != 1 {
		t.Fatalf("expected 1 strategy input, got %d", len(r.Strategies))
	}
	si := r.Strategies[0]
	if si.Type != "mean_reversion_entry" {
		t.Errorf("expected strategy type mean_reversion_entry, got %s", si.Type)
	}
	if si.Direction != "long" {
		t.Errorf("expected strategy direction long, got %s", si.Direction)
	}
	if si.Confidence != "0.9000" {
		t.Errorf("expected strategy confidence 0.9000, got %s", si.Confidence)
	}
	if si.Timeframe != 300 {
		t.Errorf("expected strategy timeframe 300, got %d", si.Timeframe)
	}
	if si.DecisionSeverity != "high" {
		t.Errorf("expected decision severity high, got %s", si.DecisionSeverity)
	}
	if si.DecisionRationale != "RSI 10.00 below threshold" {
		t.Errorf("expected decision rationale, got %s", si.DecisionRationale)
	}
}

func TestPositionExposureEvaluator_ParametersPresent(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", "", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Parameters["max_position_pct"] == "" {
		t.Error("expected max_position_pct parameter")
	}
	if r.Parameters["max_portfolio_exposure_pct"] == "" {
		t.Error("expected max_portfolio_exposure_pct parameter")
	}
	// S251: effective parameters present.
	if r.Parameters["effective_max_position_pct"] == "" {
		t.Error("expected effective_max_position_pct parameter")
	}
	if r.Parameters["confidence_factor"] == "" {
		t.Error("expected confidence_factor parameter")
	}
	if r.Parameters["severity_limit_factor"] == "" {
		t.Error("expected severity_limit_factor parameter")
	}
}

func TestPositionExposureEvaluator_MultiSymbol_IndependentEvaluation(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt"}
	timeframes := []int{60, 300}
	now := time.Now().UTC()

	results := make(map[string]domainrisk.RiskAssessment)

	for _, sym := range symbols {
		for _, tf := range timeframes {
			eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", instrumentForSymbol(sym), tf)
			r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", "low", "", tf, now)
			if !ok {
				t.Fatalf("expected evaluation to succeed for %s/%d", sym, tf)
			}
			key := r.PartitionKey()
			results[key] = r
		}
	}

	// Verify all 4 combinations produced unique results
	expectedCount := len(symbols) * len(timeframes)
	if len(results) != expectedCount {
		t.Fatalf("expected %d unique results, got %d", expectedCount, len(results))
	}

	// Verify each result has correct symbol and timeframe
	for _, sym := range symbols {
		for _, tf := range timeframes {
			eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", instrumentForSymbol(sym), tf)
			r, _ := eval.Evaluate("mean_reversion_entry", "long", "0.8500", "low", "", tf, now)
			if r.VenueSymbol() != sym {
				t.Errorf("expected symbol %s, got %s", sym, r.VenueSymbol())
			}
			if r.Timeframe != tf {
				t.Errorf("expected timeframe %d, got %d", tf, r.Timeframe)
			}
			if prob := r.Validate(); prob != nil {
				t.Errorf("risk for %s/%d should be valid: %s", sym, tf, prob.Message)
			}
		}
	}
}

func TestPositionExposureEvaluator_MultiSymbol_NoOwnershipBleed(t *testing.T) {
	now := time.Now().UTC()

	evalBTC := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	evalETH := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", ethUSDTPerp, 60)

	rBTC, ok := evalBTC.Evaluate("mean_reversion_entry", "long", "0.8500", "high", "BTC RSI low", 60, now)
	if !ok {
		t.Fatal("BTC evaluation should succeed")
	}
	rETH, ok := evalETH.Evaluate("mean_reversion_entry", "short", "0.7500", "low", "ETH RSI low", 60, now)
	if !ok {
		t.Fatal("ETH evaluation should succeed")
	}

	// Verify symbol isolation
	if rBTC.VenueSymbol() == rETH.VenueSymbol() {
		t.Fatal("symbols should differ between evaluators")
	}
	if rBTC.VenueSymbol() != "btcusdt" {
		t.Errorf("BTC symbol bleed: got %s", rBTC.VenueSymbol())
	}
	if rETH.VenueSymbol() != "ethusdt" {
		t.Errorf("ETH symbol bleed: got %s", rETH.VenueSymbol())
	}

	// Verify partition key isolation
	if rBTC.PartitionKey() == rETH.PartitionKey() {
		t.Fatalf("partition keys should differ: %s", rBTC.PartitionKey())
	}

	// Verify dedup key isolation
	if rBTC.DeduplicationKey() == rETH.DeduplicationKey() {
		t.Fatal("dedup keys should differ between symbols")
	}

	// Verify strategies preserve their respective directions
	if rBTC.Strategies[0].Direction != "long" {
		t.Errorf("BTC strategy direction bleed: got %s", rBTC.Strategies[0].Direction)
	}
	if rETH.Strategies[0].Direction != "short" {
		t.Errorf("ETH strategy direction bleed: got %s", rETH.Strategies[0].Direction)
	}

	// Verify decision severity isolation
	if rBTC.Strategies[0].DecisionSeverity != "high" {
		t.Errorf("BTC decision severity bleed: got %s", rBTC.Strategies[0].DecisionSeverity)
	}
	if rETH.Strategies[0].DecisionSeverity != "low" {
		t.Errorf("ETH decision severity bleed: got %s", rETH.Strategies[0].DecisionSeverity)
	}
}

func TestPositionExposureEvaluator_DecisionSeverityInRationale(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", "high", "RSI 10.00 below threshold", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if !strings.Contains(r.Rationale, "decision severity high") {
		t.Errorf("expected rationale to reference decision severity, got: %s", r.Rationale)
	}
}

func TestPositionExposureEvaluator_NoSeverityInRationale_WhenNone(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", "none", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if strings.Contains(r.Rationale, "decision severity") {
		t.Errorf("expected no decision severity in rationale for none, got: %s", r.Rationale)
	}
}

func TestPositionExposureEvaluator_DecisionContextInMetadata(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", "moderate", "RSI 20.00 below threshold", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Metadata["decision_severity"] != "moderate" {
		t.Errorf("expected decision_severity=moderate in metadata, got %q", r.Metadata["decision_severity"])
	}
	if r.Metadata["decision_rationale"] != "RSI 20.00 below threshold" {
		t.Errorf("expected decision_rationale in metadata, got %q", r.Metadata["decision_rationale"])
	}
	// S251: strategy_type in metadata.
	if r.Metadata["strategy_type"] != "mean_reversion_entry" {
		t.Errorf("expected strategy_type=mean_reversion_entry in metadata, got %q", r.Metadata["strategy_type"])
	}
}

func TestPositionExposureEvaluator_NoMetadata_WhenNoDecisionContext(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", "", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	// S251: strategy_type is always present, so metadata is no longer nil.
	if r.Metadata["strategy_type"] != "mean_reversion_entry" {
		t.Errorf("expected strategy_type in metadata even without decision context, got %v", r.Metadata)
	}
}

func TestPositionExposureEvaluator_FlatWithDecisionContext(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "flat", "0.0000", "none", "RSI above threshold", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Strategies[0].DecisionSeverity != "none" {
		t.Errorf("expected decision severity none for flat, got %s", r.Strategies[0].DecisionSeverity)
	}
	// Flat still carries decision context in metadata for observability
	if r.Metadata["decision_severity"] != "none" {
		t.Errorf("expected decision_severity=none in flat metadata, got %q", r.Metadata["decision_severity"])
	}
}
