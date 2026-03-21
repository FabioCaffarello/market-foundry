package risk

import (
	"fmt"
	"strconv"
	"time"

	domainrisk "internal/domain/risk"
)

const (
	defaultMaxPositionPct         = 0.02
	defaultMaxPortfolioExposurePct = 0.10
)

// PositionExposureEvaluator assesses strategy intent against configurable exposure limits.
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives strategy values as primitive data (not strategy.Strategy structs) per domain isolation.
// Decision severity and rationale flow through for traceability and richer rationale generation.
type PositionExposureEvaluator struct {
	source                  string
	symbol                  string
	timeframe               int
	maxPositionPct          float64
	maxPortfolioExposurePct float64
}

func NewPositionExposureEvaluator(source, symbol string, timeframe int) *PositionExposureEvaluator {
	return &PositionExposureEvaluator{
		source:                  source,
		symbol:                  symbol,
		timeframe:               timeframe,
		maxPositionPct:          defaultMaxPositionPct,
		maxPortfolioExposurePct: defaultMaxPortfolioExposurePct,
	}
}

// Evaluate processes a strategy resolution and produces a risk assessment.
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

	baseParams := map[string]string{
		"max_position_pct":           fmt.Sprintf("%.4f", e.maxPositionPct),
		"max_portfolio_exposure_pct": fmt.Sprintf("%.4f", e.maxPortfolioExposurePct),
	}

	// Flat strategies are always approved with zero constraints.
	if strategyDirection == "flat" {
		return domainrisk.RiskAssessment{
			Type:        "position_exposure",
			Source:      e.source,
			Symbol:      e.symbol,
			Timeframe:   e.timeframe,
			Disposition: domainrisk.DispositionApproved,
			Confidence:  "1.0000",
			Strategies:  []domainrisk.StrategyInput{strategyInput},
			Rationale:   "Flat strategy requires no position",
			Parameters:  baseParams,
			Metadata:    e.buildMetadata(decisionSeverity, decisionRationale),
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

	// Position sizing: scale position by confidence, capped at max.
	requestedSize := confidence * e.maxPositionPct
	if requestedSize > e.maxPositionPct {
		requestedSize = e.maxPositionPct
	}

	// Determine disposition based on exposure limits.
	var disposition domainrisk.Disposition
	var rationale string
	var constraints domainrisk.Constraints

	if requestedSize <= e.maxPositionPct && requestedSize <= e.maxPortfolioExposurePct {
		disposition = domainrisk.DispositionApproved
		rationale = e.buildRationale("approved", strategyDirection, requestedSize, decisionSeverity)
		constraints = domainrisk.Constraints{
			MaxPositionSize: fmt.Sprintf("%.4f", requestedSize),
			MaxExposure:     fmt.Sprintf("%.4f", e.maxPortfolioExposurePct),
		}
	} else {
		// Cap to max allowed — modified disposition.
		cappedSize := e.maxPositionPct
		if cappedSize > e.maxPortfolioExposurePct {
			cappedSize = e.maxPortfolioExposurePct
		}
		disposition = domainrisk.DispositionModified
		rationale = e.buildRationale("modified", strategyDirection, cappedSize, decisionSeverity)
		constraints = domainrisk.Constraints{
			MaxPositionSize: fmt.Sprintf("%.4f", cappedSize),
			MaxExposure:     fmt.Sprintf("%.4f", e.maxPortfolioExposurePct),
		}
	}

	// Assess risk confidence: higher strategy confidence → higher risk confidence.
	riskConfidence := fmt.Sprintf("%.4f", confidence*0.95)

	return domainrisk.RiskAssessment{
		Type:        "position_exposure",
		Source:      e.source,
		Symbol:      e.symbol,
		Timeframe:   e.timeframe,
		Disposition: disposition,
		Confidence:  riskConfidence,
		Strategies:  []domainrisk.StrategyInput{strategyInput},
		Constraints: constraints,
		Rationale:   rationale,
		Parameters:  baseParams,
		Metadata:    e.buildMetadata(decisionSeverity, decisionRationale),
		Final:       true,
		Timestamp:   ts,
	}, true
}

// buildRationale generates a context-rich rationale incorporating decision severity.
func (e *PositionExposureEvaluator) buildRationale(outcome, direction string, positionSize float64, decisionSeverity string) string {
	base := fmt.Sprintf("%s %s position %.4f", direction, outcome, positionSize)
	switch outcome {
	case "approved":
		base = fmt.Sprintf("Position size %.4f within exposure limits", positionSize)
	case "modified":
		base = fmt.Sprintf("Position size capped to %.4f by exposure limits", positionSize)
	}
	if decisionSeverity != "" && decisionSeverity != "none" {
		base += fmt.Sprintf("; decision severity %s", decisionSeverity)
	}
	return base
}

// buildMetadata populates risk metadata with decision context for observability.
func (e *PositionExposureEvaluator) buildMetadata(decisionSeverity, decisionRationale string) map[string]string {
	meta := map[string]string{}
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
