package risk_test

import (
	"testing"
	"time"

	appexec "internal/application/execution"
	apprisk "internal/application/risk"
	domainexec "internal/domain/execution"
	domainrisk "internal/domain/risk"
)

// TestPositionExposure_StrategyTypeConfidence verifies that risk confidence
// varies by strategy type, reflecting different risk profiles.
func TestPositionExposure_StrategyTypeConfidence(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name               string
		strategyType       string
		expectedConfidence string
	}{
		{
			name:               "mean_reversion_entry gets ×0.90 confidence",
			strategyType:       "mean_reversion_entry",
			expectedConfidence: "0.7650", // 0.85 × 0.90
		},
		{
			name:               "trend_following_entry gets ×0.95 confidence",
			strategyType:       "trend_following_entry",
			expectedConfidence: "0.8075", // 0.85 × 0.95
		},
		{
			name:               "unknown strategy gets ×0.92 default",
			strategyType:       "some_unknown_strategy",
			expectedConfidence: "0.7820", // 0.85 × 0.92
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
			r, ok := eval.Evaluate(tt.strategyType, "long", "0.8500", "moderate", "", 60, now)
			if !ok {
				t.Fatal("expected evaluation to succeed")
			}
			if r.Confidence != tt.expectedConfidence {
				t.Errorf("expected confidence %s, got %s", tt.expectedConfidence, r.Confidence)
			}
		})
	}
}

// TestPositionExposure_SeverityAdjustsPositionLimit verifies that decision severity
// scales the effective position limit.
func TestPositionExposure_SeverityAdjustsPositionLimit(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name                string
		severity            string
		expectedMaxPosition string // effective_max_position_pct in Parameters
	}{
		{
			name:                "high severity → limit ×1.15",
			severity:            "high",
			expectedMaxPosition: "0.0230", // 0.02 × 1.15
		},
		{
			name:                "moderate severity → limit ×1.00",
			severity:            "moderate",
			expectedMaxPosition: "0.0200", // 0.02 × 1.00
		},
		{
			name:                "low severity → limit ×0.80",
			severity:            "low",
			expectedMaxPosition: "0.0160", // 0.02 × 0.80
		},
		{
			name:                "empty severity → limit ×1.00",
			severity:            "",
			expectedMaxPosition: "0.0200", // 0.02 × 1.00
		},
		{
			name:                "none severity → limit ×1.00",
			severity:            "none",
			expectedMaxPosition: "0.0200", // 0.02 × 1.00
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
			r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", tt.severity, "", 60, now)
			if !ok {
				t.Fatal("expected evaluation to succeed")
			}
			if r.Parameters["effective_max_position_pct"] != tt.expectedMaxPosition {
				t.Errorf("expected effective_max_position_pct %s, got %s",
					tt.expectedMaxPosition, r.Parameters["effective_max_position_pct"])
			}
		})
	}
}

