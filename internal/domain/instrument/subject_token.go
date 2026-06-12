package instrument

import (
	"strings"

	"internal/shared/problem"
)

// SubjectToken returns the canonical NATS subject token:
//
//	"{base}_{quote}_{contract}[_{expiry}]"   (all lowercase)
//
// Examples:
//   - "btc_usdt_spot"
//   - "eth_usdt_perpetual"
//   - "btc_usd_coinfutures"
//   - "btc_usdt_usdtfutures_240329" (dated futures, H-7.c)
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
//   - Non-lossy: distinct ContractTypes yield distinct tokens for
//     the same pair, and — since H-7.c activated the formerly
//     dormant 4th component (ADR-0009 erratum 2026-06-12) —
//     distinct expiries yield distinct tokens too. Tokens for
//     instruments without expiry are byte-identical to the
//     pre-H-7.c grammar (no cutover; zero expiry-bearing
//     instruments circulate until the G11 enablement gaps close).
//
// KV partition keys also use this token as of H-6.e.2
// ({source}.{SubjectToken()}.{timeframe} — ADR-0021 criterion #2
// erratum, 2026-06-10).
func (c CanonicalInstrument) SubjectToken() string {
	token := strings.ToLower(string(c.Base)) +
		"_" + strings.ToLower(string(c.Quote)) +
		"_" + strings.ToLower(string(c.Contract))
	if c.Expiry != "" {
		token += "_" + c.Expiry
	}
	return token
}

// FromSubjectToken parses a canonical subject token
// ("{base}_{quote}_{contract}[_{expiry}]", as produced by
// SubjectToken) back into a CanonicalInstrument. Returns a Problem
// if the token does not split into three or four non-empty
// '_'-separated components or any component fails validation.
//
// Parsing is unambiguous by construction: asset tickers admit only
// ASCII letters and digits (validateAssetTicker), no ContractType
// constant contains '_', and the expiry component is digits-only
// (validateExpiry) — so a well-formed token carries exactly two or
// three underscores and every split is unique.
// TestFromSubjectToken_NoUnderscoreInComponents locks the premises
// in. H-7.c revisited this parser in the same commit that activated
// the formerly dormant 4th component — exactly the pause trigger the
// H-6.f.1 lock-in armed; the 3-component grammar is untouched.
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
	if len(parts) < 3 || len(parts) > 4 {
		return CanonicalInstrument{}, problem.Validation(
			problem.InvalidArgument,
			"subject token is invalid",
			problem.ValidationIssue{
				Field:   "token",
				Message: "must have shape {base}_{quote}_{contract}[_{expiry}] (3 or 4 parts)",
				Value:   token,
			},
		)
	}
	for _, p := range parts {
		if p == "" {
			return CanonicalInstrument{}, problem.Validation(
				problem.InvalidArgument,
				"subject token is invalid",
				problem.ValidationIssue{
					Field:   "token",
					Message: "must have non-empty components",
					Value:   token,
				},
			)
		}
	}
	if len(parts) == 4 {
		return NewDelivery(parts[0], parts[1], ContractType(parts[2]), parts[3])
	}
	return New(parts[0], parts[1], ContractType(parts[2]))
}
