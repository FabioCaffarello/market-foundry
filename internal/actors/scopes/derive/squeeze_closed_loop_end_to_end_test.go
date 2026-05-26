package derive

import (
	"strconv"
	"testing"
	"time"

	domainexec "internal/domain/execution"
)

// squeeze_closed_loop_end_to_end_test.go validates the full closed-loop scenario
// for the squeeze breakout vertical slice (S291).
//
// This test exercises the complete path:
//   bollinger signal → bollinger_squeeze decision → squeeze_breakout_entry strategy
//   → risk (position_exposure + drawdown_limit) → paper_order execution
//
// Key distinction from bollinger_chain_integration_test.go (signal→decision only)
// and squeeze_breakout_entry_resolver_actor_test.go (decision→strategy only):
// this test proves the ENTIRE slice operates as one coherent vertical slice,
// from raw candle data through to paper order fills.

// --- Scenario S291-1: Full Closed Loop — Bollinger Squeeze Triggered → Paper Buy ---
// Feeds 20 tight-range candles to trigger a Bollinger squeeze, then validates every
// intermediate stage: signal, decision, strategy, dual risk, and dual execution.
func TestSqueezeClosedLoop_Triggered_FullObservability(t *testing.T) {
	e := newTestEngine(t)

	// Stage collectors.
	signalPub := newMsgCollector()
	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPubExposure := newMsgCollector()
	riskPubDrawdown := newMsgCollector()
	execPubExposure := newMsgCollector()
	execPubDrawdown := newMsgCollector()

	signalPubPID := e.Spawn(signalPub.producer(), "sq1-sig-pub")
	decisionPubPID := e.Spawn(decisionPub.producer(), "sq1-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "sq1-strat-pub")
	riskPubExposurePID := e.Spawn(riskPubExposure.producer(), "sq1-risk-pub-exp")
	riskPubDrawdownPID := e.Spawn(riskPubDrawdown.producer(), "sq1-risk-pub-dd")
	execPubExposurePID := e.Spawn(execPubExposure.producer(), "sq1-exec-pub-exp")
	execPubDrawdownPID := e.Spawn(execPubDrawdown.producer(), "sq1-exec-pub-dd")

	// Fan-out collectors (simulate SourceScopeActor routing).
	signalFanout := newMsgCollector()
	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	signalFanoutPID := e.Spawn(signalFanout.producer(), "sq1-sig-fan")
	decFanoutPID := e.Spawn(decFanout.producer(), "sq1-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "sq1-strat-fan")

	// Wire execution evaluators (one per risk type).
	execEvalExpPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubExposurePID,
	}), "sq1-exec-eval-exp")

	execEvalDdPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubDrawdownPID,
	}), "sq1-exec-eval-dd")

	// Wire risk evaluators with ScopePID → execution evaluators.
	riskExposurePID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubExposurePID,
		ScopePID:         execEvalExpPID,
	}), "sq1-risk-exp")

	riskDrawdownPID := e.Spawn(NewDrawdownLimitEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubDrawdownPID,
		ScopePID:         execEvalDdPID,
	}), "sq1-risk-dd")

	// Wire strategy resolver.
	stratResolverPID := e.Spawn(NewSqueezeBreakoutEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "sq1-strat-resolver")

	// Wire decision evaluator.
	decEvalPID := e.Spawn(NewBollingerSqueezeEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "sq1-dec-eval")

	// Wire signal sampler.
	samplerPID := e.Spawn(NewBollingerSignalSamplerActor(SignalSamplerConfig{
		Source:             "binancef",
		Symbol:             "btcusdt",
		Timeframe:          60 * time.Second,
		SignalPublisherPID: signalPubPID,
		ScopePID:           signalFanoutPID,
	}), "sq1-boll-sampler")

	time.Sleep(50 * time.Millisecond)

	// === INJECT: 20 candles with tight range (100.00–100.19) to trigger squeeze ===
	base := windowBase()
	for i := 0; i < 20; i++ {
		price := 100.0 + float64(i)*0.01
		e.Send(samplerPID, candleFinalizedMessage{
			Symbol:        "btcusdt",
			ClosePrice:    formatPrice(price),
			Timestamp:     base.Add(time.Duration(i) * time.Minute),
			CorrelationID: "sq1-closed-loop",
		})
	}

	// === OBSERVE STAGE 1: Signal ===
	signalPub.waitFor(t, 1, 3*time.Second)
	signalFanout.waitFor(t, 1, 3*time.Second)

	sigMsg := signalPub.messages()[0].(publishSignalMessage)
	sig := sigMsg.Event.Signal
	if sig.Type != "bollinger" {
		t.Fatalf("[signal] type: want bollinger, got %s", sig.Type)
	}
	if sig.VenueSymbol() != "btcusdt" {
		t.Fatalf("[signal] symbol: want btcusdt, got %s", sig.VenueSymbol())
	}
	if sig.Value == "" {
		t.Fatal("[signal] expected non-empty %B value")
	}
	if sig.Metadata["bandwidth"] == "" {
		t.Fatal("[signal] expected bandwidth in metadata")
	}
	if sig.Metadata["sma"] == "" {
		t.Fatal("[signal] expected sma in metadata")
	}
	t.Logf("[signal] type=%s value=%s bandwidth=%s sma=%s",
		sig.Type, sig.Value, sig.Metadata["bandwidth"], sig.Metadata["sma"])

	// === FORWARD: Signal → Decision ===
	sigFanoutMsg := signalFanout.messages()[0].(signalGeneratedMessage)
	e.Send(decEvalPID, sigFanoutMsg)

	// === OBSERVE STAGE 2: Decision ===
	decisionPub.waitFor(t, 1, 3*time.Second)
	decFanout.waitFor(t, 1, 3*time.Second)

	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if dec.Type != "bollinger_squeeze" {
		t.Fatalf("[decision] type: want bollinger_squeeze, got %s", dec.Type)
	}
	if string(dec.Outcome) != "triggered" {
		t.Fatalf("[decision] outcome: want triggered (tight bands), got %s", dec.Outcome)
	}
	if string(dec.Severity) == "" || string(dec.Severity) == "none" {
		t.Fatalf("[decision] severity should be high/moderate/low for triggered squeeze, got %s", dec.Severity)
	}
	if dec.Confidence == "" {
		t.Fatal("[decision] confidence should be set")
	}
	if len(dec.Signals) == 0 {
		t.Fatal("[decision] should carry signal input context")
	}
	if dec.Rationale == "" {
		t.Fatal("[decision] rationale should be set for observability")
	}
	if dec.Metadata["squeeze_threshold"] == "" {
		t.Fatal("[decision] metadata should include squeeze_threshold")
	}
	if dec.Metadata["relative_bandwidth"] == "" {
		t.Fatal("[decision] metadata should include relative_bandwidth")
	}
	t.Logf("[decision] outcome=%s severity=%s confidence=%s rationale=%q rel_bw=%s",
		dec.Outcome, dec.Severity, dec.Confidence, dec.Rationale, dec.Metadata["relative_bandwidth"])

	// === FORWARD: Decision → Strategy ===
	decMsg := decFanout.messages()[0].(decisionEvaluatedMessage)
	e.Send(stratResolverPID, decMsg)

	// === OBSERVE STAGE 3: Strategy ===
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	strat := strategyPub.messages()[0].(publishStrategyMessage).Event.Strategy
	if strat.Type != "squeeze_breakout_entry" {
		t.Fatalf("[strategy] type: want squeeze_breakout_entry, got %s", strat.Type)
	}
	if string(strat.Direction) != "long" {
		t.Fatalf("[strategy] direction: want long (triggered squeeze), got %s", strat.Direction)
	}
	if strat.Parameters["entry"] != "market" {
		t.Fatalf("[strategy] entry: want market, got %s", strat.Parameters["entry"])
	}
	if strat.Parameters["breakout_target_pct"] == "" {
		t.Fatal("[strategy] breakout_target_pct should be set")
	}
	if strat.Parameters["breakout_stop_pct"] == "" {
		t.Fatal("[strategy] breakout_stop_pct should be set")
	}
	if len(strat.Decisions) == 0 {
		t.Fatal("[strategy] should carry decision input context")
	}
	if strat.Decisions[0].Type != "bollinger_squeeze" {
		t.Fatalf("[strategy] decision input type: want bollinger_squeeze, got %s", strat.Decisions[0].Type)
	}
	if strat.Decisions[0].Severity != string(dec.Severity) {
		t.Fatalf("[strategy] decision severity forwarded: want %s, got %s", dec.Severity, strat.Decisions[0].Severity)
	}
	t.Logf("[strategy] type=%s direction=%s confidence=%s target=%s stop=%s",
		strat.Type, strat.Direction, strat.Confidence,
		strat.Parameters["breakout_target_pct"], strat.Parameters["breakout_stop_pct"])

	// === FORWARD: Strategy → Dual Risk ===
	stratMsg := stratFanout.messages()[0].(strategyResolvedMessage)
	e.Send(riskExposurePID, stratMsg)
	e.Send(riskDrawdownPID, stratMsg)

	// === OBSERVE STAGE 4: Dual Risk ===
	riskPubExposure.waitFor(t, 1, 2*time.Second)
	riskPubDrawdown.waitFor(t, 1, 2*time.Second)

	riskExp := riskPubExposure.messages()[0].(publishRiskMessage).Event.RiskAssessment
	riskDd := riskPubDrawdown.messages()[0].(publishRiskMessage).Event.RiskAssessment

	// Position exposure: approved, squeeze strategy factor applied.
	if string(riskExp.Disposition) != "approved" {
		t.Fatalf("[risk/exposure] disposition: want approved, got %s", riskExp.Disposition)
	}
	if riskExp.Type != "position_exposure" {
		t.Fatalf("[risk/exposure] type: want position_exposure, got %s", riskExp.Type)
	}
	if riskExp.Constraints.MaxPositionSize == "" {
		t.Fatal("[risk/exposure] MaxPositionSize constraint should be set")
	}
	if len(riskExp.Strategies) == 0 || riskExp.Strategies[0].Type != "squeeze_breakout_entry" {
		t.Fatal("[risk/exposure] strategy input should reference squeeze_breakout_entry")
	}
	if riskExp.Strategies[0].DecisionSeverity != string(dec.Severity) {
		t.Fatalf("[risk/exposure] decision severity lost: want %s, got %s",
			dec.Severity, riskExp.Strategies[0].DecisionSeverity)
	}

	// Drawdown limit: approved, stop distance set.
	if string(riskDd.Disposition) != "approved" {
		t.Fatalf("[risk/drawdown] disposition: want approved, got %s", riskDd.Disposition)
	}
	if riskDd.Type != "drawdown_limit" {
		t.Fatalf("[risk/drawdown] type: want drawdown_limit, got %s", riskDd.Type)
	}
	if riskDd.Constraints.StopDistance == "" {
		t.Fatal("[risk/drawdown] StopDistance constraint should be set")
	}
	if len(riskDd.Strategies) == 0 || riskDd.Strategies[0].Type != "squeeze_breakout_entry" {
		t.Fatal("[risk/drawdown] strategy input should reference squeeze_breakout_entry")
	}

	t.Logf("[risk/exposure] disposition=%s confidence=%s max_position=%s",
		riskExp.Disposition, riskExp.Confidence, riskExp.Constraints.MaxPositionSize)
	t.Logf("[risk/drawdown] disposition=%s confidence=%s stop_distance=%s",
		riskDd.Disposition, riskDd.Confidence, riskDd.Constraints.StopDistance)

	// === OBSERVE STAGE 5: Execution (paper orders from both risk paths) ===
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
			t.Fatalf("[exec/%s] side: want buy (long strategy, approved risk), got %s", label, intent.Side)
		}
		if intent.Status != domainexec.StatusFilled {
			t.Fatalf("[exec/%s] status: want filled (paper fill simulation), got %s", label, intent.Status)
		}
		if len(intent.Fills) != 1 || !intent.Fills[0].Simulated {
			t.Fatalf("[exec/%s] should have exactly 1 simulated fill", label)
		}
		if intent.CorrelationID != "sq1-closed-loop" {
			t.Fatalf("[exec/%s] correlation ID lost: got %q", label, intent.CorrelationID)
		}
		if intent.Risk.StrategyType != "squeeze_breakout_entry" {
			t.Fatalf("[exec/%s] strategy type lost: got %q", label, intent.Risk.StrategyType)
		}
		if intent.Risk.DecisionSeverity != string(dec.Severity) {
			t.Fatalf("[exec/%s] decision severity lost: got %q", label, intent.Risk.DecisionSeverity)
		}
		if !intent.Final {
			t.Fatalf("[exec/%s] Final should be true", label)
		}
		if prob := intent.Validate(); prob != nil {
			t.Fatalf("[exec/%s] domain validation failed: %s", label, prob.Message)
		}

		qty, err := strconv.ParseFloat(intent.Quantity, 64)
		if err != nil || qty <= 0 {
			t.Fatalf("[exec/%s] expected positive quantity, got %q", label, intent.Quantity)
		}
	}

	t.Logf("[exec/exposure] side=%s qty=%s risk=%s sev=%s",
		expIntent.Side, expIntent.Quantity, expIntent.Risk.Type, expIntent.Risk.DecisionSeverity)
	t.Logf("[exec/drawdown] side=%s qty=%s risk=%s sev=%s",
		ddIntent.Side, ddIntent.Quantity, ddIntent.Risk.Type, ddIntent.Risk.DecisionSeverity)
	t.Logf("[closed-loop-squeeze] PASS — all 5 stages observable: signal→decision→strategy→risk→execution")
}