// TestPositionExposure_StrategyTypeInMetadata verifies that strategy type is
// recorded in metadata for observability.
func TestPositionExposure_StrategyTypeInMetadata(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("trend_following_entry", "long", "0.8500", "high", "EMA bullish crossover", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Metadata["strategy_type"] != "trend_following_entry" {
		t.Errorf("expected strategy_type=trend_following_entry in metadata, got %q", r.Metadata["strategy_type"])
	}
}

// TestPositionExposure_RationaleIncludesStrategyType verifies that the rationale
// explains strategy-type influence.
func TestPositionExposure_RationaleIncludesStrategyType(t *testing.T) {
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", "high", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}

	// Should contain strategy type and confidence factor.
	expected := "mean_reversion_entry (confidence ×0.90)"
	if !containsSubstring(r.Rationale, expected) {
		t.Errorf("expected rationale to contain %q, got: %s", expected, r.Rationale)
	}

	// Should contain severity limit factor.
	expectedSev := "decision severity high (limit ×1.15)"
	if !containsSubstring(r.Rationale, expectedSev) {
		t.Errorf("expected rationale to contain %q, got: %s", expectedSev, r.Rationale)
	}
}

// TestPositionExposure_CombinedStrategyAndSeverity verifies end-to-end behavior
// with both strategy type and severity influencing the assessment.
func TestPositionExposure_CombinedStrategyAndSeverity(t *testing.T) {
	now := time.Now().UTC()

	// Mean reversion with high severity: more conservative confidence, but larger position allowed.
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.9000", "high", "RSI 10.00 extreme", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}

	// Risk confidence = 0.90 × 0.90 (mean_reversion factor)
	if r.Confidence != "0.8100" {
		t.Errorf("expected confidence 0.8100, got %s", r.Confidence)
	}
	// Effective position limit = 0.02 × 1.15 (high severity)
	if r.Parameters["effective_max_position_pct"] != "0.0230" {
		t.Errorf("expected effective limit 0.0230, got %s", r.Parameters["effective_max_position_pct"])
	}
	// Position size = 0.90 × 0.023 = 0.0207
	if r.Constraints.MaxPositionSize != "0.0207" {
		t.Errorf("expected max position size 0.0207, got %s", r.Constraints.MaxPositionSize)
	}
	if r.Disposition != domainrisk.DispositionApproved {
		t.Errorf("expected approved, got %s", r.Disposition)
	}

	// Trend following with low severity: less conservative confidence, but smaller position.
	r2, ok := eval.Evaluate("trend_following_entry", "long", "0.9000", "low", "EMA weak", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}

	// Risk confidence = 0.90 × 0.95 (trend_following factor)
	if r2.Confidence != "0.8550" {
		t.Errorf("expected confidence 0.8550, got %s", r2.Confidence)
	}
	// Effective position limit = 0.02 × 0.80 (low severity)
	if r2.Parameters["effective_max_position_pct"] != "0.0160" {
		t.Errorf("expected effective limit 0.0160, got %s", r2.Parameters["effective_max_position_pct"])
	}
	// Position size = 0.90 × 0.016 = 0.0144
	if r2.Constraints.MaxPositionSize != "0.0144" {
		t.Errorf("expected max position size 0.0144, got %s", r2.Constraints.MaxPositionSize)
	}
}

// TestDrawdown_StrategyTypeConfidence verifies that risk confidence
// varies by strategy type for drawdown assessment.
func TestDrawdown_StrategyTypeConfidence(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name               string
		strategyType       string
		expectedConfidence string
	}{
		{
			name:               "mean_reversion_entry gets ×0.85 confidence",
			strategyType:       "mean_reversion_entry",
			expectedConfidence: "0.7225", // 0.85 × 0.85
		},
		{
			name:               "trend_following_entry gets ×0.92 confidence",
			strategyType:       "trend_following_entry",
			expectedConfidence: "0.7820", // 0.85 × 0.92
		},
		{
			name:               "unknown strategy gets ×0.88 default",
			strategyType:       "some_unknown_strategy",
			expectedConfidence: "0.7480", // 0.85 × 0.88
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
			r, ok := eval.Evaluate(tt.strategyType, "long", "0.8500", "moderate", "", 60, now)
			if !ok {
				t.Fatal("expected evaluation to succeed")
			}
			if r.Confidence != tt.expectedConfidence {
				t.Errorf("expected confidence %s, got %s", tt.expectedConfidence, r.Confidence)
			}
		})
	}
}

// TestDrawdown_StrategyTypeAdjustsStopBase verifies that the base stop distance
// ceiling is adjusted by strategy type.
func TestDrawdown_StrategyTypeAdjustsStopBase(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name                     string
		strategyType             string
		expectedEffectiveStopPct string
	}{
		{
			name:                     "mean_reversion_entry → tighter stop (×0.85)",
			strategyType:             "mean_reversion_entry",
			expectedEffectiveStopPct: "0.0255", // 0.03 × 0.85
		},
		{
			name:                     "trend_following_entry → wider stop (×1.15)",
			strategyType:             "trend_following_entry",
			expectedEffectiveStopPct: "0.0345", // 0.03 × 1.15
		},
		{
			name:                     "unknown → neutral stop (×1.00)",
			strategyType:             "some_unknown",
			expectedEffectiveStopPct: "0.0300", // 0.03 × 1.00
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
			r, ok := eval.Evaluate(tt.strategyType, "long", "0.8500", "moderate", "", 60, now)
			if !ok {
				t.Fatal("expected evaluation to succeed")
			}
			if r.Parameters["effective_stop_distance_pct"] != tt.expectedEffectiveStopPct {
				t.Errorf("expected effective_stop_distance_pct %s, got %s",
					tt.expectedEffectiveStopPct, r.Parameters["effective_stop_distance_pct"])
			}
		})
	}
}

