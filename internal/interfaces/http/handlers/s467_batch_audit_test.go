package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"internal/application/executionclient"
	"internal/domain/execution"
	"internal/interfaces/http/handlers"
	"internal/shared/problem"
)

type mockBatchAuditUseCase struct {
	result executionclient.SessionBatchAuditReply
	prob   *problem.Problem
}

func (m *mockBatchAuditUseCase) Execute(_ context.Context, _ executionclient.SessionBatchAuditQuery) (executionclient.SessionBatchAuditReply, *problem.Problem) {
	if m.prob != nil {
		return executionclient.SessionBatchAuditReply{}, m.prob
	}
	return m.result, nil
}

type mockGetSession struct{}
func (m *mockGetSession) Execute(_ context.Context, q executionclient.SessionGetQuery) (executionclient.SessionGetReply, *problem.Problem) {
	return executionclient.SessionGetReply{}, nil
}

type mockListSessions struct{}
func (m *mockListSessions) Execute(_ context.Context, _ executionclient.SessionListQuery) (executionclient.SessionListReply, *problem.Problem) {
	return executionclient.SessionListReply{}, nil
}

type mockVerifySession struct{}
func (m *mockVerifySession) Execute(_ context.Context, _ executionclient.SessionVerifyQuery) (executionclient.SessionVerifyReply, *problem.Problem) {
	return executionclient.SessionVerifyReply{}, nil
}

type mockAuditSession struct{}
func (m *mockAuditSession) Execute(_ context.Context, _ executionclient.SessionAuditQuery) (executionclient.SessionAuditReply, *problem.Problem) {
	return executionclient.SessionAuditReply{}, nil
}

func TestS467_BatchAudit_Returns200(t *testing.T) {
	batchUC := &mockBatchAuditUseCase{
		result: executionclient.SessionBatchAuditReply{
			Result: execution.BatchAuditResult{
				Entries: []execution.BatchAuditEntry{
					{SessionID: "s1", Bundle: &execution.SessionAuditBundle{Consistency: execution.AuditConsistency{OverallVerdict: "consistent"}}},
				},
				Summary: execution.BatchAuditSummary{TotalSessions: 1, Consistent: 1},
			},
		},
	}

	handler := handlers.NewSessionWebHandler(&mockGetSession{}, &mockListSessions{}, &mockVerifySession{}, &mockAuditSession{}, batchUC, nil)

	req := httptest.NewRequest(http.MethodGet, "/session-batch-audit?status=closed", nil)
	rec := httptest.NewRecorder()
	handler.BatchAuditSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestS467_BatchAudit_NilUseCase(t *testing.T) {
	handler := handlers.NewSessionWebHandler(&mockGetSession{}, &mockListSessions{}, &mockVerifySession{}, &mockAuditSession{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/session-batch-audit", nil)
	rec := httptest.NewRecorder()
	handler.BatchAuditSessions(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestS467_BatchAudit_WithIDsParam(t *testing.T) {
	batchUC := &mockBatchAuditUseCase{
		result: executionclient.SessionBatchAuditReply{
			Result: execution.BatchAuditResult{
				Entries: []execution.BatchAuditEntry{
					{SessionID: "s1", Bundle: &execution.SessionAuditBundle{Consistency: execution.AuditConsistency{OverallVerdict: "consistent"}}},
					{SessionID: "s2", Bundle: &execution.SessionAuditBundle{Consistency: execution.AuditConsistency{OverallVerdict: "degraded"}}},
				},
				Summary: execution.BatchAuditSummary{TotalSessions: 2, Consistent: 1, Degraded: 1},
			},
		},
	}

	handler := handlers.NewSessionWebHandler(&mockGetSession{}, &mockListSessions{}, &mockVerifySession{}, &mockAuditSession{}, batchUC, nil)

	req := httptest.NewRequest(http.MethodGet, "/session-batch-audit?ids=s1,s2", nil)
	rec := httptest.NewRecorder()
	handler.BatchAuditSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
