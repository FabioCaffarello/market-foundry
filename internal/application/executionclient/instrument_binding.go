package executionclient

import (
	"strings"

	"internal/domain/instrument"
)

// instrumentFromBinding reconstructs a CanonicalInstrument from a binding
// pair (source, venueNative). TRANSITORY (H-6.b' → sunset H-6.f). See
// ADR-0021. Duplicated per-package to keep each application package
// self-contained until the canonical instrument identity flows through
// the read-path contracts (LifecycleEntry, ExecutionStatusQuery, etc.).
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
