package risk_test

import (
	"fmt"
	"testing"
	"time"

	"internal/domain/instrument"
	"internal/domain/risk"
)

func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

func mustInstrument(t *testing.T, base, quote string) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New(base, quote, instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

func validRisk(t *testing.T) risk.RiskAssessment {
	t.Helper()
	return risk.RiskAssessment{
		Type:        "position_exposure",
		Source:      "binancef",
		Instrument:  btcUSDTPerp(t),
		Timeframe:   60,
		Disposition: risk.DispositionApproved,
		Confidence:  "0.85",
		Strategies: []risk.StrategyInput{
			{Type: "mean_reversion_entry", Direction: "long", Confidence: "0.72", Timeframe: 60, DecisionSeverity: "low", DecisionRationale: "RSI 28.50 below threshold"},
		},
		Constraints: risk.Constraints{MaxPositionSize: "0.01", MaxExposure: "0.05"},
		Rationale:   "Position size within exposure limits",
		Parameters:  map[string]string{"max_position_pct": "0.02", "max_portfolio_exposure_pct": "0.10"},
		Final:       true,
		Timestamp:   time.Now().UTC(),
	}
}

func TestRiskAssessment_Validate_Valid(t *testing.T) {
	r := validRisk(t)
	if prob := r.Validate(); prob != nil {
		t.Fatalf("expected valid risk, got: %s", prob.Message)
	}
}

func TestRiskAssessment_Validate_EmptyType(t *testing.T) {
	r := validRisk(t)
	r.Type = ""
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty type")
	}
}

func TestRiskAssessment_Validate_EmptySource(t *testing.T) {
	r := validRisk(t)
	r.Source = ""
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty source")
	}
}

func TestRiskAssessment_Validate_EmptySymbol(t *testing.T) {
	r := validRisk(t)
	r.Instrument = instrument.CanonicalInstrument{}
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty symbol")
	}
}

func TestRiskAssessment_Validate_ZeroTimeframe(t *testing.T) {
	r := validRisk(t)
	r.Timeframe = 0
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for zero timeframe")
	}
}

func TestRiskAssessment_Validate_InvalidDisposition(t *testing.T) {
	r := validRisk(t)
	r.Disposition = "invalid"
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for invalid disposition")
	}
}

func TestRiskAssessment_Validate_EmptyDisposition(t *testing.T) {
	r := validRisk(t)
	r.Disposition = ""
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty disposition")
	}
}

func TestRiskAssessment_Validate_EmptyConfidence(t *testing.T) {
	r := validRisk(t)
	r.Confidence = ""
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty confidence")
	}
}

func TestRiskAssessment_Validate_EmptyRationale(t *testing.T) {
	r := validRisk(t)
	r.Rationale = ""
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty rationale")
	}
}

func TestRiskAssessment_Validate_ZeroTimestamp(t *testing.T) {
	r := validRisk(t)
	r.Timestamp = time.Time{}
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for zero timestamp")
	}
}

func TestRiskAssessment_Validate_NoStrategies(t *testing.T) {
	r := validRisk(t)
	r.Strategies = nil
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty strategies")
	}
}

func TestRiskAssessment_Validate_AllDispositions(t *testing.T) {
	for _, disp := range []risk.Disposition{risk.DispositionApproved, risk.DispositionModified, risk.DispositionRejected} {
		r := validRisk(t)
		r.Disposition = disp
		if prob := r.Validate(); prob != nil {
			t.Fatalf("disposition %s should be valid, got: %s", disp, prob.Message)
		}
	}
}

