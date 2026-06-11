package executionclient_test

import (
	"internal/domain/instrument"

	"context"
	"testing"

	"internal/application/executionclient"
	"internal/domain/execution"
	"internal/shared/problem"
)

// ── Test doubles ────────────────────────────────────────────────────────────

type stubSessionReader struct {
	session *execution.Session
}

func (s *stubSessionReader) Execute(_ context.Context, q executionclient.SessionGetQuery) (executionclient.SessionGetReply, *problem.Problem) {
	return executionclient.SessionGetReply{Session: s.session}, nil
}

type stubGateReader struct {
	status execution.GateStatus
}

func (s *stubGateReader) Execute(_ context.Context, _ executionclient.ExecutionControlQuery) (executionclient.ExecutionControlReply, *problem.Problem) {
	return executionclient.ExecutionControlReply{
		Gate: execution.ControlGate{Status: s.status},
	}, nil
}

type stubCHSummary struct {
	total int64
}

func (s *stubCHSummary) Summary(_ context.Context, _ instrument.CanonicalInstrument, _, _ int64) (int64, *problem.Problem) {
	return s.total, nil
}

type stubCHLister struct {
	rows []executionclient.VerifyCHListResult
}

func (s *stubCHLister) List(_ context.Context, _ instrument.CanonicalInstrument, _, _ string, _ int, _, _ int64) ([]executionclient.VerifyCHListResult, *problem.Problem) {
	return s.rows, nil
}

type stubConsistency struct {
	consistent bool
	evidence   map[string]any
}

func (s *stubConsistency) CheckConsistency(_ context.Context, _ string, _ instrument.CanonicalInstrument, _ int) (bool, map[string]any, *problem.Problem) {
	return s.consistent, s.evidence, nil
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestVerifySession_RequiresSessionID(t *testing.T) {
	uc := executionclient.NewVerifySessionUseCase(nil, nil, nil, nil, nil)
	_, prob := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{})
	if prob == nil {
		t.Fatal("expected problem for empty session_id")
	}
}

func TestVerifySession_AllNilDependencies(t *testing.T) {
	uc := executionclient.NewVerifySessionUseCase(nil, nil, nil, nil, nil)
	reply, prob := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{SessionID: "session_20260324_120000"})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	report := reply.Report
	if report.SessionID != "session_20260324_120000" {
		t.Errorf("session_id: expected session_20260324_120000, got %s", report.SessionID)
	}
	if len(report.Checks) != 9 {
		t.Fatalf("expected 9 checks, got %d", len(report.Checks))
	}

	// All checks that depend on nil dependencies should be skip or manual.
	for _, c := range report.Checks {
		switch c.Verdict {
		case execution.VerdictSkip, execution.VerdictManual, execution.VerdictPass:
			// OK — PO-5, PO-6 always pass; PO-2 is manual; rest skip
		default:
			t.Errorf("check %s: unexpected verdict %s with nil deps", c.CheckID, c.Verdict)
		}
	}
}

func TestVerifySession_GateHalted(t *testing.T) {
	uc := executionclient.NewVerifySessionUseCase(
		nil,
		&stubGateReader{status: execution.GateHalted},
		nil, nil, nil,
	)
	reply, _ := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{SessionID: "test"})

	po1 := reply.Report.Checks[0]
	if po1.CheckID != execution.POCheckGateHalted {
		t.Errorf("first check should be PO-1, got %s", po1.CheckID)
	}
	if po1.Verdict != execution.VerdictPass {
		t.Errorf("PO-1 should pass when gate halted, got %s", po1.Verdict)
	}
}

func TestVerifySession_GateActive(t *testing.T) {
	uc := executionclient.NewVerifySessionUseCase(
		nil,
		&stubGateReader{status: execution.GateActive},
		nil, nil, nil,
	)
	reply, _ := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{SessionID: "test"})

	po1 := reply.Report.Checks[0]
	if po1.Verdict != execution.VerdictWarn {
		t.Errorf("PO-1 should warn when gate active, got %s", po1.Verdict)
	}
}

func TestVerifySession_IntentRecordsFound(t *testing.T) {
	uc := executionclient.NewVerifySessionUseCase(
		nil, nil,
		&stubCHSummary{total: 5},
		nil, nil,
	)
	reply, _ := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{SessionID: "test"})

	po3 := reply.Report.Checks[2]
	if po3.CheckID != execution.POCheckIntentRecords {
		t.Errorf("third check should be PO-3, got %s", po3.CheckID)
	}
	if po3.Verdict != execution.VerdictPass {
		t.Errorf("PO-3 should pass with records, got %s", po3.Verdict)
	}
}

