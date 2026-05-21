package executionclient

import (
	"context"
	"testing"
	"time"

	"internal/domain/execution"
	"internal/shared/problem"
)

// stubListSessions returns a fixed list of sessions.
type stubListSessions struct {
	sessions []execution.Session
	prob     *problem.Problem
}

func (s *stubListSessions) Execute(_ context.Context, _ SessionListQuery) (SessionListReply, *problem.Problem) {
	if s.prob != nil {
		return SessionListReply{}, s.prob
	}
	return SessionListReply{Sessions: s.sessions, Total: len(s.sessions)}, nil
}

// stubAuditExecutor returns a fixed audit bundle per session.
type stubAuditExecutor struct {
	bundles map[string]execution.SessionAuditBundle
	prob    *problem.Problem
}

func (s *stubAuditExecutor) Execute(_ context.Context, q SessionAuditQuery) (SessionAuditReply, *problem.Problem) {
	if s.prob != nil {
		return SessionAuditReply{}, s.prob
	}
	if b, ok := s.bundles[q.SessionID]; ok {
		return SessionAuditReply{Bundle: b}, nil
	}
	return SessionAuditReply{}, problem.New(problem.NotFound, "session not found: "+q.SessionID)
}

func TestBatchAudit_ExplicitIDs(t *testing.T) {
	bundles := map[string]execution.SessionAuditBundle{
		"s1": {Session: execution.Session{SessionID: "s1"}, Consistency: execution.AuditConsistency{OverallVerdict: "consistent"}},
		"s2": {Session: execution.Session{SessionID: "s2"}, Consistency: execution.AuditConsistency{OverallVerdict: "degraded"}},
	}

	uc := NewBatchAuditSessionUseCase(
		&stubListSessions{},
		&stubAuditExecutor{bundles: bundles},
	)

	reply, prob := uc.Execute(context.Background(), SessionBatchAuditQuery{
		SessionIDs: []string{"s1", "s2"},
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(reply.Result.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(reply.Result.Entries))
	}
	if reply.Result.Summary.Consistent != 1 {
		t.Errorf("expected 1 consistent, got %d", reply.Result.Summary.Consistent)
	}
	if reply.Result.Summary.Degraded != 1 {
		t.Errorf("expected 1 degraded, got %d", reply.Result.Summary.Degraded)
	}
}

func TestBatchAudit_AutoResolveTerminal(t *testing.T) {
	now := time.Now().UTC()
	closed := now

	sessions := []execution.Session{
		{SessionID: "closed1", Status: execution.SessionClosed, StartedAt: now, ClosedAt: &closed, Config: execution.SessionConfigSnapshot{VenueType: "test"}},
		{SessionID: "open1", Status: execution.SessionOpen, StartedAt: now, Config: execution.SessionConfigSnapshot{VenueType: "test"}},
		{SessionID: "halted1", Status: execution.SessionHalted, HaltReason: "test", StartedAt: now, ClosedAt: &closed, Config: execution.SessionConfigSnapshot{VenueType: "test"}},
	}

	bundles := map[string]execution.SessionAuditBundle{
		"closed1": {Session: sessions[0], Consistency: execution.AuditConsistency{OverallVerdict: "consistent"}},
		"halted1": {Session: sessions[2], Consistency: execution.AuditConsistency{OverallVerdict: "inconsistent"}},
	}

	uc := NewBatchAuditSessionUseCase(
		&stubListSessions{sessions: sessions},
		&stubAuditExecutor{bundles: bundles},
	)

	reply, prob := uc.Execute(context.Background(), SessionBatchAuditQuery{})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	// Should only include terminal sessions (closed1, halted1), not open1.
	if len(reply.Result.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(reply.Result.Entries))
	}
	if reply.Result.Summary.TotalSessions != 2 {
		t.Errorf("expected 2 total, got %d", reply.Result.Summary.TotalSessions)
	}
}

func TestBatchAudit_StatusFilter(t *testing.T) {
	now := time.Now().UTC()
	closed := now

	sessions := []execution.Session{
		{SessionID: "closed1", Status: execution.SessionClosed, StartedAt: now, ClosedAt: &closed, Config: execution.SessionConfigSnapshot{VenueType: "test"}},
		{SessionID: "halted1", Status: execution.SessionHalted, HaltReason: "test", StartedAt: now, ClosedAt: &closed, Config: execution.SessionConfigSnapshot{VenueType: "test"}},
	}

	bundles := map[string]execution.SessionAuditBundle{
		"halted1": {Session: sessions[1], Consistency: execution.AuditConsistency{OverallVerdict: "inconsistent"}},
	}

	uc := NewBatchAuditSessionUseCase(
		&stubListSessions{sessions: sessions},
		&stubAuditExecutor{bundles: bundles},
	)

	reply, prob := uc.Execute(context.Background(), SessionBatchAuditQuery{StatusFilter: "halted"})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(reply.Result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(reply.Result.Entries))
	}
	if reply.Result.Entries[0].SessionID != "halted1" {
		t.Errorf("expected halted1, got %s", reply.Result.Entries[0].SessionID)
	}
}

func TestBatchAudit_PartialFailure(t *testing.T) {
	bundles := map[string]execution.SessionAuditBundle{
		"s1": {Session: execution.Session{SessionID: "s1"}, Consistency: execution.AuditConsistency{OverallVerdict: "consistent"}},
	}

	uc := NewBatchAuditSessionUseCase(
		&stubListSessions{},
		&stubAuditExecutor{bundles: bundles},
	)

	reply, prob := uc.Execute(context.Background(), SessionBatchAuditQuery{
		SessionIDs: []string{"s1", "missing"},
	})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if len(reply.Result.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(reply.Result.Entries))
	}
	if reply.Result.Summary.Consistent != 1 {
		t.Errorf("expected 1 consistent, got %d", reply.Result.Summary.Consistent)
	}
	if reply.Result.Summary.Errored != 1 {
		t.Errorf("expected 1 errored, got %d", reply.Result.Summary.Errored)
	}
	if reply.Result.Entries[1].Error == "" {
		t.Error("expected error for missing session")
	}
}

func TestBatchAudit_UnavailableDeps(t *testing.T) {
	uc := NewBatchAuditSessionUseCase(nil, nil)
	_, prob := uc.Execute(context.Background(), SessionBatchAuditQuery{SessionIDs: []string{"s1"}})
	if prob == nil {
		t.Fatal("expected error for nil deps")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}
