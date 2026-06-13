package insightsclient

import (
	"context"

	"internal/shared/problem"
)

// volumeProfileGateway is the local read interface (avoids an import
// cycle with the ports package). Satisfied by the natsinsights KV
// reader.
type volumeProfileGateway interface {
	GetLatestVolumeProfile(context.Context, VolumeProfileLatestQuery) (VolumeProfileLatestReply, *problem.Problem)
}

// GetLatestVolumeProfileUseCase serves the latest volume profile for
// a partition. Read-only (ADR-0027): no directives, no writes.
type GetLatestVolumeProfileUseCase struct {
	gateway volumeProfileGateway
}

func NewGetLatestVolumeProfileUseCase(gateway volumeProfileGateway) *GetLatestVolumeProfileUseCase {
	return &GetLatestVolumeProfileUseCase{gateway: gateway}
}

func (u *GetLatestVolumeProfileUseCase) Execute(ctx context.Context, query VolumeProfileLatestQuery) (VolumeProfileLatestReply, *problem.Problem) {
	if u == nil || u.gateway == nil {
		return VolumeProfileLatestReply{}, problem.New(problem.Unavailable, "insights gateway is unavailable")
	}
	if query.Source == "" {
		return VolumeProfileLatestReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Instrument.IsZero() {
		return VolumeProfileLatestReply{}, problem.New(problem.InvalidArgument, "instrument is required")
	}
	if query.Timeframe <= 0 {
		return VolumeProfileLatestReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	return u.gateway.GetLatestVolumeProfile(ctx, query)
}
