package executionclient

import (
	"context"
	"testing"
	"time"

	"internal/domain/execution"
	"internal/shared/problem"
)

// stubSessionReader returns a fixed session.
type stubSessionReader struct {
	session *execution.Session
	prob    *problem.Problem
}

func (s *stubSessionReader) Execute(_ context.Context, q SessionGetQuery) (SessionGetReply, *problem.Problem) {
	if s.prob != nil {
		return SessionGetReply{}, s.prob
	}
	return SessionGetReply{Session: s.session}, nil
}

// stubVerifyExecutor returns a fixed verification report.
type stubVerifyExecutor struct {
	report execution.POVerificationReport
	prob   *problem.Problem
}

func (s *stubVerifyExecutor) Execute(_ context.Context, q SessionVerifyQuery) (SessionVerifyReply, *problem.Problem) {
	if s.prob != nil {
		return SessionVerifyReply{}, s.prob
	}
	return SessionVerifyReply{Report: s.report}, nil
}

// stubLifecycleReader returns fixed lifecycle entries.
type stubLifecycleReader struct {
	entries []LifecycleEntry
	prob    *problem.Problem
}

func (s *stubLifecycleReader) Execute(_ context.Context, q LifecycleListQuery) (LifecycleListReply, *problem.Problem) {
	if s.prob != nil {
		return LifecycleListReply{}, s.prob
	}
	return LifecycleListReply{Entries: s.entries, Total: len(s.entries)}, nil
}

// stubFillReader returns fixed fill data.
type stubFillReader struct {
	rows []VerifyCHListResult
	prob *problem.Problem
}

func (s *stubFillReader) List(_ context.Context, _, _, _ string, _ int, _, _ int64) ([]VerifyCHListResult, *problem.Problem) {
	if s.prob != nil {
		return nil, s.prob
	}
	return s.rows, nil
}

func closedTestSession() *execution.Session {
	now := time.Now().UTC()
	closed := now
	return &execution.Session{
		SessionID: "session_20260324_120000",
		Operator:  "test-operator",
		Status:    execution.SessionClosed,
		StartedAt: now.Add(-1 * time.Hour),
		ClosedAt:  &closed,
		Config: execution.SessionConfigSnapshot{
			VenueType: "binance_spot",
			DryRun:    true,
			Segments:  []string{"spot"},
		},
		Activation: execution.SessionActivationSnapshot{
			Adapter:     "paper",
			Credentials: "absent",
			GateStatus:  "active",
			Effective:   "paper",
		},
		SegmentCounters: []execution.SessionSegmentCounters{
			{Segment: "spot", Processed: 5, Filled: 3, Rejected: 1, Errors: 0},
		},
	}
}

func passingVerifyReport() execution.POVerificationReport {
	report := execution.POVerificationReport{
		SessionID:  "session_20260324_120000",
		ExecutedAt: time.Now().UTC(),
		Checks: []execution.POCheckResult{
			{CheckID: execution.POCheckGateHalted, Verdict: execution.VerdictPass, Automated: true},
			{CheckID: execution.POCheckBackupCompleted, Verdict: execution.VerdictManual, Automated: false},
			{CheckID: execution.POCheckIntentRecords, Verdict: execution.VerdictPass, Automated: true},
		},
	}
	report.ComputeSummary()
	return report
}

