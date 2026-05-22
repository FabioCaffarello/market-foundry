package executionclient

import (
	"context"
	"time"

	"internal/domain/execution"
	"internal/shared/problem"
)

// sourceExplainGateway defines the ports needed to compose a source explanation.
type sourceExplainGateway interface {
	GetActivationSurface(context.Context, ActivationSurfaceQuery) (ActivationSurfaceReply, *problem.Problem)
	GetExecutionControl(context.Context, ExecutionControlQuery) (ExecutionControlReply, *problem.Problem)
	GetExecutionStatus(context.Context, ExecutionStatusQuery) (ExecutionStatusReply, *problem.Problem)
}

// SourcePathConfigProvider supplies the runtime configuration of the source-driven path.
// Implemented by the gateway binary's dependency injection.
type SourcePathConfigProvider interface {
	SourcePathConfig() execution.SourcePathConfig
}

// GetSourceExplanationUseCase composes the source-driven path explanation from
// existing activation, control, and execution status queries.
type GetSourceExplanationUseCase struct {
	gateway        sourceExplainGateway
	configProvider SourcePathConfigProvider
}

func NewGetSourceExplanationUseCase(gateway sourceExplainGateway, configProvider SourcePathConfigProvider) *GetSourceExplanationUseCase {
	return &GetSourceExplanationUseCase{gateway: gateway, configProvider: configProvider}
}

func (uc *GetSourceExplanationUseCase) Execute(ctx context.Context, query SourceExplainQuery) (SourceExplainReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return SourceExplainReply{}, problem.New(problem.Unavailable, "source explanation gateway is unavailable")
	}

	// Fetch activation surface.
	activationReply, prob := uc.gateway.GetActivationSurface(ctx, ActivationSurfaceQuery{})
	if prob != nil {
		return SourceExplainReply{}, prob
	}

	// Fetch gate status.
	controlReply, prob := uc.gateway.GetExecutionControl(ctx, ExecutionControlQuery{})
	if prob != nil {
		return SourceExplainReply{}, prob
	}

	// Fetch execution status (intent + result + propagation).
	var lastIntent, lastResult *execution.ExecutionIntent
	propagation := "none"
	if query.Source != "" && query.Symbol != "" && query.Timeframe > 0 {
		statusReply, prob := uc.gateway.GetExecutionStatus(ctx, ExecutionStatusQuery(query))
		if prob == nil {
			lastIntent = statusReply.Intent
			lastResult = statusReply.Result
			propagation = statusReply.Propagation
		}
	}

	// Compose config.
	var config execution.SourcePathConfig
	if uc.configProvider != nil {
		config = uc.configProvider.SourcePathConfig()
	}

	explanation := execution.SourcePathExplanation{
		SourcePath:   "strategy_consumer.mean_reversion_entry",
		StrategyType: "mean_reversion_entry",
		Activation:   activationReply.Surface,
		Gate:         controlReply.Gate,
		Config:       config,
		LastIntent:   lastIntent,
		LastResult:   lastResult,
		Propagation:  propagation,
		ObservedAt:   time.Now().UTC(),
	}

	return SourceExplainReply{Explanation: explanation}, nil
}
