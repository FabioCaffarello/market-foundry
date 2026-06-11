package executionclient

import (
	"internal/domain/instrument"

	"time"

	"internal/domain/execution"
)

// ExecutionLatestQuery is the request contract for querying the latest execution intent of a given type.
type ExecutionLatestQuery struct {
	Type       string                         `json:"type"`
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`
}

// ExecutionLatestReply is the response contract for the latest execution intent query.
// ExecutionIntent is always present in JSON output (null when not found) — no omitempty.
type ExecutionLatestReply struct {
	ExecutionIntent *execution.ExecutionIntent `json:"execution_intent"`
}

// RejectionDetail carries the audit-relevant fields from a VenueOrderRejectedEvent
// that are not present in the ExecutionIntent alone. This closes the read-path gap
// where rejection code, reason, and venue details were lost at query time.
//
// S407: Introduced to make rejection audit metadata queryable via both the dedicated
// rejection route and the composite status endpoint.
type RejectionDetail struct {
	RejectionCode   string         `json:"rejection_code"`
	RejectionReason string         `json:"rejection_reason"`
	VenueDetails    map[string]any `json:"venue_details,omitempty"`
}

// ExecutionRejectionReply is the response contract for the dedicated rejection query route.
// S407: Provides full rejection audit trail including code, reason, and venue details.
type ExecutionRejectionReply struct {
	ExecutionIntent *execution.ExecutionIntent `json:"execution_intent"`
	Detail          *RejectionDetail           `json:"detail,omitempty"`
}

// ExecutionStatusQuery is the request contract for the composite execution status query.
// It returns intent (paper_order), result (venue_market_order), gate, and derived propagation.
type ExecutionStatusQuery struct {
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`
}

// ExecutionStatusReply is the composite response showing end-to-end execution status.
// Intent is the latest paper_order (derive output). Result is the latest venue_market_order fill (execute output).
// Rejection is the latest venue rejection (execute output). Gate is the current control gate.
// Propagation is the effective lifecycle status derived from all surfaces.
//
// S387: Rejection field added to close the read-path gap — rejected outcomes are now
// visible alongside fills for complete lifecycle queryability.
// S407: RejectionDetail added to preserve audit metadata (code, reason, venue details)
// at the composite query level — previously lost when only ExecutionIntent was returned.
type ExecutionStatusReply struct {
	Intent          *execution.ExecutionIntent `json:"intent"`
	Result          *execution.ExecutionIntent `json:"result"`
	Rejection       *execution.ExecutionIntent `json:"rejection"`
	RejectionDetail *RejectionDetail           `json:"rejection_detail,omitempty"`
	Gate            execution.ControlGate      `json:"gate"`
	Propagation     string                     `json:"propagation"`
}

// LifecycleListQuery is the request contract for listing all tracked execution
// lifecycle entries across KV buckets. Returns a per-partition-key summary with
// the effective propagation state.
//
// S413: Consolidates operational queryability — enables "show me all active
// lifecycle entries" without requiring exact source/symbol/timeframe foreknowledge.
// S466: Optional Source/Symbol filters for operator ergonomics — when set, only
// entries matching the filter(s) are returned.
type LifecycleListQuery struct {
	Source     string                         `json:"source,omitempty"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
}

// LifecycleEntry summarizes the lifecycle state for a single partition key
// (source/symbol/timeframe combination) by reading across the three execution
// KV buckets (paper_order, venue_fill, venue_rejection).
type LifecycleEntry struct {
	Key                string     `json:"key"`
	Source             string     `json:"source"`
	Symbol             string     `json:"symbol"`
	Timeframe          int        `json:"timeframe"`
	IntentStatus       string     `json:"intent_status"`
	IntentTimestamp    *time.Time `json:"intent_timestamp,omitempty"`
	FillStatus         string     `json:"fill_status"`
	FillTimestamp      *time.Time `json:"fill_timestamp,omitempty"`
	RejectionStatus    string     `json:"rejection_status"`
	RejectionTimestamp *time.Time `json:"rejection_timestamp,omitempty"`
	Propagation        string     `json:"propagation"`
}

// LifecycleListReply is the response contract for the lifecycle list query.
type LifecycleListReply struct {
	Entries []LifecycleEntry `json:"entries"`
	Total   int              `json:"total"`
}

// DeriveEffectivePropagation returns the effective lifecycle status from the intent, result, and rejection surfaces.
// Priority: most recent terminal state (result vs rejection by timestamp) > intent status > "none".
//
// S387: Updated to include rejection in propagation derivation. When both a fill result and
// a rejection exist, the more recent one determines the effective propagation.
func DeriveEffectivePropagation(intent, result, rejection *execution.ExecutionIntent) string {
	// Determine most recent venue outcome.
	switch {
	case result != nil && rejection != nil:
		// Both exist — most recent timestamp wins.
		if rejection.Timestamp.After(result.Timestamp) {
			return string(rejection.Status)
		}
		return string(result.Status)
	case result != nil:
		return string(result.Status)
	case rejection != nil:
		return string(rejection.Status)
	}
	if intent != nil {
		return string(intent.Status)
	}
	return "none"
}
