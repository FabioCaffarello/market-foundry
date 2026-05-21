package analyticalclient

import (
	"time"

	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/execution"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
)

// CandleHistoryQuery is the request contract for querying historical candles
// from the analytical store (ClickHouse).
//
// Range semantics (all optional, combine freely):
//   - Since/Until: unix seconds, inclusive. 0 = unset.
//   - Limit: max candles returned. Default 50, max 500.
//
// When Since and Until are both set, only candles whose open_time falls
// within [since, until] are returned. Results are always newest-first.
type CandleHistoryQuery struct {
	Source    string `json:"source"`
	Symbol   string `json:"symbol"`
	Timeframe int   `json:"timeframe"`
	Limit     int   `json:"limit"`
	Since     int64  `json:"since,omitempty"` // unix seconds, inclusive lower bound (0 = unset)
	Until     int64  `json:"until,omitempty"` // unix seconds, inclusive upper bound (0 = unset)
}

// QueryMeta carries lightweight diagnostic signals from the analytical read path.
// These fields are populated by the use case layer and surfaced in HTTP responses
// so that operators can assess query health without external tooling.
type QueryMeta struct {
	QueryMs  int64 `json:"query_ms"`  // wall-clock milliseconds spent in the reader adapter
	RowCount int   `json:"row_count"` // number of rows returned by the query
}

// CandleHistoryReply is the response contract for the analytical candle history query.
type CandleHistoryReply struct {
	Candles []evidence.EvidenceCandle `json:"candles"`
	Source  string                    `json:"source"` // always "clickhouse"
	Meta    QueryMeta                `json:"meta"`
}

// SignalHistoryQuery is the request contract for querying historical signals
// from the analytical store (ClickHouse).
//
// Range semantics (all optional, combine freely):
//   - Since/Until: unix seconds, inclusive. 0 = unset.
//   - Limit: max signals returned. Default 50, max 500.
//
// When Since and Until are both set, only signals whose timestamp falls
// within [since, until] are returned. Results are always newest-first.
type SignalHistoryQuery struct {
	Type      string `json:"type"`      // signal type (e.g., "rsi")
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
	Limit     int    `json:"limit"`
	Since     int64  `json:"since,omitempty"` // unix seconds, inclusive lower bound (0 = unset)
	Until     int64  `json:"until,omitempty"` // unix seconds, inclusive upper bound (0 = unset)
}

// SignalHistoryReply is the response contract for the analytical signal history query.
type SignalHistoryReply struct {
	Signals []signal.Signal `json:"signals"`
	Source  string          `json:"source"` // always "clickhouse"
	Meta    QueryMeta       `json:"meta"`
}

// DecisionHistoryQuery is the request contract for querying historical decisions
// from the analytical store (ClickHouse).
//
// Range semantics (all optional, combine freely):
//   - Since/Until: unix seconds, inclusive. 0 = unset.
//   - Limit: max decisions returned. Default 50, max 500.
//   - Outcome: optional filter (e.g., "triggered", "not_triggered", "insufficient").
//
// When Since and Until are both set, only decisions whose timestamp falls
// within [since, until] are returned. Results are always newest-first.
type DecisionHistoryQuery struct {
	Type      string `json:"type"`      // decision type (e.g., "rsi_oversold")
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
	Outcome   string `json:"outcome,omitempty"` // optional outcome filter
	Limit     int    `json:"limit"`
	Since     int64  `json:"since,omitempty"` // unix seconds, inclusive lower bound (0 = unset)
	Until     int64  `json:"until,omitempty"` // unix seconds, inclusive upper bound (0 = unset)
}

// DecisionHistoryReply is the response contract for the analytical decision history query.
type DecisionHistoryReply struct {
	Decisions []decision.Decision `json:"decisions"`
	Source    string              `json:"source"` // always "clickhouse"
	Meta      QueryMeta           `json:"meta"`
}

