// Package pairing — continuity defines the canonical cross-session continuity model.
//
// S494: Domain types for cross-session leg discovery, multi-session windows,
// carry-forward eligibility, and continuity state classification.
//
// This is a read-side model. It does not modify the write path, carry state
// at runtime, or introduce position tracking. Sessions remain isolated at
// runtime; continuity is computed retrospectively from existing fill data.
//
// Guard rails:
//   - No write-path changes to execution or session lifecycle.
//   - No position engine or portfolio model.
//   - No runtime state carry-forward between sessions.
//   - Additive only; zero changes to existing pairing or effectiveness types.
package pairing

import (
	"time"

	"internal/domain/execution"
)

// ---------------------------------------------------------------------------
// Continuity state
// ---------------------------------------------------------------------------

// ContinuityState classifies whether a leg's lifecycle is resolved when
// considering data from multiple sessions.
type ContinuityState string

const (
	// ContinuityResolved indicates the leg was paired with a counterpart
	// (either intra-session or cross-session). The round-trip is closed.
	ContinuityResolved ContinuityState = "resolved"

	// ContinuityOpen indicates the leg has no counterpart across all
	// sessions in the lookback window. It may resolve in a future session.
	ContinuityOpen ContinuityState = "open"

	// ContinuityGenuineUnresolved indicates the leg cannot resolve because
	// of a structural condition (rejected, cancelled, non-terminal status)
	// rather than a missing counterpart. It is permanently unresolved.
	ContinuityGenuineUnresolved ContinuityState = "genuine_unresolved"

	// ContinuityArtificialUnresolved indicates the leg was classified as
	// unresolved solely because it sits at a session boundary. A counterpart
	// may exist in an adjacent session. This is the primary target of
	// cross-session resolution.
	ContinuityArtificialUnresolved ContinuityState = "artificial_unresolved"
)

