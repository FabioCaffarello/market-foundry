package strategy_test

import (
	"testing"
	"time"

	appstrategy "internal/application/strategy"
	domainstrategy "internal/domain/strategy"
)

func TestTrendFollowingEntryResolver_Triggered(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("ema_crossover", "triggered", "0.7500", "moderate", "EMA crossover bullish: fast EMA above slow EMA on 60s timeframe; trend confirmation signal", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Direction != domainstrategy.DirectionLong {
		t.Fatalf("expected long, got %s", s.Direction)
	}
	if s.Type != "trend_following_entry" {
		t.Fatalf("expected type trend_following_entry, got %s", s.Type)
	}
	if s.Confidence != "0.7500" {
		t.Fatalf("expected confidence 0.7500, got %s", s.Confidence)
	}
	if s.Parameters["entry"] != "market" {
		t.Fatalf("expected entry=market, got %s", s.Parameters["entry"])
	}
	if s.Parameters["trailing_stop_pct"] != "0.03" {
		t.Fatalf("expected trailing_stop_pct=0.03, got %s", s.Parameters["trailing_stop_pct"])
	}
	if s.Parameters["take_profit_pct"] != "0.05" {
		t.Fatalf("expected take_profit_pct=0.05, got %s", s.Parameters["take_profit_pct"])
	}
	if !s.Final {
		t.Fatal("expected final=true")
	}
	if len(s.Decisions) != 1 {
		t.Fatalf("expected 1 decision input, got %d", len(s.Decisions))
	}
	if s.Decisions[0].Type != "ema_crossover" || s.Decisions[0].Outcome != "triggered" {
		t.Fatalf("unexpected decision input: %+v", s.Decisions[0])
	}
}

func TestTrendFollowingEntryResolver_NotTriggered(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("ema_crossover", "not_triggered", "0.7500", "none", "EMA crossover bearish", 60, now)
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

func TestTrendFollowingEntryResolver_Insufficient(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("ema_crossover", "insufficient", "0.0000", "", "", 60, now)
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

func TestTrendFollowingEntryResolver_UnknownOutcome(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	_, ok := resolver.Resolve("ema_crossover", "unknown", "0.5000", "", "", 60, now)
	if ok {
		t.Fatal("expected resolution to fail for unknown outcome")
	}
}

func TestTrendFollowingEntryResolver_InvalidConfidence(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	_, ok := resolver.Resolve("ema_crossover", "triggered", "not-a-number", "moderate", "test", 60, now)
	if ok {
		t.Fatal("expected resolution to fail for invalid confidence")
	}
}

func TestTrendFollowingEntryResolver_TimestampPreserved(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolver("binancef", "btcusdt", 60)
	ts := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)

	s, ok := resolver.Resolve("ema_crossover", "triggered", "0.7500", "moderate", "EMA bullish", 60, ts)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if !s.Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, s.Timestamp)
	}
}

func TestTrendFollowingEntryResolver_Validation(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("ema_crossover", "triggered", "0.7500", "moderate", "EMA bullish", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if prob := s.Validate(); prob != nil {
		t.Fatalf("strategy should be valid, got: %s", prob.Message)
	}
}

func TestTrendFollowingEntryResolver_PartitionKey(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("ema_crossover", "triggered", "0.7500", "moderate", "test", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if got := s.PartitionKey(); got != "binancef.btcusdt.60" {
		t.Fatalf("expected binancef.btcusdt.60, got %s", got)
	}
}

func TestTrendFollowingEntryResolver_DecisionInputPreserved(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolver("binancef", "btcusdt", 300)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("ema_crossover", "triggered", "0.7500", "moderate", "EMA crossover bullish: fast EMA above slow EMA on 300s timeframe; trend confirmation signal", 300, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if len(s.Decisions) != 1 {
		t.Fatalf("expected 1 decision input, got %d", len(s.Decisions))
	}
	di := s.Decisions[0]
	if di.Type != "ema_crossover" {
		t.Errorf("expected decision type ema_crossover, got %s", di.Type)
	}
	if di.Outcome != "triggered" {
		t.Errorf("expected decision outcome triggered, got %s", di.Outcome)
	}
	if di.Confidence != "0.7500" {
		t.Errorf("expected decision confidence 0.7500, got %s", di.Confidence)
	}
	if di.Severity != "moderate" {
		t.Errorf("expected decision severity moderate, got %s", di.Severity)
	}
	if di.Rationale == "" {
		t.Error("expected decision rationale to be non-empty")
	}
	if di.Timeframe != 300 {
		t.Errorf("expected decision timeframe 300, got %d", di.Timeframe)
	}
}

func TestTrendFollowingEntryResolver_DecisionRationaleInMetadata(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	rationale := "EMA crossover bullish: fast EMA above slow EMA on 60s timeframe; trend confirmation signal"
	s, ok := resolver.Resolve("ema_crossover", "triggered", "0.7500", "moderate", rationale, 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Metadata["decision_rationale"] != rationale {
		t.Fatalf("expected decision_rationale in metadata, got %v", s.Metadata)
	}
}

func TestTrendFollowingEntryResolver_EmptyRationaleNotInMetadata(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("ema_crossover", "triggered", "0.7500", "", "", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if _, exists := s.Metadata["decision_rationale"]; exists {
		t.Fatal("expected no decision_rationale in metadata when rationale is empty")
	}
}

func TestTrendFollowingEntryResolver_SeverityPreservedForAllOutcomes(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	tests := []struct {
		outcome  string
		severity string
	}{
		{"triggered", "moderate"},
		{"not_triggered", "none"},
		{"insufficient", ""},
	}

	for _, tt := range tests {
		s, ok := resolver.Resolve("ema_crossover", tt.outcome, "0.5000", tt.severity, "test rationale", 60, now)
		if !ok {
			t.Fatalf("expected resolution to succeed for outcome %s", tt.outcome)
		}
		if s.Decisions[0].Severity != tt.severity {
			t.Errorf("outcome %s: expected severity %q, got %q", tt.outcome, tt.severity, s.Decisions[0].Severity)
		}
	}
}
