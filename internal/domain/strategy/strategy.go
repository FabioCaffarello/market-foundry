package strategy

import (
	"fmt"
	"time"

	"internal/shared/problem"
)

// Direction represents the positional intent of a strategy resolution.
type Direction string

const (
	DirectionLong  Direction = "long"
	DirectionShort Direction = "short"
	DirectionFlat  Direction = "flat"
)

// DecisionInput records which decision contributed to this strategy.
// This is a strategy-owned type — it does not import from the decision domain.
// Severity and Rationale carry the decision's semantic depth forward for traceability.
// S470: EventID added to make the causal reference to the originating decision event explicit.
type DecisionInput struct {
	Type       string `json:"type"`
	Outcome    string `json:"outcome"`
	Confidence string `json:"confidence"`
	Severity   string `json:"severity"`
	Rationale  string `json:"rationale"`
	Timeframe  int    `json:"timeframe"`
	EventID    string `json:"event_id,omitempty"`
}

// Strategy represents a discrete, typed resolution combining decisions into a directional intent.
// All strategy families share this struct; type-specific fields live in Parameters and Metadata.
type Strategy struct {
	Type       string            `json:"type"`
	Source     string            `json:"source"`
	Symbol     string            `json:"symbol"`
	Timeframe  int               `json:"timeframe"`
	Direction  Direction         `json:"direction"`
	Confidence string            `json:"confidence"`
	Decisions  []DecisionInput   `json:"decisions"`
	Parameters map[string]string `json:"parameters"`
	Metadata   map[string]string `json:"metadata"`
	Final      bool              `json:"final"`
	Timestamp  time.Time         `json:"timestamp"`
}

func (s Strategy) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if s.Type == "" {
		issues = append(issues, problem.ValidationIssue{Field: "type", Message: "must not be empty"})
	}
	if s.Source == "" {
		issues = append(issues, problem.ValidationIssue{Field: "source", Message: "must not be empty"})
	}
	if s.Symbol == "" {
		issues = append(issues, problem.ValidationIssue{Field: "symbol", Message: "must not be empty"})
	}
	if s.Timeframe <= 0 {
		issues = append(issues, problem.ValidationIssue{Field: "timeframe", Message: "must be a positive integer"})
	}
	if s.Direction == "" {
		issues = append(issues, problem.ValidationIssue{Field: "direction", Message: "must not be empty"})
	}
	if s.Direction != DirectionLong && s.Direction != DirectionShort && s.Direction != DirectionFlat {
		if s.Direction != "" {
			issues = append(issues, problem.ValidationIssue{Field: "direction", Message: "must be one of long, short, flat"})
		}
	}
	if s.Confidence == "" {
		issues = append(issues, problem.ValidationIssue{Field: "confidence", Message: "must not be empty"})
	}
	if s.Timestamp.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "timestamp", Message: "must not be zero"})
	}
	if len(s.Decisions) == 0 {
		issues = append(issues, problem.ValidationIssue{Field: "decisions", Message: "at least one decision input is required"})
	}

	if len(issues) == 0 {
		return nil
	}
	return problem.Validation(problem.InvalidArgument, "strategy is invalid", issues...)
}

// PartitionKey returns the key used for KV bucket entries: "{source}.{symbol}.{timeframe}".
func (s Strategy) PartitionKey() string {
	return fmt.Sprintf("%s.%s.%d", s.Source, s.Symbol, s.Timeframe)
}

// DeduplicationKey returns a unique key for JetStream deduplication.
func (s Strategy) DeduplicationKey() string {
	return fmt.Sprintf("strat:%s:%s:%s:%d:%d", s.Type, s.Source, s.Symbol, s.Timeframe, s.Timestamp.Unix())
}
