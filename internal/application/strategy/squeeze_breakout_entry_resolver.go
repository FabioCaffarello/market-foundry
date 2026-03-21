package strategy

import (
	"fmt"
	"time"

	domainstrategy "internal/domain/strategy"
)

const (
	// Base parameters for squeeze breakout entry.
	// Breakout from a Bollinger squeeze expects a sharp directional move.
	baseBreakoutTargetPct = 0.04 // wider target — breakouts are momentum-driven
	baseBreakoutStopPct   = 0.015 // tighter stop — invalidated quickly if squeeze fails
)

// Severity-based confidence scaling factors for squeeze breakout.
// Higher severity → higher confidence in the breakout continuation.
var squeezeBreakoutSeverityScaling = map[string]float64{
	"high":     1.00, // Strong squeeze signal — full confidence
	"moderate": 0.90, // Standard squeeze signal — slight reduction
	"low":      0.80, // Weak squeeze signal — meaningful reduction
}

// Severity-based parameter multipliers for squeeze breakout.
// Higher severity → wider target (expect larger breakout), tighter stop (more conviction).
var squeezeBreakoutTargetMultiplier = map[string]float64{
	"high":     1.50, // Strong squeeze → expect bigger move
	"moderate": 1.00, // Standard → default parameters
	"low":      0.75, // Weak squeeze → smaller expected move
}

var squeezeBreakoutStopMultiplier = map[string]float64{
	"high":     0.75, // Strong squeeze → tighter stop (high conviction)
	"moderate": 1.00, // Standard → default stop
	"low":      1.50, // Weak squeeze → wider stop (allow more noise)
}

// SqueezeBreakoutEntryResolver resolves a squeeze_breakout_entry strategy from a bollinger_squeeze decision.
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives decision values as primitive data (not decision.Decision structs) per DBI-9.
//
// Semantic distinction from other strategy resolvers:
//   - mean_reversion_entry is counter-trend: enters long on oversold conditions.
//   - trend_following_entry is pro-trend: enters long on bullish crossover.
//   - squeeze_breakout_entry is volatility-driven: enters long on Bollinger squeeze detection,
//     anticipating a sharp directional breakout after a period of low volatility.
//
// Behavioral semantics (S250):
//
//	Decision severity directly influences strategy resolution:
//	- Confidence is scaled by severity (high=1.0×, moderate=0.90×, low=0.80×)
//	- Target is adjusted by severity (high=1.5×, moderate=1.0×, low=0.75×)
//	- Stop is adjusted by severity (high=0.75×, moderate=1.0×, low=1.5×)
//	This produces strategies that target wider breakouts on strong squeeze signals
//	and protect capital more conservatively on weak squeeze signals.
//
// Input semantics (from bollinger_squeeze decision):
//
//	"triggered"     — squeeze detected → long with breakout target
//	"not_triggered" — no squeeze detected → flat (no position)
//	"insufficient"  — insufficient signal data → flat with reason
type SqueezeBreakoutEntryResolver struct {
	source    string
	symbol    string
	timeframe int
}

func NewSqueezeBreakoutEntryResolver(source, symbol string, timeframe int) *SqueezeBreakoutEntryResolver {
	return &SqueezeBreakoutEntryResolver{
		source:    source,
		symbol:    symbol,
		timeframe: timeframe,
	}
}

// Resolve processes a decision outcome and produces a squeeze breakout strategy.
// decisionSeverity influences confidence scaling and parameter adjustment (S250 behavioral activation).
// decisionRationale is carried forward for traceability and rationale composition.
// Returns a Strategy and true if resolution succeeded.
func (r *SqueezeBreakoutEntryResolver) Resolve(
	decisionType, decisionOutcome, decisionConfidence, decisionSeverity, decisionRationale string,
	decisionTimeframe int,
	ts time.Time,
) (domainstrategy.Strategy, bool) {
	var direction domainstrategy.Direction
	var confidence string
	var params map[string]string
	var metadata map[string]string
	var rationale string

	switch decisionOutcome {
	case "triggered":
		direction = domainstrategy.DirectionLong

		// Scale confidence by decision severity.
		scaledConfidence, ok := ScaleConfidence(decisionConfidence, decisionSeverity, squeezeBreakoutSeverityScaling)
		if !ok {
			return domainstrategy.Strategy{}, false
		}
		confidence = scaledConfidence

		// Adjust parameters by decision severity.
		target := AdjustParam(baseBreakoutTargetPct, decisionSeverity, squeezeBreakoutTargetMultiplier)
		stop := AdjustParam(baseBreakoutStopPct, decisionSeverity, squeezeBreakoutStopMultiplier)

		params = map[string]string{
			"entry":             "market",
			"breakout_target_pct": FormatParam(target),
			"breakout_stop_pct":   FormatParam(stop),
		}

		rationale = buildTriggeredRationale(
			"squeeze_breakout_entry", decisionType, decisionSeverity,
			decisionConfidence, confidence,
			target, stop,
		)

	case "not_triggered":
		direction = domainstrategy.DirectionFlat
		confidence = "0.0000"
		rationale = fmt.Sprintf("decision %s not_triggered; no squeeze breakout signal", decisionType)

	case "insufficient":
		direction = domainstrategy.DirectionFlat
		confidence = "0.0000"
		metadata = map[string]string{
			"reason": "insufficient_data",
		}
		rationale = fmt.Sprintf("decision %s insufficient data; cannot evaluate squeeze breakout entry", decisionType)

	default:
		return domainstrategy.Strategy{}, false
	}

	// Populate metadata with decision context and severity influence.
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata["decision_type"] = decisionType
	metadata["decision_severity"] = decisionSeverity
	metadata["rationale"] = rationale
	if decisionRationale != "" {
		metadata["decision_rationale"] = decisionRationale
	}

	return domainstrategy.Strategy{
		Type:       "squeeze_breakout_entry",
		Source:     r.source,
		Symbol:     r.symbol,
		Timeframe:  r.timeframe,
		Direction:  direction,
		Confidence: confidence,
		Decisions: []domainstrategy.DecisionInput{
			{
				Type:       decisionType,
				Outcome:    decisionOutcome,
				Confidence: decisionConfidence,
				Severity:   decisionSeverity,
				Rationale:  decisionRationale,
				Timeframe:  decisionTimeframe,
			},
		},
		Parameters: params,
		Metadata:   metadata,
		Final:      true,
		Timestamp:  ts,
	}, true
}
