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
	pairingDefaultLimit = 50
	pairingMaxLimit     = 200
)

// GetPairingUseCase computes the round-trip pairing read model from existing
// execution chain data. It fetches composite chains via CompositeReader,
// converts execution intents to legs, applies FIFO matching, and produces
// paired/unmatched round-trips with effectiveness attribution.
//
// S481: This is a read-path computation — no new write-side changes or
// ClickHouse tables required. Pairing is derived from existing fill data
// using the canonical model from S480 and effectiveness classification from S476.
//
// Guard rails:
//   - No OMS expansion; pairing is a read-path classification.
//   - No new ClickHouse tables; computed from existing execution data.
//   - No write-path changes.
//   - Additive only; zero changes to existing behavior.
type GetPairingUseCase struct {
	reader CompositeReader
	logger *slog.Logger
}

func NewGetPairingUseCase(reader CompositeReader, logger *slog.Logger) *GetPairingUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetPairingUseCase{reader: reader, logger: logger.With("component", "pairing_usecase")}
}

func (uc *GetPairingUseCase) Execute(ctx context.Context, query PairingQuery) (PairingReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return PairingReply{}, problem.New(problem.Unavailable, "pairing reader is unavailable")
	}

	start := time.Now()

	if query.CorrelationID != "" {
		return uc.executeSingle(ctx, query, start)
	}
	return uc.executeBatch(ctx, query, start)
}

