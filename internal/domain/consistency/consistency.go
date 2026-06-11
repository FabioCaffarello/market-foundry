// Package consistency provides cross-domain consistency checks for the
// decision pipeline: decision -> strategy -> risk -> execution.
//
// These checks detect silent divergence between domains by validating
// invariants that span multiple domain boundaries. Each check produces
// structured findings (violations or warnings) that are auditable.
//
// S472: This package does NOT own domain data. It operates on snapshots
// of domain values passed as primitive types to avoid creating import
// dependencies on domain packages.
package consistency

import "fmt"

// Severity classifies the importance of a consistency finding.
type Severity string

const (
	SeverityViolation Severity = "violation" // hard invariant broken
	SeverityWarning   Severity = "warning"   // soft invariant or suspicious state
)

// Finding represents a single consistency check result.
type Finding struct {
	Check    string   `json:"check"`    // check identifier (e.g., "direction_side")
	Severity Severity `json:"severity"` // violation or warning
	Domain   string   `json:"domain"`   // which domain boundary is affected
	Message  string   `json:"message"`  // human-readable description
	Got      string   `json:"got"`      // actual value observed
	Expected string   `json:"expected"` // what was expected
}

// Report is the output of running all cross-domain consistency checks
// on a single decision chain snapshot.
type Report struct {
	CorrelationID string    `json:"correlation_id,omitempty"`
	Findings      []Finding `json:"findings"`
	ChecksRun     int       `json:"checks_run"`
	Violations    int       `json:"violations"`
	Warnings      int       `json:"warnings"`
	Clean         bool      `json:"clean"`
}

// ChainSnapshot holds the cross-domain values needed for consistency checks.
// All fields use primitive types to avoid coupling to domain packages.
// Fields are optional — partial chains are valid (missing stages produce warnings).
type ChainSnapshot struct {
	CorrelationID string

	// Decision stage.
	HasDecision        bool
	DecisionOutcome    string // triggered, not_triggered, insufficient
	DecisionSeverity   string // none, low, moderate, high
	DecisionConfidence string
	DecisionSymbol     string
	DecisionSource     string
	DecisionTimeframe  int

	// Strategy stage.
	HasStrategy         bool
	StrategyDirection   string // long, short, flat
	StrategyConfidence  string
	StrategySymbol      string
	StrategySource      string
	StrategyTimeframe   int
	StrategyDecisionRef string // type of decision referenced by strategy

	// Risk stage.
	HasRisk         bool
	RiskDisposition string // approved, modified, rejected
	RiskConfidence  string
	RiskSymbol      string
	RiskSource      string
	RiskTimeframe   int
	RiskStrategyRef string // type of strategy referenced by risk
	RiskStrategyDir string // direction of strategy as seen by risk

	// Execution stage.
	HasExecution       bool
	ExecutionSide      string // buy, sell, none
	ExecutionQuantity  string
	ExecutionSymbol    string
	ExecutionSource    string
	ExecutionTimeframe int
	ExecutionRiskDisp  string // disposition as seen by execution
}

// Check runs all cross-domain consistency checks on the given chain snapshot
// and returns a structured report.
func Check(snap ChainSnapshot) Report {
	var findings []Finding
	checksRun := 0

	checks := []func(ChainSnapshot) []Finding{
		checkSeverityOutcome,
		checkDirectionSide,
		checkDispositionAction,
		checkSymbolCoherence,
		checkSourceCoherence,
		checkTimeframeCoherence,
		checkConfidenceProgression,
		checkDispositionPropagation,
		checkDirectionPropagation,
	}

	for _, check := range checks {
		checksRun++
		findings = append(findings, check(snap)...)
	}

	violations := 0
	warnings := 0
	for _, f := range findings {
		switch f.Severity {
		case SeverityViolation:
			violations++
		case SeverityWarning:
			warnings++
		}
	}

	return Report{
		CorrelationID: snap.CorrelationID,
		Findings:      findings,
		ChecksRun:     checksRun,
		Violations:    violations,
		Warnings:      warnings,
		Clean:         violations == 0 && warnings == 0,
	}
}

// checkSeverityOutcome validates that decision severity and outcome are consistent.
// Invariant: severity=none requires outcome=not_triggered or insufficient.
// Invariant: severity!=none requires outcome=triggered.
func checkSeverityOutcome(snap ChainSnapshot) []Finding {
	if !snap.HasDecision {
		return nil
	}

	var findings []Finding

	if snap.DecisionSeverity == "none" && snap.DecisionOutcome == "triggered" {
		findings = append(findings, Finding{
			Check:    "severity_outcome",
			Severity: SeverityViolation,
			Domain:   "decision",
			Message:  "triggered decision must have severity != none",
			Got:      fmt.Sprintf("outcome=%s severity=%s", snap.DecisionOutcome, snap.DecisionSeverity),
			Expected: "severity in {low, moderate, high} when outcome=triggered",
		})
	}

	if snap.DecisionSeverity != "none" && snap.DecisionSeverity != "" &&
		snap.DecisionOutcome != "triggered" {
		findings = append(findings, Finding{
			Check:    "severity_outcome",
			Severity: SeverityViolation,
			Domain:   "decision",
			Message:  "non-triggered decision must have severity=none",
			Got:      fmt.Sprintf("outcome=%s severity=%s", snap.DecisionOutcome, snap.DecisionSeverity),
			Expected: "severity=none when outcome != triggered",
		})
	}

	return findings
}

