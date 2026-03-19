package usecase

import (
	"context"

	"internal/shared/problem"
)

// GatewayFunc is the function signature for a single gateway operation.
type GatewayFunc[In, Out any] func(context.Context, In) (Out, *problem.Problem)

// Normalizable describes a command/query that supports normalize-then-validate.
type Normalizable[T any] interface {
	Normalize() T
	Validate() *problem.Problem
}

// CommandUseCase wraps a gateway function with nil safety, input normalization,
// and validation. Use this for operations whose input type implements Normalizable.
type CommandUseCase[Cmd Normalizable[Cmd], Reply any] struct {
	fn   GatewayFunc[Cmd, Reply]
	name string
}

// NewCommand creates a CommandUseCase that normalizes and validates before delegating.
func NewCommand[Cmd Normalizable[Cmd], Reply any](fn GatewayFunc[Cmd, Reply], name string) *CommandUseCase[Cmd, Reply] {
	if fn == nil {
		return nil
	}
	return &CommandUseCase[Cmd, Reply]{fn: fn, name: name}
}

// Execute normalizes the command, validates it, and delegates to the gateway function.
func (uc *CommandUseCase[Cmd, Reply]) Execute(ctx context.Context, cmd Cmd) (Reply, *problem.Problem) {
	var zero Reply
	if uc == nil || uc.fn == nil {
		return zero, problem.New(problem.Unavailable, "gateway is unavailable")
	}
	cmd = cmd.Normalize()
	if prob := cmd.Validate(); prob != nil {
		return zero, prob
	}
	return uc.fn(ctx, cmd)
}

// GatewayUseCase wraps a gateway function with nil safety only.
// Use this for operations that need no input normalization or validation.
type GatewayUseCase[In, Out any] struct {
	fn   GatewayFunc[In, Out]
	name string
}

// NewGateway creates a GatewayUseCase that delegates directly after nil check.
func NewGateway[In, Out any](fn GatewayFunc[In, Out], name string) *GatewayUseCase[In, Out] {
	if fn == nil {
		return nil
	}
	return &GatewayUseCase[In, Out]{fn: fn, name: name}
}

// Execute delegates to the gateway function after nil safety check.
func (uc *GatewayUseCase[In, Out]) Execute(ctx context.Context, in In) (Out, *problem.Problem) {
	var zero Out
	if uc == nil || uc.fn == nil {
		return zero, problem.New(problem.Unavailable, "gateway is unavailable")
	}
	return uc.fn(ctx, in)
}
