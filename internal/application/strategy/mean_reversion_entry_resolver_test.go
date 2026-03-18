package strategy_test

import (
	"testing"
	"time"

	appstrategy "internal/application/strategy"
	domainstrategy "internal/domain/strategy"
)

func TestMeanReversionEntryResolver_Triggered(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.8500", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Direction != domainstrategy.DirectionLong {
		t.Fatalf("expected long, got %s", s.Direction)
	}
	if s.Type != "mean_reversion_entry" {
		t.Fatalf("expected type mean_reversion_entry, got %s", s.Type)
	}
	if s.Confidence != "0.8500" {
		t.Fatalf("expected confidence 0.8500, got %s", s.Confidence)
	}
	if s.Parameters["entry"] != "market" {
		t.Fatalf("expected entry=market, got %s", s.Parameters["entry"])
	}
	if s.Parameters["target_offset"] != "0.02" {
		t.Fatalf("expected target_offset=0.02, got %s", s.Parameters["target_offset"])
	}
	if s.Parameters["stop_offset"] != "0.01" {
		t.Fatalf("expected stop_offset=0.01, got %s", s.Parameters["stop_offset"])
	}
	if !s.Final {
		t.Fatal("expected final=true")
	}
	if len(s.Decisions) != 1 {
		t.Fatalf("expected 1 decision input, got %d", len(s.Decisions))
	}
	if s.Decisions[0].Type != "rsi_oversold" || s.Decisions[0].Outcome != "triggered" {
		t.Fatalf("unexpected decision input: %+v", s.Decisions[0])
	}
}

func TestMeanReversionEntryResolver_NotTriggered(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("rsi_oversold", "not_triggered", "0.7500", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Direction != domainstrategy.DirectionFlat {
		t.Fatalf("expected flat, got %s", s.Direction)
	}
	if s.Confidence != "0.0000" {
		t.Fatalf("expected confidence 0.0000, got %s", s.Confidence)
	}
	if len(s.Parameters) != 0 {
		t.Fatalf("expected no parameters for flat, got %v", s.Parameters)
	}
}

func TestMeanReversionEntryResolver_Insufficient(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("rsi_oversold", "insufficient", "0.0000", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Direction != domainstrategy.DirectionFlat {
		t.Fatalf("expected flat, got %s", s.Direction)
	}
	if s.Metadata["reason"] != "insufficient_data" {
		t.Fatalf("expected reason=insufficient_data, got %s", s.Metadata["reason"])
	}
}

func TestMeanReversionEntryResolver_UnknownOutcome(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	_, ok := resolver.Resolve("rsi_oversold", "unknown", "0.5000", 60, now)
	if ok {
		t.Fatal("expected resolution to fail for unknown outcome")
	}
}

func TestMeanReversionEntryResolver_InvalidConfidence(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	_, ok := resolver.Resolve("rsi_oversold", "triggered", "not-a-number", 60, now)
	if ok {
		t.Fatal("expected resolution to fail for invalid confidence")
	}
}

func TestMeanReversionEntryResolver_TimestampPreserved(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.8500", 60, ts)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if !s.Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, s.Timestamp)
	}
}

func TestMeanReversionEntryResolver_Validation(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.8500", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if prob := s.Validate(); prob != nil {
		t.Fatalf("strategy should be valid, got: %s", prob.Message)
	}
}

func TestMeanReversionEntryResolver_PartitionKey(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.8500", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if got := s.PartitionKey(); got != "binancef.btcusdt.60" {
		t.Fatalf("expected binancef.btcusdt.60, got %s", got)
	}
}

func TestMeanReversionEntryResolver_DecisionInputPreserved(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 300)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.9000", 300, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if len(s.Decisions) != 1 {
		t.Fatalf("expected 1 decision input, got %d", len(s.Decisions))
	}
	di := s.Decisions[0]
	if di.Type != "rsi_oversold" {
		t.Errorf("expected decision type rsi_oversold, got %s", di.Type)
	}
	if di.Outcome != "triggered" {
		t.Errorf("expected decision outcome triggered, got %s", di.Outcome)
	}
	if di.Confidence != "0.9000" {
		t.Errorf("expected decision confidence 0.9000, got %s", di.Confidence)
	}
	if di.Timeframe != 300 {
		t.Errorf("expected decision timeframe 300, got %d", di.Timeframe)
	}
}
