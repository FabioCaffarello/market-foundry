package strategy

import (
	"strings"

	"internal/domain/instrument"
)

// instrumentFromBinding reconstructs a CanonicalInstrument from a binding
// pair (source, venueNative). TRANSITORY (H-6.b → sunset H-6.c). See
// internal/application/signal/instrument_binding.go for the canonical
// shape of this helper.
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
