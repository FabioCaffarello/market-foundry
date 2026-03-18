package nats

import (
	"context"

	"internal/application/evidenceclient"
	"internal/application/ports"
	"internal/shared/problem"
)

// EvidenceGateway implements the evidence query port via NATS request/reply.
// Used by the gateway (server) binary to query the store for the latest candle.
type EvidenceGateway struct {
	client   requestReplyClient
	source   string
	registry EvidenceRegistry
}

var _ ports.EvidenceGateway = (*EvidenceGateway)(nil)

func NewEvidenceGateway(client requestReplyClient, source string) *EvidenceGateway {
	return &EvidenceGateway{
		client:   client,
		source:   source,
		registry: DefaultEvidenceRegistry(),
	}
}

func (g *EvidenceGateway) GetCandleHistory(ctx context.Context, query evidenceclient.CandleHistoryQuery) (evidenceclient.CandleHistoryReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return evidenceclient.CandleHistoryReply{}, problem.New(problem.Unavailable, "evidence gateway is unavailable")
	}

	requestBytes, prob := encodeControlRequest(ctx, g.registry.CandleHistory, g.source, query)
	if prob != nil {
		return evidenceclient.CandleHistoryReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, g.registry.CandleHistory.Subject, requestBytes)
	if err != nil {
		return evidenceclient.CandleHistoryReply{}, problem.Wrap(err, problem.Unavailable, "request evidence candle history failed")
	}

	return decodeControlReply[evidenceclient.CandleHistoryReply](g.registry.CandleHistory, replyBytes)
}

func (g *EvidenceGateway) GetLatestCandle(ctx context.Context, query evidenceclient.CandleLatestQuery) (evidenceclient.CandleLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return evidenceclient.CandleLatestReply{}, problem.New(problem.Unavailable, "evidence gateway is unavailable")
	}

	requestBytes, prob := encodeControlRequest(ctx, g.registry.CandleLatest, g.source, query)
	if prob != nil {
		return evidenceclient.CandleLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, g.registry.CandleLatest.Subject, requestBytes)
	if err != nil {
		return evidenceclient.CandleLatestReply{}, problem.Wrap(err, problem.Unavailable, "request evidence latest candle failed")
	}

	return decodeControlReply[evidenceclient.CandleLatestReply](g.registry.CandleLatest, replyBytes)
}

func (g *EvidenceGateway) GetLatestTradeBurst(ctx context.Context, query evidenceclient.TradeBurstLatestQuery) (evidenceclient.TradeBurstLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return evidenceclient.TradeBurstLatestReply{}, problem.New(problem.Unavailable, "evidence gateway is unavailable")
	}

	requestBytes, prob := encodeControlRequest(ctx, g.registry.TradeBurstLatest, g.source, query)
	if prob != nil {
		return evidenceclient.TradeBurstLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, g.registry.TradeBurstLatest.Subject, requestBytes)
	if err != nil {
		return evidenceclient.TradeBurstLatestReply{}, problem.Wrap(err, problem.Unavailable, "request evidence latest trade burst failed")
	}

	return decodeControlReply[evidenceclient.TradeBurstLatestReply](g.registry.TradeBurstLatest, replyBytes)
}

func (g *EvidenceGateway) GetLatestVolume(ctx context.Context, query evidenceclient.VolumeLatestQuery) (evidenceclient.VolumeLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return evidenceclient.VolumeLatestReply{}, problem.New(problem.Unavailable, "evidence gateway is unavailable")
	}

	requestBytes, prob := encodeControlRequest(ctx, g.registry.VolumeLatest, g.source, query)
	if prob != nil {
		return evidenceclient.VolumeLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, g.registry.VolumeLatest.Subject, requestBytes)
	if err != nil {
		return evidenceclient.VolumeLatestReply{}, problem.Wrap(err, problem.Unavailable, "request evidence latest volume failed")
	}

	return decodeControlReply[evidenceclient.VolumeLatestReply](g.registry.VolumeLatest, replyBytes)
}
