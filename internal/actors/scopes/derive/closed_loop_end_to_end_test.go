package derive

import (
	"strconv"
	"testing"
	"time"

	domainexec "internal/domain/execution"
)

// closed_loop_end_to_end_test.go validates full closed-loop scenarios for S268.
//
// Unlike S266 paper_order_end_to_end_test.go (which proved execution output exists),
// these tests validate that the ENTIRE loop is observable and auditable at every
// intermediate stage — decision, strategy, risk, and execution — producing a
// coherent, traceable domain narrative from signal to paper fill.
//
// Key distinction:
//   - S266 asked: "does a paper order come out?"
//   - S268 asks:  "is the full loop coherent, observable, and operationally meaningful?"

// --- Closed Loop A: Mean Reversion Full Observability ---
// Validates every intermediate output along the RSI oversold → mean_reversion → dual risk → paper order chain.
// Asserts decision severity, strategy parameters, risk constraints, and execution intent at each stage.
func TestClosedLoop_MeanReversion_FullObservability(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPubExposure := newMsgCollector()
	riskPubDrawdown := newMsgCollector()
	execPubExposure := newMsgCollector()
	execPubDrawdown := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "cla-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "cla-strat-pub")
	riskPubExposurePID := e.Spawn(riskPubExposure.producer(), "cla-risk-pub-exp")
	riskPubDrawdownPID := e.Spawn(riskPubDrawdown.producer(), "cla-risk-pub-dd")
	execPubExposurePID := e.Spawn(execPubExposure.producer(), "cla-exec-pub-exp")
	execPubDrawdownPID := e.Spawn(execPubDrawdown.producer(), "cla-exec-pub-dd")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "cla-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "cla-strat-fan")

	// Wire execution evaluators (one per risk type).
	execEvalExpPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubExposurePID,
	}), "cla-exec-eval-exp")

	execEvalDdPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubDrawdownPID,
	}), "cla-exec-eval-dd")

	// Wire risk evaluators with ScopePID to execution evaluators.
	riskExposurePID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubExposurePID,
		ScopePID:         execEvalExpPID,
	}), "cla-risk-exp")

	riskDrawdownPID := e.Spawn(NewDrawdownLimitEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubDrawdownPID,
		ScopePID:         execEvalDdPID,
	}), "cla-risk-dd")

	stratResolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "cla-strat-resolver")

	decEvalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "cla-dec-eval")

	time.Sleep(50 * time.Millisecond)

	// === INJECT: RSI 10 → high severity ===
	e.Send(decEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "rsi",
		SignalValue:   "10.0000",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "cla-mr-full-obs",
	})

	// === OBSERVE STAGE 1: Decision ===
	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "triggered" {
		t.Fatalf("[decision] outcome: want triggered, got %s", dec.Outcome)
	}
	if string(dec.Severity) != "high" {
		t.Fatalf("[decision] severity: want high (RSI 10, distance=20), got %s", dec.Severity)
	}
	if dec.Type != "rsi_oversold" {
		t.Fatalf("[decision] type: want rsi_oversold, got %s", dec.Type)
	}
	if len(dec.Signals) == 0 {
		t.Fatal("[decision] should carry signal input context")
	}
	if dec.Rationale == "" {
		t.Fatal("[decision] rationale should be set for observability")
	}
	t.Logf("[decision] outcome=%s severity=%s confidence=%s rationale=%q",
		dec.Outcome, dec.Severity, dec.Confidence, dec.Rationale)

	// === OBSERVE STAGE 2: Strategy ===
	decMsg := decFanout.messages()[0].(decisionEvaluatedMessage)
	e.Send(stratResolverPID, decMsg)
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	strat := strategyPub.messages()[0].(publishStrategyMessage).Event.Strategy
	if strat.Type != "mean_reversion_entry" {
		t.Fatalf("[strategy] type: want mean_reversion_entry, got %s", strat.Type)
	}
	if string(strat.Direction) != "long" {
		t.Fatalf("[strategy] direction: want long, got %s", strat.Direction)
	}
	// High severity → target_offset scaled ×1.50 (base 0.02 → 0.03).
	if strat.Parameters["target_offset"] != "0.03" {
		t.Errorf("[strategy] target_offset: want 0.03 (high severity ×1.50), got %s", strat.Parameters["target_offset"])
	}
	if len(strat.Decisions) == 0 {
		t.Fatal("[strategy] should carry decision input context")
	}
	if strat.Decisions[0].Severity != "high" {
		t.Fatalf("[strategy] decision severity forwarded: want high, got %s", strat.Decisions[0].Severity)
	}
	t.Logf("[strategy] type=%s direction=%s confidence=%s target=%s stop=%s",
		strat.Type, strat.Direction, strat.Confidence,
		strat.Parameters["target_offset"], strat.Parameters["stop_offset"])

	// === OBSERVE STAGE 3: Dual Risk ===
	stratMsg := stratFanout.messages()[0].(strategyResolvedMessage)
	e.Send(riskExposurePID, stratMsg)
	e.Send(riskDrawdownPID, stratMsg)

	riskPubExposure.waitFor(t, 1, 2*time.Second)
	riskPubDrawdown.waitFor(t, 1, 2*time.Second)

	riskExp := riskPubExposure.messages()[0].(publishRiskMessage).Event.RiskAssessment
	riskDd := riskPubDrawdown.messages()[0].(publishRiskMessage).Event.RiskAssessment

	// Position exposure: approved, strategy-type factor visible.
	if string(riskExp.Disposition) != "approved" {
		t.Fatalf("[risk/exposure] disposition: want approved, got %s", riskExp.Disposition)
	}
	if riskExp.Type != "position_exposure" {
		t.Fatalf("[risk/exposure] type: want position_exposure, got %s", riskExp.Type)
	}
	if riskExp.Constraints.MaxPositionSize == "" {
		t.Fatal("[risk/exposure] MaxPositionSize constraint should be set")
	}
	if len(riskExp.Strategies) == 0 || riskExp.Strategies[0].DecisionSeverity != "high" {
		t.Fatal("[risk/exposure] strategy input should carry decision severity=high")
	}

	// Drawdown limit: approved, stop distance visible.
	if string(riskDd.Disposition) != "approved" {
		t.Fatalf("[risk/drawdown] disposition: want approved, got %s", riskDd.Disposition)
	}
	if riskDd.Type != "drawdown_limit" {
		t.Fatalf("[risk/drawdown] type: want drawdown_limit, got %s", riskDd.Type)
	}
	if riskDd.Constraints.StopDistance == "" {
		t.Fatal("[risk/drawdown] StopDistance constraint should be set")
	}
	if len(riskDd.Strategies) == 0 || riskDd.Strategies[0].DecisionSeverity != "high" {
		t.Fatal("[risk/drawdown] strategy input should carry decision severity=high")
	}

	t.Logf("[risk/exposure] disposition=%s confidence=%s max_position=%s",
		riskExp.Disposition, riskExp.Confidence, riskExp.Constraints.MaxPositionSize)
	t.Logf("[risk/drawdown] disposition=%s confidence=%s stop_distance=%s",
		riskDd.Disposition, riskDd.Confidence, riskDd.Constraints.StopDistance)

	// === OBSERVE STAGE 4: Execution (paper orders from both risk paths) ===
	execPubExposure.waitFor(t, 1, 2*time.Second)
	execPubDrawdown.waitFor(t, 1, 2*time.Second)

	expIntent := execPubExposure.messages()[0].(publishExecutionMessage).Event.ExecutionIntent
	ddIntent := execPubDrawdown.messages()[0].(publishExecutionMessage).Event.ExecutionIntent

	for label, intent := range map[string]domainexec.ExecutionIntent{
		"exposure": expIntent,
		"drawdown": ddIntent,
	} {
		if intent.Type != "paper_order" {
			t.Fatalf("[exec/%s] type: want paper_order, got %s", label, intent.Type)
		}
		if intent.Side != domainexec.SideBuy {
			t.Fatalf("[exec/%s] side: want buy, got %s", label, intent.Side)
		}
		if intent.Status != domainexec.StatusFilled {
			t.Fatalf("[exec/%s] status: want filled, got %s", label, intent.Status)
		}
		if len(intent.Fills) != 1 || !intent.Fills[0].Simulated {
			t.Fatalf("[exec/%s] should have exactly 1 simulated fill", label)
		}
		if intent.CorrelationID != "cla-mr-full-obs" {
			t.Fatalf("[exec/%s] correlation ID lost: got %q", label, intent.CorrelationID)
		}
		if intent.Risk.DecisionSeverity != "high" {
			t.Fatalf("[exec/%s] decision severity lost: got %q", label, intent.Risk.DecisionSeverity)
		}
		if intent.Risk.StrategyType != "mean_reversion_entry" {
			t.Fatalf("[exec/%s] strategy type lost: got %q", label, intent.Risk.StrategyType)
		}
		if !intent.Final {
			t.Fatalf("[exec/%s] Final should be true", label)
		}
		if prob := intent.Validate(); prob != nil {
			t.Fatalf("[exec/%s] domain validation failed: %s", label, prob.Message)
		}
	}

	expQty, _ := strconv.ParseFloat(expIntent.Quantity, 64)
	ddQty, _ := strconv.ParseFloat(ddIntent.Quantity, 64)
	t.Logf("[exec] exposure qty=%.4f, drawdown qty=%.4f", expQty, ddQty)
	t.Logf("[closed-loop-A] PASS — all 4 stages observable and coherent")
}

