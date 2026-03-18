package ports

import (
	"context"

	"internal/application/evidenceclient"
	"internal/shared/problem"
)

// EvidenceGateway is the port for querying evidence projections via NATS request/reply.
// Implemented by the NATS adapter; consumed by the gateway binary.
// The store binary serves these queries from a materialized NATS KV read model.
type EvidenceGateway interface {
	GetLatestCandle(context.Context, evidenceclient.CandleLatestQuery) (evidenceclient.CandleLatestReply, *problem.Problem)
	GetCandleHistory(context.Context, evidenceclient.CandleHistoryQuery) (evidenceclient.CandleHistoryReply, *problem.Problem)
	GetLatestTradeBurst(context.Context, evidenceclient.TradeBurstLatestQuery) (evidenceclient.TradeBurstLatestReply, *problem.Problem)
	GetLatestVolume(context.Context, evidenceclient.VolumeLatestQuery) (evidenceclient.VolumeLatestReply, *problem.Problem)
}
