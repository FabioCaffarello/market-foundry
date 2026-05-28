package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"internal/domain/signal"
)

// SignalReader queries signals from ClickHouse and maps rows back to domain
// types. It implements analyticalclient.SignalReader — the analytical read-path
// counterpart to cmd/writer/mappers.go mapSignalRow on the write path.
//
// Ownership: adapter layer. This reader owns the storage↔domain translation
// for signal reads. The gateway consumes it through the SignalReader interface
// defined in internal/application/analyticalclient.
type SignalReader struct {
	client *Client
	logger *slog.Logger
}

// NewSignalReader creates a SignalReader backed by the given ClickHouse client.
func NewSignalReader(client *Client, logger *slog.Logger) *SignalReader {
	if logger == nil {
		logger = slog.Default()
	}
	return &SignalReader{client: client, logger: logger.With("component", "signal_reader")}
}

// QuerySignalHistory queries signals from ClickHouse with filters.
// Results are ordered newest-first (DESC by timestamp).
func (r *SignalReader) QuerySignalHistory(ctx context.Context, signalType, source, symbol string, timeframe int, since, until int64, limit int) ([]signal.Signal, error) {
	query, args := BuildSignalQuery(signalType, source, symbol, timeframe, since, until, limit)

	start := time.Now()
	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		elapsed := time.Since(start)
		r.logger.Error("query failed",
			"signal_type", signalType, "source", source, "symbol", symbol, "timeframe", timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return nil, fmt.Errorf("query signal history: %w", err)
	}
	defer rows.Close()

	var signals []signal.Signal
	for rows.Next() {
		var (
			typ       string
			src       string
			sym       string
			base      string
			quote     string
			contract  string
			tf        uint32
			value     float64
			metadata  string
			final     bool
			timestamp time.Time
		)

		if err := rows.Scan(&typ, &src, &sym, &base, &quote, &contract, &tf, &value, &metadata, &final, &timestamp); err != nil {
			r.logger.Error("scan failed",
				"signal_type", signalType, "source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
			)
			return nil, fmt.Errorf("scan signal row: %w", err)
		}

		meta := ParseMetadataJSON(metadata)
		inst, instErr := instrumentFromCanonicalColumns(base, quote, contract)
		if instErr != nil {
			inst, instErr = reconstructInstrumentFromLegacy(src, sym)
			if instErr != nil {
				r.logger.Warn("signal instrument resolution failed; emitting zero instrument",
					"source", src, "symbol", sym,
					"base", base, "quote", quote, "contract", contract,
					"error", instErr,
				)
			}
		}

		signals = append(signals, signal.Signal{
			Type:       typ,
			Source:     src,
			Instrument: inst,
			Timeframe:  int(tf),
			Value:      FormatFloat(value),
			Metadata:   meta,
			Final:      final,
			Timestamp:  timestamp,
		})
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration failed",
			"signal_type", signalType, "source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
		)
		return nil, fmt.Errorf("iterate signal rows: %w", err)
	}

	elapsed := time.Since(start)
	r.logger.Debug("query completed",
		"signal_type", signalType, "source", source, "symbol", symbol, "timeframe", timeframe,
		"rows", len(signals), "elapsed_ms", elapsed.Milliseconds(),
	)

	return signals, nil
}

// BuildSignalQuery constructs a parameterized SELECT for signals.
// Exported for testing without requiring a live ClickHouse connection.
func BuildSignalQuery(signalType, source, symbol string, timeframe int, since, until int64, limit int) (string, []any) {
	return BuildQuery(
		"type, source, symbol, base, quote, contract, timeframe, value, metadata, final, timestamp",
		"signals",
		"type = ? AND source = ? AND symbol = ? AND timeframe = ?",
		[]any{signalType, source, symbol, uint32(timeframe)},
		nil,
		"timestamp", since, until, "timestamp", limit,
	)
}

// ParseMetadataJSON deserializes a JSON string into a map[string]string.
// Returns an empty map on parse failure — mirrors marshalJSON's fallback strategy.
// Exported for testing.
func ParseMetadataJSON(raw string) map[string]string {
	if raw == "" || raw == "{}" {
		return map[string]string{}
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return map[string]string{}
	}
	return m
}
