package analyticalclient

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"internal/domain/effectiveness"
	"internal/domain/pairing"
	"internal/shared/problem"
)

const (
	summaryDefaultLimit = 100
	summaryMaxLimit     = 300
)

// GetEffectivenessSummaryUseCase computes cohort-level effectiveness aggregation
// and comparative analysis across decision chains.
//
// S477: Answers Q-SE5 (comparative effectiveness analysis) by aggregating
// individual effectiveness attributions into cohort summaries, optionally
// grouped by decision_type, strategy_type, severity, or source.
//
// This is a read-path computation — no new write-side changes or ClickHouse
// tables required. It reuses the same CompositeReader and effectiveness.Classify
// as S476, adding only aggregation logic.
type GetEffectivenessSummaryUseCase struct {
	reader CompositeReader
	logger *slog.Logger
}

func NewGetEffectivenessSummaryUseCase(reader CompositeReader, logger *slog.Logger) *GetEffectivenessSummaryUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetEffectivenessSummaryUseCase{reader: reader, logger: logger.With("component", "effectiveness_summary_usecase")}
}

func (uc *GetEffectivenessSummaryUseCase) Execute(ctx context.Context, query EffectivenessSummaryQuery) (EffectivenessSummaryReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return EffectivenessSummaryReply{}, problem.New(problem.Unavailable, "effectiveness summary reader is unavailable")
	}

	if query.Source == "" {
		return EffectivenessSummaryReply{}, problem.New(problem.InvalidArgument, "source is required for effectiveness summary")
	}
	if query.Symbol == "" {
		return EffectivenessSummaryReply{}, problem.New(problem.InvalidArgument, "symbol is required for effectiveness summary")
	}
	if query.Timeframe <= 0 {
		return EffectivenessSummaryReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive for effectiveness summary")
	}
	if query.GroupBy != "" && !ValidGroupBy(query.GroupBy) {
		return EffectivenessSummaryReply{}, problem.New(problem.InvalidArgument, "group_by must be one of: decision_type, strategy_type, severity, source")
	}

	if query.Limit <= 0 {
		query.Limit = summaryDefaultLimit
	}
	if query.Limit > summaryMaxLimit {
		query.Limit = summaryMaxLimit
	}

	start := time.Now()

	chains, err := uc.reader.QueryChainsBatch(ctx, query.Source, query.Symbol, query.Timeframe, query.Since, query.Until, query.Limit)
	elapsed := time.Since(start)
	if err != nil {
		uc.logger.Warn("effectiveness summary query failed",
			"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return EffectivenessSummaryReply{}, problem.Wrap(err, problem.Unavailable, "effectiveness summary query failed")
	}

	// S481: Build legs and chain index for round-trip pairing integration.
	var legs []pairing.Leg
	chainByCorr := make(map[string]*CompositeExecutionChain, len(chains))
	var excluded int

	for i := range chains {
		if chains[i].Execution == nil {
			continue
		}
		intent := chains[i].Execution.ExecutionIntent
		if intent.Status == "rejected" {
			excluded++
			continue
		}

		chainByCorr[intent.CorrelationID] = &chains[i]

		if len(intent.Fills) > 0 {
			stratDir := inferStrategyDirection(&chains[i])
			leg := pairing.IntentToLeg(intent, stratDir)
			legs = append(legs, leg)
		}
	}

	roundTrips := pairing.MatchFIFO(legs, pairing.DefaultMatchingConfig())

	// Classify paired round-trips first, then unpaired chains.
	pairedCorrs := make(map[string]bool)
	var attributions []effectiveness.Attribution

	for _, rt := range roundTrips {
		if !rt.IsPaired() || rt.Entry == nil || rt.Exit == nil {
			continue
		}
		entryChain := chainByCorr[rt.Entry.CorrelationID]
		exitChain := chainByCorr[rt.Exit.CorrelationID]
		if entryChain == nil || exitChain == nil || entryChain.Execution == nil || exitChain.Execution == nil {
			continue
		}
		attr := effectiveness.ClassifyPair(entryChain.Execution.ExecutionIntent, exitChain.Execution.ExecutionIntent)
		if attr == nil {
			continue
		}
		enrichFromChain(attr, entryChain)
		pairedCorrs[rt.Entry.CorrelationID] = true
		pairedCorrs[rt.Exit.CorrelationID] = true
		if matchesSummaryFilters(attr, query) {
			attributions = append(attributions, *attr)
		}
	}

	// Unpaired chains fall back to single-leg Classify.
	for i := range chains {
		if chains[i].Execution == nil {
			continue
		}
		if pairedCorrs[chains[i].Execution.CorrelationID] {
			continue
		}
		attr := effectiveness.Classify(chains[i].Execution.ExecutionIntent)
		if attr == nil {
			continue
		}
		enrichFromChain(attr, &chains[i])
		if matchesSummaryFilters(attr, query) {
			attributions = append(attributions, *attr)
		}
	}

	// Aggregate into cohorts.
	var cohorts []CohortSummary
	if query.GroupBy == "" {
		cohorts = []CohortSummary{aggregateCohort("all", attributions)}
	} else {
		cohorts = aggregateByDimension(query.GroupBy, attributions)
	}

	uc.logger.Info("effectiveness summary completed",
		"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"group_by", query.GroupBy, "chains_scanned", len(chains),
		"evaluated", len(attributions), "cohorts", len(cohorts),
		"total_ms", elapsed.Milliseconds(),
	)

	return EffectivenessSummaryReply{
		Cohorts: cohorts,
		Source:  "clickhouse",
		Meta: EffectivenessMeta{
			TotalMs:         elapsed.Milliseconds(),
			EvaluationCount: len(attributions),
			ChainsScanned:   len(chains),
			Excluded:        excluded,
		},
	}, nil
}

