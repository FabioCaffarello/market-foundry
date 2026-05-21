package natsexecution_test

import (
	"testing"

	natsexecution "internal/adapters/nats/natsexecution"
)

// ==========================================================================
// S386 — Rejection event registry and consumer spec tests
//
// Validates the NATS subject, stream, and consumer contracts for rejection events:
//   - Rejection event spec is registered in DefaultRegistry
//   - Rejection stream is separate from fill stream
//   - Consumer specs follow naming conventions
//   - Subject hierarchy is correct
// ==========================================================================

func TestS386_Registry_RejectionEventSpecExists(t *testing.T) {
	reg := natsexecution.DefaultRegistry()

	spec := reg.VenueMarketOrderRejected
	if spec.Subject == "" {
		t.Fatal("VenueMarketOrderRejected subject must not be empty")
	}
	if spec.Type == "" {
		t.Fatal("VenueMarketOrderRejected type must not be empty")
	}
	if spec.Stream.Name == "" {
		t.Fatal("VenueMarketOrderRejected stream name must not be empty")
	}
}

func TestS386_Registry_RejectionSubjectFollowsConvention(t *testing.T) {
	reg := natsexecution.DefaultRegistry()

	expected := "execution.rejection.venue_market_order"
	if reg.VenueMarketOrderRejected.Subject != expected {
		t.Errorf("rejection subject: expected %q, got %q", expected, reg.VenueMarketOrderRejected.Subject)
	}
}

func TestS386_Registry_RejectionTypeFollowsConvention(t *testing.T) {
	reg := natsexecution.DefaultRegistry()

	expected := "execution.rejection.v1.venue_market_order_rejected"
	if reg.VenueMarketOrderRejected.Type != expected {
		t.Errorf("rejection type: expected %q, got %q", expected, reg.VenueMarketOrderRejected.Type)
	}
}

func TestS386_Registry_RejectionStreamIsSeparateFromFills(t *testing.T) {
	reg := natsexecution.DefaultRegistry()

	fillStream := reg.VenueMarketOrderFilled.Stream.Name
	rejStream := reg.VenueMarketOrderRejected.Stream.Name

	if fillStream == rejStream {
		t.Errorf("rejection stream must be separate from fill stream, both are %q", fillStream)
	}
	if rejStream != "EXECUTION_REJECTION_EVENTS" {
		t.Errorf("rejection stream: expected EXECUTION_REJECTION_EVENTS, got %q", rejStream)
	}
}

func TestS386_Registry_RejectionStreamSubjects(t *testing.T) {
	reg := natsexecution.DefaultRegistry()

	subjects := reg.VenueMarketOrderRejected.Stream.Config().Subjects
	if len(subjects) != 1 {
		t.Fatalf("expected 1 stream subject, got %d", len(subjects))
	}
	if subjects[0] != "execution.rejection.>" {
		t.Errorf("stream subject: expected execution.rejection.>, got %s", subjects[0])
	}
}

func TestS386_StoreRejectionConsumer_FollowsConventions(t *testing.T) {
	spec := natsexecution.StoreVenueMarketOrderRejectionConsumer()

	if spec.Durable != "store-execution-venue-rejection" {
		t.Errorf("durable: expected store-execution-venue-rejection, got %s", spec.Durable)
	}
	if spec.Event.Subject != "execution.rejection.venue_market_order.>" {
		t.Errorf("subject: expected execution.rejection.venue_market_order.>, got %s", spec.Event.Subject)
	}
	if spec.Event.Type != "execution.rejection.v1.venue_market_order_rejected" {
		t.Errorf("type: expected execution.rejection.v1.venue_market_order_rejected, got %s", spec.Event.Type)
	}
	if spec.Event.Stream.Name != "EXECUTION_REJECTION_EVENTS" {
		t.Errorf("stream: expected EXECUTION_REJECTION_EVENTS, got %s", spec.Event.Stream.Name)
	}
}

func TestS386_WriterRejectionConsumer_FollowsConventions(t *testing.T) {
	spec := natsexecution.WriterVenueMarketOrderRejectionConsumer()

	if spec.Durable != "writer-execution-venue-rejection" {
		t.Errorf("durable: expected writer-execution-venue-rejection, got %s", spec.Durable)
	}
	if spec.Event.Subject != "execution.rejection.venue_market_order.>" {
		t.Errorf("subject: expected execution.rejection.venue_market_order.>, got %s", spec.Event.Subject)
	}
}
