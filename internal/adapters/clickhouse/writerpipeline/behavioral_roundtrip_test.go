package writerpipeline

import (
	"fmt"
	"math"
	"testing"
	"time"

	ch "internal/adapters/clickhouse"
	"internal/domain/decision"
	"internal/domain/execution"
	"internal/domain/risk"
	"internal/domain/strategy"
	"internal/shared/events"
)

// behavioral_roundtrip_test.go — S255: Full-stack behavioral serialization round-trip proof.
//
// These tests prove that behavioral domain objects survive the complete
// write→read serialization cycle used by the NATS→writer→ClickHouse→reader→HTTP path.
//
// Write path:  domain event → mapXxxRow() → []any (ClickHouse row)
// Read path:   []any → FormatFloat/ParseXxxJSON → domain type
//
// The tests use real behavioral scenarios from the BEHAVIORAL-WAVE-1 charter
// to verify that severity scaling, confidence factors, strategy-type-aware
// risk treatment, and context propagation all survive the round-trip.

var rtTime = time.Date(2025, 3, 20, 10, 0, 0, 0, time.UTC)

func rtMeta() events.Metadata {
	return events.Metadata{
		ID:            "rt-test-001",
		OccurredAt:    rtTime,
		CorrelationID: "corr-rt-001",
		CausationID:   "caus-rt-001",
	}
}

// --- Scenario 1: Decision severity survives round-trip ---

func TestBehavioralRoundTrip_DecisionSeverity_High(t *testing.T) {
	event := decision.DecisionEvaluatedEvent{
		Metadata: rtMeta(),
		Decision: decision.Decision{
			Type: "rsi_oversold", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Outcome:    decision.OutcomeTriggered,
			Severity:   decision.SeverityHigh,
			Confidence: "0.8333",
			Rationale:  "RSI 10.00 is 20.00 points below threshold 30 (severity: high)",
			Signals:    []decision.SignalInput{{Type: "rsi", Value: "10.00", Timeframe: 60}},
			Metadata:   map[string]string{"threshold": "30", "distance": "20.00"},
			Final:      true, Timestamp: rtTime,
		},
	}

	row := mapDecisionRow(event)

	// Simulate ClickHouse read-back: extract serialized fields from row positions.
	// Row layout: [0]event_id [1]occurred_at [2]correlation_id [3]causation_id
	//   [4]type [5]source [6]symbol [7]timeframe [8]outcome [9]confidence
	//   [10]severity [11]rationale [12]signals [13]metadata [14]final [15]timestamp
	severity := row[10].(string)
	confidence := ch.FormatFloat(row[9].(float64))
	rationale := row[11].(string)
	signals := ch.ParseSignalInputsJSON(row[12].(string))
	metadata := ch.ParseMetadataJSON(row[13].(string))

	if severity != "high" {
		t.Errorf("severity: got %q, want %q", severity, "high")
	}
	if confidence != "0.8333" {
		t.Errorf("confidence: got %q, want %q", confidence, "0.8333")
	}
	if rationale == "" {
		t.Error("rationale lost in round-trip")
	}
	if len(signals) != 1 || signals[0].Type != "rsi" || signals[0].Value != "10.00" {
		t.Errorf("signals round-trip failed: %+v", signals)
	}
	if metadata["threshold"] != "30" || metadata["distance"] != "20.00" {
		t.Errorf("metadata round-trip failed: %+v", metadata)
	}
}

func TestBehavioralRoundTrip_DecisionSeverity_Low(t *testing.T) {
	event := decision.DecisionEvaluatedEvent{
		Metadata: rtMeta(),
		Decision: decision.Decision{
			Type: "rsi_oversold", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Outcome:    decision.OutcomeTriggered,
			Severity:   decision.SeverityLow,
			Confidence: "0.4666",
			Rationale:  "RSI 25.00 is 5.00 points below threshold 30 (severity: low)",
			Signals:    []decision.SignalInput{{Type: "rsi", Value: "25.00", Timeframe: 60}},
			Final:      true, Timestamp: rtTime,
		},
	}

	row := mapDecisionRow(event)

	severity := row[10].(string)
	confidence := ch.FormatFloat(row[9].(float64))

	if severity != "low" {
		t.Errorf("severity: got %q, want %q", severity, "low")
	}
	if confidence != "0.4666" {
		t.Errorf("confidence: got %q, want %q", confidence, "0.4666")
	}
}

func TestBehavioralRoundTrip_DecisionSeverity_AllEnumValues(t *testing.T) {
	severities := []decision.Severity{
		decision.SeverityNone,
		decision.SeverityLow,
		decision.SeverityModerate,
		decision.SeverityHigh,
	}
	expected := []string{"none", "low", "moderate", "high"}

	for i, sev := range severities {
		t.Run(string(sev), func(t *testing.T) {
			event := decision.DecisionEvaluatedEvent{
				Metadata: rtMeta(),
				Decision: decision.Decision{
					Type: "rsi_oversold", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
					Outcome: decision.OutcomeTriggered, Severity: sev,
					Confidence: "0.50", Final: true, Timestamp: rtTime,
				},
			}
			row := mapDecisionRow(event)
			got := row[10].(string)
			if got != expected[i] {
				t.Errorf("severity round-trip: got %q, want %q", got, expected[i])
			}
		})
	}
}

// --- Scenario 2: Strategy carries severity-scaled confidence and decision context ---

func TestBehavioralRoundTrip_Strategy_SeverityScaledConfidence(t *testing.T) {
	// Mean reversion with high severity: decision confidence 0.8333 × severity factor 1.00 = 0.8333
	event := strategy.StrategyResolvedEvent{
		Metadata: rtMeta(),
		Strategy: strategy.Strategy{
			Type: "mean_reversion_entry", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Direction:  strategy.DirectionLong,
			Confidence: "0.8333",
			Decisions: []strategy.DecisionInput{{
				Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.8333",
				Severity: "high", Rationale: "RSI 10.00 below threshold", Timeframe: 60,
			}},
			Parameters: map[string]string{
				"target_offset": "0.03", "stop_offset": "0.01",
			},
			Metadata: map[string]string{
				"decision_rationale": "RSI 10.00 below threshold",
				"decision_severity":  "high",
			},
			Final: true, Timestamp: rtTime,
		},
	}

	row := mapStrategyRow(event)

	// Row layout: [0]event_id .. [4]type [5]source [6]symbol [7]timeframe
	//   [8]direction [9]confidence [10]decisions [11]parameters [12]metadata [13]final [14]timestamp
	confidence := ch.FormatFloat(row[9].(float64))
	decisions := ch.ParseDecisionInputsJSON(row[10].(string))
	params := ch.ParseMetadataJSON(row[11].(string))
	metadata := ch.ParseMetadataJSON(row[12].(string))

	if confidence != "0.8333" {
		t.Errorf("strategy confidence: got %q, want %q", confidence, "0.8333")
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision input, got %d", len(decisions))
	}
	if decisions[0].Severity != "high" {
		t.Errorf("decision severity in strategy: got %q, want %q", decisions[0].Severity, "high")
	}
	if decisions[0].Rationale == "" {
		t.Error("decision rationale lost in strategy round-trip")
	}
	if decisions[0].Confidence != "0.8333" {
		t.Errorf("decision confidence in strategy: got %q, want %q", decisions[0].Confidence, "0.8333")
	}
	if params["target_offset"] != "0.03" {
		t.Errorf("target_offset: got %q, want %q", params["target_offset"], "0.03")
	}
	if params["stop_offset"] != "0.01" {
		t.Errorf("stop_offset: got %q, want %q", params["stop_offset"], "0.01")
	}
	if metadata["decision_severity"] != "high" {
		t.Errorf("metadata decision_severity: got %q, want %q", metadata["decision_severity"], "high")
	}
	if metadata["decision_rationale"] == "" {
		t.Error("metadata decision_rationale lost in round-trip")
	}
}

