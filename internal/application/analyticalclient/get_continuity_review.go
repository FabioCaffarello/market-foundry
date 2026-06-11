package analyticalclient

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"internal/domain/effectiveness"
	"internal/domain/execution"
	"internal/domain/pairing"
	"internal/shared/problem"
)

// GetContinuityReviewUseCase produces the continuity review surface by
// combining cross-session pairing, reconciliation, and effectiveness
// into a unified operator-facing view.
//
// S496: This builds on S494 (continuity model), S495 (cross-session read model),
// S482 (reconciliation), and S476 (effectiveness) to provide a single surface
// that answers: "what was carried between sessions, what resolved, and what
// remains open — with full data-quality and P&L context?"
//
// Guard rails:
//   - No new ClickHouse tables; computed from existing execution data.
//   - No write-path changes.
//   - Reuses existing CompositeReader and session reader infrastructure.
//   - No dashboard; minimal review surface only.
type GetContinuityReviewUseCase struct {
	sessions CrossSessionSessionReader
	reader   CompositeReader
	logger   *slog.Logger
}

func NewGetContinuityReviewUseCase(
	sessions CrossSessionSessionReader,
	reader CompositeReader,
	logger *slog.Logger,
) *GetContinuityReviewUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetContinuityReviewUseCase{
		sessions: sessions,
		reader:   reader,
		logger:   logger.With("component", "continuity_review_usecase"),
	}
}

