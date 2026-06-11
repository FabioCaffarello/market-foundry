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
	VenueOrderID  string
	ClientOrderID string
	Status        domainexec.Status
	Intent        domainexec.ExecutionIntent
}

// VenuePort is the adapter boundary for venue order placement.
// The execute binary calls this to submit orders. First implementation: PaperVenueAdapter.
// Future implementations: exchange-specific adapters.
// Invariant: no VenuePort implementation may bypass the kill switch or staleness guard —
// those checks happen in the actor layer before calling VenuePort.
type VenuePort interface {
	SubmitOrder(ctx context.Context, req VenueOrderRequest) (VenueOrderReceipt, *problem.Problem)
}

// VenueQueryPort is the adapter boundary for querying existing orders.
// Used for post-200 reconciliation: when SubmitOrder succeeds at the venue (HTTP 200)
// but the response body is lost, QueryOrder recovers the order status and fills
// using the deterministic client order ID.
//
// S322: This interface is intentionally separate from VenuePort to keep the
// submit path clean and avoid forcing query capability on all adapters.
type VenueQueryPort interface {
	QueryOrder(ctx context.Context, clientOrderID, symbol string) (VenueOrderReceipt, *problem.Problem)
}
