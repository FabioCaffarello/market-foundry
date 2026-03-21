package execution

import (
	"context"
	"math/rand"
	"time"

	"internal/application/ports"
	"internal/shared/problem"
)

// RetryPolicy defines the bounds for venue submission retries.
// All fields have safe defaults via DefaultRetryPolicy().
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts (1 = no retry).
	MaxAttempts int
	// BaseDelay is the initial backoff duration before the first retry.
	BaseDelay time.Duration
	// MaxDelay caps the exponential backoff.
	MaxDelay time.Duration
	// Factor is the exponential multiplier applied after each attempt.
	Factor float64
}

// DefaultRetryPolicy returns the production retry policy for venue submissions.
// Conservative: 3 attempts, 100ms base delay, 2x exponential, 2s cap.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    2 * time.Second,
		Factor:      2.0,
	}
}

// RetrySubmitter wraps a VenuePort and retries retryable failures according to
// the configured policy. It preserves idempotency because client order IDs are
// derived deterministically from the ExecutionIntent — retries send the same ID.
//
// Abort semantics:
//   - Non-retryable errors are returned immediately.
//   - Context cancellation/deadline aborts the loop.
//   - MaxAttempts exhausted returns the last error with retry metadata.
//
// Observability: when retries are exhausted the returned Problem carries
// "retry_attempts" and "retry_exhausted" in Details.
type RetrySubmitter struct {
	inner  ports.VenuePort
	policy RetryPolicy
	// sleepFn is injectable for testing; defaults to time.Sleep.
	sleepFn func(time.Duration)
}

// NewRetrySubmitter creates a retry-aware venue submitter.
// inner is the underlying VenuePort (e.g. BinanceFuturesTestnetAdapter).
// policy controls retry bounds. Use DefaultRetryPolicy() for production.
func NewRetrySubmitter(inner ports.VenuePort, policy RetryPolicy) *RetrySubmitter {
	if policy.MaxAttempts < 1 {
		policy.MaxAttempts = 1
	}
	return &RetrySubmitter{
		inner:   inner,
		policy:  policy,
		sleepFn: time.Sleep,
	}
}

// TestWithSleepFn overrides the sleep function for testing.
// Production callers should not use this.
func (r *RetrySubmitter) TestWithSleepFn(fn func(time.Duration)) *RetrySubmitter {
	r.sleepFn = fn
	return r
}

// SubmitOrder implements ports.VenuePort. It delegates to the inner port and
// retries on retryable failures up to MaxAttempts, respecting backoff and
// context boundaries.
func (r *RetrySubmitter) SubmitOrder(ctx context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	var lastProblem *problem.Problem
	delay := r.policy.BaseDelay

	for attempt := 1; attempt <= r.policy.MaxAttempts; attempt++ {
		// Check context before each attempt (including the first).
		if err := ctx.Err(); err != nil {
			if lastProblem != nil {
				return ports.VenueOrderReceipt{}, r.annotate(lastProblem, attempt-1)
			}
			return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Unavailable, "context cancelled before venue submit")
		}

		receipt, prob := r.inner.SubmitOrder(ctx, req)
		if prob == nil {
			return receipt, nil
		}

		// Non-retryable errors abort immediately.
		if !prob.Retryable {
			return ports.VenueOrderReceipt{}, prob
		}

		lastProblem = prob

		// If this was the last allowed attempt, don't sleep.
		if attempt == r.policy.MaxAttempts {
			break
		}

		// Backoff with jitter before next attempt.
		jittered := jitterDelay(delay)
		select {
		case <-ctx.Done():
			return ports.VenueOrderReceipt{}, r.annotate(lastProblem, attempt)
		default:
			r.sleepFn(jittered)
		}

		// Exponential increase, capped at MaxDelay.
		delay = time.Duration(float64(delay) * r.policy.Factor)
		if delay > r.policy.MaxDelay {
			delay = r.policy.MaxDelay
		}
	}

	return ports.VenueOrderReceipt{}, r.annotate(lastProblem, r.policy.MaxAttempts)
}

// annotate enriches the final problem with retry metadata for observability.
func (r *RetrySubmitter) annotate(prob *problem.Problem, attempts int) *problem.Problem {
	if prob == nil {
		return nil
	}
	return prob.
		WithDetail("retry_attempts", attempts).
		WithDetail("retry_exhausted", true)
}

// jitterDelay adds ±25% uniform jitter to avoid thundering herd.
func jitterDelay(base time.Duration) time.Duration {
	jitter := 0.75 + rand.Float64()*0.5 // [0.75, 1.25)
	return time.Duration(float64(base) * jitter)
}
