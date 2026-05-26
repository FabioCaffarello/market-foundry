package execution

import (
	"testing"

	"internal/domain/instrument"
)

// btcUSDTPerp returns the canonical BTC/USDT-perpetual instrument used as the
// default fixture across ExecutionIntent tests in this package.
func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

// ethUSDTSpot returns the canonical ETH/USDT-spot instrument used by
// segment-router tests that simulate spot venues.
func ethUSDTSpot(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("ETH", "USDT", instrument.ContractSpot)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}
