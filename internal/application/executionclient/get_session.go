package executionclient

import (
	"context"

	"internal/shared/problem"
)

// sessionGateway is the local interface for querying session metadata.
// This avoids an import cycle with the ports package.
type sessionGateway interface {
	GetSession(context.Context, SessionGetQuery) (SessionGetReply, *problem.Problem)
	ListSessions(context.Context, SessionListQuery) (SessionListReply, *problem.Problem)
}

// GetSessionUseCase queries the store for a session record by ID.
type GetSessionUseCase struct {
	gateway sessionGateway
}

func NewGetSessionUseCase(gateway sessionGateway) *GetSessionUseCase {
	return &GetSessionUseCase{gateway: gateway}
}

func (uc *GetSessionUseCase) Execute(ctx context.Context, query SessionGetQuery) (SessionGetReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return SessionGetReply{}, problem.New(problem.Unavailable, "session gateway is unavailable")
	}
	if query.SessionID == "" {
		return SessionGetReply{}, problem.New(problem.InvalidArgument, "session_id is required")
	}
	return uc.gateway.GetSession(ctx, query)
}

// ListSessionsUseCase queries the store for all session records.
type ListSessionsUseCase struct {
	gateway sessionGateway
}

func NewListSessionsUseCase(gateway sessionGateway) *ListSessionsUseCase {
	return &ListSessionsUseCase{gateway: gateway}
}

func (uc *ListSessionsUseCase) Execute(ctx context.Context, query SessionListQuery) (SessionListReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return SessionListReply{}, problem.New(problem.Unavailable, "session gateway is unavailable")
	}
	return uc.gateway.ListSessions(ctx, query)
}
