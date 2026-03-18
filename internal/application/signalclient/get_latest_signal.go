package signalclient

import (
	"context"

	"internal/shared/problem"
)

// signalGateway is the local interface for querying signals.
// This avoids an import cycle with the ports package.
type signalGateway interface {
	GetLatestSignal(context.Context, SignalLatestQuery) (SignalLatestReply, *problem.Problem)
}

// GetLatestSignalUseCase queries the store for the latest signal via the signal gateway.
type GetLatestSignalUseCase struct {
	gateway signalGateway
}

func NewGetLatestSignalUseCase(gateway signalGateway) *GetLatestSignalUseCase {
	return &GetLatestSignalUseCase{gateway: gateway}
}

func (uc *GetLatestSignalUseCase) Execute(ctx context.Context, query SignalLatestQuery) (SignalLatestReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return SignalLatestReply{}, problem.New(problem.Unavailable, "signal service is unavailable")
	}

	if query.Type == "" {
		return SignalLatestReply{}, problem.New(problem.InvalidArgument, "signal type is required")
	}
	if query.Source == "" {
		return SignalLatestReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return SignalLatestReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return SignalLatestReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}

	return uc.gateway.GetLatestSignal(ctx, query)
}
