package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/execution"
)

// ExecutionReader queries executions from ClickHouse and maps rows back to domain
// types. It implements analyticalclient.ExecutionReader — the analytical read-path
// counterpart to cmd/writer/mappers.go mapExecutionRow on the write path.
//
// Ownership: adapter layer. This reader owns the storage↔domain translation
// for execution reads. The gateway consumes it through the ExecutionReader interface
// defined in internal/application/analyticalclient.
type ExecutionReader struct {
	client *Client
	logger *slog.Logger
}

// NewExecutionReader creates an ExecutionReader backed by the given ClickHouse client.
func NewExecutionReader(client *Client, logger *slog.Logger) *ExecutionReader {
	if logger == nil {
		logger = slog.Default()
	}
	return &ExecutionReader{client: client, logger: logger.With("component", "execution_reader")}
}

// QueryExecutionHistory queries executions from ClickHouse with filters.
// Results are ordered newest-first (DESC by timestamp).
func (r *ExecutionReader) QueryExecutionHistory(ctx context.Context, execType, source, symbol string, timeframe int, side, status string, since, until int64, limit int) ([]execution.ExecutionIntent, error) {
	query, args := BuildExecutionQuery(execType, source, symbol, timeframe, side, status, since, until, limit)

	start := time.Now()
	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		elapsed := time.Since(start)
		r.logger.Error("query failed",
			"exec_type", execType, "source", source, "symbol", symbol, "timeframe", timeframe,
			"side", side, "status", status, "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return nil, fmt.Errorf("query execution history: %w", err)
	}
	defer rows.Close()

	var executions []execution.ExecutionIntent
	for rows.Next() {
		var (
			typ            string
			src            string
			sym            string
			tf             uint32
			sd             string
			quantity       float64
			filledQuantity float64
			st             string
			riskJSON       string
			fillsJSON      string
			parameters     string
			metadata       string
			execCorrID     string
			execCausID     string
			final          bool
			timestamp      time.Time
		)

		if err := rows.Scan(&typ, &src, &sym, &tf, &sd, &quantity, &filledQuantity, &st, &riskJSON, &fillsJSON, &parameters, &metadata, &execCorrID, &execCausID, &final, &timestamp); err != nil {
			r.logger.Error("scan failed",
				"exec_type", execType, "source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
			)
			return nil, fmt.Errorf("scan execution row: %w", err)
		}

		executions = append(executions, execution.ExecutionIntent{
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
		})
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration failed",
			"exec_type", execType, "source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
		)
		return nil, fmt.Errorf("iterate execution rows: %w", err)
	}

	elapsed := time.Since(start)
	r.logger.Debug("query completed",
		"exec_type", execType, "source", source, "symbol", symbol, "timeframe", timeframe,
		"side", side, "status", status, "rows", len(executions), "elapsed_ms", elapsed.Milliseconds(),
	)

	return executions, nil
}

// BuildExecutionQuery constructs a parameterized SELECT for executions.
// Exported for testing without requiring a live ClickHouse connection.
func BuildExecutionQuery(execType, source, symbol string, timeframe int, side, status string, since, until int64, limit int) (string, []any) {
	return BuildQuery(
		"type, source, symbol, timeframe, side, quantity, filled_quantity, status, risk, fills, parameters, metadata, exec_correlation_id, exec_causation_id, final, timestamp",
		"executions",
		"type = ? AND source = ? AND symbol = ? AND timeframe = ?",
		[]any{execType, source, symbol, uint32(timeframe)},
		[]OptionalFilter{
			{Column: "side", Value: side},
			{Column: "status", Value: status},
		},
		"timestamp", since, until, "timestamp", limit,
	)
}

// QueryLifecycleHistory queries all execution event types for a given source/symbol/timeframe
// from ClickHouse, returning a unified chronological timeline. Unlike QueryExecutionHistory,
// this does NOT filter by type — it returns paper_order, venue_market_order, and
// venue_rejection events together, ordered newest-first.
//
// S453A: This is the core historical read model query that enables lifecycle reconstruction
// without requiring separate per-type queries or reliance on latest-only KV surfaces.
func (r *ExecutionReader) QueryLifecycleHistory(ctx context.Context, source, symbol string, timeframe int, side, status string, since, until int64, limit int) ([]execution.ExecutionIntent, error) {
	query, args := BuildLifecycleHistoryQuery(source, symbol, timeframe, side, status, since, until, limit)

	start := time.Now()
	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		elapsed := time.Since(start)
		r.logger.Error("lifecycle history query failed",
			"source", source, "symbol", symbol, "timeframe", timeframe,
			"side", side, "status", status, "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return nil, fmt.Errorf("query lifecycle history: %w", err)
	}
	defer rows.Close()

	var executions []execution.ExecutionIntent
	for rows.Next() {
		var (
			typ            string
			src            string
			sym            string
			tf             uint32
			sd             string
			quantity       float64
			filledQuantity float64
			st             string
			riskJSON       string
			fillsJSON      string
			parameters     string
			metadata       string
			execCorrID     string
			execCausID     string
			final          bool
			timestamp      time.Time
		)

		if err := rows.Scan(&typ, &src, &sym, &tf, &sd, &quantity, &filledQuantity, &st, &riskJSON, &fillsJSON, &parameters, &metadata, &execCorrID, &execCausID, &final, &timestamp); err != nil {
			r.logger.Error("lifecycle history scan failed",
				"source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
			)
			return nil, fmt.Errorf("scan lifecycle history row: %w", err)
		}

		executions = append(executions, execution.ExecutionIntent{
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
		})
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("lifecycle history row iteration failed",
			"source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
		)
		return nil, fmt.Errorf("iterate lifecycle history rows: %w", err)
	}

	elapsed := time.Since(start)
	r.logger.Debug("lifecycle history query completed",
		"source", source, "symbol", symbol, "timeframe", timeframe,
		"side", side, "status", status, "rows", len(executions), "elapsed_ms", elapsed.Milliseconds(),
	)

	return executions, nil
}

