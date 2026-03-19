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
