package strategy_test

import (
	"testing"

	appstrategy "internal/application/strategy"
)

func TestScaleConfidence(t *testing.T) {
	scaling := map[string]float64{
		"high":     1.00,
		"moderate": 0.90,
		"low":      0.80,
	}

	tests := []struct {
		name       string
		confidence string
		severity   string
		want       string
		wantOK     bool
	}{
		{"high severity", "0.9000", "high", "0.9000", true},
		{"moderate severity", "0.9000", "moderate", "0.8100", true},
		{"low severity", "0.9000", "low", "0.7200", true},
		{"unknown severity defaults to 1.0", "0.9000", "unknown", "0.9000", true},
		{"empty severity defaults to 1.0", "0.9000", "", "0.9000", true},
		{"invalid confidence", "not-a-number", "high", "", false},
		{"zero confidence", "0.0000", "high", "0.0000", true},
		{"clamps to 1.0", "1.5000", "high", "1.0000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := appstrategy.ScaleConfidence(tt.confidence, tt.severity, scaling)
			if ok != tt.wantOK {
				t.Fatalf("ok: want %v, got %v", tt.wantOK, ok)
			}
			if ok && got != tt.want {
				t.Errorf("want %s, got %s", tt.want, got)
			}
		})
	}
}

func TestAdjustParam(t *testing.T) {
	multiplierMap := map[string]float64{
		"high":     1.50,
		"moderate": 1.00,
		"low":      0.75,
	}

	tests := []struct {
		name     string
		base     float64
		severity string
		want     float64
	}{
		{"high severity", 0.02, "high", 0.03},
		{"moderate severity", 0.02, "moderate", 0.02},
		{"low severity", 0.02, "low", 0.015},
		{"unknown defaults to 1.0", 0.02, "unknown", 0.02},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appstrategy.AdjustParam(tt.base, tt.severity, multiplierMap)
			if got != tt.want {
				t.Errorf("want %f, got %f", tt.want, got)
			}
		})
	}
}

// --- S256: Edge hardening tests ---

// TestScaleConfidence_SeverityCasingNormalization verifies that severity values
// with whitespace and non-standard casing are correctly normalized.
func TestScaleConfidence_SeverityCasingNormalization(t *testing.T) {
	scaling := map[string]float64{
		"high":     1.00,
		"moderate": 0.90,
		"low":      0.80,
	}

	tests := []struct {
		name     string
		severity string
		want     string
	}{
		{"uppercase HIGH", "HIGH", "0.9000"},
		{"mixed case Moderate", "Moderate", "0.8100"},
		{"uppercase LOW", "LOW", "0.7200"},
		{"leading space", " high", "0.9000"},
		{"trailing space", "low ", "0.7200"},
		{"padded with spaces", "  moderate  ", "0.8100"},
		{"whitespace only → default 1.0", "   ", "0.9000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := appstrategy.ScaleConfidence("0.9000", tt.severity, scaling)
			if !ok {
				t.Fatal("expected ScaleConfidence to succeed")
			}
			if got != tt.want {
				t.Errorf("want %s, got %s", tt.want, got)
			}
		})
	}
}

// TestAdjustParam_SeverityCasingNormalization verifies AdjustParam handles
// non-standard casing and whitespace.
func TestAdjustParam_SeverityCasingNormalization(t *testing.T) {
	multiplierMap := map[string]float64{
		"high":     1.50,
		"moderate": 1.00,
		"low":      0.75,
	}

	tests := []struct {
		name     string
		severity string
		want     float64
	}{
		{"uppercase HIGH", "HIGH", 0.03},
		{"mixed case Low", "Low", 0.015},
		{"padded moderate", " moderate ", 0.02},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appstrategy.AdjustParam(0.02, tt.severity, multiplierMap)
			if got != tt.want {
				t.Errorf("want %f, got %f", tt.want, got)
			}
		})
	}
}

func TestFormatParam(t *testing.T) {
	tests := []struct {
		value float64
		want  string
	}{
		{0.02, "0.02"},
		{0.015, "0.01"},   // IEEE 754: 0.015 is slightly less than 0.015
		{0.025, "0.03"},   // IEEE 754: 0.025 is slightly more than 0.025
		{0.0375, "0.04"},  // rounds up
		{0.075, "0.07"},   // IEEE 754: 0.075 is slightly less than 0.075
	}

	for _, tt := range tests {
		got := appstrategy.FormatParam(tt.value)
		if got != tt.want {
			t.Errorf("FormatParam(%f): want %s, got %s", tt.value, tt.want, got)
		}
	}
}
