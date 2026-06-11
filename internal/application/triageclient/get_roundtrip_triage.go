package triageclient

import (
	"context"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/triage"
	"internal/shared/problem"
)

// RoundTripReviewer fetches batch round-trip reviews for triage projection.
type RoundTripReviewer interface {
	Execute(context.Context, analyticalclient.RoundTripReviewQuery) (analyticalclient.RoundTripReviewReply, *problem.Problem)
}

// GetRoundTripTriageUseCase surfaces round-trips with data quality issues,
// ranked by severity. Wraps the round-trip review (S482) with triage projection.
//
// S487: Operators see flagged round-trips first, with clear data-quality signals.
type GetRoundTripTriageUseCase struct {
	reviewer RoundTripReviewer
}

func NewGetRoundTripTriageUseCase(reviewer RoundTripReviewer) *GetRoundTripTriageUseCase {
	return &GetRoundTripTriageUseCase{reviewer: reviewer}
}

const roundTripTriageDefaultLimit = 50
const roundTripTriageMaxLimit = 200

func (uc *GetRoundTripTriageUseCase) Execute(ctx context.Context, query RoundTripTriageQuery) (RoundTripTriageReply, *problem.Problem) {
	if uc.reviewer == nil {
		return RoundTripTriageReply{}, problem.New(problem.Unavailable, "round-trip triage dependencies unavailable")
	}

	start := time.Now()

	limit := query.Limit
	if limit <= 0 {
		limit = roundTripTriageDefaultLimit
	}
	if limit > roundTripTriageMaxLimit {
		limit = roundTripTriageMaxLimit
	}

	// Fetch flagged round-trips — the review endpoint already supports flagged filter.
	reviewReply, prob := uc.reviewer.Execute(ctx, analyticalclient.RoundTripReviewQuery{
		Source:     query.Source,
		Instrument: query.Instrument,
		Timeframe:  query.Timeframe,
		Since:      query.Since,
		Until:      query.Until,
		Limit:      limit,
		Flagged:    true, // only flagged items for triage
	})
	if prob != nil {
		return RoundTripTriageReply{}, prob
	}

	// Project reviews into triage items.
	var items []triage.RoundTripTriageItem
	var severities []triage.TriageSeverity

	for _, review := range reviewReply.Reviews {
		flagCount := len(review.Reconciliation.Flags)
		sev := triage.ClassifyRoundTripSeverity(flagCount, review.Reconciliation.PnLReliable, review.Reconciliation.FeeReliable)

		if query.SeverityFilter != "" && !severityAtOrAbove(sev, triage.TriageSeverity(query.SeverityFilter)) {
			continue
		}

		item := triage.RoundTripTriageItem{
			// S472-style projection: triage.RoundTripTriageItem.Symbol
			// stays a venue-native string by design (the triage layer
			// does not import domain packages). VenueSymbol() is the
			// transitory accessor on the migrated pairing.RoundTrip
			// (promoted via RoundTripReviewItem's anonymous embedding).
			// See PRD-0004 H-6.b'' closure for the Decision #4 cascade.
			Symbol:      review.VenueSymbol(),
			State:       string(review.State),
			Severity:    sev,
			FlagCount:   flagCount,
			PnLReliable: review.Reconciliation.PnLReliable,
			FeeReliable: review.Reconciliation.FeeReliable,
		}

		for _, f := range review.Reconciliation.Flags {
			item.Flags = append(item.Flags, string(f))
		}

		if review.Entry != nil {
			item.CorrelationID = review.Entry.CorrelationID
		}

		if review.Attribution != nil {
			item.Outcome = string(review.Attribution.Outcome)
		}

		items = append(items, item)
		severities = append(severities, sev)
	}

	triage.SortRoundTripItems(items)

	scanned := len(items)
	if len(items) > limit {
		items = items[:limit]
	}

	return RoundTripTriageReply{
		Items:   items,
		Summary: triage.ComputeDomainSummary(severities, reviewReply.Summary.PairedCount+reviewReply.Summary.UnmatchedEntries+reviewReply.Summary.UnmatchedExits),
		Source:  "clickhouse",
		Meta: TriageMeta{
			TotalMs:  time.Since(start).Milliseconds(),
			Scanned:  scanned,
			Returned: len(items),
		},
	}, nil
}
