package analyticalclient

import (
	"internal/domain/instrument"

	"time"

	"internal/domain/decision"
	"internal/domain/execution"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
)

// CompositeChainQuery is the request contract for querying composite execution chains.
//
// Lookup modes (mutually exclusive, checked by use case):
//   - CorrelationID: reconstruct the full chain for one correlation_id.
//   - Source+Symbol+Timeframe (+optional time range): batch lookup — starts from
//     executions, collects correlation_ids, then enriches each chain.
//
// Batch mode always starts from the executions table and walks backward through
// the causal chain. Limit controls how many execution-rooted chains are returned.
type CompositeChainQuery struct {
	CorrelationID string `json:"correlation_id,omitempty"` // single-chain lookup

	// Batch lookup filters (all required for batch mode).
	Source     string                         `json:"source,omitempty"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe,omitempty"`
	Since      int64                          `json:"since,omitempty"` // unix seconds, inclusive
	Until      int64                          `json:"until,omitempty"` // unix seconds, inclusive
	Limit      int                            `json:"limit,omitempty"` // default 20, max 100
}

// CompositeChainReply is the response contract for composite execution chain queries.
type CompositeChainReply struct {
	Chains []CompositeExecutionChain `json:"chains"`
	Source string                    `json:"source"` // always "clickhouse"
	Meta   CompositeQueryMeta        `json:"meta"`
}

// CompositeQueryMeta carries diagnostic signals for composite queries.
type CompositeQueryMeta struct {
	TotalMs    int64 `json:"total_ms"`    // wall-clock for entire composition
	ChainCount int   `json:"chain_count"` // number of chains returned
}

// CompositeExecutionChain is the canonical unit of the composite read model.
// It represents one complete (or partial) causal chain from signal through execution,
// unified by a shared CorrelationID.
//
// Each stage is optional — a chain may be incomplete if events are still propagating
// or if the chain was terminated early (e.g., risk rejected).
type CompositeExecutionChain struct {
	CorrelationID string              `json:"correlation_id"`
	Signal        *SignalWithTrace    `json:"signal,omitempty"`
	Decision      *DecisionWithTrace  `json:"decision,omitempty"`
	Strategy      *StrategyWithTrace  `json:"strategy,omitempty"`
	Risk          *RiskWithTrace      `json:"risk,omitempty"`
	Execution     *ExecutionWithTrace `json:"execution,omitempty"`
	Attribution   *RiskAttribution    `json:"attribution,omitempty"`    // computed from risk stage (S298)
	StageCount    int                 `json:"stage_count"`              // how many stages are present (0-5)
	ChainComplete bool                `json:"chain_complete"`           // true when all 5 stages are present
	MissingStages []string            `json:"missing_stages,omitempty"` // e.g., ["signal", "execution"]
}

// RiskAttribution is a read-side projection computed from the risk stage of a composite chain.
// It surfaces the risk gate outcome at the chain level so that Q2 (why was execution X
// rejected or modified?) is answerable without traversing the nested risk stage.
type RiskAttribution struct {
	Disposition       string                       `json:"disposition"`                // approved/modified/rejected
	Rationale         string                       `json:"rationale"`                  // human-readable explanation
	ActiveConstraints risk.Constraints             `json:"active_constraints"`         // constraints active at assessment time
	StrategyContext   []AttributionStrategyContext `json:"strategy_context,omitempty"` // contributing strategies with decision context
}

// AttributionStrategyContext captures the strategy and decision context that was
// evaluated by the risk gate. This enables tracing the chain of reasoning from
// decision severity through strategy direction to risk outcome.
type AttributionStrategyContext struct {
	Type              string `json:"type"`
	Direction         string `json:"direction"`
	Confidence        string `json:"confidence"`
	DecisionSeverity  string `json:"decision_severity,omitempty"`
	DecisionRationale string `json:"decision_rationale,omitempty"`
}

