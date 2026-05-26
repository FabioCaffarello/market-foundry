package observation

import (
	"strings"
	"time"

	"internal/domain/instrument"
	"internal/shared/problem"
)

// ObservationTrade represents a normalized trade event captured from
// an external market source. Price and Quantity are decimal strings
// to avoid IEEE 754 precision loss.
//
// Per ADR-0021, the canonical instrument identity is carried in the
// Instrument field. Adapters translate venue-native symbol shapes
// into CanonicalInstrument at the layer boundary; downstream code
// must reason about the structured form, not raw symbol strings.
type ObservationTrade struct {
	Source     string                         `json:"source"`      // Exchange identifier (e.g., "binancef")
	Instrument instrument.CanonicalInstrument `json:"instrument"`  // Canonical instrument identity (ADR-0021)
	Price      string                         `json:"price"`       // Decimal string
	Quantity   string                         `json:"quantity"`    // Decimal string
	TradeID    string                         `json:"trade_id"`    // Source-assigned trade identifier
	BuyerMaker bool                           `json:"buyer_maker"` // True if the buyer is the maker
	Timestamp  time.Time                      `json:"timestamp"`   // Exchange-reported trade time
}

// VenueSymbol returns the lowercase venue-native symbol form
// (e.g., "btcusdt") derived from the canonical instrument's Base
// and Quote tickers.
//
// TRANSITORY ADAPTER (H-6.a): this method exists to keep callers
// in the ingest/derive pipeline compiling while the migration to
// CanonicalInstrument propagates. It is slated for sunset in
// H-6.f, when the last reader of a venue-native symbol string is
// removed in favor of either canonical reasoning or explicit
// venue routing metadata.
//
// Known limitation: the derivation is lossy for delivery futures
// (a BTCUSDT_240329 contract collapses to "btcusdt"). This is
// acceptable in H-6.a because no delivery-futures contracts ride
// the current NATS routing path; H-6.e revisits the NATS routing
// shape and resolves whether the routing key should carry the
// canonical symbol form or stay venue-native.
func (t ObservationTrade) VenueSymbol() string {
	return strings.ToLower(string(t.Instrument.Base) + string(t.Instrument.Quote))
}

func (t ObservationTrade) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if t.Source == "" {
		issues = append(issues, problem.ValidationIssue{Field: "source", Message: "must not be empty"})
	}
	if t.Instrument.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "instrument", Message: "must not be zero"})
	} else if prob := t.Instrument.Validate(); prob != nil {
		return prob
	}
	if t.Price == "" {
		issues = append(issues, problem.ValidationIssue{Field: "price", Message: "must not be empty"})
	}
	if t.Quantity == "" {
		issues = append(issues, problem.ValidationIssue{Field: "quantity", Message: "must not be empty"})
	}
	if t.TradeID == "" {
		issues = append(issues, problem.ValidationIssue{Field: "trade_id", Message: "must not be empty"})
	}
	if t.Timestamp.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "timestamp", Message: "must not be zero"})
	}

	if len(issues) == 0 {
		return nil
	}
	return problem.Validation(problem.InvalidArgument, "observation trade is invalid", issues...)
}

// DeduplicationKey returns the NATS Msg-Id for JetStream deduplication.
func (t ObservationTrade) DeduplicationKey() string {
	return t.Source + ":" + t.TradeID
}
