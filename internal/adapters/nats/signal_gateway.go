package nats

import (
	"context"
	"fmt"

	"internal/application/ports"
	"internal/application/signalclient"
	"internal/shared/problem"
)

// SignalGateway implements the signal query port via NATS request/reply.
// Used by the gateway binary to query the store for the latest signal.
type SignalGateway struct {
	client   requestReplyClient
	source   string
	registry SignalRegistry
}

var _ ports.SignalGateway = (*SignalGateway)(nil)

func NewSignalGateway(client requestReplyClient, source string) *SignalGateway {
	return &SignalGateway{
		client:   client,
		source:   source,
		registry: DefaultSignalRegistry(),
	}
}

func (g *SignalGateway) GetLatestSignal(ctx context.Context, query signalclient.SignalLatestQuery) (signalclient.SignalLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return signalclient.SignalLatestReply{}, problem.New(problem.Unavailable, "signal gateway is unavailable")
	}

	spec, ok := g.registry.LatestSpecByType(query.Type)
	if !ok {
		return signalclient.SignalLatestReply{}, problem.New(problem.InvalidArgument, fmt.Sprintf("unsupported signal type: %s", query.Type))
	}

	requestBytes, prob := encodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return signalclient.SignalLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return signalclient.SignalLatestReply{}, problem.Wrap(err, problem.Unavailable, "request signal latest failed")
	}

	return decodeControlReply[signalclient.SignalLatestReply](spec, replyBytes)
}
