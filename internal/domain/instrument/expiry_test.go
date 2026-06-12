package instrument_test

import (
	"testing"

	"internal/domain/instrument"
)

// ── Expiry (H-7.c, ADR-0021 erratum 2026-06-12) ──────────────────

// Zero-impact lock-in: instruments WITHOUT expiry produce
// byte-identical Symbol() forms to the pre-H-7.c grammar, for every
// contract type. This is the load-bearing claim that made the field
// addition cutover-free; changing these literals is a wire-format
// cutover, not a refactor.
func TestExpiry_ZeroImpactOnSymbolWithoutExpiry(t *testing.T) {
	cases := []struct {
		base, quote string
		contract    instrument.ContractType
		want        string
	}{
		{"BTC", "USDT", instrument.ContractSpot, "BTC/USDT-spot"},
		{"ETH", "USDT", instrument.ContractPerpetual, "ETH/USDT-perpetual"},
		{"BTC", "USDT", instrument.ContractUSDTFutures, "BTC/USDT-usdtfutures"},
		{"BTC", "USD", instrument.ContractCoinFutures, "BTC/USD-coinfutures"},
	}
	for _, tc := range cases {
		inst, prob := instrument.New(tc.base, tc.quote, tc.contract)
		if prob != nil {
			t.Fatalf("New(%s,%s,%s): %v", tc.base, tc.quote, tc.contract, prob)
		}
		if got := inst.Symbol(); got != tc.want {
			t.Errorf("Symbol(%s) = %q, want pre-H-7.c form %q", tc.contract, got, tc.want)
		}
	}
}

func TestNewDelivery_DatedClasses(t *testing.T) {
	for _, ct := range []instrument.ContractType{
		instrument.ContractUSDTFutures,
		instrument.ContractCoinFutures,
	} {
		inst, prob := instrument.NewDelivery("BTC", "USDT", ct, "240329")
		if prob != nil {
			t.Fatalf("NewDelivery(%s): %v", ct, prob)
		}
		if inst.Expiry != "240329" {
			t.Errorf("Expiry = %q, want 240329", inst.Expiry)
		}
		if prob := inst.Validate(); prob != nil {
			t.Errorf("Validate: %v", prob)
		}
	}
}

func TestNewDelivery_Rejections(t *testing.T) {
	cases := []struct {
		name     string
		contract instrument.ContractType
		expiry   string
	}{
		{"spot_with_expiry", instrument.ContractSpot, "240329"},
		{"perpetual_with_expiry", instrument.ContractPerpetual, "240329"},
		{"too_short", instrument.ContractUSDTFutures, "2403"},
		{"too_long", instrument.ContractUSDTFutures, "20240329"},
		{"non_digits", instrument.ContractUSDTFutures, "24MAR9"},
		{"empty", instrument.ContractUSDTFutures, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, prob := instrument.NewDelivery("BTC", "USDT", tc.contract, tc.expiry); prob == nil {
				t.Errorf("NewDelivery(%s, %q) accepted, want rejection", tc.contract, tc.expiry)
			}
		})
	}
}

// Distinct expiries are distinct canonical identities — the exact
// collision G10 registered (BTCUSDT_240329 vs BTCUSDT_240628
// collapsing) no longer exists for expiry-bearing instruments.
func TestExpiry_DistinctIdentities(t *testing.T) {
	march, prob := instrument.NewDelivery("BTC", "USDT", instrument.ContractUSDTFutures, "240329")
	if prob != nil {
		t.Fatalf("NewDelivery: %v", prob)
	}
	june, prob := instrument.NewDelivery("BTC", "USDT", instrument.ContractUSDTFutures, "240628")
	if prob != nil {
		t.Fatalf("NewDelivery: %v", prob)
	}
	if march == june {
		t.Error("distinct expiries must be distinct canonical identities (G10)")
	}
	if march.Symbol() == june.Symbol() {
		t.Error("distinct expiries must yield distinct Symbol() forms")
	}
}

func TestSymbol_WithExpiryAndRoundtrip(t *testing.T) {
	inst, prob := instrument.NewDelivery("BTC", "USDT", instrument.ContractUSDTFutures, "240329")
	if prob != nil {
		t.Fatalf("NewDelivery: %v", prob)
	}
	if got, want := inst.Symbol(), "BTC/USDT-usdtfutures@240329"; got != want {
		t.Fatalf("Symbol() = %q, want %q", got, want)
	}

	back, prob := instrument.FromSymbol(inst.Symbol())
	if prob != nil {
		t.Fatalf("FromSymbol: %v", prob)
	}
	if back != inst {
		t.Errorf("roundtrip = %+v, want %+v", back, inst)
	}
}

func TestFromSymbol_ExpiryRejections(t *testing.T) {
	cases := []string{
		"BTC/USDT-usdtfutures@",        // empty expiry
		"BTC/USDT-spot@240329",         // expiry on non-dated class
		"BTC/USDT-usdtfutures@24MAR29", // non-digits
		"@240329",                      // separator at start
	}
	for _, sym := range cases {
		if _, prob := instrument.FromSymbol(sym); prob == nil {
			t.Errorf("FromSymbol(%q) accepted, want rejection", sym)
		}
	}
}

func TestIsZero_IncludesExpiry(t *testing.T) {
	if !(instrument.CanonicalInstrument{}).IsZero() {
		t.Error("zero value must report IsZero")
	}
	if (instrument.CanonicalInstrument{Expiry: "240329"}).IsZero() {
		t.Error("expiry-only value must NOT report IsZero")
	}
}
