package execution_test

import (
	"strings"
	"testing"

	"internal/domain/instrument"
)

// instrumentFromVenueSymbol parses a USDT-suffixed venue symbol and contract
// derived from source ("binances" → spot, "binancef" → perpetual) into a
// CanonicalInstrument. Used in tests that vary venue symbol by data-driven
// inputs.
func instrumentFromVenueSymbol(t *testing.T, source, venueSym string) instrument.CanonicalInstrument {
	t.Helper()
	upper := strings.ToUpper(strings.TrimSpace(venueSym))
	const quote = "USDT"
	if !strings.HasSuffix(upper, quote) || len(upper) <= len(quote) {
		t.Fatalf("setup: cannot parse venue symbol %q", venueSym)
	}
	base := upper[:len(upper)-len(quote)]
	var contract instrument.ContractType
	switch source {
	case "binances":
		contract = instrument.ContractSpot
	case "binancef":
		contract = instrument.ContractPerpetual
	default:
		t.Fatalf("setup: unknown source %q", source)
	}
	inst, prob := instrument.New(base, quote, contract)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

// btcUSDTPerp returns the canonical BTC/USDT-perpetual instrument used by
// external-package execution tests.
func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

// btcUSDTSpot returns the canonical BTC/USDT-spot instrument used by
// external-package execution tests for spot adapters.
func btcUSDTSpot(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractSpot)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

// ethUSDTPerp returns the canonical ETH/USDT-perpetual instrument used by
// external-package execution tests that need a second-base fixture.
func ethUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("ETH", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

// ethUSDTSpot returns the canonical ETH/USDT-spot instrument.
func ethUSDTSpot(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("ETH", "USDT", instrument.ContractSpot)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

// solUSDTPerp returns the canonical SOL/USDT-perpetual instrument used by
// multi-symbol concurrency tests (H-6.c.2 commit 1 — third base alongside
// btcUSDTPerp/ethUSDTPerp).
func solUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("SOL", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}
