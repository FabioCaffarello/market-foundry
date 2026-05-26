package writerpipeline

import (
	"encoding/json"
	"testing"
	"time"

	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/execution"
	"internal/domain/instrument"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
	"internal/shared/events"
)

var fixedTime = time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)

func btcUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("test setup: failed to build canonical BTC/USDT-perpetual: %v", prob)
	}
	return inst
}

func ethUSDTPerp(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("ETH", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("test setup: failed to build canonical ETH/USDT-perpetual: %v", prob)
	}
	return inst
}

func testMetadata() events.Metadata {
	return events.Metadata{
		ID:            "abc123",
		OccurredAt:    fixedTime,
		CorrelationID: "corr-1",
		CausationID:   "caus-1",
	}
}

func TestMapCandleRow_ColumnCount(t *testing.T) {
	e := evidence.CandleSampledEvent{
		Metadata: testMetadata(),
		Candle: evidence.EvidenceCandle{
			Source:     "binancef",
			Instrument: btcUSDTPerp(t),
			Timeframe:  60,
			Open:       "100.5",
			High:       "101.0",
			Low:        "99.0",
			Close:      "100.0",
			Volume:     "5000.123",
			TradeCount: 42,
			OpenTime:   fixedTime,
			CloseTime:  fixedTime.Add(time.Minute),
			Final:      true,
		},
	}

	row := mapCandleRow(e)

	if len(row) != 16 {
		t.Fatalf("expected 16 columns, got %d", len(row))
	}
}

func TestMapCandleRow_MetadataPositions(t *testing.T) {
	e := evidence.CandleSampledEvent{
		Metadata: testMetadata(),
		Candle: evidence.EvidenceCandle{
			Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Open: "1", High: "1", Low: "1", Close: "1", Volume: "1",
			OpenTime: fixedTime, CloseTime: fixedTime.Add(time.Minute),
		},
	}
	row := mapCandleRow(e)

	assertEq(t, "event_id", row[0], "abc123")
	assertEq(t, "occurred_at", row[1], fixedTime)
	assertEq(t, "correlation_id", row[2], "corr-1")
	assertEq(t, "causation_id", row[3], "caus-1")
}

func TestMapCandleRow_DomainFields(t *testing.T) {
	e := evidence.CandleSampledEvent{
		Metadata: testMetadata(),
		Candle: evidence.EvidenceCandle{
			Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 300,
			Open: "100.5", High: "101.0", Low: "99.0", Close: "100.0",
			Volume: "5000.123", TradeCount: 42,
			OpenTime: fixedTime, CloseTime: fixedTime.Add(5 * time.Minute),
			Final: true,
		},
	}
	row := mapCandleRow(e)

	assertEq(t, "source", row[4], "binancef")
	assertEq(t, "symbol", row[5], "btcusdt")
	assertEq(t, "timeframe", row[6], uint32(300))
	assertEq(t, "open", row[7], 100.5)
	assertEq(t, "high", row[8], 101.0)
	assertEq(t, "low", row[9], 99.0)
	assertEq(t, "close", row[10], 100.0)
	assertEq(t, "volume", row[11], 5000.123)
	assertEq(t, "trade_count", row[12], int64(42))
	assertEq(t, "open_time", row[13], fixedTime)
	assertEq(t, "close_time", row[14], fixedTime.Add(5*time.Minute))
	assertEq(t, "final", row[15], true)
}

func TestMapCandleRow_EmptyDecimalStrings(t *testing.T) {
	e := evidence.CandleSampledEvent{
		Metadata: testMetadata(),
		Candle: evidence.EvidenceCandle{
			Source: "x", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Open: "", High: "", Low: "", Close: "", Volume: "",
			OpenTime: fixedTime, CloseTime: fixedTime,
		},
	}
	row := mapCandleRow(e)

	for _, idx := range []int{7, 8, 9, 10, 11} {
		if row[idx].(float64) != 0 {
			t.Errorf("column %d: expected 0.0 for empty string, got %v", idx, row[idx])
		}
	}
}

