package strategyclient

import (
	"context"

	"internal/shared/problem"
)

// strategyGateway is the local interface for querying strategies.
// This avoids an import cycle with the ports package.
type strategyGateway interface {
	GetLatestStrategy(context.Context, StrategyLatestQuery) (StrategyLatestReply, *problem.Problem)
}

// GetLatestStrategyUseCase queries the store for the latest strategy via the strategy gateway.
type GetLatestStrategyUseCase struct {
	gateway strategyGateway
}

func NewGetLatestStrategyUseCase(gateway strategyGateway) *GetLatestStrategyUseCase {
	return &GetLatestStrategyUseCase{gateway: gateway}
}

func (uc *GetLatestStrategyUseCase) Execute(ctx context.Context, query StrategyLatestQuery) (StrategyLatestReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return StrategyLatestReply{}, problem.New(problem.Unavailable, "strategy service is unavailable")
	}

	if query.Type == "" {
		return StrategyLatestReply{}, problem.New(problem.InvalidArgument, "strategy type is required")
	}
	if query.Source == "" {
		return StrategyLatestReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return StrategyLatestReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return StrategyLatestReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}

	return uc.gateway.GetLatestStrategy(ctx, query)
}
