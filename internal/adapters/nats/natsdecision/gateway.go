package natsdecision

import (
	"context"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/application/decisionclient"
	"internal/application/ports"
	"internal/shared/problem"
)

// Gateway implements the decision query port via NATS request/reply.
type Gateway struct {
	client   natskit.RequestReplyClient
	source   string
	registry Registry
}

var _ ports.DecisionGateway = (*Gateway)(nil)

func NewGateway(client natskit.RequestReplyClient, source string) *Gateway {
	return &Gateway{
		client:   client,
		source:   source,
		registry: DefaultRegistry(),
	}
}

func (g *Gateway) GetLatestDecision(ctx context.Context, query decisionclient.DecisionLatestQuery) (decisionclient.DecisionLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return decisionclient.DecisionLatestReply{}, problem.New(problem.Unavailable, "decision gateway is unavailable")
	}

	spec, ok := g.registry.LatestSpecByType(query.Type)
	if !ok {
		return decisionclient.DecisionLatestReply{}, problem.New(problem.InvalidArgument, fmt.Sprintf("unsupported decision type: %s", query.Type))
	}

	requestBytes, prob := natskit.EncodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return decisionclient.DecisionLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return decisionclient.DecisionLatestReply{}, problem.Wrap(err, problem.Unavailable, "request decision latest failed")
	}

	return natskit.DecodeControlReply[decisionclient.DecisionLatestReply](spec, replyBytes)
}