// PipelineFunnelQuery is the request for stage-by-stage event counts (Q7, Q5).
// Queries all 5 domain tables independently to produce a pipeline conversion funnel.
type PipelineFunnelQuery struct {
	Type       string                         `json:"type"`
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`
	Since      int64                          `json:"since,omitempty"` // unix seconds, inclusive
	Until      int64                          `json:"until,omitempty"` // unix seconds, inclusive
}

// PipelineFunnelReply is the response for the pipeline funnel query.
type PipelineFunnelReply struct {
	Stages []StageFunnelCount `json:"stages"`
	Source string             `json:"source"` // always "clickhouse"
	Meta   CompositeQueryMeta `json:"meta"`
}

// StageFunnelCount holds the event count for one pipeline stage.
type StageFunnelCount struct {
	Stage string `json:"stage"` // signal, decision, strategy, risk, execution
	Count int64  `json:"count"`
}

// DispositionBreakdownQuery is the request for risk disposition distribution (Q6).
type DispositionBreakdownQuery struct {
	Type       string                         `json:"type"`
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`
	Since      int64                          `json:"since,omitempty"` // unix seconds, inclusive
	Until      int64                          `json:"until,omitempty"` // unix seconds, inclusive
}

// DispositionBreakdownReply is the response for the disposition breakdown query.
type DispositionBreakdownReply struct {
	Dispositions []DispositionCount `json:"dispositions"`
	Total        int64              `json:"total"`
	Source       string             `json:"source"` // always "clickhouse"
	Meta         CompositeQueryMeta `json:"meta"`
}

// DispositionCount holds the count and percentage for one risk disposition.
type DispositionCount struct {
	Disposition string  `json:"disposition"` // approved/modified/rejected
	Count       int64   `json:"count"`
	Percentage  float64 `json:"percentage"` // 0-100
}

// SignalWithTrace extends signal.Signal with causal metadata from ClickHouse.
// This closes S295 gap G1 for the signal layer.
type SignalWithTrace struct {
	signal.Signal
	EventID       string    `json:"event_id"`
	CorrelationID string    `json:"correlation_id"`
	CausationID   string    `json:"causation_id"`
	OccurredAt    time.Time `json:"occurred_at"`
}

// DecisionWithTrace extends decision.Decision with causal metadata from ClickHouse.
// This closes S295 gap G1 for the decision layer.
type DecisionWithTrace struct {
	decision.Decision
	EventID       string    `json:"event_id"`
	CorrelationID string    `json:"correlation_id"`
	CausationID   string    `json:"causation_id"`
	OccurredAt    time.Time `json:"occurred_at"`
}

// StrategyWithTrace extends strategy.Strategy with causal metadata from ClickHouse.
// This closes S295 gap G1 for the strategy layer.
type StrategyWithTrace struct {
	strategy.Strategy
	EventID       string    `json:"event_id"`
	CorrelationID string    `json:"correlation_id"`
	CausationID   string    `json:"causation_id"`
	OccurredAt    time.Time `json:"occurred_at"`
}

// RiskWithTrace extends risk.RiskAssessment with causal metadata from ClickHouse.
// This closes S295 gap G1 for the risk layer.
type RiskWithTrace struct {
	risk.RiskAssessment
	EventID       string    `json:"event_id"`
	CorrelationID string    `json:"correlation_id"`
	CausationID   string    `json:"causation_id"`
	OccurredAt    time.Time `json:"occurred_at"`
}

// ExecutionWithTrace extends execution.ExecutionIntent with event-level causal metadata.
// The ExecutionIntent already carries domain-level CorrelationID/CausationID; this adds
// the event-envelope-level IDs for composite chain reconstruction.
type ExecutionWithTrace struct {
	execution.ExecutionIntent
	EventID            string    `json:"event_id"`
	EventCorrelationID string    `json:"event_correlation_id"` // from events.Metadata, not ExecutionIntent
	EventCausationID   string    `json:"event_causation_id"`   // from events.Metadata, not ExecutionIntent
	OccurredAt         time.Time `json:"occurred_at"`
}
