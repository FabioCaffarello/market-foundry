package signal

import (
	"strings"

	"internal/domain/instrument"
)

// instrumentFromBinding reconstructs a CanonicalInstrument from a binding
// pair (source, venueNative). TRANSITORY (H-6.b → sunset H-6.c): signal
// samplers are constructed from (source, symbol, timeframe) tuples that
// originate in ingestion bindings; this helper produces the canonical
// instrument those bindings represent. H-6.c replaces the symbol/source
// pair with a first-class CanonicalInstrument parameter at the sampler
// constructor boundary.
//
// Mapping:
//   - "binances" → ContractSpot
//   - "binancef" → ContractPerpetual (lossy for delivery futures, which
//     are not in the current routing path)
//   - Unknown sources or non-USDT symbols produce a zero
//     CanonicalInstrument; the constructed Signal will fail
//     Signal.Validate() downstream, surfacing the misconfiguration.
func instrumentFromBinding(source, venueNative string) instrument.CanonicalInstrument {
	upper := strings.ToUpper(strings.TrimSpace(venueNative))
	const quote = "USDT"
	if !strings.HasSuffix(upper, quote) || len(upper) <= len(quote) {
		return instrument.CanonicalInstrument{}
	}
	base := upper[:len(upper)-len(quote)]
	var contract instrument.ContractType
	switch source {
	case "binances":
		contract = instrument.ContractSpot
	case "binancef":
		contract = instrument.ContractPerpetual
	default:
		return instrument.CanonicalInstrument{}
	}
	inst, prob := instrument.New(base, quote, contract)
	if prob != nil {
		return instrument.CanonicalInstrument{}
	}
	return inst
}
