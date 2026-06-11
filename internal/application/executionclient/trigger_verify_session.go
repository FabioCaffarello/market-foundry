package executionclient

import (
	"context"
	"log/slog"
	"time"

	"internal/domain/execution"
)

// TriggerVerifySessionUseCase reacts to session lifecycle events by running
// automated PO verification and producing the unified operational report.
// This is the event-driven complement to the manual verification path
// (HTTP endpoint + script).
//
// S490: Closes G-OA1 — verification can now be triggered by session close/halt
// events without operator intervention.
//
// S491: Extended to also produce the unified operational report after
// verification, closing G-OA2 (unified report artifact) and G-OA5
// (end-to-end automation proof). The chain is:
//
//	session halt → lifecycle event → trigger → verify → unified report → log
//
// Fail-closed semantics:
//   - If verification fails: log error, do not retry (same inputs would fail again).
//   - If report generation fails: log error, verification result still logged.
//   - If session is not terminal: skip silently (only closed/halted trigger verification).
//   - If verify use case is nil: log warning, no-op (gateway runs without verification wiring).
type TriggerVerifySessionUseCase struct {
	verify   *VerifySessionUseCase
	reportUC *GenerateUnifiedReportUseCase // S491: optional
	logger   *slog.Logger
}

func NewTriggerVerifySessionUseCase(verify *VerifySessionUseCase, reportUC *GenerateUnifiedReportUseCase, logger *slog.Logger) *TriggerVerifySessionUseCase {
	return &TriggerVerifySessionUseCase{
		verify:   verify,
		reportUC: reportUC,
		logger:   logger,
	}
}

// Handle processes a session lifecycle event. Called by the consumer callback.
// This method never returns an error — all failures are logged and absorbed
// (fail-closed: verification failure does not block the event pipeline).
func (uc *TriggerVerifySessionUseCase) Handle(event execution.SessionLifecycleEvent) {
	if uc == nil || uc.verify == nil {
		return
	}

	// Only trigger verification for terminal session states.
	if !event.Status.IsTerminal() {
		uc.logger.Debug("skipping non-terminal session lifecycle event",
			"session_id", event.SessionID,
			"status", string(event.Status),
		)
		return
	}

	uc.logger.Info("verification trigger received",
		"session_id", event.SessionID,
		"status", string(event.Status),
		"operator", event.Operator,
	)

	// Small delay to allow ClickHouse writes to settle. Verification reads from
	// ClickHouse for PO-3, PO-4, PO-7, PO-9 — if run immediately after session
	// close, recent writes may not yet be queryable.
	time.Sleep(5 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reply, prob := uc.verify.Execute(ctx, SessionVerifyQuery{SessionID: event.SessionID})
	if prob != nil {
		uc.logger.Error("event-driven verification failed",
			"session_id", event.SessionID,
			"error", prob.Message,
			"code", prob.Code,
		)
		return
	}

	report := reply.Report
	uc.logger.Info("event-driven verification completed",
		"session_id", event.SessionID,
		"total", report.Summary.Total,
		"passed", report.Summary.Passed,
		"failed", report.Summary.Failed,
		"warnings", report.Summary.Warnings,
		"skipped", report.Summary.Skipped,
		"duration_ms", report.DurationMs,
		"all_passed", report.AllPassed(),
	)

	if !report.AllPassed() {
		uc.logger.Warn("event-driven verification detected failures",
			"session_id", event.SessionID,
			"failed_count", report.Summary.Failed,
		)
	}

	// S491: Generate unified operational report if the UC is wired.
	if uc.reportUC == nil {
		return
	}

	reportCtx, reportCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer reportCancel()

	reportReply, reportProb := uc.reportUC.Execute(reportCtx, SessionUnifiedReportQuery{
		SessionID:   event.SessionID,
		GeneratedBy: "auto-trigger",
	})
	if reportProb != nil {
		uc.logger.Error("event-driven unified report generation failed",
			"session_id", event.SessionID,
			"error", reportProb.Message,
		)
		return
	}

	unified := reportReply.Report
	uc.logger.Info("event-driven unified report generated",
		"session_id", event.SessionID,
		"verdict", string(unified.Verdict),
		"duration_ms", unified.DurationMs,
		"gaps", len(unified.Gaps),
		"has_verification", unified.Verification != nil,
		"has_audit", unified.Audit != nil,
		"has_operational_state", unified.OperationalState != nil,
		"has_triage", unified.Triage != nil,
	)
}
