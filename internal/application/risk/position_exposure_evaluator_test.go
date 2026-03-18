package risk_test

import (
	"testing"
	"time"

	apprisk "internal/application/risk"
	domainrisk "internal/domain/risk"
)

func TestPositionExposureEvaluator_LongApproved(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluator("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Disposition != domainrisk.DispositionApproved {
		t.Fatalf("expected approved, got %s", r.Disposition)
	}
	if r.Type != "position_exposure" {
		t.Fatalf("expected type position_exposure, got %s", r.Type)
	}
	if r.Constraints.MaxPositionSize == "" {
		t.Fatal("expected max_position_size constraint")
	}
	if !r.Final {
		t.Fatal("expected final=true")
	}
	if len(r.Strategies) != 1 {
		t.Fatalf("expected 1 strategy input, got %d", len(r.Strategies))
	}
	if r.Strategies[0].Type != "mean_reversion_entry" || r.Strategies[0].Direction != "long" {
		t.Fatalf("unexpected strategy input: %+v", r.Strategies[0])
	}
}

func TestPositionExposureEvaluator_ShortApproved(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluator("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "short", "0.7500", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Disposition != domainrisk.DispositionApproved {
		t.Fatalf("expected approved, got %s", r.Disposition)
	}
	if r.Strategies[0].Direction != "short" {
		t.Fatalf("expected short direction, got %s", r.Strategies[0].Direction)
	}
}

func TestPositionExposureEvaluator_FlatApproved(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluator("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "flat", "0.0000", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Disposition != domainrisk.DispositionApproved {
		t.Fatalf("expected approved, got %s", r.Disposition)
	}
	if r.Confidence != "1.0000" {
		t.Fatalf("expected confidence 1.0000 for flat, got %s", r.Confidence)
	}
	if r.Rationale != "Flat strategy requires no position" {
		t.Fatalf("expected flat rationale, got %s", r.Rationale)
	}
}

func TestPositionExposureEvaluator_UnknownDirection(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluator("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	_, ok := eval.Evaluate("mean_reversion_entry", "sideways", "0.5000", 60, now)
	if ok {
		t.Fatal("expected evaluation to fail for unknown direction")
	}
}

func TestPositionExposureEvaluator_InvalidConfidence(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluator("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	_, ok := eval.Evaluate("mean_reversion_entry", "long", "not-a-number", 60, now)
	if ok {
		t.Fatal("expected evaluation to fail for invalid confidence")
	}
}

func TestPositionExposureEvaluator_TimestampPreserved(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluator("binancef", "btcusdt", 60)
	ts := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", 60, ts)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if !r.Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, r.Timestamp)
	}
}

func TestPositionExposureEvaluator_Validation(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluator("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if prob := r.Validate(); prob != nil {
		t.Fatalf("risk should be valid, got: %s", prob.Message)
	}
}

func TestPositionExposureEvaluator_PartitionKey(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluator("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if got := r.PartitionKey(); got != "binancef.btcusdt.60" {
		t.Fatalf("expected binancef.btcusdt.60, got %s", got)
	}
}

func TestPositionExposureEvaluator_StrategyInputPreserved(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluator("binancef", "btcusdt", 300)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.9000", 300, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if len(r.Strategies) != 1 {
		t.Fatalf("expected 1 strategy input, got %d", len(r.Strategies))
	}
	si := r.Strategies[0]
	if si.Type != "mean_reversion_entry" {
		t.Errorf("expected strategy type mean_reversion_entry, got %s", si.Type)
	}
	if si.Direction != "long" {
		t.Errorf("expected strategy direction long, got %s", si.Direction)
	}
	if si.Confidence != "0.9000" {
		t.Errorf("expected strategy confidence 0.9000, got %s", si.Confidence)
	}
	if si.Timeframe != 300 {
		t.Errorf("expected strategy timeframe 300, got %d", si.Timeframe)
	}
}

func TestPositionExposureEvaluator_ParametersPresent(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluator("binancef", "btcusdt", 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Parameters["max_position_pct"] == "" {
		t.Error("expected max_position_pct parameter")
	}
	if r.Parameters["max_portfolio_exposure_pct"] == "" {
		t.Error("expected max_portfolio_exposure_pct parameter")
	}
}