// TestDrawdown_SeverityAdjustsDrawdownTolerance verifies that decision severity
// scales the effective max drawdown tolerance.
func TestDrawdown_SeverityAdjustsDrawdownTolerance(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name                      string
		severity                  string
		expectedEffectiveDrawdown string
	}{
		{
			name:                      "high severity → tolerance ×1.15",
			severity:                  "high",
			expectedEffectiveDrawdown: "0.0575", // 0.05 × 1.15
		},
		{
			name:                      "moderate severity → tolerance ×1.00",
			severity:                  "moderate",
			expectedEffectiveDrawdown: "0.0500", // 0.05 × 1.00
		},
		{
			name:                      "low severity → tolerance ×0.80",
			severity:                  "low",
			expectedEffectiveDrawdown: "0.0400", // 0.05 × 0.80
		},
		{
			name:                      "empty severity → tolerance ×1.00",
			severity:                  "",
			expectedEffectiveDrawdown: "0.0500",
		},
		{
			name:                      "none severity → tolerance ×1.00",
			severity:                  "none",
			expectedEffectiveDrawdown: "0.0500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
			r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", tt.severity, "", 60, now)
			if !ok {
				t.Fatal("expected evaluation to succeed")
			}
			if r.Parameters["effective_max_drawdown_pct"] != tt.expectedEffectiveDrawdown {
				t.Errorf("expected effective_max_drawdown_pct %s, got %s",
					tt.expectedEffectiveDrawdown, r.Parameters["effective_max_drawdown_pct"])
			}
		})
	}
}

// TestDrawdown_StrategyTypeInMetadata verifies strategy type in metadata.
func TestDrawdown_StrategyTypeInMetadata(t *testing.T) {
	eval := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("trend_following_entry", "long", "0.8500", "high", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Metadata["strategy_type"] != "trend_following_entry" {
		t.Errorf("expected strategy_type=trend_following_entry in metadata, got %q", r.Metadata["strategy_type"])
	}
}

// TestDrawdown_RationaleIncludesStrategyType verifies rationale explains strategy type.
func TestDrawdown_RationaleIncludesStrategyType(t *testing.T) {
	eval := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	now := time.Now().UTC()

	r, ok := eval.Evaluate("trend_following_entry", "long", "0.8500", "high", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}

	expected := "trend_following_entry (confidence ×0.92, stop ×1.15)"
	if !containsSubstring(r.Rationale, expected) {
		t.Errorf("expected rationale to contain %q, got: %s", expected, r.Rationale)
	}

	expectedSev := "decision severity high (tolerance ×1.15)"
	if !containsSubstring(r.Rationale, expectedSev) {
		t.Errorf("expected rationale to contain %q, got: %s", expectedSev, r.Rationale)
	}
}

// TestDrawdown_CombinedStrategyAndSeverity verifies end-to-end drawdown behavior.
func TestDrawdown_CombinedStrategyAndSeverity(t *testing.T) {
	now := time.Now().UTC()
	eval := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTPerp, 60)

	// Mean reversion with high severity: tighter stop base, more drawdown tolerance.
	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.9000", "high", "RSI 10.00 extreme", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}

	// Risk confidence = 0.90 × 0.85 (mean_reversion factor)
	if r.Confidence != "0.7650" {
		t.Errorf("expected confidence 0.7650, got %s", r.Confidence)
	}
	// Effective stop base = 0.03 × 0.85 = 0.0255
	if r.Parameters["effective_stop_distance_pct"] != "0.0255" {
		t.Errorf("expected effective stop 0.0255, got %s", r.Parameters["effective_stop_distance_pct"])
	}
	// Effective max drawdown = 0.05 × 1.15 = 0.0575
	if r.Parameters["effective_max_drawdown_pct"] != "0.0575" {
		t.Errorf("expected effective max drawdown 0.0575, got %s", r.Parameters["effective_max_drawdown_pct"])
	}
	// Stop distance = 0.0255 × 0.90 = 0.02295 → 0.0230 (rounded to 4 decimals)
	// Actually: 0.0255 * 0.9 = 0.02295 → fmt.Sprintf("%.4f", 0.02295) = "0.0230"
	if r.Disposition != domainrisk.DispositionApproved {
		t.Errorf("expected approved, got %s", r.Disposition)
	}

	// Trend following with low severity: wider stop base, tighter drawdown tolerance.
	r2, ok := eval.Evaluate("trend_following_entry", "long", "0.9000", "low", "EMA weak", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}

	// Risk confidence = 0.90 × 0.92 (trend_following factor)
	if r2.Confidence != "0.8280" {
		t.Errorf("expected confidence 0.8280, got %s", r2.Confidence)
	}
	// Effective stop base = 0.03 × 1.15 = 0.0345
	// Effective max drawdown = 0.05 × 0.80 = 0.0400
	// Stop distance = 0.0345 × 0.90 = 0.03105 → 0.0311 (rounded)
	// 0.0311 <= 0.0400 → approved
	if r2.Disposition != domainrisk.DispositionApproved {
		t.Errorf("expected approved, got %s", r2.Disposition)
	}
}

