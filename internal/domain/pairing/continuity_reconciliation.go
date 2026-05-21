// Package pairing — continuity_reconciliation extends reconciliation with
// cross-session data-quality flags and carryover-boundary awareness.
//
// S496: Adds reconciliation flags specific to cross-session round-trips,
// enabling operators to distinguish data-quality issues that arise from
// session boundaries versus structural data gaps.
//
// Guard rails:
//   - No new ClickHouse tables; reconciliation is a read-path computation.
//   - No write-path changes.
//   - Additive only; zero changes to existing reconciliation types.
package pairing

import (
	"internal/domain/effectiveness"
)

// Cross-session-specific reconciliation flags.
const (
	// FlagCrossSession indicates entry and exit legs originate from
	// different sessions. Not a defect — it signals the round-trip was
	// resolved via cross-session matching and may warrant extra review.
	FlagCrossSession ReconciliationFlag = "cross_session"

	// FlagBoundaryCarryover indicates one or both legs sat at a session
	// boundary before being resolved. The time gap between entry and exit
	// may include idle time between sessions.
	FlagBoundaryCarryover ReconciliationFlag = "boundary_carryover"

	// FlagCrossSessionFeeGap indicates fee data is missing or zero on a
	// cross-session round-trip where both legs have fills. This is distinct
	// from intra-session FlagFeeGap because cross-session pairs may
	// aggregate fees from different session contexts.
	FlagCrossSessionFeeGap ReconciliationFlag = "cross_session_fee_gap"
)

// ContinuityReconciliationResult extends ReconciliationResult with
// cross-session-specific context for continuity review.
type ContinuityReconciliationResult struct {
	ReconciliationResult

	// Continuity is the ContinuityState of the originating round-trip.
	Continuity ContinuityState `json:"continuity"`

	// CrossSession is true when entry and exit come from different sessions.
	CrossSession bool `json:"cross_session"`

	// EntrySessionID is the session that produced the entry leg.
	EntrySessionID string `json:"entry_session_id,omitempty"`

	// ExitSessionID is the session that produced the exit leg.
	ExitSessionID string `json:"exit_session_id,omitempty"`

	// CarryoverReliable is true when the cross-session round-trip has
	// both fee and P&L reliability AND no boundary-specific flags.
	CarryoverReliable bool `json:"carryover_reliable"`

	// HaltedOrigin is true when at least one leg comes from a halted session.
	// S500: Halted sessions may have incomplete data.
	HaltedOrigin bool `json:"halted_origin,omitempty"`
}

// LifecycleCloseContext provides optional metadata about the session(s) that
// produced the legs in a round-trip. This enables reconciliation to detect
// edge cases specific to lifecycle close boundaries.
//
// S500: Additive context for lifecycle close hardening. Not required for
// basic reconciliation — when nil, lifecycle-close-specific flags are skipped.
type LifecycleCloseContext struct {
	// EntrySessionHalted is true if the entry leg's session was halted.
	EntrySessionHalted bool

	// ExitSessionHalted is true if the exit leg's session was halted.
	ExitSessionHalted bool

	// EntryNonTerminalAtClose is true if the entry leg's execution intent
	// was in a non-terminal state when its session closed.
	EntryNonTerminalAtClose bool

	// ExitNonTerminalAtClose is true if the exit leg's execution intent
	// was in a non-terminal state when its session closed.
	ExitNonTerminalAtClose bool
}

