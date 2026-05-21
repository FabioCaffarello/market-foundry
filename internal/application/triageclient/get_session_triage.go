package triageclient

import (
	"context"
	"time"

	"internal/application/executionclient"
	"internal/domain/execution"
	"internal/domain/triage"
	"internal/shared/problem"
)

// SessionBatchAuditor runs batch audit for session triage.
type SessionBatchAuditor interface {
	Execute(context.Context, executionclient.SessionBatchAuditQuery) (executionclient.SessionBatchAuditReply, *problem.Problem)
}

// GetSessionTriageUseCase surfaces sessions that need operator attention,
// ranked by anomaly severity. It wraps the existing batch audit (S467/S485)
// and projects results into a triage-oriented view with ranking and filtering.
//
// S487: Reduces friction for operators reviewing sessions — instead of scanning
// flat batch audit lists, they see the most problematic sessions first.
type GetSessionTriageUseCase struct {
	batchAuditor SessionBatchAuditor
}

func NewGetSessionTriageUseCase(batchAuditor SessionBatchAuditor) *GetSessionTriageUseCase {
	return &GetSessionTriageUseCase{batchAuditor: batchAuditor}
}

const sessionTriageDefaultLimit = 20
const sessionTriageMaxLimit = 50

func (uc *GetSessionTriageUseCase) Execute(ctx context.Context, query SessionTriageQuery) (SessionTriageReply, *problem.Problem) {
	if uc.batchAuditor == nil {
		return SessionTriageReply{}, problem.New(problem.Unavailable, "session triage dependencies unavailable")
	}

	start := time.Now()

	limit := query.Limit
	if limit <= 0 {
		limit = sessionTriageDefaultLimit
	}
	if limit > sessionTriageMaxLimit {
		limit = sessionTriageMaxLimit
	}

	// Fetch batch audit — we ask for all terminal sessions, then triage.
	auditReply, prob := uc.batchAuditor.Execute(ctx, executionclient.SessionBatchAuditQuery{
		StatusFilter: query.StatusFilter,
	})
	if prob != nil {
		return SessionTriageReply{}, prob
	}

	// Project audit entries into triage items.
	var items []triage.SessionTriageItem
	var severities []triage.TriageSeverity

	for _, entry := range auditReply.Result.Entries {
		item := projectSessionEntry(entry)

		// Apply check filter — only include sessions that failed the specified check.
		if query.CheckFilter != "" && !containsCheck(item.FailedChecks, query.CheckFilter) && !containsCheck(item.Warnings, query.CheckFilter) {
			continue
		}

		// Apply severity filter. Default: exclude clean (info with 0 anomalies) items.
		if query.SeverityFilter != "" {
			if !severityAtOrAbove(item.Severity, triage.TriageSeverity(query.SeverityFilter)) {
				continue
			}
		} else if item.AnomalyCount == 0 {
			// Default triage: skip clean sessions to surface only items needing attention.
			severities = append(severities, item.Severity)
			continue
		}

		items = append(items, item)
		severities = append(severities, item.Severity)
	}

	triage.SortSessionItems(items)

	scanned := len(items)
	if len(items) > limit {
		items = items[:limit]
	}

	return SessionTriageReply{
		Items:   items,
		Summary: triage.ComputeDomainSummary(severities, len(auditReply.Result.Entries)),
		Meta: TriageMeta{
			TotalMs:  time.Since(start).Milliseconds(),
			Scanned:  scanned,
			Returned: len(items),
		},
	}, nil
}

func projectSessionEntry(entry execution.BatchAuditEntry) triage.SessionTriageItem {
	item := triage.SessionTriageItem{
		SessionID: entry.SessionID,
	}

	if entry.Error != "" {
		item.Verdict = "errored"
		item.Severity = triage.SeverityCritical
		item.AnomalyCount = 1
		item.Findings = []triage.Finding{{
			Domain:   "session",
			Signal:   "audit_error",
			Detail:   entry.Error,
			Severity: triage.SeverityCritical,
		}}
		return item
	}

	if entry.Bundle == nil {
		item.Verdict = "errored"
		item.Severity = triage.SeverityCritical
		item.AnomalyCount = 1
		return item
	}

	b := entry.Bundle
	item.Status = string(b.Session.Status)
	item.Verdict = b.Consistency.OverallVerdict
	item.FailedChecks = b.CheckIndex.Failed
	item.Warnings = b.CheckIndex.Warnings
	item.Severity = triage.ClassifySessionSeverity(b.Consistency.OverallVerdict, len(b.CheckIndex.Failed), len(b.CheckIndex.Warnings))
	item.AnomalyCount = len(b.CheckIndex.Failed) + len(b.CheckIndex.Warnings)

	// Build findings from check failures and warnings.
	for _, checkID := range b.CheckIndex.Failed {
		item.Findings = append(item.Findings, triage.Finding{
			Domain:   "session",
			Signal:   "check_failed",
			Detail:   checkID,
			Severity: triage.SeverityCritical,
		})
	}
	for _, checkID := range b.CheckIndex.Warnings {
		item.Findings = append(item.Findings, triage.Finding{
			Domain:   "session",
			Signal:   "check_warning",
			Detail:   checkID,
			Severity: triage.SeverityWarning,
		})
	}

	if !b.Consistency.CountersMatchActivity {
		item.AnomalyCount++
		item.Findings = append(item.Findings, triage.Finding{
			Domain:   "session",
			Signal:   "counter_mismatch",
			Detail:   "session counters do not match observed activity",
			Severity: triage.SeverityWarning,
		})
	}

	return item
}

func containsCheck(checks []string, target string) bool {
	for _, c := range checks {
		if c == target {
			return true
		}
	}
	return false
}

func severityAtOrAbove(actual, threshold triage.TriageSeverity) bool {
	switch threshold {
	case triage.SeverityCritical:
		return actual == triage.SeverityCritical
	case triage.SeverityWarning:
		return actual == triage.SeverityCritical || actual == triage.SeverityWarning
	default:
		return true
	}
}
