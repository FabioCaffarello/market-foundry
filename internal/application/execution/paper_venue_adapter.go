package execution

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/problem"
)

// paperDefaultFillPrice is used when no PriceSource is configured or lookup fails.
const paperDefaultFillPrice = "0"

// paperPriceLookupTimeout bounds the price-source read on the paper fill
// path. A stalled PriceSource must not block paper-mode order fills.
const paperPriceLookupTimeout = 2 * time.Second

// PaperVenueAdapter implements ports.VenuePort for simulated (paper) venue execution.
// It instantly accepts and fills orders without contacting any exchange.
// All fills are marked as simulated. Used by the execute binary in paper mode.
type PaperVenueAdapter struct {
	fillDelay   time.Duration
	priceSource ports.PriceSource
}

// NewPaperVenueAdapter creates a paper venue adapter.
// fillDelay may be zero for instant fills (typical for testing).
func NewPaperVenueAdapter(fillDelay time.Duration) *PaperVenueAdapter {
	return &PaperVenueAdapter{fillDelay: fillDelay}
}

// WithPriceSource attaches a price lookup for realistic fill prices.
// When set, fills use the last observed close price instead of "0".
func (a *PaperVenueAdapter) WithPriceSource(ps ports.PriceSource) *PaperVenueAdapter {
	a.priceSource = ps
	return a
}

// SubmitOrder simulates order submission: generates a venue order ID, transitions the intent
// through submitted → sent → accepted → filled, and returns a filled receipt.
func (a *PaperVenueAdapter) SubmitOrder(_ context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	intent := req.Intent

	// No-action intents: nothing to fill.
	if intent.Side == domainexec.SideNone {
		return ports.VenueOrderReceipt{
			VenueOrderID: newVenueOrderID(),
			Status:       domainexec.StatusAccepted,
			Intent:       intent,
		}, nil
	}

	if a.fillDelay > 0 {
		time.Sleep(a.fillDelay)
	}

	// Fresh bounded context — SubmitOrder discards its ctx parameter (paper
	// adapter must not fail just because upstream was cancelled); the
	// price-source read is bounded here independently.
	priceCtx, priceCancel := context.WithTimeout(context.Background(), paperPriceLookupTimeout)
	fillPrice := a.resolvePrice(priceCtx, intent.Source, intent.VenueSymbol(), intent.Timeframe) //nolint:contextcheck // deliberate fresh ctx — see comment above
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

	return ports.VenueOrderReceipt{
		VenueOrderID: newVenueOrderID(),
		Status:       domainexec.StatusFilled,
		Intent:       filled,
	}, nil
}

// resolvePrice returns the best-effort last price for the symbol.
// Falls back to paperDefaultFillPrice ("0") when no PriceSource is configured or on error.
func (a *PaperVenueAdapter) resolvePrice(ctx context.Context, source, symbol string, timeframe int) string {
	if a.priceSource == nil {
		return paperDefaultFillPrice
	}
	price, prob := a.priceSource.LastPrice(ctx, source, symbol, timeframe)
	if prob != nil || price == "" {
		return paperDefaultFillPrice
	}
	return price
}

func newVenueOrderID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return "paper-" + hex.EncodeToString(raw[:])
}
