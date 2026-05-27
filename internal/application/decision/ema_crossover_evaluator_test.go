package decision_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	appdecision "internal/application/decision"
	domaindecision "internal/domain/decision"
)

func TestEMACrossoverEvaluator_Bullish_Triggered(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("ema_crossover", "bullish", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Outcome != domaindecision.OutcomeTriggered {
		t.Fatalf("expected triggered, got %s", d.Outcome)
	}
	if d.Type != "ema_crossover" {
		t.Fatalf("expected type ema_crossover, got %s", d.Type)
	}
	if d.Source != "binancef" || d.VenueSymbol() != "btcusdt" || d.Timeframe != 60 {
		t.Fatalf("unexpected partition: %s/%s/%d", d.Source, d.VenueSymbol(), d.Timeframe)
	}
	if !d.Final {
		t.Fatal("expected final=true")
	}
	if len(d.Signals) != 1 {
		t.Fatalf("expected 1 signal input, got %d", len(d.Signals))
	}
	if d.Signals[0].Type != "ema_crossover" || d.Signals[0].Value != "bullish" {
		t.Fatalf("unexpected signal input: %+v", d.Signals[0])
	}
}

func TestEMACrossoverEvaluator_Bearish_NotTriggered(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("ema_crossover", "bearish", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Outcome != domaindecision.OutcomeNotTriggered {
		t.Fatalf("expected not_triggered, got %s", d.Outcome)
	}
}

func TestEMACrossoverEvaluator_Neutral_NotTriggered(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("ema_crossover", "neutral", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Outcome != domaindecision.OutcomeNotTriggered {
		t.Fatalf("expected not_triggered, got %s", d.Outcome)
	}
}

func TestEMACrossoverEvaluator_InvalidValue(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	_, ok := eval.Evaluate("ema_crossover", "sideways", 60, now)
	if ok {
		t.Fatal("expected evaluation to fail for invalid value")
	}
}

func TestEMACrossoverEvaluator_EmptyValue(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	_, ok := eval.Evaluate("ema_crossover", "", 60, time.Now().UTC())
	if ok {
		t.Fatal("expected failure for empty signal value")
	}
}

func TestEMACrossoverEvaluator_Validation(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("ema_crossover", "bullish", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if prob := d.Validate(); prob != nil {
		t.Fatalf("decision should be valid, got: %s", prob.Message)
	}
}

func TestEMACrossoverEvaluator_ConfidenceBounds(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	tests := []struct {
		name      string
		signalVal string
	}{
		{"bullish", "bullish"},
		{"bearish", "bearish"},
		{"neutral", "neutral"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := eval.Evaluate("ema_crossover", tc.signalVal, 60, now)
			if !ok {
				t.Fatal("expected successful evaluation")
			}
			conf, err := strconv.ParseFloat(d.Confidence, 64)
			if err != nil {
				t.Fatalf("failed to parse confidence: %v", err)
			}
			if conf < 0.5 || conf > 1.0 {
				t.Fatalf("confidence %f out of expected range [0.5, 1.0]", conf)
			}
		})
	}
}

func TestEMACrossoverEvaluator_Severity_Bullish(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("ema_crossover", "bullish", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Severity != domaindecision.SeverityModerate {
		t.Fatalf("expected severity moderate for bullish, got %s", d.Severity)
	}
}

func TestEMACrossoverEvaluator_Severity_Bearish(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("ema_crossover", "bearish", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Severity != domaindecision.SeverityNone {
		t.Fatalf("expected severity none for bearish, got %s", d.Severity)
	}
}

func TestEMACrossoverEvaluator_Severity_Neutral(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("ema_crossover", "neutral", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Severity != domaindecision.SeverityNone {
		t.Fatalf("expected severity none for neutral, got %s", d.Severity)
	}
}

func TestEMACrossoverEvaluator_Rationale_Bullish(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("ema_crossover", "bullish", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Rationale == "" {
		t.Fatal("expected non-empty rationale")
	}
	if !strings.Contains(d.Rationale, "bullish") {
		t.Fatalf("rationale should mention 'bullish', got: %s", d.Rationale)
	}
	if !strings.Contains(d.Rationale, "fast EMA above slow EMA") {
		t.Fatalf("rationale should explain crossover, got: %s", d.Rationale)
	}
}

func TestEMACrossoverEvaluator_Rationale_Bearish(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("ema_crossover", "bearish", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if !strings.Contains(d.Rationale, "bearish") {
		t.Fatalf("rationale should mention 'bearish', got: %s", d.Rationale)
	}
	if !strings.Contains(d.Rationale, "no bullish crossover") {
		t.Fatalf("rationale should explain lack of crossover, got: %s", d.Rationale)
	}
}

func TestEMACrossoverEvaluator_Metadata_CrossoverDirection(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	for _, dir := range []string{"bullish", "bearish", "neutral"} {
		t.Run(dir, func(t *testing.T) {
			d, ok := eval.Evaluate("ema_crossover", dir, 60, now)
			if !ok {
				t.Fatal("expected evaluation to succeed")
			}
			if d.Metadata["crossover_direction"] != dir {
				t.Fatalf("expected crossover_direction=%s, got %s", dir, d.Metadata["crossover_direction"])
			}
		})
	}
}

func TestEMACrossoverEvaluator_TimestampPreserved(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	ts := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)

	d, ok := eval.Evaluate("ema_crossover", "bullish", 60, ts)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if !d.Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, d.Timestamp)
	}
}

func TestEMACrossoverEvaluator_SignalInputPreserved(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 300)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("ema_crossover", "bullish", 300, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if len(d.Signals) != 1 {
		t.Fatalf("expected 1 signal input, got %d", len(d.Signals))
	}
	si := d.Signals[0]
	if si.Type != "ema_crossover" {
		t.Errorf("expected signal type ema_crossover, got %s", si.Type)
	}
	if si.Value != "bullish" {
		t.Errorf("expected signal value bullish, got %s", si.Value)
	}
	if si.Timeframe != 300 {
		t.Errorf("expected signal timeframe 300, got %d", si.Timeframe)
	}
}

func TestEMACrossoverEvaluator_BullishConfidence_HigherThanNeutral(t *testing.T) {
	eval := appdecision.NewEMACrossoverEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	dBullish, _ := eval.Evaluate("ema_crossover", "bullish", 60, now)
	dNeutral, _ := eval.Evaluate("ema_crossover", "neutral", 60, now)

	confBullish, _ := strconv.ParseFloat(dBullish.Confidence, 64)
	confNeutral, _ := strconv.ParseFloat(dNeutral.Confidence, 64)

	if confBullish <= confNeutral {
		t.Fatalf("bullish confidence (%f) should be higher than neutral (%f)", confBullish, confNeutral)
	}
}
