package decision_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	appdecision "internal/application/decision"
	domaindecision "internal/domain/decision"
)

// -- Triggered tests ----------------------------------------------------------

func TestBollingerSqueezeEvaluator_Triggered(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{
		"bandwidth": "200.0000",
		"sma":       "50000.0000",
	}
	// relativeBW = 200/50000 = 0.004 < 0.10 → triggered
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Outcome != domaindecision.OutcomeTriggered {
		t.Fatalf("expected triggered, got %s", d.Outcome)
	}
	if d.Type != "bollinger_squeeze" {
		t.Fatalf("expected type bollinger_squeeze, got %s", d.Type)
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
	if d.Signals[0].Type != "bollinger" || d.Signals[0].Value != "0.5000" {
		t.Fatalf("unexpected signal input: %+v", d.Signals[0])
	}
}

func TestBollingerSqueezeEvaluator_NotTriggered(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{
		"bandwidth": "10000.0000",
		"sma":       "50000.0000",
	}
	// relativeBW = 10000/50000 = 0.20 > 0.10 → not triggered
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Outcome != domaindecision.OutcomeNotTriggered {
		t.Fatalf("expected not_triggered, got %s", d.Outcome)
	}
}

func TestBollingerSqueezeEvaluator_AtThreshold(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{
		"bandwidth": "5000.0000",
		"sma":       "50000.0000",
	}
	// relativeBW = 5000/50000 = 0.10 = threshold → not triggered (strictly less than)
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Outcome != domaindecision.OutcomeNotTriggered {
		t.Fatalf("expected not_triggered at threshold, got %s", d.Outcome)
	}
}

func TestBollingerSqueezeEvaluator_ExtremeSqueeze(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{
		"bandwidth": "50.0000",
		"sma":       "50000.0000",
	}
	// relativeBW = 50/50000 = 0.001 → extreme squeeze
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Outcome != domaindecision.OutcomeTriggered {
		t.Fatalf("expected triggered, got %s", d.Outcome)
	}
	if d.Severity != domaindecision.SeverityHigh {
		t.Fatalf("expected severity high for extreme squeeze, got %s", d.Severity)
	}
}

// -- Invalid input tests -------------------------------------------------------

func TestBollingerSqueezeEvaluator_InvalidValue(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{"bandwidth": "200.0000", "sma": "50000.0000"}
	_, ok := eval.Evaluate("bollinger", "not-a-number", 60, now, meta)
	if ok {
		t.Fatal("expected evaluation to fail for invalid value")
	}
}

func TestBollingerSqueezeEvaluator_MissingBandwidth(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{"sma": "50000.0000"}
	_, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if ok {
		t.Fatal("expected evaluation to fail when bandwidth is missing")
	}
}

func TestBollingerSqueezeEvaluator_MissingSMA(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{"bandwidth": "200.0000"}
	_, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if ok {
		t.Fatal("expected evaluation to fail when SMA is missing")
	}
}

func TestBollingerSqueezeEvaluator_NilMetadata(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	_, ok := eval.Evaluate("bollinger", "0.5000", 60, now, nil)
	if ok {
		t.Fatal("expected evaluation to fail for nil metadata")
	}
}

func TestBollingerSqueezeEvaluator_ZeroSMA(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{"bandwidth": "200.0000", "sma": "0.0000"}
	_, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if ok {
		t.Fatal("expected evaluation to fail for zero SMA")
	}
}

func TestBollingerSqueezeEvaluator_EmptyValue(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	_, ok := eval.Evaluate("bollinger", "", 60, time.Now().UTC(), map[string]string{
		"bandwidth": "200.0000", "sma": "50000.0000",
	})
	if ok {
		t.Fatal("expected failure for empty signal value")
	}
}

// -- Validation tests ----------------------------------------------------------

func TestBollingerSqueezeEvaluator_Validation(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{"bandwidth": "200.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if prob := d.Validate(); prob != nil {
		t.Fatalf("decision should be valid, got: %s", prob.Message)
	}
}

// -- Confidence tests ----------------------------------------------------------

func TestBollingerSqueezeEvaluator_ConfidenceBounds(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	tests := []struct {
		name      string
		bandwidth string
		sma       string
	}{
		{"extreme squeeze", "50.0000", "50000.0000"},
		{"mild squeeze", "4500.0000", "50000.0000"},
		{"at threshold", "5000.0000", "50000.0000"},
		{"wide bands", "10000.0000", "50000.0000"},
		{"very wide bands", "25000.0000", "50000.0000"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta := map[string]string{"bandwidth": tc.bandwidth, "sma": tc.sma}
			d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
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

func TestBollingerSqueezeEvaluator_ConfidenceMonotonicity(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()
	sma := "50000.0000"

	// Below threshold: tighter squeeze → higher confidence.
	dMild, _ := eval.Evaluate("bollinger", "0.5000", 60, now, map[string]string{
		"bandwidth": "4500.0000", "sma": sma,
	})
	dTight, _ := eval.Evaluate("bollinger", "0.5000", 60, now, map[string]string{
		"bandwidth": "500.0000", "sma": sma,
	})

	confMild, _ := strconv.ParseFloat(dMild.Confidence, 64)
	confTight, _ := strconv.ParseFloat(dTight.Confidence, 64)

	if confTight <= confMild {
		t.Fatalf("confidence should increase for tighter squeeze: mild=%f tight=%f", confMild, confTight)
	}

	// Above threshold: wider bands → higher confidence (more clearly not a squeeze).
	dNarrow, _ := eval.Evaluate("bollinger", "0.5000", 60, now, map[string]string{
		"bandwidth": "6000.0000", "sma": sma,
	})
	dWide, _ := eval.Evaluate("bollinger", "0.5000", 60, now, map[string]string{
		"bandwidth": "25000.0000", "sma": sma,
	})

	confNarrow, _ := strconv.ParseFloat(dNarrow.Confidence, 64)
	confWide, _ := strconv.ParseFloat(dWide.Confidence, 64)

	if confWide <= confNarrow {
		t.Fatalf("confidence should increase for wider bands: narrow=%f wide=%f", confNarrow, confWide)
	}
}

// -- Severity tests ------------------------------------------------------------

func TestBollingerSqueezeEvaluator_SeverityLow(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	// relativeBW = 4000/50000 = 0.08 → ratio = 0.08/0.10 = 0.80 > 0.50 → Low
	meta := map[string]string{"bandwidth": "4000.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Severity != domaindecision.SeverityLow {
		t.Fatalf("expected severity low, got %s", d.Severity)
	}
}

func TestBollingerSqueezeEvaluator_SeverityModerate(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	// relativeBW = 1500/50000 = 0.03 → ratio = 0.03/0.10 = 0.30 → 0.25 < 0.30 ≤ 0.50 → Moderate
	meta := map[string]string{"bandwidth": "1500.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Severity != domaindecision.SeverityModerate {
		t.Fatalf("expected severity moderate, got %s", d.Severity)
	}
}

func TestBollingerSqueezeEvaluator_SeverityHigh(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	// relativeBW = 100/50000 = 0.002 → ratio = 0.002/0.10 = 0.02 ≤ 0.25 → High
	meta := map[string]string{"bandwidth": "100.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Severity != domaindecision.SeverityHigh {
		t.Fatalf("expected severity high for extreme squeeze, got %s", d.Severity)
	}
}

func TestBollingerSqueezeEvaluator_SeverityNone_NotTriggered(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{"bandwidth": "10000.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Severity != domaindecision.SeverityNone {
		t.Fatalf("expected severity none for not_triggered, got %s", d.Severity)
	}
}

func TestBollingerSqueezeEvaluator_SeverityMonotonicity(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	severityOrder := map[domaindecision.Severity]int{
		domaindecision.SeverityNone:     0,
		domaindecision.SeverityLow:      1,
		domaindecision.SeverityModerate: 2,
		domaindecision.SeverityHigh:     3,
	}

	// Decreasing bandwidth → increasing severity.
	bandwidths := []string{"4000.0000", "2000.0000", "500.0000", "100.0000"}
	prevSeverityRank := 0
	for _, bw := range bandwidths {
		meta := map[string]string{"bandwidth": bw, "sma": "50000.0000"}
		d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
		if !ok {
			t.Fatalf("evaluation failed for bandwidth %s", bw)
		}
		rank := severityOrder[d.Severity]
		if rank < prevSeverityRank {
			t.Fatalf("severity should not decrease as bandwidth drops: bandwidth %s got %s", bw, d.Severity)
		}
		prevSeverityRank = rank
	}
}

// -- Rationale tests -----------------------------------------------------------

func TestBollingerSqueezeEvaluator_RationaleTriggered(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{"bandwidth": "200.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Rationale == "" {
		t.Fatal("expected non-empty rationale")
	}
	if !strings.Contains(d.Rationale, "squeeze detected") {
		t.Fatalf("rationale should mention 'squeeze detected', got: %s", d.Rationale)
	}
	if !strings.Contains(d.Rationale, "severity") {
		t.Fatalf("rationale should mention severity, got: %s", d.Rationale)
	}
}

func TestBollingerSqueezeEvaluator_RationaleNotTriggered(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{"bandwidth": "10000.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if !strings.Contains(d.Rationale, "No Bollinger squeeze") {
		t.Fatalf("rationale should mention 'No Bollinger squeeze', got: %s", d.Rationale)
	}
}

// -- Metadata enrichment tests -------------------------------------------------

func TestBollingerSqueezeEvaluator_MetadataContainsFields(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{"bandwidth": "200.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}

	requiredKeys := []string{"squeeze_threshold", "relative_bandwidth", "bandwidth", "sma", "pct_b", "pct_b_zone"}
	for _, key := range requiredKeys {
		if d.Metadata[key] == "" {
			t.Fatalf("expected %s in metadata", key)
		}
	}
}

func TestBollingerSqueezeEvaluator_MetadataPctBZone_Lower(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{"bandwidth": "200.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.1000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Metadata["pct_b_zone"] != "lower" {
		t.Fatalf("expected pct_b_zone=lower for %%B=0.10, got %s", d.Metadata["pct_b_zone"])
	}
}

func TestBollingerSqueezeEvaluator_MetadataPctBZone_Middle(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{"bandwidth": "200.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Metadata["pct_b_zone"] != "middle" {
		t.Fatalf("expected pct_b_zone=middle for %%B=0.50, got %s", d.Metadata["pct_b_zone"])
	}
}

func TestBollingerSqueezeEvaluator_MetadataPctBZone_Upper(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	meta := map[string]string{"bandwidth": "200.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.9000", 60, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if d.Metadata["pct_b_zone"] != "upper" {
		t.Fatalf("expected pct_b_zone=upper for %%B=0.90, got %s", d.Metadata["pct_b_zone"])
	}
}

// -- Timestamp & signal input preservation ------------------------------------

func TestBollingerSqueezeEvaluator_TimestampPreserved(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	ts := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)

	meta := map[string]string{"bandwidth": "200.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.5000", 60, ts, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if !d.Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, d.Timestamp)
	}
}

func TestBollingerSqueezeEvaluator_SignalInputPreserved(t *testing.T) {
	eval := appdecision.NewBollingerSqueezeEvaluatorForInstrument("binancef", btcUSDTPerp, 300)
	now := time.Now().UTC()

	meta := map[string]string{"bandwidth": "200.0000", "sma": "50000.0000"}
	d, ok := eval.Evaluate("bollinger", "0.5000", 300, now, meta)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if len(d.Signals) != 1 {
		t.Fatalf("expected 1 signal input, got %d", len(d.Signals))
	}
	si := d.Signals[0]
	if si.Type != "bollinger" {
		t.Errorf("expected signal type bollinger, got %s", si.Type)
	}
	if si.Value != "0.5000" {
		t.Errorf("expected signal value 0.5000, got %s", si.Value)
	}
	if si.Timeframe != 300 {
		t.Errorf("expected signal timeframe 300, got %d", si.Timeframe)
	}
}
