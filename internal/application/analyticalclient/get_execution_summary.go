package analyticalclient

import (
	"context"
	"log/slog"
	"time"

	"internal/shared/problem"
)

// ExecutionSummaryRawRow represents a raw grouped result from the analytical store.
type ExecutionSummaryRawRow struct {
	Type     string
	Status   string
	Count    int64
	LatestAt time.Time
}

// ExecutionSummaryReader is the local interface for reading execution summary
// (counts by type/status) from the analytical store.
type ExecutionSummaryReader interface {
	QueryExecutionSummary(ctx context.Context, source, symbol string, timeframe int, since, until int64) ([]ExecutionSummaryRawRow, error)
}

// GetExecutionSummaryUseCase queries the analytical store for execution counts
// grouped by (type, status).
type GetExecutionSummaryUseCase struct {
	reader ExecutionSummaryReader
	logger *slog.Logger
}

func NewGetExecutionSummaryUseCase(reader ExecutionSummaryReader, logger *slog.Logger) *GetExecutionSummaryUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetExecutionSummaryUseCase{reader: reader, logger: logger.With("component", "analytical_execution_summary_usecase")}
}

func (uc *GetExecutionSummaryUseCase) Execute(ctx context.Context, query ExecutionSummaryQuery) (ExecutionSummaryReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return ExecutionSummaryReply{}, problem.New(problem.Unavailable, "analytical execution summary reader is unavailable")
	}

	hasFilter := query.Source != "" || query.Symbol != "" || query.Timeframe > 0 || query.Since > 0 || query.Until > 0
	if !hasFilter {
		return ExecutionSummaryReply{}, problem.New(problem.InvalidArgument, "at least one filter is required (source, symbol, timeframe, since, or until)")
	}

	if query.Since < 0 {
		return ExecutionSummaryReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return ExecutionSummaryReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return ExecutionSummaryReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	start := time.Now()
	rows, err := uc.reader.QueryExecutionSummary(ctx, query.Source, query.Symbol, query.Timeframe, query.Since, query.Until)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("analytical execution summary query failed",
			"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return ExecutionSummaryReply{}, problem.Wrap(err, problem.Unavailable, "analytical execution summary query failed")
	}

	entries := make([]ExecutionSummaryEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, ExecutionSummaryEntry{
			Type:     r.Type,
			Status:   r.Status,
			Count:    r.Count,
			LatestAt: r.LatestAt.UTC().Format(time.RFC3339),
		})
	}

	queryMs := elapsed.Milliseconds()

	uc.logger.Info("analytical execution summary query completed",
		"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"groups", len(entries), "query_ms", queryMs,
	)

	return ExecutionSummaryReply{
		Entries: entries,
		Source:  "clickhouse",
		Meta: QueryMeta{
			QueryMs:  queryMs,
			RowCount: len(entries),
		},
	}, nil
}
