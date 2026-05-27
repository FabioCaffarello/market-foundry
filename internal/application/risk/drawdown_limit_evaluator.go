package risk

import (
	"fmt"
	"strconv"
	"time"

	"internal/domain/instrument"
	domainrisk "internal/domain/risk"
)

const (
	defaultMaxDrawdownPct  = 0.05
	defaultStopDistancePct = 0.03
)

// DrawdownLimitEvaluator assesses strategy intent against configurable drawdown and stop-loss limits.
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives strategy values as primitive data (not strategy.Strategy structs) per domain isolation.
//
// S251 behavioral activation:
//   - Risk confidence multiplier varies by strategy type (counter-trend vs pro-trend).
//   - Base stop distance ceiling adjusts by strategy type (tighter for counter-trend).
//   - Max drawdown tolerance scales by decision severity (strong signal → more room).
//   - Strategy type is recorded in metadata for observability.
type DrawdownLimitEvaluator struct {
	source          string
	instrument      instrument.CanonicalInstrument
	timeframe       int
	maxDrawdownPct  float64
	stopDistancePct float64
}

// NewDrawdownLimitEvaluatorForInstrument constructs the evaluator from
// a canonical Instrument directly. See NewRSISamplerForInstrument
// (signal package) for the boundary-helper rationale.
func NewDrawdownLimitEvaluatorForInstrument(source string, inst instrument.CanonicalInstrument, timeframe int) *DrawdownLimitEvaluator {
	return &DrawdownLimitEvaluator{
		source:          source,
		instrument:      inst,
		timeframe:       timeframe,
		maxDrawdownPct:  defaultMaxDrawdownPct,
		stopDistancePct: defaultStopDistancePct,
	}
}