// --- Scenario S291-2: Wide Bands — Not Triggered → Flat Strategy → No Execution ---
// Validates that wide Bollinger bands (no squeeze) produce not_triggered decision,
// flat strategy, and no paper order execution — proving the suppression path.
func TestSqueezeClosedLoop_NotTriggered_Suppression(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	execPub := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "sq2-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "sq2-strat-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "sq2-risk-pub")
	execPubPID := e.Spawn(execPub.producer(), "sq2-exec-pub")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	decFanoutPID := e.Spawn(decFanout.producer(), "sq2-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "sq2-strat-fan")

	execEvalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubPID,
	}), "sq2-exec-eval")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
		ScopePID:         execEvalPID,
	}), "sq2-risk-eval")

	stratResolverPID := e.Spawn(NewSqueezeBreakoutEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "sq2-strat-resolver")

	decEvalPID := e.Spawn(NewBollingerSqueezeEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "sq2-dec-eval")

	time.Sleep(50 * time.Millisecond)

	// Inject a bollinger signal with wide bandwidth (no squeeze).
	e.Send(decEvalPID, signalGeneratedMessage{
		Symbol:      "btcusdt",
		SignalType:  "bollinger",
		SignalValue: "0.7500",
		SignalMetadata: map[string]string{
			"bandwidth": "50.0000",
			"sma":       "100.0000",
			"upper":     "125.0000",
			"lower":     "75.0000",
			"period":    "20",
			"k":         "2.0",
		},
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "sq2-wide-bands",
	})

	// Decision: not_triggered.
	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "not_triggered" {
		t.Fatalf("[decision] outcome: want not_triggered (wide bands), got %s", dec.Outcome)
	}
	t.Logf("[decision] outcome=%s severity=%s", dec.Outcome, dec.Severity)

	// Forward decision → strategy.
	e.Send(stratResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	// Strategy: flat direction (no entry).
	strat := strategyPub.messages()[0].(publishStrategyMessage).Event.Strategy
	if strat.Type != "squeeze_breakout_entry" {
		t.Fatalf("[strategy] type: want squeeze_breakout_entry, got %s", strat.Type)
	}
	if string(strat.Direction) != "flat" {
		t.Fatalf("[strategy] direction: want flat (not_triggered), got %s", strat.Direction)
	}
	t.Logf("[strategy] type=%s direction=%s confidence=%s", strat.Type, strat.Direction, strat.Confidence)

	// Forward strategy → risk.
	e.Send(riskEvalPID, stratFanout.messages()[0].(strategyResolvedMessage))
	riskPub.waitFor(t, 1, 2*time.Second)

	risk := riskPub.messages()[0].(publishRiskMessage).Event.RiskAssessment
	if string(risk.Disposition) != "approved" {
		t.Fatalf("[risk] disposition: want approved (flat is safe), got %s", risk.Disposition)
	}
	t.Logf("[risk] disposition=%s type=%s", risk.Disposition, risk.Type)

	// Execution: side=none (flat strategy → no paper order action).
	execPub.waitFor(t, 1, 2*time.Second)
	intent := execPub.messages()[0].(publishExecutionMessage).Event.ExecutionIntent
	if intent.Side != domainexec.SideNone {
		t.Fatalf("[exec] side: want none (flat strategy), got %s", intent.Side)
	}
	if intent.CorrelationID != "sq2-wide-bands" {
		t.Fatalf("[exec] correlation ID lost: got %q", intent.CorrelationID)
	}

	t.Logf("[closed-loop-squeeze] PASS — suppression path validated: not_triggered→flat→none")
}

