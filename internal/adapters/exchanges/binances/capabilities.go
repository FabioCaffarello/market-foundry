package binances

import (
	"internal/adapters/exchanges/capabilities"
	"internal/domain/instrument"
)

// Capabilities declares which event types this adapter supports for
// which contract types, per ADR-0022 R1 (H-7.a retrofit). The
// declaration is static and must stay in lockstep with the parsing
// surface (ParseAggTrade + Normalize → observation.trade only);
// `raccoon-cli check venue-parity` enforces presence and coherence,
// and the ingest R3 guard rejects (and counts) anything outside it.
func Capabilities() capabilities.Capabilities {
	return capabilities.Capabilities{
		Venue: instrument.VenueBinance,
		EventTypes: []capabilities.EventTypeSupport{
			{Type: "observation.trade", Contracts: []instrument.ContractType{
				instrument.ContractSpot,
			}},
		},
		Contracts: []instrument.ContractType{
			instrument.ContractSpot,
		},
	}
}
