package strategy

import (
	"strconv"
	"time"

	domainstrategy "internal/domain/strategy"
)

const (
	defaultTargetOffset = "0.02"
	defaultStopOffset   = "0.01"
)

// MeanReversionEntryResolver resolves a mean_reversion_entry strategy from an RSI oversold decision.
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives decision values as primitive data (not decision.Decision structs) per DBI-9.
type MeanReversionEntryResolver struct {
	source    string
	symbol    string
	timeframe int
}

func NewMeanReversionEntryResolver(source, symbol string, timeframe int) *MeanReversionEntryResolver {
	return &MeanReversionEntryResolver{
		source:    source,
		symbol:    symbol,
		timeframe: timeframe,
	}
}

// Resolve processes a decision outcome and produces a strategy.
// decisionOutcome is the categorical result from the decision event.
// decisionConfidence is the decimal string from the decision event.
// decisionSeverity and decisionRationale carry the decision's semantic depth
// forward into DecisionInput for traceability. They do not alter resolution logic.
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

	switch decisionOutcome {
	case "triggered":
		direction = domainstrategy.DirectionLong
		confidence = decisionConfidence
		params = map[string]string{
			"entry":         "market",
			"target_offset": defaultTargetOffset,
			"stop_offset":   defaultStopOffset,
		}
	case "not_triggered":
		direction = domainstrategy.DirectionFlat
		confidence = "0.0000"
	case "insufficient":
		direction = domainstrategy.DirectionFlat
		confidence = "0.0000"
		metadata = map[string]string{
			"reason": "insufficient_data",
		}
	default:
		return domainstrategy.Strategy{}, false
	}

	// Validate confidence is parseable.
	if _, err := strconv.ParseFloat(confidence, 64); err != nil {
		return domainstrategy.Strategy{}, false
	}

	// Propagate decision rationale into strategy metadata for observability.
	if decisionRationale != "" {
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata["decision_rationale"] = decisionRationale
	}

	return domainstrategy.Strategy{
		Type:      "mean_reversion_entry",
		Source:    r.source,
		Symbol:    r.symbol,
		Timeframe: r.timeframe,
		Direction: direction,
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
