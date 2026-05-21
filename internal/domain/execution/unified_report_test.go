package execution

import (
	"testing"
	"time"
)

func TestUnifiedReportComputeVerdictPass(t *testing.T) {
	t.Parallel()

	r := &UnifiedOperationalReport{
		SessionID:   "session_1",
		GeneratedAt: time.Now().UTC(),
		Verification: &ReportVerificationSection{
			AllPassed: true,
			Summary:   POSummary{Total: 9, Passed: 8, Skipped: 1},
		},
		Audit: &ReportAuditSection{
			Consistency: AuditConsistency{OverallVerdict: "consistent"},
		},
	}
	r.ComputeVerdict()

	if r.Verdict != ReportVerdictPass {
		t.Fatalf("expected pass, got %s", r.Verdict)
	}
}

func TestUnifiedReportComputeVerdictFail(t *testing.T) {
	t.Parallel()

	r := &UnifiedOperationalReport{
		SessionID:   "session_1",
		GeneratedAt: time.Now().UTC(),
		Verification: &ReportVerificationSection{
			AllPassed: false,
			Summary:   POSummary{Total: 9, Passed: 7, Failed: 1, Skipped: 1},
		},
	}
	r.ComputeVerdict()

	if r.Verdict != ReportVerdictFail {
		t.Fatalf("expected fail, got %s", r.Verdict)
	}
}

func TestUnifiedReportComputeVerdictWarn(t *testing.T) {
	t.Parallel()

	r := &UnifiedOperationalReport{
		SessionID:   "session_1",
		GeneratedAt: time.Now().UTC(),
		Verification: &ReportVerificationSection{
			AllPassed: false,
			Summary:   POSummary{Total: 9, Passed: 7, Warnings: 2},
		},
		Audit: &ReportAuditSection{
			Consistency: AuditConsistency{OverallVerdict: "consistent"},
		},
	}
	r.ComputeVerdict()

	if r.Verdict != ReportVerdictWarn {
		t.Fatalf("expected warn, got %s", r.Verdict)
	}
}

func TestUnifiedReportComputeVerdictDegraded(t *testing.T) {
	t.Parallel()

	r := &UnifiedOperationalReport{
		SessionID:   "session_1",
		GeneratedAt: time.Now().UTC(),
		Gaps: []ReportGap{
			{Section: "verification", Reason: "not wired"},
		},
	}
	r.ComputeVerdict()

	if r.Verdict != ReportVerdictDegraded {
		t.Fatalf("expected degraded, got %s", r.Verdict)
	}
}

func TestUnifiedReportComputeVerdictTriageCritical(t *testing.T) {
	t.Parallel()

	r := &UnifiedOperationalReport{
		SessionID:   "session_1",
		GeneratedAt: time.Now().UTC(),
		Verification: &ReportVerificationSection{
			AllPassed: true,
			Summary:   POSummary{Total: 9, Passed: 9},
		},
		Triage: &ReportTriageSection{
			TotalAnomalies:  2,
			SessionCritical: 1,
		},
	}
	r.ComputeVerdict()

	if r.Verdict != ReportVerdictFail {
		t.Fatalf("expected fail from triage critical, got %s", r.Verdict)
	}
}

func TestUnifiedReportComputeVerdictAuditInconsistent(t *testing.T) {
	t.Parallel()

	r := &UnifiedOperationalReport{
		SessionID:   "session_1",
		GeneratedAt: time.Now().UTC(),
		Verification: &ReportVerificationSection{
			AllPassed: true,
			Summary:   POSummary{Total: 9, Passed: 9},
		},
		Audit: &ReportAuditSection{
			Consistency: AuditConsistency{OverallVerdict: "inconsistent"},
		},
	}
	r.ComputeVerdict()

	if r.Verdict != ReportVerdictFail {
		t.Fatalf("expected fail from audit inconsistency, got %s", r.Verdict)
	}
}

func TestUnifiedReportComputeVerdictGapsWithPassingSections(t *testing.T) {
	t.Parallel()

	r := &UnifiedOperationalReport{
		SessionID:   "session_1",
		GeneratedAt: time.Now().UTC(),
		Verification: &ReportVerificationSection{
			AllPassed: true,
			Summary:   POSummary{Total: 9, Passed: 9},
		},
		Gaps: []ReportGap{
			{Section: "triage", Reason: "not wired"},
		},
	}
	r.ComputeVerdict()

	if r.Verdict != ReportVerdictDegraded {
		t.Fatalf("expected degraded when some sections unavailable, got %s", r.Verdict)
	}
}
