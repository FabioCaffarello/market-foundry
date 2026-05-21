package execution

import "testing"

func TestNewAuditCheckIndex_PopulatesVerdicts(t *testing.T) {
	report := &POVerificationReport{
		Checks: []POCheckResult{
			{CheckID: POCheckGateHalted, Verdict: VerdictPass},
			{CheckID: POCheckBackupCompleted, Verdict: VerdictManual},
			{CheckID: POCheckIntentRecords, Verdict: VerdictFail},
			{CheckID: POCheckVenueResponses, Verdict: VerdictWarn},
		},
	}

	idx := NewAuditCheckIndex(report)

	if len(idx.Verdicts) != 4 {
		t.Errorf("expected 4 verdicts, got %d", len(idx.Verdicts))
	}
	if idx.Verdicts[string(POCheckGateHalted)] != string(VerdictPass) {
		t.Errorf("expected PO-1 pass, got %s", idx.Verdicts[string(POCheckGateHalted)])
	}
	if len(idx.Failed) != 1 || idx.Failed[0] != string(POCheckIntentRecords) {
		t.Errorf("expected PO-3 in failed, got %v", idx.Failed)
	}
	if len(idx.Warnings) != 1 || idx.Warnings[0] != string(POCheckVenueResponses) {
		t.Errorf("expected PO-4 in warnings, got %v", idx.Warnings)
	}
}

func TestNewAuditCheckIndex_NilReport(t *testing.T) {
	idx := NewAuditCheckIndex(nil)
	if len(idx.Verdicts) != 0 {
		t.Errorf("expected empty verdicts for nil report, got %d", len(idx.Verdicts))
	}
}

func TestComputeBatchSummary(t *testing.T) {
	entries := []BatchAuditEntry{
		{SessionID: "s1", Bundle: &SessionAuditBundle{Consistency: AuditConsistency{OverallVerdict: "consistent"}}},
		{SessionID: "s2", Bundle: &SessionAuditBundle{Consistency: AuditConsistency{OverallVerdict: "degraded"}}},
		{SessionID: "s3", Bundle: &SessionAuditBundle{Consistency: AuditConsistency{OverallVerdict: "inconsistent"}}},
		{SessionID: "s4", Error: "session not found"},
		{SessionID: "s5", Bundle: nil},
	}

	s := ComputeBatchSummary(entries)

	if s.TotalSessions != 5 {
		t.Errorf("expected 5 total, got %d", s.TotalSessions)
	}
	if s.Consistent != 1 {
		t.Errorf("expected 1 consistent, got %d", s.Consistent)
	}
	if s.Degraded != 1 {
		t.Errorf("expected 1 degraded, got %d", s.Degraded)
	}
	if s.Inconsistent != 1 {
		t.Errorf("expected 1 inconsistent, got %d", s.Inconsistent)
	}
	if s.Errored != 2 {
		t.Errorf("expected 2 errored, got %d", s.Errored)
	}
}
