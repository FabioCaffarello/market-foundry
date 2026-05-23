package risk_test

import (
	"fmt"
	"testing"
	"time"

	"internal/domain/risk"
)

func validRisk() risk.RiskAssessment {
	return risk.RiskAssessment{
		Type:        "position_exposure",
		Source:      "binancef",
		Symbol:      "btcusdt",
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
	r := validRisk()
	if prob := r.Validate(); prob != nil {
		t.Fatalf("expected valid risk, got: %s", prob.Message)
	}
}

func TestRiskAssessment_Validate_EmptyType(t *testing.T) {
	r := validRisk()
	r.Type = ""
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty type")
	}
}

func TestRiskAssessment_Validate_EmptySource(t *testing.T) {
	r := validRisk()
	r.Source = ""
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty source")
	}
}

func TestRiskAssessment_Validate_EmptySymbol(t *testing.T) {
	r := validRisk()
	r.Symbol = ""
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty symbol")
	}
}

func TestRiskAssessment_Validate_ZeroTimeframe(t *testing.T) {
	r := validRisk()
	r.Timeframe = 0
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for zero timeframe")
	}
}

func TestRiskAssessment_Validate_InvalidDisposition(t *testing.T) {
	r := validRisk()
	r.Disposition = "invalid"
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for invalid disposition")
	}
}

func TestRiskAssessment_Validate_EmptyDisposition(t *testing.T) {
	r := validRisk()
	r.Disposition = ""
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty disposition")
	}
}

func TestRiskAssessment_Validate_EmptyConfidence(t *testing.T) {
	r := validRisk()
	r.Confidence = ""
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty confidence")
	}
}

func TestRiskAssessment_Validate_EmptyRationale(t *testing.T) {
	r := validRisk()
	r.Rationale = ""
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty rationale")
	}
}

func TestRiskAssessment_Validate_ZeroTimestamp(t *testing.T) {
	r := validRisk()
	r.Timestamp = time.Time{}
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for zero timestamp")
	}
}

func TestRiskAssessment_Validate_NoStrategies(t *testing.T) {
	r := validRisk()
	r.Strategies = nil
	if prob := r.Validate(); prob == nil {
		t.Fatal("expected validation error for empty strategies")
	}
}

func TestRiskAssessment_Validate_AllDispositions(t *testing.T) {
	for _, disp := range []risk.Disposition{risk.DispositionApproved, risk.DispositionModified, risk.DispositionRejected} {
		r := validRisk()
		r.Disposition = disp
		if prob := r.Validate(); prob != nil {
			t.Fatalf("disposition %s should be valid, got: %s", disp, prob.Message)
		}
	}
}

func TestRiskAssessment_PartitionKey(t *testing.T) {
	r := risk.RiskAssessment{Source: "binancef", Symbol: "btcusdt", Timeframe: 60}
	expected := "binancef.btcusdt.60"
	if got := r.PartitionKey(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestRiskAssessment_DeduplicationKey(t *testing.T) {
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	r := risk.RiskAssessment{
		Type:      "position_exposure",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Timestamp: ts,
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
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	timeframes := []int{60, 300}
	keys := make(map[string]string) // partition key → symbol

	for _, sym := range symbols {
		for _, tf := range timeframes {
			r := risk.RiskAssessment{Source: "binancef", Symbol: sym, Timeframe: tf}
			key := r.PartitionKey()
			if existing, collision := keys[key]; collision {
				t.Fatalf("partition key collision: %q used by both %q and %q", key, existing, sym)
			}
			keys[key] = sym
		}
	}

	expectedCount := len(symbols) * len(timeframes)
	if len(keys) != expectedCount {
		t.Fatalf("expected %d unique partition keys, got %d", expectedCount, len(keys))
	}
}

func TestRiskAssessment_MultiSymbol_DeduplicationKeyIsolation(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt"}
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	dedupKeys := make(map[string]string)

	for _, sym := range symbols {
		r := risk.RiskAssessment{
			Type:      "position_exposure",
			Source:    "binancef",
			Symbol:    sym,
			Timeframe: 60,
			Timestamp: ts,
		}
		key := r.DeduplicationKey()
		if existing, collision := dedupKeys[key]; collision {
			t.Fatalf("dedup key collision: %q used by both %q and %q", key, existing, sym)
		}
		dedupKeys[key] = sym
	}

	if len(dedupKeys) != len(symbols) {
		t.Fatalf("expected %d unique dedup keys, got %d", len(symbols), len(dedupKeys))
	}
}

func TestRiskAssessment_StrategyInput_DecisionContextPreserved(t *testing.T) {
	r := validRisk()
	si := r.Strategies[0]
	if si.DecisionSeverity != "low" {
		t.Errorf("expected decision severity low, got %s", si.DecisionSeverity)
	}
	if si.DecisionRationale != "RSI 28.50 below threshold" {
		t.Errorf("expected decision rationale, got %s", si.DecisionRationale)
	}
}

func TestRiskAssessment_StrategyInput_EmptyDecisionContext(t *testing.T) {
	r := validRisk()
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
	r1 := validRisk()
	r1.Symbol = "btcusdt"

	r2 := validRisk()
	r2.Symbol = "ethusdt"

	if r1.Symbol == r2.Symbol {
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
