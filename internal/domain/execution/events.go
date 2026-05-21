package execution

import "internal/shared/events"

// ── Paper Family Events ──────────────────────────────────────────
//
// Owner: derive binary.
// Stream: EXECUTION_EVENTS (execution.events.paper_order.submitted.{source}.{symbol}.{timeframe}).
// Consumers: store (projection), execute (venue intake — transitional bridge in paper mode).
//
// The paper family represents simulated execution intents produced by the derive
// binary's PaperOrderEvaluatorActor. These events carry pre-filled intents
// (Final=true, simulated fills) and require no venue interaction.

const (
	// EventPaperOrderSubmitted is emitted by derive when a paper order execution intent is produced.
	EventPaperOrderSubmitted events.Name = "paper_order_submitted"
)

// PaperOrderSubmittedEvent is emitted by derive when a paper order execution intent is produced.
// Family: paper_order. Owner: derive binary.
type PaperOrderSubmittedEvent struct {
	Metadata        events.Metadata `json:"metadata"`
	ExecutionIntent ExecutionIntent `json:"execution_intent"`
}

func (e PaperOrderSubmittedEvent) EventName() events.Name         { return EventPaperOrderSubmitted }
func (e PaperOrderSubmittedEvent) EventMetadata() events.Metadata { return e.Metadata }

// ── Venue Family Events ──────────────────────────────────────────
//
// Owner: execute binary.
// Stream: EXECUTION_FILL_EVENTS (execution.fill.venue_market_order.{source}.{symbol}.{timeframe}).
// Consumers: store (fill projection).
//
// The venue family represents order results produced by the execute binary's
// VenueAdapterActor after submitting intents to a VenuePort implementation.
// In paper mode, the venue adapter uses PaperVenueAdapter (simulated fills).
// In future venue mode, a real exchange adapter will produce these events.

const (
	// EventVenueOrderFilled is emitted by execute when a venue order receives a fill.
	EventVenueOrderFilled events.Name = "venue_order_filled"
)

// VenueOrderFilledEvent is emitted by the execute binary when a venue adapter produces a fill.
// Family: venue_market_order. Owner: execute binary.
type VenueOrderFilledEvent struct {
	Metadata        events.Metadata `json:"metadata"`
	ExecutionIntent ExecutionIntent `json:"execution_intent"`
	VenueOrderID    string          `json:"venue_order_id"`
}

func (e VenueOrderFilledEvent) EventName() events.Name         { return EventVenueOrderFilled }
func (e VenueOrderFilledEvent) EventMetadata() events.Metadata { return e.Metadata }

// ── Venue Rejection Events ─────────────────────────────────────
//
// Owner: execute binary.
// Stream: EXECUTION_REJECTION_EVENTS (execution.rejection.venue_market_order.{source}.{symbol}.{timeframe}).
// Consumers: store (rejection projection), writer (ClickHouse persistence).
//
// Rejection events close the observability gap identified by S385: venue rejections
// previously existed only as Problem returns to the actor layer, with no downstream
// event trail. This event makes rejections auditable and queryable.

const (
	// EventVenueOrderRejected is emitted by execute when a venue submission is rejected.
	EventVenueOrderRejected events.Name = "venue_order_rejected"
)

// VenueOrderRejectedEvent is emitted by the execute binary when a venue adapter
// returns a non-retryable rejection (e.g. insufficient margin, invalid parameters).
// Family: venue_market_order. Owner: execute binary.
//
// The event preserves the original ExecutionIntent with Status=rejected and carries
// the rejection reason from the Problem for audit trail completeness.
type VenueOrderRejectedEvent struct {
	Metadata        events.Metadata `json:"metadata"`
	ExecutionIntent ExecutionIntent `json:"execution_intent"`
	RejectionCode   string          `json:"rejection_code"`
	RejectionReason string          `json:"rejection_reason"`
	VenueDetails    map[string]any  `json:"venue_details,omitempty"`
}

func (e VenueOrderRejectedEvent) EventName() events.Name         { return EventVenueOrderRejected }
func (e VenueOrderRejectedEvent) EventMetadata() events.Metadata { return e.Metadata }
