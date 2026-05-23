package execution

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"internal/application/ports"
	"internal/shared/healthz"
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
	// Deadline is the global retry budget. If non-zero, a deadline is applied to
	// the entire retry sequence: the loop aborts when the budget is exceeded,
	// regardless of remaining attempts. Zero means no global deadline (only
	// per-attempt context and MaxAttempts govern).
	Deadline time.Duration
}

// DefaultRetryPolicy returns the production retry policy for venue submissions.
// Conservative: 3 attempts, 100ms base delay, 2x exponential, 2s cap, 10s deadline.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    2 * time.Second,
		Factor:      2.0,
		Deadline:    10 * time.Second,
	}
}

// RetrySubmitter wraps a VenuePort and retries retryable failures according to
// the configured policy. It preserves idempotency because client order IDs are
// derived deterministically from the ExecutionIntent — retries send the same ID.
//
// Abort semantics:
//   - Non-retryable errors are returned immediately.
//   - Context cancellation/deadline aborts the loop.
//   - Global deadline (Deadline field) aborts the loop.
//   - Kill switch (GateChecker) halts the loop between attempts.
//   - MaxAttempts exhausted returns the last error with retry metadata.
//
// Observability: the returned Problem carries structured metadata:
//   - "retry_attempts": number of attempts made
//   - "retry_exhausted": true when attempts are exhausted
//   - "retry_halted": true when kill switch aborted the loop
//   - "retry_deadline_exceeded": true when global deadline was exceeded
type RetrySubmitter struct {
	inner  ports.VenuePort
	policy RetryPolicy
	// haltChecker is checked between retry attempts; nil means no halt check.
	haltChecker GateChecker
	// logger emits structured retry events; nil means no logging.
	logger *slog.Logger
	// tracker increments retry counters; nil means no counter tracking.
	tracker *healthz.Tracker
	// sleepFn is injectable for testing; defaults to time.Sleep.
	sleepFn func(time.Duration)
	// nowFn is injectable for testing; defaults to time.Now.
	nowFn func() time.Time
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
		nowFn:   time.Now,
	}
}

// WithHaltChecker attaches a kill switch checker that is evaluated between
// retry attempts. If the checker reports halted, the retry loop aborts
// immediately with retry_halted metadata. nil disables the check.
func (r *RetrySubmitter) WithHaltChecker(hc GateChecker) *RetrySubmitter {
	r.haltChecker = hc
	return r
}

// WithLogger attaches a structured logger for retry observability.
// When set, the submitter emits structured log events for retry attempts,
// success-after-retry, exhaustion, halt, and deadline abort.
// nil disables logging (default).
func (r *RetrySubmitter) WithLogger(l *slog.Logger) *RetrySubmitter {
	r.logger = l
	return r
}

// WithTracker attaches a health tracker for retry counter metrics.
// When set, the submitter increments counters: retry_attempts,
// retry_success_after_retry, retry_exhausted, retry_halted,
// retry_deadline_exceeded. nil disables counter tracking (default).
func (r *RetrySubmitter) WithTracker(t *healthz.Tracker) *RetrySubmitter {
	r.tracker = t
	return r
}

// TestWithSleepFn overrides the sleep function for testing.
// Production callers should not use this.
func (r *RetrySubmitter) TestWithSleepFn(fn func(time.Duration)) *RetrySubmitter {
	r.sleepFn = fn
	return r
}

// TestWithNowFn overrides the clock function for testing.
// Production callers should not use this.
func (r *RetrySubmitter) TestWithNowFn(fn func() time.Time) *RetrySubmitter {
	r.nowFn = fn
	return r
}