// --- S256: Edge hardening tests ---

// TestPositionExposure_RejectsZeroConfidence verifies that zero confidence
// produces a rejected disposition instead of a degenerate approved assessment.
func TestPositionExposure_RejectsZeroConfidence(t *testing.T) {
	now := time.Now().UTC()
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.0000", "high", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed (rejection is a valid assessment)")
	}
	if r.Disposition != domainrisk.DispositionRejected {
		t.Errorf("expected rejected, got %s", r.Disposition)
	}
	if r.Confidence != "0.0000" {
		t.Errorf("expected confidence 0.0000, got %s", r.Confidence)
	}
}

// TestPositionExposure_RejectsNegativeConfidence verifies that negative confidence
// produces a rejected disposition.
func TestPositionExposure_RejectsNegativeConfidence(t *testing.T) {
	now := time.Now().UTC()
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)

	r, ok := eval.Evaluate("trend_following_entry", "long", "-0.5000", "moderate", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed (rejection is a valid assessment)")
	}
	if r.Disposition != domainrisk.DispositionRejected {
		t.Errorf("expected rejected, got %s", r.Disposition)
	}
}

// TestDrawdown_RejectsZeroConfidence verifies that zero confidence
// produces a rejected disposition.
func TestDrawdown_RejectsZeroConfidence(t *testing.T) {
	now := time.Now().UTC()
	eval := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTPerp, 60)

	r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.0000", "high", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed (rejection is a valid assessment)")
	}
	if r.Disposition != domainrisk.DispositionRejected {
		t.Errorf("expected rejected, got %s", r.Disposition)
	}
	if r.Confidence != "0.0000" {
		t.Errorf("expected confidence 0.0000, got %s", r.Confidence)
	}
}

// TestDrawdown_RejectsNegativeConfidence verifies that negative confidence
// produces a rejected disposition.
func TestDrawdown_RejectsNegativeConfidence(t *testing.T) {
	now := time.Now().UTC()
	eval := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTPerp, 60)

	r, ok := eval.Evaluate("trend_following_entry", "short", "-0.1000", "low", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed (rejection is a valid assessment)")
	}
	if r.Disposition != domainrisk.DispositionRejected {
		t.Errorf("expected rejected, got %s", r.Disposition)
	}
}

