package instrument

import "strings"

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
// KV partition keys intentionally do NOT use this token until
// H-6.e.2 — they keep the VenueSymbol()-derived layout for
// back-compat with the existing bucket shape and the HTTP read
// contract (see ADR-0021 criterion #2 erratum, 2026-06-10).
func (c CanonicalInstrument) SubjectToken() string {
	return strings.ToLower(string(c.Base)) +
		"_" + strings.ToLower(string(c.Quote)) +
		"_" + strings.ToLower(string(c.Contract))
}
