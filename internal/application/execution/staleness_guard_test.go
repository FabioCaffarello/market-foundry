package execution_test

import (
	"testing"
	"time"

	appexec "internal/application/execution"
)

func TestStalenessGuard_Fresh(t *testing.T) {
	guard := appexec.NewStalenessGuard(2 * time.Minute)
	now := time.Now().UTC()
	intentTS := now.Add(-30 * time.Second)

	if guard.IsStale(intentTS, now) {
		t.Error("expected fresh intent to not be stale")
	}
}

func TestStalenessGuard_Stale(t *testing.T) {
	guard := appexec.NewStalenessGuard(2 * time.Minute)
	now := time.Now().UTC()
	intentTS := now.Add(-3 * time.Minute)

	if !guard.IsStale(intentTS, now) {
		t.Error("expected old intent to be stale")
	}
}

func TestStalenessGuard_ExactBoundary(t *testing.T) {
	guard := appexec.NewStalenessGuard(2 * time.Minute)
	now := time.Now().UTC()
	intentTS := now.Add(-2 * time.Minute)

	// At exact boundary (age == maxAge), not stale (> not >=).
	if guard.IsStale(intentTS, now) {
		t.Error("expected exact boundary to not be stale")
	}
}

func TestStalenessGuard_FutureTimestamp(t *testing.T) {
	guard := appexec.NewStalenessGuard(2 * time.Minute)
	now := time.Now().UTC()
	intentTS := now.Add(1 * time.Minute)

	if guard.IsStale(intentTS, now) {
		t.Error("expected future timestamp to not be stale")
	}
}

func TestStalenessGuard_ZeroMaxAge_EverythingStale(t *testing.T) {
	guard := appexec.NewStalenessGuard(0)
	now := time.Now().UTC()

	// Even 1ns-old intents are stale when maxAge is 0.
	intentTS := now.Add(-1 * time.Nanosecond)
	if !guard.IsStale(intentTS, now) {
		t.Error("expected any past intent to be stale with zero maxAge")
	}

	// At exact now, age==0 which is not > 0, so NOT stale.
	if guard.IsStale(now, now) {
		t.Error("expected exact-now timestamp to not be stale even with zero maxAge")
	}
}

func TestStalenessGuard_ZeroTimestamp(t *testing.T) {
	guard := appexec.NewStalenessGuard(2 * time.Minute)
	now := time.Now().UTC()

	// Zero-value timestamp is extremely old — should be stale.
	if !guard.IsStale(time.Time{}, now) {
		t.Error("expected zero timestamp to be stale")
	}
}

func TestStalenessGuard_LargeClockSkew(t *testing.T) {
	guard := appexec.NewStalenessGuard(2 * time.Minute)
	now := time.Now().UTC()

	// Large future clock skew: negative age. Not stale.
	intentTS := now.Add(1 * time.Hour)
	if guard.IsStale(intentTS, now) {
		t.Error("expected large future skew to not be stale")
	}

	// Large past: definitely stale.
	intentTS = now.Add(-24 * time.Hour)
	if !guard.IsStale(intentTS, now) {
		t.Error("expected 24h-old intent to be stale")
	}
}

func TestStalenessGuard_JustOverBoundary(t *testing.T) {
	guard := appexec.NewStalenessGuard(2 * time.Minute)
	now := time.Now().UTC()

	// 1ns past boundary — stale.
	intentTS := now.Add(-2*time.Minute - 1*time.Nanosecond)
	if !guard.IsStale(intentTS, now) {
		t.Error("expected 1ns past boundary to be stale")
	}
}

func TestStalenessGuard_JustUnderBoundary(t *testing.T) {
	guard := appexec.NewStalenessGuard(2 * time.Minute)
	now := time.Now().UTC()

	// 1ns under boundary — not stale.
	intentTS := now.Add(-2*time.Minute + 1*time.Nanosecond)
	if guard.IsStale(intentTS, now) {
		t.Error("expected 1ns under boundary to not be stale")
	}
}
