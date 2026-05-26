package signal_test

import (
	"fmt"
	"testing"
	"time"

	"internal/domain/instrument"
	"internal/domain/signal"
)

func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

func validSignal(t *testing.T) signal.Signal {
	t.Helper()
	return signal.Signal{
		Type:       "rsi",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  300,
		Value:      "72.45",
		Metadata: map[string]string{
			"period":   "14",
			"avg_gain": "0.85",
			"avg_loss": "0.32",
		},
		Final:     true,
		Timestamp: time.Now().UTC(),
	}
}

func TestSignal_Validate(t *testing.T) {
	s := validSignal(t)
	if prob := s.Validate(); prob != nil {
		t.Fatalf("expected valid signal, got: %v", prob)
	}
}

func TestSignal_Validate_RequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(signal.Signal) signal.Signal
	}{
		{"empty type", func(s signal.Signal) signal.Signal { s.Type = ""; return s }},
		{"empty source", func(s signal.Signal) signal.Signal { s.Source = ""; return s }},
		{"zero instrument", func(s signal.Signal) signal.Signal {
			s.Instrument = instrument.CanonicalInstrument{}
			return s
		}},
		{"zero timeframe", func(s signal.Signal) signal.Signal { s.Timeframe = 0; return s }},
		{"negative timeframe", func(s signal.Signal) signal.Signal { s.Timeframe = -1; return s }},
		{"empty value", func(s signal.Signal) signal.Signal { s.Value = ""; return s }},
		{"zero timestamp", func(s signal.Signal) signal.Signal { s.Timestamp = time.Time{}; return s }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.mutate(validSignal(t))
			if prob := s.Validate(); prob == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestSignal_Validate_NilMetadata(t *testing.T) {
	s := validSignal(t)
	s.Metadata = nil
	if prob := s.Validate(); prob != nil {
		t.Fatalf("nil metadata should be valid, got: %v", prob)
	}
}

func TestSignal_PartitionKey(t *testing.T) {
	s := validSignal(t)
	got := s.PartitionKey()
	want := "binancef.btcusdt.300"
	if got != want {
		t.Fatalf("PartitionKey() = %q, want %q", got, want)
	}
}

func TestSignal_DeduplicationKey(t *testing.T) {
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	s := validSignal(t)
	s.Timestamp = ts
	got := s.DeduplicationKey()
	// P4.1.11.a: dedup key precision raised to UnixNano (see signal.go doc).
	want := fmt.Sprintf("sig:rsi:binancef:btcusdt:300:%d", ts.UnixNano())
	if got != want {
		t.Fatalf("DeduplicationKey() = %q, want %q", got, want)
	}
}

func TestSignal_PartitionKey_MultiSymbolIsolation(t *testing.T) {
	ts := time.Now().UTC()
	bases := []string{"BTC", "ETH", "SOL"}
	keys := make(map[string]string)

	for _, base := range bases {
		inst, prob := instrument.New(base, "USDT", instrument.ContractPerpetual)
		if prob != nil {
			t.Fatalf("setup %s/USDT: %v", base, prob)
		}
		s := signal.Signal{
			Type:       "rsi",
			Source:     "binancef",
			Instrument: inst,
			Timeframe:  60,
			Value:      "55.00",
			Timestamp:  ts,
			Final:      true,
		}
		key := s.PartitionKey()
		if prev, exists := keys[key]; exists {
			t.Fatalf("partition key collision: %q used by both %s and %s", key, prev, base)
		}
		keys[key] = base

		// Verify key contains the venue symbol form
		want := "binancef." + s.VenueSymbol() + ".60"
		if key != want {
			t.Errorf("base %s: PartitionKey() = %q, want %q", base, key, want)
		}
	}
}

func TestSignal_DeduplicationKey_MultiSymbolIsolation(t *testing.T) {
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	bases := []string{"BTC", "ETH"}
	keys := make(map[string]string)

	for _, base := range bases {
		inst, prob := instrument.New(base, "USDT", instrument.ContractPerpetual)
		if prob != nil {
			t.Fatalf("setup %s/USDT: %v", base, prob)
		}
		s := signal.Signal{
			Type:       "rsi",
			Source:     "binancef",
			Instrument: inst,
			Timeframe:  60,
			Value:      "55.00",
			Timestamp:  ts,
			Final:      true,
		}
		key := s.DeduplicationKey()
		if prev, exists := keys[key]; exists {
			t.Fatalf("dedup key collision: %q used by both %s and %s", key, prev, base)
		}
		keys[key] = base
	}
}

func TestSignal_PartitionKey_TimeframeIsolation(t *testing.T) {
	timeframes := []int{60, 300}
	keys := make(map[string]int)

	for _, tf := range timeframes {
		s := signal.Signal{
			Type:       "rsi",
			Source:     "binancef",
			Instrument: btcUSDTPerp(t),
			Timeframe:  tf,
			Value:      "55.00",
			Timestamp:  time.Now().UTC(),
			Final:      true,
		}
		key := s.PartitionKey()
		if prev, exists := keys[key]; exists {
			t.Fatalf("partition key collision: %q used by both tf=%d and tf=%d", key, prev, tf)
		}
		keys[key] = tf
	}
}
