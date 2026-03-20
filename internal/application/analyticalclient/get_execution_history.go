package analyticalclient

import (
	"context"
	"log/slog"
	"time"

	"internal/domain/execution"
	"internal/shared/problem"
)

// ExecutionReader is the local interface for reading historical executions
// from the analytical store.
type ExecutionReader interface {
	QueryExecutionHistory(ctx context.Context, execType, source, symbol string, timeframe int, side, status string, since, until int64, limit int) ([]execution.ExecutionIntent, error)
}

// GetExecutionHistoryUseCase queries the analytical store for historical executions.
type GetExecutionHistoryUseCase struct {
	reader ExecutionReader
	logger *slog.Logger
}

func NewGetExecutionHistoryUseCase(reader ExecutionReader, logger *slog.Logger) *GetExecutionHistoryUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetExecutionHistoryUseCase{reader: reader, logger: logger.With("component", "analytical_execution_usecase")}
}

func (uc *GetExecutionHistoryUseCase) Execute(ctx context.Context, query ExecutionHistoryQuery) (ExecutionHistoryReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return ExecutionHistoryReply{}, problem.New(problem.Unavailable, "analytical execution reader is unavailable")
	}

	if query.Type == "" {
		return ExecutionHistoryReply{}, problem.New(problem.InvalidArgument, "type is required")
	}
	if query.Source == "" {
		return ExecutionHistoryReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return ExecutionHistoryReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return ExecutionHistoryReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if query.Since < 0 {
		return ExecutionHistoryReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return ExecutionHistoryReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return ExecutionHistoryReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	if query.Limit <= 0 {
		query.Limit = defaultLimit
	}
	if query.Limit > maxLimit {
		query.Limit = maxLimit
	}

	start := time.Now()
	executions, err := uc.reader.QueryExecutionHistory(ctx, query.Type, query.Source, query.Symbol, query.Timeframe, query.Side, query.Status, query.Since, query.Until, query.Limit)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("analytical execution query failed",
			"type", query.Type, "source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
			"side", query.Side, "status", query.Status, "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return ExecutionHistoryReply{}, problem.Wrap(err, problem.Unavailable, "analytical execution query failed")
	}

	if executions == nil {
		executions = []execution.ExecutionIntent{}
	}

	rowCount := len(executions)
	queryMs := elapsed.Milliseconds()

	uc.logger.Info("analytical execution query completed",
		"type", query.Type, "source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"side", query.Side, "status", query.Status, "rows", rowCount, "query_ms", queryMs,
	)

	return ExecutionHistoryReply{
		Executions: executions,
		Source:     "clickhouse",
		Meta: QueryMeta{
			QueryMs:  queryMs,
			RowCount: rowCount,
		},
	}, nil
}
