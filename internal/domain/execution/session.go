package execution

import (
	"fmt"
	"time"

	"internal/shared/problem"
)

// SessionStatus represents the lifecycle status of an operational session.
type SessionStatus string

const (
	SessionOpen   SessionStatus = "open"
	SessionClosed SessionStatus = "closed"
	SessionHalted SessionStatus = "halted"
)

// ValidSessionStatus reports whether s is a recognized session status value.
func ValidSessionStatus(s SessionStatus) bool {
	return s == SessionOpen || s == SessionClosed || s == SessionHalted
}

// IsTerminalSession reports whether s is a terminal session status.
func (s SessionStatus) IsTerminal() bool {
	return s == SessionClosed || s == SessionHalted
}

// SessionConfigSnapshot captures the venue configuration at session start.
// This is a value snapshot — it does not reference the live config.
type SessionConfigSnapshot struct {
	VenueType  string   `json:"venue_type"`
	DryRun     bool     `json:"dry_run"`
	Segments   []string `json:"segments,omitempty"`
	ConfigFile string   `json:"config_file,omitempty"`
}

// SessionActivationSnapshot captures the activation surface at session start.
type SessionActivationSnapshot struct {
	Adapter     AdapterState    `json:"adapter"`
	Credentials CredentialState `json:"credentials"`
	GateStatus  GateStatus      `json:"gate_status"`
	Effective   EffectiveMode   `json:"effective"`
}

// SessionSegmentCounters captures per-segment operational counters at session close.
type SessionSegmentCounters struct {
	Segment   string `json:"segment"`
	Processed int64  `json:"processed"`
	Filled    int64  `json:"filled"`
	Rejected  int64  `json:"rejected"`
	Errors    int64  `json:"errors"`

	// InFlight is the count of execution intents in non-terminal state
	// (submitted, sent, accepted, partially_filled) at session close.
	// S500: Surfaces boundary edge case — these orders may still be filled
	// by the venue after session ends but are not carried forward.
	InFlight int64 `json:"in_flight,omitempty"`
}

// Session represents a canonical operational session record.
//
// A session is the unit of operational accountability: it captures who ran
// what configuration, from when to when, with what outcome. Sessions are
// opened when the execute binary starts processing and closed on graceful
// shutdown or halt.
//
// Ownership: execute binary creates sessions; store binary persists and
// serves queries; gateway binary exposes HTTP endpoints.
type Session struct {
	SessionID  string        `json:"session_id"`
	Operator   string        `json:"operator,omitempty"`
	Status     SessionStatus `json:"status"`
	HaltReason string        `json:"halt_reason,omitempty"`

	StartedAt time.Time  `json:"started_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`

	Config     SessionConfigSnapshot     `json:"config"`
	Activation SessionActivationSnapshot `json:"activation"`

	SegmentCounters []SessionSegmentCounters `json:"segment_counters,omitempty"`

	Artifacts map[string]string `json:"artifacts,omitempty"`
}

// NewSessionID generates a canonical session identifier from a timestamp.
// Format: session_{YYYYMMDD}_{HHMMSS} (UTC).
func NewSessionID(t time.Time) string {
	return fmt.Sprintf("session_%s_%s", t.UTC().Format("20060102"), t.UTC().Format("150405"))
}

// Validate checks that a Session has all required fields populated with valid values.
func (s Session) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if s.SessionID == "" {
		issues = append(issues, problem.ValidationIssue{Field: "session_id", Message: "must not be empty"})
	}
	if !ValidSessionStatus(s.Status) {
		issues = append(issues, problem.ValidationIssue{Field: "status", Message: "must be one of open, closed, halted"})
	}
	if s.StartedAt.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "started_at", Message: "must not be zero"})
	}
	if s.Status.IsTerminal() && s.ClosedAt == nil {
		issues = append(issues, problem.ValidationIssue{Field: "closed_at", Message: "must be set when session is terminal"})
	}
	if s.Status == SessionHalted && s.HaltReason == "" {
		issues = append(issues, problem.ValidationIssue{Field: "halt_reason", Message: "must be set when session is halted"})
	}
	// S500: Temporal ordering invariant — ClosedAt must not precede StartedAt.
	if s.ClosedAt != nil && !s.StartedAt.IsZero() && s.ClosedAt.Before(s.StartedAt) {
		issues = append(issues, problem.ValidationIssue{Field: "closed_at", Message: "must not precede started_at"})
	}
	if s.Config.VenueType == "" {
		issues = append(issues, problem.ValidationIssue{Field: "config.venue_type", Message: "must not be empty"})
	}

	if len(issues) == 0 {
		return nil
	}
	return problem.Validation(problem.InvalidArgument, "session is invalid", issues...)
}

// Close transitions a session to closed status with counters and timestamp.
//
// S500: Returns a problem if the session is already terminal (idempotency guard).
// This prevents silent double-close which could overwrite counters and timestamps.
func (s *Session) Close(counters []SessionSegmentCounters) *problem.Problem {
	if s.Status.IsTerminal() {
		return problem.New(problem.Conflict,
			fmt.Sprintf("cannot close session %s: already in terminal state %q", s.SessionID, s.Status))
	}
	now := time.Now().UTC()
	s.Status = SessionClosed
	s.ClosedAt = &now
	s.SegmentCounters = counters
	return nil
}

// Halt transitions a session to halted status with a reason.
//
// S500: Returns a problem if the session is already terminal (idempotency guard).
func (s *Session) Halt(reason string, counters []SessionSegmentCounters) *problem.Problem {
	if s.Status.IsTerminal() {
		return problem.New(problem.Conflict,
			fmt.Sprintf("cannot halt session %s: already in terminal state %q", s.SessionID, s.Status))
	}
	now := time.Now().UTC()
	s.Status = SessionHalted
	s.HaltReason = reason
	s.ClosedAt = &now
	s.SegmentCounters = counters
	return nil
}

// HasInFlightOrders reports whether any segment has non-terminal orders at close.
// S500: Enables downstream consumers to detect boundary edge cases where
// orders may still be filled by the venue after session ends.
func (s Session) HasInFlightOrders() bool {
	for _, c := range s.SegmentCounters {
		if c.InFlight > 0 {
			return true
		}
	}
	return false
}

// TotalInFlight returns the aggregate count of in-flight orders across all segments.
func (s Session) TotalInFlight() int64 {
	var total int64
	for _, c := range s.SegmentCounters {
		total += c.InFlight
	}
	return total
}

// Duration returns the session duration. Returns zero if session is still open.
func (s Session) Duration() time.Duration {
	if s.ClosedAt == nil {
		return 0
	}
	return s.ClosedAt.Sub(s.StartedAt)
}
