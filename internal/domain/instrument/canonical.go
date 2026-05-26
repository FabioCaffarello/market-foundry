package instrument

import (
	"strings"

	"internal/shared/problem"
)

// CanonicalInstrument is the foundry-internal instrument identity.
// Identical structure across every venue; venue-native nuances
// (lot sizes, tick sizes, listing dates) live in adapter-side
// metadata and are not part of the canonical identity.
//
// Per ADR-0021, Venue is intentionally NOT a field of
// CanonicalInstrument — the same instrument identity is shared
// across venues for cross-venue capabilities, and Venue lives at
// the envelope level (ADR-0017) for routing.
type CanonicalInstrument struct {
	Base     BaseAsset    `json:"base"`
	Quote    QuoteAsset   `json:"quote"`
	Contract ContractType `json:"contract"`
}

// New constructs a CanonicalInstrument from raw asset and
// contract strings, normalizing assets (trim + uppercase) and
// validating each field. Returns a Problem if any field is
// invalid.
//
// Adapters use this entry point to translate venue-native
// symbol shapes into the canonical form at the layer boundary.
func New(base, quote string, contract ContractType) (CanonicalInstrument, *problem.Problem) {
	b, prob := NewBaseAsset(base)
	if prob != nil {
		return CanonicalInstrument{}, prob
	}
	q, prob := NewQuoteAsset(quote)
	if prob != nil {
		return CanonicalInstrument{}, prob
	}
	if prob := contract.Validate(); prob != nil {
		return CanonicalInstrument{}, prob
	}
	return CanonicalInstrument{Base: b, Quote: q, Contract: contract}, nil
}

// Symbol returns the canonical string representation:
//
//	"{BASE}/{QUOTE}-{CONTRACT}"
//
// Examples:
//   - "BTC/USDT-spot"
//   - "ETH/USDT-perpetual"
//   - "BTC/USD-coinfutures"
//
// This is the form embedded in the envelope's `instrument`
// string field (ADR-0017) and in the Sequencer's stream key
// (ADR-0020). The format is stable; downstream consumers may
// parse via FromSymbol.
func (c CanonicalInstrument) Symbol() string {
	return string(c.Base) + "/" + string(c.Quote) + "-" + string(c.Contract)
}

// Validate reports the first invalid field (if any). Validation
// is performed on the typed components; CanonicalInstrument
// values constructed via New are already valid, so this method
// is most useful for values produced via struct-literal
// construction in tests.
func (c CanonicalInstrument) Validate() *problem.Problem {
	if prob := c.Base.Validate(); prob != nil {
		return prob
	}
	if prob := c.Quote.Validate(); prob != nil {
		return prob
	}
	if prob := c.Contract.Validate(); prob != nil {
		return prob
	}
	return nil
}

// IsZero reports whether the instrument is the zero value.
// Useful for detecting unset fields in structs that embed
// CanonicalInstrument and need a "not yet populated" check.
func (c CanonicalInstrument) IsZero() bool {
	return c.Base == "" && c.Quote == "" && c.Contract == ""
}

// FromSymbol parses a canonical symbol string back into a
// CanonicalInstrument. Returns a Problem if the string does not
// match the expected "{BASE}/{QUOTE}-{CONTRACT}" shape or any
// component fails validation.
//
// Used by downstream consumers that receive the envelope's
// `instrument` string field (per ADR-0017) and need to reason
// about the structured form.
func FromSymbol(symbol string) (CanonicalInstrument, *problem.Problem) {
	s := strings.TrimSpace(symbol)
	if s == "" {
		return CanonicalInstrument{}, problem.Validation(
			problem.InvalidArgument,
			"canonical symbol is invalid",
			problem.ValidationIssue{Field: "symbol", Message: "must not be empty"},
		)
	}
	dashIdx := strings.LastIndex(s, "-")
	if dashIdx <= 0 || dashIdx == len(s)-1 {
		return CanonicalInstrument{}, problem.Validation(
			problem.InvalidArgument,
			"canonical symbol is invalid",
			problem.ValidationIssue{
				Field:   "symbol",
				Message: "must contain a contract separator '-' between non-empty parts",
				Value:   symbol,
			},
		)
	}
	pair := s[:dashIdx]
	contract := ContractType(s[dashIdx+1:])

	slashIdx := strings.Index(pair, "/")
	if slashIdx <= 0 || slashIdx == len(pair)-1 {
		return CanonicalInstrument{}, problem.Validation(
			problem.InvalidArgument,
			"canonical symbol is invalid",
			problem.ValidationIssue{
				Field:   "symbol",
				Message: "must contain a pair separator '/' between non-empty parts",
				Value:   symbol,
			},
		)
	}
	base := pair[:slashIdx]
	quote := pair[slashIdx+1:]

	return New(base, quote, contract)
}
