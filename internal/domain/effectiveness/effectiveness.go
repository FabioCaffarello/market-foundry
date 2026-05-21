// Package effectiveness defines the canonical effectiveness model for decision chains.
// It provides outcome classification (win/loss/breakeven/unresolved), P&L attribution,
// and deterministic computation from existing FillRecord data.
//
// S475: Domain types and classification rules.
// S476: Read-path computation integrated into analytical surfaces.
//
// Guard rails:
//   - No new ClickHouse tables; effectiveness is a read-path computation.
//   - No portfolio analytics; scoped to individual decision chains.
//   - No risk-adjusted metrics; raw P&L and win/loss only.
//   - Additive only; zero changes to existing domain types.
package effectiveness

import (
	"fmt"
	"math"
	"strconv"

	"internal/domain/execution"
)

// Outcome represents the effectiveness classification of a decision chain.
type Outcome string

const (
	OutcomeWin        Outcome = "win"
	OutcomeLoss       Outcome = "loss"
	OutcomeBreakeven  Outcome = "breakeven"
	OutcomeUnresolved Outcome = "unresolved"
)

// ValidOutcome reports whether o is a recognized effectiveness outcome.
func ValidOutcome(o Outcome) bool {
	return o == OutcomeWin || o == OutcomeLoss || o == OutcomeBreakeven || o == OutcomeUnresolved
}

// BreakevenThreshold is the absolute P&L tolerance below which an outcome is
// classified as breakeven rather than win or loss. Expressed in quote-asset units.
const BreakevenThreshold = 0.0001

// Attribution links an effectiveness outcome to its originating decision chain metadata.
// All fields are derived from existing domain types — no new write-path data.
type Attribution struct {
	// Outcome classification.
	Outcome Outcome `json:"outcome"`

	// P&L fields (quote-asset units).
	RealizedPnL float64 `json:"realized_pnl"`
	TotalFees   float64 `json:"total_fees"`
	GrossPnL    float64 `json:"gross_pnl"`
	NetPnL      float64 `json:"net_pnl"`

	// Cost basis from fills.
	EntryCostBasis float64 `json:"entry_cost_basis"`
	ExitCostBasis  float64 `json:"exit_cost_basis,omitempty"`
	FillCount      int     `json:"fill_count"`

	// Decision chain context (carried from execution's RiskInput).
	CorrelationID    string `json:"correlation_id"`
	DecisionType     string `json:"decision_type,omitempty"`
	DecisionSeverity string `json:"decision_severity,omitempty"`
	StrategyType     string `json:"strategy_type,omitempty"`
	Side             string `json:"side"`
	Symbol           string `json:"symbol"`
	Source           string `json:"source"`
	Timeframe        int    `json:"timeframe"`

	// Execution status that produced this classification.
	ExecutionStatus string `json:"execution_status"`
	Simulated       bool   `json:"simulated"`
}

// Classify computes the effectiveness outcome for an execution intent.
//
// Classification rules:
//   - Rejected orders: excluded (no fill, no outcome) — returns nil.
//   - Cancelled before any fill: classified as unresolved.
//   - Filled or partially_filled: P&L computed from fill data.
//   - Non-terminal status (submitted/sent/accepted): unresolved.
//   - Single-leg fills without exit: uses cost-basis-only attribution, outcome unresolved.
//
// P&L computation:
//   - For buy fills: net_pnl = -cost_basis - fees (capital deployed, no exit price in single-leg).
//   - Single-leg fills are always unresolved since there is no paired exit within session scope.
//   - The system records cost basis and fees for future exit pairing.
func Classify(intent execution.ExecutionIntent) *Attribution {
	// Rejected orders produce no effectiveness outcome.
	if intent.Status == execution.StatusRejected {
		return nil
	}

	attr := &Attribution{
		CorrelationID:    intent.CorrelationID,
		DecisionType:     intent.Risk.Type,
		DecisionSeverity: intent.Risk.DecisionSeverity,
		StrategyType:     intent.Risk.StrategyType,
		Side:             string(intent.Side),
		Symbol:           intent.Symbol,
		Source:           intent.Source,
		Timeframe:        intent.Timeframe,
		ExecutionStatus:  string(intent.Status),
		FillCount:        len(intent.Fills),
	}

	// Non-terminal or cancelled-before-fill: unresolved.
	if !intent.Status.IsTerminal() {
		attr.Outcome = OutcomeUnresolved
		return attr
	}

	if intent.Status == execution.StatusCancelled && len(intent.Fills) == 0 {
		attr.Outcome = OutcomeUnresolved
		return attr
	}

	// Compute fill-based metrics.
	var totalCostBasis, totalFees float64
	var simulated bool
	for _, fill := range intent.Fills {
		cb := parseFloat(fill.CostBasis)
		fee := parseFloat(fill.Fee)
		totalCostBasis += cb
		totalFees += fee
		if fill.Simulated {
			simulated = true
		}
	}

	attr.EntryCostBasis = totalCostBasis
	attr.TotalFees = totalFees
	attr.Simulated = simulated

	// Single-leg attribution: fills exist but no paired exit within session scope.
	// P&L requires entry AND exit. With only one leg, outcome is unresolved.
	// We still record cost basis and fees for downstream pairing or session-end mark.
	//
	// Gross P&L = cost_basis (represents capital flow, sign depends on side).
	// Net P&L = gross - fees.
	//
	// For paper/dry-run fills with CostBasis=0, the outcome is always unresolved
	// because there's no meaningful P&L to classify.
	if totalCostBasis == 0 {
		attr.Outcome = OutcomeUnresolved
		attr.GrossPnL = 0
		attr.NetPnL = -totalFees
		return attr
	}

	// For filled orders, the cost basis IS the realized value.
	// In a single-session scope, a buy fill deploys capital and a sell fill recovers it.
	// Without a paired opposite-side fill, we cannot compute realized P&L.
	// The system classifies based on what is observable.
	attr.GrossPnL = totalCostBasis
	attr.NetPnL = totalCostBasis - totalFees

	attr.Outcome = classifyByPnL(attr.NetPnL, intent.Side)

	return attr
}

