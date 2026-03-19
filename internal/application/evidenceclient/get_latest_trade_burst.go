package evidenceclient

import (
	"context"

	"internal/shared/problem"
)

// tradeBurstGateway is the local interface for querying evidence.
// This avoids an import cycle with the ports package.
type tradeBurstGateway interface {
	GetLatestTradeBurst(context.Context, TradeBurstLatestQuery) (TradeBurstLatestReply, *problem.Problem)
}

// GetLatestTradeBurstUseCase queries the store for the latest sampled trade burst via the evidence gateway.
type GetLatestTradeBurstUseCase struct {
	gateway tradeBurstGateway
}

func NewGetLatestTradeBurstUseCase(gateway tradeBurstGateway) *GetLatestTradeBurstUseCase {
	return &GetLatestTradeBurstUseCase{gateway: gateway}
}

func (uc *GetLatestTradeBurstUseCase) Execute(ctx context.Context, query TradeBurstLatestQuery) (TradeBurstLatestReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return TradeBurstLatestReply{}, problem.New(problem.Unavailable, "evidence gateway is unavailable")
	}

	if query.Source == "" {
		return TradeBurstLatestReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return TradeBurstLatestReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return TradeBurstLatestReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}

	return uc.gateway.GetLatestTradeBurst(ctx, query)
}
