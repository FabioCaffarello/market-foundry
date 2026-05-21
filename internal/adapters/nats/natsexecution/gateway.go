package natsexecution

import (
	"context"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/application/executionclient"
	"internal/application/ports"
	"internal/shared/problem"
)

// Gateway implements the execution query port via NATS request/reply.
// Used by the gateway binary to query the store for the latest execution intent.
type Gateway struct {
	client   natskit.RequestReplyClient
	source   string
	registry Registry
}

var _ ports.ExecutionGateway = (*Gateway)(nil)

func NewGateway(client natskit.RequestReplyClient, source string) *Gateway {
	return &Gateway{
		client:   client,
		source:   source,
		registry: DefaultRegistry(),
	}
}

func (g *Gateway) GetLatestExecution(ctx context.Context, query executionclient.ExecutionLatestQuery) (executionclient.ExecutionLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return executionclient.ExecutionLatestReply{}, problem.New(problem.Unavailable, "execution gateway is unavailable")
	}

	spec, ok := g.registry.LatestSpecByType(query.Type)
	if !ok {
		return executionclient.ExecutionLatestReply{}, problem.New(problem.InvalidArgument, fmt.Sprintf("unsupported execution type: %s", query.Type))
	}

	requestBytes, prob := natskit.EncodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return executionclient.ExecutionLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return executionclient.ExecutionLatestReply{}, problem.Wrap(err, problem.Unavailable, "request execution latest failed")
	}

	return natskit.DecodeControlReply[executionclient.ExecutionLatestReply](spec, replyBytes)
}

func (g *Gateway) GetExecutionStatus(ctx context.Context, query executionclient.ExecutionStatusQuery) (executionclient.ExecutionStatusReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return executionclient.ExecutionStatusReply{}, problem.New(problem.Unavailable, "execution gateway is unavailable")
	}

	spec := g.registry.StatusLatest
	requestBytes, prob := natskit.EncodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return executionclient.ExecutionStatusReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return executionclient.ExecutionStatusReply{}, problem.Wrap(err, problem.Unavailable, "request execution status failed")
	}

	return natskit.DecodeControlReply[executionclient.ExecutionStatusReply](spec, replyBytes)
}

// GetLifecycleList queries the store for all tracked execution lifecycle entries
// across KV buckets. S454A: Exposes the S413 lifecycle list via the gateway.
func (g *Gateway) GetLifecycleList(ctx context.Context, query executionclient.LifecycleListQuery) (executionclient.LifecycleListReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return executionclient.LifecycleListReply{}, problem.New(problem.Unavailable, "execution gateway is unavailable")
	}

	spec := g.registry.LifecycleList
	requestBytes, prob := natskit.EncodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return executionclient.LifecycleListReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return executionclient.LifecycleListReply{}, problem.Wrap(err, problem.Unavailable, "request lifecycle list failed")
	}

	return natskit.DecodeControlReply[executionclient.LifecycleListReply](spec, replyBytes)
}
