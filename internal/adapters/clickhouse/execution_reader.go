package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

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
