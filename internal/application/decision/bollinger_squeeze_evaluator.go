package decision

import (
	"fmt"
	"strconv"
	"time"

	domaindecision "internal/domain/decision"
	"internal/domain/instrument"
)

const (
	// defaultSqueezeThreshold is the relative bandwidth threshold below which
	// a Bollinger Squeeze is detected. bandwidth / SMA < 0.10 → squeeze.
	defaultSqueezeThreshold = 0.10
)

// BollingerSqueezeEvaluator evaluates whether a Bollinger signal indicates a squeeze
// condition (low volatility / bandwidth compression). The evaluator consumes the
// bollinger signal's %B value and bandwidth metadata to detect when the Bollinger Bands
// contract below a relative threshold, signaling potential breakout conditions.
//
// Pure application logic — no I/O, no actor references, no NATS dependency.
// Receives signal values as primitive data per DBI-9.
type BollingerSqueezeEvaluator struct {
	source           string
	instrument       instrument.CanonicalInstrument
	timeframe        int
	squeezeThreshold float64
}

// NewBollingerSqueezeEvaluatorForInstrument constructs the evaluator from
// a canonical Instrument directly — no source-string reconstruction.
// See NewRSISamplerForInstrument (signal package) for the
// boundary-helper rationale.
func NewBollingerSqueezeEvaluatorForInstrument(source string, inst instrument.CanonicalInstrument, timeframe int) *BollingerSqueezeEvaluator {
	return &BollingerSqueezeEvaluator{
		source:           source,
		instrument:       inst,
		timeframe:        timeframe,
		squeezeThreshold: defaultSqueezeThreshold,
	}
}

// Evaluate processes a bollinger signal and its metadata to detect a squeeze condition.
// signalValue is the %B decimal string. metadata must contain "bandwidth" and "sma" keys.
// Returns a Decision and true if evaluation succeeded.
func (e *BollingerSqueezeEvaluator) Evaluate(signalType, signalValue string, signalTimeframe int, ts time.Time, metadata map[string]string) (domaindecision.Decision, bool) {
	pctB, err := strconv.ParseFloat(signalValue, 64)
	if err != nil {
		return domaindecision.Decision{}, false
	}

	bandwidthStr, hasBW := metadata["bandwidth"]
	smaStr, hasSMA := metadata["sma"]
	if !hasBW || !hasSMA {
		return domaindecision.Decision{}, false
	}

	bandwidth, err := strconv.ParseFloat(bandwidthStr, 64)
	if err != nil {
		return domaindecision.Decision{}, false
	}

	sma, err := strconv.ParseFloat(smaStr, 64)
	if err != nil {
		return domaindecision.Decision{}, false
	}

	// Relative bandwidth: bandwidth / SMA. Zero SMA means degenerate data.
	if sma <= 0 {
		return domaindecision.Decision{}, false
	}
	relativeBW := bandwidth / sma

	var outcome domaindecision.Outcome
	var severity domaindecision.Severity
	var confidence float64
	var rationale string

	if relativeBW < e.squeezeThreshold {
		outcome = domaindecision.OutcomeTriggered
		severity = e.classifySeverity(relativeBW)

		// Confidence increases as relative bandwidth drops further below threshold.
		// At threshold: 0.5, approaching 0: 1.0.
		if e.squeezeThreshold > 0 {
			confidence = 0.5 + 0.5*(e.squeezeThreshold-relativeBW)/e.squeezeThreshold
		} else {
			confidence = 0.5
		}
		if confidence > 1.0 {
			confidence = 1.0
		}

		zone := e.classifyZone(pctB)
		rationale = fmt.Sprintf("Bollinger squeeze detected: relative bandwidth %.4f below threshold %.4f (severity %s); %%B=%.4f zone=%s on %ds timeframe",
			relativeBW, e.squeezeThreshold, severity, pctB, zone, signalTimeframe)
	} else {
		outcome = domaindecision.OutcomeNotTriggered
		severity = domaindecision.SeverityNone

		// Confidence increases as bandwidth moves further above threshold.
		excess := relativeBW - e.squeezeThreshold
		confidence = 0.5 + 0.5*excess/(excess+e.squeezeThreshold)
		if confidence > 1.0 {
			confidence = 1.0
		}

		rationale = fmt.Sprintf("No Bollinger squeeze: relative bandwidth %.4f above threshold %.4f; %%B=%.4f on %ds timeframe",
			relativeBW, e.squeezeThreshold, pctB, signalTimeframe)
	}

	return domaindecision.Decision{
		Type:       "bollinger_squeeze",
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
			"squeeze_threshold":  strconv.FormatFloat(e.squeezeThreshold, 'f', 4, 64),
			"relative_bandwidth": strconv.FormatFloat(relativeBW, 'f', 4, 64),
			"bandwidth":          bandwidthStr,
			"sma":                smaStr,
			"pct_b":              signalValue,
			"pct_b_zone":         e.classifyZone(pctB),
		},
		Final:     true,
		Timestamp: ts,
	}, true
}

// classifySeverity returns the severity based on how compressed the bandwidth is.
// Tighter squeeze → higher severity.
func (e *BollingerSqueezeEvaluator) classifySeverity(relativeBW float64) domaindecision.Severity {
	ratio := relativeBW / e.squeezeThreshold
	switch {
	case ratio <= 0.25:
		return domaindecision.SeverityHigh // extremely compressed (< 25% of threshold)
	case ratio <= 0.50:
		return domaindecision.SeverityModerate // moderately compressed (25-50% of threshold)
	default:
		return domaindecision.SeverityLow // mild squeeze (50-100% of threshold)
	}
}

// classifyZone returns the %B zone label for metadata enrichment.
func (e *BollingerSqueezeEvaluator) classifyZone(pctB float64) string {
	switch {
	case pctB < 0.20:
		return "lower" // near or below lower band
	case pctB > 0.80:
		return "upper" // near or above upper band
	default:
		return "middle" // between bands
	}
}
