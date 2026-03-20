package natsevidence

import (
	"context"

	"internal/adapters/nats/natskit"
	"internal/application/evidenceclient"
	"internal/application/ports"
	"internal/shared/problem"
)

type Gateway struct {
	client   natskit.RequestReplyClient
	source   string
	registry Registry
}

var _ ports.EvidenceGateway = (*Gateway)(nil)

func NewGateway(client natskit.RequestReplyClient, source string) *Gateway {
	return &Gateway{
		client:   client,
		source:   source,
		registry: DefaultRegistry(),
	}
}

func (g *Gateway) GetCandleHistory(ctx context.Context, query evidenceclient.CandleHistoryQuery) (evidenceclient.CandleHistoryReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return evidenceclient.CandleHistoryReply{}, problem.New(problem.Unavailable, "evidence gateway is unavailable")
	}

	requestBytes, prob := natskit.EncodeControlRequest(ctx, g.registry.CandleHistory, g.source, query)
	if prob != nil {
		return evidenceclient.CandleHistoryReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, g.registry.CandleHistory.Subject, requestBytes)
	if err != nil {
		return evidenceclient.CandleHistoryReply{}, problem.Wrap(err, problem.Unavailable, "request evidence candle history failed")
	}

	return natskit.DecodeControlReply[evidenceclient.CandleHistoryReply](g.registry.CandleHistory, replyBytes)
}

func (g *Gateway) GetLatestCandle(ctx context.Context, query evidenceclient.CandleLatestQuery) (evidenceclient.CandleLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return evidenceclient.CandleLatestReply{}, problem.New(problem.Unavailable, "evidence gateway is unavailable")
	}

	requestBytes, prob := natskit.EncodeControlRequest(ctx, g.registry.CandleLatest, g.source, query)
	if prob != nil {
		return evidenceclient.CandleLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, g.registry.CandleLatest.Subject, requestBytes)
	if err != nil {
		return evidenceclient.CandleLatestReply{}, problem.Wrap(err, problem.Unavailable, "request evidence latest candle failed")
	}

	return natskit.DecodeControlReply[evidenceclient.CandleLatestReply](g.registry.CandleLatest, replyBytes)
}

func (g *Gateway) GetLatestTradeBurst(ctx context.Context, query evidenceclient.TradeBurstLatestQuery) (evidenceclient.TradeBurstLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return evidenceclient.TradeBurstLatestReply{}, problem.New(problem.Unavailable, "evidence gateway is unavailable")
	}

	requestBytes, prob := natskit.EncodeControlRequest(ctx, g.registry.TradeBurstLatest, g.source, query)
	if prob != nil {
		return evidenceclient.TradeBurstLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, g.registry.TradeBurstLatest.Subject, requestBytes)
	if err != nil {
		return evidenceclient.TradeBurstLatestReply{}, problem.Wrap(err, problem.Unavailable, "request evidence latest trade burst failed")
	}

	return natskit.DecodeControlReply[evidenceclient.TradeBurstLatestReply](g.registry.TradeBurstLatest, replyBytes)
}

func (g *Gateway) GetLatestVolume(ctx context.Context, query evidenceclient.VolumeLatestQuery) (evidenceclient.VolumeLatestReply, *problem.Problem) {
	if g == nil || g.client == nil {
		return evidenceclient.VolumeLatestReply{}, problem.New(problem.Unavailable, "evidence gateway is unavailable")
	}

	requestBytes, prob := natskit.EncodeControlRequest(ctx, g.registry.VolumeLatest, g.source, query)
	if prob != nil {
		return evidenceclient.VolumeLatestReply{}, prob
	}

	replyBytes, err := g.client.Request(ctx, g.registry.VolumeLatest.Subject, requestBytes)
	if err != nil {
		return evidenceclient.VolumeLatestReply{}, problem.Wrap(err, problem.Unavailable, "request evidence latest volume failed")
	}

	return natskit.DecodeControlReply[evidenceclient.VolumeLatestReply](g.registry.VolumeLatest, replyBytes)
}
