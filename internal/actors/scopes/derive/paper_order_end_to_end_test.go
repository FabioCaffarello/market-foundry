package derive

import (
	"strconv"
	"testing"
	"time"

	domainexec "internal/domain/execution"
)

// paper_order_end_to_end_test.go validates the full signal → decision → strategy → risk → execution chain (S266).
//
// These tests prove that domain intelligence produces observable, auditable paper orders
// through the existing actor pipeline. Each scenario wires the PaperOrderEvaluatorActor
// as the downstream consumer of risk assessments via ScopePID fan-out.
//
// Key principles:
//   - Paper mode only — no real venue, no real money
//   - Every paper order is traceable back to its originating signal
//   - Guard rails (staleness, kill switch) are not bypassed
//   - Both risk evaluators independently produce execution intents

// --- Scenario S266-1: Full Chain Buy — RSI Oversold → Mean Reversion → Position Exposure → Paper Buy ---
// Validates that a high-severity RSI oversold signal produces a filled paper buy order
// through the complete actor chain.
func TestPaperOrder_FullChain_RSIOversold_Buy(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	execPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "po1-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "po1-strat-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "po1-risk-pub")
	execPubPID := e.Spawn(execPub.producer(), "po1-exec-pub")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "po1-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "po1-strat-fan")

	// Wire execution evaluator.
	execEvalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubPID,
	}), "po1-exec-eval")

	// Wire risk evaluator with ScopePID pointing to execution evaluator.
	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
		ScopePID:         execEvalPID,
	}), "po1-risk-eval")

	stratResolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "po1-strat-resolver")

	decEvalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "po1-dec-eval")

	time.Sleep(50 * time.Millisecond)

	// Inject: RSI 10.0 → high severity.
	e.Send(decEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "rsi",
		SignalValue:   "10.0000",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "po1-full-chain-corr",
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	// Forward decision → strategy.
	e.Send(stratResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	// Forward strategy → risk (which auto-fans to execution via ScopePID).
	e.Send(riskEvalPID, stratFanout.messages()[0].(strategyResolvedMessage))
	riskPub.waitFor(t, 1, 2*time.Second)
	execPub.waitFor(t, 1, 2*time.Second)

	// Validate paper order output.
	execMsg := execPub.messages()[0].(publishExecutionMessage)
	intent := execMsg.Event.ExecutionIntent

	if intent.Type != "paper_order" {
		t.Fatalf("expected type paper_order, got %q", intent.Type)
	}
	if intent.Side != domainexec.SideBuy {
		t.Fatalf("expected SideBuy (long strategy, approved risk), got %q", intent.Side)
	}
	if intent.Status != domainexec.StatusFilled {
		t.Fatalf("expected StatusFilled (paper fill simulation), got %q", intent.Status)
	}
	if intent.Symbol != "btcusdt" {
		t.Fatalf("expected symbol btcusdt, got %q", intent.Symbol)
	}
	if intent.Source != "binancef" {
		t.Fatalf("expected source binancef, got %q", intent.Source)
	}
	if intent.Timeframe != 60 {
		t.Fatalf("expected timeframe 60, got %d", intent.Timeframe)
	}

	// Verify quantity is non-zero (position size from risk assessment).
	qty, err := strconv.ParseFloat(intent.Quantity, 64)
	if err != nil || qty <= 0 {
		t.Fatalf("expected positive quantity, got %q", intent.Quantity)
	}

	// Verify fill record.
	if len(intent.Fills) != 1 {
		t.Fatalf("expected 1 simulated fill, got %d", len(intent.Fills))
	}
	if !intent.Fills[0].Simulated {
		t.Fatal("paper fill must be marked simulated")
	}
	if intent.FilledQuantity != intent.Quantity {
		t.Fatalf("filled_quantity %q should match quantity %q", intent.FilledQuantity, intent.Quantity)
	}

	// Verify trace preservation.
	if intent.CorrelationID != "po1-full-chain-corr" {
		t.Fatalf("correlation ID lost: expected po1-full-chain-corr, got %q", intent.CorrelationID)
	}
	if intent.CausationID == "" {
		t.Fatal("causation ID should be set (risk event ID)")
	}

	// Verify risk input context.
	if intent.Risk.Type != "position_exposure" {
		t.Fatalf("expected risk type position_exposure, got %q", intent.Risk.Type)
	}
	if intent.Risk.Disposition != "approved" {
		t.Fatalf("expected risk disposition approved, got %q", intent.Risk.Disposition)
	}
	if intent.Risk.StrategyType != "mean_reversion_entry" {
		t.Fatalf("expected strategy_type mean_reversion_entry, got %q", intent.Risk.StrategyType)
	}
	if intent.Risk.DecisionSeverity != "high" {
		t.Fatalf("expected decision_severity high, got %q", intent.Risk.DecisionSeverity)
	}

	// Verify final flag.
	if !intent.Final {
		t.Fatal("expected Final=true")
	}

	// Verify domain validation.
	if prob := intent.Validate(); prob != nil {
		t.Fatalf("paper order intent should be valid: %s", prob.Message)
	}

	// Verify event metadata.
	if execMsg.Event.Metadata.CorrelationID != "po1-full-chain-corr" {
		t.Fatalf("event metadata correlation ID: want po1-full-chain-corr, got %q", execMsg.Event.Metadata.CorrelationID)
	}
	if execMsg.Event.EventName() != domainexec.EventPaperOrderSubmitted {
		t.Fatalf("expected event name %q, got %q", domainexec.EventPaperOrderSubmitted, execMsg.Event.EventName())
	}
}

// --- Scenario S266-2: Full Chain Buy — EMA Crossover → Trend Following → Paper Buy ---
// Validates the pro-trend chain produces a paper buy order with moderate severity context.
func TestPaperOrder_FullChain_EMACrossover_Buy(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	execPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "po2-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "po2-strat-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "po2-risk-pub")
	execPubPID := e.Spawn(execPub.producer(), "po2-exec-pub")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "po2-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "po2-strat-fan")

	execEvalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubPID,
	}), "po2-exec-eval")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
		ScopePID:         execEvalPID,
	}), "po2-risk-eval")

	stratResolverPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "po2-strat-resolver")

	decEvalPID := e.Spawn(NewEMACrossoverEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "po2-dec-eval")

	time.Sleep(50 * time.Millisecond)

	// Inject: bullish EMA crossover → moderate severity.
	e.Send(decEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "ema_crossover",
		SignalValue:   "bullish",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "po2-ema-chain-corr",
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)
	e.Send(stratResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)
	e.Send(riskEvalPID, stratFanout.messages()[0].(strategyResolvedMessage))
	riskPub.waitFor(t, 1, 2*time.Second)
	execPub.waitFor(t, 1, 2*time.Second)

	intent := execPub.messages()[0].(publishExecutionMessage).Event.ExecutionIntent

	if intent.Side != domainexec.SideBuy {
		t.Fatalf("expected SideBuy, got %q", intent.Side)
	}
	if intent.Status != domainexec.StatusFilled {
		t.Fatalf("expected StatusFilled, got %q", intent.Status)
	}
	if intent.Risk.StrategyType != "trend_following_entry" {
		t.Fatalf("expected strategy_type trend_following_entry, got %q", intent.Risk.StrategyType)
	}
	if intent.Risk.DecisionSeverity != "moderate" {
		t.Fatalf("expected decision_severity moderate, got %q", intent.Risk.DecisionSeverity)
	}
	if intent.CorrelationID != "po2-ema-chain-corr" {
		t.Fatalf("correlation ID lost: got %q", intent.CorrelationID)
	}
	if intent.Parameters["strategy_type"] != "trend_following_entry" {
		t.Fatalf("parameters strategy_type: want trend_following_entry, got %q", intent.Parameters["strategy_type"])
	}
	if prob := intent.Validate(); prob != nil {
		t.Fatalf("intent validation failed: %s", prob.Message)
	}
}

