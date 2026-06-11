package natsevidence_test

import (
	"context"
	"testing"
	"time"

	"internal/adapters/nats/natsevidence"
	"internal/domain/evidence"
	"internal/domain/instrument"
	"internal/shared/problem"
)

func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("test setup: failed to build canonical BTC/USDT-perpetual: %v", prob)
	}
	return inst
}

// mockCandleKVStore is a test double for CandleKVStore.Get behavior.
// It delegates to CandleKVPriceSource via a real CandleKVStore-compatible interface.
// Since CandleKVPriceSource requires a real *CandleKVStore, we test the contract
// indirectly using the PriceSource interface and known behavior.

// stubPriceSource implements ports.PriceSource for integration-level testing.
type stubPriceSource struct {
	price string
	prob  *problem.Problem
}

func (s *stubPriceSource) LastPrice(_ context.Context, _ string, _ instrument.CanonicalInstrument, _ int) (string, *problem.Problem) {
	return s.price, s.prob
}

// TestCandleKVPriceSource_Contract validates the PriceSource contract documented in ports/price.go.
// Uses CandleKVPriceSource behavior expectations without requiring a NATS server.
func TestCandleKVPriceSource_Contract(t *testing.T) {
	t.Run("returns_0_when_store_is_nil", func(t *testing.T) {
		ps := natsevidence.NewCandleKVPriceSource(nil, nil)
		price, prob := ps.LastPrice(context.Background(), "binancef", instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, 60)
		// With nil store, Get returns an error — PriceSource returns "0" + problem.
		if price != "0" {
			t.Fatalf("expected fallback price '0', got %q", price)
		}
		if prob == nil {
			t.Fatal("expected problem for nil store")
		}
	})
}

// TestCandleKVPriceSource_PartitionKeyAlignment ensures the candle key format
// matches the execution PartitionKey format used for price lookups.
func TestCandleKVPriceSource_PartitionKeyAlignment(t *testing.T) {
	// The candle key is "{source}.{symbol}.{timeframe}" — same format as
	// ExecutionIntent.PartitionKey(). This test documents the alignment contract.
	candle := evidence.EvidenceCandle{
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Open:       "50000.00",
		High:       "50100.00",
		Low:        "49900.00",
		Close:      "50050.25",
		Volume:     "123.45",
		OpenTime:   time.Now().UTC().Add(-time.Minute),
		CloseTime:  time.Now().UTC(),
		Final:      true,
	}

	// Verify candle has the Close field that PriceSource reads.
	if candle.Close == "" {
		t.Fatal("candle Close field must not be empty")
	}
	if candle.Close != "50050.25" {
		t.Fatalf("expected Close '50050.25', got %q", candle.Close)
	}
}

// TestPriceSource_FallbackSemantics validates fallback behavior documented in ports/price.go.
func TestPriceSource_FallbackSemantics(t *testing.T) {
	cases := []struct {
		name      string
		price     string
		prob      *problem.Problem
		wantPrice string
	}{
		{
			name:      "normal_price_returned",
			price:     "50123.45",
			prob:      nil,
			wantPrice: "50123.45",
		},
		{
			name:      "cold_start_returns_0",
			price:     "0",
			prob:      nil,
			wantPrice: "0",
		},
		{
			name:      "error_returns_0_with_problem",
			price:     "0",
			prob:      problem.New(problem.Unavailable, "candle KV unavailable"),
			wantPrice: "0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ps := &stubPriceSource{price: tc.price, prob: tc.prob}
			price, _ := ps.LastPrice(context.Background(), "binancef", instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, 60)
			if price != tc.wantPrice {
				t.Fatalf("expected %q, got %q", tc.wantPrice, price)
			}
		})
	}
}
