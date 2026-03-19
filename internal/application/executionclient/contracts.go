package executionclient

import "internal/domain/execution"

// ExecutionLatestQuery is the request contract for querying the latest execution intent of a given type.
type ExecutionLatestQuery struct {
	Type      string `json:"type"`
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
}

// ExecutionLatestReply is the response contract for the latest execution intent query.
// ExecutionIntent is always present in JSON output (null when not found) — no omitempty.
type ExecutionLatestReply struct {
	ExecutionIntent *execution.ExecutionIntent `json:"execution_intent"`
}

// ExecutionStatusQuery is the request contract for the composite execution status query.
// It returns intent (paper_order), result (venue_market_order), gate, and derived propagation.
type ExecutionStatusQuery struct {
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
}

// ExecutionStatusReply is the composite response showing end-to-end execution status.
// Intent is the latest paper_order (derive output). Result is the latest venue_market_order (execute output).
// Gate is the current control gate. Propagation is the effective lifecycle status derived from both surfaces.
type ExecutionStatusReply struct {
	Intent      *execution.ExecutionIntent `json:"intent"`
	Result      *execution.ExecutionIntent `json:"result"`
	Gate        execution.ControlGate      `json:"gate"`
	Propagation string                     `json:"propagation"`
}

// DeriveEffectivePropagation returns the effective lifecycle status from the intent and result surfaces.
// Priority: result status > intent status > "none".
func DeriveEffectivePropagation(intent, result *execution.ExecutionIntent) string {
	if result != nil {
		return string(result.Status)
	}
	if intent != nil {
		return string(intent.Status)
	}
	return "none"
}
