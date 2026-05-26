// Package pairing defines the canonical round-trip and leg-pairing model.
// It provides domain types, FIFO matching rules, and deterministic pairing
// of entry/exit legs from existing execution data.
//
// S480: Domain types, matching rules, and invariants.
//
// Guard rails:
//   - No OMS expansion; pairing is a read-path classification.
//   - No position tracking; round-trips are historical trade outcomes.
//   - No new ClickHouse tables; pairing is computed from existing execution data.
//   - No write-path changes to execution pipeline.
//   - Additive only; zero changes to existing domain types.
package pairing

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"internal/domain/execution"
	"internal/domain/instrument"
)

// LegDirection distinguishes the role of a leg within a round-trip.
type LegDirection string

const (
	LegEntry LegDirection = "entry"
	LegExit  LegDirection = "exit"
)

// ValidLegDirection reports whether d is a recognized leg direction.
func ValidLegDirection(d LegDirection) bool {
	return d == LegEntry || d == LegExit
}

// PairingState represents the lifecycle state of a round-trip pairing attempt.
type PairingState string

const (
	StatePaired         PairingState = "paired"
	StateUnmatchedEntry PairingState = "unmatched_entry"
	StateUnmatchedExit  PairingState = "unmatched_exit"
)

// ValidPairingState reports whether s is a recognized pairing state.
func ValidPairingState(s PairingState) bool {
	return s == StatePaired || s == StateUnmatchedEntry || s == StateUnmatchedExit
}

// UnmatchedReason explains why a leg could not be paired.
type UnmatchedReason string

const (
	ReasonNone                    UnmatchedReason = ""
	ReasonNoExitFound             UnmatchedReason = "no_exit_found"
	ReasonNoEntryFound            UnmatchedReason = "no_entry_found"
	ReasonQuantityMismatchResidue UnmatchedReason = "quantity_mismatch_remainder"
	ReasonSessionBoundary         UnmatchedReason = "session_boundary"
	ReasonRejectedLeg             UnmatchedReason = "rejected_leg"
	ReasonCancelledLeg            UnmatchedReason = "cancelled_leg"
)

// Leg represents one side of a round-trip trade.
// A leg wraps the fill data from an ExecutionIntent with directional context.
//
// Per ADR-0021, the canonical instrument identity is carried in the
// Instrument field. Migrated from Symbol string in H-6.b”.
type Leg struct {
	// Direction indicates whether this is an entry or exit leg.
	Direction LegDirection `json:"direction"`

	// Side is the order side (buy/sell) from the execution intent.
	Side execution.Side `json:"side"`

	// Instrument is the canonical instrument identity (ADR-0021).
	// Propagated from ExecutionIntent.Instrument via IntentToLeg.
	Instrument instrument.CanonicalInstrument `json:"instrument"`

	// Source identifies the venue/segment (e.g. binance_spot, binance_futures).
	Source string `json:"source"`

	// Timeframe is the candle interval that produced the signal chain.
	Timeframe int `json:"timeframe"`

	// CorrelationID is the chain-wide trace identifier linking this leg
	// to its originating signal→decision→strategy→risk→execution chain.
	CorrelationID string `json:"correlation_id"`

	// Price is the fill price (decimal string, venue precision).
	Price string `json:"price"`

	// Quantity is the filled quantity (decimal string).
	Quantity string `json:"quantity"`

	// Fee is the trading commission charged (decimal string).
	Fee string `json:"fee"`

	// FeeAsset identifies the denomination of the fee.
	FeeAsset string `json:"fee_asset,omitempty"`

	// CostBasis is the total notional value (price * quantity or cumQuote).
	CostBasis string `json:"cost_basis"`

	// FeeSource indicates the provenance of fee data (S499).
	FeeSource execution.FeeSource `json:"fee_source,omitempty"`

	// Simulated is true for paper/dry-run fills.
	Simulated bool `json:"simulated"`

	// Timestamp is when the fill occurred.
	Timestamp time.Time `json:"timestamp"`
}

// VenueSymbol returns the lowercase venue-native symbol form
// (e.g., "btcusdt") derived from the canonical Instrument.
//
// TRANSITORY ADAPTER (H-6.b” → sunset H-6.f). See ADR-0021.
// Downstream readers that still reason in venue-native string
// terms (S472-style projections, JSON wire shapes that expose
// "symbol", etc.) consume this method during the transition.
func (l Leg) VenueSymbol() string {
	return strings.ToLower(string(l.Instrument.Base) + string(l.Instrument.Quote))
}

