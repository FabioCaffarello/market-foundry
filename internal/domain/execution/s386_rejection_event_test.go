package execution_test

import (
	"testing"
	"time"

	domainexec "internal/domain/execution"
	"internal/shared/events"
)

// ==========================================================================
// S386 — Rejection event domain model tests
//
// Validates the VenueOrderRejectedEvent contract:
//   - Event interface compliance
//   - Metadata and correlation chain preservation
//   - Rejection-specific fields (code, reason, venue details)
//   - ExecutionIntent carries Status=rejected and Final=true
//   - Lifecycle alignment with S383 state machine
// ==========================================================================

func s386RejectedIntent(t *testing.T) domainexec.ExecutionIntent {
	t.Helper()
	return domainexec.ExecutionIntent{
		Type:          "venue_market_order",
		Source:        "binancef",
		Instrument:    btcUSDTPerp(t),
		Timeframe:     60,
		Side:          domainexec.SideBuy,
		Quantity:      "0.001",
		Status:        domainexec.StatusRejected,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "allow", Confidence: "0.85", Timeframe: 60},
		CorrelationID: "s386-corr-001",
		CausationID:   "s386-cause-001",
		Final:         true,
		Timestamp:     time.Now().UTC().Add(-5 * time.Second),
	}
}

func TestS386_RejectedEvent_ImplementsEventInterface(t *testing.T) {
	event := domainexec.VenueOrderRejectedEvent{
		Metadata:        events.NewMetadata(),
		ExecutionIntent: s386RejectedIntent(t),
		RejectionCode:   "VAL_INVALID_ARGUMENT",
		RejectionReason: "venue rejected order (HTTP 400, code -2019): Margin is insufficient",
	}

	// VenueOrderRejectedEvent must satisfy the Event interface.
	var _ events.Event = event

	if event.EventName() != domainexec.EventVenueOrderRejected {
		t.Errorf("expected event name %q, got %q", domainexec.EventVenueOrderRejected, event.EventName())
	}
	if event.EventMetadata().ID == "" {
		t.Error("event metadata ID must not be empty")
	}
	if event.EventMetadata().OccurredAt.IsZero() {
		t.Error("event metadata OccurredAt must not be zero")
	}
}

func TestS386_RejectedEvent_PreservesCorrelationChain(t *testing.T) {
	corrID := "s386-correlation-chain"
	causeID := "s386-causation-source"

	event := domainexec.VenueOrderRejectedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(corrID).
			WithCausationID(causeID),
		ExecutionIntent: s386RejectedIntent(t),
		RejectionCode:   "VAL_INVALID_ARGUMENT",
		RejectionReason: "insufficient margin",
	}

	if event.Metadata.CorrelationID != corrID {
		t.Errorf("CorrelationID: expected %s, got %s", corrID, event.Metadata.CorrelationID)
	}
	if event.Metadata.CausationID != causeID {
		t.Errorf("CausationID: expected %s, got %s", causeID, event.Metadata.CausationID)
	}
	// Intent-level correlation must also survive.
	if event.ExecutionIntent.CorrelationID != "s386-corr-001" {
		t.Errorf("intent CorrelationID: expected s386-corr-001, got %s", event.ExecutionIntent.CorrelationID)
	}
}

func TestS386_RejectedEvent_CarriesRejectionMetadata(t *testing.T) {
	details := map[string]any{
		"venue_http_status": 400,
		"venue_error_code":  -2019,
	}

	event := domainexec.VenueOrderRejectedEvent{
		Metadata:        events.NewMetadata(),
		ExecutionIntent: s386RejectedIntent(t),
		RejectionCode:   "VAL_INVALID_ARGUMENT",
		RejectionReason: "Margin is insufficient",
		VenueDetails:    details,
	}

	if event.RejectionCode != "VAL_INVALID_ARGUMENT" {
		t.Errorf("RejectionCode: expected VAL_INVALID_ARGUMENT, got %s", event.RejectionCode)
	}
	if event.RejectionReason != "Margin is insufficient" {
		t.Errorf("RejectionReason: expected 'Margin is insufficient', got %s", event.RejectionReason)
	}
	if event.VenueDetails["venue_http_status"] != 400 {
		t.Errorf("VenueDetails[venue_http_status]: expected 400, got %v", event.VenueDetails["venue_http_status"])
	}
}

func TestS386_RejectedEvent_IntentIsTerminalAndFinal(t *testing.T) {
	intent := s386RejectedIntent(t)

	if intent.Status != domainexec.StatusRejected {
		t.Errorf("rejected intent must have Status=rejected, got %s", intent.Status)
	}
	if !intent.Final {
		t.Error("rejected intent must be Final=true")
	}
	if !intent.Status.IsTerminal() {
		t.Error("StatusRejected must be terminal per S383 lifecycle")
	}
}

func TestS386_RejectedEvent_LifecycleTransitionValid(t *testing.T) {
	// S383: submitted → rejected is a valid transition.
	if !domainexec.ValidTransition(domainexec.StatusSubmitted, domainexec.StatusRejected) {
		t.Error("submitted → rejected must be a valid transition")
	}
	// S383: sent → rejected is also valid.
	if !domainexec.ValidTransition(domainexec.StatusSent, domainexec.StatusRejected) {
		t.Error("sent → rejected must be a valid transition")
	}
	// S383: rejected is absorbing — no further transitions allowed.
	for _, target := range []domainexec.Status{
		domainexec.StatusSubmitted, domainexec.StatusSent, domainexec.StatusAccepted,
		domainexec.StatusFilled, domainexec.StatusPartiallyFilled, domainexec.StatusCancelled,
	} {
		if domainexec.ValidTransition(domainexec.StatusRejected, target) {
			t.Errorf("rejected → %s must NOT be valid (terminal state)", target)
		}
	}
}

func TestS386_RejectedEvent_IntentValidatesWithRejectedStatus(t *testing.T) {
	intent := s386RejectedIntent(t)
	if prob := intent.Validate(); prob != nil {
		t.Errorf("rejected intent must be valid, got: %s", prob.Message)
	}
}

func TestS386_RejectedEvent_EventNameIsDistinctFromFill(t *testing.T) {
	if domainexec.EventVenueOrderRejected == domainexec.EventVenueOrderFilled {
		t.Error("rejection event name must differ from fill event name")
	}
}