// aggregateCohort computes a CohortSummary for a set of attributions.
func aggregateCohort(key string, attrs []effectiveness.Attribution) CohortSummary {
	cs := CohortSummary{Key: key}

	for i := range attrs {
		cs.Evaluated++
		cs.TotalFees += attrs[i].TotalFees

		switch attrs[i].Outcome {
		case effectiveness.OutcomeWin:
			cs.WinCount++
			cs.TotalPnL += attrs[i].NetPnL
		case effectiveness.OutcomeLoss:
			cs.LossCount++
			cs.TotalPnL += attrs[i].NetPnL
		case effectiveness.OutcomeBreakeven:
			cs.BreakevenCount++
			cs.TotalPnL += attrs[i].NetPnL
		case effectiveness.OutcomeUnresolved:
			cs.UnresolvedCount++
		}
	}

	cs.Resolved = cs.WinCount + cs.LossCount + cs.BreakevenCount
	if cs.Resolved > 0 {
		cs.WinRate = float64(cs.WinCount) / float64(cs.Resolved)
		cs.AvgPnL = cs.TotalPnL / float64(cs.Resolved)
	}

	return cs
}

// aggregateByDimension groups attributions by the specified dimension and
// returns a CohortSummary per group, sorted by evaluated count descending.
func aggregateByDimension(dimension string, attrs []effectiveness.Attribution) []CohortSummary {
	groups := make(map[string][]effectiveness.Attribution)

	for i := range attrs {
		key := dimensionValue(dimension, &attrs[i])
		if key == "" {
			key = "(unknown)"
		}
		groups[key] = append(groups[key], attrs[i])
	}

	cohorts := make([]CohortSummary, 0, len(groups))
	for key, group := range groups {
		cohorts = append(cohorts, aggregateCohort(key, group))
	}

	sort.Slice(cohorts, func(i, j int) bool {
		return cohorts[i].Evaluated > cohorts[j].Evaluated
	})

	return cohorts
}

// dimensionValue extracts the grouping key from an attribution.
func dimensionValue(dimension string, attr *effectiveness.Attribution) string {
	switch dimension {
	case "decision_type":
		return attr.DecisionType
	case "strategy_type":
		return attr.StrategyType
	case "severity":
		return attr.DecisionSeverity
	case "source":
		return attr.Source
	default:
		return ""
	}
}

// matchesSummaryFilters checks whether an attribution matches the summary query's pre-aggregation filters.
func matchesSummaryFilters(attr *effectiveness.Attribution, query EffectivenessSummaryQuery) bool {
	if query.DecisionType != "" && attr.DecisionType != query.DecisionType {
		return false
	}
	if query.StrategyType != "" && attr.StrategyType != query.StrategyType {
		return false
	}
	if query.Severity != "" && attr.DecisionSeverity != query.Severity {
		return false
	}
	return true
}
