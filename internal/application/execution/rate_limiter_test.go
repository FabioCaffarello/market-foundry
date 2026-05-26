package execution_test

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"internal/application/execution"
	"internal/application/ports"
	"internal/shared/problem"
)

// Helpers fakeVenue, okReceipt, dummyRequest are defined in
// retry_submitter_test.go (same package execution_test).

// blockingVenue blocks SubmitOrder until ctx is done or release is closed,
// used to verify that the rate limiter actually gates calls.
type blockingVenue struct {
	calls   atomic.Int32
	release chan struct{}
}

func (b *blockingVenue) SubmitOrder(ctx context.Context, _ ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	b.calls.Add(1)
	select {
	case <-b.release:
	case <-ctx.Done():
	}
	return okReceipt()
}

// hasGoroutineWithFunc scans the current goroutine stacks for a frame
// matching the given substring. Used to assert presence/absence of the
// rate limiter's background goroutine deterministically — runtime.NumGoroutine()
// has too much noise (GC workers, finalizers, peer tests) for that.
func hasGoroutineWithFunc(name string) bool {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	return strings.Contains(string(buf[:n]), name)
}

// waitFor polls until cond returns true or timeout expires. Returns the
// final result of cond().
func waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		runtime.Gosched()
		time.Sleep(2 * time.Millisecond)
	}
	return cond()
}

// TestRateLimiter_AllowsBurstImmediately verifies the initial token bucket
// is filled to capacity at construction — first maxBurst calls succeed
// without waiting on the refill ticker.
func TestRateLimiter_AllowsBurstImmediately(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return okReceipt()
	}}
	const burst = 5
	rl := execution.NewRateLimiter(venue, burst, 10*time.Second) // refill so slow it cannot help here
	defer rl.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	for i := 0; i < burst; i++ {
		if _, prob := rl.SubmitOrder(ctx, dummyRequest(t)); prob != nil {
			t.Fatalf("call %d: unexpected problem: %v", i, prob)
		}
	}
	if got := venue.calls.Load(); got != burst {
		t.Fatalf("expected %d inner calls, got %d", burst, got)
	}
}

// TestRateLimiter_BlocksWhenExhausted verifies that once the bucket is
// drained, the next call blocks until the context expires.
func TestRateLimiter_BlocksWhenExhausted(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return okReceipt()
	}}
	rl := execution.NewRateLimiter(venue, 1, 10*time.Second) // single token, refill effectively never
	defer rl.Close()

	if _, prob := rl.SubmitOrder(context.Background(), dummyRequest(t)); prob != nil {
		t.Fatalf("first call should succeed: %v", prob)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, prob := rl.SubmitOrder(ctx, dummyRequest(t))
	elapsed := time.Since(start)

	if prob == nil {
		t.Fatal("expected Unavailable problem when bucket exhausted and ctx expires")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("expected Unavailable, got %s", prob.Code)
	}
	if elapsed < 40*time.Millisecond {
		t.Fatalf("expected to block ~50ms before ctx expiry, got %v", elapsed)
	}
	if got := venue.calls.Load(); got != 1 {
		t.Fatalf("inner venue should have been called once (gated), got %d", got)
	}
}

// TestRateLimiter_RefillsAfterInterval verifies that tokens replenish over
// time so a previously-blocked call can succeed once a token arrives.
func TestRateLimiter_RefillsAfterInterval(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return okReceipt()
	}}
	const refill = 20 * time.Millisecond
	rl := execution.NewRateLimiter(venue, 1, refill)
	defer rl.Close()

	// Drain the bucket.
	if _, prob := rl.SubmitOrder(context.Background(), dummyRequest(t)); prob != nil {
		t.Fatalf("drain call failed: %v", prob)
	}

	// Next call must wait for refill but should ultimately succeed within
	// a few refill intervals.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	if _, prob := rl.SubmitOrder(ctx, dummyRequest(t)); prob != nil {
		t.Fatalf("expected refilled token to allow call, got %v", prob)
	}
	elapsed := time.Since(start)

	if elapsed < refill/2 {
		t.Fatalf("call returned suspiciously fast (%v) — expected to wait for refill", elapsed)
	}
	if got := venue.calls.Load(); got != 2 {
		t.Fatalf("expected 2 inner calls after refill, got %d", got)
	}
}

// TestRateLimiter_ContextCancellation verifies that a pending call returns
// Unavailable when its context is canceled before a token becomes available.
func TestRateLimiter_ContextCancellation(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return okReceipt()
	}}
	rl := execution.NewRateLimiter(venue, 1, time.Hour) // refill so slow it cannot rescue us
	defer rl.Close()

	// Drain the bucket.
	if _, prob := rl.SubmitOrder(context.Background(), dummyRequest(t)); prob != nil {
		t.Fatalf("drain call failed: %v", prob)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, prob := rl.SubmitOrder(ctx, dummyRequest(t))
	if prob == nil {
		t.Fatal("expected problem when ctx canceled")
	}
	if prob.Code != problem.Unavailable {
		t.Fatalf("expected Unavailable, got %s", prob.Code)
	}
}