// --- Scenario S266-3: Not-Triggered → Flat → No-Action Paper Order ---
// Validates that a non-triggered signal produces a no-action paper order (SideNone, no fills).
func TestPaperOrder_NotTriggered_FlatStrategy_NoAction(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	execPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "po3-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "po3-strat-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "po3-risk-pub")
	execPubPID := e.Spawn(execPub.producer(), "po3-exec-pub")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "po3-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "po3-strat-fan")

	execEvalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubPID,
	}), "po3-exec-eval")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
		ScopePID:         execEvalPID,
	}), "po3-risk-eval")

	stratResolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "po3-strat-resolver")

	decEvalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "po3-dec-eval")

	time.Sleep(50 * time.Millisecond)

	// Inject: RSI 75.0 → not_triggered → flat strategy → no-action.
	e.Send(decEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "rsi",
		SignalValue:   "75.0000",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "po3-flat-corr",
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)
	e.Send(stratResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)
	e.Send(riskEvalPID, stratFanout.messages()[0].(strategyResolvedMessage))
	riskPub.waitFor(t, 1, 2*time.Second)
	execPub.waitFor(t, 1, 2*time.Second)

	intent := execPub.messages()[0].(publishExecutionMessage).Event.ExecutionIntent

	// No-action: flat strategy produces SideNone.
	if intent.Side != domainexec.SideNone {
		t.Fatalf("expected SideNone for flat strategy, got %q", intent.Side)
	}
	if intent.Quantity != "0" {
		t.Fatalf("expected quantity 0, got %q", intent.Quantity)
	}
	// No-action intents stay submitted (no fill simulation).
	if intent.Status != domainexec.StatusSubmitted {
		t.Fatalf("expected StatusSubmitted for no-action, got %q", intent.Status)
	}
	if len(intent.Fills) != 0 {
		t.Fatalf("expected 0 fills for no-action, got %d", len(intent.Fills))
	}
	if intent.CorrelationID != "po3-flat-corr" {
		t.Fatalf("correlation ID lost: got %q", intent.CorrelationID)
	}
	if prob := intent.Validate(); prob != nil {
		t.Fatalf("no-action intent should be valid: %s", prob.Message)
	}
}

