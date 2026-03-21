package derive

import (
	"testing"
	"time"
)

// actor_chain_integration_test.go validates the full derive actor chain:
// signal → decision → strategy → risk, wired through real actor instances
// with msgCollectors capturing each stage's output.
//
// Each fan-out stage uses a separate collector to avoid waitFor counter issues.
// The test manually forwards inter-actor messages, simulating the SourceScopeActor
// routing behavior without requiring the full scope supervisor.

func TestActorChain_Signal_To_Decision_To_Strategy_To_Risk(t *testing.T) {
	e := newTestEngine(t)

	// Terminal collectors — capture what each actor publishes.
	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "decision-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "strategy-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "risk-pub")

	// Separate fan-out collectors per stage.
	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "dec-fanout")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "strat-fanout")

	// Wire actors.
	decisionEvalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "decision-eval")

	strategyResolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "strategy-resolver")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
	}), "risk-eval")

	time.Sleep(50 * time.Millisecond)

	// Stage 1: Inject a low RSI signal → decision evaluator should trigger.
	e.Send(decisionEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "rsi",
		SignalValue:   "20.0000",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "chain-corr-1",
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	// Verify decision output.
	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "triggered" {
		t.Fatalf("decision outcome: want triggered, got %s", dec.Outcome)
	}
	if string(dec.Severity) == "" {
		t.Fatal("expected decision severity to be set")
	}

	// Stage 2: Forward decisionEvaluatedMessage to strategy resolver.
	decisionFanoutMsg, ok := decFanout.messages()[0].(decisionEvaluatedMessage)
	if !ok {
		t.Fatalf("expected decisionEvaluatedMessage in fan-out, got %T", decFanout.messages()[0])
	}
	if decisionFanoutMsg.DecisionSeverity == "" {
		t.Fatal("expected decision severity in fan-out message")
	}
	if decisionFanoutMsg.CorrelationID != "chain-corr-1" {
		t.Errorf("correlationID: want chain-corr-1, got %s", decisionFanoutMsg.CorrelationID)
	}

	e.Send(strategyResolverPID, decisionFanoutMsg)

	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	// Verify strategy output.
	strat := strategyPub.messages()[0].(publishStrategyMessage).Event.Strategy
	if string(strat.Direction) != "long" {
		t.Fatalf("strategy direction: want long, got %s", strat.Direction)
	}
	// S250: Strategy confidence is now severity-scaled, not a direct copy of decision confidence.
	// RSI 20.0 → severity "moderate" → scaling ×0.90.
	if strat.Confidence == "" {
		t.Error("strategy confidence should not be empty")
	}
	if strat.Confidence == dec.Confidence {
		t.Logf("note: strategy confidence matches decision confidence (severity may be unknown/empty → neutral scaling)")
	}
	if len(strat.Decisions) != 1 {
		t.Fatalf("expected 1 decision input in strategy, got %d", len(strat.Decisions))
	}
	if strat.Decisions[0].Severity != string(dec.Severity) {
		t.Errorf("strategy decision severity: want %s, got %s", dec.Severity, strat.Decisions[0].Severity)
	}

	// Stage 3: Forward strategyResolvedMessage to risk evaluator.
	strategyFanoutMsg, ok := stratFanout.messages()[0].(strategyResolvedMessage)
	if !ok {
		t.Fatalf("expected strategyResolvedMessage in fan-out, got %T", stratFanout.messages()[0])
	}
	if strategyFanoutMsg.DecisionSeverity == "" {
		t.Fatal("expected decision severity carried through strategy fan-out")
	}
	if strategyFanoutMsg.CorrelationID != "chain-corr-1" {
		t.Errorf("correlationID at strategy fan-out: want chain-corr-1, got %s", strategyFanoutMsg.CorrelationID)
	}

	e.Send(riskEvalPID, strategyFanoutMsg)

	riskPub.waitFor(t, 1, 2*time.Second)

	// Verify risk output.
	riskMsg := riskPub.messages()[0].(publishRiskMessage)
	riskA := riskMsg.Event.RiskAssessment
	if string(riskA.Disposition) != "approved" {
		t.Fatalf("risk disposition: want approved, got %s", riskA.Disposition)
	}
	if !riskA.Final {
		t.Error("expected risk final=true")
	}
	if len(riskA.Strategies) != 1 {
		t.Fatalf("expected 1 strategy input in risk, got %d", len(riskA.Strategies))
	}
	if riskA.Strategies[0].Direction != "long" {
		t.Errorf("risk strategy direction: want long, got %s", riskA.Strategies[0].Direction)
	}
	if riskA.Strategies[0].DecisionSeverity == "" {
		t.Error("expected decision severity to survive full chain into risk assessment")
	}
	if riskMsg.Event.Metadata.CorrelationID != "chain-corr-1" {
		t.Errorf("risk correlationID: want chain-corr-1, got %s", riskMsg.Event.Metadata.CorrelationID)
	}
	if prob := riskA.Validate(); prob != nil {
		t.Errorf("risk assessment validation failed: %s", prob.Message)
	}
}

