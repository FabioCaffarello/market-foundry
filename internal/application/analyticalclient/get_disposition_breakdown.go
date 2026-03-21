package analyticalclient

import (
	"context"
	"log/slog"
	"time"

	"internal/shared/problem"
)

// GetDispositionBreakdownUseCase queries the risk_assessments table for a count
// of each disposition (approved/modified/rejected). This answers Q6 (how many
// executions were blocked vs approved in period T?).
type GetDispositionBreakdownUseCase struct {
	reader AggregationReader
	logger *slog.Logger
}

func NewGetDispositionBreakdownUseCase(reader AggregationReader, logger *slog.Logger) *GetDispositionBreakdownUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetDispositionBreakdownUseCase{reader: reader, logger: logger.With("component", "analytical_disposition_usecase")}
}

func (uc *GetDispositionBreakdownUseCase) Execute(ctx context.Context, query DispositionBreakdownQuery) (DispositionBreakdownReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return DispositionBreakdownReply{}, problem.New(problem.Unavailable, "disposition breakdown reader is unavailable")
	}
	if query.Type == "" {
		return DispositionBreakdownReply{}, problem.New(problem.InvalidArgument, "type is required")
	}
	if query.Source == "" {
		return DispositionBreakdownReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return DispositionBreakdownReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return DispositionBreakdownReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if query.Since < 0 {
		return DispositionBreakdownReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return DispositionBreakdownReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return DispositionBreakdownReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	start := time.Now()
	dispositions, err := uc.reader.QueryDispositionBreakdown(ctx, query.Type, query.Source, query.Symbol, query.Timeframe, query.Since, query.Until)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("disposition breakdown query failed",
			"type", query.Type, "source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return DispositionBreakdownReply{}, problem.Wrap(err, problem.Unavailable, "disposition breakdown query failed")
	}

	if dispositions == nil {
		dispositions = []DispositionCount{}
	}

	var total int64
	for _, d := range dispositions {
		total += d.Count
	}
	if total > 0 {
		for i := range dispositions {
			dispositions[i].Percentage = float64(dispositions[i].Count) * 100.0 / float64(total)
		}
	}

	uc.logger.Info("disposition breakdown query completed",
		"type", query.Type, "source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"total", total, "total_ms", elapsed.Milliseconds(),
	)

	return DispositionBreakdownReply{
		Dispositions: dispositions,
		Total:        total,
		Source:       "clickhouse",
		Meta: CompositeQueryMeta{
			TotalMs:    elapsed.Milliseconds(),
			ChainCount: len(dispositions),
		},
	}, nil
}