// --- Scenario S291-3: Severity Contrast — High vs Low Squeeze → Different Parameters ---
// Validates that different squeeze severities produce observably different strategy parameters,
// risk constraints, and execution quantities through the same pipeline.
func TestSqueezeClosedLoop_SeverityContrast_HighVsLow(t *testing.T) {
	type stageResult struct {
		decisionSeverity   string
		strategyConfidence string
		targetPct          string
		stopPct            string
		riskMaxPosition    string
		execQuantity       string
	}

	runChain := func(t *testing.T, prefix string, signalMeta map[string]string, pctB string) stageResult {
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

		stratResolverPID := e.Spawn(NewSqueezeBreakoutEntryResolverActor(StrategyResolverConfig{
			Source:               "binancef",
			Symbol:               "btcusdt",
			Timeframe:            60 * time.Second,
			StrategyPublisherPID: strategyPubPID,
			ScopePID:             stratFanoutPID,
		}), prefix+"-strat-resolver")

		decEvalPID := e.Spawn(NewBollingerSqueezeEvaluatorActor(DecisionEvaluatorConfig{
			Source:               "binancef",
			Symbol:               "btcusdt",
			Timeframe:            60 * time.Second,
			DecisionPublisherPID: decisionPubPID,
			ScopePID:             decFanoutPID,
		}), prefix+"-dec-eval")

		time.Sleep(50 * time.Millisecond)

		e.Send(decEvalPID, signalGeneratedMessage{
			Symbol:         "btcusdt",
			SignalType:     "bollinger",
			SignalValue:    pctB,
			SignalMetadata: signalMeta,
			Timeframe:      60,
			Timestamp:      windowBase(),
			CorrelationID:  prefix + "-corr",
		})

		decisionPub.waitFor(t, 1, 2*time.Second)
		decFanout.waitFor(t, 1, 2*time.Second)

		dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
		if string(dec.Outcome) != "triggered" {
			t.Fatalf("[%s/decision] expected triggered, got %s", prefix, dec.Outcome)
		}

		e.Send(stratResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))
		strategyPub.waitFor(t, 1, 2*time.Second)
		stratFanout.waitFor(t, 1, 2*time.Second)

		strat := strategyPub.messages()[0].(publishStrategyMessage).Event.Strategy

		e.Send(riskEvalPID, stratFanout.messages()[0].(strategyResolvedMessage))
		riskPub.waitFor(t, 1, 2*time.Second)
		execPub.waitFor(t, 1, 2*time.Second)

		risk := riskPub.messages()[0].(publishRiskMessage).Event.RiskAssessment
		intent := execPub.messages()[0].(publishExecutionMessage).Event.ExecutionIntent

		return stageResult{
			decisionSeverity:   string(dec.Severity),
			strategyConfidence: strat.Confidence,
			targetPct:          strat.Parameters["breakout_target_pct"],
			stopPct:            strat.Parameters["breakout_stop_pct"],
			riskMaxPosition:    risk.Constraints.MaxPositionSize,
			execQuantity:       intent.Quantity,
		}
	}

	// High squeeze: very tight bands (bandwidth 1.0, sma 100 → relBW=0.01, threshold=0.10).
	highMeta := map[string]string{
		"bandwidth": "1.0000",
		"sma":       "100.0000",
		"upper":     "100.5000",
		"lower":     "99.5000",
		"period":    "20",
		"k":         "2.0",
	}
	high := runChain(t, "sq3-high", highMeta, "0.5000")

	// Low squeeze: mildly tight bands (bandwidth 8.0, sma 100 → relBW=0.08, threshold=0.10).
	lowMeta := map[string]string{
		"bandwidth": "8.0000",
		"sma":       "100.0000",
		"upper":     "104.0000",
		"lower":     "96.0000",
		"period":    "20",
		"k":         "2.0",
	}
	low := runChain(t, "sq3-low", lowMeta, "0.5000")

	// Verify severity distinction.
	if high.decisionSeverity == low.decisionSeverity {
		t.Fatalf("severity should differ: high=%s low=%s", high.decisionSeverity, low.decisionSeverity)
	}
	t.Logf("[severity] high=%s low=%s", high.decisionSeverity, low.decisionSeverity)

	// Verify strategy parameters differ.
	if high.targetPct == low.targetPct {
		t.Errorf("breakout_target_pct should differ by severity: high=%s low=%s",
			high.targetPct, low.targetPct)
	}
	t.Logf("[strategy] high target=%s stop=%s | low target=%s stop=%s",
		high.targetPct, high.stopPct, low.targetPct, low.stopPct)

	// Verify execution quantities differ (higher severity → larger position).
	highQty, _ := strconv.ParseFloat(high.execQuantity, 64)
	lowQty, _ := strconv.ParseFloat(low.execQuantity, 64)
	if highQty <= lowQty {
		t.Errorf("high severity should produce larger quantity: high=%.4f low=%.4f", highQty, lowQty)
	}
	t.Logf("[exec] high qty=%.4f low qty=%.4f", highQty, lowQty)
	t.Logf("[severity-contrast] PASS — high/low severities produce observably different outputs")
}

