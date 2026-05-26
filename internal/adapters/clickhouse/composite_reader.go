package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/decision"
	"internal/domain/execution"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
)

// CompositeReader queries all five domain tables by correlation_id and assembles
// composite execution chains. It implements the analyticalclient.CompositeReader
// interface — the read-side composition layer for the composite execution model.
//
// Design: application-side composition over 5 independent ClickHouse queries.
// No ClickHouse JOINs, no materialized views, no CDC. Each query hits one table
// filtered by correlation_id (which is indexed via the MergeTree order key prefix
// in combination with source/symbol/timeframe).
type CompositeReader struct {
	client *Client
	logger *slog.Logger
}

// NewCompositeReader creates a CompositeReader backed by the given ClickHouse client.
func NewCompositeReader(client *Client, logger *slog.Logger) *CompositeReader {
	if logger == nil {
		logger = slog.Default()
	}
	return &CompositeReader{client: client, logger: logger.With("component", "composite_reader")}
}

// QueryChainByCorrelationID reconstructs a single composite chain for the given correlation_id,
// scoped to the specified symbol. The symbol filter ensures that correlation-based lookups
// never return events belonging to a different symbol (S301 isolation fix).
func (r *CompositeReader) QueryChainByCorrelationID(ctx context.Context, correlationID, symbol string) (*analyticalclient.CompositeExecutionChain, error) {
	chain := &analyticalclient.CompositeExecutionChain{
		CorrelationID: correlationID,
	}

	// Query each table independently. Errors are logged but do not prevent
	// partial chain assembly — a missing table result is treated as an absent stage.
	sig, err := r.querySignalByCorrelation(ctx, correlationID, symbol)
	if err != nil {
		r.logger.Warn("signal query failed for composite chain", "correlation_id", correlationID, "symbol", symbol, "error", err)
	} else if sig != nil {
		chain.Signal = sig
	}

	dec, err := r.queryDecisionByCorrelation(ctx, correlationID, symbol)
	if err != nil {
		r.logger.Warn("decision query failed for composite chain", "correlation_id", correlationID, "symbol", symbol, "error", err)
	} else if dec != nil {
		chain.Decision = dec
	}

	strat, err := r.queryStrategyByCorrelation(ctx, correlationID, symbol)
	if err != nil {
		r.logger.Warn("strategy query failed for composite chain", "correlation_id", correlationID, "symbol", symbol, "error", err)
	} else if strat != nil {
		chain.Strategy = strat
	}

	rsk, err := r.queryRiskByCorrelation(ctx, correlationID, symbol)
	if err != nil {
		r.logger.Warn("risk query failed for composite chain", "correlation_id", correlationID, "symbol", symbol, "error", err)
	} else if rsk != nil {
		chain.Risk = rsk
	}

	exec, err := r.queryExecutionByCorrelation(ctx, correlationID, symbol)
	if err != nil {
		r.logger.Warn("execution query failed for composite chain", "correlation_id", correlationID, "symbol", symbol, "error", err)
	} else if exec != nil {
		chain.Execution = exec
	}

	computeChainCompleteness(chain)
	return chain, nil
}

// QueryChainsBatch queries executions matching the given filters, collects their
// correlation_ids, and enriches each with the full causal chain. Returns at most
// `limit` chains, ordered by execution timestamp DESC.
func (r *CompositeReader) QueryChainsBatch(ctx context.Context, source, symbol string, timeframe int, since, until int64, limit int) ([]analyticalclient.CompositeExecutionChain, error) {
	// Step 1: query executions table for correlation_ids.
	corrIDs, err := r.queryExecutionCorrelationIDs(ctx, source, symbol, timeframe, since, until, limit)
	if err != nil {
		return nil, fmt.Errorf("query execution correlation_ids: %w", err)
	}
	if len(corrIDs) == 0 {
		return []analyticalclient.CompositeExecutionChain{}, nil
	}

	// Step 2: enrich each correlation_id into a full chain.
	chains := make([]analyticalclient.CompositeExecutionChain, 0, len(corrIDs))
	for _, corrID := range corrIDs {
		chain, err := r.QueryChainByCorrelationID(ctx, corrID, symbol)
		if err != nil {
			r.logger.Warn("chain enrichment failed", "correlation_id", corrID, "error", err)
			continue
		}
		chains = append(chains, *chain)
	}

	return chains, nil
}

