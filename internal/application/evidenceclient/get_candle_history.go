package evidenceclient

import (
	"context"

	"internal/shared/problem"
)

const (
	defaultHistoryLimit = 10
	maxHistoryLimit     = 100
)

// candleHistoryGateway is the local interface for querying candle history.
type candleHistoryGateway interface {
	GetCandleHistory(context.Context, CandleHistoryQuery) (CandleHistoryReply, *problem.Problem)
}

// GetCandleHistoryUseCase queries the store for historical candles via the evidence gateway.
type GetCandleHistoryUseCase struct {
	gateway candleHistoryGateway
}

func NewGetCandleHistoryUseCase(gateway candleHistoryGateway) *GetCandleHistoryUseCase {
	return &GetCandleHistoryUseCase{gateway: gateway}
}

func (uc *GetCandleHistoryUseCase) Execute(ctx context.Context, query CandleHistoryQuery) (CandleHistoryReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return CandleHistoryReply{}, problem.New(problem.Unavailable, "evidence gateway is unavailable")
	}

	if query.Source == "" {
		return CandleHistoryReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return CandleHistoryReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return CandleHistoryReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}

	if query.Since < 0 {
		return CandleHistoryReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return CandleHistoryReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return CandleHistoryReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	if query.Limit <= 0 {
		query.Limit = defaultHistoryLimit
	}
	if query.Limit > maxHistoryLimit {
		query.Limit = maxHistoryLimit
	}

	return uc.gateway.GetCandleHistory(ctx, query)
}
