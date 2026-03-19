package evidenceclient

import (
	"context"

	"internal/shared/problem"
)

// volumeGateway is the local interface for querying evidence.
// This avoids an import cycle with the ports package.
type volumeGateway interface {
	GetLatestVolume(context.Context, VolumeLatestQuery) (VolumeLatestReply, *problem.Problem)
}

type GetLatestVolumeUseCase struct {
	gateway volumeGateway
}

func NewGetLatestVolumeUseCase(gateway volumeGateway) *GetLatestVolumeUseCase {
	return &GetLatestVolumeUseCase{gateway: gateway}
}

func (u *GetLatestVolumeUseCase) Execute(ctx context.Context, query VolumeLatestQuery) (VolumeLatestReply, *problem.Problem) {
	if u == nil || u.gateway == nil {
		return VolumeLatestReply{}, problem.New(problem.Unavailable, "evidence gateway is unavailable")
	}
	if query.Source == "" {
		return VolumeLatestReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return VolumeLatestReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return VolumeLatestReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	return u.gateway.GetLatestVolume(ctx, query)
}