// queryExecutionCorrelationIDs returns distinct correlation_ids from the executions
// table matching the given filters, ordered by timestamp DESC.
func (r *CompositeReader) queryExecutionCorrelationIDs(ctx context.Context, source, symbol string, timeframe int, since, until int64, limit int) ([]string, error) {
	// ClickHouse requires GROUP BY for ORDER BY max() with DISTINCT.
	q := "SELECT correlation_id\nFROM executions\nWHERE source = ? AND symbol = ? AND timeframe = ? AND correlation_id != ''"
	args := []any{source, symbol, uint32(timeframe)}

	if since > 0 {
		q += " AND timestamp >= ?"
		args = append(args, time.Unix(since, 0))
	}
	if until > 0 {
		q += " AND timestamp <= ?"
		args = append(args, time.Unix(until, 0))
	}

	q += " GROUP BY correlation_id ORDER BY max(timestamp) DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.client.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query correlation_ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan correlation_id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate correlation_ids: %w", err)
	}
	return ids, nil
}

// querySignalByCorrelation queries signals table for the most recent signal with the given correlation_id and symbol.
func (r *CompositeReader) querySignalByCorrelation(ctx context.Context, correlationID, symbol string) (*analyticalclient.SignalWithTrace, error) {
	q := `SELECT event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, value, metadata, final, timestamp
FROM signals
WHERE correlation_id = ? AND symbol = ?
ORDER BY timestamp DESC LIMIT 1`

	rows, err := r.client.Query(ctx, q, correlationID, symbol)
	if err != nil {
		return nil, fmt.Errorf("query signal by correlation: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("signal rows error: %w", err)
		}
		return nil, nil
	}

	var (
		eventID, corrID, causID string
		occurredAt              time.Time
		typ, src, sym           string
		tf                      uint32
		value                   float64
		metadata                string
		final                   bool
		timestamp               time.Time
	)

	if err := rows.Scan(&eventID, &occurredAt, &corrID, &causID, &typ, &src, &sym, &tf, &value, &metadata, &final, &timestamp); err != nil {
		return nil, fmt.Errorf("scan signal: %w", err)
	}

	sigInst, _ := reconstructInstrumentFromLegacy(src, sym)
	return &analyticalclient.SignalWithTrace{
		Signal: signal.Signal{
			Type:       typ,
			Source:     src,
			Instrument: sigInst,
			Timeframe:  int(tf),
			Value:      FormatFloat(value),
			Metadata:   ParseMetadataJSON(metadata),
			Final:      final,
			Timestamp:  timestamp,
		},
		EventID:       eventID,
		CorrelationID: corrID,
		CausationID:   causID,
		OccurredAt:    occurredAt,
	}, nil
}