func (uc *GetContinuityReviewUseCase) Execute(ctx context.Context, query ContinuityReviewQuery) (ContinuityReviewReply, *problem.Problem) {
	if uc == nil || uc.reader == nil || uc.sessions == nil {
		return ContinuityReviewReply{}, problem.New(problem.Unavailable, "continuity review is unavailable")
	}

	start := time.Now()

	// Validate required fields.
	if query.Source == "" || query.Instrument.IsZero() || query.Timeframe <= 0 || query.Since <= 0 {
		return ContinuityReviewReply{}, problem.New(problem.InvalidArgument,
			"source, symbol, timeframe, and since are required for continuity review")
	}

	maxSessions := query.MaxSessions
	if maxSessions <= 0 {
		maxSessions = crossSessionDefaultMaxSessions
	}
	if maxSessions > crossSessionMaxMaxSessions {
		maxSessions = crossSessionMaxMaxSessions
	}

	// Step 1: Fetch sessions from KV.
	allSessions, err := uc.sessions.ListSessions(ctx)
	if err != nil {
		return ContinuityReviewReply{}, problem.New(problem.Unavailable, "failed to list sessions: "+err.Error())
	}

	sinceTime := time.Unix(query.Since, 0)
	var untilTime time.Time
	if query.Until > 0 {
		untilTime = time.Unix(query.Until, 0)
	} else {
		untilTime = time.Now()
	}

	var filtered []execution.Session
	for _, s := range allSessions {
		if s.StartedAt.Before(sinceTime) || s.StartedAt.After(untilTime) {
			continue
		}
		filtered = append(filtered, s)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartedAt.Before(filtered[j].StartedAt)
	})

	if len(filtered) > maxSessions {
		filtered = filtered[len(filtered)-maxSessions:]
	}

	if len(filtered) == 0 {
		elapsed := time.Since(start)
		return ContinuityReviewReply{
			Reviews:        []ContinuityReviewItem{},
			Continuity:     pairing.ContinuitySummary{},
			Reconciliation: pairing.ContinuityReconciliationSummary{FlagCounts: make(map[string]int)},
			Effectiveness:  ContinuityEffectivenessSummary{},
			Source:         "clickhouse+kv",
			Meta: ContinuityReviewMeta{
				TotalMs:         elapsed.Milliseconds(),
				SessionsFetched: 0,
			},
		}, nil
	}

	// Step 2: Fetch chains per session and build session legs.
	var allLegs []pairing.SessionLeg
	var sessionIDs []string
	chainMap := make(map[string]*CompositeExecutionChain)
	var totalChains int

	for _, sess := range filtered {
		sessSince := sess.StartedAt.Unix()
		var sessUntil int64
		if sess.ClosedAt != nil {
			sessUntil = sess.ClosedAt.Unix()
		} else {
			sessUntil = untilTime.Unix()
		}

		chains, qErr := uc.reader.QueryChainsBatch(ctx, query.Source, query.Instrument, query.Timeframe, sessSince, sessUntil, crossSessionDefaultLimit)
		if qErr != nil {
			uc.logger.Warn("chain query failed for session",
				"session_id", sess.SessionID, "error", qErr)
			continue
		}

		if len(chains) == 0 {
			continue
		}

		sessionIDs = append(sessionIDs, sess.SessionID)
		totalChains += len(chains)

		for i := range chains {
			if chains[i].Execution == nil {
				continue
			}
			intent := chains[i].Execution.ExecutionIntent

			eligibility := pairing.ClassifyCarryForward(intent)
			if eligibility != pairing.CarryEligible {
				continue
			}

			stratDir := inferStrategyDirection(&chains[i])
			leg := pairing.IntentToLeg(intent, stratDir)

			sessionLeg := pairing.SessionLeg{
				Leg:              leg,
				SessionID:        sess.SessionID,
				SessionStartedAt: sess.StartedAt,
				SessionClosedAt:  sess.ClosedAt,
			}

			allLegs = append(allLegs, sessionLeg)
			chainMap[intent.CorrelationID] = &chains[i]
		}
	}

	// Step 3: Sort by timestamp (FIFO invariant M4).
	sort.Slice(allLegs, func(i, j int) bool {
		return allLegs[i].Timestamp.Before(allLegs[j].Timestamp)
	})

	// CrossSessionWindow.VenueSymbol stays as a venue-native string
	// filter per Decision #2 of H-6.b'' (no Instrument promotion —
	// would force regression-prone source-string reconstruction).
	legSet := pairing.CrossSessionLegSet{
		Window: pairing.CrossSessionWindow{
			VenueSymbol: query.Instrument.LegacyFilterValue(),
			Source:      query.Source,
			Timeframe:   query.Timeframe,
			Since:       sinceTime,
			Until:       untilTime,
			MaxSessions: maxSessions,
		},
		Sessions: sessionIDs,
		Legs:     allLegs,
	}

	// Step 4: FIFO matching.
	plainLegs := legSet.ExtractLegs()
	roundTrips := pairing.MatchFIFO(plainLegs, pairing.DefaultMatchingConfig())

	// Step 5: Annotate with session provenance and continuity.
	sessionIndex := legSet.SessionLegIndex()
	csRoundTrips := pairing.AnnotateRoundTrips(roundTrips, sessionIndex)

	// Step 6: Build review items with reconciliation and effectiveness.
	reviews, reconResults := uc.buildReviewItems(csRoundTrips, chainMap, query)

	// Step 7: Compute summaries.
	continuity := pairing.SummarizeContinuity(csRoundTrips)
	reconSummary := pairing.SummarizeContinuityReconciliation(reconResults)
	effectSummary := buildContinuityEffectivenessSummary(reviews)

	elapsed := time.Since(start)

	uc.logger.Info("continuity review completed",
		"source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
		"sessions", len(sessionIDs), "chains", totalChains, "legs", len(allLegs),
		"round_trips", len(csRoundTrips), "reviewed", len(reviews),
		"resolved", continuity.ResolvedCount,
		"cross_session_pairs", continuity.CrossSessionPairedCount,
		"clean", reconSummary.CleanCount, "flagged", reconSummary.FlaggedCount,
		"total_ms", elapsed.Milliseconds(),
	)

	return ContinuityReviewReply{
		Reviews:        reviews,
		Continuity:     continuity,
		Reconciliation: reconSummary,
		Effectiveness:  effectSummary,
		Source:         "clickhouse+kv",
		Meta: ContinuityReviewMeta{
			TotalMs:         elapsed.Milliseconds(),
			SessionsFetched: len(filtered),
			ChainsScanned:   totalChains,
			LegsProduced:    len(allLegs),
			RoundTrips:      len(csRoundTrips),
			Reviewed:        len(reviews),
		},
	}, nil
}