// StrategyHistoryQuery is the request contract for querying historical strategies
// from the analytical store (ClickHouse).
//
// Range semantics (all optional, combine freely):
//   - Since/Until: unix seconds, inclusive. 0 = unset.
//   - Limit: max strategies returned. Default 50, max 500.
//   - Direction: optional filter (e.g., "long", "short", "flat").
//
// When Since and Until are both set, only strategies whose timestamp falls
// within [since, until] are returned. Results are always newest-first.
type StrategyHistoryQuery struct {
	Type      string `json:"type"`                // strategy type (e.g., "mean_reversion_entry")
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
	Direction string `json:"direction,omitempty"` // optional direction filter
	Limit     int    `json:"limit"`
	Since     int64  `json:"since,omitempty"`     // unix seconds, inclusive lower bound (0 = unset)
	Until     int64  `json:"until,omitempty"`     // unix seconds, inclusive upper bound (0 = unset)
}

// StrategyHistoryReply is the response contract for the analytical strategy history query.
type StrategyHistoryReply struct {
	Strategies []strategy.Strategy `json:"strategies"`
	Source     string              `json:"source"` // always "clickhouse"
	Meta       QueryMeta           `json:"meta"`
}

// RiskHistoryQuery is the request contract for querying historical risk assessments
// from the analytical store (ClickHouse).
//
// Range semantics (all optional, combine freely):
//   - Since/Until: unix seconds, inclusive. 0 = unset.
//   - Limit: max risk assessments returned. Default 50, max 500.
//   - Disposition: optional filter (e.g., "approved", "modified", "rejected").
//
// When Since and Until are both set, only risk assessments whose timestamp falls
// within [since, until] are returned. Results are always newest-first.
type RiskHistoryQuery struct {
	Type        string `json:"type"`                  // risk type (e.g., "position_exposure")
	Source      string `json:"source"`
	Symbol      string `json:"symbol"`
	Timeframe   int    `json:"timeframe"`
	Disposition string `json:"disposition,omitempty"` // optional disposition filter
	Limit       int    `json:"limit"`
	Since       int64  `json:"since,omitempty"`       // unix seconds, inclusive lower bound (0 = unset)
	Until       int64  `json:"until,omitempty"`       // unix seconds, inclusive upper bound (0 = unset)
}

// RiskHistoryReply is the response contract for the analytical risk history query.
type RiskHistoryReply struct {
	RiskAssessments []risk.RiskAssessment `json:"risk_assessments"`
	Source          string                `json:"source"` // always "clickhouse"
	Meta            QueryMeta             `json:"meta"`
}

// ExecutionHistoryQuery is the request contract for querying historical executions
// from the analytical store (ClickHouse).
//
// Range semantics (all optional, combine freely):
//   - Since/Until: unix seconds, inclusive. 0 = unset.
//   - Limit: max executions returned. Default 50, max 500.
//   - Side: optional filter (e.g., "buy", "sell", "none").
//   - Status: optional filter (e.g., "submitted", "filled", "rejected").
//
// When Since and Until are both set, only executions whose timestamp falls
// within [since, until] are returned. Results are always newest-first.
type ExecutionHistoryQuery struct {
	Type      string `json:"type"`                // execution type (e.g., "paper_order")
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
	Side      string `json:"side,omitempty"`      // optional side filter
	Status    string `json:"status,omitempty"`    // optional status filter
	Limit     int    `json:"limit"`
	Since     int64  `json:"since,omitempty"`     // unix seconds, inclusive lower bound (0 = unset)
	Until     int64  `json:"until,omitempty"`     // unix seconds, inclusive upper bound (0 = unset)
}

// ExecutionHistoryReply is the response contract for the analytical execution history query.
type ExecutionHistoryReply struct {
	Executions []execution.ExecutionIntent `json:"executions"`
	Source     string                      `json:"source"` // always "clickhouse"
	Meta       QueryMeta                   `json:"meta"`
}

