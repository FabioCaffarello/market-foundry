package signalclient

import "internal/domain/signal"

// SignalLatestQuery is the request contract for querying the latest signal of a given type.
type SignalLatestQuery struct {
	Type      string `json:"type"`
	Source    string `json:"source"`
	Symbol   string `json:"symbol"`
	Timeframe int   `json:"timeframe"`
}

// SignalLatestReply is the response contract for the latest signal query.
// Signal is always present in JSON output (null when not found) — no omitempty.
type SignalLatestReply struct {
	Signal *signal.Signal `json:"signal"`
}
