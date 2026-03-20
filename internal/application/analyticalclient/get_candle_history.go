package analyticalclient

import (
	"context"
	"log/slog"
	"time"

	"internal/domain/evidence"
	"internal/shared/problem"
)

const (
	defaultLimit = 50
	maxLimit     = 500
)

// CandleReader is the local interface for reading historical candles
// from the analytical store.
type CandleReader interface {
	QueryCandleHistory(ctx context.Context, source, symbol string, timeframe int, since, until int64, limit int) ([]evidence.EvidenceCandle, error)
}

// GetCandleHistoryUseCase queries the analytical store for historical candles.
type GetCandleHistoryUseCase struct {
	reader CandleReader
	logger *slog.Logger
}

func NewGetCandleHistoryUseCase(reader CandleReader, logger *slog.Logger) *GetCandleHistoryUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetCandleHistoryUseCase{reader: reader, logger: logger.With("component", "analytical_usecase")}
}

func (uc *GetCandleHistoryUseCase) Execute(ctx context.Context, query CandleHistoryQuery) (CandleHistoryReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return CandleHistoryReply{}, problem.New(problem.Unavailable, "analytical candle reader is unavailable")
	}

	if query.Source == "" {
		return CandleHistoryReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return CandleHistoryReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return CandleHistoryReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if query.Since < 0 {
		return CandleHistoryReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return CandleHistoryReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return CandleHistoryReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	if query.Limit <= 0 {
		query.Limit = defaultLimit
	}
	if query.Limit > maxLimit {
		query.Limit = maxLimit
	}

	start := time.Now()
	candles, err := uc.reader.QueryCandleHistory(ctx, query.Source, query.Symbol, query.Timeframe, query.Since, query.Until, query.Limit)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("analytical query failed",
			"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return CandleHistoryReply{}, problem.Wrap(err, problem.Unavailable, "analytical query failed")
	}

	if candles == nil {
		candles = []evidence.EvidenceCandle{}
	}

	rowCount := len(candles)
	queryMs := elapsed.Milliseconds()

	uc.logger.Info("analytical query completed",
		"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"rows", rowCount, "query_ms", queryMs,
	)

	return CandleHistoryReply{
		Candles: candles,
		Source:  "clickhouse",
		Meta: QueryMeta{
			QueryMs:  queryMs,
			RowCount: rowCount,
		},
	}, nil
}
