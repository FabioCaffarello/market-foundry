package execution

import (
	"context"
	"time"

	"internal/application/ports"
	"internal/shared/problem"
)

// Post200Reconciler wraps a VenuePort and a VenueQueryPort. When a SubmitOrder call
// fails with a body-read-failure-after-200 (the venue accepted the order but the
// response body was lost), it automatically queries the venue to recover the order
// status and fills using the deterministic client order ID.
//
// Design invariants (S322):
//   - Never re-submits the order (no duplicate execution risk).
//   - Uses the same deterministic client order ID from the failed submit.
//   - Query uses a fresh context with its own deadline (independent of submit timeout).
//   - If recovery fails, both the original and recovery errors are returned.
//   - Non-body-read failures pass through unchanged.
//
// This is a surgical reconciliation mechanism for the single case identified by
// R-S320-1, not a general-purpose OMS reconciliation layer.
type Post200Reconciler struct {
	submit       ports.VenuePort
	query        ports.VenueQueryPort
	queryTimeout time.Duration
}

// NewPost200Reconciler creates a reconciler that wraps submit and query ports.
// queryTimeout is the deadline for the recovery query (default 10s if zero).
func NewPost200Reconciler(submit ports.VenuePort, query ports.VenueQueryPort, queryTimeout time.Duration) *Post200Reconciler {
	if queryTimeout <= 0 {
		queryTimeout = defaultRequestDeadline
	}
	return &Post200Reconciler{
		submit:       submit,
		query:        query,
		queryTimeout: queryTimeout,
	}
}

// SubmitOrder implements ports.VenuePort. It delegates to the inner submit port
// and, on body-read-failure-after-200, attempts recovery via QueryOrder.
func (r *Post200Reconciler) SubmitOrder(ctx context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	receipt, prob := r.submit.SubmitOrder(ctx, req)
	if prob == nil {
		return receipt, nil
	}

	// Only intercept body-read-failure-after-200.
	if !isBodyReadFailureAfter200(prob) {
		return receipt, prob
	}

	clientOrderID, _ := prob.Details["client_order_id"].(string)
	if clientOrderID == "" {
		return receipt, prob
	}

	// Attempt recovery with a fresh context (the original may be expired).
	queryCtx, cancel := context.WithTimeout(context.Background(), r.queryTimeout)
	defer cancel()

	recovered, queryProb := r.query.QueryOrder(queryCtx, clientOrderID, req.Intent.Symbol) //nolint:contextcheck // fresh ctx is deliberate — original ctx may have expired triggering this recovery path
	if queryProb != nil {
		// Recovery failed. Return original error enriched with recovery failure info.
		return receipt, prob.
			WithDetail("reconciliation_attempted", true).
			WithDetail("reconciliation_failed", true).
			WithDetail("reconciliation_error", queryProb.Message)
	}

	// Recovery succeeded. Restore the original intent with recovered status/fills.
	restoredIntent := req.Intent
	restoredIntent.Status = recovered.Status
	restoredIntent.FilledQuantity = recovered.Intent.FilledQuantity
	restoredIntent.Fills = recovered.Intent.Fills

	return ports.VenueOrderReceipt{
		VenueOrderID:  recovered.VenueOrderID,
		ClientOrderID: clientOrderID,
		Status:        recovered.Status,
		Intent:        restoredIntent,
	}, nil
}

// isBodyReadFailureAfter200 checks whether a Problem represents the specific
// body-read-failure-after-200 case marked by the adapter.
func isBodyReadFailureAfter200(prob *problem.Problem) bool {
	if prob == nil || prob.Details == nil {
		return false
	}
	v, ok := prob.Details["body_read_failure_after_200"]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