func TestBehavioralRoundTrip_Strategy_LowSeverity_ReducedConfidence(t *testing.T) {
	// Mean reversion with low severity: confidence 0.8333 × 0.80 = 0.6666
	event := strategy.StrategyResolvedEvent{
		Metadata: rtMeta(),
		Strategy: strategy.Strategy{
			Type: "mean_reversion_entry", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Direction:  strategy.DirectionLong,
			Confidence: "0.6666",
			Decisions: []strategy.DecisionInput{{
				Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.8333",
				Severity: "low", Rationale: "RSI 25.00 below threshold", Timeframe: 60,
			}},
			Parameters: map[string]string{
				"target_offset": "0.01", "stop_offset": "0.01",
			},
			Metadata: map[string]string{
				"decision_severity": "low",
			},
			Final: true, Timestamp: rtTime,
		},
	}

	row := mapStrategyRow(event)

	confidence := ch.FormatFloat(row[9].(float64))
	decisions := ch.ParseDecisionInputsJSON(row[10].(string))

	// Low severity produces lower strategy confidence than the original decision confidence.
	stratConf := row[9].(float64)
	decConf := parseFloat(decisions[0].Confidence)
	if stratConf >= decConf {
		t.Errorf("low severity should reduce strategy confidence (%f) below decision confidence (%f)", stratConf, decConf)
	}
	if confidence != "0.6666" {
		t.Errorf("strategy confidence: got %q, want %q", confidence, "0.6666")
	}
	if decisions[0].Severity != "low" {
		t.Errorf("decision severity: got %q, want %q", decisions[0].Severity, "low")
	}
}

// --- Scenario 3: Risk assessment carries strategy-type-aware factors ---

func TestBehavioralRoundTrip_Risk_PositionExposure_CounterTrend(t *testing.T) {
	// Counter-trend (mean_reversion): confidence_factor=0.90, severity_limit_factor=1.15
	event := risk.RiskAssessedEvent{
		Metadata: rtMeta(),
		RiskAssessment: risk.RiskAssessment{
			Type: "position_exposure", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Disposition: risk.DispositionApproved,
			Confidence:  "0.7500",
			Strategies: []risk.StrategyInput{{
				Type: "mean_reversion_entry", Direction: "long", Confidence: "0.8333",
				Timeframe: 60, DecisionSeverity: "high", DecisionRationale: "RSI 10.00 below threshold",
			}},
			Constraints: risk.Constraints{
				MaxPositionSize: "0.0192",
				MaxExposure:     "1000.00",
				StopDistance:    "0.0085",
			},
			Rationale: "position_exposure: approved within limits",
			Parameters: map[string]string{
				"effective_max_position_pct": "0.0192",
			},
			Metadata: map[string]string{
				"strategy_type":         "mean_reversion_entry",
				"confidence_factor":     "0.90",
				"severity_limit_factor": "1.15",
				"decision_severity":     "high",
			},
			Final: true, Timestamp: rtTime,
		},
	}

	row := mapRiskRow(event)

	// Row layout: [0]event_id .. [4]type [5]source [6]symbol [7]timeframe
	//   [8]disposition [9]confidence [10]strategies [11]constraints [12]rationale
	//   [13]parameters [14]metadata [15]final [16]timestamp
	disposition := row[8].(string)
	confidence := ch.FormatFloat(row[9].(float64))
	strategies := ch.ParseStrategyInputsJSON(row[10].(string))
	constraints := ch.ParseConstraintsJSON(row[11].(string))
	params := ch.ParseMetadataJSON(row[13].(string))
	metadata := ch.ParseMetadataJSON(row[14].(string))

	if disposition != "approved" {
		t.Errorf("disposition: got %q, want %q", disposition, "approved")
	}
	if confidence != "0.75" {
		t.Errorf("risk confidence: got %q, want %q", confidence, "0.75")
	}

	// Strategy input round-trip with decision context.
	if len(strategies) != 1 {
		t.Fatalf("expected 1 strategy input, got %d", len(strategies))
	}
	if strategies[0].Type != "mean_reversion_entry" {
		t.Errorf("strategy type: got %q, want %q", strategies[0].Type, "mean_reversion_entry")
	}
	if strategies[0].DecisionSeverity != "high" {
		t.Errorf("decision severity in risk: got %q, want %q", strategies[0].DecisionSeverity, "high")
	}
	if strategies[0].DecisionRationale == "" {
		t.Error("decision rationale lost in risk round-trip")
	}

	// Constraints round-trip.
	if constraints.MaxPositionSize != "0.0192" {
		t.Errorf("max_position_size: got %q, want %q", constraints.MaxPositionSize, "0.0192")
	}
	if constraints.MaxExposure != "1000.00" {
		t.Errorf("max_exposure: got %q, want %q", constraints.MaxExposure, "1000.00")
	}
	if constraints.StopDistance != "0.0085" {
		t.Errorf("stop_distance: got %q, want %q", constraints.StopDistance, "0.0085")
	}

	// Behavioral metadata round-trip.
	if metadata["strategy_type"] != "mean_reversion_entry" {
		t.Errorf("metadata strategy_type: got %q, want %q", metadata["strategy_type"], "mean_reversion_entry")
	}
	if metadata["confidence_factor"] != "0.90" {
		t.Errorf("metadata confidence_factor: got %q, want %q", metadata["confidence_factor"], "0.90")
	}
	if metadata["severity_limit_factor"] != "1.15" {
		t.Errorf("metadata severity_limit_factor: got %q, want %q", metadata["severity_limit_factor"], "1.15")
	}
	if metadata["decision_severity"] != "high" {
		t.Errorf("metadata decision_severity: got %q, want %q", metadata["decision_severity"], "high")
	}

	// Parameters round-trip.
	if params["effective_max_position_pct"] != "0.0192" {
		t.Errorf("params effective_max_position_pct: got %q, want %q", params["effective_max_position_pct"], "0.0192")
	}
}

func TestBehavioralRoundTrip_Risk_DrawdownLimit_ProTrend(t *testing.T) {
	// Pro-trend (trend_following): confidence_factor=0.92, stop_type_factor=1.15 (wider stops)
	event := risk.RiskAssessedEvent{
		Metadata: rtMeta(),
		RiskAssessment: risk.RiskAssessment{
			Type: "drawdown_limit", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Disposition: risk.DispositionApproved,
			Confidence:  "0.8280",
			Strategies: []risk.StrategyInput{{
				Type: "trend_following_entry", Direction: "long", Confidence: "0.9000",
				Timeframe: 60, DecisionSeverity: "moderate", DecisionRationale: "EMA bullish crossover",
			}},
			Constraints: risk.Constraints{
				MaxPositionSize: "0.0150",
				StopDistance:    "0.0345",
			},
			Rationale: "drawdown_limit: approved within tolerance",
			Metadata: map[string]string{
				"strategy_type":             "trend_following_entry",
				"confidence_factor":         "0.92",
				"stop_type_factor":          "1.15",
				"severity_tolerance_factor": "1.00",
				"decision_severity":         "moderate",
			},
			Final: true, Timestamp: rtTime,
		},
	}

	row := mapRiskRow(event)

	strategies := ch.ParseStrategyInputsJSON(row[10].(string))
	constraints := ch.ParseConstraintsJSON(row[11].(string))
	metadata := ch.ParseMetadataJSON(row[14].(string))

	// Strategy-type-aware risk treatment survives round-trip.
	if strategies[0].Type != "trend_following_entry" {
		t.Errorf("strategy type: got %q, want %q", strategies[0].Type, "trend_following_entry")
	}
	if strategies[0].DecisionSeverity != "moderate" {
		t.Errorf("decision severity: got %q, want %q", strategies[0].DecisionSeverity, "moderate")
	}

	// Drawdown-specific constraints survive.
	if constraints.StopDistance != "0.0345" {
		t.Errorf("stop_distance: got %q, want %q", constraints.StopDistance, "0.0345")
	}

	// Pro-trend behavioral metadata.
	if metadata["confidence_factor"] != "0.92" {
		t.Errorf("confidence_factor: got %q, want %q", metadata["confidence_factor"], "0.92")
	}
	if metadata["stop_type_factor"] != "1.15" {
		t.Errorf("stop_type_factor: got %q, want %q", metadata["stop_type_factor"], "1.15")
	}
}