// SubmitOrder implements ports.VenuePort. It delegates to the inner port and
// retries on retryable failures up to MaxAttempts, respecting backoff,
// context boundaries, global deadline, and kill switch state.
func (r *RetrySubmitter) SubmitOrder(ctx context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	var lastProblem *problem.Problem
	delay := r.policy.BaseDelay

	// Compute absolute deadline from the global budget.
	var deadlineAt time.Time
	if r.policy.Deadline > 0 {
		deadlineAt = r.nowFn().Add(r.policy.Deadline)
	}

	for attempt := 1; attempt <= r.policy.MaxAttempts; attempt++ {
		// Check context before each attempt (including the first).
		if err := ctx.Err(); err != nil {
			if lastProblem != nil {
				return ports.VenueOrderReceipt{}, r.annotate(lastProblem, attempt-1)
			}
			return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Unavailable, "context cancelled before venue submit")
		}

		// Check global deadline before each attempt.
		if !deadlineAt.IsZero() && r.nowFn().After(deadlineAt) {
			r.logWarn("retry deadline exceeded", "attempts", attempt-1)
			r.incCounter("retry_deadline_exceeded")
			if lastProblem != nil {
				return ports.VenueOrderReceipt{}, r.annotateDeadline(lastProblem, attempt-1)
			}
			return ports.VenueOrderReceipt{}, problem.New(problem.Unavailable, "retry deadline exceeded before venue submit").
				WithDetail("retry_deadline_exceeded", true)
		}

		receipt, prob := r.inner.SubmitOrder(ctx, req)
		if prob == nil {
			if attempt > 1 {
				r.logInfo("retry succeeded", "attempts", attempt)
				r.incCounter("retry_success_after_retry")
			}
			return receipt, nil
		}

		// Non-retryable errors abort immediately.
		if !prob.Retryable {
			return ports.VenueOrderReceipt{}, prob
		}

		lastProblem = prob

		// If this was the last allowed attempt, don't sleep.
		if attempt == r.policy.MaxAttempts {
			r.logWarn("retry exhausted",
				"attempts", attempt,
				"max_attempts", r.policy.MaxAttempts,
				"last_error", prob.Message,
			)
			r.incCounter("retry_exhausted")
			break
		}

		// Log the non-terminal retry failure.
		r.logWarn("retry attempt failed",
			"attempt", attempt,
			"max_attempts", r.policy.MaxAttempts,
			"error", prob.Message,
		)
		r.incCounter("retry_attempts")

		// Check kill switch between attempts — abort if halted.
		if r.isHalted(ctx) {
			r.logWarn("retry halted by kill switch", "attempts", attempt)
			r.incCounter("retry_halted")
			return ports.VenueOrderReceipt{}, r.annotateHalted(lastProblem, attempt)
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

// isHalted checks the kill switch. Returns false if no checker is configured
// (fail-open) or if the check times out. The caller's ctx is honored — the
// halt probe is bounded by min(2s, caller deadline) so a halted retry loop
// cannot outrun the outer SubmitOrder deadline.
func (r *RetrySubmitter) isHalted(ctx context.Context) bool {
	if r.haltChecker == nil {
		return false
	}
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return r.haltChecker.IsHalted(probeCtx)
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

// annotateHalted enriches the problem when the kill switch aborted the retry loop.
func (r *RetrySubmitter) annotateHalted(prob *problem.Problem, attempts int) *problem.Problem {
	if prob == nil {
		return nil
	}
	return prob.
		WithDetail("retry_attempts", attempts).
		WithDetail("retry_halted", true)
}

// annotateDeadline enriches the problem when the global deadline was exceeded.
func (r *RetrySubmitter) annotateDeadline(prob *problem.Problem, attempts int) *problem.Problem {
	if prob == nil {
		return nil
	}
	return prob.
		WithDetail("retry_attempts", attempts).
		WithDetail("retry_deadline_exceeded", true)
}

// logInfo emits a structured info log if a logger is configured.
func (r *RetrySubmitter) logInfo(msg string, args ...any) {
	if r.logger != nil {
		r.logger.Info(msg, args...)
	}
}

// logWarn emits a structured warn log if a logger is configured.
func (r *RetrySubmitter) logWarn(msg string, args ...any) {
	if r.logger != nil {
		r.logger.Warn(msg, args...)
	}
}

// incCounter increments a named counter if a tracker is configured.
func (r *RetrySubmitter) incCounter(name string) {
	if r.tracker != nil {
		r.tracker.Counter(name).Add(1)
	}
}

// jitterDelay adds ±25% uniform jitter to avoid thundering herd.
func jitterDelay(base time.Duration) time.Duration {
	jitter := 0.75 + rand.Float64()*0.5 // [0.75, 1.25)
	return time.Duration(float64(base) * jitter)
}
