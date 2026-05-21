package natsexecution

import (
	"context"

	"internal/adapters/nats/natskit"
	"internal/application/executionclient"
	"internal/application/ports"
	"internal/shared/problem"
)

// SessionGateway implements the session query port via NATS request/reply.
// S460: Used by the gateway binary to query session metadata from the store.
type SessionGateway struct {
	client   natskit.RequestReplyClient
	source   string
	registry Registry
}

var _ ports.SessionGateway = (*SessionGateway)(nil)

func NewSessionGateway(client natskit.RequestReplyClient, source string) *SessionGateway {
	return &SessionGateway{
		client:   client,
		source:   source,
		registry: DefaultRegistry(),
	}
}

func (g *SessionGateway) GetSession(ctx context.Context, query executionclient.SessionGetQuery) (executionclient.SessionGetReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return executionclient.SessionGetReply{}, problem.New(problem.Unavailable, "session gateway is unavailable")
	}

	spec := g.registry.SessionGet
	requestBytes, prob := natskit.EncodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return executionclient.SessionGetReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return executionclient.SessionGetReply{}, problem.Wrap(err, problem.Unavailable, "request session get failed")
	}

	return natskit.DecodeControlReply[executionclient.SessionGetReply](spec, replyBytes)
}

func (g *SessionGateway) ListSessions(ctx context.Context, query executionclient.SessionListQuery) (executionclient.SessionListReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return executionclient.SessionListReply{}, problem.New(problem.Unavailable, "session gateway is unavailable")
	}

	spec := g.registry.SessionList
	requestBytes, prob := natskit.EncodeControlRequest(ctx, spec, g.source, query)
	if prob != nil {
		return executionclient.SessionListReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, spec.Subject, requestBytes)
	if err != nil {
		return executionclient.SessionListReply{}, problem.Wrap(err, problem.Unavailable, "request session list failed")
	}

	return natskit.DecodeControlReply[executionclient.SessionListReply](spec, replyBytes)
}
