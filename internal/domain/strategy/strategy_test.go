package strategy_test

import (
	"fmt"
	"testing"
	"time"

	"internal/domain/strategy"
)

func validStrategy() strategy.Strategy {
	return strategy.Strategy{
		Type:       "mean_reversion_entry",
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Direction:  strategy.DirectionLong,
		Confidence: "0.85",
		Decisions: []strategy.DecisionInput{
			{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Timeframe: 60},
		},
		Parameters: map[string]string{"entry": "market", "target_offset": "0.02", "stop_offset": "0.01"},
		Final:      true,
		Timestamp:  time.Now().UTC(),
	}
}

func TestStrategy_Validate_Valid(t *testing.T) {
	s := validStrategy()
	if prob := s.Validate(); prob != nil {
		t.Fatalf("expected valid strategy, got: %s", prob.Message)
	}
}

func TestStrategy_Validate_EmptyType(t *testing.T) {
	s := validStrategy()
	s.Type = ""
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty type")
	}
}

func TestStrategy_Validate_EmptySource(t *testing.T) {
	s := validStrategy()
	s.Source = ""
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty source")
	}
}

func TestStrategy_Validate_EmptySymbol(t *testing.T) {
	s := validStrategy()
	s.Symbol = ""
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty symbol")
	}
}

func TestStrategy_Validate_ZeroTimeframe(t *testing.T) {
	s := validStrategy()
	s.Timeframe = 0
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for zero timeframe")
	}
}

func TestStrategy_Validate_InvalidDirection(t *testing.T) {
	s := validStrategy()
	s.Direction = "invalid"
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for invalid direction")
	}
}

func TestStrategy_Validate_EmptyDirection(t *testing.T) {
	s := validStrategy()
	s.Direction = ""
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty direction")
	}
}

func TestStrategy_Validate_EmptyConfidence(t *testing.T) {
	s := validStrategy()
	s.Confidence = ""
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty confidence")
	}
}

func TestStrategy_Validate_ZeroTimestamp(t *testing.T) {
	s := validStrategy()
	s.Timestamp = time.Time{}
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for zero timestamp")
	}
}

func TestStrategy_Validate_NoDecisions(t *testing.T) {
	s := validStrategy()
	s.Decisions = nil
	if prob := s.Validate(); prob == nil {
		t.Fatal("expected validation error for empty decisions")
	}
}

func TestStrategy_Validate_AllDirections(t *testing.T) {
	for _, dir := range []strategy.Direction{strategy.DirectionLong, strategy.DirectionShort, strategy.DirectionFlat} {
		s := validStrategy()
		s.Direction = dir
		if prob := s.Validate(); prob != nil {
			t.Fatalf("direction %s should be valid, got: %s", dir, prob.Message)
		}
	}
}

func TestStrategy_PartitionKey(t *testing.T) {
	s := strategy.Strategy{Source: "binancef", Symbol: "btcusdt", Timeframe: 60}
	expected := "binancef.btcusdt.60"
	if got := s.PartitionKey(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestStrategy_DeduplicationKey(t *testing.T) {
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	s := strategy.Strategy{
		Type:      "mean_reversion_entry",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Timestamp: ts,
	}
	got := s.DeduplicationKey()
	prefix := "strat:mean_reversion_entry:binancef:btcusdt:60:"
	if got[:len(prefix)] != prefix {
		t.Fatalf("expected prefix %q, got %q", prefix, got)
	}
	// Verify the unix timestamp suffix matches.
	expectedSuffix := fmt.Sprintf("%d", ts.Unix())
	if got[len(prefix):] != expectedSuffix {
		t.Fatalf("expected suffix %q, got %q", expectedSuffix, got[len(prefix):])
	}
}
