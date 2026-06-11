package triageclient

import (
	"context"
	"testing"

	"internal/application/executionclient"
	"internal/domain/execution"
	"internal/domain/triage"
	"internal/shared/problem"
)

type stubBatchAuditor struct {
	reply executionclient.SessionBatchAuditReply
	prob  *problem.Problem
}

func (s *stubBatchAuditor) Execute(_ context.Context, _ executionclient.SessionBatchAuditQuery) (executionclient.SessionBatchAuditReply, *problem.Problem) {
	return s.reply, s.prob
}

func TestGetSessionTriage_RanksAnomaliesFirst(t *testing.T) {
	auditor := &stubBatchAuditor{
		reply: executionclient.SessionBatchAuditReply{
			Result: execution.BatchAuditResult{
				Entries: []execution.BatchAuditEntry{
					{
						SessionID: "clean",
						Bundle: &execution.SessionAuditBundle{
							Session:     execution.Session{Status: execution.SessionClosed},
							Consistency: execution.AuditConsistency{OverallVerdict: "consistent", CountersMatchActivity: true},
							CheckIndex:  execution.AuditCheckIndex{Verdicts: map[string]string{}},
						},
					},
					{
						SessionID: "broken",
						Bundle: &execution.SessionAuditBundle{
							Session:     execution.Session{Status: execution.SessionClosed},
							Consistency: execution.AuditConsistency{OverallVerdict: "inconsistent", CountersMatchActivity: false},
							CheckIndex: execution.AuditCheckIndex{
								Verdicts: map[string]string{"PO-1": "fail", "PO-3": "fail"},
								Failed:   []string{"PO-1", "PO-3"},
							},
						},
					},
					{
						SessionID: "warn",
						Bundle: &execution.SessionAuditBundle{
							Session:     execution.Session{Status: execution.SessionClosed},
							Consistency: execution.AuditConsistency{OverallVerdict: "degraded", CountersMatchActivity: true},
							CheckIndex: execution.AuditCheckIndex{
								Verdicts: map[string]string{"PO-5": "warn"},
								Warnings: []string{"PO-5"},
							},
						},
					},
				},
			},
		},
	}

	uc := NewGetSessionTriageUseCase(auditor)
	reply, prob := uc.Execute(context.Background(), SessionTriageQuery{})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	// The clean session should be excluded (info severity, no anomalies).
	// Only broken and warn should appear, with broken first.
	if len(reply.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(reply.Items))
	}

	if reply.Items[0].SessionID != "broken" {
		t.Errorf("expected broken first, got %s", reply.Items[0].SessionID)
	}
	if reply.Items[0].Severity != triage.SeverityCritical {
		t.Errorf("expected critical severity, got %s", reply.Items[0].Severity)
	}
	if reply.Items[0].AnomalyCount != 3 { // 2 failed + 1 counter mismatch
		t.Errorf("expected 3 anomalies, got %d", reply.Items[0].AnomalyCount)
	}

	if reply.Items[1].SessionID != "warn" {
		t.Errorf("expected warn second, got %s", reply.Items[1].SessionID)
	}
}

func TestGetSessionTriage_CheckFilter(t *testing.T) {
	auditor := &stubBatchAuditor{
		reply: executionclient.SessionBatchAuditReply{
			Result: execution.BatchAuditResult{
				Entries: []execution.BatchAuditEntry{
					{
						SessionID: "s1",
						Bundle: &execution.SessionAuditBundle{
							Session:     execution.Session{Status: execution.SessionClosed},
							Consistency: execution.AuditConsistency{OverallVerdict: "inconsistent"},
							CheckIndex: execution.AuditCheckIndex{
								Failed: []string{"PO-1"},
							},
						},
					},
					{
						SessionID: "s2",
						Bundle: &execution.SessionAuditBundle{
							Session:     execution.Session{Status: execution.SessionClosed},
							Consistency: execution.AuditConsistency{OverallVerdict: "inconsistent"},
							CheckIndex: execution.AuditCheckIndex{
								Failed: []string{"PO-3"},
							},
						},
					},
				},
			},
		},
	}

	uc := NewGetSessionTriageUseCase(auditor)
	reply, prob := uc.Execute(context.Background(), SessionTriageQuery{
		CheckFilter: "PO-1",
	})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(reply.Items) != 1 {
		t.Fatalf("expected 1 item (filtered by PO-1), got %d", len(reply.Items))
	}
	if reply.Items[0].SessionID != "s1" {
		t.Errorf("expected s1, got %s", reply.Items[0].SessionID)
	}
}

func TestGetSessionTriage_SeverityFilter(t *testing.T) {
	auditor := &stubBatchAuditor{
		reply: executionclient.SessionBatchAuditReply{
			Result: execution.BatchAuditResult{
				Entries: []execution.BatchAuditEntry{
					{
						SessionID: "critical-session",
						Bundle: &execution.SessionAuditBundle{
							Session:     execution.Session{Status: execution.SessionClosed},
							Consistency: execution.AuditConsistency{OverallVerdict: "inconsistent"},
							CheckIndex:  execution.AuditCheckIndex{Failed: []string{"PO-1"}},
						},
					},
					{
						SessionID: "warn-session",
						Bundle: &execution.SessionAuditBundle{
							Session:     execution.Session{Status: execution.SessionClosed},
							Consistency: execution.AuditConsistency{OverallVerdict: "degraded"},
							CheckIndex:  execution.AuditCheckIndex{Warnings: []string{"PO-5"}},
						},
					},
				},
			},
		},
	}

	uc := NewGetSessionTriageUseCase(auditor)
	reply, prob := uc.Execute(context.Background(), SessionTriageQuery{
		SeverityFilter: "critical",
	})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(reply.Items) != 1 {
		t.Fatalf("expected 1 critical item, got %d", len(reply.Items))
	}
	if reply.Items[0].SessionID != "critical-session" {
		t.Errorf("expected critical-session, got %s", reply.Items[0].SessionID)
	}
}

func TestGetSessionTriage_ErrorEntry(t *testing.T) {
	auditor := &stubBatchAuditor{
		reply: executionclient.SessionBatchAuditReply{
			Result: execution.BatchAuditResult{
				Entries: []execution.BatchAuditEntry{
					{SessionID: "errored", Error: "failed to fetch session"},
				},
			},
		},
	}

	uc := NewGetSessionTriageUseCase(auditor)
	reply, prob := uc.Execute(context.Background(), SessionTriageQuery{})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(reply.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(reply.Items))
	}
	if reply.Items[0].Severity != triage.SeverityCritical {
		t.Errorf("errored entry should be critical, got %s", reply.Items[0].Severity)
	}
}

func TestGetSessionTriage_NilDependency(t *testing.T) {
	uc := NewGetSessionTriageUseCase(nil)
	_, prob := uc.Execute(context.Background(), SessionTriageQuery{})

	if prob == nil {
		t.Fatal("expected problem when dependency is nil")
	}
}