// ClassifyPair computes effectiveness for a paired entry/exit within the same session.
// This is the canonical round-trip P&L computation.
//
// Entry and exit must be for the same symbol and correlation chain.
// Entry is the initial fill (buy for long, sell for short).
// Exit is the closing fill (sell for long, buy for short).
func ClassifyPair(entry, exit execution.ExecutionIntent) *Attribution {
	if entry.Status == execution.StatusRejected || exit.Status == execution.StatusRejected {
		return nil
	}

	entryCost := sumCostBasis(entry.Fills)
	entryFees := sumFees(entry.Fills)
	exitCost := sumCostBasis(exit.Fills)
	exitFees := sumFees(exit.Fills)

	totalFees := entryFees + exitFees

	// Gross P&L: exit value - entry value.
	// For a long (buy entry, sell exit): gross = exitCost - entryCost.
	// For a short (sell entry, buy exit): gross = entryCost - exitCost.
	var grossPnL float64
	if entry.Side == execution.SideBuy {
		grossPnL = exitCost - entryCost
	} else {
		grossPnL = entryCost - exitCost
	}

	netPnL := grossPnL - totalFees

	simulated := isSimulated(entry.Fills) || isSimulated(exit.Fills)

	attr := &Attribution{
		Outcome:          classifyByNetPnL(netPnL),
		RealizedPnL:      netPnL,
		TotalFees:        totalFees,
		GrossPnL:         grossPnL,
		NetPnL:           netPnL,
		EntryCostBasis:   entryCost,
		ExitCostBasis:    exitCost,
		FillCount:        len(entry.Fills) + len(exit.Fills),
		CorrelationID:    entry.CorrelationID,
		DecisionType:     entry.Risk.Type,
		DecisionSeverity: entry.Risk.DecisionSeverity,
		StrategyType:     entry.Risk.StrategyType,
		Side:             string(entry.Side),
		Symbol:           entry.Symbol,
		Source:            entry.Source,
		Timeframe:        entry.Timeframe,
		ExecutionStatus:  string(exit.Status),
		Simulated:        simulated,
	}

	return attr
}

// classifyByPnL determines outcome from net P&L and side.
// For single-leg fills, all are unresolved regardless of P&L sign,
// because we don't have a paired exit.
func classifyByPnL(_ float64, _ execution.Side) Outcome {
	// Single-leg fills cannot determine win/loss without an exit.
	return OutcomeUnresolved
}

// classifyByNetPnL determines outcome from realized net P&L (round-trip).
func classifyByNetPnL(netPnL float64) Outcome {
	if math.Abs(netPnL) <= BreakevenThreshold {
		return OutcomeBreakeven
	}
	if netPnL > 0 {
		return OutcomeWin
	}
	return OutcomeLoss
}

// Helpers

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func sumCostBasis(fills []execution.FillRecord) float64 {
	var total float64
	for _, f := range fills {
		total += parseFloat(f.CostBasis)
	}
	return total
}

func sumFees(fills []execution.FillRecord) float64 {
	var total float64
	for _, f := range fills {
		total += parseFloat(f.Fee)
	}
	return total
}

func isSimulated(fills []execution.FillRecord) bool {
	for _, f := range fills {
		if f.Simulated {
			return true
		}
	}
	return false
}

// Explain returns a human-readable summary of the effectiveness attribution.
func (a *Attribution) Explain() string {
	if a == nil {
		return "No effectiveness data: execution did not produce classifiable fills."
	}

	switch a.Outcome {
	case OutcomeUnresolved:
		if a.FillCount == 0 {
			return fmt.Sprintf("Effectiveness unresolved: %s execution %s with no fills (status=%s).",
				a.Side, a.CorrelationID, a.ExecutionStatus)
		}
		return fmt.Sprintf("Effectiveness unresolved: %s execution has %d fill(s) with cost_basis=%.6f, fees=%.6f but no paired exit within session scope.",
			a.Side, a.FillCount, a.EntryCostBasis, a.TotalFees)

	case OutcomeWin:
		return fmt.Sprintf("Effectiveness: WIN. Realized P&L=%.6f (gross=%.6f, fees=%.6f). %d fill(s), %s %s.",
			a.NetPnL, a.GrossPnL, a.TotalFees, a.FillCount, a.Side, a.Symbol)

	case OutcomeLoss:
		return fmt.Sprintf("Effectiveness: LOSS. Realized P&L=%.6f (gross=%.6f, fees=%.6f). %d fill(s), %s %s.",
			a.NetPnL, a.GrossPnL, a.TotalFees, a.FillCount, a.Side, a.Symbol)

	case OutcomeBreakeven:
		return fmt.Sprintf("Effectiveness: BREAKEVEN. Realized P&L=%.6f (within threshold=%.4f). %d fill(s), %s %s.",
			a.NetPnL, BreakevenThreshold, a.FillCount, a.Side, a.Symbol)

	default:
		return fmt.Sprintf("Effectiveness: unknown outcome %q.", a.Outcome)
	}
}
