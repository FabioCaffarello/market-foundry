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

// PaperVenueAdapter implements ports.VenuePort for simulated (paper) venue execution.
// It instantly accepts and fills orders without contacting any exchange.
// All fills are marked as simulated. Used by the execute binary in paper mode.
type PaperVenueAdapter struct {
	fillDelay time.Duration
}

// NewPaperVenueAdapter creates a paper venue adapter.
// fillDelay may be zero for instant fills (typical for testing).
func NewPaperVenueAdapter(fillDelay time.Duration) *PaperVenueAdapter {
	return &PaperVenueAdapter{fillDelay: fillDelay}
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

	filled := intent
	filled.Status = domainexec.StatusFilled
	filled.FilledQuantity = intent.Quantity
	filled.Fills = []domainexec.FillRecord{
		{
			Price:     "0",
			Quantity:  intent.Quantity,
			Fee:       "0",
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

func newVenueOrderID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return "paper-" + hex.EncodeToString(raw[:])
}