func TestMapSignalRow_ColumnCount(t *testing.T) {
	e := signal.SignalGeneratedEvent{
		Metadata: testMetadata(),
		Signal: signal.Signal{
			Type: "rsi", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Value: "35.2", Metadata: map[string]string{"period": "14"},
			Final: true, Timestamp: fixedTime,
		},
	}
	row := mapSignalRow(e)

	if len(row) != 12 {
		t.Fatalf("expected 12 columns, got %d", len(row))
	}
}

func TestMapSignalRow_DomainFields(t *testing.T) {
	meta := map[string]string{"period": "14", "avg_gain": "1.5"}
	e := signal.SignalGeneratedEvent{
		Metadata: testMetadata(),
		Signal: signal.Signal{
			Type: "rsi", Source: "binancef", Instrument: ethUSDTPerp(t), Timeframe: 300,
			Value: "72.5", Metadata: meta,
			Final: true, Timestamp: fixedTime,
		},
	}
	row := mapSignalRow(e)

	assertEq(t, "type", row[4], "rsi")
	assertEq(t, "source", row[5], "binancef")
	assertEq(t, "symbol", row[6], "ethusdt")
	assertEq(t, "timeframe", row[7], uint32(300))
	assertEq(t, "value", row[8], 72.5)
	assertEq(t, "final", row[10], true)
	assertEq(t, "timestamp", row[11], fixedTime)

	metaJSON := row[9].(string)
	var parsed map[string]string
	if err := json.Unmarshal([]byte(metaJSON), &parsed); err != nil {
		t.Fatalf("metadata is not valid JSON: %v", err)
	}
	if parsed["period"] != "14" {
		t.Errorf("expected period=14, got %q", parsed["period"])
	}
}

func TestMapSignalRow_NilMetadata(t *testing.T) {
	e := signal.SignalGeneratedEvent{
		Metadata: testMetadata(),
		Signal: signal.Signal{
			Type: "rsi", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Value: "50", Metadata: nil,
			Final: false, Timestamp: fixedTime,
		},
	}
	row := mapSignalRow(e)

	metaJSON := row[9].(string)
	if metaJSON != "null" {
		t.Errorf("expected null for nil metadata, got %q", metaJSON)
	}
}

func TestMapDecisionRow_ColumnCount(t *testing.T) {
	e := decision.DecisionEvaluatedEvent{
		Metadata: testMetadata(),
		Decision: decision.Decision{
			Type: "rsi_oversold", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Outcome: decision.OutcomeTriggered, Severity: decision.SeverityLow,
			Confidence: "0.85", Rationale: "RSI 28.5 below threshold",
			Signals: []decision.SignalInput{{Type: "rsi", Value: "28.5", Timeframe: 60}},
			Final:   true, Timestamp: fixedTime,
		},
	}
	row := mapDecisionRow(e)

	if len(row) != 16 {
		t.Fatalf("expected 16 columns, got %d", len(row))
	}
}

func TestMapDecisionRow_DomainFields(t *testing.T) {
	signals := []decision.SignalInput{
		{Type: "rsi", Value: "28.5", Timeframe: 60},
		{Type: "ema_crossover", Value: "1", Timeframe: 300},
	}
	e := decision.DecisionEvaluatedEvent{
		Metadata: testMetadata(),
		Decision: decision.Decision{
			Type: "rsi_oversold", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Outcome: decision.OutcomeTriggered, Severity: decision.SeverityLow,
			Confidence: "0.85", Rationale: "RSI 28.5 below threshold",
			Signals: signals, Metadata: map[string]string{"threshold": "30"},
			Final: true, Timestamp: fixedTime,
		},
	}
	row := mapDecisionRow(e)

	assertEq(t, "type", row[4], "rsi_oversold")
	assertEq(t, "outcome", row[8], "triggered")
	assertEq(t, "confidence", row[9], 0.85)
	assertEq(t, "severity", row[10], "low")
	assertEq(t, "rationale", row[11], "RSI 28.5 below threshold")
	assertEq(t, "final", row[14], true)

	var parsedSignals []decision.SignalInput
	if err := json.Unmarshal([]byte(row[12].(string)), &parsedSignals); err != nil {
		t.Fatalf("signals is not valid JSON: %v", err)
	}
	if len(parsedSignals) != 2 {
		t.Errorf("expected 2 signals, got %d", len(parsedSignals))
	}
}