// queryDecisionByCorrelation queries decisions table for the most recent decision with the given correlation_id and symbol.
func (r *CompositeReader) queryDecisionByCorrelation(ctx context.Context, correlationID, symbol string) (*analyticalclient.DecisionWithTrace, error) {
	q := `SELECT event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, outcome, confidence, severity, rationale, signals, metadata, final, timestamp
FROM decisions
WHERE correlation_id = ? AND symbol = ?
ORDER BY timestamp DESC LIMIT 1`

	rows, err := r.client.Query(ctx, q, correlationID, symbol)
	if err != nil {
		return nil, fmt.Errorf("query decision by correlation: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("decision rows error: %w", err)
		}
		return nil, nil
	}

	var (
		eventID, corrID, causID       string
		occurredAt                    time.Time
		typ, src, sym                 string
		tf                            uint32
		out                           string
		confidence                    float64
		sev, rationale, signals, meta string
		final                         bool
		timestamp                     time.Time
	)

	if err := rows.Scan(&eventID, &occurredAt, &corrID, &causID, &typ, &src, &sym, &tf, &out, &confidence, &sev, &rationale, &signals, &meta, &final, &timestamp); err != nil {
		return nil, fmt.Errorf("scan decision: %w", err)
	}

	decInst, _ := reconstructInstrumentFromLegacy(src, sym)
	return &analyticalclient.DecisionWithTrace{
		Decision: decision.Decision{
			Type:       typ,
			Source:     src,
			Instrument: decInst,
			Timeframe:  int(tf),
			Outcome:    decision.Outcome(out),
			Severity:   decision.Severity(sev),
			Confidence: FormatFloat(confidence),
			Rationale:  rationale,
			Signals:    ParseSignalInputsJSON(signals),
			Metadata:   ParseMetadataJSON(meta),
			Final:      final,
			Timestamp:  timestamp,
		},
		EventID:       eventID,
		CorrelationID: corrID,
		CausationID:   causID,
		OccurredAt:    occurredAt,
	}, nil
}

// queryStrategyByCorrelation queries strategies table for the most recent strategy with the given correlation_id and symbol.
func (r *CompositeReader) queryStrategyByCorrelation(ctx context.Context, correlationID, symbol string) (*analyticalclient.StrategyWithTrace, error) {
	q := `SELECT event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, direction, confidence, decisions, parameters, metadata, final, timestamp
FROM strategies
WHERE correlation_id = ? AND symbol = ?
ORDER BY timestamp DESC LIMIT 1`

	rows, err := r.client.Query(ctx, q, correlationID, symbol)
	if err != nil {
		return nil, fmt.Errorf("query strategy by correlation: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("strategy rows error: %w", err)
		}
		return nil, nil
	}

	var (
		eventID, corrID, causID         string
		occurredAt                      time.Time
		typ, src, sym                   string
		tf                              uint32
		dir                             string
		confidence                      float64
		decisions, parameters, metadata string
		final                           bool
		timestamp                       time.Time
	)

	if err := rows.Scan(&eventID, &occurredAt, &corrID, &causID, &typ, &src, &sym, &tf, &dir, &confidence, &decisions, &parameters, &metadata, &final, &timestamp); err != nil {
		return nil, fmt.Errorf("scan strategy: %w", err)
	}

	return &analyticalclient.StrategyWithTrace{
		Strategy: strategy.Strategy{
			Type:       typ,
			Source:     src,
			Symbol:     sym,
			Timeframe:  int(tf),
			Direction:  strategy.Direction(dir),
			Confidence: FormatFloat(confidence),
			Decisions:  ParseDecisionInputsJSON(decisions),
			Parameters: ParseMetadataJSON(parameters),
			Metadata:   ParseMetadataJSON(metadata),
			Final:      final,
			Timestamp:  timestamp,
		},
		EventID:       eventID,
		CorrelationID: corrID,
		CausationID:   causID,
		OccurredAt:    occurredAt,
	}, nil
}