// RoundTrip represents a paired or unpaired trade lifecycle.
//
// A fully paired round-trip has both an entry and exit leg for the same
// symbol/source/segment. An unmatched round-trip has only one leg.
//
// Semantics:
//   - paired: both entry and exit present, P&L is realized and classifiable.
//   - unmatched_entry: entry exists without a corresponding exit; outcome is open/unresolved.
//   - unmatched_exit: exit exists without a corresponding entry; data gap or orphan.
type RoundTrip struct {
	// Entry is the opening leg (buy for long, sell for short). Nil if unmatched_exit.
	Entry *Leg `json:"entry,omitempty"`

	// Exit is the closing leg (sell for long, buy for short). Nil if unmatched_entry.
	Exit *Leg `json:"exit,omitempty"`

	// State indicates whether the round-trip is fully paired or has an unmatched leg.
	State PairingState `json:"state"`

	// UnmatchedReason explains why a leg could not be paired. Empty when State == paired.
	UnmatchedReason UnmatchedReason `json:"unmatched_reason,omitempty"`

	// MatchedQuantity is the quantity that was actually paired between entry and exit.
	// For a fully matched pair this equals min(entry.Quantity, exit.Quantity).
	// For unmatched legs this is "0".
	MatchedQuantity string `json:"matched_quantity"`

	// Symbol is the traded instrument (denormalized for query convenience).
	Symbol string `json:"symbol"`

	// Source is the venue/segment (denormalized for query convenience).
	Source string `json:"source"`
}

// IsPaired reports whether the round-trip has both entry and exit legs.
func (rt RoundTrip) IsPaired() bool {
	return rt.State == StatePaired
}

// IsOpen reports whether the round-trip has an entry but no exit (position is open).
func (rt RoundTrip) IsOpen() bool {
	return rt.State == StateUnmatchedEntry
}

// MatchingConfig controls the behavior of the FIFO matching algorithm.
type MatchingConfig struct {
	// AllowPartialMatch enables proportional matching when entry and exit
	// quantities differ. When true, the smaller quantity is fully matched
	// and the remainder produces an additional unmatched leg.
	AllowPartialMatch bool
}

// DefaultMatchingConfig returns the default matching configuration.
func DefaultMatchingConfig() MatchingConfig {
	return MatchingConfig{
		AllowPartialMatch: true,
	}
}

// legCandidate is an internal representation used during matching.
type legCandidate struct {
	leg              Leg
	remainingQty     float64
	originalIntentID string // CorrelationID for traceability
}

