package analyticalclient

import (
	"internal/domain/instrument"

	"context"
	"log/slog"
	"time"

	"internal/domain/decision"
	"internal/shared/problem"
)

// DecisionReader is the local interface for reading historical decisions
// from the analytical store.
type DecisionReader interface {
	QueryDecisionHistory(ctx context.Context, decisionType, source string, inst instrument.CanonicalInstrument, timeframe int, outcome string, since, until int64, limit int) ([]decision.Decision, error)
}

// GetDecisionHistoryUseCase queries the analytical store for historical decisions.
type GetDecisionHistoryUseCase struct {
	reader DecisionReader
	logger *slog.Logger
}

func NewGetDecisionHistoryUseCase(reader DecisionReader, logger *slog.Logger) *GetDecisionHistoryUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetDecisionHistoryUseCase{reader: reader, logger: logger.With("component", "analytical_decision_usecase")}
}

func (uc *GetDecisionHistoryUseCase) Execute(ctx context.Context, query DecisionHistoryQuery) (DecisionHistoryReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return DecisionHistoryReply{}, problem.New(problem.Unavailable, "analytical decision reader is unavailable")
	}

	if query.Type == "" {
		return DecisionHistoryReply{}, problem.New(problem.InvalidArgument, "type is required")
	}
	if query.Source == "" {
		return DecisionHistoryReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Instrument.IsZero() {
		return DecisionHistoryReply{}, problem.New(problem.InvalidArgument, "instrument is required")
	}
	if query.Timeframe <= 0 {
		return DecisionHistoryReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if query.Since < 0 {
		return DecisionHistoryReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return DecisionHistoryReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return DecisionHistoryReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	if query.Limit <= 0 {
		query.Limit = defaultLimit
	}
	if query.Limit > maxLimit {
		query.Limit = maxLimit
	}

	start := time.Now()
	decisions, err := uc.reader.QueryDecisionHistory(ctx, query.Type, query.Source, query.Instrument, query.Timeframe, query.Outcome, query.Since, query.Until, query.Limit)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("analytical decision query failed",
			"type", query.Type, "source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
			"outcome", query.Outcome, "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return DecisionHistoryReply{}, problem.Wrap(err, problem.Unavailable, "analytical decision query failed")
	}

	if decisions == nil {
		decisions = []decision.Decision{}
	}

	rowCount := len(decisions)
	queryMs := elapsed.Milliseconds()

	uc.logger.Info("analytical decision query completed",
		"type", query.Type, "source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
		"outcome", query.Outcome, "rows", rowCount, "query_ms", queryMs,
	)

	return DecisionHistoryReply{
		Decisions: decisions,
		Source:    "clickhouse",
		Meta: QueryMeta{
			QueryMs:  queryMs,
			RowCount: rowCount,
		},
	}, nil
}
