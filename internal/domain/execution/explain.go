package execution

import "time"

// SourcePathExplanation is the composite explainability surface for the
// source-driven execution path. It aggregates activation state, gate status,
// strategy consumer configuration, and last evaluation outcome into a single
// queryable view.
//
// S361: This type answers "why did the source generate execution or why was it blocked?"
// without requiring multiple endpoint round trips.
type SourcePathExplanation struct {
	// SourcePath identifies the canonical source-driven path.
	SourcePath string `json:"source_path"`

	// StrategyType is the strategy family being consumed.
	StrategyType string `json:"strategy_type"`

	// Activation is the current three-dimensional activation surface.
	Activation ActivationSurface `json:"activation"`

	// Gate is the current execution control gate state.
	Gate ControlGate `json:"gate"`

	// Config holds the operational configuration of the source-driven path.
	Config SourcePathConfig `json:"config"`

	// LastIntent is the most recent execution intent from the source path (nil if none).
	LastIntent *ExecutionIntent `json:"last_intent"`

	// LastResult is the most recent venue execution result (nil if none).
	LastResult *ExecutionIntent `json:"last_result"`

	// Propagation is the effective lifecycle status derived from intent and result.
	Propagation string `json:"propagation"`

	// ObservedAt is when this explanation was composed.
	ObservedAt time.Time `json:"observed_at"`
}

// SourcePathConfig holds the operational configuration exposed for explainability.
type SourcePathConfig struct {
	// MaxPositionPct is the current position size cap.
	MaxPositionPct string `json:"max_position_pct"`

	// MinConfidence is the minimum confidence threshold (empty = disabled).
	MinConfidence string `json:"min_confidence,omitempty"`

	// StalenessMaxAge is the maximum intent age before rejection.
	StalenessMaxAge string `json:"staleness_max_age"`

	// RiskType is the risk evaluation mode (always "pass_through" currently).
	RiskType string `json:"risk_type"`
}