// checkDirectionSide validates strategy direction maps to correct execution side.
// Invariant: long -> buy, short -> sell, flat -> none.
func checkDirectionSide(snap ChainSnapshot) []Finding {
	if !snap.HasStrategy || !snap.HasExecution {
		return nil
	}

	expected := directionToSide(snap.StrategyDirection)
	if expected == "" {
		return nil // unknown direction, caught by domain validation
	}

	// When risk rejected, execution side must be none regardless of direction.
	if snap.HasRisk && snap.RiskDisposition == "rejected" {
		if snap.ExecutionSide != "none" {
			return []Finding{{
				Check:    "direction_side",
				Severity: SeverityViolation,
				Domain:   "strategy->execution",
				Message:  "risk-rejected chain must produce side=none",
				Got:      fmt.Sprintf("risk_disposition=rejected execution_side=%s", snap.ExecutionSide),
				Expected: "side=none",
			}}
		}
		return nil
	}

	if snap.ExecutionSide != expected {
		return []Finding{{
			Check:    "direction_side",
			Severity: SeverityViolation,
			Domain:   "strategy->execution",
			Message:  fmt.Sprintf("strategy direction %q should produce side %q", snap.StrategyDirection, expected),
			Got:      fmt.Sprintf("direction=%s side=%s", snap.StrategyDirection, snap.ExecutionSide),
			Expected: fmt.Sprintf("side=%s", expected),
		}}
	}

	return nil
}

// checkDispositionAction validates risk disposition is consistent with execution action.
// Invariant: rejected -> side=none, quantity=0.
// Invariant: approved/modified -> side != none (unless direction=flat).
func checkDispositionAction(snap ChainSnapshot) []Finding {
	if !snap.HasRisk || !snap.HasExecution {
		return nil
	}

	var findings []Finding

	if snap.RiskDisposition == "rejected" {
		if snap.ExecutionSide != "none" {
			findings = append(findings, Finding{
				Check:    "disposition_action",
				Severity: SeverityViolation,
				Domain:   "risk->execution",
				Message:  "rejected risk must produce no-action execution",
				Got:      fmt.Sprintf("disposition=rejected side=%s", snap.ExecutionSide),
				Expected: "side=none",
			})
		}
		if snap.ExecutionQuantity != "0" && snap.ExecutionQuantity != "" {
			findings = append(findings, Finding{
				Check:    "disposition_action",
				Severity: SeverityViolation,
				Domain:   "risk->execution",
				Message:  "rejected risk must produce zero quantity",
				Got:      fmt.Sprintf("disposition=rejected quantity=%s", snap.ExecutionQuantity),
				Expected: "quantity=0",
			})
		}
	}

	if (snap.RiskDisposition == "approved" || snap.RiskDisposition == "modified") &&
		snap.HasStrategy && snap.StrategyDirection != "flat" {
		if snap.ExecutionSide == "none" {
			findings = append(findings, Finding{
				Check:    "disposition_action",
				Severity: SeverityWarning,
				Domain:   "risk->execution",
				Message:  "approved/modified non-flat chain has no-action execution",
				Got:      fmt.Sprintf("disposition=%s direction=%s side=none", snap.RiskDisposition, snap.StrategyDirection),
				Expected: "side=buy or side=sell",
			})
		}
	}

	return findings
}

// checkSymbolCoherence validates symbol is identical across all present stages.
func checkSymbolCoherence(snap ChainSnapshot) []Finding {
	symbols := map[string]string{}
	if snap.HasDecision {
		symbols["decision"] = snap.DecisionSymbol
	}
	if snap.HasStrategy {
		symbols["strategy"] = snap.StrategySymbol
	}
	if snap.HasRisk {
		symbols["risk"] = snap.RiskSymbol
	}
	if snap.HasExecution {
		symbols["execution"] = snap.ExecutionSymbol
	}

	return checkFieldCoherence("symbol", symbols)
}

// checkSourceCoherence validates source is identical across all present stages.
func checkSourceCoherence(snap ChainSnapshot) []Finding {
	sources := map[string]string{}
	if snap.HasDecision {
		sources["decision"] = snap.DecisionSource
	}
	if snap.HasStrategy {
		sources["strategy"] = snap.StrategySource
	}
	if snap.HasRisk {
		sources["risk"] = snap.RiskSource
	}
	if snap.HasExecution {
		sources["execution"] = snap.ExecutionSource
	}

	return checkFieldCoherence("source", sources)
}

