package strategy_test

import (
	"testing"
	"time"

	appstrategy "internal/application/strategy"
	domainstrategy "internal/domain/strategy"
)

func TestSqueezeBreakoutEntryResolver_Triggered(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("bollinger_squeeze", "triggered", "0.7500", "moderate", "Bollinger squeeze detected: bandwidth narrowing below threshold on 60s timeframe", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Direction != domainstrategy.DirectionLong {
		t.Fatalf("expected long, got %s", s.Direction)
	}
	if s.Type != "squeeze_breakout_entry" {
		t.Fatalf("expected type squeeze_breakout_entry, got %s", s.Type)
	}
	// Moderate severity scales confidence by 0.90: 0.7500 * 0.90 = 0.6750
	if s.Confidence != "0.6750" {
		t.Fatalf("expected confidence 0.6750 (0.7500×0.90 for moderate severity), got %s", s.Confidence)
	}
	if s.Parameters["entry"] != "market" {
		t.Fatalf("expected entry=market, got %s", s.Parameters["entry"])
	}
	// Moderate severity: breakout_target_pct = 0.04 * 1.00 = 0.04
	if s.Parameters["breakout_target_pct"] != "0.04" {
		t.Fatalf("expected breakout_target_pct=0.04 (moderate severity ×1.0), got %s", s.Parameters["breakout_target_pct"])
	}
	// Moderate severity: breakout_stop_pct = 0.015 * 1.00 = 0.015 → "0.01" (2 decimal, rounds down)
	if s.Parameters["breakout_stop_pct"] != "0.01" {
		t.Fatalf("expected breakout_stop_pct=0.01 (moderate severity ×1.0), got %s", s.Parameters["breakout_stop_pct"])
	}
	if !s.Final {
		t.Fatal("expected final=true")
	}
	if len(s.Decisions) != 1 {
		t.Fatalf("expected 1 decision input, got %d", len(s.Decisions))
	}
	if s.Decisions[0].Type != "bollinger_squeeze" || s.Decisions[0].Outcome != "triggered" {
		t.Fatalf("unexpected decision input: %+v", s.Decisions[0])
	}
}

func TestSqueezeBreakoutEntryResolver_NotTriggered(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("bollinger_squeeze", "not_triggered", "0.7500", "none", "Bandwidth above threshold", 60, now)
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

func TestSqueezeBreakoutEntryResolver_Insufficient(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("bollinger_squeeze", "insufficient", "0.0000", "", "", 60, now)
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

func TestSqueezeBreakoutEntryResolver_UnknownOutcome(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	_, ok := resolver.Resolve("bollinger_squeeze", "unknown", "0.5000", "", "", 60, now)
	if ok {
		t.Fatal("expected resolution to fail for unknown outcome")
	}
}

func TestSqueezeBreakoutEntryResolver_InvalidConfidence(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	_, ok := resolver.Resolve("bollinger_squeeze", "triggered", "not-a-number", "moderate", "test", 60, now)
	if ok {
		t.Fatal("expected resolution to fail for invalid confidence")
	}
}

func TestSqueezeBreakoutEntryResolver_TimestampPreserved(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	ts := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)

	s, ok := resolver.Resolve("bollinger_squeeze", "triggered", "0.7500", "moderate", "squeeze detected", 60, ts)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if !s.Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, s.Timestamp)
	}
}

