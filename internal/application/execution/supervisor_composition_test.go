package execution_test

// S328 — Supervisor Composition Tests.
//
// These tests verify that the decorator pipeline composes correctly:
//   Post200Reconciler → RetrySubmitter(+hooks) → rawAdapter
//
// They exercise the exact composition order used in VenueAdapterActor.start(),
// proving that the decorators interoperate without breaking invariants.

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/healthz"
	"internal/shared/problem"
)

// ---------------------------------------------------------------------------
// SC-01: Full stack composition — retry + reconciler + hooks
// ---------------------------------------------------------------------------
func TestSC01_FullComposition_RetryThenReconciler(t *testing.T) {
	// Scenario: first attempt → retryable 503, second attempt → body-read-failure-after-200.
	// Expected: RetrySubmitter retries the 503, then body-read-failure passes through
	// to Post200Reconciler which recovers via query.

	var callCount atomic.Int32

	venue := &fakeVenue{behavior: func(n int) (ports.VenueOrderReceipt, *problem.Problem) {
		call := callCount.Add(1)
		switch call {
		case 1:
			// First attempt: retryable failure.
			return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "503 unavailable").MarkRetryable()
		case 2:
			// Second attempt: body-read-failure-after-200.
			return ports.VenueOrderReceipt{}, problem.New(problem.Internal, "read response body failed").
				WithDetail("body_read_failure_after_200", true).
				WithDetail("client_order_id", "test-client-id-001")
		default:
			t.Fatalf("SC-01: unexpected submit call %d", call)
			return ports.VenueOrderReceipt{}, nil
		}
	}}

	queryVenue := &fakeQueryVenue{behavior: func(clientOrderID, symbol string) (ports.VenueOrderReceipt, *problem.Problem) {
		if clientOrderID != "test-client-id-001" {
			t.Fatalf("SC-01: expected client order ID test-client-id-001, got %s", clientOrderID)
		}
		return ports.VenueOrderReceipt{
			VenueOrderID:  "recovered-777",
			ClientOrderID: clientOrderID,
			Status:        domainexec.StatusFilled,
			Intent: domainexec.ExecutionIntent{
				Instrument:     instrumentFromVenueSymbol(t, "binancef", symbol),
				FilledQuantity: "0.001",
				Fills: []domainexec.FillRecord{
					{Price: "65000.00", Quantity: "0.001", Fee: "65.00", Timestamp: time.Now().UTC()},
				},
			},
		}, nil
	}}

	// Compose: Post200Reconciler → RetrySubmitter → fakeVenue
	tracker := healthz.NewTracker("test-composition")
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	retrier := appexec.NewRetrySubmitter(venue, appexec.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}).
		WithLogger(logger).
		WithTracker(tracker).
		TestWithSleepFn(noSleep)

	reconciler := appexec.NewPost200Reconciler(retrier, queryVenue, 5*time.Second)

	receipt, prob := reconciler.SubmitOrder(context.Background(), ports.VenueOrderRequest{
		Intent: testBuyIntent(t),
	})
	if prob != nil {
		t.Fatalf("SC-01: expected recovery, got error: %s (details: %v)", prob.Message, prob.Details)
	}

	if receipt.VenueOrderID != "recovered-777" {
		t.Fatalf("SC-01: expected venue order ID recovered-777, got %s", receipt.VenueOrderID)
	}
	if receipt.Status != domainexec.StatusFilled {
		t.Fatalf("SC-01: expected status filled, got %s", receipt.Status)
	}

	// Verify observability: retry counter should have incremented.
	retryAttempts := tracker.Counter("retry_attempts").Load()
	if retryAttempts != 1 {
		t.Fatalf("SC-01: expected 1 retry_attempts counter, got %d", retryAttempts)
	}

	// Verify structured log contains retry attempt.
	if !bytes.Contains(logBuf.Bytes(), []byte("retry attempt failed")) {
		t.Fatal("SC-01: expected retry log entry in structured output")
	}
}

