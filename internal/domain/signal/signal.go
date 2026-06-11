package signal

import (
	"fmt"
	"strings"
	"time"

	"internal/domain/instrument"
	"internal/shared/problem"
)

// Signal represents a derived interpretation of evidence.
// All signal families share this struct; type-specific fields live in Metadata.
// Value is always a decimal string representing the primary indicator output.
//
// Per ADR-0021, the canonical instrument identity is carried in the
// Instrument field. Migrated from Symbol string in H-6.b.
type Signal struct {
	Type       string                         `json:"type"`       // e.g., "rsi", "macd"
	Source     string                         `json:"source"`     // Exchange identifier (e.g., "binancef")
	Instrument instrument.CanonicalInstrument `json:"instrument"` // Canonical instrument identity (ADR-0021)
	Timeframe  int                            `json:"timeframe"`  // Evidence window duration in seconds
	Value      string                         `json:"value"`      // Decimal string — primary signal output
	Metadata   map[string]string              `json:"metadata"`   // Type-specific fields (e.g., period, avg_gain, avg_loss for RSI)
	Final      bool                           `json:"final"`      // True = finalized signal; false = interim
	Timestamp  time.Time                      `json:"timestamp"`  // When this signal was computed
}

// VenueSymbol returns the lowercase venue-native symbol form
// (e.g., "btcusdt") derived from the canonical instrument.
//
// TRANSITORY ADAPTER (H-6.b → sunset H-6.f). See ADR-0021.
func (s Signal) VenueSymbol() string {
	return strings.ToLower(string(s.Instrument.Base) + string(s.Instrument.Quote))
}

func (s Signal) Validate() *problem.Problem {
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

// PartitionKey returns the key used for KV bucket entries:
// "{source}.{subject_token}.{timeframe}" — canonical token via
// SubjectToken() since H-6.e.2 (read side composes the same shape).
//
// Per ADR-0021 H-6.b: the partition key shape is preserved
// (venue-native symbol form) via VenueSymbol() so the existing
// KV bucket layout stays back-compatible. H-6.e decides whether
// the canonical form replaces the venue-native form here.
func (s Signal) PartitionKey() string {
	return fmt.Sprintf("%s.%s.%d", s.Source, s.Instrument.SubjectToken(), s.Timeframe)
}

// DeduplicationKey returns a unique key for JetStream deduplication.
// Nanosecond precision (see P4.1.10 — Strategy.DeduplicationKey doc).
// Canonical SubjectToken() since H-6.f.1 (Decisão #4).
func (s Signal) DeduplicationKey() string {
	return fmt.Sprintf("sig:%s:%s:%s:%d:%d", s.Type, s.Source, s.Instrument.SubjectToken(), s.Timeframe, s.Timestamp.UnixNano())
}