// BuildLifecycleHistoryQuery constructs a parameterized SELECT for lifecycle history.
// Unlike BuildExecutionQuery, the mandatory WHERE filters by source/symbol/timeframe
// only — type is NOT a mandatory filter, allowing all event types in the result.
// Exported for testing without requiring a live ClickHouse connection.
func BuildLifecycleHistoryQuery(source, symbol string, timeframe int, side, status string, since, until int64, limit int) (string, []any) {
	return BuildQuery(
		"type, source, symbol, timeframe, side, quantity, filled_quantity, status, risk, fills, parameters, metadata, exec_correlation_id, exec_causation_id, final, timestamp",
		"executions",
		"source = ? AND symbol = ? AND timeframe = ?",
		[]any{source, symbol, uint32(timeframe)},
		[]OptionalFilter{
			{Column: "side", Value: side},
			{Column: "status", Value: status},
		},
		"timestamp", since, until, "timestamp", limit,
	)
}

// QueryExecutionList queries executions from ClickHouse with relaxed filters.
// Unlike QueryExecutionHistory, none of the partition key fields (type, source,
// symbol, timeframe) are individually mandatory — but at least one filter must
// be provided to prevent unbounded scans.
//
// S454A: Enables operational list queries like "show all rejected orders" or
// "show all fills in the last hour" without requiring full partition key foreknowledge.
func (r *ExecutionReader) QueryExecutionList(ctx context.Context, execType, source, symbol string, timeframe int, side, status string, since, until int64, limit int) ([]execution.ExecutionIntent, error) {
	query, args, err := BuildExecutionListQuery(execType, source, symbol, timeframe, side, status, since, until, limit)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	rows, qErr := r.client.Query(ctx, query, args...)
	if qErr != nil {
		elapsed := time.Since(start)
		r.logger.Error("execution list query failed",
			"exec_type", execType, "source", source, "symbol", symbol, "timeframe", timeframe,
			"side", side, "status", status, "elapsed_ms", elapsed.Milliseconds(), "error", qErr,
		)
		return nil, fmt.Errorf("query execution list: %w", qErr)
	}
	defer rows.Close()

	var executions []execution.ExecutionIntent
	for rows.Next() {
		var (
			typ            string
			src            string
			sym            string
			tf             uint32
			sd             string
			quantity       float64
			filledQuantity float64
			st             string
			riskJSON       string
			fillsJSON      string
			parameters     string
			metadata       string
			execCorrID     string
			execCausID     string
			final          bool
			timestamp      time.Time
		)

		if err := rows.Scan(&typ, &src, &sym, &tf, &sd, &quantity, &filledQuantity, &st, &riskJSON, &fillsJSON, &parameters, &metadata, &execCorrID, &execCausID, &final, &timestamp); err != nil {
			r.logger.Error("execution list scan failed", "error", err)
			return nil, fmt.Errorf("scan execution list row: %w", err)
		}

		executions = append(executions, execution.ExecutionIntent{
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
		})
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("execution list row iteration failed", "error", err)
		return nil, fmt.Errorf("iterate execution list rows: %w", err)
	}

	elapsed := time.Since(start)
	r.logger.Debug("execution list query completed",
		"exec_type", execType, "source", source, "symbol", symbol, "timeframe", timeframe,
		"side", side, "status", status, "rows", len(executions), "elapsed_ms", elapsed.Milliseconds(),
	)

	return executions, nil
}

