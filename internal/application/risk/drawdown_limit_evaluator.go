package risk

import (
	"fmt"
	"strconv"
	"time"

	domainrisk "internal/domain/risk"
)

const (
	defaultMaxDrawdownPct = 0.05
	defaultStopDistancePct = 0.03
)

// DrawdownLimitEvaluator assesses strategy intent against configurable drawdown and stop-loss limits.
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives strategy values as primitive data (not strategy.Strategy structs) per domain isolation.
// Decision severity and rationale flow through for traceability and richer rationale generation.
type DrawdownLimitEvaluator struct {
	source          string
	symbol          string
	timeframe       int
	maxDrawdownPct  float64
	stopDistancePct float64
}

func NewDrawdownLimitEvaluator(source, symbol string, timeframe int) *DrawdownLimitEvaluator {
	return &DrawdownLimitEvaluator{
		source:          source,
		symbol:          symbol,
		timeframe:       timeframe,
		maxDrawdownPct:  defaultMaxDrawdownPct,
		stopDistancePct: defaultStopDistancePct,
	}
}

// Evaluate processes a strategy resolution and produces a drawdown risk assessment.
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

	baseParams := map[string]string{
		"max_drawdown_pct":  fmt.Sprintf("%.4f", e.maxDrawdownPct),
		"stop_distance_pct": fmt.Sprintf("%.4f", e.stopDistancePct),
	}

	// Flat strategies are always approved — no drawdown risk.
	if strategyDirection == "flat" {
		return domainrisk.RiskAssessment{
			Type:        "drawdown_limit",
			Source:      e.source,
			Symbol:      e.symbol,
			Timeframe:   e.timeframe,
			Disposition: domainrisk.DispositionApproved,
			Confidence:  "1.0000",
			Strategies:  []domainrisk.StrategyInput{strategyInput},
			Rationale:   "Flat strategy has no drawdown risk",
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

	// Stop distance: scale inversely with confidence — lower confidence → tighter stop.
	// High-confidence trades get wider stops; low-confidence trades get tighter stops.
	stopDistance := e.stopDistancePct * confidence
	if stopDistance < 0.0050 {
		stopDistance = 0.0050 // floor at 0.5% to avoid unrealistic stops
	}
	if stopDistance > e.stopDistancePct {
		stopDistance = e.stopDistancePct
	}

	// Determine disposition based on drawdown limits.
	var disposition domainrisk.Disposition
	var rationale string
	var constraints domainrisk.Constraints

	if stopDistance <= e.maxDrawdownPct {
		disposition = domainrisk.DispositionApproved
		rationale = e.buildRationale("approved", strategyDirection, stopDistance, decisionSeverity)
		constraints = domainrisk.Constraints{
			StopDistance: fmt.Sprintf("%.4f", stopDistance),
			MaxExposure:  fmt.Sprintf("%.4f", e.maxDrawdownPct),
		}
	} else {
		// Cap stop distance to max drawdown — modified disposition.
		cappedStop := e.maxDrawdownPct
		disposition = domainrisk.DispositionModified
		rationale = e.buildRationale("modified", strategyDirection, cappedStop, decisionSeverity)
		constraints = domainrisk.Constraints{
			StopDistance: fmt.Sprintf("%.4f", cappedStop),
			MaxExposure:  fmt.Sprintf("%.4f", e.maxDrawdownPct),
		}
	}

	// Risk confidence: higher strategy confidence → higher risk confidence.
	riskConfidence := fmt.Sprintf("%.4f", confidence*0.90)

	return domainrisk.RiskAssessment{
		Type:        "drawdown_limit",
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
func (e *DrawdownLimitEvaluator) buildRationale(outcome, direction string, stopDistance float64, decisionSeverity string) string {
	var base string
	switch outcome {
	case "approved":
		base = fmt.Sprintf("Stop distance %.4f within drawdown limits for %s", stopDistance, direction)
	case "modified":
		base = fmt.Sprintf("Stop distance capped to %.4f by drawdown limits for %s", stopDistance, direction)
	default:
		base = fmt.Sprintf("%s %s stop distance %.4f", direction, outcome, stopDistance)
	}
	if decisionSeverity != "" && decisionSeverity != "none" {
		base += fmt.Sprintf("; decision severity %s", decisionSeverity)
	}
	return base
}

// buildMetadata populates risk metadata with decision context for observability.
func (e *DrawdownLimitEvaluator) buildMetadata(decisionSeverity, decisionRationale string) map[string]string {
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
