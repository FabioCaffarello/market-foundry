package execution_test

import (
	"testing"

	"internal/domain/execution"
)

func TestAllPOChecks_ReturnsNine(t *testing.T) {
	checks := execution.AllPOChecks()
	if len(checks) != 9 {
		t.Fatalf("expected 9 PO checks, got %d", len(checks))
	}
	// Verify canonical order PO-1 through PO-9.
	expected := []execution.POCheckID{
		execution.POCheckGateHalted,
		execution.POCheckBackupCompleted,
		execution.POCheckIntentRecords,
		execution.POCheckVenueResponses,
		execution.POCheckKVState,
		execution.POCheckSystemStatus,
		execution.POCheckFeeFields,
		execution.POCheckLifecycleConsist,
		execution.POCheckScopeContainment,
	}
	for i, exp := range expected {
		if checks[i] != exp {
			t.Errorf("check[%d]: expected %s, got %s", i, exp, checks[i])
		}
	}
}

func TestPOVerificationReport_ComputeSummary(t *testing.T) {
	report := execution.POVerificationReport{
		SessionID: "session_20260324_120000",
		Checks: []execution.POCheckResult{
			{CheckID: "PO-1", Verdict: execution.VerdictPass, Automated: true},
			{CheckID: "PO-2", Verdict: execution.VerdictManual, Automated: false},
			{CheckID: "PO-3", Verdict: execution.VerdictPass, Automated: true},
			{CheckID: "PO-4", Verdict: execution.VerdictPass, Automated: true},
			{CheckID: "PO-5", Verdict: execution.VerdictPass, Automated: true},
			{CheckID: "PO-6", Verdict: execution.VerdictPass, Automated: true},
			{CheckID: "PO-7", Verdict: execution.VerdictWarn, Automated: true},
			{CheckID: "PO-8", Verdict: execution.VerdictSkip, Automated: true},
			{CheckID: "PO-9", Verdict: execution.VerdictFail, Automated: true},
		},
	}

	report.ComputeSummary()

	if report.Summary.Total != 9 {
		t.Errorf("total: expected 9, got %d", report.Summary.Total)
	}
	if report.Summary.Passed != 5 {
		t.Errorf("passed: expected 5, got %d", report.Summary.Passed)
	}
	if report.Summary.Failed != 1 {
		t.Errorf("failed: expected 1, got %d", report.Summary.Failed)
	}
	if report.Summary.Warnings != 1 {
		t.Errorf("warnings: expected 1, got %d", report.Summary.Warnings)
	}
	if report.Summary.Skipped != 1 {
		t.Errorf("skipped: expected 1, got %d", report.Summary.Skipped)
	}
	if report.Summary.Manual != 1 {
		t.Errorf("manual: expected 1, got %d", report.Summary.Manual)
	}
	if report.Summary.Automated != 8 {
		t.Errorf("automated: expected 8, got %d", report.Summary.Automated)
	}
}

func TestPOVerificationReport_AllPassed(t *testing.T) {
	tests := []struct {
		name     string
		checks   []execution.POCheckResult
		expected bool
	}{
		{
			name: "all pass",
			checks: []execution.POCheckResult{
				{Verdict: execution.VerdictPass},
				{Verdict: execution.VerdictPass},
			},
			expected: true,
		},
		{
			name: "one fail",
			checks: []execution.POCheckResult{
				{Verdict: execution.VerdictPass},
				{Verdict: execution.VerdictFail},
			},
			expected: false,
		},
		{
			name: "warn and skip are ok",
			checks: []execution.POCheckResult{
				{Verdict: execution.VerdictPass},
				{Verdict: execution.VerdictWarn},
				{Verdict: execution.VerdictSkip},
				{Verdict: execution.VerdictManual},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := execution.POVerificationReport{Checks: tt.checks}
			r.ComputeSummary()
			if r.AllPassed() != tt.expected {
				t.Errorf("AllPassed: expected %v, got %v", tt.expected, r.AllPassed())
			}
		})
	}
}
