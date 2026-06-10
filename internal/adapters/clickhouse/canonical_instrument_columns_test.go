package clickhouse

import (
	"errors"
	"testing"

	"internal/domain/instrument"
)

func TestInstrumentFromCanonicalColumns_AllEmpty_LegacyRow(t *testing.T) {
	_, err := instrumentFromCanonicalColumns("", "", "")
	if !errors.Is(err, ErrLegacyRow) {
		t.Fatalf("expected ErrLegacyRow, got %v", err)
	}
}

func TestInstrumentFromCanonicalColumns_PartiallyEmpty_LegacyRow(t *testing.T) {
	cases := []struct {
		name             string
		base, quote, ctr string
	}{
		{"base empty", "", "USDT", "perpetual"},
		{"quote empty", "BTC", "", "perpetual"},
		{"contract empty", "BTC", "USDT", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := instrumentFromCanonicalColumns(tc.base, tc.quote, tc.ctr)
			if !errors.Is(err, ErrLegacyRow) {
				t.Fatalf("expected ErrLegacyRow, got %v", err)
			}
		})
	}
}

func TestInstrumentFromCanonicalColumns_ValidTriple_Constructs(t *testing.T) {
	cases := []struct {
		name             string
		base, quote, ctr string
		want             instrument.ContractType
	}{
		{"spot", "BTC", "USDT", "spot", instrument.ContractSpot},
		{"perpetual", "BTC", "USDT", "perpetual", instrument.ContractPerpetual},
		{"usdt-futures", "BTC", "USDT", "usdtfutures", instrument.ContractUSDTFutures},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inst, err := instrumentFromCanonicalColumns(tc.base, tc.quote, tc.ctr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if inst.Contract != tc.want {
				t.Fatalf("contract: want %q got %q", tc.want, inst.Contract)
			}
			if string(inst.Base) != tc.base || string(inst.Quote) != tc.quote {
				t.Fatalf("identity drift: base=%q quote=%q", inst.Base, inst.Quote)
			}
		})
	}
}

func TestInstrumentFromCanonicalColumns_InvalidContract_PropagatesError(t *testing.T) {
	_, err := instrumentFromCanonicalColumns("BTC", "USDT", "not-a-contract-type")
	if err == nil {
		t.Fatal("expected error for invalid contract type")
	}
	if errors.Is(err, ErrLegacyRow) {
		t.Fatalf("invalid contract must NOT be misclassified as legacy row: %v", err)
	}
}