// ValidContinuityState reports whether cs is a recognized continuity state.
func ValidContinuityState(cs ContinuityState) bool {
	switch cs {
	case ContinuityResolved, ContinuityOpen,
		ContinuityGenuineUnresolved, ContinuityArtificialUnresolved:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Session leg — a Leg annotated with session provenance
// ---------------------------------------------------------------------------

// SessionLeg wraps a Leg with the session context in which it was produced.
// This enables cross-session pairing to preserve provenance: when an entry
// from session N pairs with an exit from session M, both session IDs are
// available for audit and reconciliation.
type SessionLeg struct {
	Leg

	// SessionID is the identifier of the originating session (e.g. "session_20260326_120030").
	SessionID string `json:"session_id"`

	// SessionStartedAt is the start timestamp of the originating session.
	SessionStartedAt time.Time `json:"session_started_at"`

	// SessionClosedAt is the close timestamp of the originating session.
	// Nil (zero) if the session is still open.
	SessionClosedAt *time.Time `json:"session_closed_at,omitempty"`
}

// IsCrossSession reports whether two session legs originate from different sessions.
func IsCrossSession(a, b SessionLeg) bool {
	return a.SessionID != "" && b.SessionID != "" && a.SessionID != b.SessionID
}

// ---------------------------------------------------------------------------
// Cross-session window
// ---------------------------------------------------------------------------

// CrossSessionWindow is a query filter for cross-session reconciliation.
// VenueSymbol carries the venue-native lowercase form (e.g., "btcusdt")
// as supplied by the caller (HTTP query, CLI flag); it is metadata only
// and is NOT consulted by the matching algorithm — only validated for
// non-emptiness by Validate().
//
// Promoted to instrument.CanonicalInstrument would require source-string
// reconstruction at the boundary (the same pattern that caused the
// H-6.b' regression, commit 37f8ddd). Kept as string per architectural
// decision documented in PRD-0004 H-6.b” closure section. The
// declaration in policies/domain_types.toml uses the "string_filter"
// migration_state (introduced in commit 1 of H-6.b”) to record that
// the field is venue-native string by design rather than transient.
//
// Renamed from `Symbol string` in H-6.b” to clarify the field's role
// (transport metadata, not domain projection).
type CrossSessionWindow struct {
	// VenueSymbol restricts discovery to legs whose venue-native symbol
	// matches this string (e.g., "btcusdt"). Compared lexicographically
	// against Leg.VenueSymbol() — never reconstructed into a canonical
	// instrument.
	VenueSymbol string `json:"venue_symbol"`

	// Source restricts discovery to legs from this venue/segment (e.g. "binance_spot").
	Source string `json:"source"`

	// Timeframe restricts discovery to legs from this candle interval.
	Timeframe int `json:"timeframe"`

	// Since is the earliest timestamp (inclusive) for session start.
	// Legs from sessions that started before this are excluded.
	Since time.Time `json:"since"`

	// Until is the latest timestamp (inclusive) for session close.
	// Legs from sessions that closed after this are excluded.
	// Zero means "up to now" (includes the current open session if any).
	Until time.Time `json:"until,omitempty"`

	// MaxSessions limits the number of sessions to include (most recent first).
	// Zero means no limit (bounded by Since/Until instead).
	MaxSessions int `json:"max_sessions,omitempty"`
}

// DefaultLookbackDays is the default lookback window for cross-session
// discovery when no explicit Since is provided. 30 days balances relevance
// against query cost.
const DefaultLookbackDays = 30

// DefaultMaxSessions is the maximum number of sessions to include in a
// cross-session discovery query when no explicit MaxSessions is provided.
const DefaultMaxSessions = 30

// Validate reports whether the window has the minimum required fields.
func (w CrossSessionWindow) Validate() bool {
	return w.VenueSymbol != "" && w.Source != "" && w.Timeframe > 0 && !w.Since.IsZero()
}

// ---------------------------------------------------------------------------
// Cross-session leg set
// ---------------------------------------------------------------------------

// CrossSessionLegSet is an ordered collection of SessionLegs drawn from
// multiple sessions. Legs are sorted by timestamp (ascending) to preserve
// the FIFO temporal invariant (M4) across session boundaries.
//
// The leg set is the input to MatchFIFO for cross-session pairing.
type CrossSessionLegSet struct {
	// Window is the discovery scope that produced this leg set.
	Window CrossSessionWindow `json:"window"`

	// Sessions lists the session IDs that contributed legs, in chronological order.
	Sessions []string `json:"sessions"`

	// Legs is the ordered collection of session-annotated legs.
	Legs []SessionLeg `json:"legs"`
}

// LegCount returns the number of legs in the set.
func (ls CrossSessionLegSet) LegCount() int {
	return len(ls.Legs)
}

// SessionCount returns the number of distinct sessions that contributed legs.
func (ls CrossSessionLegSet) SessionCount() int {
	return len(ls.Sessions)
}

// ExtractLegs returns plain Legs (without session metadata) suitable for
// passing to MatchFIFO. The temporal ordering is preserved.
func (ls CrossSessionLegSet) ExtractLegs() []Leg {
	if len(ls.Legs) == 0 {
		return nil
	}
	legs := make([]Leg, len(ls.Legs))
	for i := range ls.Legs {
		legs[i] = ls.Legs[i].Leg
	}
	return legs
}

// SessionLegIndex returns a map from CorrelationID to SessionLeg for
// efficient provenance lookup after matching.
func (ls CrossSessionLegSet) SessionLegIndex() map[string]SessionLeg {
	idx := make(map[string]SessionLeg, len(ls.Legs))
	for _, sl := range ls.Legs {
		idx[sl.CorrelationID] = sl
	}
	return idx
}

// ---------------------------------------------------------------------------
// Carry-forward eligibility
// ---------------------------------------------------------------------------

// CarryForwardEligibility classifies whether a leg is eligible for
// cross-session carry-forward (i.e., whether it should be included in a
// cross-session discovery query as an unmatched candidate).
type CarryForwardEligibility string

const (
	// CarryEligible means the leg is a filled entry without a matching exit
	// within its session. It is the primary candidate for cross-session pairing.
	CarryEligible CarryForwardEligibility = "eligible"

	// CarryIneligibleRejected means the leg's execution was rejected.
	// Rejected orders never produce fills and cannot participate in pairing.
	CarryIneligibleRejected CarryForwardEligibility = "ineligible_rejected"

	// CarryIneligibleCancelled means the leg's execution was cancelled
	// before any fill. No fill data exists for pairing.
	CarryIneligibleCancelled CarryForwardEligibility = "ineligible_cancelled"

	// CarryIneligibleNonTerminal means the leg's execution is still in a
	// non-terminal state (submitted, sent, accepted). It cannot carry
	// forward because its lifecycle is incomplete.
	CarryIneligibleNonTerminal CarryForwardEligibility = "ineligible_non_terminal"

	// CarryIneligibleNoFills means the execution reached a terminal state
	// but has no fill records. Without fills there is no leg to carry.
	CarryIneligibleNoFills CarryForwardEligibility = "ineligible_no_fills"

	// CarryIneligibleAlreadyPaired means the leg was already paired within
	// its originating session. It does not need cross-session resolution.
	CarryIneligibleAlreadyPaired CarryForwardEligibility = "ineligible_already_paired"
)

// ValidCarryForwardEligibility reports whether e is a recognized eligibility value.
func ValidCarryForwardEligibility(e CarryForwardEligibility) bool {
	switch e {
	case CarryEligible, CarryIneligibleRejected, CarryIneligibleCancelled,
		CarryIneligibleNonTerminal, CarryIneligibleNoFills, CarryIneligibleAlreadyPaired:
		return true
	}
	return false
}

// ClassifyCarryForward determines whether an ExecutionIntent is eligible for
// cross-session carry-forward. This is a pure function.
//
// Rules:
//   - R-CF1: Rejected intents are ineligible (no fills).
//   - R-CF2: Cancelled-before-fill intents are ineligible (no fills).
//   - R-CF3: Non-terminal intents are ineligible (lifecycle incomplete).
//   - R-CF4: Terminal intents with zero fills are ineligible (no leg data).
//   - R-CF5: Filled intents with fills are eligible for carry-forward.
//
// The caller determines whether the leg is already paired within its session
// (CarryIneligibleAlreadyPaired) by running intra-session MatchFIFO first.
func ClassifyCarryForward(intent execution.ExecutionIntent) CarryForwardEligibility {
	// R-CF1: Rejected.
	if intent.Status == execution.StatusRejected {
		return CarryIneligibleRejected
	}
	// R-CF3: Non-terminal.
	if !intent.Status.IsTerminal() {
		return CarryIneligibleNonTerminal
	}
	// R-CF2: Cancelled before fill.
	if intent.Status == execution.StatusCancelled && len(intent.Fills) == 0 {
		return CarryIneligibleCancelled
	}
	// R-CF4: Terminal but no fills.
	if len(intent.Fills) == 0 {
		return CarryIneligibleNoFills
	}
	// R-CF5: Eligible.
	return CarryEligible
}

// ---------------------------------------------------------------------------
// Continuity classification for round-trips
// ---------------------------------------------------------------------------

// ClassifyContinuity determines the ContinuityState for a RoundTrip,
// using the session leg index to detect cross-session provenance.
//
// Rules:
//   - C-1: Paired round-trip → ContinuityResolved.
//   - C-2: Unmatched entry with ReasonSessionBoundary → ContinuityArtificialUnresolved.
//   - C-3: Unmatched entry with ReasonRejectedLeg or ReasonCancelledLeg → ContinuityGenuineUnresolved.
//   - C-4: Unmatched entry with ReasonNoExitFound → ContinuityOpen.
//   - C-5: Unmatched exit (orphan) → ContinuityGenuineUnresolved.
//   - C-6: Unmatched entry with ReasonQuantityMismatchResidue → ContinuityOpen (partial may resolve).
func ClassifyContinuity(rt RoundTrip) ContinuityState {
	switch rt.State {
	case StatePaired:
		// C-1: Both legs present.
		return ContinuityResolved

	case StateUnmatchedEntry:
		switch rt.UnmatchedReason {
		case ReasonSessionBoundary:
			// C-2: Session boundary — likely resolvable in adjacent session.
			return ContinuityArtificialUnresolved
		case ReasonRejectedLeg:
			// C-3: Structural failure.
			return ContinuityGenuineUnresolved
		case ReasonCancelledLeg:
			// C-3: Structural failure.
			return ContinuityGenuineUnresolved
		case ReasonQuantityMismatchResidue:
			// C-6: Partial remainder — may resolve with future exit.
			return ContinuityOpen
		default:
			// C-4: No exit found — position is open, may resolve later.
			return ContinuityOpen
		}

	case StateUnmatchedExit:
		// C-5: Orphan exit — no entry found. Data gap or error.
		return ContinuityGenuineUnresolved

	default:
		return ContinuityOpen
	}
}

// ---------------------------------------------------------------------------
// Cross-session round-trip — annotated with session provenance
// ---------------------------------------------------------------------------

// CrossSessionRoundTrip extends a RoundTrip with session provenance and
// continuity classification. This is the output type for cross-session
// pairing queries.
type CrossSessionRoundTrip struct {
	RoundTrip

	// EntrySessionID is the session that produced the entry leg. Empty if unmatched_exit.
	EntrySessionID string `json:"entry_session_id,omitempty"`

	// ExitSessionID is the session that produced the exit leg. Empty if unmatched_entry.
	ExitSessionID string `json:"exit_session_id,omitempty"`

	// CrossSession is true when entry and exit originate from different sessions.
	CrossSession bool `json:"cross_session"`

	// Continuity classifies the lifecycle resolution state of this round-trip.
	Continuity ContinuityState `json:"continuity"`
}

// AnnotateRoundTrips takes round-trips produced by MatchFIFO on a
// CrossSessionLegSet and annotates them with session provenance and
// continuity classification.
//
// The sessionIndex maps CorrelationID → SessionLeg for provenance lookup.
func AnnotateRoundTrips(roundTrips []RoundTrip, sessionIndex map[string]SessionLeg) []CrossSessionRoundTrip {
	if len(roundTrips) == 0 {
		return nil
	}

	result := make([]CrossSessionRoundTrip, len(roundTrips))
	for i, rt := range roundTrips {
		csrt := CrossSessionRoundTrip{
			RoundTrip:  rt,
			Continuity: ClassifyContinuity(rt),
		}

		if rt.Entry != nil {
			if sl, ok := sessionIndex[rt.Entry.CorrelationID]; ok {
				csrt.EntrySessionID = sl.SessionID
			}
		}
		if rt.Exit != nil {
			if sl, ok := sessionIndex[rt.Exit.CorrelationID]; ok {
				csrt.ExitSessionID = sl.SessionID
			}
		}

		csrt.CrossSession = csrt.EntrySessionID != "" &&
			csrt.ExitSessionID != "" &&
			csrt.EntrySessionID != csrt.ExitSessionID

		result[i] = csrt
	}

	return result
}
