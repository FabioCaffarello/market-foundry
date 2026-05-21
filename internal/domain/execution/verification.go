package execution

import "time"

// POCheckID identifies a specific post-operation check.
type POCheckID string

const (
	POCheckGateHalted       POCheckID = "PO-1" // Kill-switch gate is halted
	POCheckBackupCompleted  POCheckID = "PO-2" // Post-session backup succeeded
	POCheckIntentRecords    POCheckID = "PO-3" // ClickHouse intent records exist
	POCheckVenueResponses   POCheckID = "PO-4" // ClickHouse venue response records exist
	POCheckKVState          POCheckID = "PO-5" // NATS KV state captured and validated
	POCheckSystemStatus     POCheckID = "PO-6" // System status healthy at close
	POCheckFeeFields        POCheckID = "PO-7" // Fee/commission fields populated on fills
	POCheckLifecycleConsist POCheckID = "PO-8" // ClickHouse vs KV lifecycle consistency
	POCheckScopeContainment POCheckID = "PO-9" // No out-of-scope executions
)

// AllPOChecks returns the canonical ordered list of PO checks.
func AllPOChecks() []POCheckID {
	return []POCheckID{
		POCheckGateHalted,
		POCheckBackupCompleted,
		POCheckIntentRecords,
		POCheckVenueResponses,
		POCheckKVState,
		POCheckSystemStatus,
		POCheckFeeFields,
		POCheckLifecycleConsist,
		POCheckScopeContainment,
	}
}

// POCheckVerdict represents the outcome of a single PO check.
type POCheckVerdict string

const (
	VerdictPass   POCheckVerdict = "pass"
	VerdictFail   POCheckVerdict = "fail"
	VerdictWarn   POCheckVerdict = "warn"
	VerdictSkip   POCheckVerdict = "skip"   // check not applicable or data unavailable
	VerdictManual POCheckVerdict = "manual" // requires human review
)

// VerificationScope describes the session-derived boundaries for verification.
// S485: Makes verification session-aware rather than hardcoded to 24h/BTCUSDT.
type VerificationScope struct {
	Symbols   []string  `json:"symbols"`
	Since     time.Time `json:"since"`
	Until     time.Time `json:"until"`
	Segments  []string  `json:"segments,omitempty"`
	DryRun    bool      `json:"dry_run"`
	VenueType string    `json:"venue_type,omitempty"`
}

// DefaultVerificationScope returns a fallback scope when session metadata
// is unavailable. Uses 24h window and BTCUSDT as legacy default.
func DefaultVerificationScope() VerificationScope {
	now := time.Now().UTC()
	return VerificationScope{
		Symbols: []string{"BTCUSDT"},
		Since:   now.Add(-24 * time.Hour),
		Until:   now,
	}
}

// POCheckResult captures the outcome of a single PO check execution.
type POCheckResult struct {
	CheckID     POCheckID      `json:"check_id"`
	Name        string         `json:"name"`
	Verdict     POCheckVerdict `json:"verdict"`
	Detail      string         `json:"detail"`
	Evidence    map[string]any `json:"evidence,omitempty"`
	ExecutedAt  time.Time      `json:"executed_at"`
	DurationMs  int64          `json:"duration_ms"`
	Automated   bool           `json:"automated"` // true if fully automated, false if manual review needed
}

// POVerificationReport is the structured output of a full PO verification run.
type POVerificationReport struct {
	SessionID   string              `json:"session_id"`
	Operator    string              `json:"operator,omitempty"`
	ExecutedAt  time.Time           `json:"executed_at"`
	DurationMs  int64               `json:"duration_ms"`
	Scope       *VerificationScope  `json:"scope,omitempty"` // S485: session-derived verification boundaries
	Checks      []POCheckResult     `json:"checks"`
	Summary     POSummary           `json:"summary"`
}

// POSummary aggregates verdict counts for quick assessment.
type POSummary struct {
	Total     int `json:"total"`
	Passed    int `json:"passed"`
	Failed    int `json:"failed"`
	Warnings  int `json:"warnings"`
	Skipped   int `json:"skipped"`
	Manual    int `json:"manual"`
	Automated int `json:"automated"` // count of checks that ran without human intervention
}

// ComputeSummary calculates the summary from the check results.
func (r *POVerificationReport) ComputeSummary() {
	r.Summary = POSummary{Total: len(r.Checks)}
	for _, c := range r.Checks {
		switch c.Verdict {
		case VerdictPass:
			r.Summary.Passed++
		case VerdictFail:
			r.Summary.Failed++
		case VerdictWarn:
			r.Summary.Warnings++
		case VerdictSkip:
			r.Summary.Skipped++
		case VerdictManual:
			r.Summary.Manual++
		}
		if c.Automated {
			r.Summary.Automated++
		}
	}
}

// AllPassed reports whether all non-skipped checks passed without failure.
func (r *POVerificationReport) AllPassed() bool {
	return r.Summary.Failed == 0
}
