package risk

import (
	"strings"
)

// risk_scaling.go provides strategy-type-aware and severity-aware scaling functions
// used by risk evaluators. These functions implement the S251 behavioral activation:
// risk assessment varies by strategy type (counter-trend vs pro-trend risk profiles)
// and by decision severity (strong signals warrant different limits than weak ones).
//
// Design rules:
//   - Pure functions — no I/O, no state, no side effects
//   - Scaling maps are declared at package level for each evaluator's concern
//   - Unknown strategy types default to neutral factors to prevent silent failures
//   - Unknown severity values default to 1.0× (neutral) for backward compatibility
//   - All arithmetic uses float64; output is formatted to 4 decimal places

// --- Position Exposure: strategy-type confidence multiplier ---
// Replaces the former fixed ×0.95 multiplier.
// Counter-trend strategies carry inherently higher risk → lower confidence multiplier.
// Pro-trend strategies align with market momentum → higher confidence multiplier.
var positionExposureConfidenceFactor = map[string]float64{
	"mean_reversion_entry":   0.90, // counter-trend: higher risk → more conservative confidence
	"trend_following_entry":  0.95, // pro-trend: lower risk → less conservative confidence
	"squeeze_breakout_entry": 0.93, // momentum/volatility: moderate risk — false breakout exposure
}

const positionExposureConfidenceDefault = 0.92

// --- Position Exposure: severity-based position limit multiplier ---
// Strong decision signals justify larger position limits.
// Weak decision signals warrant more conservative sizing.
var positionExposureSeverityFactor = map[string]float64{
	"high":     1.15, // strong signal → allow up to 15% larger position
	"moderate": 1.00, // neutral
	"low":      0.80, // weak signal → reduce position limit by 20%
}

// --- Drawdown Limit: strategy-type confidence multiplier ---
// Replaces the former fixed ×0.90 multiplier.
// Counter-trend needs stricter risk assessment → lower multiplier.
// Pro-trend can tolerate slightly more → higher multiplier.
var drawdownConfidenceFactor = map[string]float64{
	"mean_reversion_entry":   0.85, // counter-trend: stricter drawdown assessment
	"trend_following_entry":  0.92, // pro-trend: slightly more tolerant
	"squeeze_breakout_entry": 0.90, // momentum/volatility: between counter-trend and pro-trend
}

const drawdownConfidenceDefault = 0.88

// --- Drawdown Limit: strategy-type stop distance multiplier ---
// Adjusts the base stop distance ceiling per strategy type.
// Counter-trend entries need tighter stops (quicker exit if wrong).
// Pro-trend entries need wider stops (room for the trend to develop).
var drawdownStopFactor = map[string]float64{
	"mean_reversion_entry":   0.85, // counter-trend → tighter stop distance ceiling
	"trend_following_entry":  1.15, // pro-trend → wider stop distance ceiling
	"squeeze_breakout_entry": 1.05, // momentum → slightly wider ceiling for breakout development
}

// --- Drawdown Limit: severity-based drawdown tolerance multiplier ---
// Strong decision signals justify slightly more drawdown tolerance.
// Weak decision signals warrant tighter drawdown limits.
var drawdownSeverityFactor = map[string]float64{
	"high":     1.15, // strong signal → tolerate 15% more drawdown
	"moderate": 1.00, // neutral
	"low":      0.80, // weak signal → tighten drawdown tolerance by 20%
}

// lookupFactor retrieves a scaling factor from a map, returning defaultVal if not found.
func lookupFactor(key string, factorMap map[string]float64, defaultVal float64) float64 {
	if f, ok := factorMap[key]; ok {
		return f
	}
	return defaultVal
}

// lookupSeverityFactor retrieves a severity-based factor, defaulting to 1.0 for unknown/empty/none.
// Normalizes input: trims whitespace and lowercases for defensive matching.
func lookupSeverityFactor(severity string, factorMap map[string]float64) float64 {
	normalized := strings.ToLower(strings.TrimSpace(severity))
	if normalized == "" || normalized == "none" {
		return 1.0
	}
	if f, ok := factorMap[normalized]; ok {
		return f
	}
	return 1.0
}
