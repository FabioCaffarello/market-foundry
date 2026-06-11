package main

import (
	"internal/domain/instrument"

	"context"
	"log/slog"

	"internal/adapters/clickhouse"
	"internal/application/executionclient"
	"internal/domain/execution"
	"internal/shared/problem"
)

// sessionCHSummaryAdapter bridges clickhouse.ExecutionReader to executionclient.VerifyCHSummary.
// S465: Enables PO verification wiring in the gateway composition.
type sessionCHSummaryAdapter struct {
	reader *clickhouse.ExecutionReader
	logger *slog.Logger
}

func newSessionCHSummaryAdapter(reader *clickhouse.ExecutionReader, logger *slog.Logger) *sessionCHSummaryAdapter {
	return &sessionCHSummaryAdapter{reader: reader, logger: logger}
}

// Summary queries ClickHouse for execution records within the given time bounds.
// S485: Replaces Summary24h — accepts session-derived since/until.
func (a *sessionCHSummaryAdapter) Summary(ctx context.Context, inst instrument.CanonicalInstrument, since, until int64) (int64, *problem.Problem) {
	rows, err := a.reader.QueryExecutionList(ctx, "", "", inst, 0, "", "", since, until, 1000)
	if err != nil {
		a.logger.Warn("ch summary failed", "instrument", inst.Symbol(), "since", since, "until", until, "error", err)
		return 0, problem.New(problem.Internal, "clickhouse summary query failed: "+err.Error())
	}
	return int64(len(rows)), nil
}

// sessionCHListerAdapter bridges clickhouse.ExecutionReader to executionclient.VerifyCHLister
// and executionclient.AuditCHFillReader (both define List with the same signature).
// S465: Enables both PO verification and audit fee analysis wiring.
// S485: Updated to accept session-derived time bounds.
type sessionCHListerAdapter struct {
	reader *clickhouse.ExecutionReader
	logger *slog.Logger
}

func newSessionCHListerAdapter(reader *clickhouse.ExecutionReader, logger *slog.Logger) *sessionCHListerAdapter {
	return &sessionCHListerAdapter{reader: reader, logger: logger}
}

// List queries ClickHouse for execution records within the given time bounds.
// S485: Replaces List24h — accepts session-derived since/until.
func (a *sessionCHListerAdapter) List(ctx context.Context, inst instrument.CanonicalInstrument, execType, status string, limit int, since, until int64) ([]executionclient.VerifyCHListResult, *problem.Problem) {
	if limit <= 0 {
		limit = 100
	}

	intents, err := a.reader.QueryExecutionList(ctx, execType, "", inst, 0, "", status, since, until, limit)
	if err != nil {
		a.logger.Warn("ch list failed", "instrument", inst.Symbol(), "exec_type", execType, "status", status, "since", since, "until", until, "error", err)
		return nil, problem.New(problem.Internal, "clickhouse list query failed: "+err.Error())
	}

	results := make([]executionclient.VerifyCHListResult, 0, len(intents))
	for _, intent := range intents {
		fills := intent.Fills
		if fills == nil {
			fills = []execution.FillRecord{}
		}
		results = append(results, executionclient.VerifyCHListResult{
			Symbol: intent.VenueSymbol(),
			Status: string(intent.Status),
			Type:   intent.Type,
			Fills:  fills,
		})
	}

	return results, nil
}

// crossSessionSessionAdapter bridges the ListSessions use case to the
// analyticalclient.CrossSessionSessionReader interface.
// S495: Enables cross-session pairing to discover sessions from KV.
type crossSessionSessionAdapter struct {
	listUC interface {
		Execute(context.Context, executionclient.SessionListQuery) (executionclient.SessionListReply, *problem.Problem)
	}
}

func (a *crossSessionSessionAdapter) ListSessions(ctx context.Context) ([]execution.Session, error) {
	if a.listUC == nil {
		return nil, nil
	}
	reply, prob := a.listUC.Execute(ctx, executionclient.SessionListQuery{})
	if prob != nil {
		return nil, prob
	}
	return reply.Sessions, nil
}
