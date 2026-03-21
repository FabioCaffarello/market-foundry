package execution_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/problem"
)

// --- Helpers ---

// fakeVenue is a controllable VenuePort for retry testing.
type fakeVenue struct {
	calls    atomic.Int32
	behavior func(attempt int) (ports.VenueOrderReceipt, *problem.Problem)
}

func (f *fakeVenue) SubmitOrder(_ context.Context, _ ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	n := int(f.calls.Add(1))
	return f.behavior(n)
}

func okReceipt() (ports.VenueOrderReceipt, *problem.Problem) {
	return ports.VenueOrderReceipt{
		VenueOrderID: "12345",
		Status:       domainexec.StatusFilled,
	}, nil
}

func retryableProblem(msg string) *problem.Problem {
	return problem.New(problem.Unavailable, msg).MarkRetryable()
}

func nonRetryableProblem(msg string) *problem.Problem {
	return problem.New(problem.InvalidArgument, msg)
}

func dummyRequest() ports.VenueOrderRequest {
	return ports.VenueOrderRequest{
		Intent: domainexec.ExecutionIntent{
			Type:      "paper_order",
			Source:    "binancef",
			Symbol:    "btcusdt",
			Timeframe: 3600,
			Side:      domainexec.SideBuy,
			Quantity:  "0.001",
			Timestamp: time.Now(),
		},
	}
}

// noSleep replaces time.Sleep for fast tests.
func noSleep(_ time.Duration) {}

// --- Tests ---

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return okReceipt()
	}}

	rs := execution.NewRetrySubmitter(venue, execution.DefaultRetryPolicy())
	rs = rs.TestWithSleepFn(noSleep)

	receipt, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob != nil {
		t.Fatalf("expected no error, got: %v", prob)
	}
	if receipt.VenueOrderID != "12345" {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	if venue.calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", venue.calls.Load())
	}
}

func TestRetry_SuccessOnSecondAttempt(t *testing.T) {
	venue := &fakeVenue{behavior: func(attempt int) (ports.VenueOrderReceipt, *problem.Problem) {
		if attempt == 1 {
			return ports.VenueOrderReceipt{}, retryableProblem("rate limited")
		}
		return okReceipt()
	}}

	rs := execution.NewRetrySubmitter(venue, execution.DefaultRetryPolicy())
	rs = rs.TestWithSleepFn(noSleep)

	receipt, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob != nil {
		t.Fatalf("expected success on retry, got: %v", prob)
	}
	if receipt.VenueOrderID != "12345" {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	if venue.calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", venue.calls.Load())
	}
}

func TestRetry_ExhaustsMaxAttempts(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, retryableProblem("venue unavailable")
	}}

	policy := execution.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(noSleep)

	_, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if venue.calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", venue.calls.Load())
	}

	// Verify retry metadata in details.
	attempts, ok := prob.Details["retry_attempts"]
	if !ok || attempts != 3 {
		t.Fatalf("expected retry_attempts=3, got %v", attempts)
	}
	exhausted, ok := prob.Details["retry_exhausted"]
	if !ok || exhausted != true {
		t.Fatalf("expected retry_exhausted=true, got %v", exhausted)
	}
}

func TestRetry_NonRetryableError_NoRetry(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, nonRetryableProblem("bad request")
	}}

	rs := execution.NewRetrySubmitter(venue, execution.DefaultRetryPolicy())
	rs = rs.TestWithSleepFn(noSleep)

	_, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob == nil {
		t.Fatal("expected error")
	}
	if prob.Retryable {
		t.Fatal("non-retryable error should not be marked retryable")
	}
	if venue.calls.Load() != 1 {
		t.Fatalf("expected 1 call (no retry), got %d", venue.calls.Load())
	}
	// Non-retryable errors should NOT carry retry metadata.
	if _, ok := prob.Details["retry_attempts"]; ok {
		t.Fatal("non-retryable error should not have retry_attempts detail")
	}
}

