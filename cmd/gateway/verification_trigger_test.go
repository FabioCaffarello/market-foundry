package main

import (
	"log/slog"
	"os"
	"testing"

	"internal/application/executionclient"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

// TestStartVerificationTriggerNilVerifyUC validates that startVerificationTrigger
// handles a nil verify use case gracefully (returns nil, no panic).
func TestStartVerificationTriggerNilVerifyUC(t *testing.T) {
	t.Parallel()

	trigger := startVerificationTrigger("nats://localhost:4222", nil, nil, testLogger)
	if trigger != nil {
		t.Fatal("expected nil trigger when verify UC is nil")
	}
}

// TestVerificationTriggerCloseNilSafe validates that Close on a nil trigger
// does not panic.
func TestVerificationTriggerCloseNilSafe(t *testing.T) {
	t.Parallel()

	var trigger *verificationTrigger
	trigger.Close() // must not panic
}

// TestTriggerVerifySessionUseCaseConstructable validates that the trigger use case
// can be constructed from a VerifySessionUseCase.
func TestTriggerVerifySessionUseCaseConstructable(t *testing.T) {
	t.Parallel()

	verifyUC := executionclient.NewVerifySessionUseCase(nil, nil, nil, nil, nil)
	triggerUC := executionclient.NewTriggerVerifySessionUseCase(verifyUC, nil, testLogger)
	if triggerUC == nil {
		t.Fatal("expected non-nil TriggerVerifySessionUseCase")
	}
}

// TestTriggerWithReportUCConstructable validates that the trigger use case
// can be constructed with both verify and report UCs.
// S491: E2E chain requires both UCs wired.
func TestTriggerWithReportUCConstructable(t *testing.T) {
	t.Parallel()

	verifyUC := executionclient.NewVerifySessionUseCase(nil, nil, nil, nil, nil)
	reportUC := executionclient.NewGenerateUnifiedReportUseCase(verifyUC, nil, nil, nil)
	triggerUC := executionclient.NewTriggerVerifySessionUseCase(verifyUC, reportUC, testLogger)
	if triggerUC == nil {
		t.Fatal("expected non-nil TriggerVerifySessionUseCase with report UC")
	}
}
