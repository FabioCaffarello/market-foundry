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
			{Type: "mean_reversion_entry", Direction: "long", Confidence: "0.72", Timeframe: 60},
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
	expectedSuffix := fmt.Sprintf("%d", ts.Unix())
	if got[len(prefix):] != expectedSuffix {
		t.Fatalf("expected suffix %q, got %q", expectedSuffix, got[len(prefix):])
	}
}
