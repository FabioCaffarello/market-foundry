package strategyclient

import "internal/domain/strategy"

// StrategyLatestQuery is the request contract for querying the latest strategy of a given type.
type StrategyLatestQuery struct {
	Type      string `json:"type"`
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
}

// StrategyLatestReply is the response contract for the latest strategy query.
// Strategy is always present in JSON output (null when not found) — no omitempty.
type StrategyLatestReply struct {
	Strategy *strategy.Strategy `json:"strategy"`
}
