package analyticalclient

import (
	"context"
	"log/slog"
	"time"

	"internal/domain/effectiveness"
	"internal/domain/pairing"
	"internal/shared/problem"
)

const (
	effectivenessDefaultLimit = 20
	effectivenessMaxLimit     = 100
)

// GetEffectivenessUseCase computes effectiveness attribution for decision chains
// by reusing the CompositeReader to fetch full causal chains, then classifying each
// chain's execution outcome using the effectiveness domain model.
//
// S476: This is a read-path computation — no new write-side changes or ClickHouse
// tables required. Effectiveness is derived from existing fill data.
type GetEffectivenessUseCase struct {
	reader CompositeReader
	logger *slog.Logger
}

func NewGetEffectivenessUseCase(reader CompositeReader, logger *slog.Logger) *GetEffectivenessUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetEffectivenessUseCase{reader: reader, logger: logger.With("component", "effectiveness_usecase")}
}

func (uc *GetEffectivenessUseCase) Execute(ctx context.Context, query EffectivenessQuery) (EffectivenessReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return EffectivenessReply{}, problem.New(problem.Unavailable, "effectiveness reader is unavailable")
	}

	start := time.Now()

	if query.CorrelationID != "" {
		return uc.executeSingle(ctx, query, start)
	}
	return uc.executeBatch(ctx, query, start)
}