func TestRiskAssessment_PartitionKey(t *testing.T) {
	r := risk.RiskAssessment{Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60}
	expected := "binancef.btcusdt.60"
	if got := r.PartitionKey(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestRiskAssessment_DeduplicationKey(t *testing.T) {
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	r := risk.RiskAssessment{
		Type:       "position_exposure",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Timestamp:  ts,
	}
	got := r.DeduplicationKey()
	prefix := "risk:position_exposure:binancef:btcusdt:60:"
	if got[:len(prefix)] != prefix {
		t.Fatalf("expected prefix %q, got %q", prefix, got)
	}
	// P4.1.11.a: dedup key precision raised to UnixNano (see risk.go doc).
	expectedSuffix := fmt.Sprintf("%d", ts.UnixNano())
	if got[len(prefix):] != expectedSuffix {
		t.Fatalf("expected suffix %q, got %q", expectedSuffix, got[len(prefix):])
	}
}

func TestRiskAssessment_MultiSymbol_PartitionKeyIsolation(t *testing.T) {
	insts := []instrument.CanonicalInstrument{
		mustInstrument(t, "BTC", "USDT"),
		mustInstrument(t, "ETH", "USDT"),
		mustInstrument(t, "SOL", "USDT"),
	}
	timeframes := []int{60, 300}
	keys := make(map[string]string) // partition key → venue symbol

	for _, inst := range insts {
		for _, tf := range timeframes {
			r := risk.RiskAssessment{Source: "binancef", Instrument: inst, Timeframe: tf}
			key := r.PartitionKey()
			if existing, collision := keys[key]; collision {
				t.Fatalf("partition key collision: %q used by both %q and %q", key, existing, r.VenueSymbol())
			}
			keys[key] = r.VenueSymbol()
		}
	}

	expectedCount := len(insts) * len(timeframes)
	if len(keys) != expectedCount {
		t.Fatalf("expected %d unique partition keys, got %d", expectedCount, len(keys))
	}
}

func TestRiskAssessment_MultiSymbol_DeduplicationKeyIsolation(t *testing.T) {
	insts := []instrument.CanonicalInstrument{
		mustInstrument(t, "BTC", "USDT"),
		mustInstrument(t, "ETH", "USDT"),
	}
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	dedupKeys := make(map[string]string)

	for _, inst := range insts {
		r := risk.RiskAssessment{
			Type:       "position_exposure",
			Source:     "binancef",
			Instrument: inst,
			Timeframe:  60,
			Timestamp:  ts,
		}
		key := r.DeduplicationKey()
		if existing, collision := dedupKeys[key]; collision {
			t.Fatalf("dedup key collision: %q used by both %q and %q", key, existing, r.VenueSymbol())
		}
		dedupKeys[key] = r.VenueSymbol()
	}

	if len(dedupKeys) != len(insts) {
		t.Fatalf("expected %d unique dedup keys, got %d", len(insts), len(dedupKeys))
	}
}

func TestRiskAssessment_StrategyInput_DecisionContextPreserved(t *testing.T) {
	r := validRisk(t)
	si := r.Strategies[0]
	if si.DecisionSeverity != "low" {
		t.Errorf("expected decision severity low, got %s", si.DecisionSeverity)
	}
	if si.DecisionRationale != "RSI 28.50 below threshold" {
		t.Errorf("expected decision rationale, got %s", si.DecisionRationale)
	}
}

func TestRiskAssessment_StrategyInput_EmptyDecisionContext(t *testing.T) {
	r := validRisk(t)
	r.Strategies = []risk.StrategyInput{
		{Type: "mean_reversion_entry", Direction: "flat", Confidence: "0.0000", Timeframe: 60},
	}
	if prob := r.Validate(); prob != nil {
		t.Fatalf("risk with empty decision context should be valid, got: %s", prob.Message)
	}
	if r.Strategies[0].DecisionSeverity != "" {
		t.Errorf("expected empty decision severity, got %s", r.Strategies[0].DecisionSeverity)
	}
}

func TestRiskAssessment_MultiSymbol_NoOwnershipBleed(t *testing.T) {
	// Verify that two assessments for different symbols maintain independent field values
	// and no cross-symbol contamination occurs through shared references.
	r1 := validRisk(t)
	r1.Instrument = mustInstrument(t, "BTC", "USDT")

	r2 := validRisk(t)
	r2.Instrument = mustInstrument(t, "ETH", "USDT")

	if r1.VenueSymbol() == r2.VenueSymbol() {
		t.Fatal("symbols should differ")
	}
	if r1.PartitionKey() == r2.PartitionKey() {
		t.Fatalf("partition keys should differ: %q vs %q", r1.PartitionKey(), r2.PartitionKey())
	}
	if r1.Source != r2.Source {
		t.Fatal("source should be shared across symbols")
	}
	if r1.Type != r2.Type {
		t.Fatal("type should be shared across symbols")
	}
	// Validate both independently pass validation
	if prob := r1.Validate(); prob != nil {
		t.Fatalf("r1 should be valid: %s", prob.Message)
	}
	if prob := r2.Validate(); prob != nil {
		t.Fatalf("r2 should be valid: %s", prob.Message)
	}
}
