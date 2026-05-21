package execution

import (
	"context"
	"sync"
	"time"

	"internal/application/ports"
	"internal/shared/problem"
)

// RateLimiter is a VenuePort decorator that enforces a token-bucket rate limit
// on venue API calls. It sits between the real adapter and higher-level decorators
// (DryRunSubmitter, RetrySubmitter) in the pipeline.
//
// S433: Introduced for mainnet adapters where Binance enforces strict rate limits.
// Testnet adapters can also use it, but the primary motivation is mainnet safety.
//
// The limiter uses a simple token-bucket algorithm:
//   - Bucket capacity = maxBurst (maximum concurrent requests allowed)
//   - Refill rate = 1 token per refillInterval
//   - If no token is available, the call blocks until one is or the context expires
//
// Pipeline position (innermost -> outermost):
//
//	rawAdapter -> RateLimiter -> RetrySubmitter -> Post200Reconciler -> DryRunSubmitter
type RateLimiter struct {
	inner          ports.VenuePort
	tokens         chan struct{}
	refillInterval time.Duration
	done           chan struct{}
	once           sync.Once
}

// NewRateLimiter creates a rate-limiting decorator around a venue adapter.
// maxBurst is the token bucket capacity (e.g., 10 for 10 concurrent requests).
// refillInterval is the time between token refills (e.g., 100ms for ~10 req/s).
func NewRateLimiter(inner ports.VenuePort, maxBurst int, refillInterval time.Duration) *RateLimiter {
	if maxBurst < 1 {
		maxBurst = 1
	}
	tokens := make(chan struct{}, maxBurst)
	for range maxBurst {
		tokens <- struct{}{}
	}
	rl := &RateLimiter{
		inner:          inner,
		tokens:         tokens,
		refillInterval: refillInterval,
		done:           make(chan struct{}),
	}
	go rl.refillLoop()
	return rl
}

// SubmitOrder acquires a rate limit token before delegating to the inner adapter.
// If the context expires before a token is available, returns Unavailable.
func (rl *RateLimiter) SubmitOrder(ctx context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	select {
	case <-rl.tokens:
		// Token acquired — proceed.
	case <-ctx.Done():
		return ports.VenueOrderReceipt{}, problem.New(
			problem.Unavailable,
			"rate limiter: context expired waiting for token",
		)
	}
	return rl.inner.SubmitOrder(ctx, req)
}

func (rl *RateLimiter) refillLoop() {
	ticker := time.NewTicker(rl.refillInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Bucket full — discard.
			}
		case <-rl.done:
			return
		}
	}
}

// Close stops the refill goroutine. Safe to call multiple times.
func (rl *RateLimiter) Close() {
	rl.once.Do(func() { close(rl.done) })
}