func TestVerifySession_ScopeContainmentPass(t *testing.T) {
	uc := executionclient.NewVerifySessionUseCase(
		nil, nil, nil,
		&stubCHLister{rows: []executionclient.VerifyCHListResult{
			{Symbol: "btcusdt", Type: "venue_market_order"},
		}},
		nil,
	)
	reply, _ := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{SessionID: "test"})

	po9 := reply.Report.Checks[8]
	if po9.CheckID != execution.POCheckScopeContainment {
		t.Errorf("ninth check should be PO-9, got %s", po9.CheckID)
	}
	if po9.Verdict != execution.VerdictPass {
		t.Errorf("PO-9 should pass with only BTCUSDT, got %s", po9.Verdict)
	}
}

func TestVerifySession_ScopeContainmentFail(t *testing.T) {
	uc := executionclient.NewVerifySessionUseCase(
		nil, nil, nil,
		&stubCHLister{rows: []executionclient.VerifyCHListResult{
			{Symbol: "btcusdt", Type: "venue_market_order"},
			{Symbol: "ETHUSDT", Type: "venue_market_order"},
		}},
		nil,
	)
	reply, _ := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{SessionID: "test"})

	po9 := reply.Report.Checks[8]
	if po9.Verdict != execution.VerdictFail {
		t.Errorf("PO-9 should fail with non-BTCUSDT, got %s", po9.Verdict)
	}
}

func TestVerifySession_FeeFieldsPass(t *testing.T) {
	uc := executionclient.NewVerifySessionUseCase(
		nil, nil, nil,
		&stubCHLister{rows: []executionclient.VerifyCHListResult{
			{
				Symbol: "btcusdt", Status: "filled",
				Fills: []execution.FillRecord{
					{Fee: "0.001", FeeAsset: "BNB"},
				},
			},
		}},
		nil,
	)
	reply, _ := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{SessionID: "test"})

	po7 := reply.Report.Checks[6]
	if po7.CheckID != execution.POCheckFeeFields {
		t.Errorf("seventh check should be PO-7, got %s", po7.CheckID)
	}
	if po7.Verdict != execution.VerdictPass {
		t.Errorf("PO-7 should pass with fee fields, got %s", po7.Verdict)
	}
}

func TestVerifySession_ConsistencyPass(t *testing.T) {
	uc := executionclient.NewVerifySessionUseCase(
		nil, nil, nil, nil,
		&stubConsistency{consistent: true, evidence: map[string]any{"detail": "all match"}},
	)
	reply, _ := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{SessionID: "test"})

	po8 := reply.Report.Checks[7]
	if po8.CheckID != execution.POCheckLifecycleConsist {
		t.Errorf("eighth check should be PO-8, got %s", po8.CheckID)
	}
	if po8.Verdict != execution.VerdictPass {
		t.Errorf("PO-8 should pass when consistent, got %s", po8.Verdict)
	}
}

func TestVerifySession_FullReport(t *testing.T) {
	uc := executionclient.NewVerifySessionUseCase(
		&stubSessionReader{session: &execution.Session{Operator: "operator1"}},
		&stubGateReader{status: execution.GateHalted},
		&stubCHSummary{total: 3},
		&stubCHLister{rows: []executionclient.VerifyCHListResult{
			{Symbol: "BTCUSDT", Type: "venue_market_order", Status: "filled",
				Fills: []execution.FillRecord{{Fee: "0.001", FeeAsset: "BNB"}}},
		}},
		&stubConsistency{consistent: true, evidence: map[string]any{}},
	)

	reply, prob := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{SessionID: "session_20260324_120000"})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	report := reply.Report
	if report.Operator != "operator1" {
		t.Errorf("operator: expected operator1, got %s", report.Operator)
	}
	if report.Summary.Total != 9 {
		t.Errorf("total checks: expected 9, got %d", report.Summary.Total)
	}
	if report.Summary.Failed != 0 {
		t.Errorf("expected 0 failures, got %d", report.Summary.Failed)
	}
	// PO-2 is manual, rest should be pass/skip.
	if report.Summary.Manual != 1 {
		t.Errorf("expected 1 manual check, got %d", report.Summary.Manual)
	}
	if report.Summary.Automated != 8 {
		t.Errorf("expected 8 automated checks, got %d", report.Summary.Automated)
	}
}
