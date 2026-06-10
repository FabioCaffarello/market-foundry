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
