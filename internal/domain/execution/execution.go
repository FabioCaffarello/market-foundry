package execution

import (
	"fmt"
	"strings"
	"time"

	"internal/domain/instrument"
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

// FeeSource indicates the provenance of fee data in a FillRecord.
// S499: Enables downstream logic to distinguish why a fee is zero.
type FeeSource string

const (
	// FeeSourceVenue indicates real commission data from the exchange.
	FeeSourceVenue FeeSource = "venue"

	// FeeSourceUnavailable indicates the venue API did not return commission
	// (e.g. Binance Futures RESULT response). Fee=0 is expected, not a gap.
	FeeSourceUnavailable FeeSource = "unavailable"

	// FeeSourceSimulated indicates a paper/dry-run fill with no real fee.
	FeeSourceSimulated FeeSource = "simulated"

	// FeeSourceFallback indicates a venue fill where the fills[] array was
	// empty (unexpected for FULL response type). Fee=0 may be a data gap.
	FeeSourceFallback FeeSource = "fallback"
)

// FillRecord represents a single fill event within an execution.
//
// S428 fee normalization: Fee is the actual trading commission charged by the venue.
// CostBasis is the total notional value of the fill (price * quantity or cumQuote).
// FeeAsset identifies the denomination of the fee (e.g. "BNB", "USDT").
//
// S499 fee provenance: FeeSource indicates why Fee has its current value,
// enabling downstream reconciliation to distinguish expected zeros from gaps.
//
// Semantics by segment:
//   - Spot: Fee = aggregated commission from fills[], FeeAsset = commissionAsset, CostBasis = cummulativeQuoteQty, FeeSource = venue
//   - Futures: Fee = "0" (not available from RESULT response), FeeAsset = "", CostBasis = cumQuote, FeeSource = unavailable
//   - Paper/DryRun: Fee = "0", FeeAsset = "", CostBasis = "0", FeeSource = simulated
type FillRecord struct {
	Price     string    `json:"price"`
	Quantity  string    `json:"quantity"`
	Fee       string    `json:"fee"`
	FeeAsset  string    `json:"fee_asset,omitempty"`
	CostBasis string    `json:"cost_basis,omitempty"`
	FeeSource FeeSource `json:"fee_source,omitempty"`
	Simulated bool      `json:"simulated"`
	Timestamp time.Time `json:"timestamp"`
}

// RiskInput records which risk assessment contributed to this execution intent.
// This is an execution-owned type — it does not import from the risk domain.
// S265: StrategyType and DecisionSeverity added to preserve full causal context
// across the risk→execution boundary for traceability and behavioral analysis.
// S470: EventID added to make the causal reference to the originating risk event explicit.
type RiskInput struct {
	Type             string `json:"type"`
	Disposition      string `json:"disposition"`
	Confidence       string `json:"confidence"`
	Timeframe        int    `json:"timeframe"`
	StrategyType     string `json:"strategy_type,omitempty"`
	DecisionSeverity string `json:"decision_severity,omitempty"`
	EventID          string `json:"event_id,omitempty"`
}

// ExecutionIntent represents a discrete, typed execution intent derived from a risk assessment.
//
// Per ADR-0021, the canonical instrument identity is carried in the
// Instrument field. Migrated from Symbol string in H-6.b'.
type ExecutionIntent struct {
	Type           string                         `json:"type"`
	Source         string                         `json:"source"`
	Instrument     instrument.CanonicalInstrument `json:"instrument"`
	Timeframe      int                            `json:"timeframe"`
	Side           Side                           `json:"side"`
	Quantity       string                         `json:"quantity"`
	FilledQuantity string                         `json:"filled_quantity"`
	Status         Status                         `json:"status"`
	Risk           RiskInput                      `json:"risk"`
	Fills          []FillRecord                   `json:"fills"`
	Parameters     map[string]string              `json:"parameters"`
	Metadata       map[string]string              `json:"metadata"`
	CorrelationID  string                         `json:"correlation_id,omitempty"`
	CausationID    string                         `json:"causation_id,omitempty"`
	Final          bool                           `json:"final"`
	Timestamp      time.Time                      `json:"timestamp"`
}

// VenueSymbol returns the lowercase venue-native symbol form.
//
// TRANSITORY ADAPTER (H-6.b' → sunset H-6.f). See ADR-0021.
func (e ExecutionIntent) VenueSymbol() string {
	return strings.ToLower(string(e.Instrument.Base) + string(e.Instrument.Quote))
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
	if e.Instrument.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "instrument", Message: "must not be zero"})
	} else if prob := e.Instrument.Validate(); prob != nil {
		return prob
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

// PartitionKey returns the key used for KV bucket entries:
// "{source}.{subject_token}.{timeframe}" — the canonical token via
// SubjectToken() since H-6.e.2 (read side composes the same shape;
// pre-cutover keys in the old venue-native shape are inert orphans).
func (e ExecutionIntent) PartitionKey() string {
	return fmt.Sprintf("%s.%s.%d", e.Source, e.Instrument.SubjectToken(), e.Timeframe)
}

// DeduplicationKey returns a unique key for JetStream deduplication.
// Nanosecond precision mirrors Strategy.DeduplicationKey (see P4.1.10):
// whole-second precision causes silent JetStream Duplicate-Window drops
// when siblings publish within the same wall-clock second. Production
// is safe (kline cadence ≥1s) but rapid-publish integration tests
// (writerpipeline + natsexecution restart_recovery) require precision.
// Canonical SubjectToken() since H-6.f.1 (Decisão #4) — last message
// surface off VenueSymbol(); transition breaks the 2-minute JetStream
// duplicate window once at deploy (documented, accepted).
func (e ExecutionIntent) DeduplicationKey() string {
	return fmt.Sprintf("exec:%s:%s:%s:%d:%d", e.Type, e.Source, e.Instrument.SubjectToken(), e.Timeframe, e.Timestamp.UnixNano())
}
