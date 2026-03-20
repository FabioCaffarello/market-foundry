package natssignal

import (
	"context"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/application/ports"
	"internal/application/signalclient"
	"internal/shared/problem"
)

// Gateway implements the signal query port via NATS request/reply.
// Used by the gateway binary to query the store for the latest signal.
type Gateway struct {
	client   natskit.RequestReplyClient
	source   string
	registry Registry
}

var _ ports.SignalGateway = (*Gateway)(nil)

func NewGateway(client natskit.RequestReplyClient, source string) *Gateway {
	return &Gateway{
		client:   client,
		source:   source,
		registry: DefaultRegistry(),
	}
}

func (g *Gateway) GetLatestSignal(ctx context.Context, query signalclient.SignalLatestQuery) (signalclient.SignalLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return signalclient.SignalLatestReply{}, problem.New(problem.Unavailable, "signal gateway is unavailable")
	}

	spec, ok := g.registry.LatestSpecByType(query.Type)
	if !ok {
		return signalclient.SignalLatestReply{}, problem.New(problem.InvalidArgument, fmt.Sprintf("unsupported signal type: %s", query.Type))
	}

	requestBytes, prob := natskit.EncodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return signalclient.SignalLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return signalclient.SignalLatestReply{}, problem.Wrap(err, problem.Unavailable, "request signal latest failed")
	}

	return natskit.DecodeControlReply[signalclient.SignalLatestReply](spec, replyBytes)
}