// Evaluate processes a strategy resolution and produces a drawdown risk assessment.
// strategyType identifies the strategy family (e.g., "mean_reversion_entry", "trend_following_entry").
// strategyDirection is "long", "short", or "flat".
// strategyConfidence is a decimal string from the strategy event.
// decisionSeverity and decisionRationale carry the originating decision's semantic depth
// forward into StrategyInput and risk rationale for end-to-end traceability.
// Returns a RiskAssessment and true if evaluation succeeded.
func (e *DrawdownLimitEvaluator) Evaluate(
	strategyType, strategyDirection, strategyConfidence string,
	decisionSeverity, decisionRationale string,
	strategyTimeframe int,
	ts time.Time,
) (domainrisk.RiskAssessment, bool) {
	strategyInput := domainrisk.StrategyInput{
		Type:              strategyType,
		Direction:         strategyDirection,
		Confidence:        strategyConfidence,
		Timeframe:         strategyTimeframe,
		DecisionSeverity:  decisionSeverity,
		DecisionRationale: decisionRationale,
	}

	// S251: Look up strategy-type-specific factors.
	confidenceFactor := lookupFactor(strategyType, drawdownConfidenceFactor, drawdownConfidenceDefault)
	stopFactor := lookupFactor(strategyType, drawdownStopFactor, 1.0)

	// S251: Look up severity-based drawdown tolerance factor.
	sevFactor := lookupSeverityFactor(decisionSeverity, drawdownSeverityFactor)

	// Effective stop distance base adjusted by strategy type.
	effectiveStopBase := e.stopDistancePct * stopFactor

	// Effective max drawdown adjusted by severity.
	effectiveMaxDrawdown := e.maxDrawdownPct * sevFactor

	baseParams := map[string]string{
		"max_drawdown_pct":            fmt.Sprintf("%.4f", e.maxDrawdownPct),
		"stop_distance_pct":           fmt.Sprintf("%.4f", e.stopDistancePct),
		"effective_stop_distance_pct": fmt.Sprintf("%.4f", effectiveStopBase),
		"effective_max_drawdown_pct":  fmt.Sprintf("%.4f", effectiveMaxDrawdown),
		"confidence_factor":           fmt.Sprintf("%.2f", confidenceFactor),
		"stop_type_factor":            fmt.Sprintf("%.2f", stopFactor),
		"severity_tolerance_factor":   fmt.Sprintf("%.2f", sevFactor),
	}

	// Flat strategies are always approved — no drawdown risk.
	if strategyDirection == "flat" {
		return domainrisk.RiskAssessment{
			Type:        "drawdown_limit",
			Source:      e.source,
			Instrument:  e.instrument,
			Timeframe:   e.timeframe,
			Disposition: domainrisk.DispositionApproved,
			Confidence:  "1.0000",
			Strategies:  []domainrisk.StrategyInput{strategyInput},
			Rationale:   "Flat strategy has no drawdown risk",
			Parameters:  baseParams,
			Metadata:    e.buildMetadata(strategyType, decisionSeverity, decisionRationale),
			Final:       true,
			Timestamp:   ts,
		}, true
	}

	// Validate direction.
	if strategyDirection != "long" && strategyDirection != "short" {
		return domainrisk.RiskAssessment{}, false
	}

	// Parse confidence.
	confidence, err := strconv.ParseFloat(strategyConfidence, 64)
	if err != nil {
		return domainrisk.RiskAssessment{}, false
	}

	// S256: Reject zero or negative confidence — degenerate input should not produce an assessment.
	if confidence <= 0 {
		return domainrisk.RiskAssessment{
			Type:        "drawdown_limit",
			Source:      e.source,
			Instrument:  e.instrument,
			Timeframe:   e.timeframe,
			Disposition: domainrisk.DispositionRejected,
			Confidence:  "0.0000",
			Strategies:  []domainrisk.StrategyInput{strategyInput},
			Rationale:   fmt.Sprintf("Rejected: non-positive confidence %s for %s", strategyConfidence, strategyDirection),
			Parameters:  baseParams,
			Metadata:    e.buildMetadata(strategyType, decisionSeverity, decisionRationale),
			Final:       true,
			Timestamp:   ts,
		}, true
	}

	// S251: Stop distance uses strategy-type-adjusted base, scaled by confidence.
	// Lower confidence → smaller stop (tighter risk control).
	stopDistance := effectiveStopBase * confidence
	if stopDistance < 0.0050 {
		stopDistance = 0.0050 // floor at 0.5% to avoid unrealistic stops
	}
	if stopDistance > effectiveStopBase {
		stopDistance = effectiveStopBase
	}

	// S251: Disposition checks against severity-adjusted max drawdown.
	var disposition domainrisk.Disposition
	var rationale string
	var constraints domainrisk.Constraints

	if stopDistance <= effectiveMaxDrawdown {
		disposition = domainrisk.DispositionApproved
		rationale = e.buildRationale("approved", strategyType, strategyDirection, stopDistance, confidenceFactor, stopFactor, decisionSeverity, sevFactor)
		constraints = domainrisk.Constraints{
			StopDistance: fmt.Sprintf("%.4f", stopDistance),
			MaxExposure:  fmt.Sprintf("%.4f", effectiveMaxDrawdown),
		}
	} else {
		// Cap stop distance to effective max drawdown — modified disposition.
		cappedStop := effectiveMaxDrawdown
		disposition = domainrisk.DispositionModified
		rationale = e.buildRationale("modified", strategyType, strategyDirection, cappedStop, confidenceFactor, stopFactor, decisionSeverity, sevFactor)
		constraints = domainrisk.Constraints{
			StopDistance: fmt.Sprintf("%.4f", cappedStop),
			MaxExposure:  fmt.Sprintf("%.4f", effectiveMaxDrawdown),
		}
	}

	// S251: Risk confidence scales by strategy-type-specific factor (was fixed ×0.90).
	riskConfidence := fmt.Sprintf("%.4f", confidence*confidenceFactor)

	return domainrisk.RiskAssessment{
		Type:        "drawdown_limit",
		Source:      e.source,
		Instrument:  e.instrument,
		Timeframe:   e.timeframe,
		Disposition: disposition,
		Confidence:  riskConfidence,
		Strategies:  []domainrisk.StrategyInput{strategyInput},
		Constraints: constraints,
		Rationale:   rationale,
		Parameters:  baseParams,
		Metadata:    e.buildMetadata(strategyType, decisionSeverity, decisionRationale),
		Final:       true,
		Timestamp:   ts,
	}, true
}

// buildRationale generates a context-rich rationale incorporating strategy type and decision severity.
func (e *DrawdownLimitEvaluator) buildRationale(
	outcome, strategyType, direction string,
	stopDistance, confidenceFactor, stopFactor float64,
	decisionSeverity string, sevFactor float64,
) string {
	var base string
	switch outcome {
	case "approved":
		base = fmt.Sprintf("Stop distance %.4f within drawdown limits for %s", stopDistance, direction)
	case "modified":
		base = fmt.Sprintf("Stop distance capped to %.4f by drawdown limits for %s", stopDistance, direction)
	default:
		base = fmt.Sprintf("%s %s stop distance %.4f", direction, outcome, stopDistance)
	}

	// Strategy type context.
	base += fmt.Sprintf("; %s (confidence ×%.2f, stop ×%.2f)", strategyType, confidenceFactor, stopFactor)

	// Severity context.
	if decisionSeverity != "" && decisionSeverity != "none" {
		base += fmt.Sprintf("; decision severity %s (tolerance ×%.2f)", decisionSeverity, sevFactor)
	}
	return base
}

// buildMetadata populates risk metadata with strategy type and decision context for observability.
func (e *DrawdownLimitEvaluator) buildMetadata(strategyType, decisionSeverity, decisionRationale string) map[string]string {
	meta := map[string]string{}
	if strategyType != "" {
		meta["strategy_type"] = strategyType
	}
	if decisionSeverity != "" {
		meta["decision_severity"] = decisionSeverity
	}
	if decisionRationale != "" {
		meta["decision_rationale"] = decisionRationale
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}
