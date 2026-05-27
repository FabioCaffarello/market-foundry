package strategy_test

import (
	"testing"
	"time"

	appstrategy "internal/application/strategy"
	domainstrategy "internal/domain/strategy"
)

func TestTrendFollowingEntryResolver_Triggered(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
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
	// Moderate severity scales confidence by 0.90: 0.7500 * 0.90 = 0.6750
	if s.Confidence != "0.6750" {
		t.Fatalf("expected confidence 0.6750 (0.7500×0.90 for moderate severity), got %s", s.Confidence)
	}
	if s.Parameters["entry"] != "market" {
		t.Fatalf("expected entry=market, got %s", s.Parameters["entry"])
	}
	// Moderate severity: trailing_stop_pct = 0.03 * 1.00 = 0.03
	if s.Parameters["trailing_stop_pct"] != "0.03" {
		t.Fatalf("expected trailing_stop_pct=0.03 (moderate severity ×1.0), got %s", s.Parameters["trailing_stop_pct"])
	}
	// Moderate severity: take_profit_pct = 0.05 * 1.00 = 0.05
	if s.Parameters["take_profit_pct"] != "0.05" {
		t.Fatalf("expected take_profit_pct=0.05 (moderate severity ×1.0), got %s", s.Parameters["take_profit_pct"])
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
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
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
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
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
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	_, ok := resolver.Resolve("ema_crossover", "unknown", "0.5000", "", "", 60, now)
	if ok {
		t.Fatal("expected resolution to fail for unknown outcome")
	}
}

func TestTrendFollowingEntryResolver_InvalidConfidence(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	_, ok := resolver.Resolve("ema_crossover", "triggered", "not-a-number", "moderate", "test", 60, now)
	if ok {
		t.Fatal("expected resolution to fail for invalid confidence")
	}
}

func TestTrendFollowingEntryResolver_TimestampPreserved(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
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
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
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
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
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
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 300)
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
		t.Errorf("expected decision confidence 0.7500 (raw, not scaled), got %s", di.Confidence)
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
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
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
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
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
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
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

// --- S250 behavioral activation tests ---

func TestTrendFollowingEntryResolver_SeverityScalesConfidence(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	tests := []struct {
		name               string
		severity           string
		rawConfidence      string
		expectedConfidence string
	}{
		{"high severity → full confidence", "high", "0.8000", "0.8000"},      // ×1.00
		{"moderate severity → 0.90×", "moderate", "0.8000", "0.7200"},        // ×0.90
		{"low severity → 0.80×", "low", "0.8000", "0.6400"},                  // ×0.80
		{"unknown severity → neutral (1.0×)", "unknown", "0.8000", "0.8000"}, // default
		{"empty severity → neutral (1.0×)", "", "0.8000", "0.8000"},          // default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := resolver.Resolve("ema_crossover", "triggered", tt.rawConfidence, tt.severity, "test", 60, now)
			if !ok {
				t.Fatal("expected resolution to succeed")
			}
			if s.Confidence != tt.expectedConfidence {
				t.Errorf("expected confidence %s, got %s", tt.expectedConfidence, s.Confidence)
			}
		})
	}
}

func TestTrendFollowingEntryResolver_SeverityAdjustsParameters(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	tests := []struct {
		name                    string
		severity                string
		expectedTrailingStopPct string // base=0.03
		expectedTakeProfitPct   string // base=0.05
	}{
		{"high severity → tighter trail, wider target", "high", "0.02", "0.08"}, // 0.03×0.75=0.0225→0.02, 0.05×1.50=0.075→0.08
		{"moderate severity → default params", "moderate", "0.03", "0.05"},      // 0.03×1.00, 0.05×1.00
		{"low severity → wider trail, smaller target", "low", "0.04", "0.04"},   // 0.03×1.50, 0.05×0.75 → 0.045→0.04, 0.0375→0.04
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := resolver.Resolve("ema_crossover", "triggered", "0.7500", tt.severity, "test", 60, now)
			if !ok {
				t.Fatal("expected resolution to succeed")
			}
			if s.Parameters["trailing_stop_pct"] != tt.expectedTrailingStopPct {
				t.Errorf("trailing_stop_pct: want %s, got %s", tt.expectedTrailingStopPct, s.Parameters["trailing_stop_pct"])
			}
			if s.Parameters["take_profit_pct"] != tt.expectedTakeProfitPct {
				t.Errorf("take_profit_pct: want %s, got %s", tt.expectedTakeProfitPct, s.Parameters["take_profit_pct"])
			}
		})
	}
}

func TestTrendFollowingEntryResolver_DecisionTypeInMetadata(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("ema_crossover", "triggered", "0.7500", "moderate", "test", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Metadata["decision_type"] != "ema_crossover" {
		t.Errorf("expected decision_type=ema_crossover in metadata, got %s", s.Metadata["decision_type"])
	}
	if s.Metadata["decision_severity"] != "moderate" {
		t.Errorf("expected decision_severity=moderate in metadata, got %s", s.Metadata["decision_severity"])
	}
}

func TestTrendFollowingEntryResolver_DecisionInputPreservesRawConfidence(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("ema_crossover", "triggered", "0.7500", "low", "test", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}

	// DecisionInput must carry the original (raw) confidence, not the scaled one.
	if s.Decisions[0].Confidence != "0.7500" {
		t.Errorf("DecisionInput.Confidence should be raw (0.7500), got %s", s.Decisions[0].Confidence)
	}
	// Strategy confidence should be scaled: 0.7500 × 0.80 = 0.6000
	if s.Confidence != "0.6000" {
		t.Errorf("Strategy.Confidence should be scaled (0.6000), got %s", s.Confidence)
	}
}

func TestTrendFollowingEntryResolver_NotTriggeredHasDecisionContext(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("ema_crossover", "not_triggered", "0.5000", "none", "EMA bearish", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Metadata["decision_type"] != "ema_crossover" {
		t.Errorf("expected decision_type in metadata for not_triggered, got %v", s.Metadata)
	}
	if s.Metadata["decision_severity"] != "none" {
		t.Errorf("expected decision_severity in metadata for not_triggered, got %v", s.Metadata)
	}
}

func TestTrendFollowingEntryResolver_HighSeverityMaxAggression(t *testing.T) {
	resolver := appstrategy.NewTrendFollowingEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("ema_crossover", "triggered", "0.9000", "high", "Strong bullish crossover", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}

	// High severity: confidence ×1.00 = 0.9000
	if s.Confidence != "0.9000" {
		t.Errorf("expected confidence 0.9000 (high=1.0×), got %s", s.Confidence)
	}
	// High severity: trailing_stop 0.03×0.75 = 0.0225 → 0.02 (tighter — ride the trend closer)
	if s.Parameters["trailing_stop_pct"] != "0.02" {
		t.Errorf("expected trailing_stop_pct=0.02 (high severity), got %s", s.Parameters["trailing_stop_pct"])
	}
	// High severity: take_profit 0.05×1.50 = 0.075 → 0.08 (wider — expect bigger move)
	if s.Parameters["take_profit_pct"] != "0.08" {
		t.Errorf("expected take_profit_pct=0.08 (high severity), got %s", s.Parameters["take_profit_pct"])
	}
}
