package pairing

// ContinuitySummary aggregates continuity classification counts across a set
// of cross-session round-trips.
//
// S495: This is the read-model projection of the S494 continuity states.
// It answers "how many legs resolved, how many are open, how many are
// genuinely vs artificially unresolved?" at the summary level.
type ContinuitySummary struct {
	// ResolvedCount is the number of round-trips with ContinuityResolved.
	ResolvedCount int `json:"resolved_count"`

	// OpenCount is the number of round-trips with ContinuityOpen.
	OpenCount int `json:"open_count"`

	// GenuineUnresolvedCount is the number of round-trips with ContinuityGenuineUnresolved.
	GenuineUnresolvedCount int `json:"genuine_unresolved_count"`

	// ArtificialUnresolvedCount is the number of round-trips with ContinuityArtificialUnresolved.
	// After cross-session matching, this should decrease compared to intra-session-only.
	ArtificialUnresolvedCount int `json:"artificial_unresolved_count"`

	// CrossSessionPairedCount is the number of resolved round-trips where
	// entry and exit originate from different sessions.
	CrossSessionPairedCount int `json:"cross_session_paired_count"`

	// IntraSessionPairedCount is the number of resolved round-trips where
	// entry and exit originate from the same session.
	IntraSessionPairedCount int `json:"intra_session_paired_count"`

	// Total is the total number of round-trips summarized.
	Total int `json:"total"`

	// ResolutionRate is ResolvedCount / Total (0 when Total=0).
	ResolutionRate float64 `json:"resolution_rate"`

	// CrossSessionResolutionRate is CrossSessionPairedCount / (CrossSessionPairedCount + ArtificialUnresolvedCount).
	// This measures how effective cross-session matching was at resolving boundary artifacts.
	// 0 when denominator is 0.
	CrossSessionResolutionRate float64 `json:"cross_session_resolution_rate"`
}

// SummarizeContinuity computes a ContinuitySummary from a set of cross-session round-trips.
// This is a pure function — no side effects or I/O.
func SummarizeContinuity(rts []CrossSessionRoundTrip) ContinuitySummary {
	var s ContinuitySummary

	for _, rt := range rts {
		s.Total++

		switch rt.Continuity {
		case ContinuityResolved:
			s.ResolvedCount++
			if rt.CrossSession {
				s.CrossSessionPairedCount++
			} else {
				s.IntraSessionPairedCount++
			}
		case ContinuityOpen:
			s.OpenCount++
		case ContinuityGenuineUnresolved:
			s.GenuineUnresolvedCount++
		case ContinuityArtificialUnresolved:
			s.ArtificialUnresolvedCount++
		}
	}

	if s.Total > 0 {
		s.ResolutionRate = float64(s.ResolvedCount) / float64(s.Total)
	}

	crossDenom := s.CrossSessionPairedCount + s.ArtificialUnresolvedCount
	if crossDenom > 0 {
		s.CrossSessionResolutionRate = float64(s.CrossSessionPairedCount) / float64(crossDenom)
	}

	return s
}