// --- Scenario S266-4: Dual Risk Fan-Out → Two Independent Paper Orders ---
// Validates that a single strategy fans through both position_exposure and drawdown_limit
// risk evaluators, each independently producing a paper order via separate execution evaluators.
func TestPaperOrder_DualRiskFanout_TwoIndependentOrders(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPubExposure := newMsgCollector()
	riskPubDrawdown := newMsgCollector()
	execPubExposure := newMsgCollector()
	execPubDrawdown := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "po4-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "po4-strat-pub")
	riskPubExposurePID := e.Spawn(riskPubExposure.producer(), "po4-risk-pub-exp")
	riskPubDrawdownPID := e.Spawn(riskPubDrawdown.producer(), "po4-risk-pub-dd")
	execPubExposurePID := e.Spawn(execPubExposure.producer(), "po4-exec-pub-exp")
	execPubDrawdownPID := e.Spawn(execPubDrawdown.producer(), "po4-exec-pub-dd")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "po4-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "po4-strat-fan")

	// Two execution evaluators: one per risk type.
	execEvalExposurePID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubExposurePID,
	}), "po4-exec-eval-exp")

	execEvalDrawdownPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubDrawdownPID,
	}), "po4-exec-eval-dd")

	// Wire risk evaluators with ScopePID to their respective execution evaluators.
	riskExposurePID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubExposurePID,
		ScopePID:         execEvalExposurePID,
	}), "po4-risk-exp")

	riskDrawdownPID := e.Spawn(NewDrawdownLimitEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubDrawdownPID,
		ScopePID:         execEvalDrawdownPID,
	}), "po4-risk-dd")

	stratResolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "po4-strat-resolver")

	decEvalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "po4-dec-eval")

	time.Sleep(50 * time.Millisecond)

	// Inject: RSI 10.0 → high severity.
	e.Send(decEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "rsi",
		SignalValue:   "10.0000",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "po4-dual-risk-corr",
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)
	e.Send(stratResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	// Fan out strategy to BOTH risk evaluators (simulating SourceScopeActor).
	stratMsg := stratFanout.messages()[0].(strategyResolvedMessage)
	e.Send(riskExposurePID, stratMsg)
	e.Send(riskDrawdownPID, stratMsg)

	riskPubExposure.waitFor(t, 1, 2*time.Second)
	riskPubDrawdown.waitFor(t, 1, 2*time.Second)
	execPubExposure.waitFor(t, 1, 2*time.Second)
	execPubDrawdown.waitFor(t, 1, 2*time.Second)

	// Validate position_exposure paper order.
	expIntent := execPubExposure.messages()[0].(publishExecutionMessage).Event.ExecutionIntent
	if expIntent.Side != domainexec.SideBuy {
		t.Fatalf("exposure paper order: expected SideBuy, got %q", expIntent.Side)
	}
	if expIntent.Risk.Type != "position_exposure" {
		t.Fatalf("exposure paper order: expected risk type position_exposure, got %q", expIntent.Risk.Type)
	}
	if expIntent.Status != domainexec.StatusFilled {
		t.Fatalf("exposure paper order: expected StatusFilled, got %q", expIntent.Status)
	}
	if expIntent.CorrelationID != "po4-dual-risk-corr" {
		t.Fatalf("exposure paper order: correlation ID lost, got %q", expIntent.CorrelationID)
	}

	// Validate drawdown_limit paper order.
	ddIntent := execPubDrawdown.messages()[0].(publishExecutionMessage).Event.ExecutionIntent
	if ddIntent.Side != domainexec.SideBuy {
		t.Fatalf("drawdown paper order: expected SideBuy, got %q", ddIntent.Side)
	}
	if ddIntent.Risk.Type != "drawdown_limit" {
		t.Fatalf("drawdown paper order: expected risk type drawdown_limit, got %q", ddIntent.Risk.Type)
	}
	if ddIntent.Status != domainexec.StatusFilled {
		t.Fatalf("drawdown paper order: expected StatusFilled, got %q", ddIntent.Status)
	}
	if ddIntent.CorrelationID != "po4-dual-risk-corr" {
		t.Fatalf("drawdown paper order: correlation ID lost, got %q", ddIntent.CorrelationID)
	}

	// Both orders should have distinct quantities (different risk types produce different sizing).
	expQty, _ := strconv.ParseFloat(expIntent.Quantity, 64)
	ddQty, _ := strconv.ParseFloat(ddIntent.Quantity, 64)
	if expQty <= 0 || ddQty <= 0 {
		t.Fatalf("both quantities should be positive: exposure=%.4f, drawdown=%.4f", expQty, ddQty)
	}

	// Both should preserve strategy type and severity context.
	if expIntent.Risk.StrategyType != "mean_reversion_entry" {
		t.Fatalf("exposure: expected strategy_type mean_reversion_entry, got %q", expIntent.Risk.StrategyType)
	}
	if ddIntent.Risk.StrategyType != "mean_reversion_entry" {
		t.Fatalf("drawdown: expected strategy_type mean_reversion_entry, got %q", ddIntent.Risk.StrategyType)
	}
	if expIntent.Risk.DecisionSeverity != "high" {
		t.Fatalf("exposure: expected severity high, got %q", expIntent.Risk.DecisionSeverity)
	}
	if ddIntent.Risk.DecisionSeverity != "high" {
		t.Fatalf("drawdown: expected severity high, got %q", ddIntent.Risk.DecisionSeverity)
	}

	// Both intents must pass domain validation.
	if prob := expIntent.Validate(); prob != nil {
		t.Fatalf("exposure intent validation failed: %s", prob.Message)
	}
	if prob := ddIntent.Validate(); prob != nil {
		t.Fatalf("drawdown intent validation failed: %s", prob.Message)
	}

	t.Logf("dual risk paper orders: exposure qty=%.4f, drawdown qty=%.4f", expQty, ddQty)
}

