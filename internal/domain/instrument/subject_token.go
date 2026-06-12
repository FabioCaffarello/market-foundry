package instrument

import (
	"strings"

	"internal/shared/problem"
)

// SubjectToken returns the canonical NATS subject token:
//
//	"{base}_{quote}_{contract}"   (all lowercase)
//
// Examples:
//   - "btc_usdt_spot"
//   - "eth_usdt_perpetual"
//   - "btc_usd_coinfutures"
//
// This is the ONLY sanctioned derivation for the {symbol} token of
// the NATS subject taxonomy (ADR-0009, erratum 2026-06-10 — Onda
// H-6.e). Subject builders must call this method and never format
// the token themselves; raccoon-cli `check subjects` enforces it.
//
// Contract:
//   - Subject-safe: the token contains no '.', '/', '*', '>',
//     spaces, or uppercase — every component is lowercased and the
//     components are joined by '_'.
//   - Non-lossy beyond what the canonical model itself permits
//     today: distinct ContractTypes yield distinct tokens for the
//     same pair. Expiry is NOT yet a field of CanonicalInstrument,
//     so delivery-futures contracts with different expiries collide
//     in canonical identity itself — a registered modeling debt
//     (PROGRAM-0004 → H-6.e.2 scope; RESUMPTION known-gaps), not a
//     token-formatting concern. When expiry enters the model, the
//     token grows a fourth "_{expiry}" component (dormant slot per
//     the ADR-0009 erratum).
//
// KV partition keys also use this token as of H-6.e.2
// ({source}.{SubjectToken()}.{timeframe} — ADR-0021 criterion #2
// erratum, 2026-06-10).
func (c CanonicalInstrument) SubjectToken() string {
	return strings.ToLower(string(c.Base)) +
		"_" + strings.ToLower(string(c.Quote)) +
		"_" + strings.ToLower(string(c.Contract))
}

// FromSubjectToken parses a canonical subject token
// ("{base}_{quote}_{contract}", as produced by SubjectToken) back
// into a CanonicalInstrument. Returns a Problem if the token does
// not split into exactly three non-empty '_'-separated components
// or any component fails validation.
//
// Parsing is unambiguous by construction: asset tickers admit only
// ASCII letters and digits (validateAssetTicker) and no ContractType
// constant contains '_', so a well-formed token carries exactly two
// underscores. TestFromSubjectToken_NoUnderscoreInComponents locks
// both premises in; if a future ContractType gains an underscore, or
// the dormant "_{expiry}" slot (ADR-0009 erratum) activates, this
// parser must be revisited first — pause-and-report per H-6.f.1
// wave protocol.
//
// The direction is canonical→canonical: the token is a sanctioned
// derivation of the canonical identity, so no venue inference is
// involved — unlike the banned source-string reconstruction pattern
// (anti_patterns.toml), which guesses quote asset and contract from
// a venue-native shape.
func FromSubjectToken(token string) (CanonicalInstrument, *problem.Problem) {
	s := strings.TrimSpace(token)
	if s == "" {
		return CanonicalInstrument{}, problem.Validation(
			problem.InvalidArgument,
			"subject token is invalid",
			problem.ValidationIssue{Field: "token", Message: "must not be empty"},
		)
	}
	parts := strings.Split(s, "_")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return CanonicalInstrument{}, problem.Validation(
			problem.InvalidArgument,
			"subject token is invalid",
			problem.ValidationIssue{
				Field:   "token",
				Message: "must have shape {base}_{quote}_{contract} with non-empty parts",
				Value:   token,
			},
		)
	}
	return New(parts[0], parts[1], ContractType(parts[2]))
}
