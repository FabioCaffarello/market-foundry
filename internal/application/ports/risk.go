package ports

import (
	"context"

	"internal/application/riskclient"
	"internal/shared/problem"
)

// RiskGateway is the port for querying risk projections via NATS request/reply.
// Implemented by the NATS adapter; consumed by the gateway binary.
// The store binary serves these queries from materialized NATS KV read models.
type RiskGateway interface {
	GetLatestRisk(context.Context, riskclient.RiskLatestQuery) (riskclient.RiskLatestReply, *problem.Problem)
}
