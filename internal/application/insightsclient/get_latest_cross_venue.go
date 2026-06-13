package insightsclient

import (
	"context"

	"internal/shared/problem"
)

// crossVenueGateway is the local read interface (avoids an import cycle
// with the ports package). Satisfied by the natsinsights KV reader.
type crossVenueGateway interface {
	GetLatestCrossVenue(context.Context, CrossVenueLatestQuery) (CrossVenueLatestReply, *problem.Problem)
}

// GetLatestCrossVenueUseCase serves the latest cross-venue snapshot for
// a partition. Read-only (ADR-0027): no directives, no writes.
type GetLatestCrossVenueUseCase struct {
	gateway crossVenueGateway
}

func NewGetLatestCrossVenueUseCase(gateway crossVenueGateway) *GetLatestCrossVenueUseCase {
	return &GetLatestCrossVenueUseCase{gateway: gateway}
}

func (u *GetLatestCrossVenueUseCase) Execute(ctx context.Context, query CrossVenueLatestQuery) (CrossVenueLatestReply, *problem.Problem) {
	if u == nil || u.gateway == nil {
		return CrossVenueLatestReply{}, problem.New(problem.Unavailable, "insights gateway is unavailable")
	}
	if query.Instrument.IsZero() {
		return CrossVenueLatestReply{}, problem.New(problem.InvalidArgument, "instrument is required")
	}
	if query.Timeframe <= 0 {
		return CrossVenueLatestReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	return u.gateway.GetLatestCrossVenue(ctx, query)
}
