package decision

import (
	"fmt"
	"time"

	domaindecision "internal/domain/decision"
	"internal/domain/instrument"
)

// EMACrossoverEvaluator evaluates whether an EMA crossover signal indicates a bullish trend.
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives signal values as primitive data (not signal.Signal structs) per DBI-9.
//
// Input semantics (from EMA crossover signal sampler):
//
//	"bullish"  — fast EMA above slow EMA → triggered
//	"bearish"  — fast EMA below slow EMA → not_triggered
//	"neutral"  — EMAs equal within tolerance → not_triggered
type EMACrossoverEvaluator struct {
	source     string
	instrument instrument.CanonicalInstrument
	timeframe  int
}

// NewEMACrossoverEvaluatorForInstrument constructs the evaluator from
// a canonical Instrument directly — no source-string reconstruction.
// See NewRSISamplerForInstrument (signal package) for the
// boundary-helper rationale.
func NewEMACrossoverEvaluatorForInstrument(source string, inst instrument.CanonicalInstrument, timeframe int) *EMACrossoverEvaluator {
	return &EMACrossoverEvaluator{
		source:     source,
		instrument: inst,
		timeframe:  timeframe,
	}
}

// Evaluate processes an EMA crossover signal value and produces a decision.
// signalValue is the crossover direction string from the signal event: "bullish", "bearish", or "neutral".
// Returns a Decision and true if evaluation succeeded.
func (e *EMACrossoverEvaluator) Evaluate(signalType, signalValue string, signalTimeframe int, ts time.Time) (domaindecision.Decision, bool) {
	var outcome domaindecision.Outcome
	var severity domaindecision.Severity
	var confidence string
	var rationale string

	switch signalValue {
	case "bullish":
		outcome = domaindecision.OutcomeTriggered
		severity = domaindecision.SeverityModerate
		confidence = "0.7500"
		rationale = fmt.Sprintf("EMA crossover bullish: fast EMA above slow EMA on %ds timeframe; trend confirmation signal",
			signalTimeframe)

	case "bearish":
		outcome = domaindecision.OutcomeNotTriggered
		severity = domaindecision.SeverityNone
		confidence = "0.7500"
		rationale = fmt.Sprintf("EMA crossover bearish: fast EMA below slow EMA on %ds timeframe; no bullish crossover",
			signalTimeframe)

	case "neutral":
		outcome = domaindecision.OutcomeNotTriggered
		severity = domaindecision.SeverityNone
		confidence = "0.5000"
		rationale = fmt.Sprintf("EMA crossover neutral: fast and slow EMA within tolerance on %ds timeframe; insufficient divergence",
			signalTimeframe)

	default:
		return domaindecision.Decision{}, false
	}

	return domaindecision.Decision{
		Type:       "ema_crossover",
		Source:     e.source,
		Instrument: e.instrument,
		Timeframe:  e.timeframe,
		Outcome:    outcome,
		Severity:   severity,
		Confidence: confidence,
		Rationale:  rationale,
		Signals: []domaindecision.SignalInput{
			{Type: signalType, Value: signalValue, Timeframe: signalTimeframe},
		},
		Metadata: map[string]string{
			"crossover_direction": signalValue,
		},
		Final:     true,
		Timestamp: ts,
	}, true
}
