package decision_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	appdecision "internal/application/decision"
	domaindecision "internal/domain/decision"
)

func TestRSIOversoldEvaluator_Triggered(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "25.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Outcome != domaindecision.OutcomeTriggered {
		t.Fatalf("expected triggered, got %s", d.Outcome)
	}
	if d.Type != "rsi_oversold" {
		t.Fatalf("expected type rsi_oversold, got %s", d.Type)
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
	if d.Signals[0].Type != "rsi" || d.Signals[0].Value != "25.00" {
		t.Fatalf("unexpected signal input: %+v", d.Signals[0])
	}
}

func TestRSIOversoldEvaluator_NotTriggered(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "65.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Outcome != domaindecision.OutcomeNotTriggered {
		t.Fatalf("expected not_triggered, got %s", d.Outcome)
	}
}

func TestRSIOversoldEvaluator_AtThreshold(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "30.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	// At exactly 30.0: not triggered (strictly less than threshold).
	if d.Outcome != domaindecision.OutcomeNotTriggered {
		t.Fatalf("expected not_triggered at threshold, got %s", d.Outcome)
	}
}

func TestRSIOversoldEvaluator_ExtremeOversold(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "5.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Outcome != domaindecision.OutcomeTriggered {
		t.Fatalf("expected triggered, got %s", d.Outcome)
	}
	if d.Confidence == "" {
		t.Fatal("expected non-empty confidence")
	}
}

func TestRSIOversoldEvaluator_InvalidValue(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	_, ok := eval.Evaluate("rsi", "not-a-number", 60, now)
	if ok {
		t.Fatal("expected evaluation to fail for invalid value")
	}
}

func TestRSIOversoldEvaluator_Validation(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "25.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if prob := d.Validate(); prob != nil {
		t.Fatalf("decision should be valid, got: %s", prob.Message)
	}
}

func TestRSIOversoldEvaluator_MetadataContainsThreshold(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "25.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Metadata["threshold"] == "" {
		t.Fatal("expected threshold in metadata")
	}
}

func TestRSIOversoldEvaluator_EmptyValue(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	_, ok := eval.Evaluate("rsi", "", 60, time.Now().UTC())
	if ok {
		t.Fatal("expected failure for empty signal value")
	}
}

func TestRSIOversoldEvaluator_ConfidenceBounds(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	tests := []struct {
		name     string
		rsiValue string
	}{
		{"deeply oversold", "5.00"},
		{"just below threshold", "29.99"},
		{"at threshold", "30.00"},
		{"slightly above threshold", "30.01"},
		{"high RSI", "85.00"},
		{"max RSI", "100.00"},
		{"zero RSI", "0.00"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := eval.Evaluate("rsi", tc.rsiValue, 60, now)
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

func TestRSIOversoldEvaluator_ConfidenceMonotonicity(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	// Below threshold: lower RSI → higher confidence.
	dNear, _ := eval.Evaluate("rsi", "29.00", 60, now)
	dFar, _ := eval.Evaluate("rsi", "10.00", 60, now)

	confNear, _ := strconv.ParseFloat(dNear.Confidence, 64)
	confFar, _ := strconv.ParseFloat(dFar.Confidence, 64)

	if confFar <= confNear {
		t.Fatalf("confidence should increase further from threshold: near=%f far=%f", confNear, confFar)
	}

	// Above threshold: higher RSI → higher confidence.
	dNearAbove, _ := eval.Evaluate("rsi", "31.00", 60, now)
	dFarAbove, _ := eval.Evaluate("rsi", "80.00", 60, now)

	confNearAbove, _ := strconv.ParseFloat(dNearAbove.Confidence, 64)
	confFarAbove, _ := strconv.ParseFloat(dFarAbove.Confidence, 64)

	if confFarAbove <= confNearAbove {
		t.Fatalf("confidence should increase further from threshold: near=%f far=%f", confNearAbove, confFarAbove)
	}
}

func TestRSIOversoldEvaluator_NegativeRSI_CapsConfidence(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "-5.00", 60, now)
	if !ok {
		t.Fatal("expected successful evaluation even for negative RSI")
	}
	if d.Outcome != domaindecision.OutcomeTriggered {
		t.Fatalf("expected triggered for negative RSI, got %s", d.Outcome)
	}
	conf, _ := strconv.ParseFloat(d.Confidence, 64)
	if conf > 1.0 {
		t.Fatalf("confidence must be capped at 1.0, got %f", conf)
	}
}

func TestRSIOversoldEvaluator_TimestampPreserved(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	d, ok := eval.Evaluate("rsi", "25.00", 60, ts)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if !d.Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, d.Timestamp)
	}
}

