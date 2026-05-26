package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"internal/domain/decision"
)

// DecisionReader queries decisions from ClickHouse and maps rows back to domain
// types. It implements analyticalclient.DecisionReader — the analytical read-path
// counterpart to cmd/writer/mappers.go mapDecisionRow on the write path.
//
// Ownership: adapter layer. This reader owns the storage-to-domain translation
// for decision reads. The gateway consumes it through the DecisionReader interface
// defined in internal/application/analyticalclient.
type DecisionReader struct {
	client *Client
	logger *slog.Logger
}

// NewDecisionReader creates a DecisionReader backed by the given ClickHouse client.
func NewDecisionReader(client *Client, logger *slog.Logger) *DecisionReader {
	if logger == nil {
		logger = slog.Default()
	}
	return &DecisionReader{client: client, logger: logger.With("component", "decision_reader")}
}

// QueryDecisionHistory queries decisions from ClickHouse with filters.
// Results are ordered newest-first (DESC by timestamp).
func (r *DecisionReader) QueryDecisionHistory(ctx context.Context, decisionType, source, symbol string, timeframe int, outcome string, since, until int64, limit int) ([]decision.Decision, error) {
	query, args := BuildDecisionQuery(decisionType, source, symbol, timeframe, outcome, since, until, limit)

	start := time.Now()
	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		elapsed := time.Since(start)
		r.logger.Error("query failed",
			"decision_type", decisionType, "source", source, "symbol", symbol, "timeframe", timeframe,
			"outcome", outcome, "elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return nil, fmt.Errorf("query decision history: %w", err)
	}
	defer rows.Close()

	var decisions []decision.Decision
	for rows.Next() {
		var (
			typ        string
			src        string
			sym        string
			tf         uint32
			out        string
			confidence float64
			sev        string
			rationale  string
			signals    string
			metadata   string
			final      bool
			timestamp  time.Time
		)

		if err := rows.Scan(&typ, &src, &sym, &tf, &out, &confidence, &sev, &rationale, &signals, &metadata, &final, &timestamp); err != nil {
			r.logger.Error("scan failed",
				"decision_type", decisionType, "source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
			)
			return nil, fmt.Errorf("scan decision row: %w", err)
		}

		inst, instErr := reconstructInstrumentFromLegacy(src, sym)
		if instErr != nil {
			r.logger.Warn("decision instrument reconstruction failed; emitting zero instrument",
				"source", src, "symbol", sym, "error", instErr,
			)
		}

		decisions = append(decisions, decision.Decision{
			Type:       typ,
			Source:     src,
			Instrument: inst,
			Timeframe:  int(tf),
			Outcome:    decision.Outcome(out),
			Severity:   decision.Severity(sev),
			Confidence: FormatFloat(confidence),
			Rationale:  rationale,
			Signals:    ParseSignalInputsJSON(signals),
			Metadata:   ParseMetadataJSON(metadata),
			Final:      final,
			Timestamp:  timestamp,
		})
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration failed",
			"decision_type", decisionType, "source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
		)
		return nil, fmt.Errorf("iterate decision rows: %w", err)
	}

	elapsed := time.Since(start)
	r.logger.Debug("query completed",
		"decision_type", decisionType, "source", source, "symbol", symbol, "timeframe", timeframe,
		"outcome", outcome, "rows", len(decisions), "elapsed_ms", elapsed.Milliseconds(),
	)

	return decisions, nil
}

// BuildDecisionQuery constructs a parameterized SELECT for decisions.
// Exported for testing without requiring a live ClickHouse connection.
func BuildDecisionQuery(decisionType, source, symbol string, timeframe int, outcome string, since, until int64, limit int) (string, []any) {
	return BuildQuery(
		"type, source, symbol, timeframe, outcome, confidence, severity, rationale, signals, metadata, final, timestamp",
		"decisions",
		"type = ? AND source = ? AND symbol = ? AND timeframe = ?",
		[]any{decisionType, source, symbol, uint32(timeframe)},
		[]OptionalFilter{{Column: "outcome", Value: outcome}},
		"timestamp", since, until, "timestamp", limit,
	)
}

// ParseSignalInputsJSON deserializes a JSON string into []decision.SignalInput.
// Returns an empty slice on parse failure — mirrors marshalJSON's fallback strategy.
// Exported for testing.
func ParseSignalInputsJSON(raw string) []decision.SignalInput {
	if raw == "" || raw == "[]" || raw == "{}" {
		return []decision.SignalInput{}
	}
	var signals []decision.SignalInput
	if err := json.Unmarshal([]byte(raw), &signals); err != nil {
		return []decision.SignalInput{}
	}
	return signals
}
