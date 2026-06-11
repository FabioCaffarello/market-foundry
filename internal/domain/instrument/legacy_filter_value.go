package instrument

import "strings"

// LegacyFilterValue returns the venue-native lowercase pair form
// ("btcusdt") derived from the canonical instrument — the exact
// value stored in the legacy `symbol` column of every ClickHouse
// analytical table (the writers populated it via the aggregates'
// VenueSymbol(), which uses this same lower(base+quote) derivation).
//
// TRANSITORY (H-6.e.2 → sunset H-6.f): exists ONLY so ClickHouse
// query-argument construction can keep the `WHERE … symbol = ?`
// shape unchanged while the read contract is already canonical —
// the legitimate canonical→venue direction (the banned direction is
// the venue→canonical string inference eliminated in H-6.c). The
// WHERE flips to the canonical base/quote/contract columns in
// H-6.f, post the 90-day legacy-row TTL (~2026-08-26), and this
// helper is deleted in the same commit (registered in PROGRAM-0004
// → H-6.f scope).
//
// Do NOT use for KV partition keys (those use SubjectToken() as of
// H-6.e.2), for NATS subjects (SubjectToken(), H-6.e), or for any
// new surface.
//
// Same lossiness as VenueSymbol(): Contract is dropped, so this
// value alone does not discriminate spot vs perpetual — exactly why
// it is only valid as the legacy-column filter (paired with the
// `source` predicate, mirroring how the rows were written) and not
// as an identity.
func (c CanonicalInstrument) LegacyFilterValue() string {
	return strings.ToLower(string(c.Base) + string(c.Quote))
}