func TestSqueezeBreakoutEntryResolver_Validation(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("bollinger_squeeze", "triggered", "0.7500", "moderate", "squeeze detected", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if prob := s.Validate(); prob != nil {
		t.Fatalf("strategy should be valid, got: %s", prob.Message)
	}
}

func TestSqueezeBreakoutEntryResolver_PartitionKey(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("bollinger_squeeze", "triggered", "0.7500", "moderate", "test", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if got := s.PartitionKey(); got != "binancef.btcusdt.60" {
		t.Fatalf("expected binancef.btcusdt.60, got %s", got)
	}
}

func TestSqueezeBreakoutEntryResolver_DecisionInputPreserved(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 300)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("bollinger_squeeze", "triggered", "0.7500", "moderate", "Bollinger squeeze detected: bandwidth narrowing below threshold on 300s timeframe", 300, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if len(s.Decisions) != 1 {
		t.Fatalf("expected 1 decision input, got %d", len(s.Decisions))
	}
	di := s.Decisions[0]
	if di.Type != "bollinger_squeeze" {
		t.Errorf("expected decision type bollinger_squeeze, got %s", di.Type)
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

func TestSqueezeBreakoutEntryResolver_DecisionRationaleInMetadata(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	rationale := "Bollinger squeeze detected: bandwidth narrowing below threshold on 60s timeframe"
	s, ok := resolver.Resolve("bollinger_squeeze", "triggered", "0.7500", "moderate", rationale, 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Metadata["decision_rationale"] != rationale {
		t.Fatalf("expected decision_rationale in metadata, got %v", s.Metadata)
	}
}

func TestSqueezeBreakoutEntryResolver_EmptyRationaleNotInMetadata(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("bollinger_squeeze", "triggered", "0.7500", "", "", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if _, exists := s.Metadata["decision_rationale"]; exists {
		t.Fatal("expected no decision_rationale in metadata when rationale is empty")
	}
}

func TestSqueezeBreakoutEntryResolver_SeverityPreservedForAllOutcomes(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
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
		s, ok := resolver.Resolve("bollinger_squeeze", tt.outcome, "0.5000", tt.severity, "test rationale", 60, now)
		if !ok {
			t.Fatalf("expected resolution to succeed for outcome %s", tt.outcome)
		}
		if s.Decisions[0].Severity != tt.severity {
			t.Errorf("outcome %s: expected severity %q, got %q", tt.outcome, tt.severity, s.Decisions[0].Severity)
		}
	}
}

// --- S250 behavioral activation tests ---

func TestSqueezeBreakoutEntryResolver_SeverityScalesConfidence(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
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
			s, ok := resolver.Resolve("bollinger_squeeze", "triggered", tt.rawConfidence, tt.severity, "test", 60, now)
			if !ok {
				t.Fatal("expected resolution to succeed")
			}
			if s.Confidence != tt.expectedConfidence {
				t.Errorf("expected confidence %s, got %s", tt.expectedConfidence, s.Confidence)
			}
		})
	}
}

func TestSqueezeBreakoutEntryResolver_SeverityAdjustsParameters(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	tests := []struct {
		name                      string
		severity                  string
		expectedBreakoutTargetPct string // base=0.04
		expectedBreakoutStopPct   string // base=0.015
	}{
		{"high severity → wider target, tighter stop", "high", "0.06", "0.01"}, // 0.04×1.50=0.06, 0.015×0.75=0.01125→0.01
		{"moderate severity → default params", "moderate", "0.04", "0.01"},     // 0.04×1.00, 0.015×1.00=0.015→0.01
		{"low severity → smaller target, wider stop", "low", "0.03", "0.02"},   // 0.04×0.75=0.03, 0.015×1.50=0.0225→0.02
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := resolver.Resolve("bollinger_squeeze", "triggered", "0.7500", tt.severity, "test", 60, now)
			if !ok {
				t.Fatal("expected resolution to succeed")
			}
			if s.Parameters["breakout_target_pct"] != tt.expectedBreakoutTargetPct {
				t.Errorf("breakout_target_pct: want %s, got %s", tt.expectedBreakoutTargetPct, s.Parameters["breakout_target_pct"])
			}
			if s.Parameters["breakout_stop_pct"] != tt.expectedBreakoutStopPct {
				t.Errorf("breakout_stop_pct: want %s, got %s", tt.expectedBreakoutStopPct, s.Parameters["breakout_stop_pct"])
			}
		})
	}
}

func TestSqueezeBreakoutEntryResolver_DecisionTypeInMetadata(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("bollinger_squeeze", "triggered", "0.7500", "moderate", "test", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Metadata["decision_type"] != "bollinger_squeeze" {
		t.Errorf("expected decision_type=bollinger_squeeze in metadata, got %s", s.Metadata["decision_type"])
	}
	if s.Metadata["decision_severity"] != "moderate" {
		t.Errorf("expected decision_severity=moderate in metadata, got %s", s.Metadata["decision_severity"])
	}
}

func TestSqueezeBreakoutEntryResolver_DecisionInputPreservesRawConfidence(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("bollinger_squeeze", "triggered", "0.7500", "low", "test", 60, now)
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

func TestSqueezeBreakoutEntryResolver_NotTriggeredHasDecisionContext(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("bollinger_squeeze", "not_triggered", "0.5000", "none", "Bandwidth above threshold", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}
	if s.Metadata["decision_type"] != "bollinger_squeeze" {
		t.Errorf("expected decision_type in metadata for not_triggered, got %v", s.Metadata)
	}
	if s.Metadata["decision_severity"] != "none" {
		t.Errorf("expected decision_severity in metadata for not_triggered, got %v", s.Metadata)
	}
}

func TestSqueezeBreakoutEntryResolver_HighSeverityMaxAggression(t *testing.T) {
	resolver := appstrategy.NewSqueezeBreakoutEntryResolverForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	s, ok := resolver.Resolve("bollinger_squeeze", "triggered", "0.9000", "high", "Strong squeeze detected", 60, now)
	if !ok {
		t.Fatal("expected resolution to succeed")
	}

	// High severity: confidence ×1.00 = 0.9000
	if s.Confidence != "0.9000" {
		t.Errorf("expected confidence 0.9000 (high=1.0×), got %s", s.Confidence)
	}
	// High severity: breakout_target_pct 0.04×1.50 = 0.06
	if s.Parameters["breakout_target_pct"] != "0.06" {
		t.Errorf("expected breakout_target_pct=0.06 (high severity), got %s", s.Parameters["breakout_target_pct"])
	}
	// High severity: breakout_stop_pct 0.015×0.75 = 0.01125 → 0.01
	if s.Parameters["breakout_stop_pct"] != "0.01" {
		t.Errorf("expected breakout_stop_pct=0.01 (high severity), got %s", s.Parameters["breakout_stop_pct"])
	}
}