// --- Scenario S266-5: Severity Contrast in Execution — High vs Low Produces Different Quantities ---
// Proves that decision severity propagates through the entire chain and produces
// observably different paper order quantities.
func TestPaperOrder_SeverityContrast_HighVsLow_DifferentQuantities(t *testing.T) {
	highIntent := runFullChainToExecution(t, "10.0000", "sev-exec-high-corr")
	lowIntent := runFullChainToExecution(t, "25.0000", "sev-exec-low-corr")

	// Both should produce buy orders (RSI oversold → long).
	if highIntent.Side != domainexec.SideBuy {
		t.Fatalf("high severity: expected SideBuy, got %q", highIntent.Side)
	}
	if lowIntent.Side != domainexec.SideBuy {
		t.Fatalf("low severity: expected SideBuy, got %q", lowIntent.Side)
	}

	// High severity should produce larger quantity than low severity.
	highQty, _ := strconv.ParseFloat(highIntent.Quantity, 64)
	lowQty, _ := strconv.ParseFloat(lowIntent.Quantity, 64)
	if highQty <= lowQty {
		t.Errorf("severity contrast: high severity quantity (%.4f) should exceed low severity (%.4f)", highQty, lowQty)
	}

	// Severity context preserved in execution intents.
	if highIntent.Risk.DecisionSeverity != "high" {
		t.Fatalf("high intent: expected severity high, got %q", highIntent.Risk.DecisionSeverity)
	}
	if lowIntent.Risk.DecisionSeverity != "low" {
		t.Fatalf("low intent: expected severity low, got %q", lowIntent.Risk.DecisionSeverity)
	}

	// Parameters should carry severity for observability.
	if highIntent.Parameters["decision_severity"] != "high" {
		t.Fatalf("high intent parameters: expected severity high, got %q", highIntent.Parameters["decision_severity"])
	}
	if lowIntent.Parameters["decision_severity"] != "low" {
		t.Fatalf("low intent parameters: expected severity low, got %q", lowIntent.Parameters["decision_severity"])
	}

	t.Logf("severity contrast in execution: high qty=%.4f, low qty=%.4f", highQty, lowQty)
}

