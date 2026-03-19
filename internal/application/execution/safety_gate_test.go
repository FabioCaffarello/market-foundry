package execution_test

import (
	"context"
	"testing"
	"time"

	appexec "internal/application/execution"
)

// ---------- Mock GateChecker ----------

type mockGateChecker struct {
	halted bool
}

func (m *mockGateChecker) IsHalted(_ context.Context) bool {
	return m.halted
}

// slowGateChecker simulates a kill switch read that exceeds the timeout.
type slowGateChecker struct {
	delay  time.Duration
	halted bool
}

func (m *slowGateChecker) IsHalted(ctx context.Context) bool {
	select {
	case <-time.After(m.delay):
		return m.halted
	case <-ctx.Done():
		// Timeout expired — caller treats this as fail-open (not halted).
		return false
	}
}

// ---------- Gate 1: Kill Switch Tests ----------

func TestSafetyGate_KillSwitch_Halted_BlocksSubmission(t *testing.T) {
	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: true},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	verdict := gate.Check(now.Add(-10*time.Second), now)
	if verdict.Allowed {
		t.Fatal("expected blocked when kill switch is halted")
	}
	if verdict.Reason != "kill_switch" {
		t.Fatalf("expected reason kill_switch, got %q", verdict.Reason)
	}
}

func TestSafetyGate_KillSwitch_Active_AllowsSubmission(t *testing.T) {
	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	verdict := gate.Check(now.Add(-10*time.Second), now)
	if !verdict.Allowed {
		t.Fatalf("expected allowed when kill switch is active, got reason: %q", verdict.Reason)
	}
}

func TestSafetyGate_KillSwitch_Nil_FailOpen(t *testing.T) {
	// When gateChecker is nil (KV store unavailable), execution proceeds.
	gate := appexec.NewSafetyGate(
		nil,
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	verdict := gate.Check(now.Add(-10*time.Second), now)
	if !verdict.Allowed {
		t.Fatalf("expected fail-open when gate checker is nil, got reason: %q", verdict.Reason)
	}
}

func TestSafetyGate_KillSwitch_Timeout_FailOpen(t *testing.T) {
	// If the kill switch read takes longer than the timeout, fail-open.
	gate := appexec.NewSafetyGate(
		&slowGateChecker{delay: 5 * time.Second, halted: true},
		50*time.Millisecond, // Very short timeout
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	verdict := gate.Check(now.Add(-10*time.Second), now)
	if !verdict.Allowed {
		t.Fatalf("expected fail-open on kill switch timeout, got reason: %q", verdict.Reason)
	}
}

// ---------- Gate 2: Staleness Guard Tests ----------

func TestSafetyGate_Staleness_StaleIntent_Blocked(t *testing.T) {
	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	verdict := gate.Check(now.Add(-5*time.Minute), now)
	if verdict.Allowed {
		t.Fatal("expected blocked for stale intent")
	}
	if verdict.Reason != "stale" {
		t.Fatalf("expected reason stale, got %q", verdict.Reason)
	}
}

func TestSafetyGate_Staleness_FreshIntent_Allowed(t *testing.T) {
	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	verdict := gate.Check(now.Add(-30*time.Second), now)
	if !verdict.Allowed {
		t.Fatalf("expected allowed for fresh intent, got reason: %q", verdict.Reason)
	}
}

func TestSafetyGate_Staleness_ExactBoundary_Allowed(t *testing.T) {
	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	// At exact boundary, staleness uses > (not >=), so this is NOT stale.
	verdict := gate.Check(now.Add(-2*time.Minute), now)
	if !verdict.Allowed {
		t.Fatalf("expected allowed at exact boundary, got reason: %q", verdict.Reason)
	}
}

func TestSafetyGate_Staleness_NilGuard_SkipsCheck(t *testing.T) {
	// If staleness guard is nil, staleness check is skipped.
	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		2*time.Second,
		nil,
	)
	now := time.Now().UTC()

	// Even very old intents pass if there's no staleness guard.
	verdict := gate.Check(now.Add(-1*time.Hour), now)
	if !verdict.Allowed {
		t.Fatalf("expected allowed when staleness guard is nil, got reason: %q", verdict.Reason)
	}
}

// ---------- Gate Ordering: Kill Switch Takes Priority ----------

func TestSafetyGate_KillSwitchBlocksBeforeStaleness(t *testing.T) {
	// Even with a fresh intent, kill switch blocks first.
	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: true},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	verdict := gate.Check(now.Add(-10*time.Second), now)
	if verdict.Allowed {
		t.Fatal("expected blocked by kill switch even for fresh intent")
	}
	if verdict.Reason != "kill_switch" {
		t.Fatalf("expected kill_switch reason (takes priority), got %q", verdict.Reason)
	}
}

func TestSafetyGate_KillSwitchHalted_StaleIntent_ReportsKillSwitch(t *testing.T) {
	// When both gates would block, kill switch reason is reported (evaluated first).
	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: true},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	verdict := gate.Check(now.Add(-10*time.Minute), now)
	if verdict.Allowed {
		t.Fatal("expected blocked")
	}
	if verdict.Reason != "kill_switch" {
		t.Fatalf("expected kill_switch (priority over stale), got %q", verdict.Reason)
	}
}

// ---------- Combined: Happy Path ----------

func TestSafetyGate_AllGatesPass(t *testing.T) {
	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	verdict := gate.Check(now.Add(-30*time.Second), now)
	if !verdict.Allowed {
		t.Fatalf("expected all gates to pass, got reason: %q", verdict.Reason)
	}
	if verdict.Reason != "" {
		t.Fatalf("expected empty reason on allowed, got %q", verdict.Reason)
	}
}

// ---------- Edge Cases ----------

func TestSafetyGate_FutureTimestamp_Allowed(t *testing.T) {
	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	// Future timestamp: negative age, should not be stale.
	verdict := gate.Check(now.Add(1*time.Minute), now)
	if !verdict.Allowed {
		t.Fatalf("expected allowed for future timestamp, got reason: %q", verdict.Reason)
	}
}

func TestSafetyGate_ZeroTimestamp_Stale(t *testing.T) {
	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		2*time.Second,
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	// Zero timestamp: very old, should be stale.
	verdict := gate.Check(time.Time{}, now)
	if verdict.Allowed {
		t.Fatal("expected blocked for zero timestamp")
	}
	if verdict.Reason != "stale" {
		t.Fatalf("expected reason stale, got %q", verdict.Reason)
	}
}

func TestSafetyGate_DefaultGateReadTimeout(t *testing.T) {
	// When gateReadTimeout is 0, it defaults to 2s.
	gate := appexec.NewSafetyGate(
		&mockGateChecker{halted: false},
		0, // Should default to 2s
		appexec.NewStalenessGuard(2*time.Minute),
	)
	now := time.Now().UTC()

	verdict := gate.Check(now.Add(-10*time.Second), now)
	if !verdict.Allowed {
		t.Fatalf("expected allowed, got reason: %q", verdict.Reason)
	}
}

func TestSafetyGate_BothNil_FullyOpen(t *testing.T) {
	// No gate checker, no staleness guard — everything passes.
	gate := appexec.NewSafetyGate(nil, 0, nil)
	now := time.Now().UTC()

	verdict := gate.Check(now.Add(-24*time.Hour), now)
	if !verdict.Allowed {
		t.Fatalf("expected fully open when both components are nil, got reason: %q", verdict.Reason)
	}
}
