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
	// Expiry is the settlement date of a dated futures contract in
	// canonical YYMMDD form (e.g. "240329"), or empty for contracts
	// without expiry. Optional field added in H-7.c (ADR-0021
	// erratum 2026-06-12, closing gap G10): permitted ONLY for the
	// dated contract classes (usdtfutures/coinfutures). Empty stays
	// legal for those classes too (pre-H-7.c constructions carry
	// the collapsed-identity caveat of G10). Instruments without
	// expiry keep byte-identical Symbol()/SubjectToken() forms.
	Expiry string `json:"expiry,omitempty"`
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

// NewDelivery constructs a dated-futures CanonicalInstrument:
// New() semantics plus a canonical YYMMDD expiry. Returns a Problem
// when the contract class does not carry expiry (spot/perpetual)
// or the expiry is not exactly six digits. Adapters use this entry
// point when the venue-native symbol carries a delivery date
// (e.g. Binance "BTCUSDT_240329").
func NewDelivery(base, quote string, contract ContractType, expiry string) (CanonicalInstrument, *problem.Problem) {
	inst, prob := New(base, quote, contract)
	if prob != nil {
		return CanonicalInstrument{}, prob
	}
	if prob := validateExpiry(expiry, contract); prob != nil {
		return CanonicalInstrument{}, prob
	}
	inst.Expiry = expiry
	return inst, nil
}

// validateExpiry enforces the canonical expiry shape (ADR-0021
// erratum 2026-06-12): exactly six ASCII digits (YYMMDD), permitted
// only for the dated contract classes. Empty expiry is validated by
// Validate() (legal everywhere); this helper assumes non-empty.
func validateExpiry(expiry string, contract ContractType) *problem.Problem {
	if contract != ContractUSDTFutures && contract != ContractCoinFutures {
		return problem.Validation(
			problem.InvalidArgument,
			"canonical instrument is invalid",
			problem.ValidationIssue{
				Field:   "expiry",
				Message: "expiry is permitted only for dated contract classes (usdtfutures, coinfutures)",
				Value:   string(contract) + "@" + expiry,
			},
		)
	}
	if len(expiry) != 6 {
		return problem.Validation(
			problem.InvalidArgument,
			"canonical instrument is invalid",
			problem.ValidationIssue{
				Field:   "expiry",
				Message: "must be exactly six digits (canonical YYMMDD)",
				Value:   expiry,
			},
		)
	}
	for _, r := range expiry {
		if r < '0' || r > '9' {
			return problem.Validation(
				problem.InvalidArgument,
				"canonical instrument is invalid",
				problem.ValidationIssue{
					Field:   "expiry",
					Message: "must contain only digits (canonical YYMMDD)",
					Value:   expiry,
				},
			)
		}
	}
	return nil
}

// Symbol returns the canonical string representation:
//
//	"{BASE}/{QUOTE}-{CONTRACT}[@{EXPIRY}]"
//
// Examples:
//   - "BTC/USDT-spot"
//   - "ETH/USDT-perpetual"
//   - "BTC/USD-coinfutures"
//   - "BTC/USDT-usdtfutures@240329" (dated futures, H-7.c)
//
// This is the form embedded in the envelope's `instrument`
// string field (ADR-0017) and in the Sequencer's stream key
// (ADR-0020). The format is stable; downstream consumers may
// parse via FromSymbol. Instruments without expiry produce the
// exact pre-H-7.c form (zero-impact lock-in).
func (c CanonicalInstrument) Symbol() string {
	s := string(c.Base) + "/" + string(c.Quote) + "-" + string(c.Contract)
	if c.Expiry != "" {
		s += "@" + c.Expiry
	}
	return s
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
	if c.Expiry != "" {
		if prob := validateExpiry(c.Expiry, c.Contract); prob != nil {
			return prob
		}
	}
	return nil
}

// IsZero reports whether the instrument is the zero value.
// Useful for detecting unset fields in structs that embed
// CanonicalInstrument and need a "not yet populated" check.
func (c CanonicalInstrument) IsZero() bool {
	return c.Base == "" && c.Quote == "" && c.Contract == "" && c.Expiry == ""
}

// FromSymbol parses a canonical symbol string back into a
// CanonicalInstrument. Returns a Problem if the string does not
// match the expected "{BASE}/{QUOTE}-{CONTRACT}[@{EXPIRY}]" shape
// or any component fails validation.
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

	// Optional dated-futures expiry segment (H-7.c).
	expiry := ""
	if atIdx := strings.LastIndex(s, "@"); atIdx >= 0 {
		if atIdx == len(s)-1 || atIdx == 0 {
			return CanonicalInstrument{}, problem.Validation(
				problem.InvalidArgument,
				"canonical symbol is invalid",
				problem.ValidationIssue{
					Field:   "symbol",
					Message: "expiry separator '@' must sit between non-empty parts",
					Value:   symbol,
				},
			)
		}
		expiry = s[atIdx+1:]
		s = s[:atIdx]
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

	if expiry != "" {
		return NewDelivery(base, quote, contract, expiry)
	}
	return New(base, quote, contract)
}
