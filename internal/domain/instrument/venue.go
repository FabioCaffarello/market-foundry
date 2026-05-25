package instrument

import "internal/shared/problem"

// Venue identifies an exchange (or exchange-family product). Per
// ADR-0021, Venue lives at the envelope level (ADR-0017), not
// inside CanonicalInstrument — the same instrument identity is
// shared across venues to enable cross-venue capabilities.
//
// The enum is intentionally restricted in H-6.a to the venues
// with shipping adapters (binance, binancef). Future venues
// (bybit, coinbase, hyperliquid, kraken, krakenf — listed in
// ADR-0021 design intent) gain their entries when their adapters
// land in PROGRAM-0004 H-7 and beyond. This mirrors the
// "declare what's in use" discipline from H-5's check metrics
// allowlist.
type Venue string

const (
	// VenueBinance is Binance Spot.
	VenueBinance Venue = "binance"
	// VenueBinanceFutures is Binance USDT-margined Futures
	// (the `binancef` adapter family).
	VenueBinanceFutures Venue = "binancef"
)

// ValidVenue reports whether v is a recognized venue value.
// Returns false for any string not in the declared enum, which
// is the safer default — adding a new venue requires explicit
// extension here and a paired adapter, never accidental
// acceptance.
func ValidVenue(v Venue) bool {
	switch v {
	case VenueBinance, VenueBinanceFutures:
		return true
	default:
		return false
	}
}

// String returns the venue's string form.
func (v Venue) String() string { return string(v) }

// Validate reports whether the venue value is recognized.
func (v Venue) Validate() *problem.Problem {
	if !ValidVenue(v) {
		return problem.Validation(
			problem.InvalidArgument,
			"venue is invalid",
			problem.ValidationIssue{
				Field:   "venue",
				Message: "must be one of: binance, binancef",
				Value:   string(v),
			},
		)
	}
	return nil
}