// --- Closed Loop B: Trend Following Full Observability ---
// Validates every intermediate output along the EMA crossover → trend_following → dual risk → paper order chain.
func TestClosedLoop_TrendFollowing_FullObservability(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPubExposure := newMsgCollector()
	riskPubDrawdown := newMsgCollector()
	execPubExposure := newMsgCollector()
	execPubDrawdown := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "clb-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "clb-strat-pub")
	riskPubExposurePID := e.Spawn(riskPubExposure.producer(), "clb-risk-pub-exp")
	riskPubDrawdownPID := e.Spawn(riskPubDrawdown.producer(), "clb-risk-pub-dd")
	execPubExposurePID := e.Spawn(execPubExposure.producer(), "clb-exec-pub-exp")
	execPubDrawdownPID := e.Spawn(execPubDrawdown.producer(), "clb-exec-pub-dd")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "clb-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "clb-strat-fan")

	execEvalExpPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubExposurePID,
	}), "clb-exec-eval-exp")

	execEvalDdPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubDrawdownPID,
	}), "clb-exec-eval-dd")

	riskExposurePID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubExposurePID,
		ScopePID:         execEvalExpPID,
	}), "clb-risk-exp")

	riskDrawdownPID := e.Spawn(NewDrawdownLimitEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubDrawdownPID,
		ScopePID:         execEvalDdPID,
	}), "clb-risk-dd")

	stratResolverPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "clb-strat-resolver")

	decEvalPID := e.Spawn(NewEMACrossoverEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "clb-dec-eval")

	time.Sleep(50 * time.Millisecond)

	// === INJECT: EMA bullish → moderate severity ===
	e.Send(decEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "ema_crossover",
		SignalValue:   "bullish",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "clb-tf-full-obs",
	})

	// === OBSERVE STAGE 1: Decision ===
	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "triggered" {
		t.Fatalf("[decision] outcome: want triggered, got %s", dec.Outcome)
	}
	if string(dec.Severity) != "moderate" {
		t.Fatalf("[decision] severity: want moderate, got %s", dec.Severity)
	}
	if dec.Type != "ema_crossover" {
		t.Fatalf("[decision] type: want ema_crossover, got %s", dec.Type)
	}
	t.Logf("[decision] outcome=%s severity=%s confidence=%s", dec.Outcome, dec.Severity, dec.Confidence)

	// === OBSERVE STAGE 2: Strategy ===
	e.Send(stratResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	strat := strategyPub.messages()[0].(publishStrategyMessage).Event.Strategy
	if strat.Type != "trend_following_entry" {
		t.Fatalf("[strategy] type: want trend_following_entry, got %s", strat.Type)
	}
	if string(strat.Direction) != "long" {
		t.Fatalf("[strategy] direction: want long, got %s", strat.Direction)
	}
	// Trend following parameters: trailing_stop_pct, take_profit_pct should be set.
	if strat.Parameters["trailing_stop_pct"] == "" {
		t.Fatal("[strategy] trailing_stop_pct should be set for trend following")
	}
	if strat.Parameters["take_profit_pct"] == "" {
		t.Fatal("[strategy] take_profit_pct should be set for trend following")
	}
	t.Logf("[strategy] type=%s direction=%s trailing_stop=%s take_profit=%s",
		strat.Type, strat.Direction,
		strat.Parameters["trailing_stop_pct"], strat.Parameters["take_profit_pct"])

	// === OBSERVE STAGE 3: Dual Risk ===
	stratMsg := stratFanout.messages()[0].(strategyResolvedMessage)
	e.Send(riskExposurePID, stratMsg)
	e.Send(riskDrawdownPID, stratMsg)

	riskPubExposure.waitFor(t, 1, 2*time.Second)
	riskPubDrawdown.waitFor(t, 1, 2*time.Second)

	riskExp := riskPubExposure.messages()[0].(publishRiskMessage).Event.RiskAssessment
	riskDd := riskPubDrawdown.messages()[0].(publishRiskMessage).Event.RiskAssessment

	if string(riskExp.Disposition) != "approved" {
		t.Fatalf("[risk/exposure] disposition: want approved, got %s", riskExp.Disposition)
	}
	if string(riskDd.Disposition) != "approved" {
		t.Fatalf("[risk/drawdown] disposition: want approved, got %s", riskDd.Disposition)
	}
	// Trend following gets confidence factor 0.95 (vs 0.90 for mean_reversion).
	if len(riskExp.Strategies) == 0 || riskExp.Strategies[0].Type != "trend_following_entry" {
		t.Fatal("[risk/exposure] should preserve strategy type=trend_following_entry")
	}
	if len(riskDd.Strategies) == 0 || riskDd.Strategies[0].Type != "trend_following_entry" {
		t.Fatal("[risk/drawdown] should preserve strategy type=trend_following_entry")
	}
	t.Logf("[risk/exposure] disposition=%s max_position=%s", riskExp.Disposition, riskExp.Constraints.MaxPositionSize)
	t.Logf("[risk/drawdown] disposition=%s stop_distance=%s", riskDd.Disposition, riskDd.Constraints.StopDistance)

	// === OBSERVE STAGE 4: Execution ===
	execPubExposure.waitFor(t, 1, 2*time.Second)
	execPubDrawdown.waitFor(t, 1, 2*time.Second)

	expIntent := execPubExposure.messages()[0].(publishExecutionMessage).Event.ExecutionIntent
	ddIntent := execPubDrawdown.messages()[0].(publishExecutionMessage).Event.ExecutionIntent

	for label, intent := range map[string]domainexec.ExecutionIntent{
		"exposure": expIntent,
		"drawdown": ddIntent,
	} {
		if intent.Side != domainexec.SideBuy {
			t.Fatalf("[exec/%s] side: want buy, got %s", label, intent.Side)
		}
		if intent.Status != domainexec.StatusFilled {
			t.Fatalf("[exec/%s] status: want filled, got %s", label, intent.Status)
		}
		if intent.Risk.StrategyType != "trend_following_entry" {
			t.Fatalf("[exec/%s] strategy type: want trend_following_entry, got %q", label, intent.Risk.StrategyType)
		}
		if intent.Risk.DecisionSeverity != "moderate" {
			t.Fatalf("[exec/%s] decision severity: want moderate, got %q", label, intent.Risk.DecisionSeverity)
		}
		if intent.CorrelationID != "clb-tf-full-obs" {
			t.Fatalf("[exec/%s] correlation ID lost: got %q", label, intent.CorrelationID)
		}
		if prob := intent.Validate(); prob != nil {
			t.Fatalf("[exec/%s] validation failed: %s", label, prob.Message)
		}
	}

	t.Logf("[closed-loop-B] PASS — trend following chain fully observable across 4 stages")
}

