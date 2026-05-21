package analyticalclient

import (
	"context"
	"log/slog"
	"time"

	"internal/domain/execution"
	"internal/shared/problem"
)

// ExecutionListReader is the local interface for reading executions with relaxed filters.
//
// S454A: Unlike ExecutionReader (which requires type+source+symbol+timeframe),
// this reader accepts any combination of filters with at least one required.
type ExecutionListReader interface {
	QueryExecutionList(ctx context.Context, execType, source, symbol string, timeframe int, side, status string, since, until int64, limit int) ([]execution.ExecutionIntent, error)
}

// GetExecutionListUseCase queries the analytical store for executions with relaxed filters.
type GetExecutionListUseCase struct {
	reader ExecutionListReader
	logger *slog.Logger
}

func NewGetExecutionListUseCase(reader ExecutionListReader, logger *slog.Logger) *GetExecutionListUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetExecutionListUseCase{reader: reader, logger: logger.With("component", "analytical_execution_list_usecase")}
}

func (uc *GetExecutionListUseCase) Execute(ctx context.Context, query ExecutionListQuery) (ExecutionListReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return ExecutionListReply{}, problem.New(problem.Unavailable, "analytical execution list reader is unavailable")
	}

	// At least one filter must be provided.
	hasFilter := query.Type != "" || query.Source != "" || query.Symbol != "" || query.Timeframe > 0 ||
		query.Side != "" || query.Status != "" || query.Since > 0 || query.Until > 0
	if !hasFilter {
		return ExecutionListReply{}, problem.New(problem.InvalidArgument, "at least one filter is required (type, source, symbol, timeframe, side, status, since, or until)")
	}

	if query.Since < 0 {
		return ExecutionListReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return ExecutionListReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return ExecutionListReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	if query.Limit <= 0 {
		query.Limit = defaultLimit
	}
	if query.Limit > maxLimit {
		query.Limit = maxLimit
	}

	start := time.Now()
	intents, err := uc.reader.QueryExecutionList(ctx, query.Type, query.Source, query.Symbol, query.Timeframe, query.Side, query.Status, query.Since, query.Until, query.Limit)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("analytical execution list query failed",
			"type", query.Type, "source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
			"side", query.Side, "status", query.Status, "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return ExecutionListReply{}, problem.Wrap(err, problem.Unavailable, "analytical execution list query failed")
	}

	entries := make([]LifecycleHistoryEntry, 0, len(intents))
	for _, intent := range intents {
		entries = append(entries, intentToLifecycleEntry(intent))
	}

	rowCount := len(entries)
	queryMs := elapsed.Milliseconds()

	uc.logger.Info("analytical execution list query completed",
		"type", query.Type, "source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"side", query.Side, "status", query.Status, "rows", rowCount, "query_ms", queryMs,
	)

	return ExecutionListReply{
		Entries: entries,
		Source:  "clickhouse",
		Meta: QueryMeta{
			QueryMs:  queryMs,
			RowCount: rowCount,
		},
	}, nil
}