// --- Scenario 4: Severity contrast produces different serialized values ---

func TestBehavioralRoundTrip_SeverityContrast_HighVsLow(t *testing.T) {
	makeRiskEvent := func(sev, conf, pos string) risk.RiskAssessedEvent {
		return risk.RiskAssessedEvent{
			Metadata: rtMeta(),
			RiskAssessment: risk.RiskAssessment{
				Type: "position_exposure", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
				Disposition: risk.DispositionApproved,
				Confidence:  conf,
				Strategies: []risk.StrategyInput{{
					Type: "mean_reversion_entry", Direction: "long", Confidence: conf,
					Timeframe: 60, DecisionSeverity: sev,
				}},
				Constraints: risk.Constraints{MaxPositionSize: pos},
				Metadata:    map[string]string{"decision_severity": sev},
				Final:       true, Timestamp: rtTime,
			},
		}
	}

	highEvent := makeRiskEvent("high", "0.7500", "0.0192")
	lowEvent := makeRiskEvent("low", "0.4200", "0.0075")

	highRow := mapRiskRow(highEvent)
	lowRow := mapRiskRow(lowEvent)

	highConf := highRow[9].(float64)
	lowConf := lowRow[9].(float64)

	if highConf <= lowConf {
		t.Errorf("high severity confidence (%f) should exceed low severity confidence (%f)", highConf, lowConf)
	}

	highConstraints := ch.ParseConstraintsJSON(highRow[11].(string))
	lowConstraints := ch.ParseConstraintsJSON(lowRow[11].(string))

	highPos := parseFloat(highConstraints.MaxPositionSize)
	lowPos := parseFloat(lowConstraints.MaxPositionSize)

	if highPos <= lowPos {
		t.Errorf("high severity position (%f) should exceed low severity position (%f)", highPos, lowPos)
	}

	// Verify the severity enum itself survives.
	highMeta := ch.ParseMetadataJSON(highRow[14].(string))
	lowMeta := ch.ParseMetadataJSON(lowRow[14].(string))
	if highMeta["decision_severity"] != "high" {
		t.Errorf("high event severity metadata: got %q", highMeta["decision_severity"])
	}
	if lowMeta["decision_severity"] != "low" {
		t.Errorf("low event severity metadata: got %q", lowMeta["decision_severity"])
	}
}

// --- Scenario 5: Cross-chain strategy type produces different risk profiles ---

func TestBehavioralRoundTrip_CrossChain_RiskProfileDivergence(t *testing.T) {
	makeRisk := func(riskType, stratType, confFactor string, confidence float64) risk.RiskAssessedEvent {
		return risk.RiskAssessedEvent{
			Metadata: rtMeta(),
			RiskAssessment: risk.RiskAssessment{
				Type: riskType, Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
				Disposition: risk.DispositionApproved,
				Confidence:  fmt.Sprintf("%.4f", confidence),
				Strategies: []risk.StrategyInput{{
					Type: stratType, Direction: "long",
					Confidence: fmt.Sprintf("%.4f", confidence/0.90),
					Timeframe:  60, DecisionSeverity: "high",
				}},
				Constraints: risk.Constraints{MaxPositionSize: "0.01"},
				Metadata: map[string]string{
					"strategy_type":     stratType,
					"confidence_factor": confFactor,
				},
				Final: true, Timestamp: rtTime,
			},
		}
	}

	counterTrend := makeRisk("position_exposure", "mean_reversion_entry", "0.90", 0.7650)
	proTrend := makeRisk("position_exposure", "trend_following_entry", "0.95", 0.8550)

	ctRow := mapRiskRow(counterTrend)
	ptRow := mapRiskRow(proTrend)

	ctMeta := ch.ParseMetadataJSON(ctRow[14].(string))
	ptMeta := ch.ParseMetadataJSON(ptRow[14].(string))

	if ctMeta["strategy_type"] != "mean_reversion_entry" {
		t.Errorf("counter-trend strategy_type: got %q", ctMeta["strategy_type"])
	}
	if ptMeta["strategy_type"] != "trend_following_entry" {
		t.Errorf("pro-trend strategy_type: got %q", ptMeta["strategy_type"])
	}
	if ctMeta["confidence_factor"] != "0.90" {
		t.Errorf("counter-trend confidence_factor: got %q", ctMeta["confidence_factor"])
	}
	if ptMeta["confidence_factor"] != "0.95" {
		t.Errorf("pro-trend confidence_factor: got %q", ptMeta["confidence_factor"])
	}

	// Pro-trend should have higher risk confidence than counter-trend.
	ctConf := ctRow[9].(float64)
	ptConf := ptRow[9].(float64)
	if ptConf <= ctConf {
		t.Errorf("pro-trend confidence (%f) should exceed counter-trend confidence (%f)", ptConf, ctConf)
	}
}

// --- Scenario 6: Not-triggered path produces clean round-trip ---

func TestBehavioralRoundTrip_NotTriggered_CleanFlow(t *testing.T) {
	decEvent := decision.DecisionEvaluatedEvent{
		Metadata: rtMeta(),
		Decision: decision.Decision{
			Type: "rsi_oversold", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Outcome:    decision.OutcomeNotTriggered,
			Severity:   decision.SeverityNone,
			Confidence: "0.0000",
			Rationale:  "RSI 75.00 above threshold 30",
			Final:      true, Timestamp: rtTime,
		},
	}

	decRow := mapDecisionRow(decEvent)
	severity := decRow[10].(string)
	confidence := decRow[9].(float64)

	if severity != "none" {
		t.Errorf("not-triggered severity: got %q, want %q", severity, "none")
	}
	if confidence != 0 {
		t.Errorf("not-triggered confidence: got %f, want 0", confidence)
	}

	stratEvent := strategy.StrategyResolvedEvent{
		Metadata: rtMeta(),
		Strategy: strategy.Strategy{
			Type: "mean_reversion_entry", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Direction:  strategy.DirectionFlat,
			Confidence: "0.0000",
			Decisions: []strategy.DecisionInput{{
				Type: "rsi_oversold", Outcome: "not_triggered", Confidence: "0.0000",
				Severity: "none", Timeframe: 60,
			}},
			Final: true, Timestamp: rtTime,
		},
	}

	stratRow := mapStrategyRow(stratEvent)
	direction := stratRow[8].(string)
	stratConf := stratRow[9].(float64)
	decisions := ch.ParseDecisionInputsJSON(stratRow[10].(string))

	if direction != "flat" {
		t.Errorf("not-triggered direction: got %q, want %q", direction, "flat")
	}
	if stratConf != 0 {
		t.Errorf("not-triggered strategy confidence: got %f, want 0", stratConf)
	}
	if decisions[0].Severity != "none" {
		t.Errorf("not-triggered decision severity in strategy: got %q, want %q", decisions[0].Severity, "none")
	}
}

// --- Scenario 7: Confidence precision survives float64 round-trip ---

func TestBehavioralRoundTrip_ConfidencePrecision(t *testing.T) {
	values := []string{"0.8333", "0.6666", "0.7650", "0.8280", "0.4666", "0.9500", "1.0000", "0.0000"}

	for _, v := range values {
		t.Run(v, func(t *testing.T) {
			event := decision.DecisionEvaluatedEvent{
				Metadata: rtMeta(),
				Decision: decision.Decision{
					Type: "rsi_oversold", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
					Outcome: decision.OutcomeTriggered, Severity: decision.SeverityHigh,
					Confidence: v, Final: true, Timestamp: rtTime,
				},
			}
			row := mapDecisionRow(event)
			roundTripped := ch.FormatFloat(row[9].(float64))

			// FormatFloat uses -1 precision (natural), parseFloat uses strconv.ParseFloat.
			// Verify the values are within floating-point tolerance.
			original := parseFloat(v)
			recovered := parseFloat(roundTripped)
			if math.Abs(original-recovered) > 1e-10 {
				t.Errorf("confidence precision lost: original=%q → float64=%f → recovered=%q (delta=%e)",
					v, row[9].(float64), roundTripped, math.Abs(original-recovered))
			}
		})
	}
}

