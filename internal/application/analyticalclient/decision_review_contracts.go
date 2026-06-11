package analyticalclient

import (
	"internal/domain/instrument"

	"time"

	"internal/domain/consistency"
	"internal/domain/decision"
	"internal/domain/risk"
)

// DecisionReviewQuery is the request contract for reviewing how a decision was formed
// and what downstream effects it produced.
//
// Lookup modes:
//   - CorrelationID+Symbol: review the chain for one specific decision event.
//   - Source+Symbol+Timeframe: batch review of recent decisions with their full evidence bundles.
//
// The decision review surface answers:
//   - What signal inputs contributed to this decision?
//   - What was the decision outcome and why (rationale, severity, confidence)?
//   - What strategy was resolved from it?
//   - What risk constraints were applied?
//   - Did an execution intent result, and what was its fate?
//
// S471: Introduced for decision-centric review and evidence bundling.
type DecisionReviewQuery struct {
	CorrelationID string `json:"correlation_id,omitempty"` // single-decision lookup

	// Batch lookup filters.
	Source     string                         `json:"source,omitempty"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe,omitempty"`
	Outcome    string                         `json:"outcome,omitempty"` // optional: triggered, not_triggered, insufficient
	Since      int64                          `json:"since,omitempty"`
	Until      int64                          `json:"until,omitempty"`
	Limit      int                            `json:"limit,omitempty"` // default 20, max 100
}

// DecisionReviewReply is the response contract for decision review queries.
type DecisionReviewReply struct {
	Reviews []DecisionReviewBundle `json:"reviews"`
	Source  string                 `json:"source"` // always "clickhouse"
	Meta    CompositeQueryMeta     `json:"meta"`
}

// DecisionReviewBundle is the canonical evidence unit for reviewing a single decision.
// It is anchored on the decision event and assembles all causally related artifacts
// into a single auditable structure.
//
// The bundle answers six questions:
//  1. Inputs: what signals fed this decision?
//  2. Transform: what logic produced the outcome (type, rationale, severity)?
//  3. Resolution: what strategy was derived?
//  4. Constraints: what risk assessment was applied, and what limits were imposed?
//  5. Output: did an execution intent result, and what happened to it?
//  6. Effectiveness: was the decision good? (S476)
//
// Fields are optional — a decision that was not_triggered may have no strategy or execution.
type DecisionReviewBundle struct {
	CorrelationID string `json:"correlation_id"`

	// Inputs: signal evidence that fed the decision.
	Inputs *ReviewInputs `json:"inputs,omitempty"`

	// Transform: the decision itself — the core of the review.
	Transform *ReviewTransform `json:"transform"`

	// Resolution: strategy derived from the decision.
	Resolution *ReviewResolution `json:"resolution,omitempty"`

	// Constraints: risk assessment and limits applied.
	Constraints *ReviewConstraints `json:"constraints,omitempty"`

	// Output: execution intent and its lifecycle outcome.
	Output *ReviewOutput `json:"output,omitempty"`

	// S476: Effectiveness attribution — outcome quality of the decision chain.
	// Present only when execution reached a terminal state with classifiable data.
	// Null/absent for in-progress, not-triggered, or rejected executions.
	Effectiveness *ReviewEffectiveness `json:"effectiveness,omitempty"`

	// Chain completeness metadata.
	StageCount    int      `json:"stage_count"`
	ChainComplete bool     `json:"chain_complete"`
	MissingStages []string `json:"missing_stages,omitempty"`

	// S472: Cross-domain consistency check results.
	Consistency *consistency.Report `json:"consistency,omitempty"`

	// Explainability summary — a human-readable synthesis.
	Explanation string `json:"explanation"`
}

// ReviewInputs describes the signal evidence that contributed to a decision.
type ReviewInputs struct {
	Signals []decision.SignalInput `json:"signals"`
	EventID string                 `json:"event_id,omitempty"`
	At      time.Time              `json:"at"`
}

// ReviewTransform describes the decision evaluation itself.
//
// String-filter semantics (H-6.c.2 commit 3, Decisão Q2 of H-6.c.1):
// the Symbol field is venue-native lowercase (e.g. "btcusdt") populated
// from the upstream domain type via d.VenueSymbol() at
// get_decision_review.go:188 — the canonical-derived display string
// projected from decision.Decision.Instrument (migrated in H-6.b).
// This DTO does NOT carry CanonicalInstrument by design: it is a
// query-result projection consumed by HTTP handlers + the decision
// triage downstream surface, where venue-native display is the
// canonical wire shape. Promoting to Instrument would force
// source-string reconstruction at the DTO boundary — the same
// regression-shape as commit 37f8ddd. The architectural decision
// is recorded in tools/raccoon-cli/policies/domain_types.toml under
// [domain_types.review_transform] with migration_state =
// "string_filter".
type ReviewTransform struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	// Symbol is the venue-native lowercase symbol form (e.g.
	// "btcusdt"), populated from the upstream decision.Decision via
	// d.VenueSymbol(). String-filter by design — see struct godoc.
	Symbol     string `json:"symbol"`
	Timeframe  int    `json:"timeframe"`
	Outcome    string `json:"outcome"`
	Severity   string `json:"severity"`
	Confidence string `json:"confidence"`
	Rationale  string `json:"rationale"`
	Final      bool   `json:"final"`

	EventID  string            `json:"event_id,omitempty"`
	At       time.Time         `json:"at"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ReviewResolution describes the strategy resolved from the decision.
type ReviewResolution struct {
	Type       string `json:"type"`
	Direction  string `json:"direction"`
	Confidence string `json:"confidence"`

	// DecisionInputs carried into the strategy — shows what the strategy saw.
	DecisionInputs []ReviewDecisionRef `json:"decision_inputs,omitempty"`
	Parameters     map[string]string   `json:"parameters,omitempty"`

	EventID string    `json:"event_id,omitempty"`
	At      time.Time `json:"at"`
}

// ReviewDecisionRef is a lightweight reference to a decision as seen by the strategy.
type ReviewDecisionRef struct {
	Type       string `json:"type"`
	Outcome    string `json:"outcome"`
	Confidence string `json:"confidence"`
	Severity   string `json:"severity"`
	Rationale  string `json:"rationale"`
}

// ReviewConstraints describes the risk assessment applied to the chain.
type ReviewConstraints struct {
	Type        string           `json:"type"`
	Disposition string           `json:"disposition"` // approved, modified, rejected
	Confidence  string           `json:"confidence"`
	Rationale   string           `json:"rationale"`
	Limits      risk.Constraints `json:"limits"`

	// StrategyContext shows what the risk evaluator saw.
	StrategyContext []AttributionStrategyContext `json:"strategy_context,omitempty"`

	EventID string    `json:"event_id,omitempty"`
	At      time.Time `json:"at"`
}

// ReviewOutput describes the execution intent and its lifecycle outcome.
type ReviewOutput struct {
	Type           string `json:"type"`
	Side           string `json:"side"`
	Quantity       string `json:"quantity"`
	FilledQuantity string `json:"filled_quantity"`
	Status         string `json:"status"`
	Final          bool   `json:"final"`

	CorrelationID string `json:"correlation_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`

	EventID string    `json:"event_id,omitempty"`
	At      time.Time `json:"at"`
}
