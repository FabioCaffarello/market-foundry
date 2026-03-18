package signal

import (
	"fmt"
	"time"

	"internal/shared/problem"
)

// Signal represents a derived interpretation of evidence.
// All signal families share this struct; type-specific fields live in Metadata.
// Value is always a decimal string representing the primary indicator output.
type Signal struct {
	Type      string            `json:"type"`      // e.g., "rsi", "macd"
	Source    string            `json:"source"`     // Exchange identifier (e.g., "binancef")
	Symbol   string            `json:"symbol"`     // Trading pair, lowercase (e.g., "btcusdt")
	Timeframe int              `json:"timeframe"`  // Evidence window duration in seconds
	Value     string            `json:"value"`      // Decimal string — primary signal output
	Metadata  map[string]string `json:"metadata"`   // Type-specific fields (e.g., period, avg_gain, avg_loss for RSI)
	Final     bool              `json:"final"`      // True = finalized signal; false = interim
	Timestamp time.Time         `json:"timestamp"`  // When this signal was computed
}

func (s Signal) Validate() *problem.Problem {
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
	if s.Value == "" {
		issues = append(issues, problem.ValidationIssue{Field: "value", Message: "must not be empty"})
	}
	if s.Timestamp.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "timestamp", Message: "must not be zero"})
	}

	if len(issues) == 0 {
		return nil
	}
	return problem.Validation(problem.InvalidArgument, "signal is invalid", issues...)
}

// PartitionKey returns the key used for KV bucket entries: "{source}.{symbol}.{timeframe}".
func (s Signal) PartitionKey() string {
	return fmt.Sprintf("%s.%s.%d", s.Source, s.Symbol, s.Timeframe)
}

// DeduplicationKey returns a unique key for JetStream deduplication.
func (s Signal) DeduplicationKey() string {
	return fmt.Sprintf("sig:%s:%s:%s:%d:%d", s.Type, s.Source, s.Symbol, s.Timeframe, s.Timestamp.Unix())
}
