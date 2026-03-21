package strategy

import (
	"fmt"
	"time"

	domainstrategy "internal/domain/strategy"
)

const (
	// Base parameters for trend following entry.
	baseTrailingStopPct = 0.03
	baseTakeProfitPct   = 0.05
)

// Severity-based confidence scaling factors for trend following.
// Higher severity → higher confidence in trend continuation.
var trendFollowingSeverityScaling = map[string]float64{
	"high":     1.00, // Strong trend signal — full confidence
	"moderate": 0.90, // Standard trend signal — slight reduction
	"low":      0.80, // Weak trend signal — meaningful reduction
}

// Severity-based parameter multipliers for trend following.
// Higher severity → tighter trailing stop (capture more of the trend), wider take profit (expect bigger move).
var trendFollowingTrailingStopMultiplier = map[string]float64{
	"high":     0.75, // Strong trend → tighter trailing stop (ride the trend closer)
	"moderate": 1.00, // Standard → default parameters
	"low":      1.50, // Weak trend → wider trailing stop (allow more noise)
}

var trendFollowingTakeProfitMultiplier = map[string]float64{
	"high":     1.50, // Strong trend → larger expected move
	"moderate": 1.00, // Standard → default target
	"low":      0.75, // Weak trend → smaller expected move
}

// TrendFollowingEntryResolver resolves a trend_following_entry strategy from an EMA crossover decision.
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives decision values as primitive data (not decision.Decision structs) per DBI-9.
//
// Semantic distinction from mean_reversion_entry:
//   - mean_reversion_entry is counter-trend: enters long on oversold conditions expecting reversion.
//   - trend_following_entry is pro-trend: enters long on bullish crossover expecting continuation.
//
// Behavioral semantics (S250):
//
//	Decision severity directly influences strategy resolution:
//	- Confidence is scaled by severity (high=1.0×, moderate=0.90×, low=0.80×)
//	- Trailing stop is adjusted by severity (high=0.75×, moderate=1.0×, low=1.5×)
//	- Take profit is adjusted by severity (high=1.5×, moderate=1.0×, low=0.75×)
//	This produces strategies that ride trends more aggressively on strong signals
//	and protect capital more conservatively on weak signals.
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
// decisionSeverity influences confidence scaling and parameter adjustment (S250 behavioral activation).
// decisionRationale is carried forward for traceability and rationale composition.
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
	var rationale string

	switch decisionOutcome {
	case "triggered":
		direction = domainstrategy.DirectionLong

		// Scale confidence by decision severity.
		scaledConfidence, ok := ScaleConfidence(decisionConfidence, decisionSeverity, trendFollowingSeverityScaling)
		if !ok {
			return domainstrategy.Strategy{}, false
		}
		confidence = scaledConfidence

		// Adjust parameters by decision severity.
		trailingStop := AdjustParam(baseTrailingStopPct, decisionSeverity, trendFollowingTrailingStopMultiplier)
		takeProfit := AdjustParam(baseTakeProfitPct, decisionSeverity, trendFollowingTakeProfitMultiplier)

		params = map[string]string{
			"entry":             "market",
			"trailing_stop_pct": FormatParam(trailingStop),
			"take_profit_pct":   FormatParam(takeProfit),
		}

		rationale = buildTriggeredRationale(
			"trend_following_entry", decisionType, decisionSeverity,
			decisionConfidence, confidence,
			trailingStop, takeProfit,
		)

	case "not_triggered":
		direction = domainstrategy.DirectionFlat
		confidence = "0.0000"
		rationale = fmt.Sprintf("decision %s not_triggered; no trend entry signal", decisionType)

	case "insufficient":
		direction = domainstrategy.DirectionFlat
		confidence = "0.0000"
		metadata = map[string]string{
			"reason": "insufficient_data",
		}
		rationale = fmt.Sprintf("decision %s insufficient data; cannot evaluate trend following entry", decisionType)

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
		Type:       "trend_following_entry",
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
