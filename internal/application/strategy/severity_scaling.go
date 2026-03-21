package strategy

import (
	"fmt"
	"strconv"
	"strings"
)

// severity_scaling.go provides shared severity-based behavioral adjustment functions
// used by all strategy resolvers. These functions implement the S250 behavioral activation:
// decision severity influences confidence scaling and parameter adjustment.
//
// Design rules:
//   - Pure functions — no I/O, no state, no side effects
//   - Scaling maps are owned by each resolver (not shared) to preserve type-specific semantics
//   - Unknown severity values default to 1.0× (neutral) to prevent silent failures
//   - All arithmetic uses float64; output is formatted to 4 decimal places

// ScaleConfidence applies a severity-based scaling factor to the decision confidence.
// Returns the scaled confidence as a 4-decimal string and true on success.
// Returns empty string and false if the input confidence is not parseable.
func ScaleConfidence(decisionConfidence, severity string, scalingMap map[string]float64) (string, bool) {
	raw, err := strconv.ParseFloat(decisionConfidence, 64)
	if err != nil {
		return "", false
	}

	factor := severityFactor(severity, scalingMap)
	scaled := raw * factor

	// Clamp to [0.0, 1.0].
	if scaled > 1.0 {
		scaled = 1.0
	}
	if scaled < 0.0 {
		scaled = 0.0
	}

	return fmt.Sprintf("%.4f", scaled), true
}

// AdjustParam applies a severity-based multiplier to a base parameter value.
// Unknown severity defaults to the base value (multiplier 1.0).
func AdjustParam(base float64, severity string, multiplierMap map[string]float64) float64 {
	factor := severityFactor(severity, multiplierMap)
	return base * factor
}

// FormatParam formats a parameter value to 2-decimal string for human readability.
func FormatParam(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

// severityFactor looks up the scaling factor for a given severity.
// Returns 1.0 for unknown or empty severity values (neutral behavior).
// Normalizes input: trims whitespace and lowercases for defensive matching.
func severityFactor(severity string, scalingMap map[string]float64) float64 {
	normalized := strings.ToLower(strings.TrimSpace(severity))
	if factor, ok := scalingMap[normalized]; ok {
		return factor
	}
	return 1.0
}

// buildTriggeredRationale constructs a human-readable rationale explaining
// the behavioral adjustments made by a strategy resolver based on decision context.
func buildTriggeredRationale(
	strategyType, decisionType, severity string,
	rawConfidence, scaledConfidence string,
	param1, param2 float64,
) string {
	if severity == "" || severity == "none" {
		return fmt.Sprintf(
			"%s triggered by %s; confidence %s (no severity adjustment); params [%.2f, %.2f]",
			strategyType, decisionType, scaledConfidence, param1, param2,
		)
	}
	return fmt.Sprintf(
		"%s triggered by %s (severity %s); confidence %s→%s; params adjusted [%.2f, %.2f]",
		strategyType, decisionType, severity, rawConfidence, scaledConfidence, param1, param2,
	)
}