// buildReviewItems converts cross-session round-trips into review items with
// reconciliation and effectiveness, applying query filters.
func (uc *GetContinuityReviewUseCase) buildReviewItems(
	csRoundTrips []pairing.CrossSessionRoundTrip,
	chainMap map[string]*CompositeExecutionChain,
	query ContinuityReviewQuery,
) ([]ContinuityReviewItem, []pairing.ContinuityReconciliationResult) {
	var reviews []ContinuityReviewItem
	var reconResults []pairing.ContinuityReconciliationResult

	for _, csrt := range csRoundTrips {
		// Apply continuity filter.
		if query.Continuity != "" && string(csrt.Continuity) != query.Continuity {
			continue
		}

		// Apply cross-only filter.
		if query.CrossOnly && !csrt.CrossSession {
			continue
		}

		// Compute effectiveness attribution for paired round-trips.
		var attr *effectiveness.Attribution
		if csrt.IsPaired() && csrt.Entry != nil && csrt.Exit != nil && chainMap != nil {
			entryChain := chainMap[csrt.Entry.CorrelationID]
			exitChain := chainMap[csrt.Exit.CorrelationID]
			if entryChain != nil && exitChain != nil &&
				entryChain.Execution != nil && exitChain.Execution != nil {
				attr = effectiveness.ClassifyPair(
					entryChain.Execution.ExecutionIntent,
					exitChain.Execution.ExecutionIntent,
				)
				if attr != nil {
					enrichFromChain(attr, entryChain)
				}
			}
		}

		// Apply outcome filter.
		if query.Outcome != "" {
			if attr == nil && query.Outcome != "unresolved" {
				continue
			}
			if attr != nil && string(attr.Outcome) != query.Outcome {
				continue
			}
		}

		// Compute continuity reconciliation.
		recon := pairing.ReconcileCrossSessionRoundTrip(csrt, attr)
		reconResults = append(reconResults, recon)

		// Apply flagged filter.
		if query.Flagged && recon.Clean {
			continue
		}

		reviews = append(reviews, ContinuityReviewItem{
			CrossSessionRoundTrip: csrt,
			Attribution:           attr,
			Reconciliation:        recon,
		})
	}

	if reviews == nil {
		reviews = []ContinuityReviewItem{}
	}

	return reviews, reconResults
}

// buildContinuityEffectivenessSummary aggregates effectiveness outcomes
// across the review items, distinguishing cross-session from intra-session.
func buildContinuityEffectivenessSummary(reviews []ContinuityReviewItem) ContinuityEffectivenessSummary {
	var s ContinuityEffectivenessSummary

	for _, r := range reviews {
		if r.Attribution == nil {
			continue
		}

		s.TotalPaired++
		s.TotalFees += r.Attribution.TotalFees
		s.TotalGrossPnL += r.Attribution.GrossPnL
		s.TotalNetPnL += r.Attribution.NetPnL

		switch r.Attribution.Outcome {
		case effectiveness.OutcomeWin:
			s.WinCount++
			if r.CrossSession {
				s.CrossSessionWins++
				s.CrossSessionPnL += r.Attribution.NetPnL
			} else {
				s.IntraSessionWins++
				s.IntraSessionPnL += r.Attribution.NetPnL
			}
		case effectiveness.OutcomeLoss:
			s.LossCount++
			if r.CrossSession {
				s.CrossSessionLosses++
				s.CrossSessionPnL += r.Attribution.NetPnL
			} else {
				s.IntraSessionLosses++
				s.IntraSessionPnL += r.Attribution.NetPnL
			}
		case effectiveness.OutcomeBreakeven:
			s.BreakevenCount++
			if r.CrossSession {
				s.CrossSessionPnL += r.Attribution.NetPnL
			} else {
				s.IntraSessionPnL += r.Attribution.NetPnL
			}
		case effectiveness.OutcomeUnresolved:
			s.UnresolvedCount++
		}
	}

	return s
}
