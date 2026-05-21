package execution

import (
	domainexec "internal/domain/execution"
)

// PaperFillSimulator transitions a submitted execution intent through the paper execution
// lifecycle: submitted → accepted → filled. Paper fills are instantaneous and simulated.
// Pure application logic — no I/O, no actor references, no NATS dependency.
type PaperFillSimulator struct{}

// SimulateFill applies the paper execution lifecycle to a submitted intent.
// For actionable intents (side = buy or sell), it produces a filled intent with a simulated fill record.
// For no-action intents (side = none), it returns the intent unchanged — there is nothing to fill.
// Returns false only if the intent is in an unexpected state for simulation.
func (s *PaperFillSimulator) SimulateFill(intent domainexec.ExecutionIntent) (domainexec.ExecutionIntent, bool) {
	if intent.Status != domainexec.StatusSubmitted {
		return intent, false
	}

	// No-action intents: nothing to fill.
	if intent.Side == domainexec.SideNone {
		return intent, true
	}

	// Paper fill: submitted → accepted → filled (instant).
	filled := intent
	filled.Status = domainexec.StatusFilled
	filled.FilledQuantity = intent.Quantity
	filled.Fills = []domainexec.FillRecord{
		{
			Price:     "0",
			Quantity:  intent.Quantity,
			Fee:       "0",
			FeeSource: domainexec.FeeSourceSimulated,
			Simulated: true,
			Timestamp: intent.Timestamp,
		},
	}

	return filled, true
}
