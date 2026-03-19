package executionclient

import (
	"context"

	"internal/shared/problem"
)

// executionControlGateway is the local interface for querying execution control state.
type executionControlGateway interface {
	GetExecutionControl(context.Context, ExecutionControlQuery) (ExecutionControlReply, *problem.Problem)
	SetExecutionControl(context.Context, SetExecutionControlCommand) (ExecutionControlReply, *problem.Problem)
}

// GetExecutionControlUseCase queries the store for the current execution control gate.
type GetExecutionControlUseCase struct {
	gateway executionControlGateway
}

func NewGetExecutionControlUseCase(gateway executionControlGateway) *GetExecutionControlUseCase {
	return &GetExecutionControlUseCase{gateway: gateway}
}

func (uc *GetExecutionControlUseCase) Execute(ctx context.Context, query ExecutionControlQuery) (ExecutionControlReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return ExecutionControlReply{}, problem.New(problem.Unavailable, "execution control gateway is unavailable")
	}
	return uc.gateway.GetExecutionControl(ctx, query)
}

// SetExecutionControlUseCase updates the execution control gate via the store.
type SetExecutionControlUseCase struct {
	gateway executionControlGateway
}

func NewSetExecutionControlUseCase(gateway executionControlGateway) *SetExecutionControlUseCase {
	return &SetExecutionControlUseCase{gateway: gateway}
}

func (uc *SetExecutionControlUseCase) Execute(ctx context.Context, cmd SetExecutionControlCommand) (ExecutionControlReply, *problem.Problem) {
	if uc == nil || uc.gateway == nil {
		return ExecutionControlReply{}, problem.New(problem.Unavailable, "execution control gateway is unavailable")
	}

	if cmd.Status != "active" && cmd.Status != "halted" {
		return ExecutionControlReply{}, problem.New(problem.InvalidArgument, "status must be 'active' or 'halted'")
	}

	return uc.gateway.SetExecutionControl(ctx, cmd)
}
