package triageclient

import (
	"context"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/triage"
	"internal/shared/problem"
)

// DecisionReviewer fetches batch decision reviews for triage projection.
type DecisionReviewer interface {
	Execute(context.Context, analyticalclient.DecisionReviewQuery) (analyticalclient.DecisionReviewReply, *problem.Problem)
}

// GetDecisionTriageUseCase surfaces decisions with consistency violations,
// incomplete chains, or problematic effectiveness outcomes, ranked by severity.
//
// S487: Wraps decision review (S471) and consistency checks (S472) into a
// triage-oriented view so operators see the most problematic decisions first.
type GetDecisionTriageUseCase struct {
	reviewer DecisionReviewer
}

func NewGetDecisionTriageUseCase(reviewer DecisionReviewer) *GetDecisionTriageUseCase {
	return &GetDecisionTriageUseCase{reviewer: reviewer}
}

const decisionTriageDefaultLimit = 20
const decisionTriageMaxLimit = 100

func (uc *GetDecisionTriageUseCase) Execute(ctx context.Context, query DecisionTriageQuery) (DecisionTriageReply, *problem.Problem) {
	if uc.reviewer == nil {
		return DecisionTriageReply{}, problem.New(problem.Unavailable, "decision triage dependencies unavailable")
	}

	start := time.Now()

	limit := query.Limit
	if limit <= 0 {
		limit = decisionTriageDefaultLimit
	}
	if limit > decisionTriageMaxLimit {
		limit = decisionTriageMaxLimit
	}

	// Fetch decision reviews — ask for more than limit to have room after filtering.
	fetchLimit := limit * 3
	if fetchLimit > decisionTriageMaxLimit {
		fetchLimit = decisionTriageMaxLimit
	}

	reviewReply, prob := uc.reviewer.Execute(ctx, analyticalclient.DecisionReviewQuery{
		Source:     query.Source,
		Instrument: query.Instrument,
		Timeframe:  query.Timeframe,
		Since:      query.Since,
		Until:      query.Until,
		Limit:      fetchLimit,
	})
	if prob != nil {
		return DecisionTriageReply{}, prob
	}

	// Project reviews into triage items, keeping only those with anomalies.
	var items []triage.DecisionTriageItem
	var allSeverities []triage.TriageSeverity

	for _, review := range reviewReply.Reviews {
		violations := 0
		if review.Consistency != nil {
			violations = review.Consistency.Violations
		}

		incomplete := !review.ChainComplete

		sev := triage.ClassifyDecisionSeverity(violations, incomplete)
		allSeverities = append(allSeverities, sev)

		// Only include items that have actual anomalies (not info-level clean items)
		// unless no severity filter is applied.
		if sev == triage.SeverityInfo && query.SeverityFilter == "" {
			continue // skip clean items from triage list (they're counted in summary)
		}

		if query.SeverityFilter != "" && !severityAtOrAbove(sev, triage.TriageSeverity(query.SeverityFilter)) {
			continue
		}

		item := triage.DecisionTriageItem{
			CorrelationID: review.CorrelationID,
			Severity:      sev,
			Violations:    violations,
			Incomplete:    incomplete,
		}

		if review.Transform != nil {
			item.Symbol = review.Transform.Symbol
			item.DecisionType = review.Transform.Type
			item.Outcome = review.Transform.Outcome
		}

		if review.Effectiveness != nil {
			item.Effectiveness = review.Effectiveness.Outcome
		}

		// Build findings.
		if violations > 0 && review.Consistency != nil {
			for _, f := range review.Consistency.Findings {
				item.Findings = append(item.Findings, triage.Finding{
					Domain:   "decision",
					Signal:   "consistency_" + string(f.Severity),
					Detail:   f.Message,
					Severity: consistencyToTriageSeverity(string(f.Severity)),
				})
			}
		}

		if incomplete {
			item.Findings = append(item.Findings, triage.Finding{
				Domain:   "decision",
				Signal:   "incomplete_chain",
				Detail:   "chain missing stages: " + joinStages(review.MissingStages),
				Severity: triage.SeverityWarning,
			})
		}

		items = append(items, item)
	}

	triage.SortDecisionItems(items)

	scanned := len(items)
	if len(items) > limit {
		items = items[:limit]
	}

	return DecisionTriageReply{
		Items:   items,
		Summary: triage.ComputeDomainSummary(allSeverities, len(reviewReply.Reviews)),
		Source:  "clickhouse",
		Meta: TriageMeta{
			TotalMs:  time.Since(start).Milliseconds(),
			Scanned:  scanned,
			Returned: len(items),
		},
	}, nil
}

func consistencyToTriageSeverity(consistencySeverity string) triage.TriageSeverity {
	switch consistencySeverity {
	case "violation":
		return triage.SeverityCritical
	case "warning":
		return triage.SeverityWarning
	default:
		return triage.SeverityInfo
	}
}

func joinStages(stages []string) string {
	if len(stages) == 0 {
		return "unknown"
	}
	result := stages[0]
	for i := 1; i < len(stages); i++ {
		result += ", " + stages[i]
	}
	return result
}
