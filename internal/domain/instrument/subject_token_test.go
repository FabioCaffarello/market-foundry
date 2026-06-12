package instrument_test

import (
	"strings"
	"testing"

	"internal/domain/instrument"
)

// ── SubjectToken (H-6.e Decisão #1) ──────────────────────────────

// Lock-in of the exact derivation per contract type. These literals
// are the wire-visible subject grammar (ADR-0009 erratum 2026-06-10);
// changing them is a routing cutover, not a refactor — any diff here
// must come with its own erratum and migration plan.
func TestSubjectToken_LockInPerContractType(t *testing.T) {
	cases := []struct {
		base, quote string
		contract    instrument.ContractType
		want        string
	}{
		{"BTC", "USDT", instrument.ContractSpot, "btc_usdt_spot"},
		{"ETH", "USDT", instrument.ContractPerpetual, "eth_usdt_perpetual"},
		{"BTC", "USDT", instrument.ContractUSDTFutures, "btc_usdt_usdtfutures"},
		{"BTC", "USD", instrument.ContractCoinFutures, "btc_usd_coinfutures"},
	}
	for _, tc := range cases {
		inst, prob := instrument.New(tc.base, tc.quote, tc.contract)
		if prob != nil {
			t.Fatalf("New(%s,%s,%s): %v", tc.base, tc.quote, tc.contract, prob)
		}
		if got := inst.SubjectToken(); got != tc.want {
			t.Errorf("SubjectToken(%s/%s-%s) = %q, want %q",
				tc.base, tc.quote, tc.contract, got, tc.want)
		}
	}
}

// Distinct contract types on the same pair must yield distinct
// tokens — the "non-lossy beyond what the model permits today"
// claim of H-6.e. (Expiry is not a model field; delivery-futures
// expiry collision is the registered modeling debt of H-6.e.2 and
// is intentionally NOT covered here.)
func TestSubjectToken_DistinctAcrossContractTypes(t *testing.T) {
	contracts := []instrument.ContractType{
		instrument.ContractSpot,
		instrument.ContractPerpetual,
		instrument.ContractUSDTFutures,
		instrument.ContractCoinFutures,
	}
	seen := make(map[string]instrument.ContractType, len(contracts))
	for _, ct := range contracts {
		inst, prob := instrument.New("BTC", "USDT", ct)
		if prob != nil {
			t.Fatalf("New(BTC,USDT,%s): %v", ct, prob)
		}
		tok := inst.SubjectToken()
		if prev, dup := seen[tok]; dup {
			t.Errorf("token collision: %q produced by both %s and %s", tok, prev, ct)
		}
		seen[tok] = ct
	}
}

// ── FromSubjectToken (H-6.f.1 Decisão #3) ────────────────────────

// Roundtrip lock-in: SubjectToken → FromSubjectToken must recover
// the identical CanonicalInstrument for every contract type. This
// is the contract the audit read-path (executionclient) relies on
// after the H-6.f.1 regression fix.
func TestFromSubjectToken_RoundtripPerContractType(t *testing.T) {
	contracts := []instrument.ContractType{
		instrument.ContractSpot,
		instrument.ContractPerpetual,
		instrument.ContractUSDTFutures,
		instrument.ContractCoinFutures,
	}
	for _, ct := range contracts {
		inst, prob := instrument.New("BTC", "USDT", ct)
		if prob != nil {
			t.Fatalf("New(BTC,USDT,%s): %v", ct, prob)
		}
		got, prob := instrument.FromSubjectToken(inst.SubjectToken())
		if prob != nil {
			t.Fatalf("FromSubjectToken(%q): %v", inst.SubjectToken(), prob)
		}
		if got != inst {
			t.Errorf("roundtrip(%s) = %+v, want %+v", ct, got, inst)
		}
	}
}

// Rejections: malformed shapes, invalid contract, empty input must
// all return a Problem and a zero instrument — never a partial one.
func TestFromSubjectToken_Rejections(t *testing.T) {
	cases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"whitespace_only", "   "},
		{"missing_components", "btc_usdt"},
		// 4 parts parse since H-7.c, but expiry on a non-dated
		// class is a validation rejection (was a shape rejection
		// pre-H-7.c — same verdict, different reason).
		{"expiry_on_spot", "btc_usdt_spot_240329"},
		{"five_components", "btc_usdt_usdtfutures_240329_extra"},
		{"non_digit_expiry", "btc_usdt_usdtfutures_24mar29"},
		{"short_expiry", "btc_usdt_usdtfutures_2403"},
		{"empty_expiry", "btc_usdt_usdtfutures_"},
		{"empty_base", "_usdt_spot"},
		{"empty_quote", "btc__spot"},
		{"empty_contract", "btc_usdt_"},
		{"invalid_contract", "btc_usdt_swap"},
		{"venue_native_shape", "btcusdt"},
		{"canonical_symbol_shape", "BTC/USDT-spot"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, prob := instrument.FromSubjectToken(tc.token)
			if prob == nil {
				t.Fatalf("FromSubjectToken(%q) accepted, want rejection (got %+v)", tc.token, got)
			}
			if !got.IsZero() {
				t.Errorf("FromSubjectToken(%q) returned non-zero instrument %+v with error", tc.token, got)
			}
		})
	}
}

