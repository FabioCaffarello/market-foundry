package natsexecution

import (
	"time"

	"internal/adapters/nats/natskit"

	"github.com/nats-io/nats.go/jetstream"
)

// Registry defines the NATS subject and stream contracts for the execution domain.
//
// The execution domain contains two distinct families with separate ownership:
//
//	Paper Family (paper_order):
//	  Owner: derive binary.
//	  Produces PaperOrderSubmittedEvent on EXECUTION_EVENTS stream.
//	  Store materializes to EXECUTION_PAPER_ORDER_LATEST KV bucket.
//	  Represents simulated intent evaluation — no venue interaction.
//
//	Venue Family (venue_market_order):
//	  Owner: execute binary.
//	  Consumes from EXECUTION_EVENTS (intake from paper family as transitional bridge).
//	  Produces VenueOrderFilledEvent on EXECUTION_FILL_EVENTS stream.
//	  Store materializes to EXECUTION_VENUE_MARKET_ORDER_LATEST KV bucket.
//	  Represents venue-submitted order results.
//
//	Cross-Family (shared):
//	  StatusLatest — composite query reading both families + control gate.
//	  ControlGet/ControlSet — global execution gate (kill switch).
//
// NOTE: In paper mode, the execute binary's intake consumer subscribes to
// paper_order subjects as a transitional bridge. When real venue intent
// subjects are introduced, the intake consumer will migrate to
// venue-specific subjects. See docs/architecture/execution-family-separation-after-paper-step.md.
type Registry struct {
	// ── Paper Family ─────────────────────────────────────────────
	PaperOrderSubmitted natskit.EventSpec
	PaperOrderLatest    natskit.ControlSpec

	// ── Venue Family ─────────────────────────────────────────────
	VenueMarketOrderFilled natskit.EventSpec
	VenueMarketOrderLatest natskit.ControlSpec

	// ── Cross-Family (shared) ────────────────────────────────────
	StatusLatest         natskit.ControlSpec
	ControlGet           natskit.ControlSpec
	ControlSet           natskit.ControlSpec
	ActivationSurfaceGet natskit.ControlSpec
}

