package decision

import (
	"fmt"
	"time"

	"internal/shared/problem"
)

// Outcome represents the categorical result of a decision evaluation.
type Outcome string

const (
	OutcomeTriggered    Outcome = "triggered"
	OutcomeNotTriggered Outcome = "not_triggered"
	OutcomeInsufficient Outcome = "insufficient"
)

// Severity classifies how strong or extreme the evaluated condition is.
// For triggered decisions, higher severity means a more extreme signal.
// For not_triggered decisions, severity is always "none".
type Severity string

const (
	SeverityNone     Severity = "none"
	SeverityLow      Severity = "low"
	SeverityModerate Severity = "moderate"
	SeverityHigh     Severity = "high"
)

// SignalInput records which signal contributed to this decision.
// This is a decision-owned type — it does not import from the signal domain.
// S470: EventID added to make the causal reference to the originating signal event explicit.
type SignalInput struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Timeframe int    `json:"timeframe"`
	EventID   string `json:"event_id,omitempty"`
}

// Decision represents a discrete, typed evaluation combining signals into a categorical judgment.
// All decision families share this struct; type-specific fields live in Metadata.
type Decision struct {
	Type       string            `json:"type"`
	Source     string            `json:"source"`
	Symbol     string            `json:"symbol"`
	Timeframe  int               `json:"timeframe"`
	Outcome    Outcome           `json:"outcome"`
	Severity   Severity          `json:"severity"`
	Confidence string            `json:"confidence"`
	Rationale  string            `json:"rationale"`
	Signals    []SignalInput     `json:"signals"`
	Metadata   map[string]string `json:"metadata"`
	Final      bool              `json:"final"`
	Timestamp  time.Time         `json:"timestamp"`
}

func (d Decision) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if d.Type == "" {
		issues = append(issues, problem.ValidationIssue{Field: "type", Message: "must not be empty"})
	}
	if d.Source == "" {
		issues = append(issues, problem.ValidationIssue{Field: "source", Message: "must not be empty"})
	}
	if d.Symbol == "" {
		issues = append(issues, problem.ValidationIssue{Field: "symbol", Message: "must not be empty"})
	}
	if d.Timeframe <= 0 {
		issues = append(issues, problem.ValidationIssue{Field: "timeframe", Message: "must be a positive integer"})
	}
	if d.Outcome == "" {
		issues = append(issues, problem.ValidationIssue{Field: "outcome", Message: "must not be empty"})
	}
	if d.Outcome != OutcomeTriggered && d.Outcome != OutcomeNotTriggered && d.Outcome != OutcomeInsufficient {
		if d.Outcome != "" {
			issues = append(issues, problem.ValidationIssue{Field: "outcome", Message: "must be one of triggered, not_triggered, insufficient"})
		}
	}
	if d.Severity != SeverityNone && d.Severity != SeverityLow && d.Severity != SeverityModerate && d.Severity != SeverityHigh {
		if d.Severity != "" {
			issues = append(issues, problem.ValidationIssue{Field: "severity", Message: "must be one of none, low, moderate, high"})
		}
	}
	if d.Confidence == "" {
		issues = append(issues, problem.ValidationIssue{Field: "confidence", Message: "must not be empty"})
	}
	if d.Timestamp.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "timestamp", Message: "must not be zero"})
	}

	if len(issues) == 0 {
		return nil
	}
	return problem.Validation(problem.InvalidArgument, "decision is invalid", issues...)
}

// PartitionKey returns the key used for KV bucket entries: "{source}.{symbol}.{timeframe}".
func (d Decision) PartitionKey() string {
	return fmt.Sprintf("%s.%s.%d", d.Source, d.Symbol, d.Timeframe)
}

// DeduplicationKey returns a unique key for JetStream deduplication.
func (d Decision) DeduplicationKey() string {
	return fmt.Sprintf("dec:%s:%s:%s:%d:%d", d.Type, d.Source, d.Symbol, d.Timeframe, d.Timestamp.Unix())
}
