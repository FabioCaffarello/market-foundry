package clickhouse

import (
	"internal/domain/instrument"

	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"internal/domain/strategy"
)

// StrategyReader queries strategies from ClickHouse and maps rows back to domain
// types. It implements analyticalclient.StrategyReader — the analytical read-path
// counterpart to cmd/writer/mappers.go mapStrategyRow on the write path.
//
// Ownership: adapter layer. This reader owns the storage-to-domain translation
// for strategy reads. The gateway consumes it through the StrategyReader interface
// defined in internal/application/analyticalclient.
type StrategyReader struct {
	client *Client
	logger *slog.Logger
}

// NewStrategyReader creates a StrategyReader backed by the given ClickHouse client.
func NewStrategyReader(client *Client, logger *slog.Logger) *StrategyReader {
	if logger == nil {
		logger = slog.Default()
	}
	return &StrategyReader{client: client, logger: logger.With("component", "strategy_reader")}
}

// QueryStrategyHistory queries strategies from ClickHouse with filters.
// Results are ordered newest-first (DESC by timestamp).
func (r *StrategyReader) QueryStrategyHistory(ctx context.Context, strategyType, source string, inst instrument.CanonicalInstrument, timeframe int, direction string, since, until int64, limit int) ([]strategy.Strategy, error) {
	symbol := inst.LegacyFilterValue()
	query, args := BuildStrategyQuery(strategyType, source, symbol, timeframe, direction, since, until, limit)

	start := time.Now()
	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		elapsed := time.Since(start)
		r.logger.Error("query failed",
			"strategy_type", strategyType, "source", source, "symbol", symbol, "timeframe", timeframe,
			"direction", direction, "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return nil, fmt.Errorf("query strategy history: %w", err)
	}
	defer rows.Close()

	var strategies []strategy.Strategy
	for rows.Next() {
		var (
			typ        string
			src        string
			sym        string
			base       string
			quote      string
			contract   string
			tf         uint32
			dir        string
			confidence float64
			decisions  string
			parameters string
			metadata   string
			final      bool
			timestamp  time.Time
		)

		if err := rows.Scan(&typ, &src, &sym, &base, &quote, &contract, &tf, &dir, &confidence, &decisions, &parameters, &metadata, &final, &timestamp); err != nil {
			r.logger.Error("scan failed",
				"strategy_type", strategyType, "source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
			)
			return nil, fmt.Errorf("scan strategy row: %w", err)
		}

		inst, instErr := instrumentFromCanonicalColumns(base, quote, contract)
		if instErr != nil {
			inst, instErr = reconstructInstrumentFromLegacy(src, sym)
			if instErr != nil {
				r.logger.Warn("strategy instrument resolution failed; emitting zero instrument",
					"source", src, "symbol", sym,
					"base", base, "quote", quote, "contract", contract,
					"error", instErr,
				)
			}
		}

		strategies = append(strategies, strategy.Strategy{
			Type:       typ,
			Source:     src,
			Instrument: inst,
			Timeframe:  int(tf),
			Direction:  strategy.Direction(dir),
			Confidence: FormatFloat(confidence),
			Decisions:  ParseDecisionInputsJSON(decisions),
			Parameters: ParseMetadataJSON(parameters),
			Metadata:   ParseMetadataJSON(metadata),
			Final:      final,
			Timestamp:  timestamp,
		})
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration failed",
			"strategy_type", strategyType, "source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
		)
		return nil, fmt.Errorf("iterate strategy rows: %w", err)
	}

	elapsed := time.Since(start)
	r.logger.Debug("query completed",
		"strategy_type", strategyType, "source", source, "symbol", symbol, "timeframe", timeframe,
		"direction", direction, "rows", len(strategies), "elapsed_ms", elapsed.Milliseconds(),
	)

	return strategies, nil
}

// BuildStrategyQuery constructs a parameterized SELECT for strategies.
// Exported for testing without requiring a live ClickHouse connection.
func BuildStrategyQuery(strategyType, source, symbol string, timeframe int, direction string, since, until int64, limit int) (string, []any) {
	return BuildQuery(
		"type, source, symbol, base, quote, contract, timeframe, direction, confidence, decisions, parameters, metadata, final, timestamp",
		"strategies",
		"type = ? AND source = ? AND symbol = ? AND timeframe = ?",
		[]any{strategyType, source, symbol, uint32(timeframe)},
		[]OptionalFilter{{Column: "direction", Value: direction}},
		"timestamp", since, until, "timestamp", limit,
	)
}

// ParseDecisionInputsJSON deserializes a JSON string into []strategy.DecisionInput.
// Returns an empty slice on parse failure — mirrors marshalJSON's fallback strategy.
// Exported for testing.
func ParseDecisionInputsJSON(raw string) []strategy.DecisionInput {
	if raw == "" || raw == "[]" || raw == "{}" {
		return []strategy.DecisionInput{}
	}
	var inputs []strategy.DecisionInput
	if err := json.Unmarshal([]byte(raw), &inputs); err != nil {
		return []strategy.DecisionInput{}
	}
	return inputs
}