func TestMapStrategyRow_ColumnCount(t *testing.T) {
	e := strategy.StrategyResolvedEvent{
		Metadata: testMetadata(),
		Strategy: strategy.Strategy{
			Type: "mean_reversion_entry", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Direction: strategy.DirectionLong, Confidence: "0.75",
			Decisions: []strategy.DecisionInput{{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Severity: "low", Rationale: "RSI below threshold", Timeframe: 60}},
			Final:     true, Timestamp: fixedTime,
		},
	}
	row := mapStrategyRow(e)

	if len(row) != 15 {
		t.Fatalf("expected 15 columns, got %d", len(row))
	}
}

func TestMapStrategyRow_DomainFields(t *testing.T) {
	e := strategy.StrategyResolvedEvent{
		Metadata: testMetadata(),
		Strategy: strategy.Strategy{
			Type: "mean_reversion_entry", Source: "binancef", Instrument: ethUSDTPerp(t), Timeframe: 300,
			Direction: strategy.DirectionShort, Confidence: "0.65",
			Decisions:  []strategy.DecisionInput{{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Severity: "moderate", Rationale: "RSI 15.00 below threshold", Timeframe: 60}},
			Parameters: map[string]string{"lookback": "5"},
			Metadata:   map[string]string{"version": "1"},
			Final:      true, Timestamp: fixedTime,
		},
	}
	row := mapStrategyRow(e)

	assertEq(t, "type", row[4], "mean_reversion_entry")
	assertEq(t, "direction", row[8], "short")
	assertEq(t, "confidence", row[9], 0.65)
	assertEq(t, "final", row[13], true)

	var parsedDec []strategy.DecisionInput
	if err := json.Unmarshal([]byte(row[10].(string)), &parsedDec); err != nil {
		t.Fatalf("decisions is not valid JSON: %v", err)
	}
	if parsedDec[0].Type != "rsi_oversold" {
		t.Errorf("expected decision type rsi_oversold, got %q", parsedDec[0].Type)
	}
	if parsedDec[0].Severity != "moderate" {
		t.Errorf("expected decision severity moderate, got %q", parsedDec[0].Severity)
	}
	if parsedDec[0].Rationale == "" {
		t.Error("expected decision rationale to be non-empty")
	}
}

func TestMapRiskRow_ColumnCount(t *testing.T) {
	e := risk.RiskAssessedEvent{
		Metadata: testMetadata(),
		RiskAssessment: risk.RiskAssessment{
			Type: "position_exposure", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Disposition: risk.DispositionApproved, Confidence: "0.9",
			Strategies:  []risk.StrategyInput{{Type: "mean_reversion_entry", Direction: "long", Confidence: "0.75", Timeframe: 60, DecisionSeverity: "low", DecisionRationale: "RSI below threshold"}},
			Constraints: risk.Constraints{MaxPositionSize: "0.1", MaxExposure: "1000", StopDistance: "50"},
			Rationale:   "within limits",
			Final:       true, Timestamp: fixedTime,
		},
	}
	row := mapRiskRow(e)

	if len(row) != 17 {
		t.Fatalf("expected 17 columns, got %d", len(row))
	}
}

func TestMapRiskRow_DomainFields(t *testing.T) {
	e := risk.RiskAssessedEvent{
		Metadata: testMetadata(),
		RiskAssessment: risk.RiskAssessment{
			Type: "position_exposure", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Disposition: risk.DispositionModified, Confidence: "0.7",
			Strategies:  []risk.StrategyInput{{Type: "mean_reversion_entry", Direction: "long", Confidence: "0.75", Timeframe: 60, DecisionSeverity: "moderate", DecisionRationale: "RSI 15.00 below threshold"}},
			Constraints: risk.Constraints{MaxPositionSize: "0.05"},
			Rationale:   "position too large, modified",
			Parameters:  map[string]string{"max_risk": "0.02"},
			Metadata:    map[string]string{"version": "1"},
			Final:       true, Timestamp: fixedTime,
		},
	}
	row := mapRiskRow(e)

	assertEq(t, "disposition", row[8], "modified")
	assertEq(t, "confidence", row[9], 0.7)
	assertEq(t, "rationale", row[12], "position too large, modified")
	assertEq(t, "final", row[15], true)

	var parsedConstraints risk.Constraints
	if err := json.Unmarshal([]byte(row[11].(string)), &parsedConstraints); err != nil {
		t.Fatalf("constraints is not valid JSON: %v", err)
	}
	if parsedConstraints.MaxPositionSize != "0.05" {
		t.Errorf("expected max_position_size=0.05, got %q", parsedConstraints.MaxPositionSize)
	}
}

