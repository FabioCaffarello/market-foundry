package strategy

import (
	"fmt"
	"strings"
	"time"

	"internal/domain/instrument"
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
//
// Per ADR-0021, the canonical instrument identity is carried in the
// Instrument field. Migrated from Symbol string in H-6.b.
type Strategy struct {
	Type       string                         `json:"type"`
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`
	Direction  Direction                      `json:"direction"`
	Confidence string                         `json:"confidence"`
	Decisions  []DecisionInput                `json:"decisions"`
	Parameters map[string]string              `json:"parameters"`
	Metadata   map[string]string              `json:"metadata"`
	Final      bool                           `json:"final"`
	Timestamp  time.Time                      `json:"timestamp"`
}

// VenueSymbol returns the lowercase venue-native symbol form.
//
// TRANSITORY ADAPTER (H-6.b → sunset H-6.f). See ADR-0021.
func (s Strategy) VenueSymbol() string {
	return strings.ToLower(string(s.Instrument.Base) + string(s.Instrument.Quote))
}

func (s Strategy) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if s.Type == "" {
		issues = append(issues, problem.ValidationIssue{Field: "type", Message: "must not be empty"})
	}
	if s.Source == "" {
		issues = append(issues, problem.ValidationIssue{Field: "source", Message: "must not be empty"})
	}
	if s.Instrument.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "instrument", Message: "must not be zero"})
	} else if prob := s.Instrument.Validate(); prob != nil {
		return prob
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

// PartitionKey returns the key used for KV bucket entries:
// "{source}.{venuesymbol}.{timeframe}". Preserves H-6.b back-compat.
func (s Strategy) PartitionKey() string {
	return fmt.Sprintf("%s.%s.%d", s.Source, s.VenueSymbol(), s.Timeframe)
}

// DeduplicationKey returns a unique key for JetStream deduplication.
// Nanosecond precision is required so multiple resolutions within the same
// wall-clock second produce distinct Nats-Msg-Id values. With whole-second
// precision (the prior behaviour) JetStream's 2-minute Duplicate Window
// silently dropped same-second siblings — diagnosed in P4.1.9.
// Production kline cadence is ≥1s so the bug was latent, but rapid-publish
// integration tests and any future sub-second producers depend on this.
func (s Strategy) DeduplicationKey() string {
	return fmt.Sprintf("strat:%s:%s:%s:%d:%d", s.Type, s.Source, s.VenueSymbol(), s.Timeframe, s.Timestamp.UnixNano())
}
