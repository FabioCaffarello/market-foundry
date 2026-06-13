package ports

import (
	"context"

	"internal/application/insightsclient"
	"internal/shared/problem"
)

// InsightsGateway is the read port for insights projections
// (PROGRAM-0005 / ADR-0027). Unlike the evidence/signal gateways
// (NATS request/reply to the store query responder), insights is
// read directly from the KV latest bucket by the gateway binary —
// a free KV reader (single-writer is the store; readers are
// unrestricted per ADR-0008). Read-only: no directives.
type InsightsGateway interface {
	GetLatestVolumeProfile(context.Context, insightsclient.VolumeProfileLatestQuery) (insightsclient.VolumeProfileLatestReply, *problem.Problem)
	GetLatestTPOProfile(context.Context, insightsclient.TPOProfileLatestQuery) (insightsclient.TPOProfileLatestReply, *problem.Problem)
	GetLatestCrossVenue(context.Context, insightsclient.CrossVenueLatestQuery) (insightsclient.CrossVenueLatestReply, *problem.Problem)
}
