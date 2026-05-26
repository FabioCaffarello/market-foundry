package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"internal/domain/risk"
)

// RiskReader queries risk assessments from ClickHouse and maps rows back to domain
// types. It implements analyticalclient.RiskReader — the analytical read-path
// counterpart to cmd/writer/mappers.go mapRiskRow on the write path.
//
// Ownership: adapter layer. This reader owns the storage↔domain translation
// for risk assessment reads. The gateway consumes it through the RiskReader interface
// defined in internal/application/analyticalclient.
type RiskReader struct {
	client *Client
	logger *slog.Logger
}

// NewRiskReader creates a RiskReader backed by the given ClickHouse client.
func NewRiskReader(client *Client, logger *slog.Logger) *RiskReader {
	if logger == nil {
		logger = slog.Default()
	}
	return &RiskReader{client: client, logger: logger.With("component", "risk_reader")}
}

// QueryRiskHistory queries risk assessments from ClickHouse with filters.
// Results are ordered newest-first (DESC by timestamp).
func (r *RiskReader) QueryRiskHistory(ctx context.Context, riskType, source, symbol string, timeframe int, disposition string, since, until int64, limit int) ([]risk.RiskAssessment, error) {
	query, args := BuildRiskQuery(riskType, source, symbol, timeframe, disposition, since, until, limit)

	start := time.Now()
	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		elapsed := time.Since(start)
		r.logger.Error("query failed",
			"risk_type", riskType, "source", source, "symbol", symbol, "timeframe", timeframe,
			"disposition", disposition, "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return nil, fmt.Errorf("query risk history: %w", err)
	}
	defer rows.Close()

	var assessments []risk.RiskAssessment
	for rows.Next() {
		var (
			typ         string
			src         string
			sym         string
			tf          uint32
			disp        string
			confidence  float64
			strategies  string
			constraints string
			rationale   string
			parameters  string
			metadata    string
			final       bool
			timestamp   time.Time
		)

		if err := rows.Scan(&typ, &src, &sym, &tf, &disp, &confidence, &strategies, &constraints, &rationale, &parameters, &metadata, &final, &timestamp); err != nil {
			r.logger.Error("scan failed",
				"risk_type", riskType, "source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
			)
			return nil, fmt.Errorf("scan risk row: %w", err)
		}

		inst, instErr := reconstructInstrumentFromLegacy(src, sym)
		if instErr != nil {
			r.logger.Warn("risk instrument reconstruction failed; emitting zero instrument",
				"source", src, "symbol", sym, "error", instErr,
			)
		}

		assessments = append(assessments, risk.RiskAssessment{
			Type:        typ,
			Source:      src,
			Instrument:  inst,
			Timeframe:   int(tf),
			Disposition: risk.Disposition(disp),
			Confidence:  FormatFloat(confidence),
			Strategies:  ParseStrategyInputsJSON(strategies),
			Constraints: ParseConstraintsJSON(constraints),
			Rationale:   rationale,
			Parameters:  ParseMetadataJSON(parameters),
			Metadata:    ParseMetadataJSON(metadata),
			Final:       final,
			Timestamp:   timestamp,
		})
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration failed",
			"risk_type", riskType, "source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
		)
		return nil, fmt.Errorf("iterate risk rows: %w", err)
	}

	elapsed := time.Since(start)
	r.logger.Debug("query completed",
		"risk_type", riskType, "source", source, "symbol", symbol, "timeframe", timeframe,
		"disposition", disposition, "rows", len(assessments), "elapsed_ms", elapsed.Milliseconds(),
	)

	return assessments, nil
}

// BuildRiskQuery constructs a parameterized SELECT for risk assessments.
// Exported for testing without requiring a live ClickHouse connection.
func BuildRiskQuery(riskType, source, symbol string, timeframe int, disposition string, since, until int64, limit int) (string, []any) {
	return BuildQuery(
		"type, source, symbol, timeframe, disposition, confidence, strategies, constraints, rationale, parameters, metadata, final, timestamp",
		"risk_assessments",
		"type = ? AND source = ? AND symbol = ? AND timeframe = ?",
		[]any{riskType, source, symbol, uint32(timeframe)},
		[]OptionalFilter{{Column: "disposition", Value: disposition}},
		"timestamp", since, until, "timestamp", limit,
	)
}

// ParseStrategyInputsJSON deserializes a JSON string into []risk.StrategyInput.
// Returns an empty slice on parse failure — mirrors marshalJSON's fallback strategy.
// Exported for testing.
func ParseStrategyInputsJSON(raw string) []risk.StrategyInput {
	if raw == "" || raw == "[]" || raw == "{}" {
		return []risk.StrategyInput{}
	}
	var inputs []risk.StrategyInput
	if err := json.Unmarshal([]byte(raw), &inputs); err != nil {
		return []risk.StrategyInput{}
	}
	return inputs
}

// ParseConstraintsJSON deserializes a JSON string into risk.Constraints.
// Returns a zero-value Constraints on parse failure.
// Exported for testing.
func ParseConstraintsJSON(raw string) risk.Constraints {
	if raw == "" || raw == "{}" {
		return risk.Constraints{}
	}
	var c risk.Constraints
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return risk.Constraints{}
	}
	return c
}