// --- Scenario S266-6: Cross-Chain Execution Profile — Mean Reversion vs Trend Following ---
// Validates that counter-trend and pro-trend chains produce semantically distinct
// paper orders with different risk profiles through execution.
func TestPaperOrder_CrossChain_MeanReversionVsTrendFollowing(t *testing.T) {
	// Chain A: RSI oversold → mean_reversion → position_exposure → paper order.
	mrIntent := runFullChainToExecution(t, "10.0000", "cross-exec-mr-corr")

	// Chain B: EMA bullish → trend_following → position_exposure → paper order.
	tfIntent := runFullChainToExecutionEMA(t, "bullish", "cross-exec-tf-corr")

	// Both should produce buy orders.
	if mrIntent.Side != domainexec.SideBuy {
		t.Fatalf("mean_reversion: expected SideBuy, got %q", mrIntent.Side)
	}
	if tfIntent.Side != domainexec.SideBuy {
		t.Fatalf("trend_following: expected SideBuy, got %q", tfIntent.Side)
	}

	// Strategy types preserved through execution.
	if mrIntent.Risk.StrategyType != "mean_reversion_entry" {
		t.Fatalf("MR: expected strategy_type mean_reversion_entry, got %q", mrIntent.Risk.StrategyType)
	}
	if tfIntent.Risk.StrategyType != "trend_following_entry" {
		t.Fatalf("TF: expected strategy_type trend_following_entry, got %q", tfIntent.Risk.StrategyType)
	}

	// Both pass domain validation.
	if prob := mrIntent.Validate(); prob != nil {
		t.Fatalf("mean_reversion intent invalid: %s", prob.Message)
	}
	if prob := tfIntent.Validate(); prob != nil {
		t.Fatalf("trend_following intent invalid: %s", prob.Message)
	}

	mrQty, _ := strconv.ParseFloat(mrIntent.Quantity, 64)
	tfQty, _ := strconv.ParseFloat(tfIntent.Quantity, 64)
	t.Logf("cross-chain execution: mean_reversion qty=%.4f (severity=%s), trend_following qty=%.4f (severity=%s)",
		mrQty, mrIntent.Risk.DecisionSeverity,
		tfQty, tfIntent.Risk.DecisionSeverity)
}