// TestPositionExposure_SeverityCasingNormalization verifies that severity values
// with different casing are correctly normalized (S256: OD-BW4 edge hardening).
func TestPositionExposure_SeverityCasingNormalization(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name                string
		severity            string
		expectedMaxPosition string
	}{
		{"uppercase HIGH", "HIGH", "0.0230"},
		{"mixed case High", "High", "0.0230"},
		{"leading space", " high", "0.0230"},
		{"trailing space", "high ", "0.0230"},
		{"padded with spaces", "  moderate  ", "0.0200"},
		{"uppercase LOW", "LOW", "0.0160"},
		{"whitespace only → neutral", "   ", "0.0200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
			r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", tt.severity, "", 60, now)
			if !ok {
				t.Fatal("expected evaluation to succeed")
			}
			if r.Parameters["effective_max_position_pct"] != tt.expectedMaxPosition {
				t.Errorf("expected effective_max_position_pct %s, got %s",
					tt.expectedMaxPosition, r.Parameters["effective_max_position_pct"])
			}
		})
	}
}

// TestDrawdown_SeverityCasingNormalization verifies that severity values
// with different casing are correctly normalized for drawdown evaluator.
func TestDrawdown_SeverityCasingNormalization(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name                      string
		severity                  string
		expectedEffectiveDrawdown string
	}{
		{"uppercase HIGH", "HIGH", "0.0575"},
		{"mixed case Moderate", "Moderate", "0.0500"},
		{"padded low", " low ", "0.0400"},
		{"uppercase NONE → neutral", "NONE", "0.0500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
			r, ok := eval.Evaluate("mean_reversion_entry", "long", "0.8500", tt.severity, "", 60, now)
			if !ok {
				t.Fatal("expected evaluation to succeed")
			}
			if r.Parameters["effective_max_drawdown_pct"] != tt.expectedEffectiveDrawdown {
				t.Errorf("expected effective_max_drawdown_pct %s, got %s",
					tt.expectedEffectiveDrawdown, r.Parameters["effective_max_drawdown_pct"])
			}
		})
	}
}

// TestPositionExposure_BoundaryConfidence verifies behavior at confidence boundaries.
func TestPositionExposure_BoundaryConfidence(t *testing.T) {
	now := time.Now().UTC()
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)

	// Confidence = 1.0000 (maximum valid)
	r, ok := eval.Evaluate("trend_following_entry", "long", "1.0000", "moderate", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Disposition != domainrisk.DispositionApproved {
		t.Errorf("expected approved for max confidence, got %s", r.Disposition)
	}

	// Very small positive confidence should still produce approved (not rejected).
	r2, ok := eval.Evaluate("trend_following_entry", "long", "0.0001", "moderate", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r2.Disposition == domainrisk.DispositionRejected {
		t.Error("expected non-rejected for tiny positive confidence")
	}
}

// --- S290: Squeeze breakout entry risk scaling tests ---

// TestPositionExposure_SqueezeBreakoutConfidence verifies that squeeze_breakout_entry
// gets its explicit ×0.93 confidence factor, not the default.
func TestPositionExposure_SqueezeBreakoutConfidence(t *testing.T) {
	now := time.Now().UTC()
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)

	r, ok := eval.Evaluate("squeeze_breakout_entry", "long", "0.8500", "moderate", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	// Risk confidence = 0.85 × 0.93
	if r.Confidence != "0.7905" {
		t.Errorf("expected confidence 0.7905, got %s", r.Confidence)
	}
	if r.Metadata["strategy_type"] != "squeeze_breakout_entry" {
		t.Errorf("expected strategy_type=squeeze_breakout_entry in metadata, got %q", r.Metadata["strategy_type"])
	}
}

// TestDrawdown_SqueezeBreakoutConfidence verifies that squeeze_breakout_entry
// gets its explicit ×0.90 drawdown confidence factor.
func TestDrawdown_SqueezeBreakoutConfidence(t *testing.T) {
	now := time.Now().UTC()
	eval := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTPerp, 60)

	r, ok := eval.Evaluate("squeeze_breakout_entry", "long", "0.8500", "moderate", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	// Risk confidence = 0.85 × 0.90
	if r.Confidence != "0.7650" {
		t.Errorf("expected confidence 0.7650, got %s", r.Confidence)
	}
}

// TestDrawdown_SqueezeBreakoutStopFactor verifies that squeeze_breakout_entry
// gets its explicit ×1.05 stop distance factor.
func TestDrawdown_SqueezeBreakoutStopFactor(t *testing.T) {
	now := time.Now().UTC()
	eval := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTPerp, 60)

	r, ok := eval.Evaluate("squeeze_breakout_entry", "long", "0.8500", "moderate", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	// Effective stop base = 0.03 × 1.05 = 0.0315
	if r.Parameters["effective_stop_distance_pct"] != "0.0315" {
		t.Errorf("expected effective_stop_distance_pct 0.0315, got %s", r.Parameters["effective_stop_distance_pct"])
	}
}

// TestPositionExposure_SqueezeBreakoutCombinedHighSeverity verifies end-to-end
// squeeze breakout risk assessment with high severity.
func TestPositionExposure_SqueezeBreakoutCombinedHighSeverity(t *testing.T) {
	now := time.Now().UTC()
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)

	r, ok := eval.Evaluate("squeeze_breakout_entry", "long", "0.9000", "high", "Bollinger squeeze strong", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}

	// Risk confidence = 0.90 × 0.93
	if r.Confidence != "0.8370" {
		t.Errorf("expected confidence 0.8370, got %s", r.Confidence)
	}
	// Effective position limit = 0.02 × 1.15 (high severity)
	if r.Parameters["effective_max_position_pct"] != "0.0230" {
		t.Errorf("expected effective limit 0.0230, got %s", r.Parameters["effective_max_position_pct"])
	}
	// Position size = 0.90 × 0.023 = 0.0207
	if r.Constraints.MaxPositionSize != "0.0207" {
		t.Errorf("expected max position size 0.0207, got %s", r.Constraints.MaxPositionSize)
	}
	if r.Disposition != domainrisk.DispositionApproved {
		t.Errorf("expected approved, got %s", r.Disposition)
	}
	// Rationale should reference squeeze_breakout_entry and confidence factor.
	expected := "squeeze_breakout_entry (confidence ×0.93)"
	if !containsSubstring(r.Rationale, expected) {
		t.Errorf("expected rationale to contain %q, got: %s", expected, r.Rationale)
	}
}

