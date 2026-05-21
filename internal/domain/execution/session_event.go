package execution

import "time"

// SessionLifecycleEvent is published when a session transitions to a terminal state.
// S490: Event-driven trigger for automated verification.
type SessionLifecycleEvent struct {
	SessionID  string        `json:"session_id"`
	Status     SessionStatus `json:"status"`
	Operator   string        `json:"operator,omitempty"`
	HaltReason string        `json:"halt_reason,omitempty"`
	ClosedAt   time.Time     `json:"closed_at"`

	// Config snapshot for routing and scope derivation.
	VenueType string   `json:"venue_type"`
	DryRun    bool     `json:"dry_run"`
	Segments  []string `json:"segments,omitempty"`
}

// DeduplicationKey returns a unique key for JetStream message deduplication.
// One event per session per terminal status transition.
func (e SessionLifecycleEvent) DeduplicationKey() string {
	return "session-lifecycle:" + e.SessionID + ":" + string(e.Status)
}
