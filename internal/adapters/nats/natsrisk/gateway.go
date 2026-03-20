package natsrisk

import (
	"context"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/application/ports"
	"internal/application/riskclient"
	"internal/shared/problem"
)

// Gateway implements the risk query port via NATS request/reply.
// Used by the gateway binary to query the store for the latest risk assessment.
type Gateway struct {
	client   natskit.RequestReplyClient
	source   string
	registry Registry
}

var _ ports.RiskGateway = (*Gateway)(nil)

func NewGateway(client natskit.RequestReplyClient, source string) *Gateway {
	return &Gateway{
		client:   client,
		source:   source,
		registry: DefaultRegistry(),
	}
}

func (g *Gateway) GetLatestRisk(ctx context.Context, query riskclient.RiskLatestQuery) (riskclient.RiskLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return riskclient.RiskLatestReply{}, problem.New(problem.Unavailable, "risk gateway is unavailable")
	}

	spec, ok := g.registry.LatestSpecByType(query.Type)
	if !ok {
		return riskclient.RiskLatestReply{}, problem.New(problem.InvalidArgument, fmt.Sprintf("unsupported risk type: %s", query.Type))
	}

	requestBytes, prob := natskit.EncodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return riskclient.RiskLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return riskclient.RiskLatestReply{}, problem.Wrap(err, problem.Unavailable, "request risk latest failed")
	}

	return natskit.DecodeControlReply[riskclient.RiskLatestReply](spec, replyBytes)
}
