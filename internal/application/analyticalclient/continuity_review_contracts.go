package analyticalclient

import (
	"internal/domain/instrument"

	"internal/domain/effectiveness"
	"internal/domain/pairing"
)

// ContinuityReviewQuery is the request contract for the continuity review surface.
//
// S496: Provides a unified review of cross-session continuity, combining
// pairing, reconciliation, effectiveness, and boundary carryover analysis
// into a single operator-facing surface.
//
// The query accepts the same partition key as cross-session pairing
// (source, symbol, timeframe, time range) and returns enriched round-trips
// with continuity reconciliation, effectiveness attribution, and aggregate
// summaries that answer: "what was carried, what resolved, and what remains open?"
type ContinuityReviewQuery struct {
	// Window filters — all required.
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`

	// Time range — Since is required. Until defaults to now.
	Since int64 `json:"since"`           // unix seconds, inclusive
	Until int64 `json:"until,omitempty"` // unix seconds, inclusive (0 = now)

	// MaxSessions limits the number of sessions to include (most recent first).
	// Default: 30. Max: 50.
	MaxSessions int `json:"max_sessions,omitempty"`

	// Post-computation filters.
	Continuity string `json:"continuity,omitempty"` // resolved, open, genuine_unresolved, artificial_unresolved
	CrossOnly  bool   `json:"cross_only,omitempty"` // only cross-session round-trips
	Flagged    bool   `json:"flagged,omitempty"`    // only round-trips with reconciliation flags
	Outcome    string `json:"outcome,omitempty"`    // win, loss, breakeven, unresolved
}

// ContinuityReviewReply is the response contract for the continuity review surface.
type ContinuityReviewReply struct {
	Reviews        []ContinuityReviewItem                  `json:"reviews"`
	Continuity     pairing.ContinuitySummary               `json:"continuity"`
	Reconciliation pairing.ContinuityReconciliationSummary `json:"reconciliation"`
	Effectiveness  ContinuityEffectivenessSummary          `json:"effectiveness"`
	Source         string                                  `json:"source"` // always "clickhouse+kv"
	Meta           ContinuityReviewMeta                    `json:"meta"`
}

// ContinuityReviewItem is a single cross-session round-trip enriched with
// reconciliation, effectiveness attribution, and continuity context.
type ContinuityReviewItem struct {
	pairing.CrossSessionRoundTrip

	// Attribution is present only for paired round-trips with classifiable fills.
	Attribution *effectiveness.Attribution `json:"attribution,omitempty"`

	// Reconciliation provides continuity-aware data-quality flags.
	Reconciliation pairing.ContinuityReconciliationResult `json:"reconciliation"`
}

// ContinuityEffectivenessSummary aggregates effectiveness outcomes across
// the continuity review result set, distinguishing cross-session from
// intra-session contributions.
type ContinuityEffectivenessSummary struct {
	// TotalPaired is the number of paired round-trips with attribution.
	TotalPaired int `json:"total_paired"`

	// Outcome counts.
	WinCount        int `json:"win_count"`
	LossCount       int `json:"loss_count"`
	BreakevenCount  int `json:"breakeven_count"`
	UnresolvedCount int `json:"unresolved_count"`

	// P&L aggregates.
	TotalNetPnL   float64 `json:"total_net_pnl"`
	TotalGrossPnL float64 `json:"total_gross_pnl"`
	TotalFees     float64 `json:"total_fees"`

	// Cross-session-specific.
	CrossSessionWins   int     `json:"cross_session_wins"`
	CrossSessionLosses int     `json:"cross_session_losses"`
	CrossSessionPnL    float64 `json:"cross_session_pnl"`

	// Intra-session-specific.
	IntraSessionWins   int     `json:"intra_session_wins"`
	IntraSessionLosses int     `json:"intra_session_losses"`
	IntraSessionPnL    float64 `json:"intra_session_pnl"`
}

// ContinuityReviewMeta carries diagnostic signals for continuity review queries.
type ContinuityReviewMeta struct {
	TotalMs         int64 `json:"total_ms"`
	SessionsFetched int   `json:"sessions_fetched"`
	ChainsScanned   int   `json:"chains_scanned"`
	LegsProduced    int   `json:"legs_produced"`
	RoundTrips      int   `json:"round_trips"`
	Reviewed        int   `json:"reviewed"` // after filtering
}
