package bybits

import (
	"internal/application/ports"
	"internal/domain/instrument"
)

// Capabilities declares which event types this adapter supports for
// which contract types, per ADR-0022 R1 (H-7.b — first non-Binance
// venue). The declaration is static and must stay in lockstep with
// the parsing surface (ParsePublicTrade + Normalize →
// observation.trade only); `raccoon-cli check venue-parity` enforces
// presence and coherence, and the ingest R3 guard rejects (and
// counts) anything outside it.
//
// Bybit Spot also publishes orderbook/ticker topics — intentionally
// NOT declared: the foundry's observation plane consumes trades
// only, and ADR-0022 R1 declares what the adapter SUPPORTS, which
// is exactly the shipped parsing surface (declared gaps are the
// policy's purpose, not a defect).
func Capabilities() ports.Capabilities {
	return ports.Capabilities{
		Venue: instrument.VenueBybit,
		EventTypes: []ports.EventTypeSupport{
			{Type: "observation.trade", Contracts: []instrument.ContractType{
				instrument.ContractSpot,
			}},
		},
		Contracts: []instrument.ContractType{
			instrument.ContractSpot,
		},
	}
}