// LifecycleHistoryQuery is the request contract for querying the historical
// lifecycle timeline from the analytical store (ClickHouse).
//
// Unlike ExecutionHistoryQuery, this query does NOT require a specific execution
// type — it returns all event types (paper_order, venue_market_order,
// venue_rejection) for a given source/symbol/timeframe in chronological order.
// This enables reconstructing the full lifecycle trajectory of an order.
//
// S453A: Introduced to provide a unified historical read model for execution
// lifecycle, reducing dependence on latest-only KV surfaces.
//
// Range semantics (all optional, combine freely):
//   - Since/Until: unix seconds, inclusive. 0 = unset.
//   - Limit: max entries returned. Default 50, max 500.
//   - Status: optional filter (e.g., "filled", "rejected").
//   - Side: optional filter (e.g., "buy", "sell").
type LifecycleHistoryQuery struct {
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
	Status    string `json:"status,omitempty"`
	Side      string `json:"side,omitempty"`
	Limit     int    `json:"limit"`
	Since     int64  `json:"since,omitempty"`
	Until     int64  `json:"until,omitempty"`
}

// LifecycleHistoryEntry represents a single historical event in the execution
// lifecycle timeline. The Type field distinguishes the event source
// (paper_order, venue_market_order, venue_rejection).
//
// S453A: This entry carries enough context to reconstruct the lifecycle
// trajectory without needing to correlate across separate per-type queries.
// S455A: Risk and Parameters fields added to close the cross-surface parity gap —
// these fields were present in ExecutionIntent but omitted from the lifecycle entry,
// making ClickHouse lifecycle queries less informative than KV reads.
type LifecycleHistoryEntry struct {
	Type           string                     `json:"type"`
	Source         string                     `json:"source"`
	Symbol         string                     `json:"symbol"`
	Timeframe      int                        `json:"timeframe"`
	Side           string                     `json:"side"`
	Quantity       string                     `json:"quantity"`
	FilledQuantity string                     `json:"filled_quantity"`
	Status         string                     `json:"status"`
	Risk           execution.RiskInput        `json:"risk"`
	Fills          []execution.FillRecord     `json:"fills"`
	Parameters     map[string]string          `json:"parameters,omitempty"`
	Metadata       map[string]string          `json:"metadata,omitempty"`
	CorrelationID  string                     `json:"correlation_id"`
	CausationID    string                     `json:"causation_id"`
	Final          bool                       `json:"final"`
	Timestamp      string                     `json:"timestamp"`
}

// intentToLifecycleEntry converts an ExecutionIntent to a LifecycleHistoryEntry.
// This is the single conversion point to ensure field parity between the domain
// model and the lifecycle read surface.
func intentToLifecycleEntry(intent execution.ExecutionIntent) LifecycleHistoryEntry {
	return LifecycleHistoryEntry{
		Type:           intent.Type,
		Source:         intent.Source,
		Symbol:         intent.Symbol,
		Timeframe:      intent.Timeframe,
		Side:           string(intent.Side),
		Quantity:       intent.Quantity,
		FilledQuantity: intent.FilledQuantity,
		Status:         string(intent.Status),
		Risk:           intent.Risk,
		Fills:          intent.Fills,
		Parameters:     intent.Parameters,
		Metadata:       intent.Metadata,
		CorrelationID:  intent.CorrelationID,
		CausationID:    intent.CausationID,
		Final:          intent.Final,
		Timestamp:      intent.Timestamp.UTC().Format(time.RFC3339),
	}
}

// LifecycleHistoryReply is the response contract for the lifecycle history query.
type LifecycleHistoryReply struct {
	Entries []LifecycleHistoryEntry `json:"entries"`
	Source  string                  `json:"source"` // always "clickhouse"
	Meta    QueryMeta               `json:"meta"`
}

