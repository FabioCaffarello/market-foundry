package strategy_test

import (
	"fmt"
	"testing"
	"time"

	"internal/domain/instrument"
	"internal/domain/strategy"
)

func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

func mustInstrument(t *testing.T, base, quote string) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New(base, quote, instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

func validStrategy(t *testing.T) strategy.Strategy {
	t.Helper()
	return strategy.Strategy{
		Type:       "mean_reversion_entry",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Direction:  strategy.DirectionLong,
		Confidence: "0.85",
		Decisions: []strategy.DecisionInput{
			{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Severity: "low", Rationale: "RSI 28.50 below oversold threshold 30.0 (distance 5.0%); severity low", Timeframe: 60},
		},
		Parameters: map[string]string{"entry": "market", "target_offset": "0.02", "stop_offset": "0.01"},
		Final:      true,
		Timestamp:  time.Now().UTC(),
	}
}

func TestStrategy_Validate_Valid(t *testing.T) {
	s := validStrategy(t)
	if prob := s.Validate(); prob != nil {
		t.Fatalf("expected valid strategy, got: %s", prob.Message)
	}
}

func TestStrategy_Validate_EmptyType(t *testing.T) {
	s := validStrategy(t)
	s.Type = ""
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty type")
	}
}

func TestStrategy_Validate_EmptySource(t *testing.T) {
	s := validStrategy(t)
	s.Source = ""
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty source")
	}
}

func TestStrategy_Validate_EmptySymbol(t *testing.T) {
	s := validStrategy(t)
	s.Instrument = instrument.CanonicalInstrument{}
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty symbol")
	}
}

func TestStrategy_Validate_ZeroTimeframe(t *testing.T) {
	s := validStrategy(t)
	s.Timeframe = 0
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for zero timeframe")
	}
}

func TestStrategy_Validate_InvalidDirection(t *testing.T) {
	s := validStrategy(t)
	s.Direction = "invalid"
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for invalid direction")
	}
}

func TestStrategy_Validate_EmptyDirection(t *testing.T) {
	s := validStrategy(t)
	s.Direction = ""
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty direction")
	}
}

func TestStrategy_Validate_EmptyConfidence(t *testing.T) {
	s := validStrategy(t)
	s.Confidence = ""
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty confidence")
	}
}

func TestStrategy_Validate_ZeroTimestamp(t *testing.T) {
	s := validStrategy(t)
	s.Timestamp = time.Time{}
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for zero timestamp")
	}
}

func TestStrategy_Validate_NoDecisions(t *testing.T) {
	s := validStrategy(t)
	s.Decisions = nil
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty decisions")
	}
}

func TestStrategy_Validate_AllDirections(t *testing.T) {
	for _, dir := range []strategy.Direction{strategy.DirectionLong, strategy.DirectionShort, strategy.DirectionFlat} {
		s := validStrategy(t)
		s.Direction = dir
		if prob := s.Validate(); prob != nil {
			t.Fatalf("direction %s should be valid, got: %s", dir, prob.Message)
		}
	}
}

func TestStrategy_PartitionKey(t *testing.T) {
	s := strategy.Strategy{Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60}
	expected := "binancef.btc_usdt_perpetual.60"
	if got := s.PartitionKey(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestStrategy_MultiSymbol_PartitionKeyIsolation(t *testing.T) {
	insts := []instrument.CanonicalInstrument{
		mustInstrument(t, "BTC", "USDT"),
		mustInstrument(t, "ETH", "USDT"),
		mustInstrument(t, "SOL", "USDT"),
	}
	timeframes := []int{60, 300}
	seen := make(map[string]bool)

	for _, inst := range insts {
		for _, tf := range timeframes {
			s := validStrategy(t)
			s.Instrument = inst
			s.Timeframe = tf
			key := s.PartitionKey()
			if seen[key] {
				t.Fatalf("duplicate partition key: %s", key)
			}
			seen[key] = true
		}
	}
	if len(seen) != 6 {
		t.Fatalf("expected 6 unique keys, got %d", len(seen))
	}
}

func TestStrategy_MultiSymbol_DeduplicationKeyIsolation(t *testing.T) {
	insts := []instrument.CanonicalInstrument{
		mustInstrument(t, "BTC", "USDT"),
		mustInstrument(t, "ETH", "USDT"),
		mustInstrument(t, "SOL", "USDT"),
	}
	timeframes := []int{60, 300}
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	seen := make(map[string]bool)

	for _, inst := range insts {
		for _, tf := range timeframes {
			s := validStrategy(t)
			s.Instrument = inst
			s.Timeframe = tf
			s.Timestamp = ts
			key := s.DeduplicationKey()
			if seen[key] {
				t.Fatalf("duplicate deduplication key: %s", key)
			}
			seen[key] = true
		}
	}
	if len(seen) != 6 {
		t.Fatalf("expected 6 unique keys, got %d", len(seen))
	}
}

func TestStrategy_Validate_NegativeTimeframe(t *testing.T) {
	s := validStrategy(t)
	s.Timeframe = -1
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for negative timeframe")
	}
}

func TestStrategy_Validate_NilDecisions(t *testing.T) {
	s := validStrategy(t)
	s.Decisions = []strategy.DecisionInput{}
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty decisions slice")
	}
}

func TestStrategy_DeduplicationKey(t *testing.T) {
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	s := strategy.Strategy{
		Type:       "mean_reversion_entry",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Timestamp:  ts,
	}
	got := s.DeduplicationKey()
	prefix := "strat:mean_reversion_entry:binancef:btc_usdt_perpetual:60:"
	if got[:len(prefix)] != prefix {
		t.Fatalf("expected prefix %q, got %q", prefix, got)
	}
	// Verify the nanosecond timestamp suffix matches (P4.1.10:
	// precision raised from Unix() to UnixNano() to prevent silent
	// dedup drops under rapid same-second publishes).
	expectedSuffix := fmt.Sprintf("%d", ts.UnixNano())
	if got[len(prefix):] != expectedSuffix {
		t.Fatalf("expected suffix %q, got %q", expectedSuffix, got[len(prefix):])
	}
}