// ── Expiry component (H-7.c — dormant slot activated) ────────────

// Lock-in of the 4-component grammar: dated futures derive
// "{base}_{quote}_{contract}_{expiry}" and roundtrip through
// FromSubjectToken. The 3-component tokens stay byte-identical to
// the pre-H-7.c grammar (asserted by the lock-in tests above).
func TestSubjectToken_ExpiryComponentRoundtrip(t *testing.T) {
	inst, prob := instrument.NewDelivery("BTC", "USDT", instrument.ContractUSDTFutures, "240329")
	if prob != nil {
		t.Fatalf("NewDelivery: %v", prob)
	}
	if got, want := inst.SubjectToken(), "btc_usdt_usdtfutures_240329"; got != want {
		t.Fatalf("SubjectToken() = %q, want %q", got, want)
	}
	back, prob := instrument.FromSubjectToken(inst.SubjectToken())
	if prob != nil {
		t.Fatalf("FromSubjectToken: %v", prob)
	}
	if back != inst {
		t.Errorf("roundtrip = %+v, want %+v", back, inst)
	}
}

// Distinct expiries yield distinct tokens — the routing-layer half
// of the G10 fix (the identity half is asserted in expiry_test.go).
func TestSubjectToken_DistinctAcrossExpiries(t *testing.T) {
	march, prob := instrument.NewDelivery("BTC", "USDT", instrument.ContractUSDTFutures, "240329")
	if prob != nil {
		t.Fatalf("NewDelivery: %v", prob)
	}
	june, prob := instrument.NewDelivery("BTC", "USDT", instrument.ContractUSDTFutures, "240628")
	if prob != nil {
		t.Fatalf("NewDelivery: %v", prob)
	}
	if march.SubjectToken() == june.SubjectToken() {
		t.Error("distinct expiries must yield distinct subject tokens (G10)")
	}
}

// Premise lock-in: the parser's non-ambiguity rests on no component
// of a well-formed token containing '_' — no ContractType constant
// has one, asset tickers admit only ASCII letters and digits, and
// the expiry component (4th, active since H-7.c) is digits-only.
// If this test fails, FromSubjectToken's split strategy is no
// longer sound: pause-and-report before changing either side.
// (The H-6.f.1 version of this comment armed a pause trigger for
// the dormant "_{expiry}" slot activation; H-7.c executed that
// revisit in the same commit that activated the component.)
func TestFromSubjectToken_NoUnderscoreInComponents(t *testing.T) {
	contracts := []instrument.ContractType{
		instrument.ContractSpot,
		instrument.ContractPerpetual,
		instrument.ContractUSDTFutures,
		instrument.ContractCoinFutures,
	}
	for _, ct := range contracts {
		if strings.Contains(string(ct), "_") {
			t.Errorf("ContractType %q contains '_' — breaks FromSubjectToken non-ambiguity premise", ct)
		}
	}
	if _, prob := instrument.New("A_B", "USDT", instrument.ContractSpot); prob == nil {
		t.Error("NewBaseAsset accepted '_' — breaks FromSubjectToken non-ambiguity premise")
	}
	if _, prob := instrument.New("BTC", "U_SDT", instrument.ContractSpot); prob == nil {
		t.Error("NewQuoteAsset accepted '_' — breaks FromSubjectToken non-ambiguity premise")
	}
}

// Subject-safety: the token must never contain NATS-significant or
// taxonomy-breaking characters, regardless of asset input casing.
func TestSubjectToken_SubjectSafe(t *testing.T) {
	inst, prob := instrument.New("btc", "usdt", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("New: %v", prob)
	}
	tok := inst.SubjectToken()
	if strings.ContainsAny(tok, "./*> ") {
		t.Errorf("token %q contains NATS-significant characters", tok)
	}
	if tok != strings.ToLower(tok) {
		t.Errorf("token %q is not lowercase", tok)
	}
	if got, want := tok, "btc_usdt_perpetual"; got != want {
		t.Errorf("lowercase-input derivation = %q, want %q (assets normalize via New)", got, want)
	}
}
