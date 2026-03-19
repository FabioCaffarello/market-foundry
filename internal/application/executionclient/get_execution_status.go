package executionclient

import (
	"context"

	"internal/shared/problem"
)

// executionStatusGateway is the local interface for querying composite execution status.
type executionStatusGateway interface {
	GetExecutionStatus(context.Context, ExecutionStatusQuery) (ExecutionStatusReply, *problem.Problem)
}

// GetExecutionStatusUseCase queries the store for the composite execution status:
// intent (paper_order) + result (venue_market_order) + control gate + derived propagation.
type GetExecutionStatusUseCase struct {
	gateway executionStatusGateway
}

func NewGetExecutionStatusUseCase(gateway executionStatusGateway) *GetExecutionStatusUseCase {
	return &GetExecutionStatusUseCase{gateway: gateway}
}

func (uc *GetExecutionStatusUseCase) Execute(ctx context.Context, query ExecutionStatusQuery) (ExecutionStatusReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return ExecutionStatusReply{}, problem.New(problem.Unavailable, "execution status service is unavailable")
	}

	if query.Source == "" {
		return ExecutionStatusReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return ExecutionStatusReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return ExecutionStatusReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}

	return uc.gateway.GetExecutionStatus(ctx, query)
}
