package insightsclient

import (
	"context"

	"internal/shared/problem"
)

// tpoProfileGateway is the local read interface (avoids an import cycle
// with the ports package). Satisfied by the natsinsights KV reader.
type tpoProfileGateway interface {
	GetLatestTPOProfile(context.Context, TPOProfileLatestQuery) (TPOProfileLatestReply, *problem.Problem)
}

// GetLatestTPOProfileUseCase serves the latest TPO profile for a
// partition. Read-only (ADR-0027): no directives, no writes.
type GetLatestTPOProfileUseCase struct {
	gateway tpoProfileGateway
}

func NewGetLatestTPOProfileUseCase(gateway tpoProfileGateway) *GetLatestTPOProfileUseCase {
	return &GetLatestTPOProfileUseCase{gateway: gateway}
}

func (u *GetLatestTPOProfileUseCase) Execute(ctx context.Context, query TPOProfileLatestQuery) (TPOProfileLatestReply, *problem.Problem) {
	if u == nil || u.gateway == nil {
		return TPOProfileLatestReply{}, problem.New(problem.Unavailable, "insights gateway is unavailable")
	}
	if query.Source == "" {
		return TPOProfileLatestReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Instrument.IsZero() {
		return TPOProfileLatestReply{}, problem.New(problem.InvalidArgument, "instrument is required")
	}
	if query.Timeframe <= 0 {
		return TPOProfileLatestReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	return u.gateway.GetLatestTPOProfile(ctx, query)
}
