package nats

import (
	"context"
	"fmt"

	"internal/application/decisionclient"
	"internal/application/ports"
	"internal/shared/problem"
)

// DecisionGateway implements the decision query port via NATS request/reply.
// Used by the gateway binary to query the store for the latest decision.
type DecisionGateway struct {
	client   requestReplyClient
	source   string
	registry DecisionRegistry
}

var _ ports.DecisionGateway = (*DecisionGateway)(nil)

func NewDecisionGateway(client requestReplyClient, source string) *DecisionGateway {
	return &DecisionGateway{
		client:   client,
		source:   source,
		registry: DefaultDecisionRegistry(),
	}
}

func (g *DecisionGateway) GetLatestDecision(ctx context.Context, query decisionclient.DecisionLatestQuery) (decisionclient.DecisionLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return decisionclient.DecisionLatestReply{}, problem.New(problem.Unavailable, "decision gateway is unavailable")
	}

	spec, ok := g.registry.LatestSpecByType(query.Type)
	if !ok {
		return decisionclient.DecisionLatestReply{}, problem.New(problem.InvalidArgument, fmt.Sprintf("unsupported decision type: %s", query.Type))
	}

	requestBytes, prob := encodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return decisionclient.DecisionLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return decisionclient.DecisionLatestReply{}, problem.Wrap(err, problem.Unavailable, "request decision latest failed")
	}

	return decodeControlReply[decisionclient.DecisionLatestReply](spec, replyBytes)
}
