package risk

import (
	"fmt"
	"strconv"
	"time"

	"internal/domain/instrument"
	domainrisk "internal/domain/risk"
)

const (
	defaultMaxPositionPct          = 0.02
	defaultMaxPortfolioExposurePct = 0.10
)

// PositionExposureEvaluator assesses strategy intent against configurable exposure limits.
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives strategy values as primitive data (not strategy.Strategy structs) per domain isolation.
//
// S251 behavioral activation:
//   - Risk confidence multiplier varies by strategy type (counter-trend vs pro-trend).
//   - Effective position limit scales by decision severity (strong signal → more room).
//   - Strategy type is recorded in metadata for observability.
type PositionExposureEvaluator struct {
	source                  string
	instrument              instrument.CanonicalInstrument
	timeframe               int
	maxPositionPct          float64
	maxPortfolioExposurePct float64
}

// NewPositionExposureEvaluatorForInstrument constructs the evaluator from
// a canonical Instrument directly. See NewRSISamplerForInstrument
// (signal package) for the boundary-helper rationale.
func NewPositionExposureEvaluatorForInstrument(source string, inst instrument.CanonicalInstrument, timeframe int) *PositionExposureEvaluator {
	return &PositionExposureEvaluator{
		source:                  source,
		instrument:              inst,
		timeframe:               timeframe,
		maxPositionPct:          defaultMaxPositionPct,
		maxPortfolioExposurePct: defaultMaxPortfolioExposurePct,
	}
}

// Evaluate processes a strategy resolution and produces a risk assessment.
// strategyType identifies the strategy family (e.g., "mean_reversion_entry", "trend_following_entry").
// strategyDirection is "long", "short", or "flat".
// strategyConfidence is a decimal string from the strategy event.
// decisionSeverity and decisionRationale carry the originating decision's semantic depth
// forward into StrategyInput and risk rationale for end-to-end traceability.
// Returns a RiskAssessment and true if evaluation succeeded.
func (e *PositionExposureEvaluator) Evaluate(
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

	// S251: Look up strategy-type-specific confidence factor.
	confidenceFactor := lookupFactor(strategyType, positionExposureConfidenceFactor, positionExposureConfidenceDefault)

	// S251: Look up severity-based position limit factor.
	severityFactor := lookupSeverityFactor(decisionSeverity, positionExposureSeverityFactor)

	// Effective position limit adjusted by decision severity.
	effectiveMaxPosition := e.maxPositionPct * severityFactor

	baseParams := map[string]string{
		"max_position_pct":           fmt.Sprintf("%.4f", e.maxPositionPct),
		"max_portfolio_exposure_pct": fmt.Sprintf("%.4f", e.maxPortfolioExposurePct),
		"effective_max_position_pct": fmt.Sprintf("%.4f", effectiveMaxPosition),
		"confidence_factor":          fmt.Sprintf("%.2f", confidenceFactor),
		"severity_limit_factor":      fmt.Sprintf("%.2f", severityFactor),
	}

	// Flat strategies are always approved with zero constraints.
	if strategyDirection == "flat" {
		return domainrisk.RiskAssessment{
			Type:        "position_exposure",
			Source:      e.source,
			Instrument:  e.instrument,
			Timeframe:   e.timeframe,
			Disposition: domainrisk.DispositionApproved,
			Confidence:  "1.0000",
			Strategies:  []domainrisk.StrategyInput{strategyInput},
			Rationale:   "Flat strategy requires no position",
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
			Type:        "position_exposure",
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

	// Position sizing: scale position by confidence against severity-adjusted limit.
	requestedSize := confidence * effectiveMaxPosition
	if requestedSize > effectiveMaxPosition {
		requestedSize = effectiveMaxPosition
	}

	// Determine disposition based on exposure limits.
	var disposition domainrisk.Disposition
	var rationale string
	var constraints domainrisk.Constraints

	if requestedSize <= effectiveMaxPosition && requestedSize <= e.maxPortfolioExposurePct {
		disposition = domainrisk.DispositionApproved
		rationale = e.buildRationale("approved", strategyType, strategyDirection, requestedSize, confidenceFactor, decisionSeverity, severityFactor)
		constraints = domainrisk.Constraints{
			MaxPositionSize: fmt.Sprintf("%.4f", requestedSize),
			MaxExposure:     fmt.Sprintf("%.4f", e.maxPortfolioExposurePct),
		}
	} else {
		// Cap to max allowed — modified disposition.
		cappedSize := effectiveMaxPosition
		if cappedSize > e.maxPortfolioExposurePct {
			cappedSize = e.maxPortfolioExposurePct
		}
		disposition = domainrisk.DispositionModified
		rationale = e.buildRationale("modified", strategyType, strategyDirection, cappedSize, confidenceFactor, decisionSeverity, severityFactor)
		constraints = domainrisk.Constraints{
			MaxPositionSize: fmt.Sprintf("%.4f", cappedSize),
			MaxExposure:     fmt.Sprintf("%.4f", e.maxPortfolioExposurePct),
		}
	}

	// S251: Risk confidence scales by strategy-type-specific factor (was fixed ×0.95).
	riskConfidence := fmt.Sprintf("%.4f", confidence*confidenceFactor)

	return domainrisk.RiskAssessment{
		Type:        "position_exposure",
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
func (e *PositionExposureEvaluator) buildRationale(
	outcome, strategyType, direction string,
	positionSize, confidenceFactor float64,
	decisionSeverity string, severityFactor float64,
) string {
	var base string
	switch outcome {
	case "approved":
		base = fmt.Sprintf("Position size %.4f within exposure limits", positionSize)
	case "modified":
		base = fmt.Sprintf("Position size capped to %.4f by exposure limits", positionSize)
	default:
		base = fmt.Sprintf("%s %s position %.4f", direction, outcome, positionSize)
	}

	// Strategy type context.
	base += fmt.Sprintf("; %s (confidence ×%.2f)", strategyType, confidenceFactor)

	// Severity context.
	if decisionSeverity != "" && decisionSeverity != "none" {
		base += fmt.Sprintf("; decision severity %s (limit ×%.2f)", decisionSeverity, severityFactor)
	}
	return base
}

// buildMetadata populates risk metadata with strategy type and decision context for observability.
func (e *PositionExposureEvaluator) buildMetadata(strategyType, decisionSeverity, decisionRationale string) map[string]string {
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
