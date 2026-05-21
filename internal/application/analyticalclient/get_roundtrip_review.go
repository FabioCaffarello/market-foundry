package analyticalclient

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"internal/domain/effectiveness"
	"internal/domain/pairing"
	"internal/shared/problem"
)

const (
	reviewDefaultLimit = 50
	reviewMaxLimit     = 200
)

// GetRoundTripReviewUseCase produces the round-trip review surface by combining
// pairing, effectiveness classification, and reconciliation into a single
// operator-facing view.
//
// S482: This builds on S480 (pairing model), S481 (pairing read model + attribution),
// and S476 (effectiveness classification) to add data-quality flags and
// reconciliation status per round-trip.
//
// Guard rails:
//   - No new ClickHouse tables; computed from existing execution data.
//   - No write-path changes.
//   - Reuses existing CompositeReader and pairing infrastructure.
type GetRoundTripReviewUseCase struct {
	reader CompositeReader
	logger *slog.Logger
}

func NewGetRoundTripReviewUseCase(reader CompositeReader, logger *slog.Logger) *GetRoundTripReviewUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetRoundTripReviewUseCase{reader: reader, logger: logger.With("component", "roundtrip_review_usecase")}
}

func (uc *GetRoundTripReviewUseCase) Execute(ctx context.Context, query RoundTripReviewQuery) (RoundTripReviewReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return RoundTripReviewReply{}, problem.New(problem.Unavailable, "round-trip review is unavailable")
	}

	start := time.Now()

	if query.CorrelationID != "" {
		return uc.executeSingle(ctx, query, start)
	}
	return uc.executeBatch(ctx, query, start)
}

func (uc *GetRoundTripReviewUseCase) executeSingle(ctx context.Context, query RoundTripReviewQuery, start time.Time) (RoundTripReviewReply, *problem.Problem) {
	if query.Symbol == "" {
		return RoundTripReviewReply{}, problem.New(problem.InvalidArgument, "symbol is required for single-chain review lookup (S301 isolation)")
	}

	chain, err := uc.reader.QueryChainByCorrelationID(ctx, query.CorrelationID, query.Symbol)
	elapsed := time.Since(start)
	if err != nil {
		return RoundTripReviewReply{}, problem.Wrap(err, problem.Unavailable, "review query failed")
	}

	var legs []pairing.Leg
	chainsScanned := 0
	if chain != nil && chain.Execution != nil {
		chainsScanned = 1
		stratDir := inferStrategyDirection(chain)
		leg := pairing.IntentToLeg(chain.Execution.ExecutionIntent, stratDir)
		legs = append(legs, leg)
	}

	roundTrips := pairing.MatchFIFO(legs, pairing.DefaultMatchingConfig())
	reviews := uc.buildReviews(roundTrips, nil, query)
	summary := buildReviewSummary(roundTrips, reviews)

	return RoundTripReviewReply{
		Reviews: reviews,
		Summary: summary,
		Source:  "clickhouse",
		Meta: ReviewMeta{
			TotalMs:       elapsed.Milliseconds(),
			ChainsScanned: chainsScanned,
			LegsProduced:  len(legs),
			RoundTrips:    len(roundTrips),
			Reviewed:      len(reviews),
		},
	}, nil
}