// checkTimeframeCoherence validates timeframe is identical across all present stages.
func checkTimeframeCoherence(snap ChainSnapshot) []Finding {
	timeframes := map[string]int{}
	if snap.HasDecision {
		timeframes["decision"] = snap.DecisionTimeframe
	}
	if snap.HasStrategy {
		timeframes["strategy"] = snap.StrategyTimeframe
	}
	if snap.HasRisk {
		timeframes["risk"] = snap.RiskTimeframe
	}
	if snap.HasExecution {
		timeframes["execution"] = snap.ExecutionTimeframe
	}

	if len(timeframes) <= 1 {
		return nil
	}

	var ref int
	var refStage string
	first := true
	var findings []Finding

	for stage, tf := range timeframes {
		if first {
			ref = tf
			refStage = stage
			first = false
			continue
		}
		if tf != ref {
			findings = append(findings, Finding{
				Check:    "timeframe_coherence",
				Severity: SeverityViolation,
				Domain:   fmt.Sprintf("%s->%s", refStage, stage),
				Message:  "timeframe mismatch across chain stages",
				Got:      fmt.Sprintf("%s=%d %s=%d", refStage, ref, stage, tf),
				Expected: "identical timeframe across all stages",
			})
		}
	}

	return findings
}

// checkConfidenceProgression validates that risk confidence does not exceed
// strategy confidence. Risk applies a discount factor, so the output confidence
// should be <= the input strategy confidence.
func checkConfidenceProgression(snap ChainSnapshot) []Finding {
	if !snap.HasStrategy || !snap.HasRisk {
		return nil
	}

	// Only check when both are parseable as non-empty strings.
	if snap.StrategyConfidence == "" || snap.RiskConfidence == "" {
		return nil
	}

	// Compare as strings since they're decimal formatted.
	// A lexicographic comparison works for fixed-width decimal strings (e.g., "0.9500").
	if snap.RiskConfidence > snap.StrategyConfidence {
		return []Finding{{
			Check:    "confidence_progression",
			Severity: SeverityWarning,
			Domain:   "strategy->risk",
			Message:  "risk confidence exceeds strategy confidence (expected discount factor)",
			Got:      fmt.Sprintf("strategy_confidence=%s risk_confidence=%s", snap.StrategyConfidence, snap.RiskConfidence),
			Expected: "risk_confidence <= strategy_confidence",
		}}
	}

	return nil
}

// checkDispositionPropagation validates risk disposition is correctly carried into execution.
func checkDispositionPropagation(snap ChainSnapshot) []Finding {
	if !snap.HasRisk || !snap.HasExecution {
		return nil
	}

	if snap.ExecutionRiskDisp == "" {
		return nil // not all paths populate this
	}

	if snap.ExecutionRiskDisp != snap.RiskDisposition {
		return []Finding{{
			Check:    "disposition_propagation",
			Severity: SeverityViolation,
			Domain:   "risk->execution",
			Message:  "execution risk.disposition does not match originating risk assessment disposition",
			Got:      fmt.Sprintf("risk_disposition=%s execution_risk_disposition=%s", snap.RiskDisposition, snap.ExecutionRiskDisp),
			Expected: "identical disposition",
		}}
	}

	return nil
}

// checkDirectionPropagation validates strategy direction is correctly carried into risk.
func checkDirectionPropagation(snap ChainSnapshot) []Finding {
	if !snap.HasStrategy || !snap.HasRisk {
		return nil
	}

	if snap.RiskStrategyDir == "" {
		return nil // not all paths populate this
	}

	if snap.RiskStrategyDir != snap.StrategyDirection {
		return []Finding{{
			Check:    "direction_propagation",
			Severity: SeverityViolation,
			Domain:   "strategy->risk",
			Message:  "risk strategy input direction does not match originating strategy direction",
			Got:      fmt.Sprintf("strategy_direction=%s risk_strategy_direction=%s", snap.StrategyDirection, snap.RiskStrategyDir),
			Expected: "identical direction",
		}}
	}

	return nil
}

// --- helpers ---

func directionToSide(direction string) string {
	switch direction {
	case "long":
		return "buy"
	case "short":
		return "sell"
	case "flat":
		return "none"
	default:
		return ""
	}
}

func checkFieldCoherence(field string, values map[string]string) []Finding {
	if len(values) <= 1 {
		return nil
	}

	var ref string
	var refStage string
	first := true
	var findings []Finding

	for stage, v := range values {
		if first {
			ref = v
			refStage = stage
			first = false
			continue
		}
		if v != ref {
			findings = append(findings, Finding{
				Check:    field + "_coherence",
				Severity: SeverityViolation,
				Domain:   fmt.Sprintf("%s->%s", refStage, stage),
				Message:  fmt.Sprintf("%s mismatch across chain stages", field),
				Got:      fmt.Sprintf("%s=%q %s=%q", refStage, ref, stage, v),
				Expected: fmt.Sprintf("identical %s across all stages", field),
			})
		}
	}

	return findings
}
