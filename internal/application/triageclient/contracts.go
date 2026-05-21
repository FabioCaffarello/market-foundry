package triageclient

import "internal/domain/triage"

// SessionTriageQuery is the request contract for session triage.
//
// S487: Enables operators to find sessions that need attention, ranked by
// anomaly severity. Wraps batch audit with triage ranking and filtering.
type SessionTriageQuery struct {
	// StatusFilter narrows to sessions with a specific status (e.g. "closed").
	StatusFilter string `json:"status,omitempty"`

	// CheckFilter narrows to sessions that failed a specific PO check.
	CheckFilter string `json:"check,omitempty"`

	// SeverityFilter narrows to items at or above this severity.
	SeverityFilter string `json:"severity,omitempty"` // critical, warning

	// Limit caps the number of returned items. Default 20, max 50.
	Limit int `json:"limit,omitempty"`
}

// SessionTriageReply is the response for session triage queries.
type SessionTriageReply struct {
	Items   []triage.SessionTriageItem `json:"items"`
	Summary triage.TriageDomainSummary `json:"summary"`
	Meta    TriageMeta                 `json:"meta"`
}

// DecisionTriageQuery is the request contract for decision triage.
//
// S487: Surfaces decisions with consistency violations, incomplete chains,
// or poor effectiveness outcomes ranked by severity for operator review.
type DecisionTriageQuery struct {
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
	Since     int64  `json:"since,omitempty"`
	Until     int64  `json:"until,omitempty"`
	Limit     int    `json:"limit,omitempty"` // default 20, max 100

	// SeverityFilter narrows to items at or above this severity.
	SeverityFilter string `json:"severity,omitempty"` // critical, warning
}

// DecisionTriageReply is the response for decision triage queries.
type DecisionTriageReply struct {
	Items   []triage.DecisionTriageItem `json:"items"`
	Summary triage.TriageDomainSummary  `json:"summary"`
	Source  string                      `json:"source"`
	Meta    TriageMeta                  `json:"meta"`
}

// RoundTripTriageQuery is the request contract for round-trip triage.
//
// S487: Surfaces round-trips with data quality issues ranked by severity.
type RoundTripTriageQuery struct {
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
	Since     int64  `json:"since,omitempty"`
	Until     int64  `json:"until,omitempty"`
	Limit     int    `json:"limit,omitempty"` // default 50, max 200

	// SeverityFilter narrows to items at or above this severity.
	SeverityFilter string `json:"severity,omitempty"` // critical, warning
}

// RoundTripTriageReply is the response for round-trip triage queries.
type RoundTripTriageReply struct {
	Items   []triage.RoundTripTriageItem `json:"items"`
	Summary triage.TriageDomainSummary   `json:"summary"`
	Source  string                       `json:"source"`
	Meta    TriageMeta                   `json:"meta"`
}

// TriageOverviewQuery is the request contract for the cross-domain triage overview.
//
// S487: Single-call "what needs attention?" across sessions, decisions, and round-trips.
type TriageOverviewQuery struct {
	// Session triage parameters.
	SessionStatus string `json:"session_status,omitempty"`

	// Decision/round-trip triage parameters.
	Source    string `json:"source,omitempty"`
	Symbol    string `json:"symbol,omitempty"`
	Timeframe int    `json:"timeframe,omitempty"`
	Since     int64  `json:"since,omitempty"`
	Until     int64  `json:"until,omitempty"`
}

// TriageOverviewReply is the response for the cross-domain triage overview.
type TriageOverviewReply struct {
	Overview triage.TriageOverview `json:"overview"`
	Meta     TriageMeta           `json:"meta"`
}

// TriageMeta carries diagnostic signals for triage queries.
type TriageMeta struct {
	TotalMs    int64 `json:"total_ms"`
	Scanned    int   `json:"scanned"`    // total items scanned before filtering
	Returned   int   `json:"returned"`   // items returned after filtering
}
