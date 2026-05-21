package executionclient

import (
	"context"
	"time"

	"internal/domain/execution"
	"internal/shared/problem"
)

// BatchAuditSessionUseCase audits multiple sessions in a single call.
//
// S467: Reduces operational friction by allowing operators to review all
// terminal sessions (or a filtered subset) without N individual audit calls.
// Each session is audited independently; failures are captured per-entry
// rather than aborting the entire batch.
type BatchAuditSessionUseCase struct {
	listSessions listSessionsExecutor
	auditSession auditSessionExecutor
}

type listSessionsExecutor interface {
	Execute(context.Context, SessionListQuery) (SessionListReply, *problem.Problem)
}

type auditSessionExecutor interface {
	Execute(context.Context, SessionAuditQuery) (SessionAuditReply, *problem.Problem)
}

func NewBatchAuditSessionUseCase(
	listSessions listSessionsExecutor,
	auditSession auditSessionExecutor,
) *BatchAuditSessionUseCase {
	return &BatchAuditSessionUseCase{
		listSessions: listSessions,
		auditSession: auditSession,
	}
}

// BatchAuditMaxSessions caps the number of sessions in a single batch to
// prevent unbounded resource usage. Operators can narrow the set with
// explicit session IDs or a status filter.
const BatchAuditMaxSessions = 50

func (uc *BatchAuditSessionUseCase) Execute(ctx context.Context, query SessionBatchAuditQuery) (SessionBatchAuditReply, *problem.Problem) {
	if uc.listSessions == nil || uc.auditSession == nil {
		return SessionBatchAuditReply{}, problem.New(problem.Unavailable, "batch audit dependencies unavailable")
	}

	start := time.Now()

	sessionIDs, prob := uc.resolveSessionIDs(ctx, query)
	if prob != nil {
		return SessionBatchAuditReply{}, prob
	}

	if len(sessionIDs) > BatchAuditMaxSessions {
		sessionIDs = sessionIDs[:BatchAuditMaxSessions]
	}

	entries := make([]execution.BatchAuditEntry, 0, len(sessionIDs))
	for _, sid := range sessionIDs {
		entry := execution.BatchAuditEntry{SessionID: sid}
		reply, auditProb := uc.auditSession.Execute(ctx, SessionAuditQuery{SessionID: sid})
		if auditProb != nil {
			entry.Error = auditProb.Message
		} else {
			bundle := reply.Bundle
			entry.Bundle = &bundle
		}
		entries = append(entries, entry)
	}

	result := execution.BatchAuditResult{
		Entries:     entries,
		Summary:     execution.ComputeBatchSummary(entries),
		AssembledAt: start.UTC(),
		AssemblyMs:  time.Since(start).Milliseconds(),
	}

	return SessionBatchAuditReply{Result: result}, nil
}

func (uc *BatchAuditSessionUseCase) resolveSessionIDs(ctx context.Context, query SessionBatchAuditQuery) ([]string, *problem.Problem) {
	if len(query.SessionIDs) > 0 {
		return query.SessionIDs, nil
	}

	reply, prob := uc.listSessions.Execute(ctx, SessionListQuery{})
	if prob != nil {
		return nil, problem.New(problem.Unavailable, "failed to list sessions: "+prob.Message)
	}

	ids := make([]string, 0, len(reply.Sessions))
	for _, s := range reply.Sessions {
		if query.StatusFilter != "" && string(s.Status) != query.StatusFilter {
			continue
		}
		// Default: only terminal sessions when no explicit IDs provided.
		if query.StatusFilter == "" && !s.Status.IsTerminal() {
			continue
		}
		ids = append(ids, s.SessionID)
	}
	return ids, nil
}