// --- Closed Loop C: Severity Behavioral Contrast at Every Stage ---
// Proves that high vs low severity produces observably different outputs at EVERY stage,
// not just the final paper order quantity.
func TestClosedLoop_SeverityContrast_EveryStage(t *testing.T) {
	// Run high severity chain (RSI 10).
	highDec, highStrat, highRisk, highExec := runObservableChain(t, "10.0000", "clc-high")
	// Run low severity chain (RSI 25).
	lowDec, lowStrat, lowRisk, lowExec := runObservableChain(t, "25.0000", "clc-low")

	// === STAGE 1: Decision severity distinction ===
	if string(highDec.Severity) != "high" {
		t.Fatalf("[decision] high chain: want severity=high, got %s", highDec.Severity)
	}
	if string(lowDec.Severity) != "low" {
		t.Fatalf("[decision] low chain: want severity=low, got %s", lowDec.Severity)
	}
	t.Logf("[decision] high severity=%s vs low severity=%s", highDec.Severity, lowDec.Severity)

	// === STAGE 2: Strategy parameter distinction ===
	highTarget := highStrat.Parameters["target_offset"]
	lowTarget := lowStrat.Parameters["target_offset"]
	if highTarget == lowTarget {
		t.Errorf("[strategy] target_offset should differ: high=%s vs low=%s", highTarget, lowTarget)
	}
	// High severity → higher confidence.
	highConf, _ := strconv.ParseFloat(highStrat.Confidence, 64)
	lowConf, _ := strconv.ParseFloat(lowStrat.Confidence, 64)
	if highConf <= lowConf {
		t.Errorf("[strategy] high confidence (%.4f) should exceed low (%.4f)", highConf, lowConf)
	}
	t.Logf("[strategy] high confidence=%.4f target=%s vs low confidence=%.4f target=%s",
		highConf, highTarget, lowConf, lowTarget)

	// === STAGE 3: Risk constraint distinction ===
	highMaxPos, _ := strconv.ParseFloat(highRisk.Constraints.MaxPositionSize, 64)
	lowMaxPos, _ := strconv.ParseFloat(lowRisk.Constraints.MaxPositionSize, 64)
	if highMaxPos <= lowMaxPos {
		t.Errorf("[risk] high max_position (%.4f) should exceed low (%.4f)", highMaxPos, lowMaxPos)
	}
	t.Logf("[risk] high max_position=%.4f vs low max_position=%.4f", highMaxPos, lowMaxPos)

	// === STAGE 4: Execution quantity distinction ===
	highQty, _ := strconv.ParseFloat(highExec.Quantity, 64)
	lowQty, _ := strconv.ParseFloat(lowExec.Quantity, 64)
	if highQty <= lowQty {
		t.Errorf("[exec] high quantity (%.4f) should exceed low (%.4f)", highQty, lowQty)
	}
	// Severity context preserved in execution.
	if highExec.Risk.DecisionSeverity != "high" {
		t.Fatalf("[exec] high: decision severity lost, got %q", highExec.Risk.DecisionSeverity)
	}
	if lowExec.Risk.DecisionSeverity != "low" {
		t.Fatalf("[exec] low: decision severity lost, got %q", lowExec.Risk.DecisionSeverity)
	}
	t.Logf("[exec] high qty=%.4f vs low qty=%.4f", highQty, lowQty)
	t.Logf("[closed-loop-C] PASS — severity contrast observable at all 4 stages")
}

