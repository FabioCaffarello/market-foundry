package analyticalclient

import (
	"internal/domain/instrument"

	"context"
	"log/slog"
	"time"

	"internal/shared/problem"
)

const (
	compositeDefaultLimit = 20
	compositeMaxLimit     = 100
)

// CompositeReader is the local interface for reading composite execution chains
// from the analytical store. Implemented by clickhouse.CompositeReader.
type CompositeReader interface {
	QueryChainByCorrelationID(ctx context.Context, correlationID string, inst instrument.CanonicalInstrument) (*CompositeExecutionChain, error)
	QueryChainsBatch(ctx context.Context, source string, inst instrument.CanonicalInstrument, timeframe int, since, until int64, limit int) ([]CompositeExecutionChain, error)
}

// GetCompositeChainUseCase queries the analytical store for composite execution chains.
// It supports two modes: single-chain lookup (by correlation_id) and batch lookup
// (by source/symbol/timeframe with optional time range).
type GetCompositeChainUseCase struct {
	reader CompositeReader
	logger *slog.Logger
}

func NewGetCompositeChainUseCase(reader CompositeReader, logger *slog.Logger) *GetCompositeChainUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetCompositeChainUseCase{reader: reader, logger: logger.With("component", "analytical_composite_usecase")}
}

func (uc *GetCompositeChainUseCase) Execute(ctx context.Context, query CompositeChainQuery) (CompositeChainReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return CompositeChainReply{}, problem.New(problem.Unavailable, "composite chain reader is unavailable")
	}

	start := time.Now()

	// Determine mode: single vs batch.
	if query.CorrelationID != "" {
		if query.Instrument.IsZero() {
			return CompositeChainReply{}, problem.New(problem.InvalidArgument, "symbol is required for single-chain lookup (S301 isolation)")
		}
		return uc.executeSingle(ctx, query.CorrelationID, query.Instrument, start)
	}

	return uc.executeBatch(ctx, query, start)
}

func (uc *GetCompositeChainUseCase) executeSingle(ctx context.Context, correlationID string, inst instrument.CanonicalInstrument, start time.Time) (CompositeChainReply, *problem.Problem) {
	chain, err := uc.reader.QueryChainByCorrelationID(ctx, correlationID, inst)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("composite chain query failed",
			"correlation_id", correlationID, "instrument", inst.Symbol(), "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return CompositeChainReply{}, problem.Wrap(err, problem.Unavailable, "composite chain query failed")
	}

	chains := []CompositeExecutionChain{}
	if chain != nil && chain.StageCount > 0 {
		computeAttribution(chain)
		chains = append(chains, *chain)
	}

	uc.logger.Info("composite chain query completed",
		"correlation_id", correlationID, "instrument", inst.Symbol(), "chains", len(chains), "total_ms", elapsed.Milliseconds(),
	)

	return CompositeChainReply{
		Chains: chains,
		Source: "clickhouse",
		Meta: CompositeQueryMeta{
			TotalMs:    elapsed.Milliseconds(),
			ChainCount: len(chains),
		},
	}, nil
}

func (uc *GetCompositeChainUseCase) executeBatch(ctx context.Context, query CompositeChainQuery, start time.Time) (CompositeChainReply, *problem.Problem) {
	if query.Source == "" {
		return CompositeChainReply{}, problem.New(problem.InvalidArgument, "source is required for batch lookup")
	}
	if query.Instrument.IsZero() {
		return CompositeChainReply{}, problem.New(problem.InvalidArgument, "symbol is required for batch lookup")
	}
	if query.Timeframe <= 0 {
		return CompositeChainReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive for batch lookup")
	}
	if query.Since < 0 {
		return CompositeChainReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return CompositeChainReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return CompositeChainReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	if query.Limit <= 0 {
		query.Limit = compositeDefaultLimit
	}
	if query.Limit > compositeMaxLimit {
		query.Limit = compositeMaxLimit
	}

	chains, err := uc.reader.QueryChainsBatch(ctx, query.Source, query.Instrument, query.Timeframe, query.Since, query.Until, query.Limit)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("composite batch query failed",
			"source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return CompositeChainReply{}, problem.Wrap(err, problem.Unavailable, "composite batch query failed")
	}

	if chains == nil {
		chains = []CompositeExecutionChain{}
	}
	for i := range chains {
		computeAttribution(&chains[i])
	}

	uc.logger.Info("composite batch query completed",
		"source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
		"chains", len(chains), "total_ms", elapsed.Milliseconds(),
	)

	return CompositeChainReply{
		Chains: chains,
		Source: "clickhouse",
		Meta: CompositeQueryMeta{
			TotalMs:    elapsed.Milliseconds(),
			ChainCount: len(chains),
		},
	}, nil
}

// computeAttribution derives a RiskAttribution from the risk stage of a chain.
// This is a read-side projection — no write-side changes required. The attribution
// surfaces the risk gate outcome at the chain level for Q2 explainability.
func computeAttribution(chain *CompositeExecutionChain) {
	if chain.Risk == nil {
		return
	}
	r := chain.Risk.RiskAssessment
	attr := &RiskAttribution{
		Disposition:       string(r.Disposition),
		Rationale:         r.Rationale,
		ActiveConstraints: r.Constraints,
	}
	for _, si := range r.Strategies {
		attr.StrategyContext = append(attr.StrategyContext, AttributionStrategyContext{
			Type:              si.Type,
			Direction:         si.Direction,
			Confidence:        si.Confidence,
			DecisionSeverity:  si.DecisionSeverity,
			DecisionRationale: si.DecisionRationale,
		})
	}
	chain.Attribution = attr
}