func (uc *GetRoundTripReviewUseCase) executeBatch(ctx context.Context, query RoundTripReviewQuery, start time.Time) (RoundTripReviewReply, *problem.Problem) {
	if query.Source == "" {
		return RoundTripReviewReply{}, problem.New(problem.InvalidArgument, "source is required for batch review")
	}
	if query.Symbol == "" {
		return RoundTripReviewReply{}, problem.New(problem.InvalidArgument, "symbol is required for batch review")
	}
	if query.Timeframe <= 0 {
		return RoundTripReviewReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive for batch review")
	}

	if query.Limit <= 0 {
		query.Limit = reviewDefaultLimit
	}
	if query.Limit > reviewMaxLimit {
		query.Limit = reviewMaxLimit
	}

	chains, err := uc.reader.QueryChainsBatch(ctx, query.Source, query.Symbol, query.Timeframe, query.Since, query.Until, query.Limit)
	elapsed := time.Since(start)
	if err != nil {
		uc.logger.Warn("review batch query failed",
			"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return RoundTripReviewReply{}, problem.Wrap(err, problem.Unavailable, "review batch query failed")
	}

	var legs []pairing.Leg
	chainMap := make(map[string]*CompositeExecutionChain, len(chains))
	for i := range chains {
		if chains[i].Execution == nil {
			continue
		}
		intent := chains[i].Execution.ExecutionIntent
		if intent.Status == "rejected" {
			continue
		}
		if len(intent.Fills) == 0 {
			continue
		}

		stratDir := inferStrategyDirection(&chains[i])
		leg := pairing.IntentToLeg(intent, stratDir)
		legs = append(legs, leg)
		chainMap[intent.CorrelationID] = &chains[i]
	}

	roundTrips := pairing.MatchFIFO(legs, pairing.DefaultMatchingConfig())
	reviews := uc.buildReviews(roundTrips, chainMap, query)
	summary := buildReviewSummary(roundTrips, reviews)

	uc.logger.Info("review batch completed",
		"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"chains_scanned", len(chains), "legs_produced", len(legs),
		"round_trips", len(roundTrips), "reviewed", len(reviews),
		"clean", summary.CleanCount, "flagged", summary.FlaggedCount,
		"total_ms", elapsed.Milliseconds(),
	)

	return RoundTripReviewReply{
		Reviews: reviews,
		Summary: summary,
		Source:  "clickhouse",
		Meta: ReviewMeta{
			TotalMs:       elapsed.Milliseconds(),
			ChainsScanned: len(chains),
			LegsProduced:  len(legs),
			RoundTrips:    len(roundTrips),
			Reviewed:      len(reviews),
		},
	}, nil
}

// buildReviews converts round-trips to review items with attribution and reconciliation,
// applying filters from the query.
func (uc *GetRoundTripReviewUseCase) buildReviews(roundTrips []pairing.RoundTrip, chainMap map[string]*CompositeExecutionChain, query RoundTripReviewQuery) []RoundTripReviewItem {
	var reviews []RoundTripReviewItem

	for _, rt := range roundTrips {
		if !matchesReviewFilters(rt, query) {
			continue
		}

		var attr *effectiveness.Attribution
		if rt.IsPaired() && rt.Entry != nil && rt.Exit != nil {
			attr = classifyPairedRoundTrip(rt, chainMap)
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

		recon := pairing.ReconcileRoundTrip(rt, attr)

		// Apply flagged filter.
		if query.Flagged && recon.Clean {
			continue
		}

		reviews = append(reviews, RoundTripReviewItem{
			RoundTrip:      rt,
			Attribution:    attr,
			Reconciliation: recon,
		})
	}

	return reviews
}

// matchesReviewFilters applies state and side filters (same as pairing).
func matchesReviewFilters(rt pairing.RoundTrip, query RoundTripReviewQuery) bool {
	if query.State != "" && string(rt.State) != query.State {
		return false
	}
	if query.Side != "" {
		legSide := ""
		if rt.Entry != nil {
			legSide = string(rt.Entry.Side)
		} else if rt.Exit != nil {
			legSide = string(rt.Exit.Side)
		}
		if legSide != query.Side {
			return false
		}
	}
	return true
}

// buildReviewSummary computes aggregated statistics from all round-trips and filtered reviews.
func buildReviewSummary(roundTrips []pairing.RoundTrip, reviews []RoundTripReviewItem) ReviewSummary {
	pr := pairing.Summarize(roundTrips)

	summary := ReviewSummary{
		TotalEntries:     pr.TotalEntries,
		TotalExits:       pr.TotalExits,
		PairedCount:      pr.PairedCount,
		UnmatchedEntries: pr.UnmatchedEntries,
		UnmatchedExits:   pr.UnmatchedExits,
		ResolvedRate:     pr.ResolvedRate,
		FlagCounts:       make(map[string]int),
	}

	// S499: Track fee coverage across all fills in the review set.
	totalFills := 0
	fillsWithFee := 0

	for _, r := range reviews {
		// Effectiveness aggregation.
		if r.Attribution != nil {
			switch r.Attribution.Outcome {
			case effectiveness.OutcomeWin:
				summary.WinCount++
				summary.TotalPnL += r.Attribution.NetPnL
			case effectiveness.OutcomeLoss:
				summary.LossCount++
				summary.TotalPnL += r.Attribution.NetPnL
			case effectiveness.OutcomeBreakeven:
				summary.BreakevenCount++
				summary.TotalPnL += r.Attribution.NetPnL
			case effectiveness.OutcomeUnresolved:
				summary.UnresolvedCount++
			}
			summary.TotalFees += r.Attribution.TotalFees
			summary.TotalCostBasis += r.Attribution.EntryCostBasis + r.Attribution.ExitCostBasis
		} else if r.State == pairing.StateUnmatchedEntry || r.State == pairing.StateUnmatchedExit {
			summary.UnresolvedCount++
		}

		// S499: Count fills with/without fee for coverage ratio.
		if r.Entry != nil {
			totalFills++
			if pf, _ := strconv.ParseFloat(r.Entry.Fee, 64); pf > 0 {
				fillsWithFee++
			}
		}
		if r.Exit != nil {
			totalFills++
			if pf, _ := strconv.ParseFloat(r.Exit.Fee, 64); pf > 0 {
				fillsWithFee++
			}
		}

		// Reconciliation aggregation.
		if r.Reconciliation.Clean {
			summary.CleanCount++
		} else {
			summary.FlaggedCount++
		}
		for _, f := range r.Reconciliation.Flags {
			summary.FlagCounts[string(f)]++
		}
		if r.Reconciliation.FeeReliable {
			summary.FeeReliableCount++
		}
		if r.Reconciliation.PnLReliable {
			summary.PnLReliableCount++
		}
	}

	// S499: Fee coverage ratio string.
	if totalFills > 0 {
		summary.FeeCoverageRatio = fmt.Sprintf("%d/%d", fillsWithFee, totalFills)
	} else {
		summary.FeeCoverageRatio = "0/0"
	}

	return summary
}
