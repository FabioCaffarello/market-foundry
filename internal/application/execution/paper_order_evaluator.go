package execution

import (
	"strings"
	"time"

	domainexec "internal/domain/execution"
	"internal/domain/instrument"
)

// PaperOrderEvaluator translates a risk assessment into a paper order execution intent.
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives risk values as primitive data (not risk.RiskAssessment structs) per domain isolation.
type PaperOrderEvaluator struct {
	source     string
	symbol     string
	instrument instrument.CanonicalInstrument
	timeframe  int
}

func NewPaperOrderEvaluator(source, symbol string, timeframe int) *PaperOrderEvaluator {
	return &PaperOrderEvaluator{
		source:     source,
		symbol:     symbol,
		instrument: instrumentFromBinding(source, symbol),
		timeframe:  timeframe,
	}
}

// NewPaperOrderEvaluatorForInstrument constructs an evaluator with an explicit
// CanonicalInstrument, bypassing the source-based reconstruction. Used when the
// caller already carries the canonical instrument (e.g., the strategy consumer
// reading from `Strategy.Instrument`) and the upstream `Source` label is not a
// venue identifier recognized by `instrumentFromBinding`.
//
// TRANSITORY (H-6.b' → sunset H-6.c when the evaluator API itself migrates to
// CanonicalInstrument as the single source of truth).
func NewPaperOrderEvaluatorForInstrument(source string, inst instrument.CanonicalInstrument, timeframe int) *PaperOrderEvaluator {
	return &PaperOrderEvaluator{
		source:     source,
		symbol:     strings.ToLower(string(inst.Base) + string(inst.Quote)),
		instrument: inst,
		timeframe:  timeframe,
	}
}

// Evaluate processes a risk assessment and produces a paper order execution intent.
// riskDisposition is "approved", "modified", or "rejected".
// strategyDirection is "long", "short", or "flat".
// maxPositionPct is the risk-constrained position size (decimal string).
// strategyType identifies the originating strategy family for traceability.
// decisionSeverity carries the originating decision's severity for behavioral context.
// Returns an ExecutionIntent and true if evaluation succeeded.
func (e *PaperOrderEvaluator) Evaluate(
	riskType, riskDisposition, riskConfidence, maxPositionPct string,
	strategyDirection, strategyConfidence string,
	strategyType, decisionSeverity string,
	riskTimeframe int,
	ts time.Time,
) (domainexec.ExecutionIntent, bool) {
	// Determine side from disposition + direction.
	var side domainexec.Side
	var quantity string

	switch {
	case riskDisposition == "rejected":
		side = domainexec.SideNone
		quantity = "0"
	case strategyDirection == "flat":
		side = domainexec.SideNone
		quantity = "0"
	case strategyDirection == "long" && (riskDisposition == "approved" || riskDisposition == "modified"):
		side = domainexec.SideBuy
		quantity = maxPositionPct
	case strategyDirection == "short" && (riskDisposition == "approved" || riskDisposition == "modified"):
		side = domainexec.SideSell
		quantity = maxPositionPct
	default:
		// Unknown direction or disposition — no action.
		side = domainexec.SideNone
		quantity = "0"
	}

	if quantity == "" {
		quantity = "0"
	}

	return domainexec.ExecutionIntent{
		Type:       "paper_order",
		Source:     e.source,
		Instrument: e.instrument,
		Timeframe:  e.timeframe,
		Side:       side,
		Quantity:   quantity,
		Status:     domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:             riskType,
			Disposition:      riskDisposition,
			Confidence:       riskConfidence,
			Timeframe:        riskTimeframe,
			StrategyType:     strategyType,
			DecisionSeverity: decisionSeverity,
		},
		Parameters: map[string]string{
			"risk_type":           riskType,
			"risk_disposition":    riskDisposition,
			"strategy_direction":  strategyDirection,
			"strategy_confidence": strategyConfidence,
			"strategy_type":       strategyType,
			"decision_severity":   decisionSeverity,
			"max_position_pct":    maxPositionPct,
		},
		Final:     true,
		Timestamp: ts,
	}, true
}
