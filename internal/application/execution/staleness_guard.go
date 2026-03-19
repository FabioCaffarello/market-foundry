package execution

import "time"

// StalenessGuard checks whether an execution intent's timestamp is too old to act upon.
// maxAge defines the maximum acceptable age. Intents older than maxAge are considered stale.
type StalenessGuard struct {
	maxAge time.Duration
}

// NewStalenessGuard creates a staleness guard with the given maximum age.
func NewStalenessGuard(maxAge time.Duration) *StalenessGuard {
	return &StalenessGuard{maxAge: maxAge}
}

// IsStale returns true if the intent timestamp is older than maxAge relative to now.
func (g *StalenessGuard) IsStale(intentTimestamp time.Time, now time.Time) bool {
	return now.Sub(intentTimestamp) > g.maxAge
}