// --- Scenario 8: Full behavioral chain round-trip (decision→strategy→risk) ---

func TestBehavioralRoundTrip_FullChain_HighSeverity_MeanReversion(t *testing.T) {
	// Simulate the full behavioral chain as produced by the derive pipeline.

	// Step 1: Decision with high severity.
	decEvent := decision.DecisionEvaluatedEvent{
		Metadata: events.Metadata{ID: "dec-001", OccurredAt: rtTime, CorrelationID: "chain-001", CausationID: "signal-001"},
		Decision: decision.Decision{
			Type: "rsi_oversold", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Outcome: decision.OutcomeTriggered, Severity: decision.SeverityHigh,
			Confidence: "0.8333",
			Rationale:  "RSI 10.00 is 20.00 points below threshold 30 (severity: high)",
			Signals:    []decision.SignalInput{{Type: "rsi", Value: "10.00", Timeframe: 60}},
			Metadata:   map[string]string{"threshold": "30"},
			Final:      true, Timestamp: rtTime,
		},
	}

	// Step 2: Strategy with severity-scaled confidence (high → ×1.00).
	stratEvent := strategy.StrategyResolvedEvent{
		Metadata: events.Metadata{ID: "strat-001", OccurredAt: rtTime, CorrelationID: "chain-001", CausationID: "dec-001"},
		Strategy: strategy.Strategy{
			Type: "mean_reversion_entry", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Direction: strategy.DirectionLong, Confidence: "0.8333",
			Decisions: []strategy.DecisionInput{{
				Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.8333",
				Severity: "high", Rationale: "RSI 10.00 is 20.00 points below threshold 30 (severity: high)", Timeframe: 60,
			}},
			Parameters: map[string]string{"target_offset": "0.03", "stop_offset": "0.01"},
			Metadata:   map[string]string{"decision_rationale": "RSI 10.00 is 20.00 points below threshold 30 (severity: high)", "decision_severity": "high"},
			Final:      true, Timestamp: rtTime,
		},
	}

	// Step 3: Risk with strategy-type-aware factors (mean_reversion → 0.90 confidence factor).
	riskEvent := risk.RiskAssessedEvent{
		Metadata: events.Metadata{ID: "risk-001", OccurredAt: rtTime, CorrelationID: "chain-001", CausationID: "strat-001"},
		RiskAssessment: risk.RiskAssessment{
			Type: "position_exposure", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Disposition: risk.DispositionApproved, Confidence: "0.7500",
			Strategies: []risk.StrategyInput{{
				Type: "mean_reversion_entry", Direction: "long", Confidence: "0.8333",
				Timeframe: 60, DecisionSeverity: "high",
				DecisionRationale: "RSI 10.00 is 20.00 points below threshold 30 (severity: high)",
			}},
			Constraints: risk.Constraints{MaxPositionSize: "0.0192", MaxExposure: "1000.00", StopDistance: "0.0085"},
			Rationale:   "position_exposure: approved within limits",
			Parameters:  map[string]string{"effective_max_position_pct": "0.0192"},
			Metadata:    map[string]string{"strategy_type": "mean_reversion_entry", "confidence_factor": "0.90", "decision_severity": "high"},
			Final:       true, Timestamp: rtTime,
		},
	}

	// Map all three to rows.
	decRow := mapDecisionRow(decEvent)
	stratRow := mapStrategyRow(stratEvent)
	riskRow := mapRiskRow(riskEvent)

	// --- Verify the behavioral chain survives the round-trip ---

	// 1. Correlation ID preserved across chain.
	if decRow[2].(string) != "chain-001" {
		t.Errorf("decision correlation_id: got %q", decRow[2].(string))
	}
	if stratRow[2].(string) != "chain-001" {
		t.Errorf("strategy correlation_id: got %q", stratRow[2].(string))
	}
	if riskRow[2].(string) != "chain-001" {
		t.Errorf("risk correlation_id: got %q", riskRow[2].(string))
	}

	// 2. Causation chain: signal → decision → strategy.
	if stratRow[3].(string) != "dec-001" {
		t.Errorf("strategy causation_id: got %q, want dec-001", stratRow[3].(string))
	}
	if riskRow[3].(string) != "strat-001" {
		t.Errorf("risk causation_id: got %q, want strat-001", riskRow[3].(string))
	}

	// 3. Decision severity propagates through strategy decisions[] to risk strategies[].
	stratDecisions := ch.ParseDecisionInputsJSON(stratRow[10].(string))
	riskStrategies := ch.ParseStrategyInputsJSON(riskRow[10].(string))

	if stratDecisions[0].Severity != "high" {
		t.Errorf("strategy→decision severity: got %q", stratDecisions[0].Severity)
	}
	if riskStrategies[0].DecisionSeverity != "high" {
		t.Errorf("risk→strategy decision_severity: got %q", riskStrategies[0].DecisionSeverity)
	}

	// 4. Decision rationale survives all three stages.
	decRationale := decRow[11].(string)
	stratDecRationale := stratDecisions[0].Rationale
	riskDecRationale := riskStrategies[0].DecisionRationale

	if decRationale == "" || stratDecRationale == "" || riskDecRationale == "" {
		t.Error("rationale lost in chain")
	}
	if decRationale != stratDecRationale {
		t.Errorf("rationale diverged: decision=%q, strategy.decisions[0]=%q", decRationale, stratDecRationale)
	}
	if decRationale != riskDecRationale {
		t.Errorf("rationale diverged: decision=%q, risk.strategies[0]=%q", decRationale, riskDecRationale)
	}

	// 5. Confidence ordering: risk ≤ strategy ≤ decision.
	decConf := decRow[9].(float64)
	stratConf := stratRow[9].(float64)
	riskConf := riskRow[9].(float64)

	if stratConf > decConf+1e-10 {
		t.Errorf("strategy confidence (%f) should not exceed decision confidence (%f)", stratConf, decConf)
	}
	if riskConf > stratConf+1e-10 {
		t.Errorf("risk confidence (%f) should not exceed strategy confidence (%f)", riskConf, stratConf)
	}

	// 6. Constraints are non-zero for an approved assessment.
	riskConstraints := ch.ParseConstraintsJSON(riskRow[11].(string))
	if riskConstraints.MaxPositionSize == "" || riskConstraints.MaxPositionSize == "0" {
		t.Error("approved risk should have non-zero max_position_size")
	}
}

// =============================================================================
// S272: Execution paper analytical round-trip proof
// =============================================================================
//
// These tests prove that execution paper_order events survive the complete
// write→read serialization cycle: mapExecutionRow() → []any → ParseXxxJSON/FormatFloat.
//
// Row layout for mapExecutionRow:
//   [0]event_id [1]occurred_at [2]correlation_id [3]causation_id
//   [4]type [5]source [6]symbol [7]timeframe [8]side [9]quantity
//   [10]filled_quantity [11]status [12]risk [13]fills [14]parameters
//   [15]metadata [16]exec_correlation_id [17]exec_causation_id [18]final [19]timestamp

func execMeta() events.Metadata {
	return events.Metadata{
		ID:            "exec-rt-001",
		OccurredAt:    rtTime,
		CorrelationID: "corr-exec-001",
		CausationID:   "caus-exec-001",
	}
}

// --- Scenario 9: Basic execution paper_order survives round-trip ---

