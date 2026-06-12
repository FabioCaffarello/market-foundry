package bybitf

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
// Only perpetual is declared: Bybit linear delivery futures
// (dash-separated expiry symbols) are rejected by the parser until
// the G11 enablement wave, and inverse (coin-margined)
// is a different v5 category outside this adapter. Bybit's
// orderbook/ticker/liquidation topics are intentionally NOT
// declared — the shipped parsing surface is trades only (declared
// gaps are the policy's purpose, not a defect).
func Capabilities() ports.Capabilities {
	return ports.Capabilities{
		Venue: instrument.VenueBybitFutures,
		EventTypes: []ports.EventTypeSupport{
			{Type: "observation.trade", Contracts: []instrument.ContractType{
				instrument.ContractPerpetual,
			}},
		},
		Contracts: []instrument.ContractType{
			instrument.ContractPerpetual,
		},
		Notes: map[string]string{
			"delivery": "linear delivery futures rejected at the parser; gated by G11 (dash-format mapping + enablement)",
		},
	}
}