// ExecutionListQuery is the request contract for the operational list query
// that relaxes the mandatory partition key filters. At least one filter must
// be provided, but none are individually required.
//
// S454A: Enables "show all rejected orders" or "show all fills in the last hour"
// without requiring full source/symbol/timeframe foreknowledge.
type ExecutionListQuery struct {
	Type      string `json:"type,omitempty"`
	Source    string `json:"source,omitempty"`
	Symbol    string `json:"symbol,omitempty"`
	Timeframe int    `json:"timeframe,omitempty"`
	Side      string `json:"side,omitempty"`
	Status    string `json:"status,omitempty"`
	Limit     int    `json:"limit"`
	Since     int64  `json:"since,omitempty"`
	Until     int64  `json:"until,omitempty"`
}

// ExecutionListReply is the response contract for the execution list query.
type ExecutionListReply struct {
	Entries []LifecycleHistoryEntry `json:"entries"`
	Source  string                  `json:"source"`
	Meta    QueryMeta               `json:"meta"`
}

// ExecutionSummaryQuery is the request contract for the execution summary query
// that returns counts grouped by (type, status).
//
// S454A: Enables operational overview like "how many rejected vs filled?"
type ExecutionSummaryQuery struct {
	Source    string `json:"source,omitempty"`
	Symbol    string `json:"symbol,omitempty"`
	Timeframe int    `json:"timeframe,omitempty"`
	Since     int64  `json:"since,omitempty"`
	Until     int64  `json:"until,omitempty"`
}

// ExecutionSummaryEntry represents a single group in the summary.
type ExecutionSummaryEntry struct {
	Type     string `json:"type"`
	Status   string `json:"status"`
	Count    int64  `json:"count"`
	LatestAt string `json:"latest_at"`
}

// ExecutionSummaryReply is the response contract for the execution summary query.
type ExecutionSummaryReply struct {
	Entries []ExecutionSummaryEntry `json:"entries"`
	Source  string                  `json:"source"`
	Meta    QueryMeta               `json:"meta"`
}

// SessionExplainQuery is the request contract for the session explainability endpoint.
// It requires the full partition key (source/symbol/timeframe) and returns a unified
// explanation combining KV latest state, ClickHouse history, and consistency checks.
//
// S455A: Introduced to provide a single surface for operational explainability —
// answering "what happened with this order?" without querying multiple endpoints.
type SessionExplainQuery struct {
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
	Limit     int    `json:"limit"`
}

// ConsistencyCheck captures a single cross-surface consistency finding.
type ConsistencyCheck struct {
	Surface  string `json:"surface"`
	Field    string `json:"field"`
	Status   string `json:"status"` // "consistent", "divergent", "unavailable"
	KVValue  string `json:"kv_value,omitempty"`
	CHValue  string `json:"ch_value,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// SessionExplainReply is the response contract for the session explainability endpoint.
// It combines current KV state, ClickHouse history, and cross-surface consistency
// into a single operational explanation.
//
// S455A: This reply is the primary explainability surface for session/order lifecycle.
type SessionExplainReply struct {
	// Partition key.
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`

	// KV latest state (from NATS KV via execution gateway).
	KVIntentStatus    string `json:"kv_intent_status"`
	KVFillStatus      string `json:"kv_fill_status"`
	KVRejectionStatus string `json:"kv_rejection_status"`
	KVPropagation     string `json:"kv_propagation"`
	KVAvailable       bool   `json:"kv_available"`

	// ClickHouse history (most recent events, newest-first).
	History []LifecycleHistoryEntry `json:"history"`

	// ClickHouse-derived latest status per type.
	CHLatestIntentStatus    string `json:"ch_latest_intent_status"`
	CHLatestFillStatus      string `json:"ch_latest_fill_status"`
	CHLatestRejectionStatus string `json:"ch_latest_rejection_status"`
	CHPropagation           string `json:"ch_propagation"`
	CHAvailable             bool   `json:"ch_available"`

	// Cross-surface consistency checks.
	Consistency []ConsistencyCheck `json:"consistency"`
	Consistent  bool               `json:"consistent"` // true if no divergences found

	// Structured explanation.
	Explanation string `json:"explanation"`

	Meta QueryMeta `json:"meta"`
}