func TestBehavioralRoundTrip_Execution_BasicPaperOrder(t *testing.T) {
	event := execution.PaperOrderSubmittedEvent{
		Metadata: execMeta(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "paper_order", Source: "derive", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.0192", FilledQuantity: "0.0192",
			Status: execution.StatusFilled,
			Risk: execution.RiskInput{
				Type: "position_exposure", Disposition: "approved",
				Confidence: "0.7500", Timeframe: 60,
				StrategyType: "mean_reversion_entry", DecisionSeverity: "high",
			},
			Fills: []execution.FillRecord{{
				Price: "45000.00", Quantity: "0.0192", Fee: "0.96",
				Simulated: true, Timestamp: rtTime,
			}},
			Parameters: map[string]string{
				"target_offset": "0.03", "stop_offset": "0.01",
			},
			Metadata: map[string]string{
				"decision_severity": "high", "strategy_type": "mean_reversion_entry",
			},
			CorrelationID: "chain-exec-001",
			CausationID:   "risk-001",
			Final:         true,
			Timestamp:     rtTime,
		},
	}

	row := mapExecutionRow(event)

	// Verify envelope metadata.
	if row[0].(string) != "exec-rt-001" {
		t.Errorf("event_id: got %q, want %q", row[0], "exec-rt-001")
	}
	if row[2].(string) != "corr-exec-001" {
		t.Errorf("envelope correlation_id: got %q, want %q", row[2], "corr-exec-001")
	}
	if row[3].(string) != "caus-exec-001" {
		t.Errorf("envelope causation_id: got %q, want %q", row[3], "caus-exec-001")
	}

	// Core fields.
	if row[4].(string) != "paper_order" {
		t.Errorf("type: got %q, want %q", row[4], "paper_order")
	}
	if row[5].(string) != "derive" {
		t.Errorf("source: got %q", row[5])
	}
	if row[6].(string) != "btcusdt" {
		t.Errorf("symbol: got %q", row[6])
	}
	if row[7].(uint32) != 60 {
		t.Errorf("timeframe: got %d", row[7])
	}

	// Side and status enums.
	if row[8].(string) != "buy" {
		t.Errorf("side: got %q, want %q", row[8], "buy")
	}
	if row[11].(string) != "filled" {
		t.Errorf("status: got %q, want %q", row[11], "filled")
	}

	// Quantity round-trip via FormatFloat.
	quantity := ch.FormatFloat(row[9].(float64))
	filledQty := ch.FormatFloat(row[10].(float64))
	if quantity != "0.0192" {
		t.Errorf("quantity: got %q, want %q", quantity, "0.0192")
	}
	if filledQty != "0.0192" {
		t.Errorf("filled_quantity: got %q, want %q", filledQty, "0.0192")
	}

	// Risk causal context round-trip.
	riskInput := ch.ParseRiskInputJSON(row[12].(string))
	if riskInput.Type != "position_exposure" {
		t.Errorf("risk.type: got %q", riskInput.Type)
	}
	if riskInput.Disposition != "approved" {
		t.Errorf("risk.disposition: got %q", riskInput.Disposition)
	}
	if riskInput.Confidence != "0.7500" {
		t.Errorf("risk.confidence: got %q, want %q", riskInput.Confidence, "0.7500")
	}
	if riskInput.StrategyType != "mean_reversion_entry" {
		t.Errorf("risk.strategy_type: got %q", riskInput.StrategyType)
	}
	if riskInput.DecisionSeverity != "high" {
		t.Errorf("risk.decision_severity: got %q", riskInput.DecisionSeverity)
	}

	// Fills round-trip.
	fills := ch.ParseFillsJSON(row[13].(string))
	if len(fills) != 1 {
		t.Fatalf("fills: expected 1, got %d", len(fills))
	}
	if fills[0].Price != "45000.00" {
		t.Errorf("fill.price: got %q", fills[0].Price)
	}
	if fills[0].Quantity != "0.0192" {
		t.Errorf("fill.quantity: got %q", fills[0].Quantity)
	}
	if fills[0].Fee != "0.96" {
		t.Errorf("fill.fee: got %q", fills[0].Fee)
	}
	if !fills[0].Simulated {
		t.Error("fill.simulated should be true")
	}

	// Parameters and metadata round-trip.
	params := ch.ParseMetadataJSON(row[14].(string))
	metadata := ch.ParseMetadataJSON(row[15].(string))
	if params["target_offset"] != "0.03" {
		t.Errorf("params target_offset: got %q", params["target_offset"])
	}
	if params["stop_offset"] != "0.01" {
		t.Errorf("params stop_offset: got %q", params["stop_offset"])
	}
	if metadata["decision_severity"] != "high" {
		t.Errorf("metadata decision_severity: got %q", metadata["decision_severity"])
	}
	if metadata["strategy_type"] != "mean_reversion_entry" {
		t.Errorf("metadata strategy_type: got %q", metadata["strategy_type"])
	}

	// Exec-specific correlation/causation IDs (distinct from envelope).
	if row[16].(string) != "chain-exec-001" {
		t.Errorf("exec_correlation_id: got %q, want %q", row[16], "chain-exec-001")
	}
	if row[17].(string) != "risk-001" {
		t.Errorf("exec_causation_id: got %q, want %q", row[17], "risk-001")
	}

	// Final flag.
	if row[18].(bool) != true {
		t.Error("final should be true")
	}
}

// --- Scenario 10: Execution side enum fidelity ---

func TestBehavioralRoundTrip_Execution_SideEnumValues(t *testing.T) {
	sides := []execution.Side{execution.SideBuy, execution.SideSell, execution.SideNone}
	expected := []string{"buy", "sell", "none"}

	for i, side := range sides {
		t.Run(string(side), func(t *testing.T) {
			event := execution.PaperOrderSubmittedEvent{
				Metadata: execMeta(),
				ExecutionIntent: execution.ExecutionIntent{
					Type: "paper_order", Source: "derive", Instrument: btcUSDTPerp(t), Timeframe: 60,
					Side: side, Quantity: "0.01", FilledQuantity: "0",
					Status: execution.StatusSubmitted,
					Risk: execution.RiskInput{
						Type: "position_exposure", Disposition: "approved",
						Confidence: "0.50", Timeframe: 60,
					},
					Final: true, Timestamp: rtTime,
				},
			}
			row := mapExecutionRow(event)
			got := row[8].(string)
			if got != expected[i] {
				t.Errorf("side: got %q, want %q", got, expected[i])
			}
		})
	}
}

// --- Scenario 11: Execution status lifecycle enum fidelity ---

func TestBehavioralRoundTrip_Execution_StatusEnumValues(t *testing.T) {
	statuses := []execution.Status{
		execution.StatusSubmitted, execution.StatusSent, execution.StatusAccepted,
		execution.StatusFilled, execution.StatusPartiallyFilled,
		execution.StatusRejected, execution.StatusCancelled,
	}
	expected := []string{"submitted", "sent", "accepted", "filled", "partially_filled", "rejected", "cancelled"}

	for i, st := range statuses {
		t.Run(string(st), func(t *testing.T) {
			event := execution.PaperOrderSubmittedEvent{
				Metadata: execMeta(),
				ExecutionIntent: execution.ExecutionIntent{
					Type: "paper_order", Source: "derive", Instrument: btcUSDTPerp(t), Timeframe: 60,
					Side: execution.SideBuy, Quantity: "0.01", FilledQuantity: "0",
					Status: st,
					Risk: execution.RiskInput{
						Type: "position_exposure", Disposition: "approved",
						Confidence: "0.50", Timeframe: 60,
					},
					Final: true, Timestamp: rtTime,
				},
			}
			row := mapExecutionRow(event)
			got := row[11].(string)
			if got != expected[i] {
				t.Errorf("status: got %q, want %q", got, expected[i])
			}
		})
	}
}

// --- Scenario 12: Execution risk causal context with strategy-type-aware fields ---

func TestBehavioralRoundTrip_Execution_RiskCausalContext_CounterTrend(t *testing.T) {
	event := execution.PaperOrderSubmittedEvent{
		Metadata: execMeta(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "paper_order", Source: "derive", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.0192", FilledQuantity: "0.0192",
			Status: execution.StatusFilled,
			Risk: execution.RiskInput{
				Type: "position_exposure", Disposition: "approved",
				Confidence: "0.7500", Timeframe: 60,
				StrategyType: "mean_reversion_entry", DecisionSeverity: "high",
			},
			CorrelationID: "chain-001", CausationID: "risk-001",
			Final: true, Timestamp: rtTime,
		},
	}

	row := mapExecutionRow(event)
	riskInput := ch.ParseRiskInputJSON(row[12].(string))

	if riskInput.StrategyType != "mean_reversion_entry" {
		t.Errorf("risk.strategy_type: got %q, want %q", riskInput.StrategyType, "mean_reversion_entry")
	}
	if riskInput.DecisionSeverity != "high" {
		t.Errorf("risk.decision_severity: got %q, want %q", riskInput.DecisionSeverity, "high")
	}
	if riskInput.Confidence != "0.7500" {
		t.Errorf("risk.confidence: got %q, want %q", riskInput.Confidence, "0.7500")
	}
}