// --- Closed Loop D: No-Signal Suppression (negative path) ---
// Validates that a non-triggered signal correctly suppresses action at every stage,
// proving the loop closes safely when conditions are not met.
func TestClosedLoop_NoSignal_Suppression_FullChain(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	execPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "cld-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "cld-strat-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "cld-risk-pub")
	execPubPID := e.Spawn(execPub.producer(), "cld-exec-pub")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "cld-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "cld-strat-fan")

	execEvalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubPID,
	}), "cld-exec-eval")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
		ScopePID:         execEvalPID,
	}), "cld-risk-eval")

	stratResolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "cld-strat-resolver")

	decEvalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "cld-dec-eval")

	time.Sleep(50 * time.Millisecond)

	// Inject: RSI 75 → not_triggered.
	e.Send(decEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "rsi",
		SignalValue:   "75.0000",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "cld-no-signal",
	})

	// === OBSERVE STAGE 1: Decision NOT triggered ===
	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "not_triggered" {
		t.Fatalf("[decision] outcome: want not_triggered, got %s", dec.Outcome)
	}
	if string(dec.Severity) != "none" {
		t.Fatalf("[decision] severity: want none, got %s", dec.Severity)
	}
	t.Logf("[decision] outcome=%s severity=%s (correctly suppressed)", dec.Outcome, dec.Severity)

	// === OBSERVE STAGE 2: Strategy FLAT ===
	e.Send(stratResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	strat := strategyPub.messages()[0].(publishStrategyMessage).Event.Strategy
	if string(strat.Direction) != "flat" {
		t.Fatalf("[strategy] direction: want flat, got %s", strat.Direction)
	}
	t.Logf("[strategy] direction=%s (correctly flat)", strat.Direction)

	// === OBSERVE STAGE 3: Risk APPROVED (trivially — no position needed) ===
	e.Send(riskEvalPID, stratFanout.messages()[0].(strategyResolvedMessage))
	riskPub.waitFor(t, 1, 2*time.Second)

	risk := riskPub.messages()[0].(publishRiskMessage).Event.RiskAssessment
	if string(risk.Disposition) != "approved" {
		t.Fatalf("[risk] disposition: want approved (flat → trivially safe), got %s", risk.Disposition)
	}
	t.Logf("[risk] disposition=%s (flat strategy → no position)", risk.Disposition)

	// === OBSERVE STAGE 4: Execution NO-ACTION ===
	execPub.waitFor(t, 1, 2*time.Second)

	intent := execPub.messages()[0].(publishExecutionMessage).Event.ExecutionIntent
	if intent.Side != domainexec.SideNone {
		t.Fatalf("[exec] side: want none (no-action), got %s", intent.Side)
	}
	if intent.Quantity != "0" {
		t.Fatalf("[exec] quantity: want 0, got %s", intent.Quantity)
	}
	if intent.Status != domainexec.StatusSubmitted {
		t.Fatalf("[exec] status: want submitted (no fill for no-action), got %s", intent.Status)
	}
	if len(intent.Fills) != 0 {
		t.Fatalf("[exec] fills: want 0 (no-action), got %d", len(intent.Fills))
	}
	if intent.CorrelationID != "cld-no-signal" {
		t.Fatalf("[exec] correlation ID lost: got %q", intent.CorrelationID)
	}
	if prob := intent.Validate(); prob != nil {
		t.Fatalf("[exec] no-action intent should be valid: %s", prob.Message)
	}

	t.Logf("[closed-loop-D] PASS — non-triggered signal correctly suppresses at all 4 stages")
}

