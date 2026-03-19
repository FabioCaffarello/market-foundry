package executionclient

import (
	"context"

	"internal/shared/problem"
)

// executionGateway is the local interface for querying execution intents.
// This avoids an import cycle with the ports package.
type executionGateway interface {
	GetLatestExecution(context.Context, ExecutionLatestQuery) (ExecutionLatestReply, *problem.Problem)
}

// GetLatestExecutionUseCase queries the store for the latest execution intent via the execution gateway.
type GetLatestExecutionUseCase struct {
	gateway executionGateway
}

func NewGetLatestExecutionUseCase(gateway executionGateway) *GetLatestExecutionUseCase {
	return &GetLatestExecutionUseCase{gateway: gateway}
}

func (uc *GetLatestExecutionUseCase) Execute(ctx context.Context, query ExecutionLatestQuery) (ExecutionLatestReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return ExecutionLatestReply{}, problem.New(problem.Unavailable, "execution service is unavailable")
	}

	if query.Type == "" {
		return ExecutionLatestReply{}, problem.New(problem.InvalidArgument, "execution type is required")
	}
	if query.Source == "" {
		return ExecutionLatestReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return ExecutionLatestReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return ExecutionLatestReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}

	return uc.gateway.GetLatestExecution(ctx, query)
}
