package consistency

import (
	"testing"
)

// validChain returns a fully consistent chain snapshot for baseline testing.
func validChain() ChainSnapshot {
	return ChainSnapshot{
		CorrelationID: "corr-001",

		HasDecision:        true,
		DecisionOutcome:    "triggered",
		DecisionSeverity:   "moderate",
		DecisionConfidence: "0.8500",
		DecisionSymbol:     "BTCUSDT",
		DecisionSource:     "binance",
		DecisionTimeframe:  60,

		HasStrategy:        true,
		StrategyDirection:  "long",
		StrategyConfidence: "0.8500",
		StrategySymbol:     "BTCUSDT",
		StrategySource:     "binance",
		StrategyTimeframe:  60,

		HasRisk:         true,
		RiskDisposition: "approved",
		RiskConfidence:  "0.8075",
		RiskSymbol:      "BTCUSDT",
		RiskSource:      "binance",
		RiskTimeframe:   60,
		RiskStrategyDir: "long",

		HasExecution:       true,
		ExecutionSide:      "buy",
		ExecutionQuantity:  "0.0200",
		ExecutionSymbol:    "BTCUSDT",
		ExecutionSource:    "binance",
		ExecutionTimeframe: 60,
		ExecutionRiskDisp:  "approved",
	}
}

func TestCheck_CleanChain(t *testing.T) {
	snap := validChain()
	report := Check(snap)

	if !report.Clean {
		t.Errorf("expected clean report, got %d violations, %d warnings", report.Violations, report.Warnings)
		for _, f := range report.Findings {
			t.Logf("  %s [%s] %s: got=%s expected=%s", f.Check, f.Severity, f.Message, f.Got, f.Expected)
		}
	}

	if report.ChecksRun == 0 {
		t.Error("expected at least one check to run")
	}

	if report.CorrelationID != "corr-001" {
		t.Errorf("expected correlation_id corr-001, got %s", report.CorrelationID)
	}
}

func TestCheck_SeverityOutcome_TriggeredWithNone(t *testing.T) {
	snap := validChain()
	snap.DecisionSeverity = "none"
	snap.DecisionOutcome = "triggered"

	report := Check(snap)
	assertHasFinding(t, report, "severity_outcome", SeverityViolation)
}

func TestCheck_SeverityOutcome_NotTriggeredWithSeverity(t *testing.T) {
	snap := validChain()
	snap.DecisionOutcome = "not_triggered"
	snap.DecisionSeverity = "high"
	// Adjust downstream to avoid other violations.
	snap.StrategyDirection = "flat"
	snap.ExecutionSide = "none"
	snap.ExecutionQuantity = "0"
	snap.RiskStrategyDir = "flat"

	report := Check(snap)
	assertHasFinding(t, report, "severity_outcome", SeverityViolation)
}

func TestCheck_DirectionSide_LongMustBeBuy(t *testing.T) {
	snap := validChain()
	snap.StrategyDirection = "long"
	snap.ExecutionSide = "sell" // wrong

	report := Check(snap)
	assertHasFinding(t, report, "direction_side", SeverityViolation)
}

func TestCheck_DirectionSide_ShortMustBeSell(t *testing.T) {
	snap := validChain()
	snap.StrategyDirection = "short"
	snap.ExecutionSide = "buy" // wrong
	snap.RiskStrategyDir = "short"

	report := Check(snap)
	assertHasFinding(t, report, "direction_side", SeverityViolation)
}

func TestCheck_DirectionSide_RejectedMustBeNone(t *testing.T) {
	snap := validChain()
	snap.RiskDisposition = "rejected"
	snap.ExecutionSide = "buy" // wrong — should be none
	snap.ExecutionRiskDisp = "rejected"

	report := Check(snap)
	assertHasFinding(t, report, "direction_side", SeverityViolation)
}

func TestCheck_DispositionAction_RejectedWithAction(t *testing.T) {
	snap := validChain()
	snap.RiskDisposition = "rejected"
	snap.ExecutionSide = "buy"
	snap.ExecutionQuantity = "0.0200"
	snap.ExecutionRiskDisp = "rejected"

	report := Check(snap)
	assertHasFinding(t, report, "disposition_action", SeverityViolation)
}

func TestCheck_DispositionAction_ApprovedWithNoAction(t *testing.T) {
	snap := validChain()
	snap.RiskDisposition = "approved"
	snap.StrategyDirection = "long"
	snap.ExecutionSide = "none" // warning: approved long with no action

	report := Check(snap)
	assertHasFinding(t, report, "disposition_action", SeverityWarning)
}

