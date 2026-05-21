package executionclient

import (
	"context"

	"internal/shared/problem"
)

// lifecycleListGateway is the local interface for querying the lifecycle list.
type lifecycleListGateway interface {
	GetLifecycleList(context.Context, LifecycleListQuery) (LifecycleListReply, *problem.Problem)
}

// GetLifecycleListUseCase queries the store for all tracked execution lifecycle entries.
// S454A: Exposes the S413 lifecycle list (previously NATS-only) through the gateway HTTP surface.
type GetLifecycleListUseCase struct {
	gateway lifecycleListGateway
}

func NewGetLifecycleListUseCase(gateway lifecycleListGateway) *GetLifecycleListUseCase {
	return &GetLifecycleListUseCase{gateway: gateway}
}

func (uc *GetLifecycleListUseCase) Execute(ctx context.Context, query LifecycleListQuery) (LifecycleListReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return LifecycleListReply{}, problem.New(problem.Unavailable, "lifecycle list gateway is unavailable")
	}

	return uc.gateway.GetLifecycleList(ctx, query)
}