// ---------------------------------------------------------------------------
// SC-02: Full stack — success on first attempt, no retry/reconciliation
// ---------------------------------------------------------------------------
func TestSC02_FullComposition_SuccessOnFirstAttempt(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{
			VenueOrderID: "direct-success-123",
			Status:       domainexec.StatusFilled,
			Intent:       testBuyIntent(t),
		}, nil
	}}
	queryVenue := &fakeQueryVenue{behavior: func(_, _ string) (ports.VenueOrderReceipt, *problem.Problem) {
		t.Fatal("SC-02: query must not be called on direct success")
		return ports.VenueOrderReceipt{}, nil
	}}

	retrier := appexec.NewRetrySubmitter(venue, appexec.DefaultRetryPolicy()).
		TestWithSleepFn(noSleep)
	reconciler := appexec.NewPost200Reconciler(retrier, queryVenue, 5*time.Second)

	receipt, prob := reconciler.SubmitOrder(context.Background(), ports.VenueOrderRequest{
		Intent: testBuyIntent(t),
	})
	if prob != nil {
		t.Fatalf("SC-02: expected success, got: %s", prob.Message)
	}
	if receipt.VenueOrderID != "direct-success-123" {
		t.Fatalf("SC-02: expected order ID direct-success-123, got %s", receipt.VenueOrderID)
	}
}

// ---------------------------------------------------------------------------
// SC-03: Full stack — non-retryable error passes through both decorators
// ---------------------------------------------------------------------------
func TestSC03_FullComposition_NonRetryablePassthrough(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, problem.New(problem.InvalidArgument, "bad symbol")
	}}
	queryVenue := &fakeQueryVenue{behavior: func(_, _ string) (ports.VenueOrderReceipt, *problem.Problem) {
		t.Fatal("SC-03: query must not be called for non-body-read errors")
		return ports.VenueOrderReceipt{}, nil
	}}

	retrier := appexec.NewRetrySubmitter(venue, appexec.DefaultRetryPolicy()).
		TestWithSleepFn(noSleep)
	reconciler := appexec.NewPost200Reconciler(retrier, queryVenue, 5*time.Second)

	_, prob := reconciler.SubmitOrder(context.Background(), ports.VenueOrderRequest{
		Intent: testBuyIntent(t),
	})
	if prob == nil {
		t.Fatal("SC-03: expected error")
	}
	if prob.Code != problem.InvalidArgument {
		t.Fatalf("SC-03: expected InvalidArgument, got %s", prob.Code)
	}
}

// ---------------------------------------------------------------------------
// SC-04: Full stack — halt checker aborts retry loop
// ---------------------------------------------------------------------------
func TestSC04_FullComposition_HaltCheckerAborts(t *testing.T) {
	var calls atomic.Int32
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		calls.Add(1)
		return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "unavailable").MarkRetryable()
	}}
	queryVenue := &fakeQueryVenue{behavior: func(_, _ string) (ports.VenueOrderReceipt, *problem.Problem) {
		t.Fatal("SC-04: query must not be called when halted")
		return ports.VenueOrderReceipt{}, nil
	}}

	haltChecker := &mockGateChecker{halted: true}
	tracker := healthz.NewTracker("test-halt")

	retrier := appexec.NewRetrySubmitter(venue, appexec.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}).
		WithHaltChecker(haltChecker).
		WithTracker(tracker).
		TestWithSleepFn(noSleep)
	reconciler := appexec.NewPost200Reconciler(retrier, queryVenue, 5*time.Second)

	_, prob := reconciler.SubmitOrder(context.Background(), ports.VenueOrderRequest{
		Intent: testBuyIntent(t),
	})
	if prob == nil {
		t.Fatal("SC-04: expected error when halt checker fires")
	}

	// Should have halted after first attempt.
	if calls.Load() != 1 {
		t.Fatalf("SC-04: expected exactly 1 attempt before halt, got %d", calls.Load())
	}

	// Verify halt metadata.
	if v, ok := prob.Details["retry_halted"]; !ok || v != true {
		t.Fatal("SC-04: expected retry_halted=true in details")
	}

	// Verify counter.
	if tracker.Counter("retry_halted").Load() != 1 {
		t.Fatalf("SC-04: expected retry_halted counter=1, got %d", tracker.Counter("retry_halted").Load())
	}
}