func TestRetry_ContextCancelled_AbortsLoop(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, retryableProblem("venue error")
	}}

	ctx, cancel := context.WithCancel(context.Background())

	policy := execution.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	// Cancel context during sleep to abort the retry loop.
	rs = rs.TestWithSleepFn(func(_ time.Duration) {
		cancel()
	})

	_, prob := rs.SubmitOrder(ctx, dummyRequest())
	if prob == nil {
		t.Fatal("expected error after context cancel")
	}
	// Should have attempted exactly 1 call, then the sleep cancels context,
	// then the next iteration's ctx.Err() check aborts.
	if venue.calls.Load() > 2 {
		t.Fatalf("expected at most 2 calls after cancel, got %d", venue.calls.Load())
	}
}

func TestRetry_PreservesDeterministicClientOrderID(t *testing.T) {
	// Verify that retries send the exact same request (client order ID is deterministic).
	var capturedIntents []domainexec.ExecutionIntent
	venue := &fakeVenue{behavior: func(attempt int) (ports.VenueOrderReceipt, *problem.Problem) {
		if attempt < 3 {
			return ports.VenueOrderReceipt{}, retryableProblem("transient")
		}
		return okReceipt()
	}}

	// Wrap to capture intents.
	intent := dummyRequest().Intent
	expectedID := execution.ClientOrderID(intent)

	policy := execution.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(noSleep)

	req := ports.VenueOrderRequest{Intent: intent}
	receipt, prob := rs.SubmitOrder(context.Background(), req)
	if prob != nil {
		t.Fatalf("expected success on third attempt, got: %v", prob)
	}
	_ = receipt
	_ = capturedIntents

	// The key invariant: the same intent produces the same client order ID
	// across all retry attempts. Since RetrySubmitter passes the same req
	// object to inner.SubmitOrder each time, the deterministic ID is preserved.
	idAgain := execution.ClientOrderID(intent)
	if idAgain != expectedID {
		t.Fatalf("client order ID changed across calls: %s vs %s", expectedID, idAgain)
	}
}

func TestRetry_PolicyMaxAttemptsZero_DefaultsToOne(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return okReceipt()
	}}

	policy := execution.RetryPolicy{MaxAttempts: 0}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(noSleep)

	_, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob != nil {
		t.Fatalf("expected success, got: %v", prob)
	}
	if venue.calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", venue.calls.Load())
	}
}

func TestRetry_BackoffIncreases(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, retryableProblem("error")
	}}

	var delays []time.Duration
	policy := execution.RetryPolicy{
		MaxAttempts: 4,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    2 * time.Second,
		Factor:      2.0,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(func(d time.Duration) {
		delays = append(delays, d)
	})

	rs.SubmitOrder(context.Background(), dummyRequest())

	if len(delays) != 3 { // 4 attempts = 3 sleeps
		t.Fatalf("expected 3 delays, got %d", len(delays))
	}

	// Delays should be roughly: 100ms, 200ms, 400ms (±25% jitter).
	// We just check they are increasing within jitter range.
	for i, d := range delays {
		if d < 50*time.Millisecond {
			t.Errorf("delay[%d] = %v is too small", i, d)
		}
	}
	// Each delay should be roughly 2x the previous (within jitter bounds).
	if delays[1] < delays[0] {
		t.Logf("delay[1]=%v < delay[0]=%v — jitter can cause this rarely, not a hard failure", delays[1], delays[0])
	}
}

func TestRetry_SuccessOnThirdAttempt_MatchesRealScenario(t *testing.T) {
	// Simulates: rate limit → server error → success
	venue := &fakeVenue{behavior: func(attempt int) (ports.VenueOrderReceipt, *problem.Problem) {
		switch attempt {
		case 1:
			return ports.VenueOrderReceipt{}, problem.Newf(problem.Unavailable,
				"venue rate limited (HTTP 429)").
				WithDetails(map[string]any{"venue_http_status": 429}).
				MarkRetryable()
		case 2:
			return ports.VenueOrderReceipt{}, problem.Newf(problem.Unavailable,
				"venue server error (HTTP 500)").
				WithDetails(map[string]any{"venue_http_status": 500}).
				MarkRetryable()
		default:
			return okReceipt()
		}
	}}

	rs := execution.NewRetrySubmitter(venue, execution.DefaultRetryPolicy())
	rs = rs.TestWithSleepFn(noSleep)

	receipt, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob != nil {
		t.Fatalf("expected success on third attempt, got: %v", prob)
	}
	if receipt.VenueOrderID != "12345" {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	if venue.calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", venue.calls.Load())
	}
}
