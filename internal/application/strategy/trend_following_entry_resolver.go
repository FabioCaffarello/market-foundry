package strategy

import (
	"strconv"
	"time"

	domainstrategy "internal/domain/strategy"
)

const (
	defaultTrailingStopPct = "0.03"
	defaultTakeProfitPct   = "0.05"
)

// TrendFollowingEntryResolver resolves a trend_following_entry strategy from an EMA crossover decision.
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives decision values as primitive data (not decision.Decision structs) per DBI-9.
//
// Semantic distinction from mean_reversion_entry:
//   - mean_reversion_entry is counter-trend: enters long on oversold conditions expecting reversion.
//   - trend_following_entry is pro-trend: enters long on bullish crossover expecting continuation.
//
// Input semantics (from ema_crossover decision):
//
//	"triggered"     — bullish crossover confirmed → long with trailing stop
//	"not_triggered" — no bullish crossover → flat (no position)
//	"insufficient"  — insufficient signal data → flat with reason
type TrendFollowingEntryResolver struct {
	source    string
	symbol    string
	timeframe int
}

func NewTrendFollowingEntryResolver(source, symbol string, timeframe int) *TrendFollowingEntryResolver {
	return &TrendFollowingEntryResolver{
		source:    source,
		symbol:    symbol,
		timeframe: timeframe,
	}
}

// Resolve processes a decision outcome and produces a trend-following strategy.
// decisionSeverity and decisionRationale carry the decision's semantic depth
// forward into DecisionInput for traceability. They do not alter resolution logic.
// Returns a Strategy and true if resolution succeeded.
func (r *TrendFollowingEntryResolver) Resolve(
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
			"entry":            "market",
			"trailing_stop_pct": defaultTrailingStopPct,
			"take_profit_pct":   defaultTakeProfitPct,
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
		Type:      "trend_following_entry",
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
