package analyticalclient

import (
	"context"
	"log/slog"
	"time"

	"internal/domain/signal"
	"internal/shared/problem"
)

// SignalReader is the local interface for reading historical signals
// from the analytical store.
type SignalReader interface {
	QuerySignalHistory(ctx context.Context, signalType, source, symbol string, timeframe int, since, until int64, limit int) ([]signal.Signal, error)
}

// GetSignalHistoryUseCase queries the analytical store for historical signals.
type GetSignalHistoryUseCase struct {
	reader SignalReader
	logger *slog.Logger
}

func NewGetSignalHistoryUseCase(reader SignalReader, logger *slog.Logger) *GetSignalHistoryUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetSignalHistoryUseCase{reader: reader, logger: logger.With("component", "analytical_signal_usecase")}
}

func (uc *GetSignalHistoryUseCase) Execute(ctx context.Context, query SignalHistoryQuery) (SignalHistoryReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return SignalHistoryReply{}, problem.New(problem.Unavailable, "analytical signal reader is unavailable")
	}

	if query.Type == "" {
		return SignalHistoryReply{}, problem.New(problem.InvalidArgument, "type is required")
	}
	if query.Source == "" {
		return SignalHistoryReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return SignalHistoryReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return SignalHistoryReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if query.Since < 0 {
		return SignalHistoryReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return SignalHistoryReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return SignalHistoryReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	if query.Limit <= 0 {
		query.Limit = defaultLimit
	}
	if query.Limit > maxLimit {
		query.Limit = maxLimit
	}

	start := time.Now()
	signals, err := uc.reader.QuerySignalHistory(ctx, query.Type, query.Source, query.Symbol, query.Timeframe, query.Since, query.Until, query.Limit)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("analytical signal query failed",
			"type", query.Type, "source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return SignalHistoryReply{}, problem.Wrap(err, problem.Unavailable, "analytical signal query failed")
	}

	if signals == nil {
		signals = []signal.Signal{}
	}

	rowCount := len(signals)
	queryMs := elapsed.Milliseconds()

	uc.logger.Info("analytical signal query completed",
		"type", query.Type, "source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"rows", rowCount, "query_ms", queryMs,
	)

	return SignalHistoryReply{
		Signals: signals,
		Source:  "clickhouse",
		Meta: QueryMeta{
			QueryMs:  queryMs,
			RowCount: rowCount,
		},
	}, nil
}
