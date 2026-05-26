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

const (
	crossSessionDefaultMaxSessions = 30
	crossSessionMaxMaxSessions     = 50
	crossSessionDefaultLimit       = 200
)

// CrossSessionSessionReader reads session metadata for cross-session discovery.
// This is a narrow interface to avoid coupling to the full SessionGateway port.
type CrossSessionSessionReader interface {
	ListSessions(ctx context.Context) ([]execution.Session, error)
}

// GetCrossSessionPairingUseCase computes the cross-session pairing read model
// by fetching sessions from KV, querying chains per session from ClickHouse,
// applying carry-forward eligibility rules, assembling a cross-session leg set,
// running FIFO matching, and annotating results with continuity classification.
//
// S495: This is the application-layer orchestration for the S494 continuity model.
// It composes data from two sources (KV sessions + ClickHouse chains) into a
// single cross-session pairing response.
//
// Guard rails:
//   - No write-path changes.
//   - No new ClickHouse tables; uses existing execution data.
//   - No position engine; retrospective read model.
//   - No runtime carry-forward; sessions remain isolated at runtime.
type GetCrossSessionPairingUseCase struct {
	sessions CrossSessionSessionReader
	reader   CompositeReader
	logger   *slog.Logger
}

func NewGetCrossSessionPairingUseCase(
	sessions CrossSessionSessionReader,
	reader CompositeReader,
	logger *slog.Logger,
) *GetCrossSessionPairingUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetCrossSessionPairingUseCase{
		sessions: sessions,
		reader:   reader,
		logger:   logger.With("component", "cross_session_pairing_usecase"),
	}
}

