package ports

import (
	"context"

	"internal/application/signalclient"
	"internal/shared/problem"
)

// SignalGateway is the port for querying signal projections via NATS request/reply.
// Implemented by the NATS adapter; consumed by the gateway binary.
// The store binary serves these queries from materialized NATS KV read models.
type SignalGateway interface {
	GetLatestSignal(context.Context, signalclient.SignalLatestQuery) (signalclient.SignalLatestReply, *problem.Problem)
}