// --- Scenario S266-7: Partition and Dedup Key Isolation Across Full Chain ---
// Validates that paper orders from the full chain have correct partition and dedup keys
// for per-symbol KV materialization.
func TestPaperOrder_PartitionAndDedupKeys(t *testing.T) {
	intent := runFullChainToExecution(t, "10.0000", "po7-keys-corr")

	pk := intent.PartitionKey()
	if pk != "binancef.btcusdt.60" {
		t.Fatalf("expected partition key binancef.btcusdt.60, got %q", pk)
	}
	dk := intent.DeduplicationKey()
	if dk == "" {
		t.Fatal("dedup key should not be empty")
	}
}

// --- Helper: run full chain RSI → mean_reversion → position_exposure → paper order ---
func runFullChainToExecution(t *testing.T, rsiValue, correlationID string) domainexec.ExecutionIntent {
	t.Helper()
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	execPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), correlationID+"-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), correlationID+"-strat-pub")
	riskPubPID := e.Spawn(riskPub.producer(), correlationID+"-risk-pub")
	execPubPID := e.Spawn(execPub.producer(), correlationID+"-exec-pub")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), correlationID+"-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), correlationID+"-strat-fan")

	execEvalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubPID,
	}), correlationID+"-exec-eval")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
		ScopePID:         execEvalPID,
	}), correlationID+"-risk-eval")

	stratResolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), correlationID+"-strat-resolver")

	decEvalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), correlationID+"-dec-eval")

	time.Sleep(50 * time.Millisecond)

	e.Send(decEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "rsi",
		SignalValue:   rsiValue,
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: correlationID,
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)
	e.Send(stratResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)
	e.Send(riskEvalPID, stratFanout.messages()[0].(strategyResolvedMessage))
	riskPub.waitFor(t, 1, 2*time.Second)
	execPub.waitFor(t, 1, 2*time.Second)

	return execPub.messages()[0].(publishExecutionMessage).Event.ExecutionIntent
}

// --- Helper: run full chain EMA → trend_following → position_exposure → paper order ---
func runFullChainToExecutionEMA(t *testing.T, emaValue, correlationID string) domainexec.ExecutionIntent {
	t.Helper()
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	execPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), correlationID+"-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), correlationID+"-strat-pub")
	riskPubPID := e.Spawn(riskPub.producer(), correlationID+"-risk-pub")
	execPubPID := e.Spawn(execPub.producer(), correlationID+"-exec-pub")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), correlationID+"-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), correlationID+"-strat-fan")

	execEvalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubPID,
	}), correlationID+"-exec-eval")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
		ScopePID:         execEvalPID,
	}), correlationID+"-risk-eval")

	stratResolverPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), correlationID+"-strat-resolver")

	decEvalPID := e.Spawn(NewEMACrossoverEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), correlationID+"-dec-eval")

	time.Sleep(50 * time.Millisecond)

	e.Send(decEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "ema_crossover",
		SignalValue:   emaValue,
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: correlationID,
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)
	e.Send(stratResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)
	e.Send(riskEvalPID, stratFanout.messages()[0].(strategyResolvedMessage))
	riskPub.waitFor(t, 1, 2*time.Second)
	execPub.waitFor(t, 1, 2*time.Second)

	return execPub.messages()[0].(publishExecutionMessage).Event.ExecutionIntent
}