func TestBehavioralRoundTrip_Execution_RiskCausalContext_ProTrend(t *testing.T) {
	event := execution.PaperOrderSubmittedEvent{
		Metadata: execMeta(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "paper_order", Source: "derive", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.0150", FilledQuantity: "0.0150",
			Status: execution.StatusFilled,
			Risk: execution.RiskInput{
				Type: "position_exposure", Disposition: "approved",
				Confidence: "0.8550", Timeframe: 60,
				StrategyType: "trend_following_entry", DecisionSeverity: "moderate",
			},
			CorrelationID: "chain-002", CausationID: "risk-002",
			Final: true, Timestamp: rtTime,
		},
	}

	row := mapExecutionRow(event)
	riskInput := ch.ParseRiskInputJSON(row[12].(string))

	if riskInput.StrategyType != "trend_following_entry" {
		t.Errorf("risk.strategy_type: got %q, want %q", riskInput.StrategyType, "trend_following_entry")
	}
	if riskInput.DecisionSeverity != "moderate" {
		t.Errorf("risk.decision_severity: got %q, want %q", riskInput.DecisionSeverity, "moderate")
	}
	if riskInput.Confidence != "0.8550" {
		t.Errorf("risk.confidence: got %q, want %q", riskInput.Confidence, "0.8550")
	}
}

// --- Scenario 13: Multiple fills survive JSON round-trip ---

func TestBehavioralRoundTrip_Execution_MultipleFills(t *testing.T) {
	event := execution.PaperOrderSubmittedEvent{
		Metadata: execMeta(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "paper_order", Source: "derive", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.05", FilledQuantity: "0.05",
			Status: execution.StatusFilled,
			Risk: execution.RiskInput{
				Type: "position_exposure", Disposition: "approved",
				Confidence: "0.80", Timeframe: 60,
			},
			Fills: []execution.FillRecord{
				{Price: "45000.00", Quantity: "0.03", Fee: "0.60", Simulated: true, Timestamp: rtTime},
				{Price: "45010.50", Quantity: "0.02", Fee: "0.40", Simulated: true, Timestamp: rtTime.Add(time.Second)},
			},
			CorrelationID: "chain-fills", CausationID: "risk-fills",
			Final: true, Timestamp: rtTime,
		},
	}

	row := mapExecutionRow(event)
	fills := ch.ParseFillsJSON(row[13].(string))

	if len(fills) != 2 {
		t.Fatalf("expected 2 fills, got %d", len(fills))
	}

	if fills[0].Price != "45000.00" {
		t.Errorf("fill[0].price: got %q", fills[0].Price)
	}
	if fills[1].Price != "45010.50" {
		t.Errorf("fill[1].price: got %q", fills[1].Price)
	}
	if fills[0].Quantity != "0.03" {
		t.Errorf("fill[0].quantity: got %q", fills[0].Quantity)
	}
	if fills[1].Quantity != "0.02" {
		t.Errorf("fill[1].quantity: got %q", fills[1].Quantity)
	}
	if fills[0].Fee != "0.60" {
		t.Errorf("fill[0].fee: got %q", fills[0].Fee)
	}
	if fills[1].Fee != "0.40" {
		t.Errorf("fill[1].fee: got %q", fills[1].Fee)
	}
	if !fills[0].Simulated || !fills[1].Simulated {
		t.Error("all fills should be simulated=true")
	}
}

// --- Scenario 14: Execution with empty fills (submitted, not yet filled) ---

func TestBehavioralRoundTrip_Execution_EmptyFills(t *testing.T) {
	event := execution.PaperOrderSubmittedEvent{
		Metadata: execMeta(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "paper_order", Source: "derive", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.01", FilledQuantity: "0",
			Status: execution.StatusSubmitted,
			Risk: execution.RiskInput{
				Type: "position_exposure", Disposition: "approved",
				Confidence: "0.80", Timeframe: 60,
			},
			Fills:         []execution.FillRecord{},
			CorrelationID: "chain-empty", CausationID: "risk-empty",
			Final: false, Timestamp: rtTime,
		},
	}

	row := mapExecutionRow(event)
	fills := ch.ParseFillsJSON(row[13].(string))

	if len(fills) != 0 {
		t.Errorf("expected 0 fills for submitted order, got %d", len(fills))
	}
	if row[18].(bool) != false {
		t.Error("submitted order should have final=false")
	}
	if ch.FormatFloat(row[10].(float64)) != "0" {
		t.Errorf("filled_quantity should be 0 for submitted order, got %q", ch.FormatFloat(row[10].(float64)))
	}
}

// --- Scenario 15: Execution quantity precision round-trip ---

func TestBehavioralRoundTrip_Execution_QuantityPrecision(t *testing.T) {
	values := []string{"0.0192", "0.0150", "1.5000", "0.0001", "100.0000", "0.0000"}

	for _, v := range values {
		t.Run(v, func(t *testing.T) {
			event := execution.PaperOrderSubmittedEvent{
				Metadata: execMeta(),
				ExecutionIntent: execution.ExecutionIntent{
					Type: "paper_order", Source: "derive", Instrument: btcUSDTPerp(t), Timeframe: 60,
					Side: execution.SideBuy, Quantity: v, FilledQuantity: v,
					Status: execution.StatusFilled,
					Risk: execution.RiskInput{
						Type: "position_exposure", Disposition: "approved",
						Confidence: "0.80", Timeframe: 60,
					},
					Final: true, Timestamp: rtTime,
				},
			}
			row := mapExecutionRow(event)
			original := parseFloat(v)
			recovered := row[9].(float64)
			if math.Abs(original-recovered) > 1e-10 {
				t.Errorf("quantity precision lost: %q → %f (delta=%e)", v, recovered, math.Abs(original-recovered))
			}
		})
	}
}

// --- Scenario 16: Full chain decision→strategy→risk→execution round-trip ---