func (uc *GetEffectivenessUseCase) executeSingle(ctx context.Context, query EffectivenessQuery, start time.Time) (EffectivenessReply, *problem.Problem) {
	if query.Instrument.IsZero() {
		return EffectivenessReply{}, problem.New(problem.InvalidArgument, "symbol is required for single-chain effectiveness lookup (S301 isolation)")
	}

	chain, err := uc.reader.QueryChainByCorrelationID(ctx, query.CorrelationID, query.Instrument)
	elapsed := time.Since(start)
	if err != nil {
		uc.logger.Warn("effectiveness single query failed",
			"correlation_id", query.CorrelationID, "instrument", query.Instrument.Symbol(),
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return EffectivenessReply{}, problem.Wrap(err, problem.Unavailable, "effectiveness query failed")
	}

	var evaluations []effectiveness.Attribution
	var excluded int
	if chain != nil && chain.Execution != nil {
		attr := effectiveness.Classify(chain.Execution.ExecutionIntent)
		if attr != nil {
			enrichFromChain(attr, chain)
			if matchesFilters(attr, query) {
				evaluations = append(evaluations, *attr)
			}
		} else {
			excluded++
		}
	}

	return EffectivenessReply{
		Evaluations: evaluations,
		Source:      "clickhouse",
		Meta: EffectivenessMeta{
			TotalMs:         elapsed.Milliseconds(),
			EvaluationCount: len(evaluations),
			ChainsScanned:   1,
			Excluded:        excluded,
		},
	}, nil
}

func (uc *GetEffectivenessUseCase) executeBatch(ctx context.Context, query EffectivenessQuery, start time.Time) (EffectivenessReply, *problem.Problem) {
	if query.Source == "" {
		return EffectivenessReply{}, problem.New(problem.InvalidArgument, "source is required for batch effectiveness evaluation")
	}
	if query.Instrument.IsZero() {
		return EffectivenessReply{}, problem.New(problem.InvalidArgument, "symbol is required for batch effectiveness evaluation")
	}
	if query.Timeframe <= 0 {
		return EffectivenessReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive for batch effectiveness evaluation")
	}

	if query.Limit <= 0 {
		query.Limit = effectivenessDefaultLimit
	}
	if query.Limit > effectivenessMaxLimit {
		query.Limit = effectivenessMaxLimit
	}

	// Fetch more chains than requested to account for post-filter exclusions.
	fetchLimit := query.Limit
	if hasPostFilter(query) {
		fetchLimit = query.Limit * 3
		if fetchLimit > compositeMaxLimit {
			fetchLimit = compositeMaxLimit
		}
	}

	chains, err := uc.reader.QueryChainsBatch(ctx, query.Source, query.Instrument, query.Timeframe, query.Since, query.Until, fetchLimit)
	elapsed := time.Since(start)
	if err != nil {
		uc.logger.Warn("effectiveness batch query failed",
			"source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return EffectivenessReply{}, problem.Wrap(err, problem.Unavailable, "effectiveness batch query failed")
	}

	// S481: Build legs and chain index for round-trip pairing integration.
	// Paired round-trips use ClassifyPair for realized P&L attribution;
	// unpaired legs fall back to single-leg Classify (which returns unresolved).
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

	// Run FIFO matching to find paired round-trips.
	roundTrips := pairing.MatchFIFO(legs, pairing.DefaultMatchingConfig())

	// Build a set of correlation IDs that were paired (to avoid double-counting).
	pairedCorrs := make(map[string]bool)

	var evaluations []effectiveness.Attribution

	// First pass: classify paired round-trips using ClassifyPair.
	for _, rt := range roundTrips {
		if !rt.IsPaired() || rt.Entry == nil || rt.Exit == nil {
			continue
		}

		entryChain := chainByCorr[rt.Entry.CorrelationID]
		exitChain := chainByCorr[rt.Exit.CorrelationID]
		if entryChain == nil || exitChain == nil || entryChain.Execution == nil || exitChain.Execution == nil {
			continue
		}

		attr := effectiveness.ClassifyPair(
			entryChain.Execution.ExecutionIntent,
			exitChain.Execution.ExecutionIntent,
		)
		if attr == nil {
			continue
		}

		enrichFromChain(attr, entryChain)
		pairedCorrs[rt.Entry.CorrelationID] = true
		pairedCorrs[rt.Exit.CorrelationID] = true

		if !matchesFilters(attr, query) {
			continue
		}

		evaluations = append(evaluations, *attr)
		if len(evaluations) >= query.Limit {
			break
		}
	}

	// Second pass: classify unpaired chains using single-leg Classify.
	if len(evaluations) < query.Limit {
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

			if !matchesFilters(attr, query) {
				continue
			}

			evaluations = append(evaluations, *attr)
			if len(evaluations) >= query.Limit {
				break
			}
		}
	}

	pairedCount := len(pairedCorrs) / 2 // each pair contributes 2 correlation IDs

	uc.logger.Info("effectiveness batch completed",
		"source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
		"chains_scanned", len(chains), "evaluations", len(evaluations), "excluded", excluded,
		"paired_round_trips", pairedCount,
		"total_ms", elapsed.Milliseconds(),
	)

	return EffectivenessReply{
		Evaluations: evaluations,
		Source:      "clickhouse",
		Meta: EffectivenessMeta{
			TotalMs:         elapsed.Milliseconds(),
			EvaluationCount: len(evaluations),
			ChainsScanned:   len(chains),
			Excluded:        excluded,
		},
	}, nil
}

// enrichFromChain populates attribution fields from the composite chain's decision
// and strategy stages when available. This enriches the execution-derived attribution
// with upstream context that may not be present in execution.RiskInput alone.
func enrichFromChain(attr *effectiveness.Attribution, chain *CompositeExecutionChain) {
	if chain.Decision != nil {
		if attr.DecisionType == "" {
			attr.DecisionType = chain.Decision.Type
		}
		if attr.DecisionSeverity == "" {
			attr.DecisionSeverity = string(chain.Decision.Severity)
		}
	}
	if chain.Strategy != nil {
		if attr.StrategyType == "" {
			attr.StrategyType = chain.Strategy.Type
		}
	}
}

// matchesFilters checks whether an attribution matches the query's post-filters.
func matchesFilters(attr *effectiveness.Attribution, query EffectivenessQuery) bool {
	if query.DecisionType != "" && attr.DecisionType != query.DecisionType {
		return false
	}
	if query.StrategyType != "" && attr.StrategyType != query.StrategyType {
		return false
	}
	if query.Severity != "" && attr.DecisionSeverity != query.Severity {
		return false
	}
	if query.Effectiveness != "" && string(attr.Outcome) != query.Effectiveness {
		return false
	}
	return true
}

// hasPostFilter returns true if the query has any post-fetch filter that could exclude results.
func hasPostFilter(query EffectivenessQuery) bool {
	return query.DecisionType != "" || query.StrategyType != "" ||
		query.Severity != "" || query.Effectiveness != ""
}
