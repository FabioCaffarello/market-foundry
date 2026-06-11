package execution_test

import (
	"testing"
	"time"

	"internal/domain/execution"
	"internal/shared/clock"
)

func TestDefaultVerificationScope_Uses24hWindow(t *testing.T) {
	before := time.Now().UTC()
	scope := execution.DefaultVerificationScope(clock.SystemClock{})
	after := time.Now().UTC()

	if len(scope.Symbols) != 1 || scope.Symbols[0] != "btcusdt" {
		t.Errorf("expected [BTCUSDT], got %v", scope.Symbols)
	}

	// Since should be ~24h before now.
	expectedSince := before.Add(-24 * time.Hour)
	if scope.Since.Before(expectedSince.Add(-time.Second)) || scope.Since.After(after.Add(-24*time.Hour).Add(time.Second)) {
		t.Errorf("since %v not within expected 24h window", scope.Since)
	}

	if scope.Until.Before(before) || scope.Until.After(after.Add(time.Second)) {
		t.Errorf("until %v not within expected range", scope.Until)
	}
}

func TestVerificationScope_InReport(t *testing.T) {
	scope := execution.VerificationScope{
		Symbols:   []string{"BTCUSDT"},
		Since:     time.Now().UTC().Add(-1 * time.Hour),
		Until:     time.Now().UTC(),
		Segments:  []string{"spot"},
		DryRun:    true,
		VenueType: "binance_spot",
	}

	report := execution.POVerificationReport{
		SessionID: "session_test",
		Scope:     &scope,
		Checks: []execution.POCheckResult{
			{CheckID: execution.POCheckGateHalted, Verdict: execution.VerdictPass, Automated: true},
		},
	}
	report.ComputeSummary()

	if report.Scope == nil {
		t.Fatal("expected scope in report")
	}
	if report.Scope.VenueType != "binance_spot" {
		t.Errorf("expected venue_type binance_spot, got %s", report.Scope.VenueType)
	}
	if !report.Scope.DryRun {
		t.Error("expected dry_run=true")
	}
}

func TestBatchCheckAggregation_InSummary(t *testing.T) {
	entries := []execution.BatchAuditEntry{
		{
			SessionID: "s1",
			Bundle: &execution.SessionAuditBundle{
				Consistency: execution.AuditConsistency{OverallVerdict: "consistent"},
				Verification: &execution.POVerificationReport{
					Checks: []execution.POCheckResult{
						{CheckID: execution.POCheckGateHalted, Verdict: execution.VerdictPass},
						{CheckID: execution.POCheckBackupCompleted, Verdict: execution.VerdictManual},
						{CheckID: execution.POCheckIntentRecords, Verdict: execution.VerdictPass},
						{CheckID: execution.POCheckFeeFields, Verdict: execution.VerdictWarn},
					},
				},
			},
		},
		{
			SessionID: "s2",
			Bundle: &execution.SessionAuditBundle{
				Consistency: execution.AuditConsistency{OverallVerdict: "inconsistent"},
				Verification: &execution.POVerificationReport{
					Checks: []execution.POCheckResult{
						{CheckID: execution.POCheckGateHalted, Verdict: execution.VerdictPass},
						{CheckID: execution.POCheckBackupCompleted, Verdict: execution.VerdictManual},
						{CheckID: execution.POCheckIntentRecords, Verdict: execution.VerdictFail},
						{CheckID: execution.POCheckFeeFields, Verdict: execution.VerdictFail},
					},
				},
			},
		},
		{
			SessionID: "s3",
			Error:     "session not found",
		},
	}

	s := execution.ComputeBatchSummary(entries)

	// Session-level verdicts.
	if s.TotalSessions != 3 {
		t.Errorf("expected 3 total, got %d", s.TotalSessions)
	}
	if s.Consistent != 1 {
		t.Errorf("expected 1 consistent, got %d", s.Consistent)
	}
	if s.Inconsistent != 1 {
		t.Errorf("expected 1 inconsistent, got %d", s.Inconsistent)
	}
	if s.Errored != 1 {
		t.Errorf("expected 1 errored, got %d", s.Errored)
	}

	// S485: Check aggregation.
	if len(s.CheckAggregation) == 0 {
		t.Fatal("expected non-empty check aggregation")
	}

	// Find PO-1 (gate halted) — should be 2 pass, 0 fail.
	found := false
	for _, agg := range s.CheckAggregation {
		if agg.CheckID == string(execution.POCheckGateHalted) {
			found = true
			if agg.PassCount != 2 {
				t.Errorf("PO-1 pass count: expected 2, got %d", agg.PassCount)
			}
			if agg.FailCount != 0 {
				t.Errorf("PO-1 fail count: expected 0, got %d", agg.FailCount)
			}
		}
	}
	if !found {
		t.Error("PO-1 not found in check aggregation")
	}

	// Find PO-3 (intent records) — should be 1 pass, 1 fail.
	for _, agg := range s.CheckAggregation {
		if agg.CheckID == string(execution.POCheckIntentRecords) {
			if agg.PassCount != 1 {
				t.Errorf("PO-3 pass count: expected 1, got %d", agg.PassCount)
			}
			if agg.FailCount != 1 {
				t.Errorf("PO-3 fail count: expected 1, got %d", agg.FailCount)
			}
		}
	}

	// Find PO-7 (fee fields) — should be 0 pass, 1 fail, 1 warn.
	for _, agg := range s.CheckAggregation {
		if agg.CheckID == string(execution.POCheckFeeFields) {
			if agg.WarnCount != 1 {
				t.Errorf("PO-7 warn count: expected 1, got %d", agg.WarnCount)
			}
			if agg.FailCount != 1 {
				t.Errorf("PO-7 fail count: expected 1, got %d", agg.FailCount)
			}
		}
	}
}

func TestBatchCheckAggregation_EmptyWhenNoVerification(t *testing.T) {
	entries := []execution.BatchAuditEntry{
		{
			SessionID: "s1",
			Bundle: &execution.SessionAuditBundle{
				Consistency: execution.AuditConsistency{OverallVerdict: "degraded"},
				// No verification report.
			},
		},
	}

	s := execution.ComputeBatchSummary(entries)
	if len(s.CheckAggregation) != 0 {
		t.Errorf("expected empty check aggregation, got %d entries", len(s.CheckAggregation))
	}
}