// ---------------------------------------------------------------------------
// SC-05: Composition without query port — no reconciler, retry only
// ---------------------------------------------------------------------------
func TestSC05_CompositionWithoutQueryPort_RetryOnly(t *testing.T) {
	// This simulates the paper adapter case where VenueQuery is nil.
	var calls atomic.Int32
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		n := calls.Add(1)
		if n < 3 {
			return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "transient").MarkRetryable()
		}
		return ports.VenueOrderReceipt{
			VenueOrderID: "retry-success-456",
			Status:       domainexec.StatusFilled,
			Intent:       testBuyIntent(t),
		}, nil
	}}

	retrier := appexec.NewRetrySubmitter(venue, appexec.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}).TestWithSleepFn(noSleep)

	// No reconciler — just the retrier as the composed venue.
	receipt, prob := retrier.SubmitOrder(context.Background(), ports.VenueOrderRequest{
		Intent: testBuyIntent(t),
	})
	if prob != nil {
		t.Fatalf("SC-05: expected success after retries, got: %s", prob.Message)
	}
	if receipt.VenueOrderID != "retry-success-456" {
		t.Fatalf("SC-05: expected order ID retry-success-456, got %s", receipt.VenueOrderID)
	}
	if calls.Load() != 3 {
		t.Fatalf("SC-05: expected 3 total attempts, got %d", calls.Load())
	}
}

// ---------------------------------------------------------------------------
// SC-06: Decorator order verification — retry metadata surfaces through reconciler
// ---------------------------------------------------------------------------
func TestSC06_RetryMetadataSurfaces_ThroughReconciler(t *testing.T) {
	// All retries exhaust → non-body-read error → passes through reconciler unchanged.
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "always fails").MarkRetryable()
	}}
	queryVenue := &fakeQueryVenue{behavior: func(_, _ string) (ports.VenueOrderReceipt, *problem.Problem) {
		t.Fatal("SC-06: query must not be called for exhausted retries")
		return ports.VenueOrderReceipt{}, nil
	}}

	retrier := appexec.NewRetrySubmitter(venue, appexec.RetryPolicy{
		MaxAttempts: 2,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}).TestWithSleepFn(noSleep)
	reconciler := appexec.NewPost200Reconciler(retrier, queryVenue, 5*time.Second)

	_, prob := reconciler.SubmitOrder(context.Background(), ports.VenueOrderRequest{
		Intent: testBuyIntent(t),
	})
	if prob == nil {
		t.Fatal("SC-06: expected error after retry exhaustion")
	}

	// Retry metadata must be present (surfaced through the reconciler).
	if v, ok := prob.Details["retry_attempts"]; !ok {
		t.Fatal("SC-06: expected retry_attempts in details")
	} else if v.(int) != 2 {
		t.Fatalf("SC-06: expected 2 retry_attempts, got %v", v)
	}
	if v, ok := prob.Details["retry_exhausted"]; !ok || v != true {
		t.Fatal("SC-06: expected retry_exhausted=true in details")
	}
}

// ---------------------------------------------------------------------------
// SC-07: Log observability — structured log contains component tag
// ---------------------------------------------------------------------------
func TestSC07_StructuredLog_ContainsComponentTag(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "fail").MarkRetryable()
	}}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil)).With("component", "retry-submitter")

	retrier := appexec.NewRetrySubmitter(venue, appexec.RetryPolicy{
		MaxAttempts: 2,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Factor:      2.0,
	}).
		WithLogger(logger).
		TestWithSleepFn(noSleep)

	retrier.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})

	// Parse log lines and verify component tag.
	for _, line := range bytes.Split(logBuf.Bytes(), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if comp, ok := entry["component"]; ok {
			if comp != "retry-submitter" {
				t.Fatalf("SC-07: expected component=retry-submitter, got %v", comp)
			}
			return // found at least one log line with correct component
		}
	}
	t.Fatal("SC-07: no log lines found with component=retry-submitter")
}