// queryRiskByCorrelation queries risk_assessments table for the most recent risk assessment with the given correlation_id and symbol.
func (r *CompositeReader) queryRiskByCorrelation(ctx context.Context, correlationID, symbol string) (*analyticalclient.RiskWithTrace, error) {
	q := `SELECT event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, disposition, confidence, strategies, constraints, rationale, parameters, metadata, final, timestamp
FROM risk_assessments
WHERE correlation_id = ? AND symbol = ?
ORDER BY timestamp DESC LIMIT 1`

	rows, err := r.client.Query(ctx, q, correlationID, symbol)
	if err != nil {
		return nil, fmt.Errorf("query risk by correlation: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("risk rows error: %w", err)
		}
		return nil, nil
	}

	var (
		eventID, corrID, causID                              string
		occurredAt                                           time.Time
		typ, src, sym                                        string
		tf                                                   uint32
		disp                                                 string
		confidence                                           float64
		strategies, constraints, rationale, parameters, meta string
		final                                                bool
		timestamp                                            time.Time
	)

	if err := rows.Scan(&eventID, &occurredAt, &corrID, &causID, &typ, &src, &sym, &tf, &disp, &confidence, &strategies, &constraints, &rationale, &parameters, &meta, &final, &timestamp); err != nil {
		return nil, fmt.Errorf("scan risk: %w", err)
	}

	return &analyticalclient.RiskWithTrace{
		RiskAssessment: risk.RiskAssessment{
			Type:        typ,
			Source:      src,
			Symbol:      sym,
			Timeframe:   int(tf),
			Disposition: risk.Disposition(disp),
			Confidence:  FormatFloat(confidence),
			Strategies:  ParseStrategyInputsJSON(strategies),
			Constraints: ParseConstraintsJSON(constraints),
			Rationale:   rationale,
			Parameters:  ParseMetadataJSON(parameters),
			Metadata:    ParseMetadataJSON(meta),
			Final:       final,
			Timestamp:   timestamp,
		},
		EventID:       eventID,
		CorrelationID: corrID,
		CausationID:   causID,
		OccurredAt:    occurredAt,
	}, nil
}

// queryExecutionByCorrelation queries executions table for the most recent execution with the given correlation_id and symbol.
func (r *CompositeReader) queryExecutionByCorrelation(ctx context.Context, correlationID, symbol string) (*analyticalclient.ExecutionWithTrace, error) {
	q := `SELECT event_id, occurred_at, correlation_id, causation_id, type, source, symbol, timeframe, side, quantity, filled_quantity, status, risk, fills, parameters, metadata, exec_correlation_id, exec_causation_id, final, timestamp
FROM executions
WHERE correlation_id = ? AND symbol = ?
ORDER BY timestamp DESC LIMIT 1`

	rows, err := r.client.Query(ctx, q, correlationID, symbol)
	if err != nil {
		return nil, fmt.Errorf("query execution by correlation: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("execution rows error: %w", err)
		}
		return nil, nil
	}

	var (
		eventID, corrID, causID  string
		occurredAt               time.Time
		typ, src, sym            string
		tf                       uint32
		sd                       string
		quantity, filledQuantity float64
		st                       string
		riskJSON, fillsJSON      string
		parameters, metadata     string
		execCorrID, execCausID   string
		final                    bool
		timestamp                time.Time
	)

	if err := rows.Scan(&eventID, &occurredAt, &corrID, &causID, &typ, &src, &sym, &tf, &sd, &quantity, &filledQuantity, &st, &riskJSON, &fillsJSON, &parameters, &metadata, &execCorrID, &execCausID, &final, &timestamp); err != nil {
		return nil, fmt.Errorf("scan execution: %w", err)
	}

	return &analyticalclient.ExecutionWithTrace{
		ExecutionIntent: execution.ExecutionIntent{
			Type:           typ,
			Source:         src,
			Symbol:         sym,
			Timeframe:      int(tf),
			Side:           execution.Side(sd),
			Quantity:       FormatFloat(quantity),
			FilledQuantity: FormatFloat(filledQuantity),
			Status:         execution.Status(st),
			Risk:           ParseRiskInputJSON(riskJSON),
			Fills:          ParseFillsJSON(fillsJSON),
			Parameters:     ParseMetadataJSON(parameters),
			Metadata:       ParseMetadataJSON(metadata),
			CorrelationID:  execCorrID,
			CausationID:    execCausID,
			Final:          final,
			Timestamp:      timestamp,
		},
		EventID:            eventID,
		EventCorrelationID: corrID,
		EventCausationID:   causID,
		OccurredAt:         occurredAt,
	}, nil
}