func (uc *GetPairingUseCase) executeSingle(ctx context.Context, query PairingQuery, start time.Time) (PairingReply, *problem.Problem) {
	if query.Symbol == "" {
		return PairingReply{}, problem.New(problem.InvalidArgument, "symbol is required for single-chain pairing lookup (S301 isolation)")
	}

	chain, err := uc.reader.QueryChainByCorrelationID(ctx, query.CorrelationID, query.Symbol)
	elapsed := time.Since(start)
	if err != nil {
		uc.logger.Warn("pairing single query failed",
			"correlation_id", query.CorrelationID, "symbol", query.Symbol,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return PairingReply{}, problem.Wrap(err, problem.Unavailable, "pairing query failed")
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
	views := uc.buildViews(roundTrips, nil, query)

	summary := buildPairingSummary(roundTrips, views)

	return PairingReply{
		RoundTrips: views,
		Summary:    summary,
		Source:     "clickhouse",
		Meta: PairingMeta{
			TotalMs:       elapsed.Milliseconds(),
			ChainsScanned: chainsScanned,
			LegsProduced:  len(legs),
			RoundTrips:    len(views),
		},
	}, nil
}

func (uc *GetPairingUseCase) executeBatch(ctx context.Context, query PairingQuery, start time.Time) (PairingReply, *problem.Problem) {
	if query.Source == "" {
		return PairingReply{}, problem.New(problem.InvalidArgument, "source is required for batch pairing")
	}
	if query.Symbol == "" {
		return PairingReply{}, problem.New(problem.InvalidArgument, "symbol is required for batch pairing")
	}
	if query.Timeframe <= 0 {
		return PairingReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive for batch pairing")
	}

	if query.Limit <= 0 {
		query.Limit = pairingDefaultLimit
	}
	if query.Limit > pairingMaxLimit {
		query.Limit = pairingMaxLimit
	}

	chains, err := uc.reader.QueryChainsBatch(ctx, query.Source, query.Symbol, query.Timeframe, query.Since, query.Until, query.Limit)
	elapsed := time.Since(start)
	if err != nil {
		uc.logger.Warn("pairing batch query failed",
			"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return PairingReply{}, problem.Wrap(err, problem.Unavailable, "pairing batch query failed")
	}

	// Convert all chains to legs, collecting per-chain data for attribution.
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
	views := uc.buildViews(roundTrips, chainMap, query)

	summary := buildPairingSummary(roundTrips, views)

	uc.logger.Info("pairing batch completed",
		"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"chains_scanned", len(chains), "legs_produced", len(legs),
		"round_trips", len(views), "paired", summary.PairedCount,
		"resolved_rate", summary.ResolvedRate,
		"total_ms", elapsed.Milliseconds(),
	)

	return PairingReply{
		RoundTrips: views,
		Summary:    summary,
		Source:     "clickhouse",
		Meta: PairingMeta{
			TotalMs:       elapsed.Milliseconds(),
			ChainsScanned: len(chains),
			LegsProduced:  len(legs),
			RoundTrips:    len(views),
		},
	}, nil
}

// buildViews converts round-trips to views with optional attribution,
// applying state and side filters from the query.
func (uc *GetPairingUseCase) buildViews(roundTrips []pairing.RoundTrip, chainMap map[string]*CompositeExecutionChain, query PairingQuery) []RoundTripView {
	var views []RoundTripView

	for _, rt := range roundTrips {
		if !matchesPairingFilters(rt, query) {
			continue
		}

		view := RoundTripView{RoundTrip: rt}

		// Compute attribution for paired round-trips.
		if rt.IsPaired() && rt.Entry != nil && rt.Exit != nil {
			view.Attribution = classifyPairedRoundTrip(rt, chainMap)
		}

		views = append(views, view)
	}

	return views
}

// classifyPairedRoundTrip computes effectiveness attribution for a paired
// round-trip using the entry and exit legs' source chains.
func classifyPairedRoundTrip(rt pairing.RoundTrip, chainMap map[string]*CompositeExecutionChain) *effectiveness.Attribution {
	if chainMap == nil {
		return nil
	}

	entryChain := chainMap[rt.Entry.CorrelationID]
	exitChain := chainMap[rt.Exit.CorrelationID]

	if entryChain == nil || exitChain == nil {
		return nil
	}
	if entryChain.Execution == nil || exitChain.Execution == nil {
		return nil
	}

	attr := effectiveness.ClassifyPair(
		entryChain.Execution.ExecutionIntent,
		exitChain.Execution.ExecutionIntent,
	)
	if attr != nil {
		enrichFromChain(attr, entryChain)
	}
	return attr
}

// matchesPairingFilters applies post-computation filters to round-trips.
func matchesPairingFilters(rt pairing.RoundTrip, query PairingQuery) bool {
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

// inferStrategyDirection extracts strategy direction from a composite chain.
// Defaults to "long" when strategy stage is absent or direction is empty.
func inferStrategyDirection(chain *CompositeExecutionChain) string {
	if chain.Strategy != nil && chain.Strategy.Direction != "" {
		return string(chain.Strategy.Direction)
	}
	return "long"
}

// buildPairingSummary computes aggregated statistics from round-trips and views.
func buildPairingSummary(roundTrips []pairing.RoundTrip, views []RoundTripView) PairingSummary {
	pr := pairing.Summarize(roundTrips)

	summary := PairingSummary{
		TotalEntries:     pr.TotalEntries,
		TotalExits:       pr.TotalExits,
		PairedCount:      pr.PairedCount,
		UnmatchedEntries: pr.UnmatchedEntries,
		UnmatchedExits:   pr.UnmatchedExits,
		ResolvedRate:     pr.ResolvedRate,
	}

	// Aggregate effectiveness from paired views.
	for _, v := range views {
		if v.Attribution == nil {
			continue
		}
		switch v.Attribution.Outcome {
		case effectiveness.OutcomeWin:
			summary.WinCount++
			summary.TotalPnL += v.Attribution.NetPnL
		case effectiveness.OutcomeLoss:
			summary.LossCount++
			summary.TotalPnL += v.Attribution.NetPnL
		case effectiveness.OutcomeBreakeven:
			summary.BreakevenCount++
			summary.TotalPnL += v.Attribution.NetPnL
		}
		summary.TotalFees += v.Attribution.TotalFees
	}

	return summary
}
