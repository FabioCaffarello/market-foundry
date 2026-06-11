package analyticalclient

import (
	"internal/domain/instrument"

	"internal/domain/effectiveness"
	"internal/domain/pairing"
)

// CrossSessionPairingQuery is the request contract for the cross-session
// pairing read model.
//
// S495: Answers the question "what happened to legs that were unresolved within
// a single session when we look across session boundaries?"
//
// The query accepts a CrossSessionWindow (symbol, source, timeframe, time range)
// and returns all round-trips produced by cross-session FIFO matching, annotated
// with session provenance and continuity classification.
//
// Requires both ClickHouse (chains) and SessionGateway (session metadata).
type CrossSessionPairingQuery struct {
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

	// Filters — applied after matching.
	Continuity string `json:"continuity,omitempty"` // resolved, open, genuine_unresolved, artificial_unresolved
	CrossOnly  bool   `json:"cross_only,omitempty"` // when true, only return cross-session round-trips
}

// CrossSessionPairingReply is the response contract for cross-session pairing.
type CrossSessionPairingReply struct {
	RoundTrips []CrossSessionRoundTripView `json:"round_trips"`
	Summary    CrossSessionPairingSummary  `json:"summary"`
	Continuity pairing.ContinuitySummary   `json:"continuity"`
	Source     string                      `json:"source"` // always "clickhouse+kv"
	Meta       CrossSessionPairingMeta     `json:"meta"`
}

// CrossSessionRoundTripView is the HTTP-facing representation of a cross-session
// round-trip with optional effectiveness attribution.
type CrossSessionRoundTripView struct {
	pairing.CrossSessionRoundTrip

	// Attribution is present only for paired round-trips where both entry
	// and exit have classifiable fills. Nil for unmatched legs.
	Attribution *effectiveness.Attribution `json:"attribution,omitempty"`
}

// CrossSessionPairingSummary extends PairingSummary with cross-session specifics.
type CrossSessionPairingSummary struct {
	PairingSummary

	// CrossSessionPairs is the number of round-trips where entry and exit
	// come from different sessions.
	CrossSessionPairs int `json:"cross_session_pairs"`

	// IntraSessionPairs is the number of round-trips where entry and exit
	// come from the same session.
	IntraSessionPairs int `json:"intra_session_pairs"`

	// SessionsScanned is how many sessions contributed data.
	SessionsScanned int `json:"sessions_scanned"`

	// CarryForwardResolutionRate is the fraction of artificial_unresolved legs
	// from intra-session that became resolved via cross-session matching.
	// Range: 0.0--1.0 (NaN-safe: 0 when no artificial_unresolved existed).
	CarryForwardResolutionRate float64 `json:"carry_forward_resolution_rate"`
}

// CrossSessionPairingMeta carries diagnostic signals for cross-session pairing queries.
type CrossSessionPairingMeta struct {
	TotalMs         int64 `json:"total_ms"`
	SessionsFetched int   `json:"sessions_fetched"`
	ChainsScanned   int   `json:"chains_scanned"`
	LegsProduced    int   `json:"legs_produced"`
	LegsCarried     int   `json:"legs_carried"`  // legs eligible for carry-forward
	LegsExcluded    int   `json:"legs_excluded"` // legs excluded by carry-forward rules
	RoundTrips      int   `json:"round_trips"`
}
