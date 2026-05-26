package decision_test

import (
	"fmt"
	"testing"
	"time"

	"internal/domain/decision"
	"internal/domain/instrument"
)

func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

func validDecision(t *testing.T) decision.Decision {
	t.Helper()
	return decision.Decision{
		Type:       "rsi_oversold",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Outcome:    decision.OutcomeTriggered,
		Severity:   decision.SeverityLow,
		Confidence: "0.85",
		Rationale:  "RSI 28.50 below oversold threshold 30.0 (distance 5.0%); severity low",
		Signals: []decision.SignalInput{
			{Type: "rsi", Value: "28.50", Timeframe: 60},
		},
		Metadata:  map[string]string{"threshold": "30", "rsi_zone": "low", "distance_pct": "5.0"},
		Final:     true,
		Timestamp: time.Now().UTC(),
	}
}

func TestDecision_Validate_Valid(t *testing.T) {
	d := validDecision(t)
	if prob := d.Validate(); prob != nil {
		t.Fatalf("expected valid decision, got: %s", prob.Message)
	}
}

func TestDecision_Validate_RequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*decision.Decision)
		field  string
	}{
		{"empty type", func(d *decision.Decision) { d.Type = "" }, "type"},
		{"empty source", func(d *decision.Decision) { d.Source = "" }, "source"},
		{"zero instrument", func(d *decision.Decision) { d.Instrument = instrument.CanonicalInstrument{} }, "instrument"},
		{"zero timeframe", func(d *decision.Decision) { d.Timeframe = 0 }, "timeframe"},
		{"negative timeframe", func(d *decision.Decision) { d.Timeframe = -1 }, "timeframe"},
		{"empty outcome", func(d *decision.Decision) { d.Outcome = "" }, "outcome"},
		{"empty confidence", func(d *decision.Decision) { d.Confidence = "" }, "confidence"},
		{"zero timestamp", func(d *decision.Decision) { d.Timestamp = time.Time{} }, "timestamp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := validDecision(t)
			tt.mutate(&d)
			prob := d.Validate()
			if prob == nil {
				t.Fatalf("expected validation error for field %s", tt.field)
			}
		})
	}
}

func TestDecision_Validate_InvalidOutcome(t *testing.T) {
	d := validDecision(t)
	d.Outcome = "invalid_outcome"
	prob := d.Validate()
	if prob == nil {
		t.Fatal("expected validation error for invalid outcome")
	}
}

func TestDecision_Validate_InvalidSeverity(t *testing.T) {
	d := validDecision(t)
	d.Severity = "invalid_severity"
	prob := d.Validate()
	if prob == nil {
		t.Fatal("expected validation error for invalid severity")
	}
}

func TestDecision_Validate_AllOutcomes(t *testing.T) {
	outcomes := []decision.Outcome{
		decision.OutcomeTriggered,
		decision.OutcomeNotTriggered,
		decision.OutcomeInsufficient,
	}

	for _, outcome := range outcomes {
		t.Run(string(outcome), func(t *testing.T) {
			d := validDecision(t)
			d.Outcome = outcome
			if prob := d.Validate(); prob != nil {
				t.Fatalf("expected valid outcome %s, got: %s", outcome, prob.Message)
			}
		})
	}
}

func TestDecision_Validate_AllSeverities(t *testing.T) {
	severities := []decision.Severity{
		decision.SeverityNone,
		decision.SeverityLow,
		decision.SeverityModerate,
		decision.SeverityHigh,
	}

	for _, sev := range severities {
		t.Run(string(sev), func(t *testing.T) {
			d := validDecision(t)
			d.Severity = sev
			if prob := d.Validate(); prob != nil {
				t.Fatalf("expected valid severity %s, got: %s", sev, prob.Message)
			}
		})
	}
}

func TestDecision_Validate_EmptySeverityValid(t *testing.T) {
	d := validDecision(t)
	d.Severity = ""
	if prob := d.Validate(); prob != nil {
		t.Fatalf("empty severity should be valid (optional), got: %s", prob.Message)
	}
}

func TestDecision_PartitionKey(t *testing.T) {
	d := validDecision(t)
	key := d.PartitionKey()
	expected := "binancef.btcusdt.60"
	if key != expected {
		t.Fatalf("expected %q, got %q", expected, key)
	}
}