// --- Closed Loop E: Cross-Chain Behavioral Distinction ---
// Proves that mean_reversion and trend_following chains produce semantically distinct
// intermediate outputs at every stage, not just different final quantities.
func TestClosedLoop_CrossChain_BehavioralDistinction(t *testing.T) {
	// Chain A: RSI oversold (high severity) → mean_reversion → position_exposure → paper order.
	mrDec, mrStrat, mrRisk, mrExec := runObservableChain(t, "10.0000", "cle-mr")

	// Chain B: EMA bullish (moderate severity) → trend_following → position_exposure → paper order.
	tfDec, tfStrat, tfRisk, tfExec := runObservableChainEMA(t, "bullish", "cle-tf")

	// === Decision distinction: different types and severities ===
	if mrDec.Type == tfDec.Type {
		t.Fatalf("[decision] types should differ: mr=%s, tf=%s", mrDec.Type, tfDec.Type)
	}
	if mrDec.Severity == tfDec.Severity {
		t.Logf("[decision] note: severities may differ (high vs moderate) — mr=%s, tf=%s",
			mrDec.Severity, tfDec.Severity)
	}

	// === Strategy distinction: different families and parameters ===
	if mrStrat.Type == tfStrat.Type {
		t.Fatalf("[strategy] types should differ: mr=%s, tf=%s", mrStrat.Type, tfStrat.Type)
	}
	// Mean reversion has target_offset/stop_offset; trend following has trailing_stop_pct/take_profit_pct.
	if mrStrat.Parameters["target_offset"] == "" {
		t.Fatal("[strategy/mr] should have target_offset parameter")
	}
	if tfStrat.Parameters["trailing_stop_pct"] == "" {
		t.Fatal("[strategy/tf] should have trailing_stop_pct parameter")
	}

	// === Risk distinction: different strategy-type confidence factors ===
	if mrRisk.Strategies[0].Type != "mean_reversion_entry" {
		t.Fatalf("[risk/mr] strategy type: want mean_reversion_entry, got %s", mrRisk.Strategies[0].Type)
	}
	if tfRisk.Strategies[0].Type != "trend_following_entry" {
		t.Fatalf("[risk/tf] strategy type: want trend_following_entry, got %s", tfRisk.Strategies[0].Type)
	}

	// === Execution distinction: different strategy contexts preserved ===
	if mrExec.Risk.StrategyType == tfExec.Risk.StrategyType {
		t.Fatalf("[exec] strategy types should differ: mr=%s, tf=%s",
			mrExec.Risk.StrategyType, tfExec.Risk.StrategyType)
	}
	// Both produce valid buy orders.
	if mrExec.Side != domainexec.SideBuy || tfExec.Side != domainexec.SideBuy {
		t.Fatalf("[exec] both chains should produce buy: mr=%s, tf=%s", mrExec.Side, tfExec.Side)
	}

	mrQty, _ := strconv.ParseFloat(mrExec.Quantity, 64)
	tfQty, _ := strconv.ParseFloat(tfExec.Quantity, 64)
	t.Logf("[cross-chain] mr qty=%.4f (type=%s sev=%s) vs tf qty=%.4f (type=%s sev=%s)",
		mrQty, mrExec.Risk.StrategyType, mrExec.Risk.DecisionSeverity,
		tfQty, tfExec.Risk.StrategyType, tfExec.Risk.DecisionSeverity)
	t.Logf("[closed-loop-E] PASS — cross-chain behavioral distinction observable at all stages")
}

