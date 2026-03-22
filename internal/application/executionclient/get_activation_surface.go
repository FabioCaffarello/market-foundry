package executionclient

import (
	"context"

	"internal/shared/problem"
)

// activationSurfaceGateway is the local interface for querying the activation surface.
type activationSurfaceGateway interface {
	GetActivationSurface(context.Context, ActivationSurfaceQuery) (ActivationSurfaceReply, *problem.Problem)
}

// GetActivationSurfaceUseCase queries the store for the canonical activation surface.
type GetActivationSurfaceUseCase struct {
	gateway activationSurfaceGateway
}

func NewGetActivationSurfaceUseCase(gateway activationSurfaceGateway) *GetActivationSurfaceUseCase {
	return &GetActivationSurfaceUseCase{gateway: gateway}
}

func (uc *GetActivationSurfaceUseCase) Execute(ctx context.Context, query ActivationSurfaceQuery) (ActivationSurfaceReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return ActivationSurfaceReply{}, problem.New(problem.Unavailable, "activation surface gateway is unavailable")
	}
	return uc.gateway.GetActivationSurface(ctx, query)
}