func TestMapRiskRow_StrategyInputDecisionContext(t *testing.T) {
	e := risk.RiskAssessedEvent{
		Metadata: testMetadata(),
		RiskAssessment: risk.RiskAssessment{
			Type: "position_exposure", Source: "binancef", Instrument: btcUSDTPerp(t), Timeframe: 60,
			Disposition: risk.DispositionApproved, Confidence: "0.8",
			Strategies:  []risk.StrategyInput{{Type: "mean_reversion_entry", Direction: "long", Confidence: "0.85", Timeframe: 60, DecisionSeverity: "high", DecisionRationale: "RSI 10.00 below threshold"}},
			Constraints: risk.Constraints{MaxPositionSize: "0.01"},
			Rationale:   "approved with high severity decision",
			Final:       true, Timestamp: fixedTime,
		},
	}
	row := mapRiskRow(e)

	// Verify strategies JSON preserves decision context.
	var parsedStrategies []risk.StrategyInput
	if err := json.Unmarshal([]byte(row[10].(string)), &parsedStrategies); err != nil {
		t.Fatalf("strategies is not valid JSON: %v", err)
	}
	if len(parsedStrategies) != 1 {
		t.Fatalf("expected 1 strategy, got %d", len(parsedStrategies))
	}
	if parsedStrategies[0].DecisionSeverity != "high" {
		t.Errorf("expected decision severity high, got %q", parsedStrategies[0].DecisionSeverity)
	}
	if parsedStrategies[0].DecisionRationale != "RSI 10.00 below threshold" {
		t.Errorf("expected decision rationale, got %q", parsedStrategies[0].DecisionRationale)
	}
}

func TestMapExecutionRow_ColumnCount(t *testing.T) {
	e := execution.PaperOrderSubmittedEvent{
		Metadata: testMetadata(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "paper_order", Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.1", FilledQuantity: "0.1",
			Status: execution.StatusFilled,
			Risk:   execution.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.9", Timeframe: 60},
			Fills:  []execution.FillRecord{{Price: "100.0", Quantity: "0.1", Fee: "0.01", Simulated: true, Timestamp: fixedTime}},
			Final:  true, Timestamp: fixedTime,
		},
	}
	row := mapExecutionRow(e)

	if len(row) != 20 {
		t.Fatalf("expected 20 columns, got %d", len(row))
	}
}

