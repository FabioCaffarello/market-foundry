package decisionclient

import (
	"internal/domain/instrument"

	"internal/domain/decision"
)

// DecisionLatestQuery is the request contract for querying the latest decision of a given type.
type DecisionLatestQuery struct {
	Type       string                         `json:"type"`
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`
}

// DecisionLatestReply is the response contract for the latest decision query.
// Decision is always present in JSON output (null when not found) — no omitempty.
type DecisionLatestReply struct {
	Decision *decision.Decision `json:"decision"`
}