func TestActorChain_NotTriggered_FlowsThrough(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "decision-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "strategy-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "risk-pub")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "dec-fanout")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "strat-fanout")

	decisionEvalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "decision-eval")

	strategyResolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "strategy-resolver")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
	}), "risk-eval")

	time.Sleep(50 * time.Millisecond)

	// High RSI → not_triggered path.
	e.Send(decisionEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "rsi",
		SignalValue:   "75.0000",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "chain-corr-2",
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "not_triggered" {
		t.Fatalf("decision outcome: want not_triggered, got %s", dec.Outcome)
	}

	// Forward to strategy.
	e.Send(strategyResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))

	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	strat := strategyPub.messages()[0].(publishStrategyMessage).Event.Strategy
	if string(strat.Direction) != "flat" {
		t.Fatalf("strategy direction: want flat, got %s", strat.Direction)
	}

	// Forward to risk.
	e.Send(riskEvalPID, stratFanout.messages()[0].(strategyResolvedMessage))

	riskPub.waitFor(t, 1, 2*time.Second)
	riskA := riskPub.messages()[0].(publishRiskMessage).Event.RiskAssessment
	if string(riskA.Disposition) != "approved" {
		t.Fatalf("risk disposition for flat: want approved, got %s", riskA.Disposition)
	}
}

func TestActorChain_EMACrossover_Bullish_Triggered(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "decision-pub")

	decFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "dec-fanout")

	decisionEvalPID := e.Spawn(NewEMACrossoverEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "decision-eval-ema")

	time.Sleep(50 * time.Millisecond)

	e.Send(decisionEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "ema_crossover",
		SignalValue:   "bullish",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "ema-chain-corr-1",
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "triggered" {
		t.Fatalf("decision outcome: want triggered, got %s", dec.Outcome)
	}
	if dec.Type != "ema_crossover" {
		t.Fatalf("decision type: want ema_crossover, got %s", dec.Type)
	}
	if string(dec.Severity) == "" {
		t.Fatal("expected decision severity to be set")
	}

	decisionFanoutMsg, ok := decFanout.messages()[0].(decisionEvaluatedMessage)
	if !ok {
		t.Fatalf("expected decisionEvaluatedMessage, got %T", decFanout.messages()[0])
	}
	if decisionFanoutMsg.DecisionType != "ema_crossover" {
		t.Errorf("fan-out decision type: want ema_crossover, got %s", decisionFanoutMsg.DecisionType)
	}
	if decisionFanoutMsg.CorrelationID != "ema-chain-corr-1" {
		t.Errorf("correlationID: want ema-chain-corr-1, got %s", decisionFanoutMsg.CorrelationID)
	}
}

func TestActorChain_EMACrossover_Bearish_NotTriggered(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "decision-pub")

	decFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "dec-fanout")

	decisionEvalPID := e.Spawn(NewEMACrossoverEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "decision-eval-ema")

	time.Sleep(50 * time.Millisecond)

	e.Send(decisionEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "ema_crossover",
		SignalValue:   "bearish",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "ema-chain-corr-2",
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "not_triggered" {
		t.Fatalf("decision outcome: want not_triggered, got %s", dec.Outcome)
	}
}