// TestDrawdown_SqueezeBreakoutCombinedHighSeverity verifies end-to-end
// squeeze breakout drawdown assessment with high severity.
func TestDrawdown_SqueezeBreakoutCombinedHighSeverity(t *testing.T) {
	now := time.Now().UTC()
	eval := apprisk.NewDrawdownLimitEvaluatorForInstrument("binancef", btcUSDTPerp, 60)

	r, ok := eval.Evaluate("squeeze_breakout_entry", "long", "0.9000", "high", "Bollinger squeeze strong", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}

	// Risk confidence = 0.90 × 0.90
	if r.Confidence != "0.8100" {
		t.Errorf("expected confidence 0.8100, got %s", r.Confidence)
	}
	// Effective stop base = 0.03 × 1.05 = 0.0315
	if r.Parameters["effective_stop_distance_pct"] != "0.0315" {
		t.Errorf("expected effective stop 0.0315, got %s", r.Parameters["effective_stop_distance_pct"])
	}
	// Effective max drawdown = 0.05 × 1.15 = 0.0575
	if r.Parameters["effective_max_drawdown_pct"] != "0.0575" {
		t.Errorf("expected effective max drawdown 0.0575, got %s", r.Parameters["effective_max_drawdown_pct"])
	}
	if r.Disposition != domainrisk.DispositionApproved {
		t.Errorf("expected approved, got %s", r.Disposition)
	}
	// Rationale should reference squeeze_breakout_entry with correct factors.
	expected := "squeeze_breakout_entry (confidence ×0.90, stop ×1.05)"
	if !containsSubstring(r.Rationale, expected) {
		t.Errorf("expected rationale to contain %q, got: %s", expected, r.Rationale)
	}
}