// --- Scenario S291-4: Context Preservation — Correlation ID and Causation Chain ---
// Validates that the correlation ID survives the full pipeline from bollinger signal to
// paper order, and that causation IDs are set at each stage.
func TestSqueezeClosedLoop_ContextPreservation(t *testing.T) {
	e := newTestEngine(t)

	signalPub := newMsgCollector()
	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	execPub := newMsgCollector()
	signalPubPID := e.Spawn(signalPub.producer(), "sq4-sig-pub")
	decisionPubPID := e.Spawn(decisionPub.producer(), "sq4-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "sq4-strat-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "sq4-risk-pub")
	execPubPID := e.Spawn(execPub.producer(), "sq4-exec-pub")

	signalFanout := newMsgCollector()
	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()
	signalFanoutPID := e.Spawn(signalFanout.producer(), "sq4-sig-fan")
	decFanoutPID := e.Spawn(decFanout.producer(), "sq4-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "sq4-strat-fan")

	execEvalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: execPubPID,
	}), "sq4-exec-eval")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: riskPubPID,
		ScopePID:         execEvalPID,
	}), "sq4-risk-eval")

	stratResolverPID := e.Spawn(NewSqueezeBreakoutEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: strategyPubPID,
		ScopePID:             stratFanoutPID,
	}), "sq4-strat-resolver")

	decEvalPID := e.Spawn(NewBollingerSqueezeEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "sq4-dec-eval")

	samplerPID := e.Spawn(NewBollingerSignalSamplerActor(SignalSamplerConfig{
		Source:             "binancef",
		Symbol:             "btcusdt",
		Timeframe:          60 * time.Second,
		SignalPublisherPID: signalPubPID,
		ScopePID:           signalFanoutPID,
	}), "sq4-boll-sampler")

	time.Sleep(50 * time.Millisecond)

	const traceID = "sq4-trace-preservation"

	// Feed tight-range candles.
	base := windowBase()
	for i := 0; i < 20; i++ {
		price := 100.0 + float64(i)*0.01
		e.Send(samplerPID, candleFinalizedMessage{
			Symbol:        "btcusdt",
			ClosePrice:    formatPrice(price),
			Timestamp:     base.Add(time.Duration(i) * time.Minute),
			CorrelationID: traceID,
		})
	}

	signalPub.waitFor(t, 1, 3*time.Second)
	signalFanout.waitFor(t, 1, 3*time.Second)

	// Verify signal correlation.
	sigEvent := signalPub.messages()[0].(publishSignalMessage).Event
	if sigEvent.Metadata.CorrelationID != traceID {
		t.Fatalf("[signal] correlation ID: want %s, got %s", traceID, sigEvent.Metadata.CorrelationID)
	}

	// Forward through chain.
	e.Send(decEvalPID, signalFanout.messages()[0].(signalGeneratedMessage))
	decisionPub.waitFor(t, 1, 3*time.Second)
	decFanout.waitFor(t, 1, 3*time.Second)

	decEvent := decisionPub.messages()[0].(publishDecisionMessage).Event
	if decEvent.Metadata.CorrelationID != traceID {
		t.Fatalf("[decision] correlation ID lost: got %q", decEvent.Metadata.CorrelationID)
	}
	if decEvent.Metadata.CausationID == "" {
		t.Fatal("[decision] causation ID should be set")
	}

	e.Send(stratResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	stratEvent := strategyPub.messages()[0].(publishStrategyMessage).Event
	if stratEvent.Metadata.CorrelationID != traceID {
		t.Fatalf("[strategy] correlation ID lost: got %q", stratEvent.Metadata.CorrelationID)
	}

	e.Send(riskEvalPID, stratFanout.messages()[0].(strategyResolvedMessage))
	riskPub.waitFor(t, 1, 2*time.Second)
	execPub.waitFor(t, 1, 2*time.Second)

	riskEvent := riskPub.messages()[0].(publishRiskMessage).Event
	if riskEvent.Metadata.CorrelationID != traceID {
		t.Fatalf("[risk] correlation ID lost: got %q", riskEvent.Metadata.CorrelationID)
	}

	execEvent := execPub.messages()[0].(publishExecutionMessage).Event
	if execEvent.Metadata.CorrelationID != traceID {
		t.Fatalf("[execution] correlation ID lost: got %q", execEvent.Metadata.CorrelationID)
	}
	if execEvent.ExecutionIntent.CorrelationID != traceID {
		t.Fatalf("[execution/intent] correlation ID lost: got %q", execEvent.ExecutionIntent.CorrelationID)
	}
	if execEvent.ExecutionIntent.CausationID == "" {
		t.Fatal("[execution/intent] causation ID should be set")
	}

	t.Logf("[context-preservation] PASS — correlation ID %q preserved across all 5 stages", traceID)
}
