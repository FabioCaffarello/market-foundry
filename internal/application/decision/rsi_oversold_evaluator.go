package decision

import (
	"fmt"
	"strconv"
	"time"

	domaindecision "internal/domain/decision"
	"internal/domain/instrument"
)

const (
	defaultOversoldThreshold = 30.0
)

// RSIOversoldEvaluator evaluates whether an RSI signal indicates an oversold condition.
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives signal values as primitive data (not signal.Signal structs) per DBI-9.
type RSIOversoldEvaluator struct {
	source     string
	symbol     string
	instrument instrument.CanonicalInstrument
	timeframe  int
	threshold  float64
}

// NewRSIOversoldEvaluatorForInstrument constructs the evaluator from
// a canonical Instrument directly — no source-string reconstruction.
// See NewRSISamplerForInstrument (signal package) for the regression-
// shape rationale established in H-6.c.1.
func NewRSIOversoldEvaluatorForInstrument(source string, inst instrument.CanonicalInstrument, timeframe int) *RSIOversoldEvaluator {
	return &RSIOversoldEvaluator{
		source:     source,
		instrument: inst,
		timeframe:  timeframe,
		threshold:  defaultOversoldThreshold,
	}
}

// NewRSIOversoldEvaluator is the legacy (source, symbol) constructor.
// DEPRECATED (H-6.c.1 → sunset H-6.f). Use NewRSIOversoldEvaluatorForInstrument.
func NewRSIOversoldEvaluator(source, symbol string, timeframe int) *RSIOversoldEvaluator {
	e := NewRSIOversoldEvaluatorForInstrument(source, instrumentFromBinding(source, symbol), timeframe)
	e.symbol = symbol
	return e
}

// Evaluate processes an RSI signal value and produces a decision.
// signalValue is the RSI decimal string from the signal event.
// Returns a Decision and true if evaluation succeeded.
func (e *RSIOversoldEvaluator) Evaluate(signalType, signalValue string, signalTimeframe int, ts time.Time) (domaindecision.Decision, bool) {
	rsi, err := strconv.ParseFloat(signalValue, 64)
	if err != nil {
		return domaindecision.Decision{}, false
	}

	var outcome domaindecision.Outcome
	var severity domaindecision.Severity
	var confidence float64
	var rationale string

	if rsi < e.threshold {
		outcome = domaindecision.OutcomeTriggered
		severity = e.classifySeverity(rsi)
		// Confidence increases as RSI moves further below threshold.
		// At threshold: 0.5, at 0: 1.0
		confidence = 0.5 + 0.5*(e.threshold-rsi)/e.threshold
		if confidence > 1.0 {
			confidence = 1.0
		}
		distancePct := (e.threshold - rsi) / e.threshold * 100
		rationale = fmt.Sprintf("RSI %s below oversold threshold %.1f (distance %.1f%%); severity %s",
			signalValue, e.threshold, distancePct, severity)
	} else {
		outcome = domaindecision.OutcomeNotTriggered
		severity = domaindecision.SeverityNone
		// Confidence increases as RSI moves further above threshold.
		// At threshold: 0.5, at 100: ~0.85
		confidence = 0.5 + 0.5*(rsi-e.threshold)/(100.0-e.threshold)
		if confidence > 1.0 {
			confidence = 1.0
		}
		rationale = fmt.Sprintf("RSI %s above oversold threshold %.1f; not oversold",
			signalValue, e.threshold)
	}

	distancePct := 0.0
	if rsi < e.threshold {
		distancePct = (e.threshold - rsi) / e.threshold * 100
	}

	return domaindecision.Decision{
		Type:       "rsi_oversold",
		Source:     e.source,
		Instrument: e.instrument,
		Timeframe:  e.timeframe,
		Outcome:    outcome,
		Severity:   severity,
		Confidence: strconv.FormatFloat(confidence, 'f', 4, 64),
		Rationale:  rationale,
		Signals: []domaindecision.SignalInput{
			{Type: signalType, Value: signalValue, Timeframe: signalTimeframe},
		},
		Metadata: map[string]string{
			"threshold":    strconv.FormatFloat(e.threshold, 'f', 1, 64),
			"rsi_zone":     string(e.classifyZone(rsi)),
			"distance_pct": strconv.FormatFloat(distancePct, 'f', 1, 64),
		},
		Final:     true,
		Timestamp: ts,
	}, true
}

// classifySeverity returns the severity based on how far RSI is below the threshold.
// Only meaningful for triggered decisions (rsi < threshold).
func (e *RSIOversoldEvaluator) classifySeverity(rsi float64) domaindecision.Severity {
	distance := e.threshold - rsi
	switch {
	case distance >= 20:
		return domaindecision.SeverityHigh
	case distance >= 10:
		return domaindecision.SeverityModerate
	default:
		return domaindecision.SeverityLow
	}
}

// classifyZone returns a human-readable RSI zone label for metadata.
func (e *RSIOversoldEvaluator) classifyZone(rsi float64) domaindecision.Severity {
	if rsi >= e.threshold {
		return domaindecision.SeverityNone
	}
	return e.classifySeverity(rsi)
}