// TestPositionExposure_SqueezeBreakoutFlat verifies that flat squeeze breakout
// strategies are approved with no constraints.
func TestPositionExposure_SqueezeBreakoutFlat(t *testing.T) {
	now := time.Now().UTC()
	eval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)

	r, ok := eval.Evaluate("squeeze_breakout_entry", "flat", "0.0000", "low", "", 60, now)
	if !ok {
		t.Fatal("expected evaluation to succeed")
	}
	if r.Disposition != domainrisk.DispositionApproved {
		t.Errorf("expected approved for flat, got %s", r.Disposition)
	}
	if r.Confidence != "1.0000" {
		t.Errorf("expected confidence 1.0000 for flat, got %s", r.Confidence)
	}
}

// TestPaperOrder_SqueezeBreakoutApproved verifies that an approved squeeze breakout
// risk assessment produces a buy paper order.
func TestPaperOrder_SqueezeBreakoutApproved(t *testing.T) {
	now := time.Now().UTC()

	// Simulate: squeeze breakout → position_exposure approved → paper order
	posEval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	risk, ok := posEval.Evaluate("squeeze_breakout_entry", "long", "0.9000", "high", "Bollinger squeeze strong", 60, now)
	if !ok {
		t.Fatal("expected risk evaluation to succeed")
	}
	if risk.Disposition != domainrisk.DispositionApproved {
		t.Fatalf("expected approved, got %s", risk.Disposition)
	}

	execEval := appexec.NewPaperOrderEvaluator("binancef", "btcusdt", 60)
	intent, ok := execEval.Evaluate(
		risk.Type, string(risk.Disposition), risk.Confidence, risk.Constraints.MaxPositionSize,
		"long", "0.8370", // strategy direction + risk-scaled confidence
		"squeeze_breakout_entry", "high",
		60, now,
	)
	if !ok {
		t.Fatal("expected execution evaluation to succeed")
	}
	if intent.Side != domainexec.SideBuy {
		t.Errorf("expected side buy, got %s", intent.Side)
	}
	if intent.Quantity != risk.Constraints.MaxPositionSize {
		t.Errorf("expected quantity %s, got %s", risk.Constraints.MaxPositionSize, intent.Quantity)
	}
	if intent.Risk.StrategyType != "squeeze_breakout_entry" {
		t.Errorf("expected strategy_type=squeeze_breakout_entry in risk input, got %q", intent.Risk.StrategyType)
	}
	if intent.Risk.DecisionSeverity != "high" {
		t.Errorf("expected decision_severity=high in risk input, got %q", intent.Risk.DecisionSeverity)
	}
	if intent.Status != domainexec.StatusSubmitted {
		t.Errorf("expected status submitted, got %s", intent.Status)
	}
}

// TestPaperOrder_SqueezeBreakoutRejected verifies that a rejected risk assessment
// produces a no-action paper order.
func TestPaperOrder_SqueezeBreakoutRejected(t *testing.T) {
	now := time.Now().UTC()

	// Simulate: squeeze breakout with zero confidence → rejected by risk
	posEval := apprisk.NewPositionExposureEvaluatorForInstrument("binancef", btcUSDTPerp, 60)
	risk, ok := posEval.Evaluate("squeeze_breakout_entry", "long", "0.0000", "low", "", 60, now)
	if !ok {
		t.Fatal("expected risk evaluation to succeed")
	}
	if risk.Disposition != domainrisk.DispositionRejected {
		t.Fatalf("expected rejected, got %s", risk.Disposition)
	}

	execEval := appexec.NewPaperOrderEvaluator("binancef", "btcusdt", 60)
	intent, ok := execEval.Evaluate(
		risk.Type, string(risk.Disposition), risk.Confidence, "0",
		"long", "0.0000",
		"squeeze_breakout_entry", "low",
		60, now,
	)
	if !ok {
		t.Fatal("expected execution evaluation to succeed")
	}
	if intent.Side != domainexec.SideNone {
		t.Errorf("expected side none for rejected risk, got %s", intent.Side)
	}
	if intent.Quantity != "0" {
		t.Errorf("expected quantity 0 for rejected risk, got %s", intent.Quantity)
	}
}

// containsSubstring is a test helper that checks substring presence.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
