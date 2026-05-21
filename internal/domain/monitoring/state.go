package monitoring

import (
	"time"

	"internal/domain/execution"
)

// OperationalState is the consolidated monitoring snapshot that answers
// "what is the system doing right now?" in a single query. It aggregates
// session, gate, segment health, and surface availability — information
// that an operator would otherwise need 3+ separate HTTP calls to collect.
//
// S486: Introduced to close the monitoring gap where operational state
// was dispersed across /session/list, /execution/control, and /statusz.
type OperationalState struct {
	// Timestamp when this snapshot was computed.
	ObservedAt time.Time `json:"observed_at"`

	// Session is the most recent session (open or terminal).
	// Nil when no sessions have been created.
	Session *SessionSummary `json:"session"`

	// Gate is the current execution control gate state.
	// Nil when the execution control gateway is unavailable.
	Gate *GateSummary `json:"gate"`

	// Surfaces reports which HTTP endpoint families are wired and available.
	Surfaces SurfaceAvailability `json:"surfaces"`
}

// SessionSummary is a lightweight operational view of a session.
// It carries only the fields needed for monitoring — not the full
// audit detail available from /session/:id.
type SessionSummary struct {
	SessionID string                   `json:"session_id"`
	Operator  string                   `json:"operator,omitempty"`
	Status    execution.SessionStatus  `json:"status"`
	StartedAt time.Time                `json:"started_at"`
	ClosedAt  *time.Time               `json:"closed_at,omitempty"`
	Duration  string                   `json:"duration,omitempty"`
	Config    SessionConfigSummary     `json:"config"`
	Counters  []SegmentCountersSummary `json:"counters,omitempty"`
}

// SessionConfigSummary captures the key config dimensions for monitoring display.
type SessionConfigSummary struct {
	VenueType string   `json:"venue_type"`
	DryRun    bool     `json:"dry_run"`
	Segments  []string `json:"segments,omitempty"`
}

// SegmentCountersSummary is a monitoring-oriented view of per-segment counters.
type SegmentCountersSummary struct {
	Segment   string `json:"segment"`
	Processed int64  `json:"processed"`
	Filled    int64  `json:"filled"`
	Rejected  int64  `json:"rejected"`
	Errors    int64  `json:"errors"`
}

// GateSummary is a monitoring-oriented view of the execution control gate.
type GateSummary struct {
	Status    string     `json:"status"`
	Reason    string     `json:"reason,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// SurfaceAvailability reports which endpoint families are wired in the gateway.
// Each field is true when the underlying gateway or data source was available
// at composition time. This lets operators see at a glance which surfaces are
// degraded without probing each one individually.
type SurfaceAvailability struct {
	Evidence   bool `json:"evidence"`
	Signal     bool `json:"signal"`
	Decision   bool `json:"decision"`
	Strategy   bool `json:"strategy"`
	Risk       bool `json:"risk"`
	Execution  bool `json:"execution"`
	Session    bool `json:"session"`
	Analytical bool `json:"analytical"`
	Activation bool `json:"activation"`
}

// DegradedFamilies returns the names of endpoint families that are unavailable.
func (s SurfaceAvailability) DegradedFamilies() []string {
	var degraded []string
	check := func(name string, avail bool) {
		if !avail {
			degraded = append(degraded, name)
		}
	}
	check("evidence", s.Evidence)
	check("signal", s.Signal)
	check("decision", s.Decision)
	check("strategy", s.Strategy)
	check("risk", s.Risk)
	check("execution", s.Execution)
	check("session", s.Session)
	check("analytical", s.Analytical)
	check("activation", s.Activation)
	return degraded
}

// NewSessionSummary creates a SessionSummary from a full Session entity.
func NewSessionSummary(s execution.Session) SessionSummary {
	summary := SessionSummary{
		SessionID: s.SessionID,
		Operator:  s.Operator,
		Status:    s.Status,
		StartedAt: s.StartedAt,
		ClosedAt:  s.ClosedAt,
		Config: SessionConfigSummary{
			VenueType: s.Config.VenueType,
			DryRun:    s.Config.DryRun,
			Segments:  s.Config.Segments,
		},
	}

	if s.ClosedAt != nil {
		d := s.ClosedAt.Sub(s.StartedAt).Truncate(time.Second)
		summary.Duration = d.String()
	}

	for _, c := range s.SegmentCounters {
		summary.Counters = append(summary.Counters, SegmentCountersSummary{
			Segment:   c.Segment,
			Processed: c.Processed,
			Filled:    c.Filled,
			Rejected:  c.Rejected,
			Errors:    c.Errors,
		})
	}

	return summary
}