func TestAuditSession_FullBundle(t *testing.T) {
	session := closedTestSession()
	uc := NewAuditSessionUseCase(
		&stubSessionReader{session: session},
		&stubVerifyExecutor{report: passingVerifyReport()},
		&stubLifecycleReader{entries: []LifecycleEntry{
			{Key: "binance_spot.BTCUSDT.60", Source: "binance_spot", Symbol: "BTCUSDT", Timeframe: 60,
				IntentStatus: "submitted", FillStatus: "filled", Propagation: "filled"},
		}},
		&stubFillReader{rows: []VerifyCHListResult{
			{Symbol: "BTCUSDT", Status: "filled", Type: "venue_market_order",
				Fills: []execution.FillRecord{
					{Price: "50000", Quantity: "0.01", Fee: "0.001", FeeAsset: "BNB", Timestamp: time.Now()},
				}},
		}},
	)

	reply, prob := uc.Execute(context.Background(), SessionAuditQuery{SessionID: session.SessionID})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	b := reply.Bundle

	// Session metadata
	if b.Session.SessionID != session.SessionID {
		t.Errorf("expected session ID %s, got %s", session.SessionID, b.Session.SessionID)
	}

	// Verification
	if b.Verification == nil {
		t.Fatal("expected verification report")
	}
	if !b.Consistency.VerificationRan {
		t.Error("expected VerificationRan=true")
	}

	// Lifecycle
	if len(b.Lifecycle) != 1 {
		t.Errorf("expected 1 lifecycle entry, got %d", len(b.Lifecycle))
	}

	// Order activity from session counters
	if !b.OrderActivity.FromSessionCounters {
		t.Error("expected activity from session counters")
	}
	if b.OrderActivity.TotalIntents != 5 {
		t.Errorf("expected 5 intents, got %d", b.OrderActivity.TotalIntents)
	}

	// Fee summary
	if b.FeeSummary.TotalFillRecords != 1 {
		t.Errorf("expected 1 fill record, got %d", b.FeeSummary.TotalFillRecords)
	}
	if b.FeeSummary.FillsWithFee != 1 {
		t.Errorf("expected 1 fill with fee, got %d", b.FeeSummary.FillsWithFee)
	}

	// Consistency
	if !b.Consistency.SessionFound {
		t.Error("expected SessionFound=true")
	}
	if b.Consistency.OverallVerdict != "consistent" {
		t.Errorf("expected verdict consistent, got %s", b.Consistency.OverallVerdict)
	}

	// Explanation
	if b.Explanation == "" {
		t.Error("expected non-empty explanation")
	}

	// Assembly timing
	if b.AssemblyMs < 0 {
		t.Errorf("expected non-negative assembly time, got %d", b.AssemblyMs)
	}
}

func TestAuditSession_MissingSessionID(t *testing.T) {
	uc := NewAuditSessionUseCase(
		&stubSessionReader{session: closedTestSession()},
		nil, nil, nil,
	)
	_, prob := uc.Execute(context.Background(), SessionAuditQuery{})
	if prob == nil {
		t.Fatal("expected error for missing session_id")
	}
	if prob.Code != problem.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %s", prob.Code)
	}
}

func TestAuditSession_SessionNotFound(t *testing.T) {
	uc := NewAuditSessionUseCase(
		&stubSessionReader{prob: problem.New(problem.NotFound, "not found")},
		nil, nil, nil,
	)
	_, prob := uc.Execute(context.Background(), SessionAuditQuery{SessionID: "session_20260324_999999"})
	if prob == nil {
		t.Fatal("expected error for missing session")
	}
}

func TestAuditSession_DegradedWithoutVerification(t *testing.T) {
	session := closedTestSession()
	uc := NewAuditSessionUseCase(
		&stubSessionReader{session: session},
		nil, // no verification
		nil, // no lifecycle
		nil, // no fills
	)

	reply, prob := uc.Execute(context.Background(), SessionAuditQuery{SessionID: session.SessionID})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if reply.Bundle.Consistency.VerificationRan {
		t.Error("expected VerificationRan=false")
	}
	if reply.Bundle.Consistency.OverallVerdict != "degraded" {
		t.Errorf("expected verdict degraded, got %s", reply.Bundle.Consistency.OverallVerdict)
	}
}

func TestAuditSession_OpenSession(t *testing.T) {
	session := &execution.Session{
		SessionID: "session_20260324_130000",
		Status:    execution.SessionOpen,
		StartedAt: time.Now().UTC(),
		Config: execution.SessionConfigSnapshot{
			VenueType: "binance_spot",
			DryRun:    true,
			Segments:  []string{"spot"},
		},
		Activation: execution.SessionActivationSnapshot{
			Adapter:   "paper",
			Effective: "paper",
		},
	}

	uc := NewAuditSessionUseCase(
		&stubSessionReader{session: session},
		nil, nil, nil,
	)

	reply, prob := uc.Execute(context.Background(), SessionAuditQuery{SessionID: session.SessionID})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	// Open session has no counters, activity should come from lifecycle (empty).
	if reply.Bundle.OrderActivity.FromSessionCounters {
		t.Error("expected FromSessionCounters=false for open session")
	}
}

func TestAuditSession_NilSessionReader(t *testing.T) {
	uc := NewAuditSessionUseCase(nil, nil, nil, nil)
	_, prob := uc.Execute(context.Background(), SessionAuditQuery{SessionID: "session_x"})
	if prob == nil {
		t.Fatal("expected error for nil session reader")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}