func TestDecision_DeduplicationKey(t *testing.T) {
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	d := validDecision(t)
	d.Timestamp = ts
	key := d.DeduplicationKey()
	// P4.1.11.a: dedup key precision raised to UnixNano (see decision.go doc).
	want := "dec:rsi_oversold:binancef:btcusdt:60:" + fmt.Sprintf("%d", ts.UnixNano())
	if key != want {
		t.Fatalf("DeduplicationKey() = %q, want %q", key, want)
	}
}

func TestDecision_DeduplicationKey_DifferentTimestamps(t *testing.T) {
	d1 := validDecision(t)
	d1.Timestamp = time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)

	d2 := validDecision(t)
	d2.Timestamp = time.Date(2026, 3, 17, 12, 1, 0, 0, time.UTC)

	if d1.DeduplicationKey() == d2.DeduplicationKey() {
		t.Fatal("different timestamps must produce different deduplication keys")
	}
}

func TestDecision_DeduplicationKey_DifferentTypes(t *testing.T) {
	d1 := validDecision(t)
	d1.Type = "rsi_oversold"

	d2 := validDecision(t)
	d2.Type = "macd_crossover"

	if d1.DeduplicationKey() == d2.DeduplicationKey() {
		t.Fatal("different types must produce different deduplication keys")
	}
}

func TestDecision_NilMetadata(t *testing.T) {
	d := validDecision(t)
	d.Metadata = nil
	if prob := d.Validate(); prob != nil {
		t.Fatalf("nil metadata should be valid, got: %s", prob.Message)
	}
}

func TestDecision_NilSignals(t *testing.T) {
	d := validDecision(t)
	d.Signals = nil
	if prob := d.Validate(); prob != nil {
		t.Fatalf("nil signals should be valid, got: %s", prob.Message)
	}
}

func TestDecision_MultiSymbolIsolation(t *testing.T) {
	d1 := validDecision(t)
	d1.Instrument = btcUSDTPerp(t)

	ethInst, prob := instrument.New("ETH", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup ETH/USDT: %v", prob)
	}
	d2 := validDecision(t)
	d2.Instrument = ethInst

	if d1.PartitionKey() == d2.PartitionKey() {
		t.Fatal("different symbols must have different partition keys")
	}
}

func TestDecision_TimeframeIsolation(t *testing.T) {
	d1 := validDecision(t)
	d1.Timeframe = 60

	d2 := validDecision(t)
	d2.Timeframe = 300

	if d1.PartitionKey() == d2.PartitionKey() {
		t.Fatal("different timeframes must have different partition keys")
	}
}

func TestDecision_DeduplicationKey_MultiSymbolIsolation(t *testing.T) {
	bases := []string{"BTC", "ETH", "SOL"}
	timeframes := []int{60, 300}
	ts := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	keys := make(map[string]bool)

	for _, base := range bases {
		inst, prob := instrument.New(base, "USDT", instrument.ContractPerpetual)
		if prob != nil {
			t.Fatalf("setup %s/USDT: %v", base, prob)
		}
		for _, tf := range timeframes {
			d := validDecision(t)
			d.Instrument = inst
			d.Timeframe = tf
			d.Timestamp = ts
			key := d.DeduplicationKey()
			if keys[key] {
				t.Fatalf("deduplication key collision: base=%s tf=%d key=%s", base, tf, key)
			}
			keys[key] = true
		}
	}

	expectedCount := len(bases) * len(timeframes)
	if len(keys) != expectedCount {
		t.Errorf("expected %d unique dedup keys, got %d", expectedCount, len(keys))
	}
}

func TestDecision_PartitionKey_MultiSymbolMultiTimeframe(t *testing.T) {
	bases := []string{"BTC", "ETH", "SOL"}
	timeframes := []int{60, 300}
	keys := make(map[string]bool)

	for _, base := range bases {
		inst, prob := instrument.New(base, "USDT", instrument.ContractPerpetual)
		if prob != nil {
			t.Fatalf("setup %s/USDT: %v", base, prob)
		}
		for _, tf := range timeframes {
			d := validDecision(t)
			d.Instrument = inst
			d.Timeframe = tf
			key := d.PartitionKey()
			if keys[key] {
				t.Fatalf("partition key collision: base=%s tf=%d key=%s", base, tf, key)
			}
			keys[key] = true
		}
	}

	expectedCount := len(bases) * len(timeframes)
	if len(keys) != expectedCount {
		t.Errorf("expected %d unique partition keys, got %d", expectedCount, len(keys))
	}
}