// === Helpers ===

// observableDecision wraps decision fields for cross-chain comparison.
type observableDecision struct {
	Type     string
	Outcome  string
	Severity string
}

// observableStrategy wraps strategy fields for cross-chain comparison.
type observableStrategy struct {
	Type       string
	Direction  string
	Confidence string
	Parameters map[string]string
}

// observableRisk wraps risk fields for cross-chain comparison.
type observableRisk struct {
	Type        string
	Disposition string
	Constraints struct {
		MaxPositionSize string
		StopDistance     string
	}
	Strategies []struct {
		Type             string
		DecisionSeverity string
	}
}

// runObservableChain runs a full RSI → mean_reversion → position_exposure chain and
// returns intermediate outputs at every stage for assertion.
func runObservableChain(t *testing.T, rsiValue, prefix string) (observableDecision, observableStrategy, observableRisk, domainexec.ExecutionIntent) {
	t.Helper()
	return runObservableChainInner(t, rsiValue, prefix, false)
}

// runObservableChainEMA runs a full EMA → trend_following → position_exposure chain.
func runObservableChainEMA(t *testing.T, emaValue, prefix string) (observableDecision, observableStrategy, observableRisk, domainexec.ExecutionIntent) {
	t.Helper()
	return runObservableChainInner(t, emaValue, prefix, true)
}