func (uc *GetCrossSessionPairingUseCase) Execute(ctx context.Context, query CrossSessionPairingQuery) (CrossSessionPairingReply, *problem.Problem) {
	if uc == nil || uc.reader == nil || uc.sessions == nil {
		return CrossSessionPairingReply{}, problem.New(problem.Unavailable, "cross-session pairing reader is unavailable")
	}

	start := time.Now()

	// Validate required fields.
	if query.Source == "" || query.Symbol == "" || query.Timeframe <= 0 || query.Since <= 0 {
		return CrossSessionPairingReply{}, problem.New(problem.InvalidArgument,
			"source, symbol, timeframe, and since are required for cross-session pairing")
	}

	maxSessions := query.MaxSessions
	if maxSessions <= 0 {
		maxSessions = crossSessionDefaultMaxSessions
	}
	if maxSessions > crossSessionMaxMaxSessions {
		maxSessions = crossSessionMaxMaxSessions
	}

	// Step 1: Fetch all sessions from KV and filter by time window.
	allSessions, err := uc.sessions.ListSessions(ctx)
	if err != nil {
		return CrossSessionPairingReply{}, problem.New(problem.Unavailable, "failed to list sessions: "+err.Error())
	}

	sinceTime := time.Unix(query.Since, 0)
	var untilTime time.Time
	if query.Until > 0 {
		untilTime = time.Unix(query.Until, 0)
	} else {
		untilTime = time.Now()
	}

	// Filter sessions within the time window and sort chronologically.
	var filtered []execution.Session
	for _, s := range allSessions {
		if s.StartedAt.Before(sinceTime) {
			continue
		}
		if s.StartedAt.After(untilTime) {
			continue
		}
		filtered = append(filtered, s)
	}

	// Sort by start time ascending (chronological).
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartedAt.Before(filtered[j].StartedAt)
	})

	// Limit to maxSessions (most recent).
	if len(filtered) > maxSessions {
		filtered = filtered[len(filtered)-maxSessions:]
	}

	if len(filtered) == 0 {
		elapsed := time.Since(start)
		return CrossSessionPairingReply{
			RoundTrips: []CrossSessionRoundTripView{},
			Summary:    CrossSessionPairingSummary{},
			Continuity: pairing.ContinuitySummary{},
			Source:     "clickhouse+kv",
			Meta: CrossSessionPairingMeta{
				TotalMs:         elapsed.Milliseconds(),
				SessionsFetched: 0,
			},
		}, nil
	}

	// Step 2: For each session, fetch chains and build session legs.
	// CrossSessionWindow.VenueSymbol is a venue-native string filter
	// (Decision #2 of H-6.b''); query.Symbol is the lowercase form
	// supplied by the HTTP query parameter and is passed through
	// without canonical-instrument reconstruction.
	window := pairing.CrossSessionWindow{
		VenueSymbol: query.Symbol,
		Source:      query.Source,
		Timeframe:   query.Timeframe,
		Since:       sinceTime,
		Until:       untilTime,
		MaxSessions: maxSessions,
	}

	var allLegs []pairing.SessionLeg
	var sessionIDs []string
	chainMap := make(map[string]*CompositeExecutionChain)
	var totalChains, totalExcluded int

	for _, sess := range filtered {
		// Derive time bounds from session for ClickHouse query.
		sessSince := sess.StartedAt.Unix()
		var sessUntil int64
		if sess.ClosedAt != nil {
			sessUntil = sess.ClosedAt.Unix()
		} else {
			sessUntil = untilTime.Unix()
		}

		chains, qErr := uc.reader.QueryChainsBatch(ctx, query.Source, query.Symbol, query.Timeframe, sessSince, sessUntil, crossSessionDefaultLimit)
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

			// Apply carry-forward eligibility.
			eligibility := pairing.ClassifyCarryForward(intent)
			if eligibility != pairing.CarryEligible {
				totalExcluded++
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

	// Step 3: Sort legs by timestamp (FIFO invariant M4 across sessions).
	sort.Slice(allLegs, func(i, j int) bool {
		return allLegs[i].Timestamp.Before(allLegs[j].Timestamp)
	})

	legSet := pairing.CrossSessionLegSet{
		Window:   window,
		Sessions: sessionIDs,
		Legs:     allLegs,
	}

	// Step 4: Run FIFO matching on extracted legs.
	plainLegs := legSet.ExtractLegs()
	roundTrips := pairing.MatchFIFO(plainLegs, pairing.DefaultMatchingConfig())

	// Step 5: Annotate with session provenance and continuity classification.
	sessionIndex := legSet.SessionLegIndex()
	csRoundTrips := pairing.AnnotateRoundTrips(roundTrips, sessionIndex)

	// Step 6: Compute effectiveness attribution for paired round-trips.
	views := uc.buildCrossSessionViews(csRoundTrips, chainMap, query)

	// Step 7: Compute summaries.
	continuity := pairing.SummarizeContinuity(csRoundTrips)
	summary := uc.buildCrossSessionSummary(views, csRoundTrips, len(sessionIDs))

	elapsed := time.Since(start)

	uc.logger.Info("cross-session pairing completed",
		"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"sessions", len(sessionIDs), "chains", totalChains, "legs", len(allLegs),
		"round_trips", len(views), "resolved", continuity.ResolvedCount,
		"cross_session_pairs", continuity.CrossSessionPairedCount,
		"resolution_rate", continuity.ResolutionRate,
		"total_ms", elapsed.Milliseconds(),
	)

	return CrossSessionPairingReply{
		RoundTrips: views,
		Summary:    summary,
		Continuity: continuity,
		Source:     "clickhouse+kv",
		Meta: CrossSessionPairingMeta{
			TotalMs:         elapsed.Milliseconds(),
			SessionsFetched: len(filtered),
			ChainsScanned:   totalChains,
			LegsProduced:    len(allLegs),
			LegsCarried:     len(allLegs),
			LegsExcluded:    totalExcluded,
			RoundTrips:      len(views),
		},
	}, nil
}

// buildCrossSessionViews converts annotated round-trips to HTTP views with attribution,
// applying continuity and cross-only filters.
func (uc *GetCrossSessionPairingUseCase) buildCrossSessionViews(
	csRoundTrips []pairing.CrossSessionRoundTrip,
	chainMap map[string]*CompositeExecutionChain,
	query CrossSessionPairingQuery,
) []CrossSessionRoundTripView {
	var views []CrossSessionRoundTripView

	for _, csrt := range csRoundTrips {
		// Apply continuity filter.
		if query.Continuity != "" && string(csrt.Continuity) != query.Continuity {
			continue
		}

		// Apply cross-only filter.
		if query.CrossOnly && !csrt.CrossSession {
			continue
		}

		view := CrossSessionRoundTripView{CrossSessionRoundTrip: csrt}

		// Compute attribution for paired round-trips.
		if csrt.IsPaired() && csrt.Entry != nil && csrt.Exit != nil && chainMap != nil {
			entryChain := chainMap[csrt.Entry.CorrelationID]
			exitChain := chainMap[csrt.Exit.CorrelationID]
			if entryChain != nil && exitChain != nil &&
				entryChain.Execution != nil && exitChain.Execution != nil {
				attr := effectiveness.ClassifyPair(
					entryChain.Execution.ExecutionIntent,
					exitChain.Execution.ExecutionIntent,
				)
				if attr != nil {
					enrichFromChain(attr, entryChain)
					view.Attribution = attr
				}
			}
		}

		views = append(views, view)
	}

	if views == nil {
		views = []CrossSessionRoundTripView{}
	}

	return views
}

// buildCrossSessionSummary aggregates statistics from cross-session round-trips.
func (uc *GetCrossSessionPairingUseCase) buildCrossSessionSummary(
	views []CrossSessionRoundTripView,
	csRoundTrips []pairing.CrossSessionRoundTrip,
	sessionsScanned int,
) CrossSessionPairingSummary {
	// Build base pairing summary from underlying round-trips.
	var baseRTs []pairing.RoundTrip
	for _, csrt := range csRoundTrips {
		baseRTs = append(baseRTs, csrt.RoundTrip)
	}
	var baseViews []RoundTripView
	for _, v := range views {
		baseViews = append(baseViews, RoundTripView{
			RoundTrip:   v.RoundTrip,
			Attribution: v.Attribution,
		})
	}
	baseSummary := buildPairingSummary(baseRTs, baseViews)

	summary := CrossSessionPairingSummary{
		PairingSummary:  baseSummary,
		SessionsScanned: sessionsScanned,
	}

	// Count cross-session vs intra-session pairs.
	var crossPairs, intraPairs int
	var artificialBefore int // count of artificial_unresolved (boundary artifacts remaining)
	for _, csrt := range csRoundTrips {
		if csrt.IsPaired() {
			if csrt.CrossSession {
				crossPairs++
			} else {
				intraPairs++
			}
		}
		if csrt.Continuity == pairing.ContinuityArtificialUnresolved {
			artificialBefore++
		}
	}

	summary.CrossSessionPairs = crossPairs
	summary.IntraSessionPairs = intraPairs

	// CarryForwardResolutionRate: cross-session pairs resolved / (cross resolved + still artificial).
	denom := crossPairs + artificialBefore
	if denom > 0 {
		summary.CarryForwardResolutionRate = float64(crossPairs) / float64(denom)
	}

	return summary
}
