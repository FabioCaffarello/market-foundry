package natsexecution

import (
	"context"

	"internal/adapters/nats/natskit"
	"internal/application/executionclient"
	"internal/application/ports"
	"internal/shared/problem"
)

// ControlGateway implements the execution control port via NATS request/reply.
// Used by the gateway binary to query and update the execution control gate.
type ControlGateway struct {
	client   natskit.RequestReplyClient
	source   string
	registry Registry
}

var _ ports.ExecutionControlGateway = (*ControlGateway)(nil)

func NewControlGateway(client natskit.RequestReplyClient, source string) *ControlGateway {
	return &ControlGateway{
		client:   client,
		source:   source,
		registry: DefaultRegistry(),
	}
}

func (g *ControlGateway) GetExecutionControl(ctx context.Context, query executionclient.ExecutionControlQuery) (executionclient.ExecutionControlReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return executionclient.ExecutionControlReply{}, problem.New(problem.Unavailable, "execution control gateway is unavailable")
	}

	spec := g.registry.ControlGet
	requestBytes, prob := natskit.EncodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return executionclient.ExecutionControlReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return executionclient.ExecutionControlReply{}, problem.Wrap(err, problem.Unavailable, "request execution control get failed")
	}

	return natskit.DecodeControlReply[executionclient.ExecutionControlReply](spec, replyBytes)
}

func (g *ControlGateway) SetExecutionControl(ctx context.Context, cmd executionclient.SetExecutionControlCommand) (executionclient.ExecutionControlReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return executionclient.ExecutionControlReply{}, problem.New(problem.Unavailable, "execution control gateway is unavailable")
	}

	spec := g.registry.ControlSet
	requestBytes, prob := natskit.EncodeControlRequest(ctx, spec, g.source, cmd)
	if prob != nil {
		return executionclient.ExecutionControlReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return executionclient.ExecutionControlReply{}, problem.Wrap(err, problem.Unavailable, "request execution control set failed")
	}

	return natskit.DecodeControlReply[executionclient.ExecutionControlReply](spec, replyBytes)
}
