package analyticalclient

import (
	"context"
	"log/slog"
	"time"

	"internal/domain/execution"
	"internal/shared/problem"
)

// LifecycleHistoryReader is the local interface for reading historical lifecycle
// events from the analytical store. Unlike ExecutionReader, it queries across
// all execution event types for a given source/symbol/timeframe.
//
// S453A: This interface enables the historical lifecycle read model without
// coupling the use case to the ClickHouse adapter directly.
type LifecycleHistoryReader interface {
	QueryLifecycleHistory(ctx context.Context, source, symbol string, timeframe int, side, status string, since, until int64, limit int) ([]execution.ExecutionIntent, error)
}

// GetLifecycleHistoryUseCase queries the analytical store for the historical
// lifecycle timeline of a given source/symbol/timeframe.
type GetLifecycleHistoryUseCase struct {
	reader LifecycleHistoryReader
	logger *slog.Logger
}

func NewGetLifecycleHistoryUseCase(reader LifecycleHistoryReader, logger *slog.Logger) *GetLifecycleHistoryUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetLifecycleHistoryUseCase{reader: reader, logger: logger.With("component", "analytical_lifecycle_usecase")}
}

func (uc *GetLifecycleHistoryUseCase) Execute(ctx context.Context, query LifecycleHistoryQuery) (LifecycleHistoryReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return LifecycleHistoryReply{}, problem.New(problem.Unavailable, "analytical lifecycle reader is unavailable")
	}

	if query.Source == "" {
		return LifecycleHistoryReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return LifecycleHistoryReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return LifecycleHistoryReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if query.Since < 0 {
		return LifecycleHistoryReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return LifecycleHistoryReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return LifecycleHistoryReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	if query.Limit <= 0 {
		query.Limit = defaultLimit
	}
	if query.Limit > maxLimit {
		query.Limit = maxLimit
	}

	start := time.Now()
	intents, err := uc.reader.QueryLifecycleHistory(ctx, query.Source, query.Symbol, query.Timeframe, query.Side, query.Status, query.Since, query.Until, query.Limit)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("analytical lifecycle query failed",
			"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
			"side", query.Side, "status", query.Status, "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return LifecycleHistoryReply{}, problem.Wrap(err, problem.Unavailable, "analytical lifecycle query failed")
	}

	entries := make([]LifecycleHistoryEntry, 0, len(intents))
	for _, intent := range intents {
		entries = append(entries, intentToLifecycleEntry(intent))
	}

	rowCount := len(entries)
	queryMs := elapsed.Milliseconds()

	uc.logger.Info("analytical lifecycle query completed",
		"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"side", query.Side, "status", query.Status, "rows", rowCount, "query_ms", queryMs,
	)

	return LifecycleHistoryReply{
		Entries: entries,
		Source:  "clickhouse",
		Meta: QueryMeta{
			QueryMs:  queryMs,
			RowCount: rowCount,
		},
	}, nil
}
