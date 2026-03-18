package nats

import (
	"context"
	"fmt"

	"internal/application/ports"
	"internal/application/riskclient"
	"internal/shared/problem"
)

// RiskGateway implements the risk query port via NATS request/reply.
// Used by the gateway binary to query the store for the latest risk assessment.
type RiskGateway struct {
	client   requestReplyClient
	source   string
	registry RiskRegistry
}

var _ ports.RiskGateway = (*RiskGateway)(nil)

func NewRiskGateway(client requestReplyClient, source string) *RiskGateway {
	return &RiskGateway{
		client:   client,
		source:   source,
		registry: DefaultRiskRegistry(),
	}
}

func (g *RiskGateway) GetLatestRisk(ctx context.Context, query riskclient.RiskLatestQuery) (riskclient.RiskLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return riskclient.RiskLatestReply{}, problem.New(problem.Unavailable, "risk gateway is unavailable")
	}

	spec, ok := g.registry.LatestSpecByType(query.Type)
	if !ok {
		return riskclient.RiskLatestReply{}, problem.New(problem.InvalidArgument, fmt.Sprintf("unsupported risk type: %s", query.Type))
	}

	requestBytes, prob := encodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return riskclient.RiskLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return riskclient.RiskLatestReply{}, problem.Wrap(err, problem.Unavailable, "request risk latest failed")
	}

	return decodeControlReply[riskclient.RiskLatestReply](spec, replyBytes)
}
