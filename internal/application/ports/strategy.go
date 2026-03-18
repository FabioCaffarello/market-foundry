package ports

import (
	"context"

	"internal/application/strategyclient"
	"internal/shared/problem"
)

// StrategyGateway is the port for querying strategy projections via NATS request/reply.
// Implemented by the NATS adapter; consumed by the gateway binary.
// The store binary serves these queries from materialized NATS KV read models.
type StrategyGateway interface {
	GetLatestStrategy(context.Context, strategyclient.StrategyLatestQuery) (strategyclient.StrategyLatestReply, *problem.Problem)
}
