package analyticalclient

import (
	"internal/domain/instrument"

	"internal/domain/effectiveness"
)

// EffectivenessQuery is the request contract for batch effectiveness evaluation.
//
// S476: Answers Q-SE3 (effectiveness computable from existing data) and
// Q-SE4 (batch evaluation across decision cohorts).
//
// Lookup modes:
//   - CorrelationID+Symbol: single-chain effectiveness lookup.
//   - Source+Symbol+Timeframe: batch evaluation with optional filters.
//
// Filters allow narrowing by decision type, strategy type, severity, and
// effectiveness outcome (win/loss/breakeven/unresolved).
type EffectivenessQuery struct {
	CorrelationID string `json:"correlation_id,omitempty"` // single lookup

	// Batch lookup filters.
	Source        string                         `json:"source,omitempty"`
	Instrument    instrument.CanonicalInstrument `json:"instrument"`
	Timeframe     int                            `json:"timeframe,omitempty"`
	DecisionType  string                         `json:"decision_type,omitempty"` // filter by decision evaluator type
	StrategyType  string                         `json:"strategy_type,omitempty"` // filter by strategy resolver type
	Severity      string                         `json:"severity,omitempty"`      // filter by decision severity
	Effectiveness string                         `json:"effectiveness,omitempty"` // filter by outcome: win/loss/breakeven/unresolved
	Since         int64                          `json:"since,omitempty"`         // unix seconds, inclusive
	Until         int64                          `json:"until,omitempty"`         // unix seconds, inclusive
	Limit         int                            `json:"limit,omitempty"`         // default 20, max 100
}

// EffectivenessReply is the response contract for effectiveness queries.
type EffectivenessReply struct {
	Evaluations []effectiveness.Attribution `json:"evaluations"`
	Source      string                      `json:"source"` // always "clickhouse"
	Meta        EffectivenessMeta           `json:"meta"`
}

// EffectivenessMeta carries diagnostic signals for effectiveness queries.
type EffectivenessMeta struct {
	TotalMs         int64 `json:"total_ms"`
	EvaluationCount int   `json:"evaluation_count"`
	ChainsScanned   int   `json:"chains_scanned"`
	Excluded        int   `json:"excluded"` // rejected orders excluded from evaluation
}

// EffectivenessSummaryQuery is the request contract for cohort-level effectiveness
// aggregation and comparative analysis.
//
// S477: Answers Q-SE5 (comparative effectiveness analysis).
//
// GroupBy controls the response shape:
//   - Empty: returns a single CohortSummary for all matching evaluations.
//   - "decision_type", "strategy_type", "severity", "source": returns one
//     CohortSummary per distinct value of the grouping dimension.
//
// Filters narrow the cohort before aggregation. The same Source+Symbol+Timeframe
// partition key as EffectivenessQuery is required.
type EffectivenessSummaryQuery struct {
	Source       string                         `json:"source"`
	Instrument   instrument.CanonicalInstrument `json:"instrument"`
	Timeframe    int                            `json:"timeframe"`
	DecisionType string                         `json:"decision_type,omitempty"`
	StrategyType string                         `json:"strategy_type,omitempty"`
	Severity     string                         `json:"severity,omitempty"`
	Since        int64                          `json:"since,omitempty"`
	Until        int64                          `json:"until,omitempty"`
	Limit        int                            `json:"limit,omitempty"` // max chains to scan (default 100, max 300)

	// GroupBy enables comparative analysis. When set, the response contains one
	// CohortSummary per distinct value of the grouping dimension.
	// Valid values: "decision_type", "strategy_type", "severity", "source".
	GroupBy string `json:"group_by,omitempty"`
}

// ValidGroupBy reports whether g is a recognized grouping dimension.
func ValidGroupBy(g string) bool {
	return g == "" || g == "decision_type" || g == "strategy_type" || g == "severity" || g == "source"
}

// EffectivenessSummaryReply is the response contract for cohort aggregation.
//
// When GroupBy is empty, Cohorts contains exactly one entry with Key="all".
// When GroupBy is set, Cohorts contains one entry per distinct dimension value,
// sorted by evaluated count descending.
type EffectivenessSummaryReply struct {
	Cohorts []CohortSummary   `json:"cohorts"`
	Source  string            `json:"source"` // always "clickhouse"
	Meta    EffectivenessMeta `json:"meta"`
}

// CohortSummary is the aggregated effectiveness view for a group of decision chains.
//
// Interpretation rules:
//   - WinRate is meaningful only when Resolved > 0.
//   - AvgPnL is computed over resolved chains only (win + loss + breakeven).
//   - Unresolved chains are counted but excluded from P&L statistics.
//   - When all chains are unresolved, WinRate=0, AvgPnL=0, TotalPnL=0.
type CohortSummary struct {
	Key string `json:"key"` // grouping key value ("all" when ungrouped)

	// Counts by outcome.
	WinCount        int `json:"win_count"`
	LossCount       int `json:"loss_count"`
	BreakevenCount  int `json:"breakeven_count"`
	UnresolvedCount int `json:"unresolved_count"`

	// Derived counts.
	Evaluated int `json:"evaluated"` // total chains evaluated (all outcomes)
	Resolved  int `json:"resolved"`  // win + loss + breakeven (excludes unresolved)

	// P&L statistics (over resolved chains only).
	TotalPnL  float64 `json:"total_pnl"`  // sum of net_pnl across resolved chains
	AvgPnL    float64 `json:"avg_pnl"`    // TotalPnL / Resolved (0 when Resolved=0)
	TotalFees float64 `json:"total_fees"` // sum of total_fees across all evaluated chains

	// Win rate: wins / resolved (0 when Resolved=0).
	// Expressed as a ratio 0.0--1.0, NOT a percentage.
	WinRate float64 `json:"win_rate"`
}

// ReviewEffectiveness is the effectiveness section added to DecisionReviewBundle.
// Present only when execution reached a terminal state with classifiable data.
//
// S476: Extends S471 DecisionReviewBundle with effectiveness attribution.
type ReviewEffectiveness struct {
	Outcome        string  `json:"outcome"` // win, loss, breakeven, unresolved
	RealizedPnL    float64 `json:"realized_pnl"`
	GrossPnL       float64 `json:"gross_pnl"`
	NetPnL         float64 `json:"net_pnl"`
	TotalFees      float64 `json:"total_fees"`
	EntryCostBasis float64 `json:"entry_cost_basis"`
	FillCount      int     `json:"fill_count"`
	Simulated      bool    `json:"simulated"`
	Explanation    string  `json:"explanation"`
}
