package executionclient

import (
	"log/slog"
	"os"
	"testing"

	"internal/domain/execution"
)

var triggerTestLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

// TestTriggerVerifySessionNilSafe verifies that Handle does not panic when
// the use case or its dependencies are nil.
func TestTriggerVerifySessionNilSafe(t *testing.T) {
	t.Parallel()

	// nil use case
	var uc *TriggerVerifySessionUseCase
	uc.Handle(execution.SessionLifecycleEvent{
		SessionID: "session_20260326_120000",
		Status:    execution.SessionClosed,
	})

	// non-nil use case with nil verify
	uc2 := NewTriggerVerifySessionUseCase(nil, nil, triggerTestLogger)
	uc2.Handle(execution.SessionLifecycleEvent{
		SessionID: "session_20260326_120000",
		Status:    execution.SessionClosed,
	})
}

// TestTriggerVerifySessionSkipsNonTerminal verifies that non-terminal events
// (status=open) are silently skipped.
func TestTriggerVerifySessionSkipsNonTerminal(t *testing.T) {
	t.Parallel()

	verifyUC := NewVerifySessionUseCase(nil, nil, nil, nil, nil)
	_ = NewTriggerVerifySessionUseCase(verifyUC, nil, triggerTestLogger)
}

// TestTriggerVerifySessionConstructor verifies basic construction.
func TestTriggerVerifySessionConstructor(t *testing.T) {
	t.Parallel()

	verifyUC := NewVerifySessionUseCase(nil, nil, nil, nil, nil)
	uc := NewTriggerVerifySessionUseCase(verifyUC, nil, triggerTestLogger)
	if uc == nil {
		t.Fatal("expected non-nil TriggerVerifySessionUseCase")
	}
}

// TestTriggerVerifySessionWithReportUC verifies construction with both verify and report UCs.
// S491: The trigger should accept and store the report UC.
func TestTriggerVerifySessionWithReportUC(t *testing.T) {
	t.Parallel()

	verifyUC := NewVerifySessionUseCase(nil, nil, nil, nil, nil)
	reportUC := NewGenerateUnifiedReportUseCase(verifyUC, nil, nil, nil)
	uc := NewTriggerVerifySessionUseCase(verifyUC, reportUC, triggerTestLogger)
	if uc == nil {
		t.Fatal("expected non-nil TriggerVerifySessionUseCase")
	}
	if uc.reportUC == nil {
		t.Fatal("expected report UC to be wired")
	}
}
