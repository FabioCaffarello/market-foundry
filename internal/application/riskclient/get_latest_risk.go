package riskclient

import (
	"context"

	"internal/shared/problem"
)

// riskGateway is the local interface for querying risk assessments.
// This avoids an import cycle with the ports package.
type riskGateway interface {
	GetLatestRisk(context.Context, RiskLatestQuery) (RiskLatestReply, *problem.Problem)
}

// GetLatestRiskUseCase queries the store for the latest risk assessment via the risk gateway.
type GetLatestRiskUseCase struct {
	gateway riskGateway
}

func NewGetLatestRiskUseCase(gateway riskGateway) *GetLatestRiskUseCase {
	return &GetLatestRiskUseCase{gateway: gateway}
}

func (uc *GetLatestRiskUseCase) Execute(ctx context.Context, query RiskLatestQuery) (RiskLatestReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return RiskLatestReply{}, problem.New(problem.Unavailable, "risk gateway is unavailable")
	}

	if query.Type == "" {
		return RiskLatestReply{}, problem.New(problem.InvalidArgument, "risk type is required")
	}
	if query.Source == "" {
		return RiskLatestReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Instrument.IsZero() {
		return RiskLatestReply{}, problem.New(problem.InvalidArgument, "instrument is required")
	}
	if query.Timeframe <= 0 {
		return RiskLatestReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}

	return uc.gateway.GetLatestRisk(ctx, query)
}
