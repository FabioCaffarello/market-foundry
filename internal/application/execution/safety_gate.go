package execution

import (
	"context"
	"time"
)

// GateChecker abstracts the kill switch check so the safety gate can be tested
// without a NATS connection. The production implementation is ExecutionControlKVStore.
type GateChecker interface {
	IsHalted(ctx context.Context) bool
}

// SafetyVerdict is the result of a pre-submit safety check.
type SafetyVerdict struct {
	Allowed bool
	Reason  string // non-empty when Allowed is false
}

// SafetyGate encapsulates the pre-submit safety checks that protect the execution
// pipeline. It evaluates three gates in order:
//
//  1. Kill switch — blocks all submissions when the control gate is halted.
//  2. Staleness guard — blocks intents whose timestamp exceeds the max age.
//  3. (Submit timeout is not checked here — it's a context deadline on the RPC call.)
//
// This type exists to make the safety-critical decision path independently testable
// without requiring NATS, Hollywood actors, or other infrastructure.
type SafetyGate struct {
	gateChecker     GateChecker
	gateReadTimeout time.Duration
	staleness       *StalenessGuard
}

// NewSafetyGate creates a safety gate with the given components.
// gateChecker may be nil — kill switch is skipped when unavailable (fail-open).
// gateReadTimeout is the timeout for reading the kill switch state (0 defaults to 2s).
func NewSafetyGate(gateChecker GateChecker, gateReadTimeout time.Duration, staleness *StalenessGuard) *SafetyGate {
	if gateReadTimeout == 0 {
		gateReadTimeout = 2 * time.Second
	}
	return &SafetyGate{
		gateChecker:     gateChecker,
		gateReadTimeout: gateReadTimeout,
		staleness:       staleness,
	}
}

// Check evaluates all pre-submit safety gates and returns a verdict.
// The now parameter allows deterministic testing.
func (g *SafetyGate) Check(intentTimestamp time.Time, now time.Time) SafetyVerdict {
	// Gate 1: Kill switch.
	if g.gateChecker != nil {
		ctx, cancel := context.WithTimeout(context.Background(), g.gateReadTimeout)
		halted := g.gateChecker.IsHalted(ctx)
		cancel()
		if halted {
			return SafetyVerdict{Allowed: false, Reason: "kill_switch"}
		}
	}

	// Gate 2: Staleness guard.
	if g.staleness != nil && g.staleness.IsStale(intentTimestamp, now) {
		return SafetyVerdict{Allowed: false, Reason: "stale"}
	}

	return SafetyVerdict{Allowed: true}
}
