package execution

import (
	"fmt"
	"time"

	"internal/shared/problem"
)

// Side represents the order side for an execution intent.
type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
	SideNone Side = "none"
)

// ValidSide reports whether s is a recognized side value.
func ValidSide(s Side) bool {
	return s == SideBuy || s == SideSell || s == SideNone
}

// Status represents the lifecycle status of an execution intent.
type Status string

const (
	StatusSubmitted       Status = "submitted"
	StatusSent            Status = "sent"
	StatusAccepted        Status = "accepted"
	StatusFilled          Status = "filled"
	StatusPartiallyFilled Status = "partially_filled"
	StatusRejected        Status = "rejected"
	StatusCancelled       Status = "cancelled"
)

// ValidStatus reports whether st is a recognized status value.
func ValidStatus(st Status) bool {
	switch st {
	case StatusSubmitted, StatusSent, StatusAccepted, StatusFilled,
		StatusPartiallyFilled, StatusRejected, StatusCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal reports whether st is a terminal lifecycle status.
// Terminal states cannot transition to any other state.
func (st Status) IsTerminal() bool {
	return st == StatusFilled || st == StatusRejected || st == StatusCancelled
}

// validTransitions defines the allowed state transitions for execution lifecycle.
var validTransitions = map[Status][]Status{
	StatusSubmitted:       {StatusSent, StatusAccepted, StatusRejected},
	StatusSent:            {StatusAccepted, StatusRejected},
	StatusAccepted:        {StatusFilled, StatusPartiallyFilled, StatusCancelled},
	StatusPartiallyFilled: {StatusFilled, StatusCancelled},
}

// ValidTransition reports whether transitioning from → to is allowed.
func ValidTransition(from, to Status) bool {
	targets, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// FillRecord represents a single fill event within an execution.
type FillRecord struct {
	Price     string    `json:"price"`
	Quantity  string    `json:"quantity"`
	Fee       string    `json:"fee"`
	Simulated bool      `json:"simulated"`
	Timestamp time.Time `json:"timestamp"`
}

// RiskInput records which risk assessment contributed to this execution intent.
// This is an execution-owned type — it does not import from the risk domain.
type RiskInput struct {
	Type        string `json:"type"`
	Disposition string `json:"disposition"`
	Confidence  string `json:"confidence"`
	Timeframe   int    `json:"timeframe"`
}

// ExecutionIntent represents a discrete, typed execution intent derived from a risk assessment.
type ExecutionIntent struct {
	Type           string            `json:"type"`
	Source         string            `json:"source"`
	Symbol         string            `json:"symbol"`
	Timeframe      int               `json:"timeframe"`
	Side           Side              `json:"side"`
	Quantity       string            `json:"quantity"`
	FilledQuantity string            `json:"filled_quantity"`
	Status         Status            `json:"status"`
	Risk           RiskInput         `json:"risk"`
	Fills          []FillRecord      `json:"fills"`
	Parameters     map[string]string `json:"parameters"`
	Metadata       map[string]string `json:"metadata"`
	CorrelationID  string            `json:"correlation_id,omitempty"`
	CausationID    string            `json:"causation_id,omitempty"`
	Final          bool              `json:"final"`
	Timestamp      time.Time         `json:"timestamp"`
}

// Validate checks that an ExecutionIntent has all required fields populated with valid values.
func (e ExecutionIntent) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if e.Type == "" {
		issues = append(issues, problem.ValidationIssue{Field: "type", Message: "must not be empty"})
	}
	if e.Source == "" {
		issues = append(issues, problem.ValidationIssue{Field: "source", Message: "must not be empty"})
	}
	if e.Symbol == "" {
		issues = append(issues, problem.ValidationIssue{Field: "symbol", Message: "must not be empty"})
	}
	if e.Timeframe <= 0 {
		issues = append(issues, problem.ValidationIssue{Field: "timeframe", Message: "must be a positive integer"})
	}
	if e.Side == "" {
		issues = append(issues, problem.ValidationIssue{Field: "side", Message: "must not be empty"})
	}
	if !ValidSide(e.Side) {
		if e.Side != "" {
			issues = append(issues, problem.ValidationIssue{Field: "side", Message: "must be one of buy, sell, none"})
		}
	}
	if e.Status != "" && !ValidStatus(e.Status) {
		issues = append(issues, problem.ValidationIssue{Field: "status", Message: "must be a valid lifecycle status"})
	}
	if e.Status == "" {
		issues = append(issues, problem.ValidationIssue{Field: "status", Message: "must not be empty"})
	}
	if e.Quantity == "" {
		issues = append(issues, problem.ValidationIssue{Field: "quantity", Message: "must not be empty"})
	}
	if e.Risk.Type == "" {
		issues = append(issues, problem.ValidationIssue{Field: "risk.type", Message: "must not be empty"})
	}
	if e.Risk.Disposition == "" {
		issues = append(issues, problem.ValidationIssue{Field: "risk.disposition", Message: "must not be empty"})
	}
	if e.Timestamp.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "timestamp", Message: "must not be zero"})
	}

	if len(issues) == 0 {
		return nil
	}
	return problem.Validation(problem.InvalidArgument, "execution intent is invalid", issues...)
}

// PartitionKey returns the key used for KV bucket entries: "{source}.{symbol}.{timeframe}".
func (e ExecutionIntent) PartitionKey() string {
	return fmt.Sprintf("%s.%s.%d", e.Source, e.Symbol, e.Timeframe)
}

// DeduplicationKey returns a unique key for JetStream deduplication.
func (e ExecutionIntent) DeduplicationKey() string {
	return fmt.Sprintf("exec:%s:%s:%s:%d:%d", e.Type, e.Source, e.Symbol, e.Timeframe, e.Timestamp.Unix())
}
