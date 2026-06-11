package instrument_test

import (
	"testing"

	"internal/domain/instrument"
)

// ── LegacyFilterValue (H-6.e.2 Decisão #4) ───────────────────────

// Lock-in of the exact derivation: the value MUST equal what the
// writers stored in the legacy `symbol` ClickHouse column
// (lower(base+quote), the VenueSymbol() derivation). If this test
// ever needs to change, the WHERE-argument compatibility claim of
// H-6.e.2 breaks — pause and re-frame, do not adjust the literal.
func TestLegacyFilterValue_LockIn(t *testing.T) {
	cases := []struct {
		base, quote string
		contract    instrument.ContractType
		want        string
	}{
		{"BTC", "USDT", instrument.ContractSpot, "btcusdt"},
		{"BTC", "USDT", instrument.ContractPerpetual, "btcusdt"},
		{"ETH", "USDT", instrument.ContractUSDTFutures, "ethusdt"},
		{"BTC", "USD", instrument.ContractCoinFutures, "btcusd"},
	}
	for _, tc := range cases {
		inst, prob := instrument.New(tc.base, tc.quote, tc.contract)
		if prob != nil {
			t.Fatalf("New(%s,%s,%s): %v", tc.base, tc.quote, tc.contract, prob)
		}
		if got := inst.LegacyFilterValue(); got != tc.want {
			t.Errorf("LegacyFilterValue(%s/%s-%s) = %q, want %q",
				tc.base, tc.quote, tc.contract, got, tc.want)
		}
	}
}

// The contract-dropping lossiness is BY DESIGN here (the legacy
// column never carried contract; source disambiguates at write and
// therefore at filter). Locked in so a future "fix" doesn't silently
// change the filter semantics against legacy rows.
func TestLegacyFilterValue_DropsContractByDesign(t *testing.T) {
	spot, _ := instrument.New("BTC", "USDT", instrument.ContractSpot)
	perp, _ := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if spot.LegacyFilterValue() != perp.LegacyFilterValue() {
		t.Errorf("legacy filter values must collapse contract (column semantics): %q vs %q",
			spot.LegacyFilterValue(), perp.LegacyFilterValue())
	}
}
