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

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.8500", "low", "RSI 28.50 below oversold threshold 30.0 (distance 5.0%); severity low", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Direction != domainstrategy.DirectionLong {
		t.Fatalf("expected long, got %s", s.Direction)
	}
	if s.Type != "mean_reversion_entry" {
		t.Fatalf("expected type mean_reversion_entry, got %s", s.Type)
	}
	// Low severity scales confidence by 0.80: 0.8500 * 0.80 = 0.6800
	if s.Confidence != "0.6800" {
		t.Fatalf("expected confidence 0.6800 (0.8500×0.80 for low severity), got %s", s.Confidence)
	}
	if s.Parameters["entry"] != "market" {
		t.Fatalf("expected entry=market, got %s", s.Parameters["entry"])
	}
	// Low severity: target_offset = 0.02 * 0.75 = 0.015 → "0.01"
	if s.Parameters["target_offset"] != "0.01" {
		t.Fatalf("expected target_offset=0.01 (0.02×0.75 for low severity), got %s", s.Parameters["target_offset"])
	}
	// Low severity: stop_offset = 0.01 * 1.50 = 0.015 → "0.01"
	if s.Parameters["stop_offset"] != "0.01" {
		t.Fatalf("expected stop_offset=0.01 (0.01×1.50 for low severity), got %s", s.Parameters["stop_offset"])
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

	s, ok := resolver.Resolve("rsi_oversold", "not_triggered", "0.7500", "none", "RSI 65.00 above oversold threshold 30.0; not oversold", 60, now)
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

	s, ok := resolver.Resolve("rsi_oversold", "insufficient", "0.0000", "", "", 60, now)
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

	_, ok := resolver.Resolve("rsi_oversold", "unknown", "0.5000", "", "", 60, now)
	if ok {
		t.Fatal("expected resolution to fail for unknown outcome")
	}
}

func TestMeanReversionEntryResolver_InvalidConfidence(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	_, ok := resolver.Resolve("rsi_oversold", "triggered", "not-a-number", "low", "test", 60, now)
	if ok {
		t.Fatal("expected resolution to fail for invalid confidence")
	}
}

func TestMeanReversionEntryResolver_TimestampPreserved(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.8500", "low", "RSI below threshold", 60, ts)
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

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.8500", "moderate", "RSI below threshold", 60, now)
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

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.8500", "low", "test", 60, now)
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

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.9000", "high", "RSI 5.00 below oversold threshold 30.0 (distance 83.3%); severity high", 300, now)
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
		t.Errorf("expected decision confidence 0.9000 (raw, not scaled), got %s", di.Confidence)
	}
	if di.Severity != "high" {
		t.Errorf("expected decision severity high, got %s", di.Severity)
	}
	if di.Rationale == "" {
		t.Error("expected decision rationale to be non-empty")
	}
	if di.Timeframe != 300 {
		t.Errorf("expected decision timeframe 300, got %d", di.Timeframe)
	}
}

func TestMeanReversionEntryResolver_DecisionRationaleInMetadata(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	rationale := "RSI 25.00 below oversold threshold 30.0 (distance 16.7%); severity low"
	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.8500", "low", rationale, 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Metadata["decision_rationale"] != rationale {
		t.Fatalf("expected decision_rationale in metadata, got %v", s.Metadata)
	}
}

func TestMeanReversionEntryResolver_EmptyRationaleNotInMetadata(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.8500", "", "", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if _, exists := s.Metadata["decision_rationale"]; exists {
		t.Fatal("expected no decision_rationale in metadata when rationale is empty")
	}
}

func TestMeanReversionEntryResolver_SeverityPreservedForAllOutcomes(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	tests := []struct {
		outcome  string
		severity string
	}{
		{"triggered", "high"},
		{"not_triggered", "none"},
		{"insufficient", ""},
	}

	for _, tt := range tests {
		s, ok := resolver.Resolve("rsi_oversold", tt.outcome, "0.5000", tt.severity, "test rationale", 60, now)
		if !ok {
			t.Fatalf("expected resolution to succeed for outcome %s", tt.outcome)
		}
		if s.Decisions[0].Severity != tt.severity {
			t.Errorf("outcome %s: expected severity %q, got %q", tt.outcome, tt.severity, s.Decisions[0].Severity)
		}
	}
}

// --- S250 behavioral activation tests ---

