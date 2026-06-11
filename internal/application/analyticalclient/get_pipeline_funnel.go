package analyticalclient

import (
	"internal/domain/instrument"

	"context"
	"log/slog"
	"time"

	"internal/shared/problem"
)

// AggregationReader is the local interface for aggregation queries across the
// five domain tables. Implemented by clickhouse.CompositeReader.
type AggregationReader interface {
	QueryPipelineFunnel(ctx context.Context, typ, source string, inst instrument.CanonicalInstrument, timeframe int, since, until int64) ([]StageFunnelCount, error)
	QueryDispositionBreakdown(ctx context.Context, typ, source string, inst instrument.CanonicalInstrument, timeframe int, since, until int64) ([]DispositionCount, error)
}

// GetPipelineFunnelUseCase queries all five domain tables to produce a stage-by-stage
// event count funnel. This answers Q7 (conversion rate per stage per family) and
// contributes to Q5 (where did the pipeline break for symbol S?).
type GetPipelineFunnelUseCase struct {
	reader AggregationReader
	logger *slog.Logger
}

func NewGetPipelineFunnelUseCase(reader AggregationReader, logger *slog.Logger) *GetPipelineFunnelUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetPipelineFunnelUseCase{reader: reader, logger: logger.With("component", "analytical_funnel_usecase")}
}

func (uc *GetPipelineFunnelUseCase) Execute(ctx context.Context, query PipelineFunnelQuery) (PipelineFunnelReply, *problem.Problem) {
	if uc == nil || uc.reader == nil {
		return PipelineFunnelReply{}, problem.New(problem.Unavailable, "pipeline funnel reader is unavailable")
	}
	if query.Type == "" {
		return PipelineFunnelReply{}, problem.New(problem.InvalidArgument, "type is required")
	}
	if query.Source == "" {
		return PipelineFunnelReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Instrument.IsZero() {
		return PipelineFunnelReply{}, problem.New(problem.InvalidArgument, "instrument is required")
	}
	if query.Timeframe <= 0 {
		return PipelineFunnelReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if query.Since < 0 {
		return PipelineFunnelReply{}, problem.New(problem.InvalidArgument, "since must be a non-negative unix timestamp")
	}
	if query.Until < 0 {
		return PipelineFunnelReply{}, problem.New(problem.InvalidArgument, "until must be a non-negative unix timestamp")
	}
	if query.Since > 0 && query.Until > 0 && query.Since > query.Until {
		return PipelineFunnelReply{}, problem.New(problem.InvalidArgument, "since must not be after until")
	}

	start := time.Now()
	stages, err := uc.reader.QueryPipelineFunnel(ctx, query.Type, query.Source, query.Instrument, query.Timeframe, query.Since, query.Until)
	elapsed := time.Since(start)

	if err != nil {
		uc.logger.Warn("pipeline funnel query failed",
			"type", query.Type, "source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return PipelineFunnelReply{}, problem.Wrap(err, problem.Unavailable, "pipeline funnel query failed")
	}

	if stages == nil {
		stages = []StageFunnelCount{}
	}

	uc.logger.Info("pipeline funnel query completed",
		"type", query.Type, "source", query.Source, "instrument", query.Instrument.Symbol(), "timeframe", query.Timeframe,
		"total_ms", elapsed.Milliseconds(),
	)

	return PipelineFunnelReply{
		Stages: stages,
		Source: "clickhouse",
		Meta: CompositeQueryMeta{
			TotalMs:    elapsed.Milliseconds(),
			ChainCount: len(stages),
		},
	}, nil
}
