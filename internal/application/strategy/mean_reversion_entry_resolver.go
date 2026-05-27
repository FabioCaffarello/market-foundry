package strategy

import (
	"fmt"
	"time"

	"internal/domain/instrument"
	domainstrategy "internal/domain/strategy"
)

const (
	// Base parameters for mean reversion entry.
	baseTargetOffset = 0.02
	baseStopOffset   = 0.01
)

// Severity-based confidence scaling factors for mean reversion.
// Higher severity (more extreme oversold) → higher confidence in the reversion.
var meanReversionSeverityScaling = map[string]float64{
	"high":     1.00, // Extreme oversold — full confidence in reversion
	"moderate": 0.90, // Normal oversold — slight confidence reduction
	"low":      0.80, // Weak oversold — meaningful confidence reduction
}

// Severity-based parameter multipliers for mean reversion.
// Higher severity → wider target (expecting bigger reversion), tighter stop (higher conviction).
var meanReversionTargetMultiplier = map[string]float64{
	"high":     1.50, // Extreme signal → larger expected reversion
	"moderate": 1.00, // Normal signal → standard parameters
	"low":      0.75, // Weak signal → smaller expected reversion
}

var meanReversionStopMultiplier = map[string]float64{
	"high":     0.75, // Extreme signal → tighter stop (higher conviction)
	"moderate": 1.00, // Normal signal → standard stop
	"low":      1.50, // Weak signal → wider stop (more room needed)
}

// MeanReversionEntryResolver resolves a mean_reversion_entry strategy from an RSI oversold decision.
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives decision values as primitive data (not decision.Decision structs) per DBI-9.
//
// Behavioral semantics (S250):
//
//	Decision severity directly influences strategy resolution:
//	- Confidence is scaled by severity (high=1.0×, moderate=0.90×, low=0.80×)
//	- Target offset is adjusted by severity (high=1.5×, moderate=1.0×, low=0.75×)
//	- Stop offset is adjusted inversely (high=0.75×, moderate=1.0×, low=1.5×)
//	This produces strategies that are more aggressive on strong signals and
//	more cautious on weak signals, reflecting the decision's conviction level.
type MeanReversionEntryResolver struct {
	source     string
	instrument instrument.CanonicalInstrument
	timeframe  int
}

// NewMeanReversionEntryResolverForInstrument constructs the resolver
// from a canonical Instrument directly — no source-string
// reconstruction. See NewRSISamplerForInstrument (signal package) for
// the boundary-helper rationale.
func NewMeanReversionEntryResolverForInstrument(source string, inst instrument.CanonicalInstrument, timeframe int) *MeanReversionEntryResolver {
	return &MeanReversionEntryResolver{
		source:     source,
		instrument: inst,
		timeframe:  timeframe,
	}
}

// Resolve processes a decision outcome and produces a strategy.
// decisionOutcome is the categorical result from the decision event.
// decisionConfidence is the decimal string from the decision event.
// decisionSeverity influences confidence scaling and parameter adjustment (S250 behavioral activation).
// decisionRationale is carried forward for traceability and rationale composition.
// Returns a Strategy and true if resolution succeeded.
func (r *MeanReversionEntryResolver) Resolve(
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
		scaledConfidence, ok := ScaleConfidence(decisionConfidence, decisionSeverity, meanReversionSeverityScaling)
		if !ok {
			return domainstrategy.Strategy{}, false
		}
		confidence = scaledConfidence

		// Adjust parameters by decision severity.
		targetOffset := AdjustParam(baseTargetOffset, decisionSeverity, meanReversionTargetMultiplier)
		stopOffset := AdjustParam(baseStopOffset, decisionSeverity, meanReversionStopMultiplier)

		params = map[string]string{
			"entry":         "market",
			"target_offset": FormatParam(targetOffset),
			"stop_offset":   FormatParam(stopOffset),
		}

		rationale = buildTriggeredRationale(
			"mean_reversion_entry", decisionType, decisionSeverity,
			decisionConfidence, confidence,
			targetOffset, stopOffset,
		)

	case "not_triggered":
		direction = domainstrategy.DirectionFlat
		confidence = "0.0000"
		rationale = fmt.Sprintf("decision %s not_triggered; no entry signal for mean reversion", decisionType)

	case "insufficient":
		direction = domainstrategy.DirectionFlat
		confidence = "0.0000"
		metadata = map[string]string{
			"reason": "insufficient_data",
		}
		rationale = fmt.Sprintf("decision %s insufficient data; cannot evaluate mean reversion entry", decisionType)

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
		Type:       "mean_reversion_entry",
		Source:     r.source,
		Instrument: r.instrument,
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
