package ports

import (
	"context"

	"internal/application/executionclient"
	"internal/shared/problem"
)

// ExecutionGateway is the port for querying execution projections via NATS request/reply.
// Implemented by the NATS adapter; consumed by the gateway binary.
// The store binary serves these queries from materialized NATS KV read models.
type ExecutionGateway interface {
	GetLatestExecution(context.Context, executionclient.ExecutionLatestQuery) (executionclient.ExecutionLatestReply, *problem.Problem)
	GetExecutionStatus(context.Context, executionclient.ExecutionStatusQuery) (executionclient.ExecutionStatusReply, *problem.Problem)
}

// ExecutionControlGateway is the port for querying and updating the execution control gate.
// Implemented by the NATS adapter; consumed by the gateway binary.
// The store binary manages the EXECUTION_CONTROL KV bucket.
type ExecutionControlGateway interface {
	GetExecutionControl(context.Context, executionclient.ExecutionControlQuery) (executionclient.ExecutionControlReply, *problem.Problem)
	SetExecutionControl(context.Context, executionclient.SetExecutionControlCommand) (executionclient.ExecutionControlReply, *problem.Problem)
}
