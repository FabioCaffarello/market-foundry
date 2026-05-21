package triage

import "sort"

// TriageSeverity ranks how urgently an item needs operator attention.
type TriageSeverity string

const (
	SeverityCritical TriageSeverity = "critical" // action required now
	SeverityWarning  TriageSeverity = "warning"  // should be reviewed soon
	SeverityInfo     TriageSeverity = "info"     // notable but not blocking
)

// severityRank returns a numeric rank for sorting (lower = more urgent).
func severityRank(s TriageSeverity) int {
	switch s {
	case SeverityCritical:
		return 0
	case SeverityWarning:
		return 1
	case SeverityInfo:
		return 2
	default:
		return 3
	}
}

// Finding is a single triage observation associated with an item.
type Finding struct {
	Domain  string         `json:"domain"`  // session, decision, roundtrip
	Signal  string         `json:"signal"`  // e.g. "check_failed", "consistency_violation", "flagged_roundtrip"
	Detail  string         `json:"detail"`  // human-readable explanation
	Severity TriageSeverity `json:"severity"`
}

// SessionTriageItem is a session ranked by how much attention it needs.
type SessionTriageItem struct {
	SessionID  string         `json:"session_id"`
	Status     string         `json:"status"`
	Verdict    string         `json:"verdict"` // consistent, degraded, inconsistent, errored
	Severity   TriageSeverity `json:"severity"`
	FailedChecks []string     `json:"failed_checks,omitempty"`
	Warnings     []string     `json:"warnings,omitempty"`
	Findings     []Finding    `json:"findings"`
	AnomalyCount int          `json:"anomaly_count"`
}

// DecisionTriageItem is a decision chain ranked by consistency/effectiveness issues.
type DecisionTriageItem struct {
	CorrelationID string         `json:"correlation_id"`
	Symbol        string         `json:"symbol"`
	DecisionType  string         `json:"decision_type"`
	Outcome       string         `json:"outcome"`
	Severity      TriageSeverity `json:"severity"`
	Violations    int            `json:"violations"`
	Incomplete    bool           `json:"incomplete"`
	Effectiveness string         `json:"effectiveness,omitempty"` // win/loss/breakeven/unresolved
	Findings      []Finding      `json:"findings"`
}

// RoundTripTriageItem is a round-trip ranked by data quality issues.
type RoundTripTriageItem struct {
	CorrelationID string         `json:"correlation_id,omitempty"`
	Symbol        string         `json:"symbol"`
	State         string         `json:"state"` // paired, unmatched_entry, unmatched_exit
	Severity      TriageSeverity `json:"severity"`
	Flags         []string       `json:"flags"`
	FlagCount     int            `json:"flag_count"`
	PnLReliable   bool           `json:"pnl_reliable"`
	FeeReliable   bool           `json:"fee_reliable"`
	Outcome       string         `json:"outcome,omitempty"`
}

// TriageOverview is a cross-domain triage summary answering "what needs attention?"
type TriageOverview struct {
	SessionSummary    TriageDomainSummary `json:"sessions"`
	DecisionSummary   TriageDomainSummary `json:"decisions"`
	RoundTripSummary  TriageDomainSummary `json:"roundtrips"`
	TopFindings       []Finding           `json:"top_findings"`
	TotalAnomalies    int                 `json:"total_anomalies"`
}

// TriageDomainSummary captures per-domain triage counts.
type TriageDomainSummary struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
	Clean    int `json:"clean"`
}

// SortBySeverity sorts any slice by severity (critical first), breaking ties
// with a secondary key function.
func SortSessionItems(items []SessionTriageItem) {
	sort.Slice(items, func(i, j int) bool {
		ri, rj := severityRank(items[i].Severity), severityRank(items[j].Severity)
		if ri != rj {
			return ri < rj
		}
		return items[i].AnomalyCount > items[j].AnomalyCount
	})
}

// SortDecisionItems sorts decision triage items by severity then violation count.
func SortDecisionItems(items []DecisionTriageItem) {
	sort.Slice(items, func(i, j int) bool {
		ri, rj := severityRank(items[i].Severity), severityRank(items[j].Severity)
		if ri != rj {
			return ri < rj
		}
		return items[i].Violations > items[j].Violations
	})
}

// SortRoundTripItems sorts round-trip triage items by severity then flag count.
func SortRoundTripItems(items []RoundTripTriageItem) {
	sort.Slice(items, func(i, j int) bool {
		ri, rj := severityRank(items[i].Severity), severityRank(items[j].Severity)
		if ri != rj {
			return ri < rj
		}
		return items[i].FlagCount > items[j].FlagCount
	})
}

// ClassifySessionSeverity determines triage severity from session audit signals.
func ClassifySessionSeverity(verdict string, failedCount, warningCount int) TriageSeverity {
	if verdict == "inconsistent" || failedCount > 0 {
		return SeverityCritical
	}
	if verdict == "degraded" || warningCount > 0 {
		return SeverityWarning
	}
	return SeverityInfo
}

// ClassifyDecisionSeverity determines triage severity from decision review signals.
func ClassifyDecisionSeverity(violations int, incomplete bool) TriageSeverity {
	if violations > 0 {
		return SeverityCritical
	}
	if incomplete {
		return SeverityWarning
	}
	return SeverityInfo
}

// ClassifyRoundTripSeverity determines triage severity from reconciliation signals.
func ClassifyRoundTripSeverity(flagCount int, pnlReliable, feeReliable bool) TriageSeverity {
	if flagCount > 2 || (!pnlReliable && flagCount > 0) {
		return SeverityCritical
	}
	if flagCount > 0 || !feeReliable {
		return SeverityWarning
	}
	return SeverityInfo
}

// ComputeDomainSummary aggregates severity counts.
func ComputeDomainSummary(severities []TriageSeverity, total int) TriageDomainSummary {
	s := TriageDomainSummary{Total: total}
	for _, sev := range severities {
		switch sev {
		case SeverityCritical:
			s.Critical++
		case SeverityWarning:
			s.Warning++
		case SeverityInfo:
			s.Info++
		}
	}
	s.Clean = total - s.Critical - s.Warning - s.Info
	if s.Clean < 0 {
		s.Clean = 0
	}
	return s
}
