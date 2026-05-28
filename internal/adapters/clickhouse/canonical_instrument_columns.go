package clickhouse

import (
	"errors"
	"fmt"

	"internal/domain/instrument"
)

// ErrLegacyRow is returned by instrumentFromCanonicalColumns when one or more
// of the canonical instrument columns (base / quote / contract) is empty on
// a scanned row. This is the expected shape for rows written before the
// H-6.d.1 migration filled the new columns: ClickHouse defaults each column
// to ” and reader callers fall back to reconstructInstrumentFromLegacy.
// Use errors.Is(err, ErrLegacyRow) to discriminate this expected case from
// hard validation failures (which propagate as descriptive errors).
var ErrLegacyRow = errors.New("clickhouse: canonical instrument columns empty (legacy row)")

// instrumentFromCanonicalColumns constructs a CanonicalInstrument from the
// (base, quote, contract) triple scanned out of a ClickHouse row.
//
// The columns were added by migrations 008–013 (H-6.d.1) and are populated
// by the writer pipeline going forward. Pre-H-6.d.1 rows leave them as the
// schema default (”) and the helper returns ErrLegacyRow so the reader
// can fall back to reconstructInstrumentFromLegacy(source, symbol).
//
// Validation beyond the empty-field check is delegated to instrument.New,
// which is the authoritative gate for BaseAsset/QuoteAsset/ContractType
// per ADR-0021. Hard validation failures (e.g., unknown contract type on
// a row whose canonical columns ARE populated but contain bad data) are
// returned as descriptive errors — not ErrLegacyRow — so they surface as
// regressions rather than silently triggering fallback.
//
// TRANSITORY (H-6.d.2–H-6.f). When H-6.f deletes
// reconstructInstrumentFromLegacy, callers will collapse to this helper
// alone and the fallback branch goes away; this helper itself stays.
func instrumentFromCanonicalColumns(base, quote, contract string) (instrument.CanonicalInstrument, error) {
	if base == "" || quote == "" || contract == "" {
		return instrument.CanonicalInstrument{}, ErrLegacyRow
	}
	inst, prob := instrument.New(base, quote, instrument.ContractType(contract))
	if prob != nil {
		return instrument.CanonicalInstrument{}, fmt.Errorf("build canonical instrument from columns (base=%q quote=%q contract=%q): %s", base, quote, contract, prob.Message)
	}
	return inst, nil
}
