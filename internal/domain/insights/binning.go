package insights

import (
	"math/big"
	"strings"

	"internal/shared/problem"
)

// BucketLevel maps a trade price to the canonical lower bound of its
// price bucket, given a bucket width. Both inputs and the result are
// decimal strings. Pure and deterministic (math/big exact rational
// arithmetic — no float drift, no time, no random) so the same
// (price, width) always yields the same bucket across replays.
//
//	BucketLevel("65000.50", "10")  → "65000"
//	BucketLevel("65000.50", "0.5") → "65000.5"
//	BucketLevel("65007",    "10")  → "65000"
//
// The bucket is [level, level+width). This is the canonical binning
// rule for volume profiles (ADR-0027 I3); samplers MUST derive bucket
// keys through it rather than formatting their own.
func BucketLevel(price, width string) (string, *problem.Problem) {
	p, ok := new(big.Rat).SetString(strings.TrimSpace(price))
	if !ok {
		return "", problem.Validation(
			problem.InvalidArgument,
			"bucket binning failed",
			problem.ValidationIssue{Field: "price", Message: "must be a decimal", Value: price},
		)
	}
	w, ok := new(big.Rat).SetString(strings.TrimSpace(width))
	if !ok || w.Sign() <= 0 {
		return "", problem.Validation(
			problem.InvalidArgument,
			"bucket binning failed",
			problem.ValidationIssue{Field: "width", Message: "must be a positive decimal", Value: width},
		)
	}

	// idx = floor(price / width); big.Int.Div is Euclidean (floor)
	// for the always-positive normalized big.Rat denominator.
	q := new(big.Rat).Quo(p, w)
	idx := new(big.Int).Div(q.Num(), q.Denom())

	level := new(big.Rat).Mul(new(big.Rat).SetInt(idx), w)
	return formatRat(level, decimalPlaces(width)), nil
}

// formatRat renders an exact (finite-decimal) big.Rat as the minimal
// canonical decimal string (no trailing zeros, no trailing dot).
// `decimals` is an upper bound on fractional digits; since the value
// is an integer multiple of a `decimals`-place width, FloatString is
// exact (no rounding).
func formatRat(r *big.Rat, decimals int) string {
	if r.IsInt() {
		return r.Num().String()
	}
	s := r.FloatString(decimals)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

// decimalPlaces counts fractional digits in a decimal string.
func decimalPlaces(d string) int {
	d = strings.TrimSpace(d)
	if i := strings.IndexByte(d, '.'); i >= 0 {
		return len(d) - i - 1
	}
	return 0
}