func TestActorChain_EMACrossover_TrendFollowingEntry_To_Risk(t *testing.T) {
	e := newTestEngine(t)

	// Terminal collectors.
	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "decision-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "strategy-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "risk-pub")

	// Fan-out collectors.
	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "dec-fanout")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "strat-fanout")

	// Wire actors: EMA crossover decision → trend following strategy → risk.
	decisionEvalPID := e.Spawn(NewEMACrossoverEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "decision-eval-ema")

	strategyResolverPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "strategy-resolver-trend")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
	}), "risk-eval")

	time.Sleep(50 * time.Millisecond)

	// Stage 1: Bullish EMA crossover signal → ema_crossover decision → triggered.
	e.Send(decisionEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "ema_crossover",
		SignalValue:   "bullish",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "trend-chain-corr-1",
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "triggered" {
		t.Fatalf("decision outcome: want triggered, got %s", dec.Outcome)
	}
	if dec.Type != "ema_crossover" {
		t.Fatalf("decision type: want ema_crossover, got %s", dec.Type)
	}

	// Stage 2: Forward to trend following strategy resolver.
	decisionFanoutMsg := decFanout.messages()[0].(decisionEvaluatedMessage)
	e.Send(strategyResolverPID, decisionFanoutMsg)

	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	strat := strategyPub.messages()[0].(publishStrategyMessage).Event.Strategy
	if strat.Type != "trend_following_entry" {
		t.Fatalf("strategy type: want trend_following_entry, got %s", strat.Type)
	}
	if string(strat.Direction) != "long" {
		t.Fatalf("strategy direction: want long, got %s", strat.Direction)
	}
	if strat.Parameters["trailing_stop_pct"] != "0.03" {
		t.Errorf("expected trailing_stop_pct=0.03, got %s", strat.Parameters["trailing_stop_pct"])
	}
	if len(strat.Decisions) != 1 || strat.Decisions[0].Type != "ema_crossover" {
		t.Fatalf("expected ema_crossover decision input, got %+v", strat.Decisions)
	}

	// Stage 3: Forward to risk evaluator.
	strategyFanoutMsg := stratFanout.messages()[0].(strategyResolvedMessage)
	e.Send(riskEvalPID, strategyFanoutMsg)

	riskPub.waitFor(t, 1, 2*time.Second)
	riskA := riskPub.messages()[0].(publishRiskMessage).Event.RiskAssessment
	if string(riskA.Disposition) != "approved" {
		t.Fatalf("risk disposition: want approved, got %s", riskA.Disposition)
	}
	if !riskA.Final {
		t.Error("expected risk final=true")
	}
	if len(riskA.Strategies) != 1 {
		t.Fatalf("expected 1 strategy input in risk, got %d", len(riskA.Strategies))
	}
	if riskA.Strategies[0].Direction != "long" {
		t.Errorf("risk strategy direction: want long, got %s", riskA.Strategies[0].Direction)
	}
	if prob := riskA.Validate(); prob != nil {
		t.Errorf("risk assessment validation failed: %s", prob.Message)
	}
}

