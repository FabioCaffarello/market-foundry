package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/execution"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
)

// mapCandleRow maps a CandleSampledEvent to ClickHouse evidence_candles row values.
// Column order matches DDL: event_id, occurred_at, correlation_id, causation_id,
// source, symbol, timeframe, open, high, low, close, volume, trade_count,
// open_time, close_time, final.
func mapCandleRow(e evidence.CandleSampledEvent) []any {
	m := e.Metadata
	c := e.Candle
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		c.Source,
		c.Symbol,
		uint32(c.Timeframe),
		parseFloat(c.Open),
		parseFloat(c.High),
		parseFloat(c.Low),
		parseFloat(c.Close),
		parseFloat(c.Volume),
		c.TradeCount,
		c.OpenTime,
		c.CloseTime,
		c.Final,
	}
}

// mapSignalRow maps a SignalGeneratedEvent to ClickHouse signals row values.
// Column order: event_id, occurred_at, correlation_id, causation_id,
// type, source, symbol, timeframe, value, metadata, final, timestamp.
func mapSignalRow(e signal.SignalGeneratedEvent) []any {
	m := e.Metadata
	s := e.Signal
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		s.Type,
		s.Source,
		s.Symbol,
		uint32(s.Timeframe),
		parseFloat(s.Value),
		marshalJSON(s.Metadata),
		s.Final,
		s.Timestamp,
	}
}

// mapDecisionRow maps a DecisionEvaluatedEvent to ClickHouse decisions row values.
// Column order: event_id, occurred_at, correlation_id, causation_id,
// type, source, symbol, timeframe, outcome, confidence, signals, metadata, final, timestamp.
func mapDecisionRow(e decision.DecisionEvaluatedEvent) []any {
	m := e.Metadata
	d := e.Decision
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		d.Type,
		d.Source,
		d.Symbol,
		uint32(d.Timeframe),
		string(d.Outcome),
		parseFloat(d.Confidence),
		marshalJSON(d.Signals),
		marshalJSON(d.Metadata),
		d.Final,
		d.Timestamp,
	}
}

// mapStrategyRow maps a StrategyResolvedEvent to ClickHouse strategies row values.
// Column order: event_id, occurred_at, correlation_id, causation_id,
// type, source, symbol, timeframe, direction, confidence, decisions, parameters, metadata, final, timestamp.
func mapStrategyRow(e strategy.StrategyResolvedEvent) []any {
	m := e.Metadata
	s := e.Strategy
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		s.Type,
		s.Source,
		s.Symbol,
		uint32(s.Timeframe),
		string(s.Direction),
		parseFloat(s.Confidence),
		marshalJSON(s.Decisions),
		marshalJSON(s.Parameters),
		marshalJSON(s.Metadata),
		s.Final,
		s.Timestamp,
	}
}

// mapRiskRow maps a RiskAssessedEvent to ClickHouse risk_assessments row values.
// Column order: event_id, occurred_at, correlation_id, causation_id,
// type, source, symbol, timeframe, disposition, confidence, strategies,
// constraints, rationale, parameters, metadata, final, timestamp.
func mapRiskRow(e risk.RiskAssessedEvent) []any {
	m := e.Metadata
	r := e.RiskAssessment
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		r.Type,
		r.Source,
		r.Symbol,
		uint32(r.Timeframe),
		string(r.Disposition),
		parseFloat(r.Confidence),
		marshalJSON(r.Strategies),
		marshalJSON(r.Constraints),
		r.Rationale,
		marshalJSON(r.Parameters),
		marshalJSON(r.Metadata),
		r.Final,
		r.Timestamp,
	}
}

// mapExecutionRow maps a PaperOrderSubmittedEvent to ClickHouse executions row values.
// Column order: event_id, occurred_at, correlation_id, causation_id,
// type, source, symbol, timeframe, side, quantity, filled_quantity, status,
// risk, fills, parameters, metadata, exec_correlation_id, exec_causation_id, final, timestamp.
func mapExecutionRow(e execution.PaperOrderSubmittedEvent) []any {
	m := e.Metadata
	x := e.ExecutionIntent
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		x.Type,
		x.Source,
		x.Symbol,
		uint32(x.Timeframe),
		string(x.Side),
		parseFloat(x.Quantity),
		parseFloat(x.FilledQuantity),
		string(x.Status),
		marshalJSON(x.Risk),
		marshalJSON(x.Fills),
		marshalJSON(x.Parameters),
		marshalJSON(x.Metadata),
		x.CorrelationID,
		x.CausationID,
		x.Final,
		x.Timestamp,
	}
}

// parseFloat converts a decimal string to float64.
// Returns 0 on parse failure and logs a warning so silent zero-value injection
// is visible to operators.
func parseFloat(s string) float64 {
	if s == "" {
		slog.Warn("parseFloat: empty string, defaulting to 0")
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		slog.Warn("parseFloat: fallback to 0", "input", s, "error", err)
		return 0
	}
	return f
}

// marshalJSON serializes a value to a JSON string for ClickHouse String columns.
// Returns "{}" on nil or marshal failure and logs a warning on actual errors
// so silent fallback injection is visible to operators.
func marshalJSON(v any) string {
	if v == nil {
		return "{}"
	}
	data, err := json.Marshal(v)
	if err != nil {
		slog.Warn("marshalJSON: fallback to empty object", "type", fmt.Sprintf("%T", v), "error", err)
		return "{}"
	}
	return string(data)
}