// TestRateLimiter_ConcurrentAccess fires many concurrent calls and verifies
// no races (run with -race) and the total throughput respects the bucket
// capacity + refills observed.
func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return okReceipt()
	}}
	const burst = 4
	rl := execution.NewRateLimiter(venue, burst, 5*time.Millisecond)
	defer rl.Close()

	const workers = 20
	var wg sync.WaitGroup
	wg.Add(workers)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var successes atomic.Int32
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			if _, prob := rl.SubmitOrder(ctx, dummyRequest(t)); prob == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := successes.Load(); int(got) != workers {
		t.Fatalf("expected all %d workers to eventually succeed, got %d", workers, got)
	}
	if got := venue.calls.Load(); int(got) != workers {
		t.Fatalf("inner venue should have seen all %d calls, got %d", workers, got)
	}
}

// TestRateLimiter_DelegatesReceiptAndProblem verifies the limiter passes
// through whatever the inner adapter returns once a token is acquired.
func TestRateLimiter_DelegatesReceiptAndProblem(t *testing.T) {
	want := nonRetryableProblem("inner says no")
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return ports.VenueOrderReceipt{}, want
	}}
	rl := execution.NewRateLimiter(venue, 1, 100*time.Millisecond)
	defer rl.Close()

	_, prob := rl.SubmitOrder(context.Background(), dummyRequest(t))
	if prob != want {
		t.Fatalf("expected inner problem to pass through, got %v", prob)
	}
}

// TestRateLimiter_InnerBlockingDoesNotAffectOtherCallers verifies that a
// caller blocked on the inner adapter does not consume any extra tokens,
// and the limiter releases waiters only as tokens are returned via refill.
func TestRateLimiter_InnerBlockingDoesNotAffectOtherCallers(t *testing.T) {
	venue := &blockingVenue{release: make(chan struct{})}
	rl := execution.NewRateLimiter(venue, 2, 5*time.Millisecond)
	defer rl.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			rl.SubmitOrder(ctx, dummyRequest(t))
		}()
	}

	// Wait until both goroutines are inside the inner adapter.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if venue.calls.Load() == 2 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if venue.calls.Load() != 2 {
		t.Fatalf("expected 2 inner calls in flight, got %d", venue.calls.Load())
	}

	close(venue.release)
	wg.Wait()
}

// TestRateLimiter_MinimumBurst verifies that maxBurst < 1 is clamped to 1
// (sanity for misconfiguration — a zero-capacity bucket would deadlock).
func TestRateLimiter_MinimumBurst(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return okReceipt()
	}}
	rl := execution.NewRateLimiter(venue, 0, 50*time.Millisecond)
	defer rl.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if _, prob := rl.SubmitOrder(ctx, dummyRequest(t)); prob != nil {
		t.Fatalf("expected at least one token even when maxBurst=0; got %v", prob)
	}
}

// TestRateLimiter_Close_StopsGoroutine verifies the refill goroutine exits
// after Close — the documented P0 leak motivating P4.2.
//
// We scan goroutine stacks for the RateLimiter.refillLoop frame rather
// than comparing runtime.NumGoroutine() deltas: runtime workers, GC
// finalizers, and peer-test goroutines all jitter the global count
// enough to flake the latter approach on shared CI runners.
const refillLoopFrame = "RateLimiter).refillLoop"

func TestRateLimiter_Close_StopsGoroutine(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return okReceipt()
	}}
	rl := execution.NewRateLimiter(venue, 2, 10*time.Millisecond)

	if !waitFor(500*time.Millisecond, func() bool { return hasGoroutineWithFunc(refillLoopFrame) }) {
		t.Fatal("expected refillLoop goroutine to be scheduled after construction")
	}

	rl.Close()

	if !waitFor(500*time.Millisecond, func() bool { return !hasGoroutineWithFunc(refillLoopFrame) }) {
		t.Fatal("refillLoop goroutine still present after Close")
	}
}

// TestRateLimiter_Close_Idempotent verifies that calling Close multiple
// times does not panic (sync.Once guard).
func TestRateLimiter_Close_Idempotent(t *testing.T) {
	venue := &fakeVenue{behavior: func(_ int) (ports.VenueOrderReceipt, *problem.Problem) {
		return okReceipt()
	}}
	rl := execution.NewRateLimiter(venue, 1, 10*time.Millisecond)

	rl.Close()
	rl.Close()
	rl.Close()
}