func TestCheck_SymbolCoherence_Mismatch(t *testing.T) {
	snap := validChain()
	snap.ExecutionSymbol = "ETHUSDT" // mismatch

	report := Check(snap)
	assertHasFinding(t, report, "symbol_coherence", SeverityViolation)
}

func TestCheck_SourceCoherence_Mismatch(t *testing.T) {
	snap := validChain()
	snap.RiskSource = "kraken" // mismatch

	report := Check(snap)
	assertHasFinding(t, report, "source_coherence", SeverityViolation)
}

func TestCheck_TimeframeCoherence_Mismatch(t *testing.T) {
	snap := validChain()
	snap.StrategyTimeframe = 300 // mismatch

	report := Check(snap)
	assertHasFinding(t, report, "timeframe_coherence", SeverityViolation)
}

func TestCheck_ConfidenceProgression_RiskExceedsStrategy(t *testing.T) {
	snap := validChain()
	snap.StrategyConfidence = "0.5000"
	snap.RiskConfidence = "0.9500" // higher than strategy

	report := Check(snap)
	assertHasFinding(t, report, "confidence_progression", SeverityWarning)
}

func TestCheck_ConfidenceProgression_NormalDiscount(t *testing.T) {
	snap := validChain()
	snap.StrategyConfidence = "0.8500"
	snap.RiskConfidence = "0.8075" // discounted

	report := Check(snap)
	assertNoFinding(t, report, "confidence_progression")
}

func TestCheck_DispositionPropagation_Mismatch(t *testing.T) {
	snap := validChain()
	snap.RiskDisposition = "rejected"
	snap.ExecutionRiskDisp = "approved" // mismatch
	snap.ExecutionSide = "none"
	snap.ExecutionQuantity = "0"

	report := Check(snap)
	assertHasFinding(t, report, "disposition_propagation", SeverityViolation)
}

func TestCheck_DirectionPropagation_Mismatch(t *testing.T) {
	snap := validChain()
	snap.StrategyDirection = "long"
	snap.RiskStrategyDir = "short" // mismatch

	report := Check(snap)
	assertHasFinding(t, report, "direction_propagation", SeverityViolation)
}

func TestCheck_PartialChain_DecisionOnly(t *testing.T) {
	snap := ChainSnapshot{
		CorrelationID:      "corr-partial",
		HasDecision:        true,
		DecisionOutcome:    "not_triggered",
		DecisionSeverity:   "none",
		DecisionConfidence: "0.3000",
		DecisionSymbol:     "BTCUSDT",
		DecisionSource:     "binance",
		DecisionTimeframe:  60,
	}

	report := Check(snap)
	// Partial chains with only decision should be clean if decision is internally consistent.
	if !report.Clean {
		t.Errorf("expected clean report for partial chain, got %d violations, %d warnings", report.Violations, report.Warnings)
		for _, f := range report.Findings {
			t.Logf("  %s [%s] %s", f.Check, f.Severity, f.Message)
		}
	}
}

func TestCheck_FlatChain_Clean(t *testing.T) {
	snap := validChain()
	snap.StrategyDirection = "flat"
	snap.ExecutionSide = "none"
	snap.ExecutionQuantity = "0"
	snap.RiskStrategyDir = "flat"

	report := Check(snap)
	if !report.Clean {
		t.Errorf("expected clean report for flat chain, got findings:")
		for _, f := range report.Findings {
			t.Logf("  %s [%s] %s: got=%s", f.Check, f.Severity, f.Message, f.Got)
		}
	}
}

func TestCheck_ShortChain_Clean(t *testing.T) {
	snap := validChain()
	snap.StrategyDirection = "short"
	snap.ExecutionSide = "sell"
	snap.RiskStrategyDir = "short"

	report := Check(snap)
	if !report.Clean {
		t.Errorf("expected clean report for short chain, got findings:")
		for _, f := range report.Findings {
			t.Logf("  %s [%s] %s: got=%s", f.Check, f.Severity, f.Message, f.Got)
		}
	}
}

// --- helpers ---

func assertHasFinding(t *testing.T, report Report, check string, severity Severity) {
	t.Helper()
	for _, f := range report.Findings {
		if f.Check == check && f.Severity == severity {
			return
		}
	}
	t.Errorf("expected finding check=%s severity=%s, got %d findings:", check, severity, len(report.Findings))
	for _, f := range report.Findings {
		t.Logf("  %s [%s] %s", f.Check, f.Severity, f.Message)
	}
}

func assertNoFinding(t *testing.T, report Report, check string) {
	t.Helper()
	for _, f := range report.Findings {
		if f.Check == check {
			t.Errorf("unexpected finding check=%s severity=%s: %s", f.Check, f.Severity, f.Message)
		}
	}
}
