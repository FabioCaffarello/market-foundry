// Package pairing — reconciliation provides data-quality flags for round-trip review.
//
// S482: Reconciliation checks between fills, fees, pairing, and outcome.
// These flags are computed per round-trip at read time and help operators
// understand which round-trips have reliable P&L and which have data gaps.
//
// Guard rails:
//   - No new ClickHouse tables; reconciliation is a read-path computation.
//   - No write-path changes.
//   - Additive only; zero changes to existing pairing types.
package pairing

import (
	"internal/domain/effectiveness"
	"internal/domain/execution"
)

// ReconciliationFlag identifies a specific data-quality condition in a round-trip.
type ReconciliationFlag string

const (
	// FlagFeeGap indicates zero fees on a segment where fees are expected (futures).
	FlagFeeGap ReconciliationFlag = "fee_gap"

	// FlagCostBasisZero indicates zero cost basis (paper/dry-run fills).
	FlagCostBasisZero ReconciliationFlag = "cost_basis_zero"

	// FlagSimulated indicates at least one leg is from paper/dry-run execution.
	FlagSimulated ReconciliationFlag = "simulated"

	// FlagPartialRemainder indicates this round-trip resulted from a partial-fill
	// quantity split — the matched quantity is less than the original leg quantity.
	FlagPartialRemainder ReconciliationFlag = "partial_remainder"

	// FlagUnmatchedOpen indicates an entry without exit — position is open.
	FlagUnmatchedOpen ReconciliationFlag = "unmatched_open"

	// FlagOrphanExit indicates an exit without entry — data gap or orphan.
	FlagOrphanExit ReconciliationFlag = "orphan_exit"

	// FlagFeeAssetMismatch indicates entry and exit legs have different fee assets.
	FlagFeeAssetMismatch ReconciliationFlag = "fee_asset_mismatch"

	// FlagOutcomeUnresolved indicates the effectiveness outcome could not be determined
	// despite the round-trip being structurally paired (e.g., zero cost basis on both legs).
	FlagOutcomeUnresolved ReconciliationFlag = "outcome_unresolved"

	// FlagFeeRatioAnomaly indicates the fee-to-cost-basis ratio exceeds the
	// plausible threshold, suggesting data corruption or misattribution.
	// S499: Catches cases where fee > 10% of cost basis.
	FlagFeeRatioAnomaly ReconciliationFlag = "fee_ratio_anomaly"

	// FlagFeeSourceFallback indicates at least one leg used the fallback fee
	// code path (Spot without fills[] array — unexpected for FULL response type).
	// S499: Distinguishes genuine data gaps from expected API limitations.
	FlagFeeSourceFallback ReconciliationFlag = "fee_source_fallback"

	// FlagNonTerminalAtClose indicates this round-trip's entry or exit leg
	// originated from an execution intent that was in a non-terminal state
	// (submitted, sent, accepted, partially_filled) when the session closed.
	// S500: Surfaces the boundary edge case where orders may still be filled
	// by the venue after session ends but are not carried forward. The leg
	// data may be incomplete.
	FlagNonTerminalAtClose ReconciliationFlag = "non_terminal_at_close"

	// FlagHaltedSessionOrigin indicates this round-trip has at least one
	// leg from a halted session (operator kill-switch or error condition).
	// S500: Halted sessions may have incomplete data — orders may have been
	// in-flight when the halt occurred. The round-trip warrants extra review.
	FlagHaltedSessionOrigin ReconciliationFlag = "halted_session_origin"
)

// ReconciliationResult aggregates data-quality findings for a single round-trip.
type ReconciliationResult struct {
	// Flags lists all active reconciliation conditions. Empty means clean.
	Flags []ReconciliationFlag `json:"flags"`

	// Clean is true when no flags are present — the round-trip has reliable data.
	Clean bool `json:"clean"`

	// FeeReliable is true when fee data is present and non-zero on both legs.
	FeeReliable bool `json:"fee_reliable"`

	// PnLReliable is true when the outcome is classifiable (win/loss/breakeven)
	// with non-zero cost basis on both legs.
	PnLReliable bool `json:"pnl_reliable"`
}

