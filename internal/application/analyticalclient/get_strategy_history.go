package analyticalclient

import (
	"internal/domain/instrument"

	"context"
	"log/slog"
	"time"

	"internal/domain/strategy"
	"internal/shared/problem"
)

// StrategyReader is the local interface for reading historical strategies
// from the analytical store.
type StrategyReader interface {
	QueryStrategyHistory(ctx context.Context, strategyType, source string, inst instrument.CanonicalInstrument, timeframe int, direction string, since, until int64, limit int) ([]strategy.Strategy, error)
}

// GetStrategyHistoryUseCase queries the analytical store for historical strategies.
type GetStrategyHistoryUseCase struct {
	reader StrategyReader
	logger *slog.Logger
}

func NewGetStrategyHistoryUseCase(reader StrategyReader, logger *slog.Logger) *GetStrategyHistoryUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetStrategyHistoryUseCase{reader: reader, logger: logger.With("component", "analytical_strategy_usecase")}
}

func (uc *GetStrategyHistoryUseCase) Execute(ctx context.Context, query StrategyHistoryQuery) (StrategyHistoryReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return StrategyHistoryReply{}, problem.New(problem.Unavailable, "analytical strategy reader is unavailable")
	}

	if query.Type == "" {
		return StrategyHistoryReply{}, problem.New(problem.InvalidArgument, "type is required")
	}
	if query.Source == "" {
		return StrategyHistoryReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Instrument.IsZero() {
		return StrategyHistoryReply{}, problem.New(problem.InvalidArgument, "instrument is required")
	}
	if query.Timeframe <= 0 {
		return StrategyHistoryReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if query.Since < 0 {
		return StrategyHistoryReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return StrategyHistoryReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return StrategyHistoryReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	if query.Limit <= 0 {
		query.Limit = defaultLimit
	}
	if query.Limit > maxLimit {
		query.Limit = maxLimit
	}

	start := time.Now()
	strategies, err := uc.reader.QueryStrategyHistory(ctx, query.Type, query.Source, query.Instrument, query.Timeframe, query.Direction, query.Since, query.Until, query.Limit)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("analytical strategy query failed",
			"type", query.Type, "source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
			"direction", query.Direction, "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return StrategyHistoryReply{}, problem.Wrap(err, problem.Unavailable, "analytical strategy query failed")
	}

	if strategies == nil {
		strategies = []strategy.Strategy{}
	}

	rowCount := len(strategies)
	queryMs := elapsed.Milliseconds()

	uc.logger.Info("analytical strategy query completed",
		"type", query.Type, "source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
		"direction", query.Direction, "rows", rowCount, "query_ms", queryMs,
	)

	return StrategyHistoryReply{
		Strategies: strategies,
		Source:     "clickhouse",
		Meta: QueryMeta{
			QueryMs:  queryMs,
			RowCount: rowCount,
		},
	}, nil
}
