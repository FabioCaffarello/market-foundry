package natsinsights

import (
	"context"

	"internal/application/insightsclient"
	"internal/application/ports"
	"internal/shared/problem"
)

// Gateway is the insights read adapter. Unlike the request/reply
// gateways, it reads the insights KV latest buckets directly — the
// gateway binary is a free KV reader (ADR-0008: single-writer is the
// store; readers unrestricted). PROGRAM-0005 / H-8.a (volume profile),
// H-8.b (TPO). A nil per-capability KV store degrades that capability
// to Unavailable without affecting the others.
type Gateway struct {
	kv           *VolumeProfileKVStore
	tpoKV        *TPOKVStore
	crossVenueKV *CrossVenueKVStore
}

var _ ports.InsightsGateway = (*Gateway)(nil)

// NewGateway builds the insights read gateway over started KV stores.
func NewGateway(kv *VolumeProfileKVStore, tpoKV *TPOKVStore, crossVenueKV *CrossVenueKVStore) *Gateway {
	return &Gateway{kv: kv, tpoKV: tpoKV, crossVenueKV: crossVenueKV}
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

func (g *Gateway) GetLatestTPOProfile(ctx context.Context, query insightsclient.TPOProfileLatestQuery) (insightsclient.TPOProfileLatestReply, *problem.Problem) {
	if g == nil || g.tpoKV == nil {
		return insightsclient.TPOProfileLatestReply{}, problem.New(problem.Unavailable, "tpo gateway is unavailable")
	}
	tp, prob := g.tpoKV.Get(ctx, query.Source, query.Instrument, query.Timeframe)
	if prob != nil {
		return insightsclient.TPOProfileLatestReply{}, prob
	}
	return insightsclient.TPOProfileLatestReply{TPOProfile: tp}, nil
}

func (g *Gateway) GetLatestCrossVenue(ctx context.Context, query insightsclient.CrossVenueLatestQuery) (insightsclient.CrossVenueLatestReply, *problem.Problem) {
	if g == nil || g.crossVenueKV == nil {
		return insightsclient.CrossVenueLatestReply{}, problem.New(problem.Unavailable, "cross venue gateway is unavailable")
	}
	cv, prob := g.crossVenueKV.Get(ctx, query.Instrument, query.Timeframe)
	if prob != nil {
		return insightsclient.CrossVenueLatestReply{}, prob
	}
	return insightsclient.CrossVenueLatestReply{CrossVenueSnapshot: cv}, nil
}