func TestMeanReversionEntryResolver_SeverityScalesConfidence(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	tests := []struct {
		name               string
		severity           string
		rawConfidence      string
		expectedConfidence string
	}{
		{"high severity → full confidence", "high", "0.9000", "0.9000"},      // ×1.00
		{"moderate severity → 0.90×", "moderate", "0.9000", "0.8100"},        // ×0.90
		{"low severity → 0.80×", "low", "0.9000", "0.7200"},                  // ×0.80
		{"unknown severity → neutral (1.0×)", "unknown", "0.9000", "0.9000"}, // default
		{"empty severity → neutral (1.0×)", "", "0.9000", "0.9000"},          // default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := resolver.Resolve("rsi_oversold", "triggered", tt.rawConfidence, tt.severity, "test", 60, now)
			if !ok {
				t.Fatal("expected resolution to succeed")
			}
			if s.Confidence != tt.expectedConfidence {
				t.Errorf("expected confidence %s, got %s", tt.expectedConfidence, s.Confidence)
			}
		})
	}
}

func TestMeanReversionEntryResolver_SeverityAdjustsParameters(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	tests := []struct {
		name                 string
		severity             string
		expectedTargetOffset string // base=0.02
		expectedStopOffset   string // base=0.01
	}{
		{"high severity → wider target, tighter stop", "high", "0.03", "0.01"}, // 0.02×1.50, 0.01×0.75
		{"moderate severity → default params", "moderate", "0.02", "0.01"},     // 0.02×1.00, 0.01×1.00
		{"low severity → smaller target, wider stop", "low", "0.01", "0.01"},   // 0.02×0.75=0.015→0.01, 0.01×1.50=0.015→0.01
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.8000", tt.severity, "test", 60, now)
			if !ok {
				t.Fatal("expected resolution to succeed")
			}
			if s.Parameters["target_offset"] != tt.expectedTargetOffset {
				t.Errorf("target_offset: want %s, got %s", tt.expectedTargetOffset, s.Parameters["target_offset"])
			}
			if s.Parameters["stop_offset"] != tt.expectedStopOffset {
				t.Errorf("stop_offset: want %s, got %s", tt.expectedStopOffset, s.Parameters["stop_offset"])
			}
		})
	}
}

func TestMeanReversionEntryResolver_DecisionTypeInMetadata(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.8000", "moderate", "test", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Metadata["decision_type"] != "rsi_oversold" {
		t.Errorf("expected decision_type=rsi_oversold in metadata, got %s", s.Metadata["decision_type"])
	}
	if s.Metadata["decision_severity"] != "moderate" {
		t.Errorf("expected decision_severity=moderate in metadata, got %s", s.Metadata["decision_severity"])
	}
}

func TestMeanReversionEntryResolver_DecisionInputPreservesRawConfidence(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.9000", "low", "test", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}

	// DecisionInput must carry the original (raw) confidence, not the scaled one.
	if s.Decisions[0].Confidence != "0.9000" {
		t.Errorf("DecisionInput.Confidence should be raw (0.9000), got %s", s.Decisions[0].Confidence)
	}
	// Strategy confidence should be scaled.
	if s.Confidence != "0.7200" {
		t.Errorf("Strategy.Confidence should be scaled (0.7200), got %s", s.Confidence)
	}
}

func TestMeanReversionEntryResolver_NotTriggeredHasDecisionContext(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("rsi_oversold", "not_triggered", "0.7500", "none", "RSI above threshold", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Metadata["decision_type"] != "rsi_oversold" {
		t.Errorf("expected decision_type in metadata for not_triggered, got %v", s.Metadata)
	}
	if s.Metadata["decision_severity"] != "none" {
		t.Errorf("expected decision_severity in metadata for not_triggered, got %v", s.Metadata)
	}
}

func TestMeanReversionEntryResolver_HighSeverityMaxAggression(t *testing.T) {
	resolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	// High severity with maximum confidence → aggressive strategy.
	s, ok := resolver.Resolve("rsi_oversold", "triggered", "0.9500", "high", "RSI extremely oversold", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}

	// High severity: confidence ×1.00 = 0.9500
	if s.Confidence != "0.9500" {
		t.Errorf("expected confidence 0.9500 (high=1.0×), got %s", s.Confidence)
	}
	// High severity: target 0.02×1.50 = 0.03 (wider target — expect bigger reversion)
	if s.Parameters["target_offset"] != "0.03" {
		t.Errorf("expected target_offset=0.03 (high severity), got %s", s.Parameters["target_offset"])
	}
	// High severity: stop 0.01×0.75 = 0.0075 → 0.01 (tighter stop — higher conviction)
	if s.Parameters["stop_offset"] != "0.01" {
		t.Errorf("expected stop_offset=0.01 (high severity), got %s", s.Parameters["stop_offset"])
	}
}
