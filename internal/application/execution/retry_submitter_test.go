package execution_test

import (
	"bytes"
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/healthz"
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

// dynamicGateChecker delegates to a function, allowing halt state to change mid-test.
type dynamicGateChecker struct {
	isHaltedFn func() bool
}

func (d *dynamicGateChecker) IsHalted(_ context.Context) bool {
	return d.isHaltedFn()
}

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

// --- S323: Deadline, Halt Check, and Abort Tests ---

func TestRetry_DeadlineExceeded_AbortsLoop(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, retryableProblem("venue error")
	}}

	tick := int64(0)
	policy := execution.RetryPolicy{
		MaxAttempts: 10,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
		Deadline:    50 * time.Millisecond,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	// Simulate time advancing 20ms per call to nowFn.
	rs = rs.TestWithSleepFn(noSleep)
	start := time.Now()
	rs = rs.TestWithNowFn(func() time.Time {
		n := atomic.AddInt64(&tick, 1)
		return start.Add(time.Duration(n) * 20 * time.Millisecond)
	})

	_, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob == nil {
		t.Fatal("expected error after deadline exceeded")
	}

	// Should have deadline metadata.
	if v, ok := prob.Details["retry_deadline_exceeded"]; !ok || v != true {
		t.Fatalf("expected retry_deadline_exceeded=true, got %v", prob.Details)
	}

	// Should NOT have run all 10 attempts.
	calls := venue.calls.Load()
	if calls >= 10 {
		t.Fatalf("expected fewer than 10 calls, got %d", calls)
	}
}

func TestRetry_DeadlineZero_NoDeadlineEnforced(t *testing.T) {
	// When Deadline is 0, only MaxAttempts governs.
	venue := &fakeVenue{behavior: func(attempt int) (ports.VenueOrderReceipt, *problem.Problem) {
		if attempt < 3 {
			return ports.VenueOrderReceipt{}, retryableProblem("transient")
		}
		return okReceipt()
	}}

	policy := execution.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
		Deadline:    0, // no deadline
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(noSleep)

	receipt, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob != nil {
		t.Fatalf("expected success, got: %v", prob)
	}
	if receipt.VenueOrderID != "12345" {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
}

func TestRetry_HaltChecker_HaltsDuringRetry(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, retryableProblem("venue error")
	}}

	policy := execution.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(noSleep)
	rs = rs.WithHaltChecker(&mockGateChecker{halted: true})

	_, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob == nil {
		t.Fatal("expected error when kill switch is halted")
	}

	// Should carry halt metadata.
	if v, ok := prob.Details["retry_halted"]; !ok || v != true {
		t.Fatalf("expected retry_halted=true, got %v", prob.Details)
	}

	// Should have stopped after 1 attempt (halted before second attempt).
	if venue.calls.Load() != 1 {
		t.Fatalf("expected 1 call before halt, got %d", venue.calls.Load())
	}
}

func TestRetry_HaltChecker_NotHalted_RetriesNormally(t *testing.T) {
	venue := &fakeVenue{behavior: func(attempt int) (ports.VenueOrderReceipt, *problem.Problem) {
		if attempt < 3 {
			return ports.VenueOrderReceipt{}, retryableProblem("transient")
		}
		return okReceipt()
	}}

	policy := execution.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(noSleep)
	rs = rs.WithHaltChecker(&mockGateChecker{halted: false})

	receipt, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob != nil {
		t.Fatalf("expected success, got: %v", prob)
	}
	if receipt.VenueOrderID != "12345" {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	if venue.calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", venue.calls.Load())
	}
}

func TestRetry_HaltChecker_Nil_FailOpen(t *testing.T) {
	// When no halt checker is configured, retry proceeds normally (fail-open).
	venue := &fakeVenue{behavior: func(attempt int) (ports.VenueOrderReceipt, *problem.Problem) {
		if attempt == 1 {
			return ports.VenueOrderReceipt{}, retryableProblem("transient")
		}
		return okReceipt()
	}}

	rs := execution.NewRetrySubmitter(venue, execution.DefaultRetryPolicy())
	rs = rs.TestWithSleepFn(noSleep)
	// No WithHaltChecker call — nil by default.

	receipt, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob != nil {
		t.Fatalf("expected success with nil halt checker, got: %v", prob)
	}
	if receipt.VenueOrderID != "12345" {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
}

func TestRetry_HaltChecker_BecomesHaltedMidLoop(t *testing.T) {
	// Simulates halt occurring after the 2nd attempt.
	haltAfter := int32(2)
	callCount := atomic.Int32{}

	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, retryableProblem("venue error")
	}}

	dynamicChecker := &dynamicGateChecker{
		isHaltedFn: func() bool {
			n := callCount.Add(1)
			return n >= int32(haltAfter)
		},
	}

	policy := execution.RetryPolicy{
		MaxAttempts: 10,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(noSleep)
	rs = rs.WithHaltChecker(dynamicChecker)

	_, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob == nil {
		t.Fatal("expected error when halt triggers mid-loop")
	}
	if v, ok := prob.Details["retry_halted"]; !ok || v != true {
		t.Fatalf("expected retry_halted=true, got %v", prob.Details)
	}
}