func TestRSIOversoldEvaluator_SignalInputPreserved(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 300)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "25.00", 300, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if len(d.Signals) != 1 {
		t.Fatalf("expected 1 signal input, got %d", len(d.Signals))
	}
	si := d.Signals[0]
	if si.Type != "rsi" {
		t.Errorf("expected signal type rsi, got %s", si.Type)
	}
	if si.Value != "25.00" {
		t.Errorf("expected signal value 25.00, got %s", si.Value)
	}
	if si.Timeframe != 300 {
		t.Errorf("expected signal timeframe 300, got %d", si.Timeframe)
	}
}

// -- Severity tests -----------------------------------------------------------

func TestRSIOversoldEvaluator_SeverityLow(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "25.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Severity != domaindecision.SeverityLow {
		t.Fatalf("expected severity low for RSI 25, got %s", d.Severity)
	}
}

func TestRSIOversoldEvaluator_SeverityModerate(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "15.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Severity != domaindecision.SeverityModerate {
		t.Fatalf("expected severity moderate for RSI 15, got %s", d.Severity)
	}
}

func TestRSIOversoldEvaluator_SeverityHigh(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "5.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Severity != domaindecision.SeverityHigh {
		t.Fatalf("expected severity high for RSI 5, got %s", d.Severity)
	}
}

func TestRSIOversoldEvaluator_SeverityNone_NotTriggered(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "65.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Severity != domaindecision.SeverityNone {
		t.Fatalf("expected severity none for not_triggered, got %s", d.Severity)
	}
}

func TestRSIOversoldEvaluator_SeverityMonotonicity(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	// Severity should be monotonically non-decreasing as RSI decreases below threshold.
	severityOrder := map[domaindecision.Severity]int{
		domaindecision.SeverityNone:     0,
		domaindecision.SeverityLow:      1,
		domaindecision.SeverityModerate: 2,
		domaindecision.SeverityHigh:     3,
	}

	rsiValues := []string{"29.00", "25.00", "15.00", "5.00"}
	prevSeverityRank := 0
	for _, rsiStr := range rsiValues {
		d, ok := eval.Evaluate("rsi", rsiStr, 60, now)
		if !ok {
			t.Fatalf("evaluation failed for RSI %s", rsiStr)
		}
		rank := severityOrder[d.Severity]
		if rank < prevSeverityRank {
			t.Fatalf("severity should not decrease as RSI drops: RSI %s got %s", rsiStr, d.Severity)
		}
		prevSeverityRank = rank
	}
}

// -- Rationale tests ----------------------------------------------------------

func TestRSIOversoldEvaluator_RationaleTriggered(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "25.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Rationale == "" {
		t.Fatal("expected non-empty rationale")
	}
	if !strings.Contains(d.Rationale, "below oversold threshold") {
		t.Fatalf("rationale should mention 'below oversold threshold', got: %s", d.Rationale)
	}
	if !strings.Contains(d.Rationale, "severity") {
		t.Fatalf("rationale should mention severity, got: %s", d.Rationale)
	}
}

func TestRSIOversoldEvaluator_RationaleNotTriggered(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "65.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if !strings.Contains(d.Rationale, "above oversold threshold") {
		t.Fatalf("rationale should mention 'above oversold threshold', got: %s", d.Rationale)
	}
	if !strings.Contains(d.Rationale, "not oversold") {
		t.Fatalf("rationale should mention 'not oversold', got: %s", d.Rationale)
	}
}

// -- Metadata enrichment tests ------------------------------------------------

func TestRSIOversoldEvaluator_MetadataRSIZone(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "25.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Metadata["rsi_zone"] == "" {
		t.Fatal("expected rsi_zone in metadata")
	}
	if d.Metadata["rsi_zone"] != "low" {
		t.Fatalf("expected rsi_zone=low for RSI 25, got %s", d.Metadata["rsi_zone"])
	}
}

func TestRSIOversoldEvaluator_MetadataDistancePct(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "25.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Metadata["distance_pct"] == "" {
		t.Fatal("expected distance_pct in metadata")
	}
	pct, err := strconv.ParseFloat(d.Metadata["distance_pct"], 64)
	if err != nil {
		t.Fatalf("distance_pct not parseable: %v", err)
	}
	// RSI 25 with threshold 30: (30-25)/30 * 100 = 16.7%
	if pct < 16.0 || pct > 17.0 {
		t.Fatalf("expected distance_pct ~16.7%% for RSI 25, got %.1f", pct)
	}
}

func TestRSIOversoldEvaluator_MetadataDistancePct_NotTriggered(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "65.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	pct, _ := strconv.ParseFloat(d.Metadata["distance_pct"], 64)
	if pct != 0 {
		t.Fatalf("expected distance_pct=0 for not triggered, got %.1f", pct)
	}
}

func TestRSIOversoldEvaluator_MetadataRSIZone_NotTriggered(t *testing.T) {
	eval := appdecision.NewRSIOversoldEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	d, ok := eval.Evaluate("rsi", "65.00", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Metadata["rsi_zone"] != "none" {
		t.Fatalf("expected rsi_zone=none for not triggered, got %s", d.Metadata["rsi_zone"])
	}
}
