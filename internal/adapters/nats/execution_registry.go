package nats

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// ExecutionRegistry defines the NATS subject and stream contracts for the execution domain.
//
// The execution domain contains two distinct families with separate ownership:
//
//   Paper Family (paper_order):
//     Owner: derive binary.
//     Produces PaperOrderSubmittedEvent on EXECUTION_EVENTS stream.
//     Store materializes to EXECUTION_PAPER_ORDER_LATEST KV bucket.
//     Represents simulated intent evaluation — no venue interaction.
//
//   Venue Family (venue_market_order):
//     Owner: execute binary.
//     Consumes from EXECUTION_EVENTS (intake from paper family as transitional bridge).
//     Produces VenueOrderFilledEvent on EXECUTION_FILL_EVENTS stream.
//     Store materializes to EXECUTION_VENUE_MARKET_ORDER_LATEST KV bucket.
//     Represents venue-submitted order results.
//
//   Cross-Family (shared):
//     StatusLatest — composite query reading both families + control gate.
//     ControlGet/ControlSet — global execution gate (kill switch).
//
// NOTE: In paper mode, the execute binary's intake consumer subscribes to
// paper_order subjects as a transitional bridge. When real venue intent
// subjects are introduced, the intake consumer will migrate to
// venue-specific subjects. See docs/architecture/execution-family-separation-after-paper-step.md.
type ExecutionRegistry struct {
	// ── Paper Family ─────────────────────────────────────────────
	PaperOrderSubmitted EventSpec
	PaperOrderLatest    ControlSpec

	// ── Venue Family ─────────────────────────────────────────────
	VenueMarketOrderFilled EventSpec
	VenueMarketOrderLatest ControlSpec

	// ── Cross-Family (shared) ────────────────────────────────────
	StatusLatest ControlSpec
	ControlGet   ControlSpec
	ControlSet   ControlSpec
}

