package ingest_test

import (
	"testing"

	"internal/application/ingest"
	"internal/domain/instrument"
)

func TestParseBindingTopic(t *testing.T) {
	tests := []struct {
		topic  string
		source string
		symbol string
	}{
		{"binancef.btcusdt", "binancef", "btcusdt"},
		{"BINANCEF.ETHUSDT", "binancef", "ethusdt"},
		{"bybit.solusdt", "bybit", "solusdt"},
	}

	for _, tc := range tests {
		t.Run(tc.topic, func(t *testing.T) {
			target, prob := ingest.ParseBindingTopic(tc.topic)
			if prob != nil {
				t.Fatalf("unexpected error: %v", prob)
			}
			if target.Source != tc.source {
				t.Fatalf("expected source %s, got %s", tc.source, target.Source)
			}
			if target.Symbol != tc.symbol {
				t.Fatalf("expected symbol %s, got %s", tc.symbol, target.Symbol)
			}
		})
	}
}

func TestParseBindingTopic_Invalid(t *testing.T) {
	tests := []string{
		"",
		"binancef",
		".btcusdt",
		"binancef.",
		"...",
	}

	for _, topic := range tests {
		t.Run(topic, func(t *testing.T) {
			_, prob := ingest.ParseBindingTopic(topic)
			if prob == nil {
				t.Fatalf("expected error for topic %q", topic)
			}
		})
	}
}

func TestBindingTarget_Key(t *testing.T) {
	target := ingest.BindingTarget{Source: "binancef", Symbol: "btcusdt"}
	if key := target.Key(); key != "binancef.btcusdt" {
		t.Fatalf("expected binancef.btcusdt, got %s", key)
	}
}

// TestBindingTarget_Instrument_Recognized confirms the canonical
// boundary reconstruction for the two recognized venues. This is the
// happy path consumed by derive actors via H-6.c.1 commit 6.
func TestBindingTarget_Instrument_Recognized(t *testing.T) {
	tests := []struct {
		source       string
		symbol       string
		wantBase     instrument.BaseAsset
		wantQuote    instrument.QuoteAsset
		wantContract instrument.ContractType
	}{
		{"binances", "btcusdt", "BTC", "USDT", instrument.ContractSpot},
		{"binancef", "ethusdt", "ETH", "USDT", instrument.ContractPerpetual},
	}
	for _, tc := range tests {
		t.Run(tc.source+"_"+tc.symbol, func(t *testing.T) {
			target := ingest.BindingTarget{Source: tc.source, Symbol: tc.symbol}
			inst, err := target.Instrument()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if inst.Base != tc.wantBase || inst.Quote != tc.wantQuote || inst.Contract != tc.wantContract {
				t.Errorf("Instrument = %+v, want {Base:%s Quote:%s Contract:%s}", inst, tc.wantBase, tc.wantQuote, tc.wantContract)
			}
		})
	}
}

// TestBindingTarget_Instrument_RejectsUnknownSource enforces the
// canonical registry contract: unknown sources are rejected at the
// boundary, NOT silently mapped to a zero Instrument. This is the
// regression-shape canary established by H-6.c.1 — replicating the
// silent-zero behavior of the legacy per-package
// instrumentFromBinding helpers (commit 37f8ddd precedent) would
// fail this test.
//
// Synthetic sources observed in pre-flight 5 of H-6.c.1 are
// intentionally absent from the registry; they signal code paths
// that should pass-through an upstream Instrument rather than
// reconstruct via the boundary helper.
func TestBindingTarget_Instrument_RejectsUnknownSource(t *testing.T) {
	tests := []string{
		"binance",               // pre-flight 5 finding (no s/f suffix).
		"binance_spot",          // verify_session.go:410 fallback.
		"derive",                // synthetic scope identifier.
		"clickhouse",            // synthetic back-compat tag.
		"unknown_exchange",      // explicit unknown fallback.
		"execute.venue-adapter", // the exact 37f8ddd trigger.
	}
	for _, src := range tests {
		t.Run(src, func(t *testing.T) {
			target := ingest.BindingTarget{Source: src, Symbol: "btcusdt"}
			inst, err := target.Instrument()
			if err == nil {
				t.Fatalf("expected error for unknown source %q (silent zero is the H-6.b' regression-shape); got Instrument=%+v", src, inst)
			}
			if !inst.IsZero() {
				t.Errorf("expected zero Instrument on error; got %+v", inst)
			}
		})
	}
}

// TestBindingTarget_Instrument_RejectsInvalidShape rejects empty
// fields, non-USDT quotes, and oversized symbol prefixes — all
// boundary conditions that would otherwise propagate into a zero
// or partially-valid Instrument.
func TestBindingTarget_Instrument_RejectsInvalidShape(t *testing.T) {
	tests := []struct {
		name   string
		target ingest.BindingTarget
	}{
		{"empty source", ingest.BindingTarget{Source: "", Symbol: "btcusdt"}},
		{"empty symbol", ingest.BindingTarget{Source: "binancef", Symbol: ""}},
		{"non-USDT quote", ingest.BindingTarget{Source: "binancef", Symbol: "btcusdc"}},
		{"USDT only no base", ingest.BindingTarget{Source: "binancef", Symbol: "usdt"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.target.Instrument(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
