//go:build integration

package execute_test

import (
	"testing"

	"internal/domain/instrument"
)

// btcUSDTPerpIntegration is the canonical BTC/USDT-perpetual helper for
// integration tests in this package. Mirrors the per-package
// btcUSDTPerp helpers added during H-6.b across the rest of the
// repo; lives in its own file with the `//go:build integration` tag
// so the regular unit-test build doesn't pull it.
func btcUSDTPerpIntegration(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup BTC/USDT-perpetual: %v", prob)
	}
	return inst
}
