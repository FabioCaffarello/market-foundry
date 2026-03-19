package ports

import (
	"context"

	domainexec "internal/domain/execution"
	"internal/shared/problem"
)

// VenueOrderRequest is the input to VenuePort.SubmitOrder.
type VenueOrderRequest struct {
	Intent domainexec.ExecutionIntent
}

// VenueOrderReceipt is the output of a successful VenuePort.SubmitOrder call.
type VenueOrderReceipt struct {
	VenueOrderID string
	Status       domainexec.Status
	Intent       domainexec.ExecutionIntent
}

// VenuePort is the adapter boundary for venue order placement.
// The execute binary calls this to submit orders. First implementation: PaperVenueAdapter.
// Future implementations: exchange-specific adapters.
// Invariant: no VenuePort implementation may bypass the kill switch or staleness guard —
// those checks happen in the actor layer before calling VenuePort.
type VenuePort interface {
	SubmitOrder(ctx context.Context, req VenueOrderRequest) (VenueOrderReceipt, *problem.Problem)
}
