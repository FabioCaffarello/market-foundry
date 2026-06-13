package natsinsights

import (
	"context"

	"internal/application/insightsclient"
	"internal/application/ports"
	"internal/shared/problem"
)

// Gateway is the insights read adapter. Unlike the request/reply
// gateways, it reads the INSIGHTS_VOLUME_PROFILE_LATEST KV bucket
// directly — the gateway binary is a free KV reader (ADR-0008:
// single-writer is the store; readers unrestricted). PROGRAM-0005 /
// H-8.a.
type Gateway struct {
	kv *VolumeProfileKVStore
}

var _ ports.InsightsGateway = (*Gateway)(nil)

// NewGateway builds the insights read gateway over a started KV
// store.
func NewGateway(kv *VolumeProfileKVStore) *Gateway {
	return &Gateway{kv: kv}
}

func (g *Gateway) GetLatestVolumeProfile(ctx context.Context, query insightsclient.VolumeProfileLatestQuery) (insightsclient.VolumeProfileLatestReply, *problem.Problem) {
	if g == nil || g.kv == nil {
		return insightsclient.VolumeProfileLatestReply{}, problem.New(problem.Unavailable, "insights gateway is unavailable")
	}
	vp, prob := g.kv.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return insightsclient.VolumeProfileLatestReply{}, prob
	}
	return insightsclient.VolumeProfileLatestReply{VolumeProfile: vp}, nil
}
