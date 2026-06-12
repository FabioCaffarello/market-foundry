package binances_test

import (
	"testing"

	"internal/adapters/exchanges/binances"
	"internal/domain/instrument"
)

// Lock-in of the H-7.a retrofit declaration (ADR-0022 R1): binances
// supports observation.trade for spot, and nothing else. Changing
// this declaration is a capabilities-contract change, not a refactor
// — it must ride a wave that also adjusts the parsing surface.
func TestCapabilities_Declaration(t *testing.T) {
	c := binances.Capabilities()

	if prob := c.Validate(); prob != nil {
		t.Fatalf("declaration incoherent: %v", prob)
	}
	if c.Venue != instrument.VenueBinance {
		t.Errorf("venue = %q, want %q", c.Venue, instrument.VenueBinance)
	}
	if !c.Allows("observation.trade", instrument.ContractSpot) {
		t.Error("observation.trade/spot must be declared")
	}
	for _, undeclared := range []instrument.ContractType{
		instrument.ContractPerpetual,
		instrument.ContractUSDTFutures,
		instrument.ContractCoinFutures,
	} {
		if c.Allows("observation.trade", undeclared) {
			t.Errorf("observation.trade/%s must NOT be declared on the spot adapter", undeclared)
		}
	}
	if c.Allows("observation.markprice", instrument.ContractSpot) {
		t.Error("observation.markprice must NOT be declared (Binance Spot has no mark price)")
	}
}
