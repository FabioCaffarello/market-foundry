package execution

import "time"

// UnifiedOperationalReport is the canonical single-artifact output of the
// automated post-session verification and operational assessment pipeline.
//
// S491: This type composes verification, audit, monitoring, and triage data
// into one archivable JSON document per session. It answers "what is the
// operational status of this session?" without requiring the operator to
// query multiple endpoints and mentally join the results.
//
// The report is produced both on-demand (GET /session/:id/report) and
// automatically by the event-driven verification trigger (S490).
type UnifiedOperationalReport struct {
	// Identity and provenance.
	SessionID   string    `json:"session_id"`
	GeneratedAt time.Time `json:"generated_at"`
	GeneratedBy string    `json:"generated_by"` // "auto-trigger" or "http-request"
	DurationMs  int64     `json:"duration_ms"`

	// Section 1: Verification (PO checks).
	Verification *ReportVerificationSection `json:"verification,omitempty"`

	// Section 2: Session audit (lifecycle, fees, consistency).
	Audit *ReportAuditSection `json:"audit,omitempty"`

	// Section 3: Operational state snapshot.
	OperationalState *ReportOperationalStateSection `json:"operational_state,omitempty"`

	// Section 4: Triage overview (cross-domain anomaly summary).
	Triage *ReportTriageSection `json:"triage,omitempty"`

	// Overall verdict derived from all sections.
	Verdict ReportVerdict `json:"verdict"`

	// Sections that could not be populated and why.
	Gaps []ReportGap `json:"gaps,omitempty"`
}

// ReportVerdict is the top-level assessment of the unified report.
type ReportVerdict string

const (
	ReportVerdictPass     ReportVerdict = "pass"
	ReportVerdictWarn     ReportVerdict = "warn"
	ReportVerdictFail     ReportVerdict = "fail"
	ReportVerdictDegraded ReportVerdict = "degraded" // some sections unavailable
)

// ReportGap records a section that could not be populated.
type ReportGap struct {
	Section string `json:"section"`
	Reason  string `json:"reason"`
}

// ReportVerificationSection wraps the PO verification report with a
// quick-scan summary for the unified artifact.
type ReportVerificationSection struct {
	AllPassed  bool      `json:"all_passed"`
	Summary    POSummary `json:"summary"`
	DurationMs int64     `json:"duration_ms"`
	// Full check details — included for archival completeness.
	Checks []POCheckResult `json:"checks"`
}

// ReportAuditSection wraps the session audit bundle highlights.
type ReportAuditSection struct {
	SessionStatus string             `json:"session_status"`
	Operator      string             `json:"operator,omitempty"`
	OrderActivity AuditOrderActivity `json:"order_activity"`
	FeeSummary    AuditFeeSummary    `json:"fee_summary"`
	Consistency   AuditConsistency   `json:"consistency"`
	CheckIndex    AuditCheckIndex    `json:"check_index"`
}

// ReportOperationalStateSection captures the gate and surface availability
// at report generation time.
type ReportOperationalStateSection struct {
	GateStatus        string   `json:"gate_status"`
	GateReason        string   `json:"gate_reason,omitempty"`
	AvailableSurfaces []string `json:"available_surfaces"`
}

// ReportTriageSection captures the cross-domain triage summary.
type ReportTriageSection struct {
	TotalAnomalies    int      `json:"total_anomalies"`
	SessionCritical   int      `json:"session_critical"`
	SessionWarning    int      `json:"session_warning"`
	DecisionCritical  int      `json:"decision_critical"`
	DecisionWarning   int      `json:"decision_warning"`
	RoundTripCritical int      `json:"round_trip_critical"`
	RoundTripWarning  int      `json:"round_trip_warning"`
	TopFindings       []string `json:"top_findings,omitempty"`
}

// ComputeVerdict derives the overall verdict from all populated sections.
func (r *UnifiedOperationalReport) ComputeVerdict() {
	if len(r.Gaps) > 0 && r.Verification == nil {
		r.Verdict = ReportVerdictDegraded
		return
	}

	hasFail := false
	hasWarn := false

	// Verification failures.
	if r.Verification != nil && !r.Verification.AllPassed {
		if r.Verification.Summary.Failed > 0 {
			hasFail = true
		} else {
			hasWarn = true
		}
	}

	// Audit inconsistency.
	if r.Audit != nil {
		switch r.Audit.Consistency.OverallVerdict {
		case "inconsistent":
			hasFail = true
		case "degraded":
			hasWarn = true
		}
	}

	// Triage anomalies.
	if r.Triage != nil {
		if r.Triage.SessionCritical > 0 || r.Triage.DecisionCritical > 0 || r.Triage.RoundTripCritical > 0 {
			hasFail = true
		}
		if r.Triage.SessionWarning > 0 || r.Triage.DecisionWarning > 0 || r.Triage.RoundTripWarning > 0 {
			hasWarn = true
		}
	}

	switch {
	case hasFail:
		r.Verdict = ReportVerdictFail
	case hasWarn:
		r.Verdict = ReportVerdictWarn
	case len(r.Gaps) > 0:
		r.Verdict = ReportVerdictDegraded
	default:
		r.Verdict = ReportVerdictPass
	}
}
