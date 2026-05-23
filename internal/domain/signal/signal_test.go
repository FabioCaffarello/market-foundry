package signal_test

import (
	"fmt"
	"testing"
	"time"

	"internal/domain/signal"
)

func validSignal() signal.Signal {
	return signal.Signal{
		Type:      "rsi",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 300,
		Value:     "72.45",
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
	s := validSignal()
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
		{"empty symbol", func(s signal.Signal) signal.Signal { s.Symbol = ""; return s }},
		{"zero timeframe", func(s signal.Signal) signal.Signal { s.Timeframe = 0; return s }},
		{"negative timeframe", func(s signal.Signal) signal.Signal { s.Timeframe = -1; return s }},
		{"empty value", func(s signal.Signal) signal.Signal { s.Value = ""; return s }},
		{"zero timestamp", func(s signal.Signal) signal.Signal { s.Timestamp = time.Time{}; return s }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.mutate(validSignal())
			if prob := s.Validate(); prob == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestSignal_Validate_NilMetadata(t *testing.T) {
	s := validSignal()
	s.Metadata = nil
	if prob := s.Validate(); prob != nil {
		t.Fatalf("nil metadata should be valid, got: %v", prob)
	}
}

func TestSignal_PartitionKey(t *testing.T) {
	s := validSignal()
	got := s.PartitionKey()
	want := "binancef.btcusdt.300"
	if got != want {
		t.Fatalf("PartitionKey() = %q, want %q", got, want)
	}
}

func TestSignal_DeduplicationKey(t *testing.T) {
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	s := validSignal()
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
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	keys := make(map[string]string)

	for _, sym := range symbols {
		s := signal.Signal{
			Type:      "rsi",
			Source:    "binancef",
			Symbol:    sym,
			Timeframe: 60,
			Value:     "55.00",
			Timestamp: ts,
			Final:     true,
		}
		key := s.PartitionKey()
		if prev, exists := keys[key]; exists {
			t.Fatalf("partition key collision: %q used by both %s and %s", key, prev, sym)
		}
		keys[key] = sym

		// Verify key contains the symbol
		want := "binancef." + sym + ".60"
		if key != want {
			t.Errorf("symbol %s: PartitionKey() = %q, want %q", sym, key, want)
		}
	}
}

func TestSignal_DeduplicationKey_MultiSymbolIsolation(t *testing.T) {
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	symbols := []string{"btcusdt", "ethusdt"}
	keys := make(map[string]string)

	for _, sym := range symbols {
		s := signal.Signal{
			Type:      "rsi",
			Source:    "binancef",
			Symbol:    sym,
			Timeframe: 60,
			Value:     "55.00",
			Timestamp: ts,
			Final:     true,
		}
		key := s.DeduplicationKey()
		if prev, exists := keys[key]; exists {
			t.Fatalf("dedup key collision: %q used by both %s and %s", key, prev, sym)
		}
		keys[key] = sym
	}
}

func TestSignal_PartitionKey_TimeframeIsolation(t *testing.T) {
	timeframes := []int{60, 300}
	keys := make(map[string]int)

	for _, tf := range timeframes {
		s := signal.Signal{
			Type:      "rsi",
			Source:    "binancef",
			Symbol:    "btcusdt",
			Timeframe: tf,
			Value:     "55.00",
			Timestamp: time.Now().UTC(),
			Final:     true,
		}
		key := s.PartitionKey()
		if prev, exists := keys[key]; exists {
			t.Fatalf("partition key collision: %q used by both tf=%d and tf=%d", key, prev, tf)
		}
		keys[key] = tf
	}
}
