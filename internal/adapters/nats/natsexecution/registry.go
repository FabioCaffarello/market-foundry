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
	VenueMarketOrderFilled    natskit.EventSpec
	VenueMarketOrderRejected  natskit.EventSpec
	VenueMarketOrderLatest    natskit.ControlSpec
	VenueRejectionLatest      natskit.ControlSpec // S407: dedicated rejection query route

	// ── Cross-Family (shared) ────────────────────────────────────
	StatusLatest         natskit.ControlSpec
	LifecycleList        natskit.ControlSpec // S413: lifecycle list query across all execution KV buckets
	ControlGet           natskit.ControlSpec
	ControlSet           natskit.ControlSpec
	ActivationSurfaceGet natskit.ControlSpec

	// ── Session Metadata (S460) ──────────────────────────────────
	SessionGet  natskit.ControlSpec
	SessionList natskit.ControlSpec

	// ── Session Lifecycle Events (S490) ──────────────────────────
	SessionLifecycle natskit.EventSpec
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

	// ── Venue Family Stream (fills) ──────────────────────────────
	// Carries venue fill events produced by execute binary.
	// Separate stream from EXECUTION_EVENTS — different family, different owner.
	fillStream := natskit.StreamSpec{
		Name:     "EXECUTION_FILL_EVENTS",
		Subjects: []string{"execution.fill.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 256 * 1024 * 1024, // 256 MB — sized for local/CI event retention
	}

	// ── Venue Family Stream (rejections) ─────────────────────────
	// S386: Carries venue rejection events produced by execute binary.
	// Separate stream from fills — rejections are a distinct audit concern.
	rejectionStream := natskit.StreamSpec{
		Name:     "EXECUTION_REJECTION_EVENTS",
		Subjects: []string{"execution.rejection.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 128 * 1024 * 1024, // 128 MB — rejections are lower volume than fills
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
		// S386: Rejection event spec for auditable venue rejections.
		VenueMarketOrderRejected: natskit.EventSpec{
			Subject: "execution.rejection.venue_market_order",
			Type:    "execution.rejection.v1.venue_market_order_rejected",
			Stream:  rejectionStream,
		},
		VenueMarketOrderLatest: natskit.ControlSpec{
			Subject:     "execution.query.venue_market_order.latest",
			RequestType: "execution.query.v1.venue_market_order_latest_request",
			ReplyType:   "execution.query.v1.venue_market_order_latest_reply",
			QueueGroup:  "execution.query",
		},
		// S407: Dedicated rejection query route — closes the gap where rejections
		// were only queryable via composite status, losing audit detail.
		VenueRejectionLatest: natskit.ControlSpec{
			Subject:     "execution.query.venue_rejection.latest",
			RequestType: "execution.query.v1.venue_rejection_latest_request",
			ReplyType:   "execution.query.v1.venue_rejection_latest_reply",
			QueueGroup:  "execution.query",
		},

		// ── Cross-Family Specs ────────────────────────────────────
		StatusLatest: natskit.ControlSpec{
			Subject:     "execution.query.status.latest",
			RequestType: "execution.query.v1.status_latest_request",
			ReplyType:   "execution.query.v1.status_latest_reply",
			QueueGroup:  "execution.query",
		},
		// S413: Lifecycle list query — enumerate all tracked partition keys with effective propagation.
		LifecycleList: natskit.ControlSpec{
			Subject:     "execution.query.lifecycle.list",
			RequestType: "execution.query.v1.lifecycle_list_request",
			ReplyType:   "execution.query.v1.lifecycle_list_reply",
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

		// ── Session Lifecycle Events (S490) ──────────────────────────
		// Carries session lifecycle transitions (closed, halted) for event-driven
		// verification triggers. Separate stream from execution events — different
		// domain concern (operational accountability vs. order processing).
		SessionLifecycle: natskit.EventSpec{
			Subject: "execution.session.lifecycle",
			Type:    "execution.session.v1.lifecycle",
			Stream: natskit.StreamSpec{
				Name:     "SESSION_LIFECYCLE_EVENTS",
				Subjects: []string{"execution.session.lifecycle.>"},
				Storage:  jetstream.FileStorage,
				MaxAge:   168 * time.Hour,          // 7 days — verification can run late
				MaxBytes: 16 * 1024 * 1024,         // 16 MB — small events, bounded count
			},
		},

		// ── Session Metadata (S460) ──────────────────────────────────
		SessionGet: natskit.ControlSpec{
			Subject:     "execution.session.get",
			RequestType: "execution.session.v1.get_request",
			ReplyType:   "execution.session.v1.get_reply",
			QueueGroup:  "execution.query",
		},
		SessionList: natskit.ControlSpec{
			Subject:     "execution.session.list",
			RequestType: "execution.session.v1.list_request",
			ReplyType:   "execution.session.v1.list_reply",
			QueueGroup:  "execution.query",
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

// VenueRejectionLatestBucket is the KV bucket for venue family rejection results.
// S387: Persists the latest rejection per source/symbol/timeframe for audit queryability.
const VenueRejectionLatestBucket = "EXECUTION_VENUE_REJECTION_LATEST"

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

// ── Venue Rejection Consumers ─────────────────────────────────────

// S386: StoreVenueMarketOrderRejectionConsumer defines the durable consumer spec
// for store consuming venue rejection events from EXECUTION_REJECTION_EVENTS.
func StoreVenueMarketOrderRejectionConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-execution-venue-rejection", "execution.rejection.venue_market_order.>", "execution.rejection.v1.venue_market_order_rejected", "EXECUTION_REJECTION_EVENTS")
}

// S386: WriterVenueMarketOrderRejectionConsumer defines the durable consumer spec
// for writer consuming venue rejection events from EXECUTION_REJECTION_EVENTS.
func WriterVenueMarketOrderRejectionConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("writer-execution-venue-rejection", "execution.rejection.venue_market_order.>", "execution.rejection.v1.venue_market_order_rejected", "EXECUTION_REJECTION_EVENTS")
}

// ── Segment-Scoped Consumer Factory ───────────────────────────────────

// S401: ExecuteVenueIntakeConsumerForSegments creates a consumer spec that
// subscribes ONLY to subjects matching the given source prefixes. This
// ensures the execute binary's intake consumer receives intents only for
// segments it has adapters for — preventing cross-segment leakage at the
// NATS subscription level.
//
// When sources is empty, falls back to the wildcard ">" (all sources) for
// backwards compatibility with single-segment configs that don't specify
// explicit segments.
//
// Subject pattern per source: execution.events.paper_order.submitted.{source}.>
func ExecuteVenueIntakeConsumerForSegments(sources []string) natskit.ConsumerSpec {
	spec := natskit.NewConsumerSpec(
		"execute-venue-market-order-intake",
		"execution.events.paper_order.submitted.>",
		"execution.events.v1.paper_order_submitted",
		"EXECUTION_EVENTS",
	)

	if len(sources) > 0 {
		subjects := make([]string, len(sources))
		for i, src := range sources {
			subjects[i] = "execution.events.paper_order.submitted." + src + ".>"
		}
		spec.FilterSubjects = subjects
	}

	return spec
}

// ── Session Lifecycle Consumer ────────────────────────────────────────

// GatewaySessionLifecycleConsumer defines the durable consumer spec for the
// gateway binary consuming session lifecycle events for verification triggers.
// S490: Event-driven verification trigger — reacts to session close/halt.
func GatewaySessionLifecycleConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec(
		"gateway-verification-trigger",
		"execution.session.lifecycle.>",
		"execution.session.v1.lifecycle",
		"SESSION_LIFECYCLE_EVENTS",
	)
}

// DefaultStalenessMaxAge is the default maximum age for execution intents before they are
// considered stale and skipped. 120 seconds = 2× 1-minute timeframe.
const DefaultStalenessMaxAge = 120 * time.Second
