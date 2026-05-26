// Package instrument provides the canonical instrument model
// per ADR-0021. Domain layer never knows venue-native symbol
// formats; adapters normalize venue strings into
// CanonicalInstrument at the layer boundary.
//
// This package is in the innermost layer (internal/domain/) and
// pure value-typed: no time, no random, no I/O. Compatible with
// the check determinism analyzer (ADR-0019 INV-D1).
package instrument

import (
	"strings"

	"internal/shared/problem"
)

// BaseAsset is the uppercase ticker symbol for the base side of
// a trading pair (e.g., "BTC" in BTC/USDT).
type BaseAsset string

// QuoteAsset is the uppercase ticker symbol for the quote side
// of a trading pair (e.g., "USDT" in BTC/USDT).
type QuoteAsset string

// NewBaseAsset normalizes input (trim + uppercase) and validates.
// Returns a Problem on empty / disallowed characters.
func NewBaseAsset(raw string) (BaseAsset, *problem.Problem) {
	s, prob := normalizeAssetTicker(raw, "base")
	if prob != nil {
		return "", prob
	}
	return BaseAsset(s), nil
}

// NewQuoteAsset normalizes input (trim + uppercase) and validates.
// Returns a Problem on empty / disallowed characters.
func NewQuoteAsset(raw string) (QuoteAsset, *problem.Problem) {
	s, prob := normalizeAssetTicker(raw, "quote")
	if prob != nil {
		return "", prob
	}
	return QuoteAsset(s), nil
}

// String returns the asset ticker as a plain string.
func (a BaseAsset) String() string  { return string(a) }
func (a QuoteAsset) String() string { return string(a) }

// Validate reports whether the asset value satisfies the asset-
// ticker shape. Used by CanonicalInstrument.Validate; callers who
// constructed via NewBaseAsset/NewQuoteAsset can rely on the
// value being already valid.
func (a BaseAsset) Validate() *problem.Problem {
	return validateAssetTicker(string(a), "base")
}

func (a QuoteAsset) Validate() *problem.Problem {
	return validateAssetTicker(string(a), "quote")
}

// normalizeAssetTicker trims whitespace, uppercases, and
// validates the result. Returns the normalized string and any
// validation issue.
func normalizeAssetTicker(raw, side string) (string, *problem.Problem) {
	s := strings.ToUpper(strings.TrimSpace(raw))
	if prob := validateAssetTicker(s, side); prob != nil {
		return "", prob
	}
	return s, nil
}

// validateAssetTicker enforces the ticker shape:
//   - non-empty
//   - length 1..16 (covers BTC=3, USDT=4, FDUSD=5, edge cases up
//     to 16 like "1000PEPE" or wrapped tokens)
//   - characters: ASCII uppercase A–Z and digits 0–9 only
//
// Lowercase, slashes, dashes, dots, spaces, and other punctuation
// are rejected. This is stricter than the venue's native shape
// (which often allows lowercase) — adapters normalize via
// NewBaseAsset/NewQuoteAsset before constructing the canonical
// instrument.
func validateAssetTicker(s, side string) *problem.Problem {
	if s == "" {
		return problem.Validation(
			problem.InvalidArgument,
			"asset ticker is invalid",
			problem.ValidationIssue{Field: side, Message: "must not be empty"},
		)
	}
	if len(s) > 16 {
		return problem.Validation(
			problem.InvalidArgument,
			"asset ticker is invalid",
			problem.ValidationIssue{
				Field:   side,
				Message: "must be at most 16 characters",
				Value:   s,
			},
		)
	}
	for _, r := range s {
		if !isUpperAlnum(r) {
			return problem.Validation(
				problem.InvalidArgument,
				"asset ticker is invalid",
				problem.ValidationIssue{
					Field:   side,
					Message: "must contain only uppercase ASCII letters and digits",
					Value:   s,
				},
			)
		}
	}
	return nil
}

func isUpperAlnum(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
