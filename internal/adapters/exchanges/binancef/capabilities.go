package binancef

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
//
// usdtfutures is declared because parseFuturesSymbol normalizes
// delivery-futures symbols (the `_YYMMDD` suffix) — the capability
// describes the adapter, not the deployment: enabling delivery
// futures at ingest stays gated by G10 (expiry is not yet a model
// field; modeling lands in H-7.c). See the note below.
func Capabilities() capabilities.Capabilities {
	return capabilities.Capabilities{
		Venue: instrument.VenueBinanceFutures,
		EventTypes: []capabilities.EventTypeSupport{
			{Type: "observation.trade", Contracts: []instrument.ContractType{
				instrument.ContractPerpetual,
				instrument.ContractUSDTFutures,
			}},
		},
		Contracts: []instrument.ContractType{
			instrument.ContractPerpetual,
			instrument.ContractUSDTFutures,
		},
		Notes: map[string]string{
			"usdtfutures": "normalization supported; ingest enablement gated by G10 (expiry modeling, H-7.c)",
		},
	}
}
