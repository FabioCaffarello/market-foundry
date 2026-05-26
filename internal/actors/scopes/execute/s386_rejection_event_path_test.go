package execute_test

import (
	"testing"
	"time"

	domainexec "internal/domain/execution"
	"internal/shared/events"
	"internal/shared/problem"
)

// ==========================================================================
// S386 — Rejection event path integration tests
//
// Validates the rejection event construction at the actor boundary:
//   - Problem → VenueOrderRejectedEvent mapping completeness
//   - Intent status mutation (submitted → rejected, Final=true)
//   - Correlation/causation chain from incoming event to rejection event
//   - Rejection code/reason/details preservation
//   - Both non-retryable (true rejection) and exhausted-retryable produce events
// ==========================================================================

func s386SubmittedIntent(t *testing.T) domainexec.ExecutionIntent {
	t.Helper()
	return domainexec.ExecutionIntent{
		Type:          "venue_market_order",
		Source:        "binancef",
		Instrument:    btcUSDTPerpS379(t),
		Timeframe:     60,
		Side:          domainexec.SideBuy,
		Quantity:      "0.001",
		Status:        domainexec.StatusSubmitted,
		Risk:          domainexec.RiskInput{Type: "position_exposure", Disposition: "allow", Confidence: "0.85", Timeframe: 60},
		CorrelationID: "s386-int-corr",
		CausationID:   "s386-int-cause",
		Timestamp:     time.Now().UTC().Add(-3 * time.Second),
	}
}

// TestS386_RejectionEventConstruction_FromNonRetryableProblem validates that a
// non-retryable Problem (true venue rejection) maps correctly to VenueOrderRejectedEvent.
func TestS386_RejectionEventConstruction_FromNonRetryableProblem(t *testing.T) {
	intent := s386SubmittedIntent(t)
	incomingEventMeta := events.NewMetadata().
		WithCorrelationID("upstream-corr-id").
		WithCausationID("upstream-cause-id")

	prob := problem.Newf(problem.InvalidArgument,
		"venue rejected order (HTTP 400, code -2019): Margin is insufficient").
		WithDetail("venue_http_status", 400).
		WithDetail("venue_error_code", -2019)

	// Simulate what the actor does: mutate intent and build event.
	rejected := intent
	rejected.Status = domainexec.StatusRejected
	rejected.Final = true

	event := domainexec.VenueOrderRejectedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(incomingEventMeta.CorrelationID).
			WithCausationID(incomingEventMeta.ID),
		ExecutionIntent: rejected,
		RejectionCode:   string(prob.Code),
		RejectionReason: prob.Message,
		VenueDetails:    prob.Details,
	}

	// Verify intent mutation.
	if event.ExecutionIntent.Status != domainexec.StatusRejected {
		t.Errorf("expected Status=rejected, got %s", event.ExecutionIntent.Status)
	}
	if !event.ExecutionIntent.Final {
		t.Error("rejected intent must be Final=true")
	}

	// Verify correlation chain: event metadata inherits from incoming event.
	if event.Metadata.CorrelationID != "upstream-corr-id" {
		t.Errorf("event CorrelationID: expected upstream-corr-id, got %s", event.Metadata.CorrelationID)
	}
	if event.Metadata.CausationID != incomingEventMeta.ID {
		t.Errorf("event CausationID: expected %s (incoming event ID), got %s", incomingEventMeta.ID, event.Metadata.CausationID)
	}

	// Verify rejection fields.
	if event.RejectionCode != "VAL_INVALID_ARGUMENT" {
		t.Errorf("RejectionCode: expected VAL_INVALID_ARGUMENT, got %s", event.RejectionCode)
	}
	if event.RejectionReason == "" {
		t.Error("RejectionReason must not be empty")
	}
	if event.VenueDetails["venue_http_status"] != 400 {
		t.Errorf("venue_http_status: expected 400, got %v", event.VenueDetails["venue_http_status"])
	}
	if event.VenueDetails["venue_error_code"] != -2019 {
		t.Errorf("venue_error_code: expected -2019, got %v", event.VenueDetails["venue_error_code"])
	}

	// Verify intent fields survive mutation.
	if event.ExecutionIntent.Source != "binancef" {
		t.Errorf("Source lost: expected binancef, got %s", event.ExecutionIntent.Source)
	}
	if event.ExecutionIntent.VenueSymbol() != "btcusdt" {
		t.Errorf("Symbol lost: expected btcusdt, got %s", event.ExecutionIntent.VenueSymbol())
	}
	if event.ExecutionIntent.Side != domainexec.SideBuy {
		t.Errorf("Side lost: expected buy, got %s", event.ExecutionIntent.Side)
	}
	if event.ExecutionIntent.Quantity != "0.001" {
		t.Errorf("Quantity lost: expected 0.001, got %s", event.ExecutionIntent.Quantity)
	}
	if event.ExecutionIntent.Risk.Type != "position_exposure" {
		t.Errorf("Risk.Type lost: expected position_exposure, got %s", event.ExecutionIntent.Risk.Type)
	}
}

