package natsstrategy

import (
	"context"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/application/ports"
	"internal/application/strategyclient"
	"internal/shared/problem"
)

// Gateway implements the strategy query port via NATS request/reply.
// Used by the gateway binary to query the store for the latest strategy.
type Gateway struct {
	client   natskit.RequestReplyClient
	source   string
	registry Registry
}

var _ ports.StrategyGateway = (*Gateway)(nil)

func NewGateway(client natskit.RequestReplyClient, source string) *Gateway {
	return &Gateway{
		client:   client,
		source:   source,
		registry: DefaultRegistry(),
	}
}

func (g *Gateway) GetLatestStrategy(ctx context.Context, query strategyclient.StrategyLatestQuery) (strategyclient.StrategyLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return strategyclient.StrategyLatestReply{}, problem.New(problem.Unavailable, "strategy gateway is unavailable")
	}

	spec, ok := g.registry.LatestSpecByType(query.Type)
	if !ok {
		return strategyclient.StrategyLatestReply{}, problem.New(problem.InvalidArgument, fmt.Sprintf("unsupported strategy type: %s", query.Type))
	}

	requestBytes, prob := natskit.EncodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return strategyclient.StrategyLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return strategyclient.StrategyLatestReply{}, problem.Wrap(err, problem.Unavailable, "request strategy latest failed")
	}

	return natskit.DecodeControlReply[strategyclient.StrategyLatestReply](spec, replyBytes)
}