// MatchFIFO applies FIFO (first-in-first-out) matching to a set of legs.
//
// Matching rules (invariants):
//   - M1: Same instrument — entry.Instrument == exit.Instrument
//     (native Go struct equality on the canonical identity per ADR-0021;
//     strictly stronger than legacy venue-symbol equality, because Contract
//     type also discriminates).
//   - M2: Same source/segment — entry.Source == exit.Source
//   - M3: Opposite side — buy entry pairs with sell exit (long), sell entry pairs with buy exit (short)
//   - M4: Temporal ordering — entry.Timestamp <= exit.Timestamp (entry must precede or equal exit)
//   - M5: FIFO priority — earliest unmatched entry pairs with earliest eligible exit
//   - M6: One-to-one — each fill participates in at most one round-trip (no double-counting)
//   - M7: Deterministic — same input always produces same output
//
// Partial-fill handling:
//
//	When AllowPartialMatch is true and quantities differ, the matched quantity
//	is min(entry, exit). The remainder produces an additional unmatched leg
//	with reason quantity_mismatch_remainder.
func MatchFIFO(legs []Leg, cfg MatchingConfig) []RoundTrip {
	if len(legs) == 0 {
		return nil
	}

	// Separate and sort entries and exits by timestamp (FIFO).
	var entries, exits []legCandidate
	for _, leg := range legs {
		qty := parseFloat(leg.Quantity)
		if qty <= 0 {
			continue
		}
		c := legCandidate{
			leg:              leg,
			remainingQty:     qty,
			originalIntentID: leg.CorrelationID,
		}
		switch leg.Direction {
		case LegEntry:
			entries = append(entries, c)
		case LegExit:
			exits = append(exits, c)
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].leg.Timestamp.Before(entries[j].leg.Timestamp)
	})
	sort.SliceStable(exits, func(i, j int) bool {
		return exits[i].leg.Timestamp.Before(exits[j].leg.Timestamp)
	})

	var results []RoundTrip

	// FIFO matching: for each entry, find the earliest eligible exit.
	for i := range entries {
		if entries[i].remainingQty <= 0 {
			continue
		}

		for j := range exits {
			if exits[j].remainingQty <= 0 {
				continue
			}

			if !isEligiblePair(entries[i].leg, exits[j].leg) {
				continue
			}

			matchQty := minFloat(entries[i].remainingQty, exits[j].remainingQty)
			if matchQty <= 0 {
				continue
			}

			entryLeg := scaleLeg(entries[i].leg, matchQty, entries[i].remainingQty)
			exitLeg := scaleLeg(exits[j].leg, matchQty, exits[j].remainingQty)

			results = append(results, RoundTrip{
				Entry:           &entryLeg,
				Exit:            &exitLeg,
				State:           StatePaired,
				MatchedQuantity: formatFloat(matchQty),
				// S472-style projection bridge: RoundTrip.Symbol stays
				// string until H-6.b'' commit 3; populate via VenueSymbol()
				// projection on the migrated Leg.Instrument.
				Symbol: entries[i].leg.VenueSymbol(),
				Source: entries[i].leg.Source,
			})

			entries[i].remainingQty -= matchQty
			exits[j].remainingQty -= matchQty

			if !cfg.AllowPartialMatch {
				break
			}

			if entries[i].remainingQty <= 0 {
				break
			}
		}
	}

	// Collect unmatched entries.
	for _, e := range entries {
		if e.remainingQty > floatEpsilon {
			leg := scaleLeg(e.leg, e.remainingQty, parseFloat(e.leg.Quantity))
			results = append(results, RoundTrip{
				Entry:           &leg,
				State:           StateUnmatchedEntry,
				UnmatchedReason: ReasonNoExitFound,
				MatchedQuantity: "0",
				// S472 bridge — see commit 2 of H-6.b''.
				Symbol: e.leg.VenueSymbol(),
				Source: e.leg.Source,
			})
		}
	}

	// Collect unmatched exits.
	for _, e := range exits {
		if e.remainingQty > floatEpsilon {
			leg := scaleLeg(e.leg, e.remainingQty, parseFloat(e.leg.Quantity))
			results = append(results, RoundTrip{
				Exit:            &leg,
				State:           StateUnmatchedExit,
				UnmatchedReason: ReasonNoEntryFound,
				MatchedQuantity: "0",
				// S472 bridge — see commit 2 of H-6.b''.
				Symbol: e.leg.VenueSymbol(),
				Source: e.leg.Source,
			})
		}
	}

	return results
}

// IntentToLeg converts an ExecutionIntent into a Leg for pairing.
// The direction is inferred from the side and the strategy direction context.
//
// For a long strategy: buy = entry, sell = exit.
// For a short strategy: sell = entry, buy = exit.
//
// The strategyDirection parameter disambiguates: "long" or "short".
// If empty, the convention defaults to long (buy=entry, sell=exit).
func IntentToLeg(intent execution.ExecutionIntent, strategyDirection string) Leg {
	dir := inferDirection(intent.Side, strategyDirection)

	// Aggregate fills into a single leg.
	var price, qty, fee, feeAsset, costBasis string
	var feeSource execution.FeeSource
	var simulated bool
	var ts time.Time

	if len(intent.Fills) > 0 {
		price = intent.Fills[0].Price
		feeAsset = intent.Fills[0].FeeAsset
		feeSource = intent.Fills[0].FeeSource
		ts = intent.Fills[0].Timestamp

		var totalQty, totalFee, totalCost float64
		for _, f := range intent.Fills {
			totalQty += parseFloat(f.Quantity)
			totalFee += parseFloat(f.Fee)
			totalCost += parseFloat(f.CostBasis)
			if f.Simulated {
				simulated = true
			}
		}
		qty = formatFloat(totalQty)
		fee = formatFloat(totalFee)
		costBasis = formatFloat(totalCost)

		// Weighted average price if multiple fills.
		if len(intent.Fills) > 1 && totalQty > 0 {
			price = formatFloat(totalCost / totalQty)
		}
	} else {
		qty = intent.Quantity
		price = "0"
		fee = "0"
		costBasis = "0"
		ts = intent.Timestamp
	}

	return Leg{
		Direction: dir,
		Side:      intent.Side,
		// Passthrough: ExecutionIntent already carries the canonical
		// Instrument (migrated in H-6.b'). No reconstruction here —
		// the regression-shape from commit 37f8ddd is avoided by
		// design.
		Instrument:    intent.Instrument,
		Source:        intent.Source,
		Timeframe:     intent.Timeframe,
		CorrelationID: intent.CorrelationID,
		Price:         price,
		Quantity:      qty,
		Fee:           fee,
		FeeAsset:      feeAsset,
		CostBasis:     costBasis,
		FeeSource:     feeSource,
		Simulated:     simulated,
		Timestamp:     ts,
	}
}