func TestBehavioralRoundTrip_FullChain_DecisionToExecution(t *testing.T) {
	// Step 1: Decision.
	decEvent := decision.DecisionEvaluatedEvent{
		Metadata: events.Metadata{ID: "dec-chain-001", OccurredAt: rtTime, CorrelationID: "full-chain-001", CausationID: "signal-chain-001"},
		Decision: decision.Decision{
			Type: "rsi_oversold", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Outcome: decision.OutcomeTriggered, Severity: decision.SeverityHigh,
			Confidence: "0.8333",
			Rationale:  "RSI 10.00 is 20.00 points below threshold 30 (severity: high)",
			Signals:    []decision.SignalInput{{Type: "rsi", Value: "10.00", Timeframe: 60}},
			Metadata:   map[string]string{"threshold": "30"},
			Final:      true, Timestamp: rtTime,
		},
	}

	// Step 2: Strategy.
	stratEvent := strategy.StrategyResolvedEvent{
		Metadata: events.Metadata{ID: "strat-chain-001", OccurredAt: rtTime, CorrelationID: "full-chain-001", CausationID: "dec-chain-001"},
		Strategy: strategy.Strategy{
			Type: "mean_reversion_entry", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Direction: strategy.DirectionLong, Confidence: "0.8333",
			Decisions: []strategy.DecisionInput{{
				Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.8333",
				Severity: "high", Rationale: "RSI 10.00 is 20.00 points below threshold 30 (severity: high)", Timeframe: 60,
			}},
			Parameters: map[string]string{"target_offset": "0.03", "stop_offset": "0.01"},
			Metadata:   map[string]string{"decision_severity": "high"},
			Final:      true, Timestamp: rtTime,
		},
	}

	// Step 3: Risk.
	riskEvent := risk.RiskAssessedEvent{
		Metadata: events.Metadata{ID: "risk-chain-001", OccurredAt: rtTime, CorrelationID: "full-chain-001", CausationID: "strat-chain-001"},
		RiskAssessment: risk.RiskAssessment{
			Type: "position_exposure", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Disposition: risk.DispositionApproved, Confidence: "0.7500",
			Strategies: []risk.StrategyInput{{
				Type: "mean_reversion_entry", Direction: "long", Confidence: "0.8333",
				Timeframe: 60, DecisionSeverity: "high",
				DecisionRationale: "RSI 10.00 is 20.00 points below threshold 30 (severity: high)",
			}},
			Constraints: risk.Constraints{MaxPositionSize: "0.0192", MaxExposure: "1000.00", StopDistance: "0.0085"},
			Rationale:   "position_exposure: approved within limits",
			Metadata:    map[string]string{"strategy_type": "mean_reversion_entry", "confidence_factor": "0.90", "decision_severity": "high"},
			Final:       true, Timestamp: rtTime,
		},
	}

	// Step 4: Execution (the missing link).
	execEvent := execution.PaperOrderSubmittedEvent{
		Metadata: events.Metadata{ID: "exec-chain-001", OccurredAt: rtTime, CorrelationID: "full-chain-001", CausationID: "risk-chain-001"},
		ExecutionIntent: execution.ExecutionIntent{
			Type: "paper_order", Source: "derive", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.0192", FilledQuantity: "0.0192",
			Status: execution.StatusFilled,
			Risk: execution.RiskInput{
				Type: "position_exposure", Disposition: "approved",
				Confidence: "0.7500", Timeframe: 60,
				StrategyType: "mean_reversion_entry", DecisionSeverity: "high",
			},
			Fills: []execution.FillRecord{{
				Price: "45000.00", Quantity: "0.0192", Fee: "0.96",
				Simulated: true, Timestamp: rtTime,
			}},
			Parameters:    map[string]string{"target_offset": "0.03", "stop_offset": "0.01"},
			Metadata:      map[string]string{"decision_severity": "high", "strategy_type": "mean_reversion_entry"},
			CorrelationID: "full-chain-001",
			CausationID:   "risk-chain-001",
			Final:         true,
			Timestamp:     rtTime,
		},
	}

	// Map all four to rows.
	decRow := mapDecisionRow(decEvent)
	stratRow := mapStrategyRow(stratEvent)
	riskRow := mapRiskRow(riskEvent)
	execRow := mapExecutionRow(execEvent)

	// --- Verify the full four-stage chain ---

	// 1. Correlation ID preserved across all four stages.
	for _, label := range []struct {
		name string
		row  []any
	}{
		{"decision", decRow}, {"strategy", stratRow}, {"risk", riskRow}, {"execution", execRow},
	} {
		if label.row[2].(string) != "full-chain-001" {
			t.Errorf("%s envelope correlation_id: got %q", label.name, label.row[2].(string))
		}
	}

	// 2. Causation chain: signal → decision → strategy → risk → execution.
	if stratRow[3].(string) != "dec-chain-001" {
		t.Errorf("strategy causation_id: got %q, want dec-chain-001", stratRow[3])
	}
	if riskRow[3].(string) != "strat-chain-001" {
		t.Errorf("risk causation_id: got %q, want strat-chain-001", riskRow[3])
	}
	if execRow[3].(string) != "risk-chain-001" {
		t.Errorf("execution envelope causation_id: got %q, want risk-chain-001", execRow[3])
	}

	// 3. Execution-specific correlation/causation IDs carry the domain chain.
	if execRow[16].(string) != "full-chain-001" {
		t.Errorf("exec_correlation_id: got %q, want full-chain-001", execRow[16])
	}
	if execRow[17].(string) != "risk-chain-001" {
		t.Errorf("exec_causation_id: got %q, want risk-chain-001", execRow[17])
	}

	// 4. Risk causal context in execution preserves decision severity from the chain.
	execRisk := ch.ParseRiskInputJSON(execRow[12].(string))
	if execRisk.DecisionSeverity != "high" {
		t.Errorf("execution risk.decision_severity: got %q, want %q", execRisk.DecisionSeverity, "high")
	}
	if execRisk.StrategyType != "mean_reversion_entry" {
		t.Errorf("execution risk.strategy_type: got %q, want %q", execRisk.StrategyType, "mean_reversion_entry")
	}

	// 5. Confidence ordering: risk ≤ strategy ≤ decision (execution inherits risk confidence via RiskInput).
	decConf := decRow[9].(float64)
	stratConf := stratRow[9].(float64)
	riskConf := riskRow[9].(float64)
	execRiskConf := parseFloat(execRisk.Confidence)

	if stratConf > decConf+1e-10 {
		t.Errorf("strategy confidence (%f) should not exceed decision confidence (%f)", stratConf, decConf)
	}
	if riskConf > stratConf+1e-10 {
		t.Errorf("risk confidence (%f) should not exceed strategy confidence (%f)", riskConf, stratConf)
	}
	if math.Abs(execRiskConf-riskConf) > 1e-10 {
		t.Errorf("execution risk confidence (%f) should match risk assessment confidence (%f)", execRiskConf, riskConf)
	}

	// 6. Execution quantity matches risk constraints max_position_size.
	riskConstraints := ch.ParseConstraintsJSON(riskRow[11].(string))
	execQuantity := ch.FormatFloat(execRow[9].(float64))
	if execQuantity != riskConstraints.MaxPositionSize {
		t.Errorf("execution quantity (%q) should match risk max_position_size (%q)", execQuantity, riskConstraints.MaxPositionSize)
	}

	// 7. Parameters propagate from strategy through execution.
	execParams := ch.ParseMetadataJSON(execRow[14].(string))
	stratParams := ch.ParseMetadataJSON(stratRow[11].(string))
	if execParams["target_offset"] != stratParams["target_offset"] {
		t.Errorf("target_offset diverged: strategy=%q, execution=%q", stratParams["target_offset"], execParams["target_offset"])
	}
	if execParams["stop_offset"] != stratParams["stop_offset"] {
		t.Errorf("stop_offset diverged: strategy=%q, execution=%q", stratParams["stop_offset"], execParams["stop_offset"])
	}

	// 8. Execution metadata carries decision severity from the chain.
	execMeta := ch.ParseMetadataJSON(execRow[15].(string))
	if execMeta["decision_severity"] != "high" {
		t.Errorf("execution metadata decision_severity: got %q", execMeta["decision_severity"])
	}
	if execMeta["strategy_type"] != "mean_reversion_entry" {
		t.Errorf("execution metadata strategy_type: got %q", execMeta["strategy_type"])
	}
}

// --- Scenario 17: Rejected execution produces clean serialization ---

func TestBehavioralRoundTrip_Execution_RejectedOrder(t *testing.T) {
	event := execution.PaperOrderSubmittedEvent{
		Metadata: execMeta(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "paper_order", Source: "derive", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Side: execution.SideNone, Quantity: "0", FilledQuantity: "0",
			Status: execution.StatusRejected,
			Risk: execution.RiskInput{
				Type: "position_exposure", Disposition: "rejected",
				Confidence: "0.20", Timeframe: 60,
				StrategyType: "mean_reversion_entry", DecisionSeverity: "low",
			},
			Fills:         []execution.FillRecord{},
			CorrelationID: "chain-rejected", CausationID: "risk-rejected",
			Final:     true,
			Timestamp: rtTime,
		},
	}

	row := mapExecutionRow(event)

	if row[8].(string) != "none" {
		t.Errorf("rejected side: got %q, want %q", row[8], "none")
	}
	if row[11].(string) != "rejected" {
		t.Errorf("rejected status: got %q, want %q", row[11], "rejected")
	}

	riskInput := ch.ParseRiskInputJSON(row[12].(string))
	if riskInput.Disposition != "rejected" {
		t.Errorf("risk.disposition: got %q, want %q", riskInput.Disposition, "rejected")
	}
	if riskInput.DecisionSeverity != "low" {
		t.Errorf("risk.decision_severity: got %q, want %q", riskInput.DecisionSeverity, "low")
	}

	fills := ch.ParseFillsJSON(row[13].(string))
	if len(fills) != 0 {
		t.Errorf("rejected order should have 0 fills, got %d", len(fills))
	}
}

