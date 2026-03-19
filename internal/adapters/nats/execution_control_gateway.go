package nats

import (
	"context"

	"internal/application/executionclient"
	"internal/application/ports"
	"internal/shared/problem"
)

// ExecutionControlGateway implements the execution control port via NATS request/reply.
// Used by the gateway binary to query and update the execution control gate.
type ExecutionControlGateway struct {
	client   requestReplyClient
	source   string
	registry ExecutionRegistry
}

var _ ports.ExecutionControlGateway = (*ExecutionControlGateway)(nil)

func NewExecutionControlGateway(client requestReplyClient, source string) *ExecutionControlGateway {
	return &ExecutionControlGateway{
		client:   client,
		source:   source,
		registry: DefaultExecutionRegistry(),
	}
}

func (g *ExecutionControlGateway) GetExecutionControl(ctx context.Context, query executionclient.ExecutionControlQuery) (executionclient.ExecutionControlReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return executionclient.ExecutionControlReply{}, problem.New(problem.Unavailable, "execution control gateway is unavailable")
	}

	spec := g.registry.ControlGet
	requestBytes, prob := encodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return executionclient.ExecutionControlReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return executionclient.ExecutionControlReply{}, problem.Wrap(err, problem.Unavailable, "request execution control get failed")
	}

	return decodeControlReply[executionclient.ExecutionControlReply](spec, replyBytes)
}

func (g *ExecutionControlGateway) SetExecutionControl(ctx context.Context, cmd executionclient.SetExecutionControlCommand) (executionclient.ExecutionControlReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return executionclient.ExecutionControlReply{}, problem.New(problem.Unavailable, "execution control gateway is unavailable")
	}

	spec := g.registry.ControlSet
	requestBytes, prob := encodeControlRequest(ctx, spec, g.source, cmd)
	if prob != nil {
		return executionclient.ExecutionControlReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return executionclient.ExecutionControlReply{}, problem.Wrap(err, problem.Unavailable, "request execution control set failed")
	}

	return decodeControlReply[executionclient.ExecutionControlReply](spec, replyBytes)
}