// QueryPipelineFunnel counts events per stage across all five domain tables for
// the given type/source/symbol/timeframe. This powers the Q7 (conversion rate)
// and Q5 (pipeline health) aggregation endpoints.
func (r *CompositeReader) QueryPipelineFunnel(ctx context.Context, typ, source, symbol string, timeframe int, since, until int64) ([]analyticalclient.StageFunnelCount, error) {
	tables := []struct {
		stage string
		table string
	}{
		{"signal", "signals"},
		{"decision", "decisions"},
		{"strategy", "strategies"},
		{"risk", "risk_assessments"},
		{"execution", "executions"},
	}

	stages := make([]analyticalclient.StageFunnelCount, 0, 5)
	for _, t := range tables {
		q := "SELECT count() FROM " + t.table + " WHERE type = ? AND source = ? AND symbol = ? AND timeframe = ?"
		args := []any{typ, source, symbol, uint32(timeframe)}

		if since > 0 {
			q += " AND timestamp >= ?"
			args = append(args, time.Unix(since, 0))
		}
		if until > 0 {
			q += " AND timestamp <= ?"
			args = append(args, time.Unix(until, 0))
		}

		rows, err := r.client.Query(ctx, q, args...)
		if err != nil {
			r.logger.Warn("funnel count query failed", "stage", t.stage, "error", err)
			stages = append(stages, analyticalclient.StageFunnelCount{Stage: t.stage, Count: 0})
			continue
		}

		var count uint64
		if rows.Next() {
			if err := rows.Scan(&count); err != nil {
				r.logger.Warn("funnel count scan failed", "stage", t.stage, "error", err)
			}
		}
		rows.Close()

		stages = append(stages, analyticalclient.StageFunnelCount{Stage: t.stage, Count: int64(count)})
	}

	return stages, nil
}

// QueryDispositionBreakdown counts risk assessments grouped by disposition for
// the given type/source/symbol/timeframe. This powers Q6 (blocked vs approved).
func (r *CompositeReader) QueryDispositionBreakdown(ctx context.Context, typ, source, symbol string, timeframe int, since, until int64) ([]analyticalclient.DispositionCount, error) {
	q := "SELECT disposition, count() as cnt FROM risk_assessments WHERE type = ? AND source = ? AND symbol = ? AND timeframe = ?"
	args := []any{typ, source, symbol, uint32(timeframe)}

	if since > 0 {
		q += " AND timestamp >= ?"
		args = append(args, time.Unix(since, 0))
	}
	if until > 0 {
		q += " AND timestamp <= ?"
		args = append(args, time.Unix(until, 0))
	}

	q += " GROUP BY disposition ORDER BY cnt DESC"

	rows, err := r.client.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query disposition breakdown: %w", err)
	}
	defer rows.Close()

	var dispositions []analyticalclient.DispositionCount
	for rows.Next() {
		var disp string
		var count uint64
		if err := rows.Scan(&disp, &count); err != nil {
			return nil, fmt.Errorf("scan disposition: %w", err)
		}
		dispositions = append(dispositions, analyticalclient.DispositionCount{
			Disposition: disp,
			Count:       int64(count),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dispositions: %w", err)
	}

	return dispositions, nil
}

// computeChainCompleteness fills the StageCount, ChainComplete, and MissingStages
// fields on a CompositeExecutionChain.
func computeChainCompleteness(chain *analyticalclient.CompositeExecutionChain) {
	stages := []struct {
		name    string
		present bool
	}{
		{"signal", chain.Signal != nil},
		{"decision", chain.Decision != nil},
		{"strategy", chain.Strategy != nil},
		{"risk", chain.Risk != nil},
		{"execution", chain.Execution != nil},
	}

	count := 0
	var missing []string
	for _, s := range stages {
		if s.present {
			count++
		} else {
			missing = append(missing, s.name)
		}
	}

	chain.StageCount = count
	chain.ChainComplete = count == 5
	chain.MissingStages = missing
}