// BuildExecutionListQuery constructs a parameterized SELECT for execution list queries
// with relaxed filters. At least one filter (type, source, symbol, timeframe, side,
// status, or a time range) must be provided.
// Exported for testing without requiring a live ClickHouse connection.
func BuildExecutionListQuery(execType, source, symbol string, timeframe int, side, status string, since, until int64, limit int) (string, []any, error) {
	selectCols := "type, source, symbol, timeframe, side, quantity, filled_quantity, status, risk, fills, parameters, metadata, exec_correlation_id, exec_causation_id, final, timestamp"
	q := "SELECT " + selectCols + "\nFROM executions\nWHERE 1=1"
	var args []any
	hasFilter := false

	if execType != "" {
		q += " AND type = ?"
		args = append(args, execType)
		hasFilter = true
	}
	if source != "" {
		q += " AND source = ?"
		args = append(args, source)
		hasFilter = true
	}
	if symbol != "" {
		q += " AND symbol = ?"
		args = append(args, symbol)
		hasFilter = true
	}
	if timeframe > 0 {
		q += " AND timeframe = ?"
		args = append(args, uint32(timeframe))
		hasFilter = true
	}
	if side != "" {
		q += " AND side = ?"
		args = append(args, side)
		hasFilter = true
	}
	if status != "" {
		q += " AND status = ?"
		args = append(args, status)
		hasFilter = true
	}
	if since > 0 {
		q += " AND timestamp >= ?"
		args = append(args, time.Unix(since, 0))
		hasFilter = true
	}
	if until > 0 {
		q += " AND timestamp <= ?"
		args = append(args, time.Unix(until, 0))
		hasFilter = true
	}

	if !hasFilter {
		return "", nil, fmt.Errorf("at least one filter is required for execution list query")
	}

	q += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	return q, args, nil
}

// QueryExecutionSummary returns execution counts grouped by (type, status) with the
// most recent timestamp per group. Filters are optional — at least one must be provided.
//
// S454A: Enables operational overview queries like "how many rejected vs filled?"
// without fetching individual rows.
func (r *ExecutionReader) QueryExecutionSummary(ctx context.Context, source, symbol string, timeframe int, since, until int64) ([]analyticalclient.ExecutionSummaryRawRow, error) {
	q, args, err := BuildExecutionSummaryQuery(source, symbol, timeframe, since, until)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	rows, qErr := r.client.Query(ctx, q, args...)
	if qErr != nil {
		elapsed := time.Since(start)
		r.logger.Error("execution summary query failed",
			"source", source, "symbol", symbol, "timeframe", timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", qErr,
		)
		return nil, fmt.Errorf("query execution summary: %w", qErr)
	}
	defer rows.Close()

	var results []analyticalclient.ExecutionSummaryRawRow
	for rows.Next() {
		var (
			typ      string
			st       string
			count    uint64
			latestTs time.Time
		)
		if err := rows.Scan(&typ, &st, &count, &latestTs); err != nil {
			return nil, fmt.Errorf("scan execution summary row: %w", err)
		}
		results = append(results, analyticalclient.ExecutionSummaryRawRow{
			Type:     typ,
			Status:   st,
			Count:    int64(count),
			LatestAt: latestTs,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate execution summary rows: %w", err)
	}

	elapsed := time.Since(start)
	r.logger.Debug("execution summary query completed",
		"source", source, "symbol", symbol, "timeframe", timeframe,
		"groups", len(results), "elapsed_ms", elapsed.Milliseconds(),
	)

	return results, nil
}

// BuildExecutionSummaryQuery constructs a GROUP BY query for execution counts
// by (type, status). At least one filter or time range must be provided.
// Exported for testing.
func BuildExecutionSummaryQuery(source, symbol string, timeframe int, since, until int64) (string, []any, error) {
	q := "SELECT type, status, count() as cnt, max(timestamp) as latest_at\nFROM executions\nWHERE 1=1"
	var args []any
	hasFilter := false

	if source != "" {
		q += " AND source = ?"
		args = append(args, source)
		hasFilter = true
	}
	if symbol != "" {
		q += " AND symbol = ?"
		args = append(args, symbol)
		hasFilter = true
	}
	if timeframe > 0 {
		q += " AND timeframe = ?"
		args = append(args, uint32(timeframe))
		hasFilter = true
	}
	if since > 0 {
		q += " AND timestamp >= ?"
		args = append(args, time.Unix(since, 0))
		hasFilter = true
	}
	if until > 0 {
		q += " AND timestamp <= ?"
		args = append(args, time.Unix(until, 0))
		hasFilter = true
	}

	if !hasFilter {
		return "", nil, fmt.Errorf("at least one filter is required for execution summary query")
	}

	q += " GROUP BY type, status ORDER BY cnt DESC"
	return q, args, nil
}

// ParseRiskInputJSON deserializes a JSON string into execution.RiskInput.
// Returns a zero-value RiskInput on parse failure.
// Exported for testing.
func ParseRiskInputJSON(raw string) execution.RiskInput {
	if raw == "" || raw == "{}" {
		return execution.RiskInput{}
	}
	var ri execution.RiskInput
	if err := json.Unmarshal([]byte(raw), &ri); err != nil {
		return execution.RiskInput{}
	}
	return ri
}

// ParseFillsJSON deserializes a JSON string into []execution.FillRecord.
// Returns an empty slice on parse failure.
// Exported for testing.
func ParseFillsJSON(raw string) []execution.FillRecord {
	if raw == "" || raw == "[]" || raw == "{}" {
		return []execution.FillRecord{}
	}
	var fills []execution.FillRecord
	if err := json.Unmarshal([]byte(raw), &fills); err != nil {
		return []execution.FillRecord{}
	}
	return fills
}
