package nats

import (
	"context"
	"fmt"

	"internal/application/ports"
	"internal/application/strategyclient"
	"internal/shared/problem"
)

// StrategyGateway implements the strategy query port via NATS request/reply.
// Used by the gateway binary to query the store for the latest strategy.
type StrategyGateway struct {
	client   requestReplyClient
	source   string
	registry StrategyRegistry
}

var _ ports.StrategyGateway = (*StrategyGateway)(nil)

func NewStrategyGateway(client requestReplyClient, source string) *StrategyGateway {
	return &StrategyGateway{
		client:   client,
		source:   source,
		registry: DefaultStrategyRegistry(),
	}
}

func (g *StrategyGateway) GetLatestStrategy(ctx context.Context, query strategyclient.StrategyLatestQuery) (strategyclient.StrategyLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return strategyclient.StrategyLatestReply{}, problem.New(problem.Unavailable, "strategy gateway is unavailable")
	}

	spec, ok := g.registry.LatestSpecByType(query.Type)
	if !ok {
		return strategyclient.StrategyLatestReply{}, problem.New(problem.InvalidArgument, fmt.Sprintf("unsupported strategy type: %s", query.Type))
	}

	requestBytes, prob := encodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return strategyclient.StrategyLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return strategyclient.StrategyLatestReply{}, problem.Wrap(err, problem.Unavailable, "request strategy latest failed")
	}

	return decodeControlReply[strategyclient.StrategyLatestReply](spec, replyBytes)
}
