package risk

import (
	"fmt"
	"time"

	"internal/shared/problem"
)

// Disposition represents the outcome of a risk assessment.
type Disposition string

const (
	DispositionApproved Disposition = "approved"
	DispositionModified Disposition = "modified"
	DispositionRejected Disposition = "rejected"
)

// StrategyInput records which strategy contributed to this risk assessment.
// This is a risk-owned type — it does not import from the strategy domain.
// DecisionSeverity and DecisionRationale carry the originating decision's semantic
// depth forward for traceability. As of S251, strategy Type and DecisionSeverity
// actively influence risk confidence multipliers, position limits, and drawdown
// tolerance via strategy-type-aware and severity-aware scaling in the evaluators.
// S470: EventID added to make the causal reference to the originating strategy event explicit.
type StrategyInput struct {
	Type              string `json:"type"`
	Direction         string `json:"direction"`
	Confidence        string `json:"confidence"`
	Timeframe         int    `json:"timeframe"`
	DecisionSeverity  string `json:"decision_severity,omitempty"`
	DecisionRationale string `json:"decision_rationale,omitempty"`
	EventID           string `json:"event_id,omitempty"`
}

// Constraints holds the risk-imposed limits on a position.
type Constraints struct {
	MaxPositionSize string `json:"max_position_size,omitempty"`
	MaxExposure     string `json:"max_exposure,omitempty"`
	StopDistance    string `json:"stop_distance,omitempty"`
}

// RiskAssessment represents a discrete, typed risk evaluation applied to a strategy intent.
// All risk families share this struct; type-specific fields live in Parameters and Metadata.
type RiskAssessment struct {
	Type        string            `json:"type"`
	Source      string            `json:"source"`
	Symbol      string            `json:"symbol"`
	Timeframe   int               `json:"timeframe"`
	Disposition Disposition       `json:"disposition"`
	Confidence  string            `json:"confidence"`
	Strategies  []StrategyInput   `json:"strategies"`
	Constraints Constraints       `json:"constraints"`
	Rationale   string            `json:"rationale"`
	Parameters  map[string]string `json:"parameters"`
	Metadata    map[string]string `json:"metadata"`
	Final       bool              `json:"final"`
	Timestamp   time.Time         `json:"timestamp"`
}

func (r RiskAssessment) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if r.Type == "" {
		issues = append(issues, problem.ValidationIssue{Field: "type", Message: "must not be empty"})
	}
	if r.Source == "" {
		issues = append(issues, problem.ValidationIssue{Field: "source", Message: "must not be empty"})
	}
	if r.Symbol == "" {
		issues = append(issues, problem.ValidationIssue{Field: "symbol", Message: "must not be empty"})
	}
	if r.Timeframe <= 0 {
		issues = append(issues, problem.ValidationIssue{Field: "timeframe", Message: "must be a positive integer"})
	}
	if r.Disposition == "" {
		issues = append(issues, problem.ValidationIssue{Field: "disposition", Message: "must not be empty"})
	}
	if r.Disposition != DispositionApproved && r.Disposition != DispositionModified && r.Disposition != DispositionRejected {
		if r.Disposition != "" {
			issues = append(issues, problem.ValidationIssue{Field: "disposition", Message: "must be one of approved, modified, rejected"})
		}
	}
	if r.Confidence == "" {
		issues = append(issues, problem.ValidationIssue{Field: "confidence", Message: "must not be empty"})
	}
	if r.Rationale == "" {
		issues = append(issues, problem.ValidationIssue{Field: "rationale", Message: "must not be empty"})
	}
	if r.Timestamp.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "timestamp", Message: "must not be zero"})
	}
	if len(r.Strategies) == 0 {
		issues = append(issues, problem.ValidationIssue{Field: "strategies", Message: "at least one strategy input is required"})
	}

	if len(issues) == 0 {
		return nil
	}
	return problem.Validation(problem.InvalidArgument, "risk assessment is invalid", issues...)
}

// PartitionKey returns the key used for KV bucket entries: "{source}.{symbol}.{timeframe}".
func (r RiskAssessment) PartitionKey() string {
	return fmt.Sprintf("%s.%s.%d", r.Source, r.Symbol, r.Timeframe)
}

// DeduplicationKey returns a unique key for JetStream deduplication.
// Nanosecond precision (see P4.1.10 — Strategy.DeduplicationKey doc).
func (r RiskAssessment) DeduplicationKey() string {
	return fmt.Sprintf("risk:%s:%s:%s:%d:%d", r.Type, r.Source, r.Symbol, r.Timeframe, r.Timestamp.UnixNano())
}
