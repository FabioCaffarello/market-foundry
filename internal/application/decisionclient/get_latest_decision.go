package decisionclient

import (
	"context"

	"internal/shared/problem"
)

// decisionGateway is the local interface for querying decisions.
// This avoids an import cycle with the ports package.
type decisionGateway interface {
	GetLatestDecision(context.Context, DecisionLatestQuery) (DecisionLatestReply, *problem.Problem)
}

// GetLatestDecisionUseCase queries the store for the latest decision via the decision gateway.
type GetLatestDecisionUseCase struct {
	gateway decisionGateway
}

func NewGetLatestDecisionUseCase(gateway decisionGateway) *GetLatestDecisionUseCase {
	return &GetLatestDecisionUseCase{gateway: gateway}
}

func (uc *GetLatestDecisionUseCase) Execute(ctx context.Context, query DecisionLatestQuery) (DecisionLatestReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return DecisionLatestReply{}, problem.New(problem.Unavailable, "decision gateway is unavailable")
	}

	if query.Type == "" {
		return DecisionLatestReply{}, problem.New(problem.InvalidArgument, "decision type is required")
	}
	if query.Source == "" {
		return DecisionLatestReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Instrument.IsZero() {
		return DecisionLatestReply{}, problem.New(problem.InvalidArgument, "instrument is required")
	}
	if query.Timeframe <= 0 {
		return DecisionLatestReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}

	return uc.gateway.GetLatestDecision(ctx, query)
}
