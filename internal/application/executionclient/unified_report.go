package executionclient

import (
	"context"
	"time"

	"internal/domain/execution"
	"internal/shared/problem"
)

// UnifiedReportMonitoringReader can read the current operational state.
// S491: Used by the unified report to capture gate and surface availability.
type UnifiedReportMonitoringReader interface {
	GetOperationalState(ctx context.Context) (gate string, gateReason string, surfaces []string, err *problem.Problem)
}

// UnifiedReportTriageReader can read the triage overview.
// S491: Used by the unified report to capture cross-domain anomaly counts.
type UnifiedReportTriageReader interface {
	GetTriageSummary(ctx context.Context) (totalAnomalies int, sessionCritical, sessionWarning, decisionCritical, decisionWarning, rtCritical, rtWarning int, topFindings []string, err *problem.Problem)
}

// SessionUnifiedReportQuery is the request contract for the unified operational report.
type SessionUnifiedReportQuery struct {
	SessionID   string `json:"session_id"`
	GeneratedBy string `json:"generated_by"` // "auto-trigger" or "http-request"
}

// SessionUnifiedReportReply is the response contract for the unified report.
type SessionUnifiedReportReply struct {
	Report execution.UnifiedOperationalReport `json:"report"`
}

// GenerateUnifiedReportUseCase composes a single archivable operational report
// from verification, audit, monitoring, and triage data for a given session.
//
// S491: Closes G-OA2 (unified operational report artifact) and participates in
// closing G-OA5 (end-to-end automation proof) when invoked by the event-driven
// trigger after automated verification.
//
// Each data source is optional — the report degrades gracefully by recording
// gaps for unavailable sections rather than failing entirely.
type GenerateUnifiedReportUseCase struct {
	verifyUC   *VerifySessionUseCase
	auditUC    *AuditSessionUseCase
	monitoring UnifiedReportMonitoringReader
	triage     UnifiedReportTriageReader
}

func NewGenerateUnifiedReportUseCase(
	verifyUC *VerifySessionUseCase,
	auditUC *AuditSessionUseCase,
	monitoring UnifiedReportMonitoringReader,
	triage UnifiedReportTriageReader,
) *GenerateUnifiedReportUseCase {
	return &GenerateUnifiedReportUseCase{
		verifyUC:   verifyUC,
		auditUC:    auditUC,
		monitoring: monitoring,
		triage:     triage,
	}
}

func (uc *GenerateUnifiedReportUseCase) Execute(ctx context.Context, query SessionUnifiedReportQuery) (SessionUnifiedReportReply, *problem.Problem) {
	if query.SessionID == "" {
		return SessionUnifiedReportReply{}, problem.New(problem.InvalidArgument, "session_id is required")
	}

	start := time.Now()
	generatedBy := query.GeneratedBy
	if generatedBy == "" {
		generatedBy = "http-request"
	}

	report := execution.UnifiedOperationalReport{
		SessionID:   query.SessionID,
		GeneratedAt: start.UTC(),
		GeneratedBy: generatedBy,
	}

	// Section 1: Verification.
	if uc.verifyUC != nil {
		reply, prob := uc.verifyUC.Execute(ctx, SessionVerifyQuery{SessionID: query.SessionID})
		if prob != nil {
			report.Gaps = append(report.Gaps, execution.ReportGap{
				Section: "verification",
				Reason:  prob.Message,
			})
		} else {
			report.Verification = &execution.ReportVerificationSection{
				AllPassed:  reply.Report.AllPassed(),
				Summary:    reply.Report.Summary,
				DurationMs: reply.Report.DurationMs,
				Checks:     reply.Report.Checks,
			}
		}
	} else {
		report.Gaps = append(report.Gaps, execution.ReportGap{
			Section: "verification",
			Reason:  "verify use case not wired",
		})
	}

	// Section 2: Audit.
	if uc.auditUC != nil {
		reply, prob := uc.auditUC.Execute(ctx, SessionAuditQuery{SessionID: query.SessionID})
		if prob != nil {
			report.Gaps = append(report.Gaps, execution.ReportGap{
				Section: "audit",
				Reason:  prob.Message,
			})
		} else {
			b := reply.Bundle
			report.Audit = &execution.ReportAuditSection{
				SessionStatus: string(b.Session.Status),
				Operator:      b.Session.Operator,
				OrderActivity: b.OrderActivity,
				FeeSummary:    b.FeeSummary,
				Consistency:   b.Consistency,
				CheckIndex:    b.CheckIndex,
			}
		}
	} else {
		report.Gaps = append(report.Gaps, execution.ReportGap{
			Section: "audit",
			Reason:  "audit use case not wired",
		})
	}

	// Section 3: Operational state.
	if uc.monitoring != nil {
		gate, gateReason, surfaces, prob := uc.monitoring.GetOperationalState(ctx)
		if prob != nil {
			report.Gaps = append(report.Gaps, execution.ReportGap{
				Section: "operational_state",
				Reason:  prob.Message,
			})
		} else {
			report.OperationalState = &execution.ReportOperationalStateSection{
				GateStatus:        gate,
				GateReason:        gateReason,
				AvailableSurfaces: surfaces,
			}
		}
	} else {
		report.Gaps = append(report.Gaps, execution.ReportGap{
			Section: "operational_state",
			Reason:  "monitoring reader not wired",
		})
	}

	// Section 4: Triage.
	if uc.triage != nil {
		totalAnomalies, sCrit, sWarn, dCrit, dWarn, rtCrit, rtWarn, findings, prob := uc.triage.GetTriageSummary(ctx)
		if prob != nil {
			report.Gaps = append(report.Gaps, execution.ReportGap{
				Section: "triage",
				Reason:  prob.Message,
			})
		} else {
			report.Triage = &execution.ReportTriageSection{
				TotalAnomalies:    totalAnomalies,
				SessionCritical:   sCrit,
				SessionWarning:    sWarn,
				DecisionCritical:  dCrit,
				DecisionWarning:   dWarn,
				RoundTripCritical: rtCrit,
				RoundTripWarning:  rtWarn,
				TopFindings:       findings,
			}
		}
	} else {
		report.Gaps = append(report.Gaps, execution.ReportGap{
			Section: "triage",
			Reason:  "triage reader not wired",
		})
	}

	report.DurationMs = time.Since(start).Milliseconds()
	report.ComputeVerdict()

	return SessionUnifiedReportReply{Report: report}, nil
}
