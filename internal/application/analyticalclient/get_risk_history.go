package analyticalclient

import (
	"internal/domain/instrument"

	"context"
	"log/slog"
	"time"

	"internal/domain/risk"
	"internal/shared/problem"
)

// RiskReader is the local interface for reading historical risk assessments
// from the analytical store.
type RiskReader interface {
	QueryRiskHistory(ctx context.Context, riskType, source string, inst instrument.CanonicalInstrument, timeframe int, disposition string, since, until int64, limit int) ([]risk.RiskAssessment, error)
}

// GetRiskHistoryUseCase queries the analytical store for historical risk assessments.
type GetRiskHistoryUseCase struct {
	reader RiskReader
	logger *slog.Logger
}

func NewGetRiskHistoryUseCase(reader RiskReader, logger *slog.Logger) *GetRiskHistoryUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetRiskHistoryUseCase{reader: reader, logger: logger.With("component", "analytical_risk_usecase")}
}

func (uc *GetRiskHistoryUseCase) Execute(ctx context.Context, query RiskHistoryQuery) (RiskHistoryReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return RiskHistoryReply{}, problem.New(problem.Unavailable, "analytical risk reader is unavailable")
	}

	if query.Type == "" {
		return RiskHistoryReply{}, problem.New(problem.InvalidArgument, "type is required")
	}
	if query.Source == "" {
		return RiskHistoryReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Instrument.IsZero() {
		return RiskHistoryReply{}, problem.New(problem.InvalidArgument, "instrument is required")
	}
	if query.Timeframe <= 0 {
		return RiskHistoryReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if query.Since < 0 {
		return RiskHistoryReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return RiskHistoryReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return RiskHistoryReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	if query.Limit <= 0 {
		query.Limit = defaultLimit
	}
	if query.Limit > maxLimit {
		query.Limit = maxLimit
	}

	start := time.Now()
	assessments, err := uc.reader.QueryRiskHistory(ctx, query.Type, query.Source, query.Instrument, query.Timeframe, query.Disposition, query.Since, query.Until, query.Limit)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("analytical risk query failed",
			"type", query.Type, "source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
			"disposition", query.Disposition, "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return RiskHistoryReply{}, problem.Wrap(err, problem.Unavailable, "analytical risk query failed")
	}

	if assessments == nil {
		assessments = []risk.RiskAssessment{}
	}

	rowCount := len(assessments)
	queryMs := elapsed.Milliseconds()

	uc.logger.Info("analytical risk query completed",
		"type", query.Type, "source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
		"disposition", query.Disposition, "rows", rowCount, "query_ms", queryMs,
	)

	return RiskHistoryReply{
		RiskAssessments: assessments,
		Source:          "clickhouse",
		Meta: QueryMeta{
			QueryMs:  queryMs,
			RowCount: rowCount,
		},
	}, nil
}