func DefaultRegistry() Registry {
	// ── Paper Family Stream ──────────────────────────────────────
	// Carries paper_order intent events produced by derive.
	// Also used as transitional intake source for execute binary in paper mode.
	eventStream := natskit.StreamSpec{
		Name:     "EXECUTION_EVENTS",
		Subjects: []string{"execution.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 256 * 1024 * 1024, // 256 MB — sized for local/CI event retention
	}

	// ── Venue Family Stream ──────────────────────────────────────
	// Carries venue fill events produced by execute binary.
	// Separate stream from EXECUTION_EVENTS — different family, different owner.
	fillStream := natskit.StreamSpec{
		Name:     "EXECUTION_FILL_EVENTS",
		Subjects: []string{"execution.fill.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 256 * 1024 * 1024, // 256 MB — sized for local/CI event retention
	}

	return Registry{
		// ── Paper Family Specs ────────────────────────────────────
		PaperOrderSubmitted: natskit.EventSpec{
			Subject: "execution.events.paper_order.submitted",
			Type:    "execution.events.v1.paper_order_submitted",
			Stream:  eventStream,
		},
		PaperOrderLatest: natskit.ControlSpec{
			Subject:     "execution.query.paper_order.latest",
			RequestType: "execution.query.v1.paper_order_latest_request",
			ReplyType:   "execution.query.v1.paper_order_latest_reply",
			QueueGroup:  "execution.query",
		},

		// ── Venue Family Specs ────────────────────────────────────
		VenueMarketOrderFilled: natskit.EventSpec{
			Subject: "execution.fill.venue_market_order",
			Type:    "execution.fill.v1.venue_market_order_filled",
			Stream:  fillStream,
		},
		VenueMarketOrderLatest: natskit.ControlSpec{
			Subject:     "execution.query.venue_market_order.latest",
			RequestType: "execution.query.v1.venue_market_order_latest_request",
			ReplyType:   "execution.query.v1.venue_market_order_latest_reply",
			QueueGroup:  "execution.query",
		},

		// ── Cross-Family Specs ────────────────────────────────────
		StatusLatest: natskit.ControlSpec{
			Subject:     "execution.query.status.latest",
			RequestType: "execution.query.v1.status_latest_request",
			ReplyType:   "execution.query.v1.status_latest_reply",
			QueueGroup:  "execution.query",
		},
		ControlGet: natskit.ControlSpec{
			Subject:     "execution.control.get",
			RequestType: "execution.control.v1.get_request",
			ReplyType:   "execution.control.v1.get_reply",
			QueueGroup:  "execution.control",
		},
		ControlSet: natskit.ControlSpec{
			Subject:     "execution.control.set",
			RequestType: "execution.control.v1.set_request",
			ReplyType:   "execution.control.v1.set_reply",
			QueueGroup:  "execution.control",
		},
		ActivationSurfaceGet: natskit.ControlSpec{
			Subject:     "execution.activation.surface",
			RequestType: "execution.activation.v1.surface_request",
			ReplyType:   "execution.activation.v1.surface_reply",
			QueueGroup:  "execution.control",
		},
	}
}

// LatestSpecByType returns the ControlSpec for the execution type's latest query.
// Returns false if the type is not registered.
func (r Registry) LatestSpecByType(execType string) (natskit.ControlSpec, bool) {
	switch execType {
	case "paper_order":
		return r.PaperOrderLatest, true
	case "venue_market_order":
		return r.VenueMarketOrderLatest, true
	default:
		return natskit.ControlSpec{}, false
	}
}

// ── Writer Consumer Specs ─────────────────────────────────────────
// Paper order is codegen-governed (markers below).

// codegen:begin consumer_spec family=paper_order source=codegen/families/paper_order.yaml
// WriterPaperOrderExecutionConsumer defines the durable consumer spec for writer consuming
// paper order execution events from EXECUTION_EVENTS.
func WriterPaperOrderExecutionConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-execution-paper-order",
		Event: natskit.EventSpec{
			Subject: "execution.events.paper_order.submitted.>",
			Type:    "execution.events.v1.paper_order_submitted",
			Stream: natskit.StreamSpec{
				Name: "EXECUTION_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=paper_order

// ── Paper Family Consumer ─────────────────────────────────────────

// StorePaperOrderExecutionConsumer defines the durable consumer spec for store consuming
// paper order execution events from EXECUTION_EVENTS.
func StorePaperOrderExecutionConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-execution-paper-order", "execution.events.paper_order.submitted.>", "execution.events.v1.paper_order_submitted", "EXECUTION_EVENTS")
}

// ── Venue Family Consumers ────────────────────────────────────────

// ExecuteVenueMarketOrderIntakeConsumer defines the durable consumer spec for the execute binary
// consuming execution intents from EXECUTION_EVENTS for venue submission.
//
// TRANSITIONAL BRIDGE (paper mode): subscribes to paper_order subjects because derive only
// produces PaperOrderSubmittedEvent. Will migrate to venue-specific subjects in a future stage.
func ExecuteVenueMarketOrderIntakeConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("execute-venue-market-order-intake", "execution.events.paper_order.submitted.>", "execution.events.v1.paper_order_submitted", "EXECUTION_EVENTS")
}

// VenueMarketOrderLatestBucket is the KV bucket for venue family fill results.
const VenueMarketOrderLatestBucket = "EXECUTION_VENUE_MARKET_ORDER_LATEST"

// StoreVenueMarketOrderFillConsumer defines the durable consumer spec for store consuming
// venue market order fill events from EXECUTION_FILL_EVENTS.
func StoreVenueMarketOrderFillConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-execution-venue-market-order-fill", "execution.fill.venue_market_order.>", "execution.fill.v1.venue_market_order_filled", "EXECUTION_FILL_EVENTS")
}

// WriterVenueMarketOrderFillConsumer defines the durable consumer spec for writer consuming
// venue market order fill events from EXECUTION_FILL_EVENTS.
// S317: closes the persistence round-trip gap — venue fills now reach ClickHouse.
func WriterVenueMarketOrderFillConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("writer-execution-venue-fill", "execution.fill.venue_market_order.>", "execution.fill.v1.venue_market_order_filled", "EXECUTION_FILL_EVENTS")
}

// DefaultStalenessMaxAge is the default maximum age for execution intents before they are
// considered stale and skipped. 120 seconds = 2× 1-minute timeframe.
const DefaultStalenessMaxAge = 120 * time.Second
