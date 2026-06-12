package binancef_test

import (
	"testing"

	"internal/adapters/exchanges/binancef"
	"internal/domain/instrument"
)

// Lock-in of the H-7.a retrofit declaration (ADR-0022 R1): binancef
// supports observation.trade for perpetual + usdtfutures (the
// parseFuturesSymbol surface), and nothing else. usdtfutures is a
// capability of the ADAPTER; ingest enablement of delivery futures
// stays gated by G10 until H-7.c (note carried in the declaration).
func TestCapabilities_Declaration(t *testing.T) {
	c := binancef.Capabilities()

	if prob := c.Validate(); prob != nil {
		t.Fatalf("declaration incoherent: %v", prob)
	}
	if c.Venue != instrument.VenueBinanceFutures {
		t.Errorf("venue = %q, want %q", c.Venue, instrument.VenueBinanceFutures)
	}
	if !c.Allows("observation.trade", instrument.ContractPerpetual) {
		t.Error("observation.trade/perpetual must be declared")
	}
	if !c.Allows("observation.trade", instrument.ContractUSDTFutures) {
		t.Error("observation.trade/usdtfutures must be declared (parser supports the _YYMMDD suffix)")
	}
	if c.Allows("observation.trade", instrument.ContractSpot) {
		t.Error("observation.trade/spot must NOT be declared on the futures adapter")
	}
	if c.Allows("observation.trade", instrument.ContractCoinFutures) {
		t.Error("observation.trade/coinfutures must NOT be declared (binancef is the USDT-margined family)")
	}
	if _, ok := c.Notes["usdtfutures"]; !ok {
		t.Error("usdtfutures G10 gating note must be carried in the declaration")
	}
}
