package evidenceclient

import (
	"context"

	"internal/shared/problem"
)

// evidenceGateway is the local interface for querying evidence.
// This avoids an import cycle with the ports package.
type evidenceGateway interface {
	GetLatestCandle(context.Context, CandleLatestQuery) (CandleLatestReply, *problem.Problem)
}

// GetLatestCandleUseCase queries the store for the latest sampled candle via the evidence gateway.
type GetLatestCandleUseCase struct {
	gateway evidenceGateway
}

func NewGetLatestCandleUseCase(gateway evidenceGateway) *GetLatestCandleUseCase {
	return &GetLatestCandleUseCase{gateway: gateway}
}

func (uc *GetLatestCandleUseCase) Execute(ctx context.Context, query CandleLatestQuery) (CandleLatestReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return CandleLatestReply{}, problem.New(problem.Unavailable, "evidence gateway is unavailable")
	}

	if query.Source == "" {
		return CandleLatestReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return CandleLatestReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return CandleLatestReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}

	return uc.gateway.GetLatestCandle(ctx, query)
}