// --- Scenario 18: S334 — Venue fill round-trip with real fill data ---

func TestBehavioralRoundTrip_VenueFill_RealFillData(t *testing.T) {
	fillTime := rtTime.Add(5 * time.Second)
	event := execution.VenueOrderFilledEvent{
		Metadata: events.Metadata{
			ID:            "s334-fill-rt-001",
			OccurredAt:    fillTime,
			CorrelationID: "corr-s334-fill",
			CausationID:   "caus-s334-intake",
		},
		ExecutionIntent: execution.ExecutionIntent{
			Type: "venue_market_order", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.001", FilledQuantity: "0.001",
			Status: execution.StatusFilled,
			Risk: execution.RiskInput{
				Type: "position_exposure", Disposition: "approved",
				Confidence: "0.85", Timeframe: 60,
				StrategyType: "mean_reversion_entry", DecisionSeverity: "high",
			},
			Fills: []execution.FillRecord{
				{Price: "98500.50", Quantity: "0.001", Fee: "0.039", Simulated: false, Timestamp: fillTime},
			},
			Parameters:    map[string]string{"max_position_pct": "0.05"},
			Metadata:      map[string]string{"venue_order_id": "1234567890"},
			CorrelationID: "corr-s334-fill",
			CausationID:   "caus-s334-risk",
			Final:         true,
			Timestamp:     fillTime,
		},
		VenueOrderID: "1234567890",
	}

	row := mapVenueFillRow(event)

	// Row layout: [0]event_id [1]occurred_at [2]correlation_id [3]causation_id
	//   [4]type [5]source [6]symbol [7]timeframe [8]side [9]quantity [10]filled_quantity
	//   [11]status [12]risk [13]fills [14]parameters [15]metadata
	//   [16]exec_correlation_id [17]exec_causation_id [18]final [19]timestamp

	if len(row) != 20 {
		t.Fatalf("venue fill row: got %d columns, want 20", len(row))
	}

	// Event metadata preservation.
	if row[0].(string) != "s334-fill-rt-001" {
		t.Errorf("event_id: got %q", row[0])
	}
	if row[2].(string) != "corr-s334-fill" {
		t.Errorf("correlation_id: got %q", row[2])
	}
	if row[3].(string) != "caus-s334-intake" {
		t.Errorf("causation_id: got %q", row[3])
	}

	// Type is venue_market_order (not paper_order).
	if row[4].(string) != "venue_market_order" {
		t.Errorf("type: got %q, want venue_market_order", row[4])
	}

	// Status is filled.
	if row[11].(string) != "filled" {
		t.Errorf("status: got %q, want filled", row[11])
	}

	// Filled quantity matches quantity (full fill).
	qty := row[9].(float64)
	filledQty := row[10].(float64)
	if qty != 0.001 {
		t.Errorf("quantity: got %f, want 0.001", qty)
	}
	if filledQty != 0.001 {
		t.Errorf("filled_quantity: got %f, want 0.001", filledQty)
	}

	// Fills JSON round-trip: real fill data with simulated=false.
	fills := ch.ParseFillsJSON(row[13].(string))
	if len(fills) != 1 {
		t.Fatalf("fills count: got %d, want 1", len(fills))
	}
	if fills[0].Price != "98500.50" {
		t.Errorf("fill.price: got %q, want 98500.50", fills[0].Price)
	}
	if fills[0].Quantity != "0.001" {
		t.Errorf("fill.quantity: got %q, want 0.001", fills[0].Quantity)
	}
	if fills[0].Fee != "0.039" {
		t.Errorf("fill.fee: got %q, want 0.039", fills[0].Fee)
	}
	if fills[0].Simulated {
		t.Error("fill.simulated: got true, want false (real venue fill)")
	}

	// Risk input preservation with strategy context.
	riskInput := ch.ParseRiskInputJSON(row[12].(string))
	if riskInput.Disposition != "approved" {
		t.Errorf("risk.disposition: got %q, want approved", riskInput.Disposition)
	}
	if riskInput.StrategyType != "mean_reversion_entry" {
		t.Errorf("risk.strategy_type: got %q", riskInput.StrategyType)
	}
	if riskInput.DecisionSeverity != "high" {
		t.Errorf("risk.decision_severity: got %q", riskInput.DecisionSeverity)
	}

	// Exec-level correlation preserved.
	if row[16].(string) != "corr-s334-fill" {
		t.Errorf("exec_correlation_id: got %q", row[16])
	}
	if row[17].(string) != "caus-s334-risk" {
		t.Errorf("exec_causation_id: got %q", row[17])
	}

	// Final flag.
	if row[18].(bool) != true {
		t.Error("final: got false, want true")
	}
}

// --- Scenario 19: S334 — Venue fill vs paper order column alignment ---

func TestBehavioralRoundTrip_VenueFill_PaperOrderColumnAlignment(t *testing.T) {
	// Both mapExecutionRow and mapVenueFillRow must produce identical column layouts.
	// This test ensures a paper_order and venue_fill for the same correlation_id
	// can coexist in the same executions table.
	paperEvent := execution.PaperOrderSubmittedEvent{
		Metadata: events.Metadata{
			ID: "s334-paper-001", OccurredAt: rtTime,
			CorrelationID: "corr-s334-align", CausationID: "caus-s334-risk",
		},
		ExecutionIntent: execution.ExecutionIntent{
			Type: "paper_order", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.001", FilledQuantity: "0",
			Status: execution.StatusSubmitted,
			Risk: execution.RiskInput{
				Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60,
			},
			Fills:         []execution.FillRecord{},
			CorrelationID: "corr-s334-align", CausationID: "caus-s334-risk",
			Final: true, Timestamp: rtTime,
		},
	}

	fillEvent := execution.VenueOrderFilledEvent{
		Metadata: events.Metadata{
			ID: "s334-fill-001", OccurredAt: rtTime.Add(2 * time.Second),
			CorrelationID: "corr-s334-align", CausationID: "caus-s334-paper",
		},
		ExecutionIntent: execution.ExecutionIntent{
			Type: "venue_market_order", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.001", FilledQuantity: "0.001",
			Status: execution.StatusFilled,
			Risk: execution.RiskInput{
				Type: "position_exposure", Disposition: "approved", Confidence: "0.85", Timeframe: 60,
			},
			Fills: []execution.FillRecord{
				{Price: "98500.00", Quantity: "0.001", Fee: "0.039", Simulated: false, Timestamp: rtTime.Add(2 * time.Second)},
			},
			CorrelationID: "corr-s334-align", CausationID: "caus-s334-risk",
			Final: true, Timestamp: rtTime.Add(2 * time.Second),
		},
		VenueOrderID: "9876543210",
	}

	paperRow := mapExecutionRow(paperEvent)
	fillRow := mapVenueFillRow(fillEvent)

	if len(paperRow) != len(fillRow) {
		t.Fatalf("column count mismatch: paper=%d, fill=%d", len(paperRow), len(fillRow))
	}

	// Type column differs (paper_order vs venue_market_order).
	if paperRow[4].(string) != "paper_order" {
		t.Errorf("paper type: got %q", paperRow[4])
	}
	if fillRow[4].(string) != "venue_market_order" {
		t.Errorf("fill type: got %q", fillRow[4])
	}

	// Status column differs (submitted vs filled).
	if paperRow[11].(string) != "submitted" {
		t.Errorf("paper status: got %q", paperRow[11])
	}
	if fillRow[11].(string) != "filled" {
		t.Errorf("fill status: got %q", fillRow[11])
	}

	// Both share the same correlation_id at exec level.
	if paperRow[16].(string) != fillRow[16].(string) {
		t.Errorf("exec_correlation_id diverged: paper=%q, fill=%q", paperRow[16], fillRow[16])
	}
}
