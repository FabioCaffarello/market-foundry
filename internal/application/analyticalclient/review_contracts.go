package analyticalclient

import (
	"internal/domain/instrument"

	"internal/domain/effectiveness"
	"internal/domain/pairing"
)

// RoundTripReviewQuery is the request contract for the round-trip review surface.
//
// S482: Provides operators with a unified view of round-trip outcomes,
// reconciliation status, and data-quality flags.
//
// Lookup modes mirror PairingQuery:
//   - CorrelationID+Symbol: single-chain review.
//   - Source+Symbol+Timeframe: batch review across recent round-trips.
//
// Additional filters:
//   - State: paired, unmatched_entry, unmatched_exit.
//   - Side: buy, sell.
//   - Outcome: win, loss, breakeven, unresolved (filters on effectiveness outcome).
//   - Flagged: when true, only return round-trips with reconciliation flags.
type RoundTripReviewQuery struct {
	CorrelationID string `json:"correlation_id,omitempty"` // single lookup

	// Batch lookup filters.
	Source     string                         `json:"source,omitempty"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe,omitempty"`
	Since      int64                          `json:"since,omitempty"`
	Until      int64                          `json:"until,omitempty"`
	Limit      int                            `json:"limit,omitempty"` // default 50, max 200

	// Post-computation filters.
	State   string `json:"state,omitempty"`   // paired, unmatched_entry, unmatched_exit
	Side    string `json:"side,omitempty"`    // buy, sell
	Outcome string `json:"outcome,omitempty"` // win, loss, breakeven, unresolved
	Flagged bool   `json:"flagged,omitempty"` // only return round-trips with flags
}

// RoundTripReviewReply is the response contract for round-trip review queries.
type RoundTripReviewReply struct {
	Reviews []RoundTripReviewItem `json:"reviews"`
	Summary ReviewSummary         `json:"summary"`
	Source  string                `json:"source"` // always "clickhouse"
	Meta    ReviewMeta            `json:"meta"`
}

// RoundTripReviewItem is a single round-trip enriched with reconciliation data.
type RoundTripReviewItem struct {
	pairing.RoundTrip

	// Attribution is present only for paired round-trips with classifiable fills.
	Attribution *effectiveness.Attribution `json:"attribution,omitempty"`

	// Reconciliation flags and data-quality signals for this round-trip.
	Reconciliation pairing.ReconciliationResult `json:"reconciliation"`
}

// ReviewSummary aggregates pairing, effectiveness, and reconciliation statistics.
type ReviewSummary struct {
	// Pairing statistics.
	TotalEntries     int     `json:"total_entries"`
	TotalExits       int     `json:"total_exits"`
	PairedCount      int     `json:"paired_count"`
	UnmatchedEntries int     `json:"unmatched_entries"`
	UnmatchedExits   int     `json:"unmatched_exits"`
	ResolvedRate     float64 `json:"resolved_rate"`

	// Effectiveness breakdown for paired round-trips.
	WinCount        int     `json:"win_count"`
	LossCount       int     `json:"loss_count"`
	BreakevenCount  int     `json:"breakeven_count"`
	UnresolvedCount int     `json:"unresolved_count"`
	TotalPnL        float64 `json:"total_pnl"`
	TotalFees       float64 `json:"total_fees"`

	// Reconciliation aggregates.
	CleanCount   int            `json:"clean_count"`   // round-trips with no flags
	FlaggedCount int            `json:"flagged_count"` // round-trips with at least one flag
	FlagCounts   map[string]int `json:"flag_counts"`   // count per flag type

	// Data reliability.
	FeeReliableCount int `json:"fee_reliable_count"` // round-trips with reliable fee data
	PnLReliableCount int `json:"pnl_reliable_count"` // round-trips with reliable P&L

	// S499: Fee coverage and cost basis totals for fee-to-volume analysis.
	TotalCostBasis   float64 `json:"total_cost_basis"`   // sum of entry+exit cost basis across paired round-trips
	FeeCoverageRatio string  `json:"fee_coverage_ratio"` // "N/M" fills with fee / total fills
}

// ReviewMeta carries diagnostic signals for review queries.
type ReviewMeta struct {
	TotalMs       int64 `json:"total_ms"`
	ChainsScanned int   `json:"chains_scanned"`
	LegsProduced  int   `json:"legs_produced"`
	RoundTrips    int   `json:"round_trips"`
	Reviewed      int   `json:"reviewed"` // after filtering
}
