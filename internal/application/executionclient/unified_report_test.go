package executionclient

import (
	"context"
	"testing"

	"internal/shared/problem"
)

// stubMonitoringReader implements UnifiedReportMonitoringReader for testing.
type stubMonitoringReader struct {
	gate       string
	gateReason string
	surfaces   []string
	err        *problem.Problem
}

func (s *stubMonitoringReader) GetOperationalState(_ context.Context) (string, string, []string, *problem.Problem) {
	return s.gate, s.gateReason, s.surfaces, s.err
}

// stubTriageReader implements UnifiedReportTriageReader for testing.
type stubTriageReader struct {
	total     int
	sCrit     int
	sWarn     int
	dCrit     int
	dWarn     int
	rtCrit    int
	rtWarn    int
	findings  []string
	err       *problem.Problem
}

func (s *stubTriageReader) GetTriageSummary(_ context.Context) (int, int, int, int, int, int, int, []string, *problem.Problem) {
	return s.total, s.sCrit, s.sWarn, s.dCrit, s.dWarn, s.rtCrit, s.rtWarn, s.findings, s.err
}

func TestGenerateUnifiedReportRequiresSessionID(t *testing.T) {
	t.Parallel()

	uc := NewGenerateUnifiedReportUseCase(nil, nil, nil, nil)
	_, prob := uc.Execute(context.Background(), SessionUnifiedReportQuery{})
	if prob == nil {
		t.Fatal("expected error for empty session_id")
	}
}

func TestGenerateUnifiedReportAllNilDeps(t *testing.T) {
	t.Parallel()

	uc := NewGenerateUnifiedReportUseCase(nil, nil, nil, nil)
	reply, prob := uc.Execute(context.Background(), SessionUnifiedReportQuery{SessionID: "session_1"})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}

	r := reply.Report
	if r.SessionID != "session_1" {
		t.Fatalf("expected session_1, got %s", r.SessionID)
	}
	if len(r.Gaps) != 4 {
		t.Fatalf("expected 4 gaps (all sections unavailable), got %d", len(r.Gaps))
	}
	if r.Verdict != "degraded" {
		t.Fatalf("expected degraded verdict, got %s", r.Verdict)
	}
}

func TestGenerateUnifiedReportWithMonitoringAndTriage(t *testing.T) {
	t.Parallel()

	monReader := &stubMonitoringReader{
		gate:       "halted",
		gateReason: "operator halt",
		surfaces:   []string{"evidence", "session"},
	}
	triageReader := &stubTriageReader{
		total:    3,
		sWarn:    2,
		dCrit:    1,
		findings: []string{"something needs attention"},
	}

	uc := NewGenerateUnifiedReportUseCase(nil, nil, monReader, triageReader)
	reply, prob := uc.Execute(context.Background(), SessionUnifiedReportQuery{SessionID: "session_2"})
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}

	r := reply.Report
	if r.OperationalState == nil {
		t.Fatal("expected operational state section")
	}
	if r.OperationalState.GateStatus != "halted" {
		t.Fatalf("expected halted gate, got %s", r.OperationalState.GateStatus)
	}
	if r.Triage == nil {
		t.Fatal("expected triage section")
	}
	if r.Triage.DecisionCritical != 1 {
		t.Fatalf("expected 1 decision critical, got %d", r.Triage.DecisionCritical)
	}
	// 2 gaps: verification and audit not wired.
	if len(r.Gaps) != 2 {
		t.Fatalf("expected 2 gaps, got %d", len(r.Gaps))
	}
}

func TestGenerateUnifiedReportGeneratedByDefault(t *testing.T) {
	t.Parallel()

	uc := NewGenerateUnifiedReportUseCase(nil, nil, nil, nil)
	reply, _ := uc.Execute(context.Background(), SessionUnifiedReportQuery{SessionID: "s1"})
	if reply.Report.GeneratedBy != "http-request" {
		t.Fatalf("expected default generated_by=http-request, got %s", reply.Report.GeneratedBy)
	}
}

func TestGenerateUnifiedReportGeneratedByAutoTrigger(t *testing.T) {
	t.Parallel()

	uc := NewGenerateUnifiedReportUseCase(nil, nil, nil, nil)
	reply, _ := uc.Execute(context.Background(), SessionUnifiedReportQuery{
		SessionID:   "s1",
		GeneratedBy: "auto-trigger",
	})
	if reply.Report.GeneratedBy != "auto-trigger" {
		t.Fatalf("expected generated_by=auto-trigger, got %s", reply.Report.GeneratedBy)
	}
}

func TestGenerateUnifiedReportMonitoringError(t *testing.T) {
	t.Parallel()

	monReader := &stubMonitoringReader{
		err: problem.New(problem.Internal, "monitoring unavailable"),
	}
	uc := NewGenerateUnifiedReportUseCase(nil, nil, monReader, nil)
	reply, prob := uc.Execute(context.Background(), SessionUnifiedReportQuery{SessionID: "s1"})
	if prob != nil {
		t.Fatalf("use case should not fail, errors become gaps: %v", prob)
	}

	hasMonGap := false
	for _, g := range reply.Report.Gaps {
		if g.Section == "operational_state" {
			hasMonGap = true
		}
	}
	if !hasMonGap {
		t.Fatal("expected operational_state gap when monitoring fails")
	}
}