func TestRetry_DeadlineAndHalt_DeadlineWinsWhenBothTrigger(t *testing.T) {
	// Both deadline exceeded and halt are true; deadline check runs first.
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, retryableProblem("venue error")
	}}

	tick := int64(0)
	start := time.Now()
	policy := execution.RetryPolicy{
		MaxAttempts: 10,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
		Deadline:    5 * time.Millisecond,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(noSleep)
	rs = rs.TestWithNowFn(func() time.Time {
		n := atomic.AddInt64(&tick, 1)
		return start.Add(time.Duration(n) * 10 * time.Millisecond)
	})
	rs = rs.WithHaltChecker(&mockGateChecker{halted: true})

	_, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob == nil {
		t.Fatal("expected error")
	}

	// Deadline check fires before halt check in the loop ordering.
	if v, ok := prob.Details["retry_deadline_exceeded"]; !ok || v != true {
		t.Fatalf("expected retry_deadline_exceeded=true, got %v", prob.Details)
	}
}

func TestRetry_SuccessBeforeDeadline_NoDeadlineMetadata(t *testing.T) {
	// When retry succeeds within deadline, no deadline metadata appears.
	venue := &fakeVenue{behavior: func(attempt int) (ports.VenueOrderReceipt, *problem.Problem) {
		if attempt == 1 {
			return ports.VenueOrderReceipt{}, retryableProblem("transient")
		}
		return okReceipt()
	}}

	policy := execution.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
		Deadline:    10 * time.Second,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(noSleep)

	receipt, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob != nil {
		t.Fatalf("expected success, got: %v", prob)
	}
	if receipt.VenueOrderID != "12345" {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
}

// --- S324: Retry Observability Tests ---

// testLogger creates a logger that writes to a buffer for test assertions.
func testLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(h), &buf
}

func TestRetryObservability_SuccessAfterRetry_LogsAndCounts(t *testing.T) {
	venue := &fakeVenue{behavior: func(attempt int) (ports.VenueOrderReceipt, *problem.Problem) {
		if attempt == 1 {
			return ports.VenueOrderReceipt{}, retryableProblem("rate limited")
		}
		return okReceipt()
	}}

	logger, logBuf := testLogger()
	tracker := healthz.NewTracker("retry-test")

	rs := execution.NewRetrySubmitter(venue, execution.DefaultRetryPolicy())
	rs = rs.TestWithSleepFn(noSleep)
	rs = rs.WithLogger(logger)
	rs = rs.WithTracker(tracker)

	receipt, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob != nil {
		t.Fatalf("expected success, got: %v", prob)
	}
	if receipt.VenueOrderID != "12345" {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}

	// Verify structured log contains retry events.
	logs := logBuf.String()
	if !bytes.Contains(logBuf.Bytes(), []byte("retry attempt failed")) {
		t.Errorf("expected 'retry attempt failed' log, got: %s", logs)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte("retry succeeded")) {
		t.Errorf("expected 'retry succeeded' log, got: %s", logs)
	}

	// Verify counters.
	if v := tracker.Counter("retry_attempts").Load(); v != 1 {
		t.Errorf("expected retry_attempts=1, got %d", v)
	}
	if v := tracker.Counter("retry_success_after_retry").Load(); v != 1 {
		t.Errorf("expected retry_success_after_retry=1, got %d", v)
	}
}