// ReconcileCrossSessionRoundTrip produces a ContinuityReconciliationResult
// for a CrossSessionRoundTrip. It applies the standard reconciliation first,
// then adds cross-session-specific flags.
//
// S500: The optional lcCtx parameter enables lifecycle-close-specific flags.
// Pass nil when lifecycle close context is not available — the function
// gracefully skips those checks.
//
// This is a pure function — no side effects or I/O.
func ReconcileCrossSessionRoundTrip(csrt CrossSessionRoundTrip, attr *effectiveness.Attribution, lcCtx ...*LifecycleCloseContext) ContinuityReconciliationResult {
	// Run standard reconciliation on the underlying round-trip.
	base := ReconcileRoundTrip(csrt.RoundTrip, attr)

	result := ContinuityReconciliationResult{
		ReconciliationResult: base,
		Continuity:           csrt.Continuity,
		CrossSession:         csrt.CrossSession,
		EntrySessionID:       csrt.EntrySessionID,
		ExitSessionID:        csrt.ExitSessionID,
	}

	// Add cross-session flag when legs span sessions.
	if csrt.CrossSession {
		result.Flags = appendIfAbsent(result.Flags, FlagCrossSession)
	}

	// Add boundary carryover flag when the round-trip was artificial_unresolved
	// before cross-session matching resolved it, or when it was resolved
	// and spans sessions (i.e., one leg waited at a boundary).
	if csrt.CrossSession && csrt.Continuity == ContinuityResolved {
		result.Flags = appendIfAbsent(result.Flags, FlagBoundaryCarryover)
	}

	// Cross-session fee gap: paired, cross-session, but fees missing on one or both legs.
	if csrt.CrossSession && csrt.IsPaired() {
		if !hasFee(csrt.Entry) || !hasFee(csrt.Exit) {
			result.Flags = appendIfAbsent(result.Flags, FlagCrossSessionFeeGap)
		}
	}

	// S500: Lifecycle close context flags.
	var ctx *LifecycleCloseContext
	if len(lcCtx) > 0 {
		ctx = lcCtx[0]
	}
	if ctx != nil {
		if ctx.EntryNonTerminalAtClose || ctx.ExitNonTerminalAtClose {
			result.Flags = appendIfAbsent(result.Flags, FlagNonTerminalAtClose)
		}
		if ctx.EntrySessionHalted || ctx.ExitSessionHalted {
			result.Flags = appendIfAbsent(result.Flags, FlagHaltedSessionOrigin)
			result.HaltedOrigin = true
		}
	}

	// Recompute clean status since we may have added flags.
	result.Clean = len(result.Flags) == 0

	// Carryover reliability: both base reliability checks pass AND
	// no boundary-specific quality flags.
	result.CarryoverReliable = base.FeeReliable && base.PnLReliable && !csrt.CrossSession
	if csrt.CrossSession {
		// Cross-session pairs can still be reliable if both fee and P&L
		// pass and there is no cross-session fee gap.
		hasCrossFeeGap := false
		for _, f := range result.Flags {
			if f == FlagCrossSessionFeeGap {
				hasCrossFeeGap = true
				break
			}
		}
		result.CarryoverReliable = base.FeeReliable && base.PnLReliable && !hasCrossFeeGap
	}

	// S500: Halted origin or non-terminal at close degrades carryover reliability.
	if result.HaltedOrigin || hasFlag(result.Flags, FlagNonTerminalAtClose) {
		result.CarryoverReliable = false
	}

	return result
}

// hasFlag reports whether flags contains the given flag.
func hasFlag(flags []ReconciliationFlag, flag ReconciliationFlag) bool {
	for _, f := range flags {
		if f == flag {
			return true
		}
	}
	return false
}

// appendIfAbsent appends flag to flags only if it is not already present.
func appendIfAbsent(flags []ReconciliationFlag, flag ReconciliationFlag) []ReconciliationFlag {
	for _, f := range flags {
		if f == flag {
			return flags
		}
	}
	return append(flags, flag)
}

// ContinuityReconciliationSummary aggregates reconciliation results across
// a set of cross-session round-trips for the review surface.
type ContinuityReconciliationSummary struct {
	// Total is the number of round-trips reconciled.
	Total int `json:"total"`

	// CleanCount is the number of round-trips with zero flags.
	CleanCount int `json:"clean_count"`

	// FlaggedCount is the number of round-trips with at least one flag.
	FlaggedCount int `json:"flagged_count"`

	// CrossSessionCount is the number of round-trips that span sessions.
	CrossSessionCount int `json:"cross_session_count"`

	// BoundaryCarryoverCount is the number of round-trips with boundary
	// carryover (resolved after crossing a session boundary).
	BoundaryCarryoverCount int `json:"boundary_carryover_count"`

	// FlagCounts maps flag name to occurrence count.
	FlagCounts map[string]int `json:"flag_counts"`

	// CarryoverReliableCount is the number of cross-session round-trips
	// with fully reliable fee and P&L data.
	CarryoverReliableCount int `json:"carryover_reliable_count"`

	// FeeReliableCount is the number of round-trips with reliable fees.
	FeeReliableCount int `json:"fee_reliable_count"`

	// PnLReliableCount is the number of round-trips with reliable P&L.
	PnLReliableCount int `json:"pnl_reliable_count"`
}

// SummarizeContinuityReconciliation computes a summary from a set of
// continuity reconciliation results. Pure function.
func SummarizeContinuityReconciliation(results []ContinuityReconciliationResult) ContinuityReconciliationSummary {
	s := ContinuityReconciliationSummary{
		FlagCounts: make(map[string]int),
	}

	for _, r := range results {
		s.Total++

		if r.Clean {
			s.CleanCount++
		} else {
			s.FlaggedCount++
		}

		if r.CrossSession {
			s.CrossSessionCount++
		}

		for _, f := range r.Flags {
			s.FlagCounts[string(f)]++
			if f == FlagBoundaryCarryover {
				s.BoundaryCarryoverCount++
			}
		}

		if r.CarryoverReliable {
			s.CarryoverReliableCount++
		}
		if r.FeeReliable {
			s.FeeReliableCount++
		}
		if r.PnLReliable {
			s.PnLReliableCount++
		}
	}

	return s
}
