package nats

import (
	"context"
	"fmt"

	"internal/application/executionclient"
	"internal/application/ports"
	"internal/shared/problem"
)

// ExecutionGateway implements the execution query port via NATS request/reply.
// Used by the gateway binary to query the store for the latest execution intent.
type ExecutionGateway struct {
	client   requestReplyClient
	source   string
	registry ExecutionRegistry
}

var _ ports.ExecutionGateway = (*ExecutionGateway)(nil)

func NewExecutionGateway(client requestReplyClient, source string) *ExecutionGateway {
	return &ExecutionGateway{
		client:   client,
		source:   source,
		registry: DefaultExecutionRegistry(),
	}
}

func (g *ExecutionGateway) GetLatestExecution(ctx context.Context, query executionclient.ExecutionLatestQuery) (executionclient.ExecutionLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return executionclient.ExecutionLatestReply{}, problem.New(problem.Unavailable, "execution gateway is unavailable")
	}

	spec, ok := g.registry.LatestSpecByType(query.Type)
	if !ok {
		return executionclient.ExecutionLatestReply{}, problem.New(problem.InvalidArgument, fmt.Sprintf("unsupported execution type: %s", query.Type))
	}

	requestBytes, prob := encodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return executionclient.ExecutionLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return executionclient.ExecutionLatestReply{}, problem.Wrap(err, problem.Unavailable, "request execution latest failed")
	}

	return decodeControlReply[executionclient.ExecutionLatestReply](spec, replyBytes)
}

func (g *ExecutionGateway) GetExecutionStatus(ctx context.Context, query executionclient.ExecutionStatusQuery) (executionclient.ExecutionStatusReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return executionclient.ExecutionStatusReply{}, problem.New(problem.Unavailable, "execution gateway is unavailable")
	}

	spec := g.registry.StatusLatest
	requestBytes, prob := encodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return executionclient.ExecutionStatusReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return executionclient.ExecutionStatusReply{}, problem.Wrap(err, problem.Unavailable, "request execution status failed")
	}

	return decodeControlReply[executionclient.ExecutionStatusReply](spec, replyBytes)
}