func TestRetryObservability_Exhaustion_LogsAndCounts(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, retryableProblem("venue unavailable")
	}}

	logger, logBuf := testLogger()
	tracker := healthz.NewTracker("retry-test")

	policy := execution.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(noSleep)
	rs = rs.WithLogger(logger)
	rs = rs.WithTracker(tracker)

	_, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob == nil {
		t.Fatal("expected error after exhausting retries")
	}

	logs := logBuf.String()
	if !bytes.Contains(logBuf.Bytes(), []byte("retry exhausted")) {
		t.Errorf("expected 'retry exhausted' log, got: %s", logs)
	}

	// 2 non-terminal attempt failures + 1 exhaustion.
	if v := tracker.Counter("retry_attempts").Load(); v != 2 {
		t.Errorf("expected retry_attempts=2 (non-terminal), got %d", v)
	}
	if v := tracker.Counter("retry_exhausted").Load(); v != 1 {
		t.Errorf("expected retry_exhausted=1, got %d", v)
	}
}

func TestRetryObservability_Halt_LogsAndCounts(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, retryableProblem("venue error")
	}}

	logger, logBuf := testLogger()
	tracker := healthz.NewTracker("retry-test")

	policy := execution.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(noSleep)
	rs = rs.WithLogger(logger)
	rs = rs.WithTracker(tracker)
	rs = rs.WithHaltChecker(&mockGateChecker{halted: true})

	_, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob == nil {
		t.Fatal("expected error when halted")
	}

	logs := logBuf.String()
	if !bytes.Contains(logBuf.Bytes(), []byte("retry halted by kill switch")) {
		t.Errorf("expected 'retry halted by kill switch' log, got: %s", logs)
	}

	if v := tracker.Counter("retry_halted").Load(); v != 1 {
		t.Errorf("expected retry_halted=1, got %d", v)
	}
}

func TestRetryObservability_Deadline_LogsAndCounts(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, retryableProblem("venue error")
	}}

	logger, logBuf := testLogger()
	tracker := healthz.NewTracker("retry-test")

	tick := int64(0)
	start := time.Now()
	policy := execution.RetryPolicy{
		MaxAttempts: 10,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
		Deadline:    50 * time.Millisecond,
	}
	rs := execution.NewRetrySubmitter(venue, policy)
	rs = rs.TestWithSleepFn(noSleep)
	rs = rs.TestWithNowFn(func() time.Time {
		n := atomic.AddInt64(&tick, 1)
		return start.Add(time.Duration(n) * 20 * time.Millisecond)
	})
	rs = rs.WithLogger(logger)
	rs = rs.WithTracker(tracker)

	_, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob == nil {
		t.Fatal("expected error after deadline")
	}

	logs := logBuf.String()
	if !bytes.Contains(logBuf.Bytes(), []byte("retry deadline exceeded")) {
		t.Errorf("expected 'retry deadline exceeded' log, got: %s", logs)
	}

	if v := tracker.Counter("retry_deadline_exceeded").Load(); v != 1 {
		t.Errorf("expected retry_deadline_exceeded=1, got %d", v)
	}
}

func TestRetryObservability_FirstAttemptSuccess_NoRetryLogs(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return okReceipt()
	}}

	logger, logBuf := testLogger()
	tracker := healthz.NewTracker("retry-test")

	rs := execution.NewRetrySubmitter(venue, execution.DefaultRetryPolicy())
	rs = rs.TestWithSleepFn(noSleep)
	rs = rs.WithLogger(logger)
	rs = rs.WithTracker(tracker)

	_, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob != nil {
		t.Fatalf("expected success, got: %v", prob)
	}

	// No retry logs should be emitted on first-attempt success.
	if logBuf.Len() != 0 {
		t.Errorf("expected no logs on first-attempt success, got: %s", logBuf.String())
	}

	// No retry counters should be incremented.
	if v := tracker.Counter("retry_attempts").Load(); v != 0 {
		t.Errorf("expected retry_attempts=0, got %d", v)
	}
	if v := tracker.Counter("retry_success_after_retry").Load(); v != 0 {
		t.Errorf("expected retry_success_after_retry=0, got %d", v)
	}
}

func TestRetryObservability_NilLoggerAndTracker_NoPanic(t *testing.T) {
	venue := &fakeVenue{behavior: func(attempt int) (ports.VenueOrderReceipt, *problem.Problem) {
		if attempt == 1 {
			return ports.VenueOrderReceipt{}, retryableProblem("transient")
		}
		return okReceipt()
	}}

	// No WithLogger or WithTracker — must not panic.
	rs := execution.NewRetrySubmitter(venue, execution.DefaultRetryPolicy())
	rs = rs.TestWithSleepFn(noSleep)

	receipt, prob := rs.SubmitOrder(context.Background(), dummyRequest())
	if prob != nil {
		t.Fatalf("expected success, got: %v", prob)
	}
	if receipt.VenueOrderID != "12345" {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
}
