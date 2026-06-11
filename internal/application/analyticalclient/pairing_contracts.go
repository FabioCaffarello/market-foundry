package analyticalclient

import (
	"internal/domain/instrument"

	"internal/domain/effectiveness"
	"internal/domain/pairing"
)

// PairingQuery is the request contract for the round-trip pairing read model.
//
// S481: Answers Q-RT1 (can identify and pair entry/exit legs), Q-RT2 (does pairing
// increase resolved rate), and Q-RT5 (computable from existing data).
//
// Lookup modes:
//   - Source+Symbol+Timeframe: batch pairing across recent execution chains.
//   - CorrelationID+Symbol: single-chain pairing lookup (returns legs for one chain).
//
// The pairing read model fetches execution chains from the CompositeReader,
// converts fills to legs via IntentToLeg, and applies MatchFIFO to produce
// paired and unmatched round-trips with attribution.
type PairingQuery struct {
	CorrelationID string `json:"correlation_id,omitempty"` // single lookup

	// Batch lookup filters.
	Source     string                         `json:"source,omitempty"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe,omitempty"`
	Since      int64                          `json:"since,omitempty"` // unix seconds, inclusive
	Until      int64                          `json:"until,omitempty"` // unix seconds, inclusive
	Limit      int                            `json:"limit,omitempty"` // default 50, max 200

	// Filters.
	State string `json:"state,omitempty"` // paired, unmatched_entry, unmatched_exit
	Side  string `json:"side,omitempty"`  // buy, sell
}

// PairingReply is the response contract for the round-trip pairing read model.
type PairingReply struct {
	RoundTrips []RoundTripView `json:"round_trips"`
	Summary    PairingSummary  `json:"summary"`
	Source     string          `json:"source"` // always "clickhouse"
	Meta       PairingMeta     `json:"meta"`
}

// RoundTripView is the HTTP-facing representation of a round-trip with
// optional effectiveness attribution for paired round-trips.
type RoundTripView struct {
	pairing.RoundTrip

	// Attribution is present only for paired round-trips where both entry
	// and exit have classifiable fills. Nil for unmatched legs.
	Attribution *effectiveness.Attribution `json:"attribution,omitempty"`
}

// PairingSummary is the aggregated view of a pairing run, mirroring
// pairing.PairingResult with additional effectiveness integration.
type PairingSummary struct {
	TotalEntries     int     `json:"total_entries"`
	TotalExits       int     `json:"total_exits"`
	PairedCount      int     `json:"paired_count"`
	UnmatchedEntries int     `json:"unmatched_entries"`
	UnmatchedExits   int     `json:"unmatched_exits"`
	ResolvedRate     float64 `json:"resolved_rate"`

	// Effectiveness breakdown for paired round-trips.
	WinCount       int     `json:"win_count"`
	LossCount      int     `json:"loss_count"`
	BreakevenCount int     `json:"breakeven_count"`
	TotalPnL       float64 `json:"total_pnl"`
	TotalFees      float64 `json:"total_fees"`
}

// PairingMeta carries diagnostic signals for pairing queries.
type PairingMeta struct {
	TotalMs       int64 `json:"total_ms"`
	ChainsScanned int   `json:"chains_scanned"`
	LegsProduced  int   `json:"legs_produced"`
	RoundTrips    int   `json:"round_trips"`
}