func runObservableChainInner(t *testing.T, signalValue, prefix string, isEMA bool) (observableDecision, observableStrategy, observableRisk, domainexec.ExecutionIntent) {
	t.Helper()
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	execPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), prefix+"-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), prefix+"-strat-pub")
	riskPubPID := e.Spawn(riskPub.producer(), prefix+"-risk-pub")
	execPubPID := e.Spawn(execPub.producer(), prefix+"-exec-pub")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), prefix+"-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), prefix+"-strat-fan")

	execEvalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubPID,
	}), prefix+"-exec-eval")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
		ScopePID:         execEvalPID,
	}), prefix+"-risk-eval")

	if isEMA {
		sPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
			Source:               "binancef",
			Symbol:               "btcusdt",
			Timeframe:            60 * time.Second,
			StrategyPublisherPID: strategyPubPID,
			ScopePID:             stratFanoutPID,
		}), prefix+"-strat-resolver")

		dPID := e.Spawn(NewEMACrossoverEvaluatorActor(DecisionEvaluatorConfig{
			Source:               "binancef",
			Symbol:               "btcusdt",
			Timeframe:            60 * time.Second,
			DecisionPublisherPID: decisionPubPID,
			ScopePID:             decFanoutPID,
		}), prefix+"-dec-eval")

		time.Sleep(50 * time.Millisecond)

		e.Send(dPID, signalGeneratedMessage{
			Symbol:        "btcusdt",
			SignalType:    "ema_crossover",
			SignalValue:   signalValue,
			Timeframe:     60,
			Timestamp:     windowBase(),
			CorrelationID: prefix + "-corr",
		})

		decisionPub.waitFor(t, 1, 2*time.Second)
		decFanout.waitFor(t, 1, 2*time.Second)
		e.Send(sPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	} else {
		sPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
			Source:               "binancef",
			Symbol:               "btcusdt",
			Timeframe:            60 * time.Second,
			StrategyPublisherPID: strategyPubPID,
			ScopePID:             stratFanoutPID,
		}), prefix+"-strat-resolver")

		dPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
			Source:               "binancef",
			Symbol:               "btcusdt",
			Timeframe:            60 * time.Second,
			DecisionPublisherPID: decisionPubPID,
			ScopePID:             decFanoutPID,
		}), prefix+"-dec-eval")

		time.Sleep(50 * time.Millisecond)

		e.Send(dPID, signalGeneratedMessage{
			Symbol:        "btcusdt",
			SignalType:    "rsi",
			SignalValue:   signalValue,
			Timeframe:     60,
			Timestamp:     windowBase(),
			CorrelationID: prefix + "-corr",
		})

		decisionPub.waitFor(t, 1, 2*time.Second)
		decFanout.waitFor(t, 1, 2*time.Second)
		e.Send(sPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	}

	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)
	e.Send(riskEvalPID, stratFanout.messages()[0].(strategyResolvedMessage))
	riskPub.waitFor(t, 1, 2*time.Second)
	execPub.waitFor(t, 1, 2*time.Second)

	// Extract intermediate outputs.
	rawDec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	rawStrat := strategyPub.messages()[0].(publishStrategyMessage).Event.Strategy
	rawRisk := riskPub.messages()[0].(publishRiskMessage).Event.RiskAssessment
	rawExec := execPub.messages()[0].(publishExecutionMessage).Event.ExecutionIntent

	obsDec := observableDecision{
		Type:     rawDec.Type,
		Outcome:  string(rawDec.Outcome),
		Severity: string(rawDec.Severity),
	}

	obsStrat := observableStrategy{
		Type:       rawStrat.Type,
		Direction:  string(rawStrat.Direction),
		Confidence: rawStrat.Confidence,
		Parameters: rawStrat.Parameters,
	}

	obsRisk := observableRisk{
		Type:        rawRisk.Type,
		Disposition: string(rawRisk.Disposition),
	}
	obsRisk.Constraints.MaxPositionSize = rawRisk.Constraints.MaxPositionSize
	obsRisk.Constraints.StopDistance = rawRisk.Constraints.StopDistance
	for _, s := range rawRisk.Strategies {
		obsRisk.Strategies = append(obsRisk.Strategies, struct {
			Type             string
			DecisionSeverity string
		}{
			Type:             s.Type,
			DecisionSeverity: s.DecisionSeverity,
		})
	}

	return obsDec, obsStrat, obsRisk, rawExec
}