func TestMapExecutionRow_DomainFields(t *testing.T) {
	e := execution.PaperOrderSubmittedEvent{
		Metadata: testMetadata(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "paper_order", Source: "binancef", Symbol: "ethusdt", Timeframe: 300,
			Side: execution.SideSell, Quantity: "1.5", FilledQuantity: "1.0",
			Status: execution.StatusPartiallyFilled,
			Risk:   execution.RiskInput{Type: "position_exposure", Disposition: "approved", Confidence: "0.9", Timeframe: 60},
			Fills: []execution.FillRecord{
				{Price: "3500.0", Quantity: "0.5", Fee: "0.35", Simulated: true, Timestamp: fixedTime},
				{Price: "3501.0", Quantity: "0.5", Fee: "0.35", Simulated: true, Timestamp: fixedTime},
			},
			Parameters:    map[string]string{"urgency": "low"},
			Metadata:      map[string]string{"origin": "paper"},
			CorrelationID: "exec-corr-1",
			CausationID:   "exec-caus-1",
			Final:         false, Timestamp: fixedTime,
		},
	}
	row := mapExecutionRow(e)

	assertEq(t, "side", row[8], "sell")
	assertEq(t, "quantity", row[9], 1.5)
	assertEq(t, "filled_quantity", row[10], 1.0)
	assertEq(t, "status", row[11], "partially_filled")
	assertEq(t, "exec_correlation_id", row[16], "exec-corr-1")
	assertEq(t, "exec_causation_id", row[17], "exec-caus-1")
	assertEq(t, "final", row[18], false)

	var parsedFills []execution.FillRecord
	if err := json.Unmarshal([]byte(row[13].(string)), &parsedFills); err != nil {
		t.Fatalf("fills is not valid JSON: %v", err)
	}
	if len(parsedFills) != 2 {
		t.Errorf("expected 2 fills, got %d", len(parsedFills))
	}

	var parsedRisk execution.RiskInput
	if err := json.Unmarshal([]byte(row[12].(string)), &parsedRisk); err != nil {
		t.Fatalf("risk is not valid JSON: %v", err)
	}
	if parsedRisk.Type != "position_exposure" {
		t.Errorf("expected risk type position_exposure, got %q", parsedRisk.Type)
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"100.5", 100.5},
		{"0", 0},
		{"-1.23", -1.23},
		{"0.000001", 0.000001},
		{"99999999.99", 99999999.99},
		{"", 0},
		{"not-a-num", 0},
	}

	for _, tt := range tests {
		got := parseFloat(tt.input)
		if got != tt.expected {
			t.Errorf("parseFloat(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestMarshalJSON_Nil(t *testing.T) {
	result := marshalJSON(nil)
	if result != "{}" {
		t.Errorf("marshalJSON(nil) = %q, want %q", result, "{}")
	}
}

func TestMarshalJSON_Map(t *testing.T) {
	m := map[string]string{"key": "value"}
	result := marshalJSON(m)

	var parsed map[string]string
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("expected key=value, got %q", parsed["key"])
	}
}

func TestMarshalJSON_Slice(t *testing.T) {
	s := []decision.SignalInput{
		{Type: "rsi", Value: "28.5", Timeframe: 60},
	}
	result := marshalJSON(s)

	var parsed []decision.SignalInput
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if len(parsed) != 1 {
		t.Errorf("expected 1 element, got %d", len(parsed))
	}
}

func TestMarshalJSON_EmptyMap(t *testing.T) {
	m := map[string]string{}
	result := marshalJSON(m)
	if result != "{}" {
		t.Errorf("marshalJSON(empty map) = %q, want %q", result, "{}")
	}
}

func TestMarshalJSON_EmptySlice(t *testing.T) {
	s := []decision.SignalInput{}
	result := marshalJSON(s)
	if result != "[]" {
		t.Errorf("marshalJSON(empty slice) = %q, want %q", result, "[]")
	}
}

func TestMarshalJSON_Struct(t *testing.T) {
	c := risk.Constraints{MaxPositionSize: "0.1", MaxExposure: "1000"}
	result := marshalJSON(c)

	var parsed risk.Constraints
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if parsed.MaxPositionSize != "0.1" {
		t.Errorf("expected max_position_size=0.1, got %q", parsed.MaxPositionSize)
	}
}

// ── S411: Venue Rejection Row Mapper Tests ──────────────────────

func TestMapVenueRejectionRow_ColumnCount(t *testing.T) {
	e := execution.VenueOrderRejectedEvent{
		Metadata: testMetadata(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "venue_market_order", Source: "binances", Symbol: "btcusdt", Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.001", FilledQuantity: "0",
			Status: execution.StatusRejected,
			Final:  true, Timestamp: fixedTime,
		},
		RejectionCode:   "INSUFFICIENT_MARGIN",
		RejectionReason: "margin below minimum",
	}
	row := mapVenueRejectionRow(e)

	if len(row) != 20 {
		t.Fatalf("expected 20 columns (same as fill/paper), got %d", len(row))
	}
}

func TestMapVenueRejectionRow_StatusIsRejected(t *testing.T) {
	e := execution.VenueOrderRejectedEvent{
		Metadata: testMetadata(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "venue_market_order", Source: "binances", Symbol: "btcusdt", Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.001", Status: execution.StatusRejected,
			Final: true, Timestamp: fixedTime,
		},
		RejectionCode:   "INVALID_PARAMS",
		RejectionReason: "invalid quantity",
	}
	row := mapVenueRejectionRow(e)

	assertEq(t, "status", row[11], "rejected")
}

func TestMapVenueRejectionRow_MetadataContainsRejectionFields(t *testing.T) {
	e := execution.VenueOrderRejectedEvent{
		Metadata: testMetadata(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "venue_market_order", Source: "binances", Symbol: "ethusdt", Timeframe: 300,
			Side: execution.SideSell, Quantity: "1.0", Status: execution.StatusRejected,
			Metadata:      map[string]string{"origin": "testnet"},
			CorrelationID: "exec-corr-rej",
			CausationID:   "exec-caus-rej",
			Final:         true, Timestamp: fixedTime,
		},
		RejectionCode:   "INSUFFICIENT_MARGIN",
		RejectionReason: "margin below minimum for ETHUSDT",
		VenueDetails:    map[string]any{"exchange_code": "-2019", "msg": "Margin is insufficient."},
	}
	row := mapVenueRejectionRow(e)

	// Metadata column is index 15.
	metaJSON := row[15].(string)
	var parsed map[string]string
	if err := json.Unmarshal([]byte(metaJSON), &parsed); err != nil {
		t.Fatalf("metadata is not valid JSON: %v", err)
	}

	// Original metadata preserved.
	if parsed["origin"] != "testnet" {
		t.Errorf("expected origin=testnet, got %q", parsed["origin"])
	}
	// Rejection fields embedded.
	if parsed["rejection_code"] != "INSUFFICIENT_MARGIN" {
		t.Errorf("expected rejection_code=INSUFFICIENT_MARGIN, got %q", parsed["rejection_code"])
	}
	if parsed["rejection_reason"] != "margin below minimum for ETHUSDT" {
		t.Errorf("expected rejection_reason, got %q", parsed["rejection_reason"])
	}
	// Venue details embedded with prefix.
	if parsed["venue_detail.exchange_code"] != "-2019" {
		t.Errorf("expected venue_detail.exchange_code=-2019, got %q", parsed["venue_detail.exchange_code"])
	}
	if parsed["venue_detail.msg"] != "Margin is insufficient." {
		t.Errorf("expected venue_detail.msg, got %q", parsed["venue_detail.msg"])
	}

	// Correlation chain preserved.
	assertEq(t, "exec_correlation_id", row[16], "exec-corr-rej")
	assertEq(t, "exec_causation_id", row[17], "exec-caus-rej")
}

func TestMapVenueRejectionRow_NilMetadataCreatesNew(t *testing.T) {
	e := execution.VenueOrderRejectedEvent{
		Metadata: testMetadata(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "venue_market_order", Source: "binances", Symbol: "btcusdt", Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.001", Status: execution.StatusRejected,
			Metadata: nil,
			Final:    true, Timestamp: fixedTime,
		},
		RejectionCode:   "TEST_CODE",
		RejectionReason: "test reason",
	}
	row := mapVenueRejectionRow(e)

	metaJSON := row[15].(string)
	var parsed map[string]string
	if err := json.Unmarshal([]byte(metaJSON), &parsed); err != nil {
		t.Fatalf("metadata is not valid JSON: %v", err)
	}
	if parsed["rejection_code"] != "TEST_CODE" {
		t.Errorf("expected rejection_code=TEST_CODE, got %q", parsed["rejection_code"])
	}
}

func TestMapVenueRejectionRow_EmptyRejectionFieldsNotEmbedded(t *testing.T) {
	e := execution.VenueOrderRejectedEvent{
		Metadata: testMetadata(),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "venue_market_order", Source: "binances", Symbol: "btcusdt", Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.001", Status: execution.StatusRejected,
			Final: true, Timestamp: fixedTime,
		},
		RejectionCode:   "",
		RejectionReason: "",
	}
	row := mapVenueRejectionRow(e)

	metaJSON := row[15].(string)
	var parsed map[string]string
	if err := json.Unmarshal([]byte(metaJSON), &parsed); err != nil {
		t.Fatalf("metadata is not valid JSON: %v", err)
	}
	if _, ok := parsed["rejection_code"]; ok {
		t.Error("empty rejection_code should not be embedded")
	}
	if _, ok := parsed["rejection_reason"]; ok {
		t.Error("empty rejection_reason should not be embedded")
	}
}

func assertEq(t *testing.T, field string, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v (%T), want %v (%T)", field, got, got, want, want)
	}
}