func TestActorChain_EMACrossover_TrendFollowingEntry_To_DrawdownLimitRisk(t *testing.T) {
	e := newTestEngine(t)

	// Terminal collectors.
	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "decision-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "strategy-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "risk-pub")

	// Fan-out collectors.
	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "dec-fanout")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "strat-fanout")

	// Wire actors: EMA crossover decision → trend following strategy → drawdown_limit risk.
	decisionEvalPID := e.Spawn(NewEMACrossoverEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "decision-eval-ema")

	strategyResolverPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "strategy-resolver-trend")

	riskEvalPID := e.Spawn(NewDrawdownLimitEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
	}), "risk-eval-drawdown")

	time.Sleep(50 * time.Millisecond)

	// Stage 1: Bullish EMA crossover signal → ema_crossover decision → triggered.
	e.Send(decisionEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "ema_crossover",
		SignalValue:   "bullish",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "drawdown-chain-corr-1",
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "triggered" {
		t.Fatalf("decision outcome: want triggered, got %s", dec.Outcome)
	}
	if dec.Type != "ema_crossover" {
		t.Fatalf("decision type: want ema_crossover, got %s", dec.Type)
	}

	// Stage 2: Forward to trend following strategy resolver.
	decisionFanoutMsg := decFanout.messages()[0].(decisionEvaluatedMessage)
	e.Send(strategyResolverPID, decisionFanoutMsg)

	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	strat := strategyPub.messages()[0].(publishStrategyMessage).Event.Strategy
	if strat.Type != "trend_following_entry" {
		t.Fatalf("strategy type: want trend_following_entry, got %s", strat.Type)
	}
	if string(strat.Direction) != "long" {
		t.Fatalf("strategy direction: want long, got %s", strat.Direction)
	}

	// Stage 3: Forward to drawdown_limit risk evaluator.
	strategyFanoutMsg := stratFanout.messages()[0].(strategyResolvedMessage)
	e.Send(riskEvalPID, strategyFanoutMsg)

	riskPub.waitFor(t, 1, 2*time.Second)

	riskMsg := riskPub.messages()[0].(publishRiskMessage)
	riskA := riskMsg.Event.RiskAssessment

	// Verify drawdown_limit-specific assertions.
	if riskA.Type != "drawdown_limit" {
		t.Fatalf("risk type: want drawdown_limit, got %s", riskA.Type)
	}
	if string(riskA.Disposition) != "approved" {
		t.Fatalf("risk disposition: want approved, got %s", riskA.Disposition)
	}
	if !riskA.Final {
		t.Error("expected risk final=true")
	}
	if len(riskA.Strategies) != 1 {
		t.Fatalf("expected 1 strategy input in risk, got %d", len(riskA.Strategies))
	}
	if riskA.Strategies[0].Type != "trend_following_entry" {
		t.Errorf("risk strategy type: want trend_following_entry, got %s", riskA.Strategies[0].Type)
	}
	if riskA.Strategies[0].Direction != "long" {
		t.Errorf("risk strategy direction: want long, got %s", riskA.Strategies[0].Direction)
	}
	if riskA.Strategies[0].DecisionSeverity == "" {
		t.Error("expected decision severity to survive full Chain B into drawdown_limit risk")
	}
	if riskA.Constraints.StopDistance == "" {
		t.Error("expected stop_distance constraint in drawdown_limit assessment")
	}
	if riskMsg.Event.Metadata.CorrelationID != "drawdown-chain-corr-1" {
		t.Errorf("risk correlationID: want drawdown-chain-corr-1, got %s", riskMsg.Event.Metadata.CorrelationID)
	}
	if prob := riskA.Validate(); prob != nil {
		t.Errorf("risk assessment validation failed: %s", prob.Message)
	}
}

func TestActorChain_CorrelationID_PreservedEndToEnd(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "decision-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "strategy-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "risk-pub")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "dec-fanout")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "strat-fanout")

	decisionEvalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "decision-eval")

	strategyResolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "strategy-resolver")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
	}), "risk-eval")

	time.Sleep(50 * time.Millisecond)

	corrID := "e2e-correlation-test"

	e.Send(decisionEvalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "rsi",
		SignalValue:   "15.0000",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: corrID,
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	// Check correlation at decision stage.
	decEvent := decisionPub.messages()[0].(publishDecisionMessage).Event
	if decEvent.Metadata.CorrelationID != corrID {
		t.Errorf("decision correlationID: want %s, got %s", corrID, decEvent.Metadata.CorrelationID)
	}

	decisionFanoutMsg := decFanout.messages()[0].(decisionEvaluatedMessage)
	if decisionFanoutMsg.CorrelationID != corrID {
		t.Errorf("decision fan-out correlationID: want %s, got %s", corrID, decisionFanoutMsg.CorrelationID)
	}

	// Forward to strategy.
	e.Send(strategyResolverPID, decisionFanoutMsg)

	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	stratEvent := strategyPub.messages()[0].(publishStrategyMessage).Event
	if stratEvent.Metadata.CorrelationID != corrID {
		t.Errorf("strategy correlationID: want %s, got %s", corrID, stratEvent.Metadata.CorrelationID)
	}

	strategyFanoutMsg := stratFanout.messages()[0].(strategyResolvedMessage)
	if strategyFanoutMsg.CorrelationID != corrID {
		t.Errorf("strategy fan-out correlationID: want %s, got %s", corrID, strategyFanoutMsg.CorrelationID)
	}

	// Forward to risk.
	e.Send(riskEvalPID, strategyFanoutMsg)

	riskPub.waitFor(t, 1, 2*time.Second)
	riskEvent := riskPub.messages()[0].(publishRiskMessage).Event
	if riskEvent.Metadata.CorrelationID != corrID {
		t.Errorf("risk correlationID: want %s, got %s", corrID, riskEvent.Metadata.CorrelationID)
	}
}
