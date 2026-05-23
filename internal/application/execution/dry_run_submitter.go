package execution

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/healthz"
	"internal/shared/problem"
)

// defaultFillPrice is used when no PriceSource is configured or when the lookup fails.
const defaultFillPrice = "0"

// priceLookupTimeout bounds the price-source read on the dry-run fill
// path. Price lookups are KV reads and should be fast; a stalled
// PriceSource must not block order fills.
const priceLookupTimeout = 2 * time.Second

// DryRunSubmitter is a VenuePort decorator that intercepts all venue submissions
// and produces auditable dry-run receipts without contacting any real venue.
//
// S379: This decorator is the outermost layer in the venue pipeline when
// venue.dry_run is true (the default). It guarantees that no real order reaches
// the venue regardless of the underlying adapter type.
//
// Pipeline composition (innermost → outermost):
//
//	rawAdapter → RetrySubmitter → Post200Reconciler → DryRunSubmitter
//
// The inner pipeline is fully composed but never called — DryRunSubmitter
// short-circuits before delegation. This preserves pipeline wiring for
// activation surface transitions (dry_run=false requires binary restart).
//
// Fail-closed properties:
//   - DryRunSubmitter never delegates to inner.SubmitOrder.
//   - All receipts carry VenueOrderID prefix "dryrun-" for audit filtering.
//   - All fills are marked Simulated=true.
//   - Structured log line emitted for every intercepted intent.
type DryRunSubmitter struct {
	inner       ports.VenuePort
	logger      *slog.Logger
	tracker     *healthz.Tracker
	priceSource ports.PriceSource
}

// NewDryRunSubmitter wraps a venue pipeline with dry-run interception.
// inner is retained for pipeline completeness but never called.
func NewDryRunSubmitter(inner ports.VenuePort) *DryRunSubmitter {
	return &DryRunSubmitter{inner: inner}
}

// WithLogger attaches structured logging for dry-run interceptions.
func (d *DryRunSubmitter) WithLogger(l *slog.Logger) *DryRunSubmitter {
	d.logger = l
	return d
}

// WithTracker attaches health counter tracking for dry-run metrics.
func (d *DryRunSubmitter) WithTracker(t *healthz.Tracker) *DryRunSubmitter {
	d.tracker = t
	return d
}

// WithPriceSource attaches a price lookup for realistic fill prices.
// When set, fills use the last observed close price instead of "0".
// Fallback: if lookup returns an error or empty, "0" is used.
func (d *DryRunSubmitter) WithPriceSource(ps ports.PriceSource) *DryRunSubmitter {
	d.priceSource = ps
	return d
}

// SubmitOrder intercepts the venue call and produces an auditable dry-run receipt.
// The inner VenuePort is never called. The receipt mirrors the structure of a real
// fill so that downstream consumers (fill publisher, store, writer) process it
// identically — the only difference is the "dryrun-" prefix on VenueOrderID
// and Simulated=true on all fill records.
func (d *DryRunSubmitter) SubmitOrder(_ context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	intent := req.Intent
	venueOrderID := newDryRunOrderID()

	// No-action intents: same as paper adapter — return accepted, no fills.
	if intent.Side == domainexec.SideNone {
		d.log("dry-run intercepted no-action intent",
			"venue_order_id", venueOrderID,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"correlation_id", intent.CorrelationID,
		)
		d.inc("dryrun_intercepted")
		d.inc("dryrun_noop")
		return ports.VenueOrderReceipt{
			VenueOrderID: venueOrderID,
			Status:       domainexec.StatusAccepted,
			Intent:       intent,
		}, nil
	}

	// Produce a simulated fill identical in shape to a real venue fill.
	// The price lookup uses a fresh bounded context rather than inheriting
	// the caller's: SubmitOrder discards its ctx parameter (the venue
	// decorator boundary is intentional — fills must not fail just because
	// the upstream call was cancelled), so a stalled PriceSource is bounded
	// here independently.
	priceCtx, priceCancel := context.WithTimeout(context.Background(), priceLookupTimeout)
	fillPrice := d.resolvePrice(priceCtx, intent.Source, intent.Symbol, intent.Timeframe) //nolint:contextcheck // deliberate fresh ctx — see comment above
	priceCancel()

	filled := intent
	filled.Status = domainexec.StatusFilled
	filled.FilledQuantity = intent.Quantity
	filled.Fills = []domainexec.FillRecord{
		{
			Price:     fillPrice,
			Quantity:  intent.Quantity,
			Fee:       "0",
			FeeSource: domainexec.FeeSourceSimulated,
			Simulated: true,
			Timestamp: time.Now().UTC(),
		},
	}

	d.log("dry-run intercepted venue submit",
		"venue_order_id", venueOrderID,
		"source", intent.Source,
		"symbol", intent.Symbol,
		"timeframe", intent.Timeframe,
		"side", string(intent.Side),
		"quantity", intent.Quantity,
		"correlation_id", intent.CorrelationID,
	)
	d.inc("dryrun_intercepted")
	d.inc("dryrun_filled")

	return ports.VenueOrderReceipt{
		VenueOrderID:  venueOrderID,
		ClientOrderID: intent.DeduplicationKey(),
		Status:        domainexec.StatusFilled,
		Intent:        filled,
	}, nil
}

func newDryRunOrderID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "dryrun-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return "dryrun-" + hex.EncodeToString(raw[:])
}

// resolvePrice returns the best-effort last price for the symbol.
// Falls back to defaultFillPrice ("0") when no PriceSource is configured or on error.
func (d *DryRunSubmitter) resolvePrice(ctx context.Context, source, symbol string, timeframe int) string {
	if d.priceSource == nil {
		return defaultFillPrice
	}
	price, prob := d.priceSource.LastPrice(ctx, source, symbol, timeframe)
	if prob != nil || price == "" {
		d.log("price lookup failed, using default",
			"source", source,
			"symbol", symbol,
			"timeframe", timeframe,
			"fallback", defaultFillPrice,
		)
		return defaultFillPrice
	}
	return price
}

func (d *DryRunSubmitter) log(msg string, args ...any) {
	if d.logger != nil {
		d.logger.Info(msg, args...)
	}
}

func (d *DryRunSubmitter) inc(name string) {
	if d.tracker != nil {
		d.tracker.Counter(name).Add(1)
	}
}
