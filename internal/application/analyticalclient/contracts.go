package analyticalclient

import (
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