func DefaultExecutionRegistry() ExecutionRegistry {
	// ── Paper Family Stream ──────────────────────────────────────
	// Carries paper_order intent events produced by derive.
	// Also used as transitional intake source for execute binary in paper mode.
	eventStream := StreamSpec{
		Name:     "EXECUTION_EVENTS",
		Subjects: []string{"execution.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 2 * 1024 * 1024 * 1024, // 2 GB
	}

	// ── Venue Family Stream ──────────────────────────────────────
	// Carries venue fill events produced by execute binary.
	// Separate stream from EXECUTION_EVENTS — different family, different owner.
	fillStream := StreamSpec{
		Name:     "EXECUTION_FILL_EVENTS",
		Subjects: []string{"execution.fill.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 2 * 1024 * 1024 * 1024, // 2 GB
	}

	return ExecutionRegistry{
		// ── Paper Family Specs ────────────────────────────────────
		PaperOrderSubmitted: EventSpec{
			Subject: "execution.events.paper_order.submitted",
			Type:    "execution.events.v1.paper_order_submitted",
			Stream:  eventStream,
		},
		PaperOrderLatest: ControlSpec{
			Subject:     "execution.query.paper_order.latest",
			RequestType: "execution.query.v1.paper_order_latest_request",
			ReplyType:   "execution.query.v1.paper_order_latest_reply",
			QueueGroup:  "execution.query",
		},

		// ── Venue Family Specs ────────────────────────────────────
		VenueMarketOrderFilled: EventSpec{
			Subject: "execution.fill.venue_market_order",
			Type:    "execution.fill.v1.venue_market_order_filled",
			Stream:  fillStream,
		},
		VenueMarketOrderLatest: ControlSpec{
			Subject:     "execution.query.venue_market_order.latest",
			RequestType: "execution.query.v1.venue_market_order_latest_request",
			ReplyType:   "execution.query.v1.venue_market_order_latest_reply",
			QueueGroup:  "execution.query",
		},

		// ── Cross-Family Specs ────────────────────────────────────
		StatusLatest: ControlSpec{
			Subject:     "execution.query.status.latest",
			RequestType: "execution.query.v1.status_latest_request",
			ReplyType:   "execution.query.v1.status_latest_reply",
			QueueGroup:  "execution.query",
		},
		ControlGet: ControlSpec{
			Subject:     "execution.control.get",
			RequestType: "execution.control.v1.get_request",
			ReplyType:   "execution.control.v1.get_reply",
			QueueGroup:  "execution.control",
		},
		ControlSet: ControlSpec{
			Subject:     "execution.control.set",
			RequestType: "execution.control.v1.set_request",
			ReplyType:   "execution.control.v1.set_reply",
			QueueGroup:  "execution.control",
		},
	}
}

// LatestSpecByType returns the ControlSpec for the execution type's latest query.
// Returns false if the type is not registered.
func (r ExecutionRegistry) LatestSpecByType(execType string) (ControlSpec, bool) {
	switch execType {
	case "paper_order":
		return r.PaperOrderLatest, true
	case "venue_market_order":
		return r.VenueMarketOrderLatest, true
	default:
		return ControlSpec{}, false
	}
}

// ── Paper Family Consumer ─────────────────────────────────────────

// StorePaperOrderExecutionConsumer defines the durable consumer spec for store consuming
// paper order execution events from EXECUTION_EVENTS.
// Family: paper_order. Owner: store binary.
func StorePaperOrderExecutionConsumer() ConsumerSpec {
	return ConsumerSpec{
		Durable: "store-execution-paper-order",
		Event: EventSpec{
			Subject: "execution.events.paper_order.submitted.>",
			Type:    "execution.events.v1.paper_order_submitted",
			Stream: StreamSpec{
				Name: "EXECUTION_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// ── Venue Family Consumers ────────────────────────────────────────

// ExecuteVenueMarketOrderIntakeConsumer defines the durable consumer spec for the execute binary
// consuming execution intents from EXECUTION_EVENTS for venue submission.
// Family: venue_market_order (intake side). Owner: execute binary.
//
// TRANSITIONAL BRIDGE (paper mode): This consumer currently subscribes to
// paper_order subjects because derive only produces PaperOrderSubmittedEvent.
// When venue-specific intent subjects are introduced (future stage), this
// consumer's filter subject will migrate to execution.events.venue_market_order.submitted.>
// and a new event type will replace PaperOrderSubmittedEvent for venue intake.
// See docs/architecture/execution-family-separation-after-paper-step.md.
func ExecuteVenueMarketOrderIntakeConsumer() ConsumerSpec {
	return ConsumerSpec{
		Durable: "execute-venue-market-order-intake",
		Event: EventSpec{
			// NOTE: Subscribes to paper_order subjects as transitional bridge.
			// This will migrate to venue-specific subjects when venue intent events are introduced.
			Subject: "execution.events.paper_order.submitted.>",
			Type:    "execution.events.v1.paper_order_submitted",
			Stream: StreamSpec{
				Name: "EXECUTION_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// ExecutionVenueMarketOrderLatestBucket is the KV bucket for venue family fill results.
// Family: venue_market_order. Authority: store binary (FillProjectionActor).
const ExecutionVenueMarketOrderLatestBucket = "EXECUTION_VENUE_MARKET_ORDER_LATEST"

// StoreVenueMarketOrderFillConsumer defines the durable consumer spec for store consuming
// venue market order fill events from EXECUTION_FILL_EVENTS.
// Family: venue_market_order. Owner: store binary.
func StoreVenueMarketOrderFillConsumer() ConsumerSpec {
	return ConsumerSpec{
		Durable: "store-execution-venue-market-order-fill",
		Event: EventSpec{
			Subject: "execution.fill.venue_market_order.>",
			Type:    "execution.fill.v1.venue_market_order_filled",
			Stream: StreamSpec{
				Name: "EXECUTION_FILL_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// DefaultStalenessMaxAge is the default maximum age for execution intents before they are
// considered stale and skipped. 120 seconds = 2× 1-minute timeframe.
const DefaultStalenessMaxAge = 120 * time.Second