// PairingResult summarizes the outcome of a matching run.
type PairingResult struct {
	RoundTrips       []RoundTrip `json:"round_trips"`
	TotalEntries     int         `json:"total_entries"`
	TotalExits       int         `json:"total_exits"`
	PairedCount      int         `json:"paired_count"`
	UnmatchedEntries int         `json:"unmatched_entries"`
	UnmatchedExits   int         `json:"unmatched_exits"`
	ResolvedRate     float64     `json:"resolved_rate"`
}

// Summarize computes a PairingResult from a set of round-trips.
func Summarize(rts []RoundTrip) PairingResult {
	result := PairingResult{RoundTrips: rts}
	for _, rt := range rts {
		switch rt.State {
		case StatePaired:
			result.PairedCount++
			result.TotalEntries++
			result.TotalExits++
		case StateUnmatchedEntry:
			result.UnmatchedEntries++
			result.TotalEntries++
		case StateUnmatchedExit:
			result.UnmatchedExits++
			result.TotalExits++
		}
	}
	total := result.PairedCount + result.UnmatchedEntries + result.UnmatchedExits
	if total > 0 {
		result.ResolvedRate = float64(result.PairedCount) / float64(total)
	}
	return result
}

// --- internal helpers ---

func isEligiblePair(entry, exit Leg) bool {
	// M1: same canonical instrument identity (ADR-0021).
	// Native Go struct equality — CanonicalInstrument is composed of
	// three string-typed components (Base, Quote, Contract) and is
	// comparable by construction.
	if entry.Instrument != exit.Instrument {
		return false
	}
	// M2: same source/segment.
	if entry.Source != exit.Source {
		return false
	}
	// M3: opposite side.
	if !isOppositeSide(entry.Side, exit.Side) {
		return false
	}
	// M4: temporal ordering (entry <= exit).
	if exit.Timestamp.Before(entry.Timestamp) {
		return false
	}
	return true
}

func isOppositeSide(a, b execution.Side) bool {
	return (a == execution.SideBuy && b == execution.SideSell) ||
		(a == execution.SideSell && b == execution.SideBuy)
}

func inferDirection(side execution.Side, strategyDirection string) LegDirection {
	switch strategyDirection {
	case "short":
		if side == execution.SideSell {
			return LegEntry
		}
		return LegExit
	default: // "long" or empty
		if side == execution.SideBuy {
			return LegEntry
		}
		return LegExit
	}
}

// scaleLeg creates a leg with adjusted quantity and proportional cost/fee.
func scaleLeg(original Leg, matchQty, totalQty float64) Leg {
	if totalQty <= 0 || matchQty >= totalQty {
		return original
	}
	ratio := matchQty / totalQty
	scaled := original
	scaled.Quantity = formatFloat(matchQty)
	scaled.CostBasis = formatFloat(parseFloat(original.CostBasis) * ratio)
	scaled.Fee = formatFloat(parseFloat(original.Fee) * ratio)
	return scaled
}

const floatEpsilon = 1e-12

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func parseFloat(s string) float64 {
	var v float64
	_, _ = fmt.Sscanf(s, "%f", &v)
	return v
}

func formatFloat(v float64) string {
	return fmt.Sprintf("%.8f", v)
}
