package execution

import (
	"context"
	"fmt"

	"internal/application/ports"
	"internal/shared/problem"
	"internal/shared/settings"
)

// SegmentRouter implements VenuePort by routing SubmitOrder calls to the
// correct segment adapter based on the intent's Source field.
//
// S400: This replaces the "first enabled segment" approach from S399 with
// true multi-segment dispatch. Each enabled segment has its own adapter
// instance and the router maps source → segment → adapter at call time.
//
// Fail-closed: intents with unrecognized sources are rejected with a
// structured Problem rather than silently dropped or misrouted.
type SegmentRouter struct {
	adapters map[settings.MarketSegment]ports.VenuePort
	queries  map[settings.MarketSegment]ports.VenueQueryPort
}

// NewSegmentRouter creates a router with no registered adapters.
// Use Register to add adapters for each segment.
func NewSegmentRouter() *SegmentRouter {
	return &SegmentRouter{
		adapters: make(map[settings.MarketSegment]ports.VenuePort),
		queries:  make(map[settings.MarketSegment]ports.VenueQueryPort),
	}
}

// Register adds a submit adapter for the given segment.
func (r *SegmentRouter) Register(seg settings.MarketSegment, adapter ports.VenuePort) {
	r.adapters[seg] = adapter
}

// RegisterQuery adds a query adapter for the given segment.
func (r *SegmentRouter) RegisterQuery(seg settings.MarketSegment, query ports.VenueQueryPort) {
	r.queries[seg] = query
}

// SubmitOrder routes the request to the adapter matching the intent's Source.
// Implements ports.VenuePort.
func (r *SegmentRouter) SubmitOrder(ctx context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	seg := settings.SegmentForSource(req.Intent.Source)
	if seg == "" {
		return ports.VenueOrderReceipt{}, problem.New(
			problem.InvalidArgument,
			fmt.Sprintf("no segment mapping for source %q — intent cannot be routed", req.Intent.Source),
		)
	}

	adapter, ok := r.adapters[seg]
	if !ok {
		return ports.VenueOrderReceipt{}, problem.New(
			problem.InvalidArgument,
			fmt.Sprintf("segment %q has no registered adapter — source %q cannot be routed", seg, req.Intent.Source),
		)
	}

	return adapter.SubmitOrder(ctx, req)
}

// QueryOrder routes the query to the adapter matching the source-implied segment.
// Implements ports.VenueQueryPort.
func (r *SegmentRouter) QueryOrder(ctx context.Context, clientOrderID, symbol string) (ports.VenueOrderReceipt, *problem.Problem) {
	// QueryOrder doesn't carry a source field — iterate registered query ports.
	// This is only used for post-200 reconciliation, which is rare.
	// When multiple segments are registered, try each until one succeeds.
	for _, query := range r.queries {
		receipt, prob := query.QueryOrder(ctx, clientOrderID, symbol)
		if prob == nil {
			return receipt, nil
		}
	}
	return ports.VenueOrderReceipt{}, problem.New(
		problem.NotFound,
		fmt.Sprintf("order %q not found across any registered segment", clientOrderID),
	)
}

// SegmentCount returns the number of registered segment adapters.
func (r *SegmentRouter) SegmentCount() int {
	return len(r.adapters)
}

// HasSegment reports whether the given segment has a registered adapter.
func (r *SegmentRouter) HasSegment(seg settings.MarketSegment) bool {
	_, ok := r.adapters[seg]
	return ok
}