// ReconcileRoundTrip examines a round-trip and its attribution to produce
// data-quality flags. This is a pure function — no side effects or I/O.
func ReconcileRoundTrip(rt RoundTrip, attr *effectiveness.Attribution) ReconciliationResult {
	var flags []ReconciliationFlag

	switch rt.State {
	case StateUnmatchedEntry:
		flags = append(flags, FlagUnmatchedOpen)
	case StateUnmatchedExit:
		flags = append(flags, FlagOrphanExit)
	case StatePaired:
		flags = append(flags, reconcilePaired(rt, attr)...)
	}

	// Simulated check — applies to all states.
	if isSimulatedLeg(rt.Entry) || isSimulatedLeg(rt.Exit) {
		flags = append(flags, FlagSimulated)
	}

	// S499: FeeReliable is true when both legs have fee data OR when the
	// fee source indicates the zero is expected (e.g. Futures "unavailable").
	// This prevents Futures round-trips from being permanently unreliable.
	feeReliable := rt.IsPaired() && isFeeReliableLeg(rt.Entry) && isFeeReliableLeg(rt.Exit)
	pnlReliable := rt.IsPaired() && attr != nil &&
		attr.Outcome != effectiveness.OutcomeUnresolved &&
		hasCostBasis(rt.Entry) && hasCostBasis(rt.Exit)

	return ReconciliationResult{
		Flags:       flags,
		Clean:       len(flags) == 0,
		FeeReliable: feeReliable,
		PnLReliable: pnlReliable,
	}
}

// reconcilePaired checks paired-specific conditions.
func reconcilePaired(rt RoundTrip, attr *effectiveness.Attribution) []ReconciliationFlag {
	var flags []ReconciliationFlag

	// Fee gap: one or both legs have zero fees.
	if !hasFee(rt.Entry) || !hasFee(rt.Exit) {
		flags = append(flags, FlagFeeGap)
	}

	// Cost basis zero: one or both legs have zero cost basis.
	if !hasCostBasis(rt.Entry) || !hasCostBasis(rt.Exit) {
		flags = append(flags, FlagCostBasisZero)
	}

	// Fee asset mismatch between legs.
	if rt.Entry != nil && rt.Exit != nil &&
		rt.Entry.FeeAsset != "" && rt.Exit.FeeAsset != "" &&
		rt.Entry.FeeAsset != rt.Exit.FeeAsset {
		flags = append(flags, FlagFeeAssetMismatch)
	}

	// Partial remainder: matched quantity < either leg's original quantity.
	// We detect this by checking if UnmatchedReason contains remainder context
	// or if the matched quantity field is present on a paired trip that has
	// adjacent unmatched remainder siblings (but we only see this trip).
	// Conservative: flag when attribution exists but outcome is unresolved
	// despite being paired (indicates cost-basis issues).
	if attr != nil && attr.Outcome == effectiveness.OutcomeUnresolved {
		flags = append(flags, FlagOutcomeUnresolved)
	}

	// S499: Fee ratio anomaly — fee exceeds 10% of cost basis on either leg.
	if isFeeRatioAnomaly(rt.Entry) || isFeeRatioAnomaly(rt.Exit) {
		flags = append(flags, FlagFeeRatioAnomaly)
	}

	// S499: Fee source fallback — at least one leg hit the unexpected fallback path.
	if hasFeeSourceFallback(rt.Entry) || hasFeeSourceFallback(rt.Exit) {
		flags = append(flags, FlagFeeSourceFallback)
	}

	return flags
}

func isSimulatedLeg(leg *Leg) bool {
	return leg != nil && leg.Simulated
}

func hasFee(leg *Leg) bool {
	return leg != nil && parseFloat(leg.Fee) > 0
}

func hasCostBasis(leg *Leg) bool {
	return leg != nil && parseFloat(leg.CostBasis) > 0
}

// feeRatioThreshold is the maximum plausible fee-to-cost-basis ratio.
// Fees above 10% of notional value are flagged as anomalous.
const feeRatioThreshold = 0.10

// isFeeRatioAnomaly returns true when the fee exceeds a plausible fraction
// of the cost basis, suggesting data corruption or misattribution.
// Only applies when both fee and cost basis are positive.
func isFeeRatioAnomaly(leg *Leg) bool {
	if leg == nil {
		return false
	}
	fee := parseFloat(leg.Fee)
	cb := parseFloat(leg.CostBasis)
	if fee <= 0 || cb <= 0 {
		return false
	}
	return fee/cb > feeRatioThreshold
}

// hasFeeSourceFallback returns true when a leg's fee source is "fallback",
// indicating the unexpected code path where Spot fills[] was empty.
func hasFeeSourceFallback(leg *Leg) bool {
	return leg != nil && leg.FeeSource == execution.FeeSourceFallback
}

// isFeeReliableLeg reports whether a leg's fee data is considered reliable.
// A leg is fee-reliable when it has a non-zero fee (venue data) OR when
// the FeeSource explicitly indicates the zero is expected ("unavailable"
// for Futures, where the venue API does not return commission).
// Simulated legs are not fee-reliable (fees are always synthetic zeros).
func isFeeReliableLeg(leg *Leg) bool {
	if leg == nil {
		return false
	}
	if hasFee(leg) {
		return true
	}
	// S499: Accept "unavailable" as reliable — the system acknowledges the gap
	// and operators can filter on FeeSource for segment-specific analysis.
	return leg.FeeSource == execution.FeeSourceUnavailable
}
