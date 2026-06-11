package triageclient

import (
	"context"
	"sort"
	"time"

	"internal/domain/triage"
	"internal/shared/problem"
)

// GetTriageOverviewUseCase produces a cross-domain "what needs attention?"
// summary by combining session, decision, and round-trip triage signals.
//
// S487: Single-call operational triage overview. Returns severity counts
// per domain and the top findings across all domains, ranked by severity.
type GetTriageOverviewUseCase struct {
	sessionTriage   *GetSessionTriageUseCase
	decisionTriage  *GetDecisionTriageUseCase
	roundTripTriage *GetRoundTripTriageUseCase
}

func NewGetTriageOverviewUseCase(
	sessionTriage *GetSessionTriageUseCase,
	decisionTriage *GetDecisionTriageUseCase,
	roundTripTriage *GetRoundTripTriageUseCase,
) *GetTriageOverviewUseCase {
	return &GetTriageOverviewUseCase{
		sessionTriage:   sessionTriage,
		decisionTriage:  decisionTriage,
		roundTripTriage: roundTripTriage,
	}
}

const triageOverviewTopFindings = 10

func (uc *GetTriageOverviewUseCase) Execute(ctx context.Context, query TriageOverviewQuery) (TriageOverviewReply, *problem.Problem) {
	start := time.Now()

	overview := triage.TriageOverview{}
	var allFindings []triage.Finding
	totalAnomalies := 0

	// Session triage — always attempted if available.
	if uc.sessionTriage != nil {
		sessionReply, prob := uc.sessionTriage.Execute(ctx, SessionTriageQuery{
			StatusFilter: query.SessionStatus,
			Limit:        sessionTriageMaxLimit,
		})
		if prob == nil {
			overview.SessionSummary = sessionReply.Summary
			for _, item := range sessionReply.Items {
				allFindings = append(allFindings, item.Findings...)
				totalAnomalies += item.AnomalyCount
			}
		}
	}

	// Decision triage — requires source/symbol/timeframe.
	if uc.decisionTriage != nil && query.Source != "" && !query.Instrument.IsZero() && query.Timeframe > 0 {
		decisionReply, prob := uc.decisionTriage.Execute(ctx, DecisionTriageQuery{
			Source:     query.Source,
			Instrument: query.Instrument,
			Timeframe:  query.Timeframe,
			Since:      query.Since,
			Until:      query.Until,
			Limit:      decisionTriageMaxLimit,
		})
		if prob == nil {
			overview.DecisionSummary = decisionReply.Summary
			for _, item := range decisionReply.Items {
				allFindings = append(allFindings, item.Findings...)
				totalAnomalies += len(item.Findings)
			}
		}
	}

	// Round-trip triage — requires source/symbol/timeframe.
	if uc.roundTripTriage != nil && query.Source != "" && !query.Instrument.IsZero() && query.Timeframe > 0 {
		rtReply, prob := uc.roundTripTriage.Execute(ctx, RoundTripTriageQuery{
			Source:     query.Source,
			Instrument: query.Instrument,
			Timeframe:  query.Timeframe,
			Since:      query.Since,
			Until:      query.Until,
			Limit:      roundTripTriageMaxLimit,
		})
		if prob == nil {
			overview.RoundTripSummary = rtReply.Summary
			totalAnomalies += rtReply.Summary.Critical + rtReply.Summary.Warning
		}
	}

	// Sort findings by severity, take top N.
	sort.Slice(allFindings, func(i, j int) bool {
		ri := severityRankForSort(allFindings[i].Severity)
		rj := severityRankForSort(allFindings[j].Severity)
		return ri < rj
	})

	if len(allFindings) > triageOverviewTopFindings {
		allFindings = allFindings[:triageOverviewTopFindings]
	}

	overview.TopFindings = allFindings
	overview.TotalAnomalies = totalAnomalies

	return TriageOverviewReply{
		Overview: overview,
		Meta: TriageMeta{
			TotalMs: time.Since(start).Milliseconds(),
		},
	}, nil
}

func severityRankForSort(s triage.TriageSeverity) int {
	switch s {
	case triage.SeverityCritical:
		return 0
	case triage.SeverityWarning:
		return 1
	default:
		return 2
	}
}