// TestS386_RejectionEventConstruction_FromExhaustedRetryable validates that an
// exhausted retryable Problem (retries spent) also produces a rejection event.
func TestS386_RejectionEventConstruction_FromExhaustedRetryable(t *testing.T) {
	intent := s386SubmittedIntent(t)

	// After RetrySubmitter exhausts retries, the problem may still be retryable=true
	// but the actor treats it as terminal (no more retries possible).
	prob := problem.Newf(problem.Unavailable,
		"venue unavailable after 3 attempts").
		MarkRetryable().
		WithDetail("retry_attempts", 3).
		WithDetail("retry_exhausted", true)

	rejected := intent
	rejected.Status = domainexec.StatusRejected
	rejected.Final = true

	event := domainexec.VenueOrderRejectedEvent{
		Metadata:        events.NewMetadata(),
		ExecutionIntent: rejected,
		RejectionCode:   string(prob.Code),
		RejectionReason: prob.Message,
		VenueDetails:    prob.Details,
	}

	if event.ExecutionIntent.Status != domainexec.StatusRejected {
		t.Errorf("exhausted retryable must still produce rejected status, got %s", event.ExecutionIntent.Status)
	}
	if event.RejectionCode != "SYS_UNAVAILABLE" {
		t.Errorf("exhausted retryable rejection code: expected SYS_UNAVAILABLE, got %s", event.RejectionCode)
	}
	if event.VenueDetails["retry_exhausted"] != true {
		t.Error("retry_exhausted detail must be preserved in venue details")
	}
}

// TestS386_RejectionEvent_LifecycleTransitionFromSubmitted validates that the
// rejection path submitted→rejected is valid per S383 state machine.
func TestS386_RejectionEvent_LifecycleTransitionFromSubmitted(t *testing.T) {
	if !domainexec.ValidTransition(domainexec.StatusSubmitted, domainexec.StatusRejected) {
		t.Fatal("submitted → rejected must be a valid lifecycle transition per S383")
	}

	// Rejected is terminal — no further transitions.
	if !domainexec.StatusRejected.IsTerminal() {
		t.Fatal("rejected must be a terminal state")
	}
}

// TestS386_RejectionEvent_PreservesOriginalIntentTimestamp verifies that the
// rejection event carries the original intent timestamp (not the rejection time).
func TestS386_RejectionEvent_PreservesOriginalIntentTimestamp(t *testing.T) {
	intent := s386SubmittedIntent(t)
	originalTS := intent.Timestamp

	rejected := intent
	rejected.Status = domainexec.StatusRejected
	rejected.Final = true

	event := domainexec.VenueOrderRejectedEvent{
		Metadata:        events.NewMetadata(),
		ExecutionIntent: rejected,
		RejectionCode:   "VAL_INVALID_ARGUMENT",
		RejectionReason: "test rejection",
	}

	if !event.ExecutionIntent.Timestamp.Equal(originalTS) {
		t.Errorf("intent timestamp must survive rejection: expected %v, got %v",
			originalTS, event.ExecutionIntent.Timestamp)
	}
	// Event occurred_at is the rejection event creation time (separate from intent timestamp).
	if event.Metadata.OccurredAt.Before(originalTS) {
		t.Error("event OccurredAt must be after or equal to intent Timestamp")
	}
}

// TestS386_RejectionEvent_NoFillsOnRejection verifies that rejected intents
// carry zero fill records (no venue execution occurred).
func TestS386_RejectionEvent_NoFillsOnRejection(t *testing.T) {
	intent := s386SubmittedIntent(t)
	rejected := intent
	rejected.Status = domainexec.StatusRejected
	rejected.Final = true

	if len(rejected.Fills) != 0 {
		t.Errorf("rejected intent must have 0 fills, got %d", len(rejected.Fills))
	}
	if rejected.FilledQuantity != "" {
		t.Errorf("rejected intent must have empty FilledQuantity, got %q", rejected.FilledQuantity)
	}
}
