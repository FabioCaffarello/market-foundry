package ports

import (
	"context"

	"internal/application/decisionclient"
	"internal/shared/problem"
)

// DecisionGateway is the port for querying decision projections via NATS request/reply.
// Implemented by the NATS adapter; consumed by the gateway binary.
// The store binary serves these queries from materialized NATS KV read models.
type DecisionGateway interface {
	GetLatestDecision(context.Context, decisionclient.DecisionLatestQuery) (decisionclient.DecisionLatestReply, *problem.Problem)
}
