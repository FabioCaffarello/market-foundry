package derive

import (
	"fmt"
	"testing"
	"time"
)

// causal_chain_validation_test.go — S295: validates CausationID DAG linkage
// and CorrelationID immutability across all 3 approved slices.
//
// Existing closed-loop tests (S268, S291) proved that CorrelationID survives
// the actor chain and that CausationID is non-empty. This test goes further:
// it asserts the exact DAG linkage — that each stage's CausationID equals the
// parent stage's event Metadata.ID — proving the causal chain is reconstructable.

// --- Slice 1: Mean Reversion (RSI oversold → mean_reversion_entry) ---
func TestCausalChain_MeanReversion_DAGLinkage(t *testing.T) {
	const traceID = "s295-mr-causal"

	e := newTestEngine(t)

	// Collectors for published events (contain Metadata.ID and Metadata.CausationID).
	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	execPub := newMsgCollector()

	decisionPubPID := e.Spawn(decisionPub.producer(), "s295-mr-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "s295-mr-strat-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "s295-mr-risk-pub")
	execPubPID := e.Spawn(execPub.producer(), "s295-mr-exec-pub")

	// Fan-out collectors (carry CausationID set by parent stage).
	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()

	decFanoutPID := e.Spawn(decFanout.producer(), "s295-mr-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "s295-mr-strat-fan")

	// Wire: execution ← risk ← strategy ← decision.
	execEvalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		ExecutionPublisherPID: execPubPID,
	}), "s295-mr-exec-eval")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		RiskPublisherPID: riskPubPID, ScopePID: execEvalPID,
	}), "s295-mr-risk-eval")

	stratResolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: strategyPubPID, ScopePID: stratFanoutPID,
	}), "s295-mr-strat-resolver")

	decEvalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		DecisionPublisherPID: decisionPubPID, ScopePID: decFanoutPID,
	}), "s295-mr-dec-eval")

	time.Sleep(50 * time.Millisecond)

	// Inject: RSI 10 → high severity (triggers full chain).
	e.Send(decEvalPID, signalGeneratedMessage{
		Symbol: "btcusdt", SignalType: "rsi", SignalValue: "10.0000",
		Timeframe: 60, Timestamp: windowBase(),
		CorrelationID: traceID,
		CausationID:   "signal-root-001", // simulated signal event ID
	})

	// Collect all stages.
	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	decEvent := decisionPub.messages()[0].(publishDecisionMessage).Event
	decFanMsg := decFanout.messages()[0].(decisionEvaluatedMessage)

	e.Send(stratResolverPID, decFanMsg)
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	stratEvent := strategyPub.messages()[0].(publishStrategyMessage).Event
	stratFanMsg := stratFanout.messages()[0].(strategyResolvedMessage)

	e.Send(riskEvalPID, stratFanMsg)
	riskPub.waitFor(t, 1, 2*time.Second)
	execPub.waitFor(t, 1, 2*time.Second)

	riskEvent := riskPub.messages()[0].(publishRiskMessage).Event
	execEvent := execPub.messages()[0].(publishExecutionMessage).Event

	// === VALIDATE: CorrelationID immutable across all 4 stages ===
	stages := []struct {
		name string
		cid  string
	}{
		{"decision", decEvent.Metadata.CorrelationID},
		{"strategy", stratEvent.Metadata.CorrelationID},
		{"risk", riskEvent.Metadata.CorrelationID},
		{"execution/event", execEvent.Metadata.CorrelationID},
		{"execution/intent", execEvent.ExecutionIntent.CorrelationID},
	}
	for _, s := range stages {
		if s.cid != traceID {
			t.Fatalf("[%s] CorrelationID: want %q, got %q", s.name, traceID, s.cid)
		}
	}
	t.Log("[correlation] immutable across all stages")

	// === VALIDATE: CausationID DAG linkage ===
	// Decision.CausationID must equal the signal event ID injected.
	if decEvent.Metadata.CausationID != "signal-root-001" {
		t.Fatalf("[decision] CausationID: want signal-root-001, got %q", decEvent.Metadata.CausationID)
	}
	t.Logf("[decision] CausationID=%s (links to signal)", decEvent.Metadata.CausationID)

	// Fan-out CausationID must equal decision's Metadata.ID.
	if decFanMsg.CausationID != decEvent.Metadata.ID {
		t.Fatalf("[decision→strategy fan-out] CausationID: want %q (decision event ID), got %q",
			decEvent.Metadata.ID, decFanMsg.CausationID)
	}

	// Strategy.CausationID must equal decision's Metadata.ID.
	if stratEvent.Metadata.CausationID != decEvent.Metadata.ID {
		t.Fatalf("[strategy] CausationID: want %q (decision event ID), got %q",
			decEvent.Metadata.ID, stratEvent.Metadata.CausationID)
	}
	t.Logf("[strategy] CausationID=%s (links to decision %s)", stratEvent.Metadata.CausationID, decEvent.Metadata.ID)

	// Strategy fan-out CausationID must equal strategy's Metadata.ID.
	if stratFanMsg.CausationID != stratEvent.Metadata.ID {
		t.Fatalf("[strategy→risk fan-out] CausationID: want %q (strategy event ID), got %q",
			stratEvent.Metadata.ID, stratFanMsg.CausationID)
	}

	// Risk.CausationID must equal strategy's Metadata.ID.
	if riskEvent.Metadata.CausationID != stratEvent.Metadata.ID {
		t.Fatalf("[risk] CausationID: want %q (strategy event ID), got %q",
			stratEvent.Metadata.ID, riskEvent.Metadata.CausationID)
	}
	t.Logf("[risk] CausationID=%s (links to strategy %s)", riskEvent.Metadata.CausationID, stratEvent.Metadata.ID)

	// Execution.CausationID must equal risk's Metadata.ID.
	if execEvent.Metadata.CausationID != riskEvent.Metadata.ID {
		t.Fatalf("[execution] CausationID: want %q (risk event ID), got %q",
			riskEvent.Metadata.ID, execEvent.Metadata.CausationID)
	}
	if execEvent.ExecutionIntent.CausationID != riskEvent.Metadata.ID {
		t.Fatalf("[execution/intent] CausationID: want %q (risk event ID), got %q",
			riskEvent.Metadata.ID, execEvent.ExecutionIntent.CausationID)
	}
	t.Logf("[execution] CausationID=%s (links to risk %s)", execEvent.Metadata.CausationID, riskEvent.Metadata.ID)

	// === VALIDATE: All Metadata.IDs are unique (DAG, no cycles) ===
	ids := map[string]string{
		"decision":  decEvent.Metadata.ID,
		"strategy":  stratEvent.Metadata.ID,
		"risk":      riskEvent.Metadata.ID,
		"execution": execEvent.Metadata.ID,
	}
	seen := map[string]bool{}
	for stage, id := range ids {
		if id == "" {
			t.Fatalf("[%s] Metadata.ID is empty", stage)
		}
		if seen[id] {
			t.Fatalf("[%s] Metadata.ID %q duplicated (DAG violated)", stage, id)
		}
		seen[id] = true
	}

	t.Log("[causal-chain/mean-reversion] PASS — full DAG linkage validated")
}

// --- Slice 2: Trend Following (EMA crossover → trend_following_entry) ---
func TestCausalChain_TrendFollowing_DAGLinkage(t *testing.T) {
	const traceID = "s295-tf-causal"

	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	execPub := newMsgCollector()

	decisionPubPID := e.Spawn(decisionPub.producer(), "s295-tf-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "s295-tf-strat-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "s295-tf-risk-pub")
	execPubPID := e.Spawn(execPub.producer(), "s295-tf-exec-pub")

	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()

	decFanoutPID := e.Spawn(decFanout.producer(), "s295-tf-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "s295-tf-strat-fan")

	execEvalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		ExecutionPublisherPID: execPubPID,
	}), "s295-tf-exec-eval")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		RiskPublisherPID: riskPubPID, ScopePID: execEvalPID,
	}), "s295-tf-risk-eval")

	stratResolverPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: strategyPubPID, ScopePID: stratFanoutPID,
	}), "s295-tf-strat-resolver")

	decEvalPID := e.Spawn(NewEMACrossoverEvaluatorActor(DecisionEvaluatorConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		DecisionPublisherPID: decisionPubPID, ScopePID: decFanoutPID,
	}), "s295-tf-dec-eval")

	time.Sleep(50 * time.Millisecond)

	e.Send(decEvalPID, signalGeneratedMessage{
		Symbol: "btcusdt", SignalType: "ema_crossover", SignalValue: "bullish",
		Timeframe: 60, Timestamp: windowBase(),
		CorrelationID: traceID,
		CausationID:   "signal-root-002",
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	decEvent := decisionPub.messages()[0].(publishDecisionMessage).Event
	e.Send(stratResolverPID, decFanout.messages()[0].(decisionEvaluatedMessage))
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	stratEvent := strategyPub.messages()[0].(publishStrategyMessage).Event
	e.Send(riskEvalPID, stratFanout.messages()[0].(strategyResolvedMessage))
	riskPub.waitFor(t, 1, 2*time.Second)
	execPub.waitFor(t, 1, 2*time.Second)

	riskEvent := riskPub.messages()[0].(publishRiskMessage).Event
	execEvent := execPub.messages()[0].(publishExecutionMessage).Event

	// CorrelationID immutable.
	for _, s := range []struct {
		name, cid string
	}{
		{"decision", decEvent.Metadata.CorrelationID},
		{"strategy", stratEvent.Metadata.CorrelationID},
		{"risk", riskEvent.Metadata.CorrelationID},
		{"execution/event", execEvent.Metadata.CorrelationID},
		{"execution/intent", execEvent.ExecutionIntent.CorrelationID},
	} {
		if s.cid != traceID {
			t.Fatalf("[%s] CorrelationID: want %q, got %q", s.name, traceID, s.cid)
		}
	}

	// CausationID DAG.
	assertCausation(t, "decision", decEvent.Metadata.CausationID, "signal-root-002")
	assertCausation(t, "strategy", stratEvent.Metadata.CausationID, decEvent.Metadata.ID)
	assertCausation(t, "risk", riskEvent.Metadata.CausationID, stratEvent.Metadata.ID)
	assertCausation(t, "execution", execEvent.Metadata.CausationID, riskEvent.Metadata.ID)
	assertCausation(t, "execution/intent", execEvent.ExecutionIntent.CausationID, riskEvent.Metadata.ID)

	// ID uniqueness.
	assertUniqueIDs(t, map[string]string{
		"decision": decEvent.Metadata.ID, "strategy": stratEvent.Metadata.ID,
		"risk": riskEvent.Metadata.ID, "execution": execEvent.Metadata.ID,
	})

	t.Log("[causal-chain/trend-following] PASS — full DAG linkage validated")
}

// --- Slice 3: Squeeze Breakout (Bollinger → squeeze_breakout_entry) ---
func TestCausalChain_SqueezeBreakout_DAGLinkage(t *testing.T) {
	const traceID = "s295-sq-causal"

	e := newTestEngine(t)

	signalPub := newMsgCollector()
	decisionPub := newMsgCollector()
	strategyPub := newMsgCollector()
	riskPub := newMsgCollector()
	execPub := newMsgCollector()

	signalPubPID := e.Spawn(signalPub.producer(), "s295-sq-sig-pub")
	decisionPubPID := e.Spawn(decisionPub.producer(), "s295-sq-dec-pub")
	strategyPubPID := e.Spawn(strategyPub.producer(), "s295-sq-strat-pub")
	riskPubPID := e.Spawn(riskPub.producer(), "s295-sq-risk-pub")
	execPubPID := e.Spawn(execPub.producer(), "s295-sq-exec-pub")

	signalFanout := newMsgCollector()
	decFanout := newMsgCollector()
	stratFanout := newMsgCollector()

	signalFanoutPID := e.Spawn(signalFanout.producer(), "s295-sq-sig-fan")
	decFanoutPID := e.Spawn(decFanout.producer(), "s295-sq-dec-fan")
	stratFanoutPID := e.Spawn(stratFanout.producer(), "s295-sq-strat-fan")

	execEvalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		ExecutionPublisherPID: execPubPID,
	}), "s295-sq-exec-eval")

	riskEvalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		RiskPublisherPID: riskPubPID, ScopePID: execEvalPID,
	}), "s295-sq-risk-eval")

	stratResolverPID := e.Spawn(NewSqueezeBreakoutEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: strategyPubPID, ScopePID: stratFanoutPID,
	}), "s295-sq-strat-resolver")

	decEvalPID := e.Spawn(NewBollingerSqueezeEvaluatorActor(DecisionEvaluatorConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		DecisionPublisherPID: decisionPubPID, ScopePID: decFanoutPID,
	}), "s295-sq-dec-eval")

	samplerPID := e.Spawn(NewBollingerSignalSamplerActor(SignalSamplerConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		SignalPublisherPID: signalPubPID, ScopePID: signalFanoutPID,
	}), "s295-sq-boll-sampler")

	time.Sleep(50 * time.Millisecond)

	// Feed 20 tight-range candles to trigger squeeze.
	base := windowBase()
	for i := 0; i < 20; i++ {
		price := 100.0 + float64(i)*0.01
		e.Send(samplerPID, candleFinalizedMessage{
			Symbol: "btcusdt", ClosePrice: fmt.Sprintf("%.4f", price),
			Timestamp:     base.Add(time.Duration(i) * time.Minute),
			CorrelationID: traceID,
		})
	}

	// Collect signal.
	signalPub.waitFor(t, 1, 3*time.Second)
	signalFanout.waitFor(t, 1, 3*time.Second)

	sigEvent := signalPub.messages()[0].(publishSignalMessage).Event
	sigFanMsg := signalFanout.messages()[0].(signalGeneratedMessage)

	// Signal CorrelationID check.
	if sigEvent.Metadata.CorrelationID != traceID {
		t.Fatalf("[signal] CorrelationID: want %q, got %q", traceID, sigEvent.Metadata.CorrelationID)
	}

	// Signal is root: CausationID is empty (no parent domain event).
	// This is by-design — signal events originate from internal candleFinalizedMessage.
	t.Logf("[signal] CorrelationID=%s CausationID=%q (root, empty by design)", sigEvent.Metadata.CorrelationID, sigEvent.Metadata.CausationID)

	// Fan-out CausationID must equal signal's Metadata.ID.
	if sigFanMsg.CausationID != sigEvent.Metadata.ID {
		t.Fatalf("[signal→decision fan-out] CausationID: want %q (signal event ID), got %q",
			sigEvent.Metadata.ID, sigFanMsg.CausationID)
	}

	// Forward through chain.
	e.Send(decEvalPID, sigFanMsg)
	decisionPub.waitFor(t, 1, 3*time.Second)
	decFanout.waitFor(t, 1, 3*time.Second)

	decEvent := decisionPub.messages()[0].(publishDecisionMessage).Event
	decFanMsg := decFanout.messages()[0].(decisionEvaluatedMessage)

	e.Send(stratResolverPID, decFanMsg)
	strategyPub.waitFor(t, 1, 2*time.Second)
	stratFanout.waitFor(t, 1, 2*time.Second)

	stratEvent := strategyPub.messages()[0].(publishStrategyMessage).Event
	stratFanMsg := stratFanout.messages()[0].(strategyResolvedMessage)

	e.Send(riskEvalPID, stratFanMsg)
	riskPub.waitFor(t, 1, 2*time.Second)
	execPub.waitFor(t, 1, 2*time.Second)

	riskEvent := riskPub.messages()[0].(publishRiskMessage).Event
	execEvent := execPub.messages()[0].(publishExecutionMessage).Event

	// === CorrelationID immutable across all 5 stages ===
	for _, s := range []struct {
		name, cid string
	}{
		{"signal", sigEvent.Metadata.CorrelationID},
		{"decision", decEvent.Metadata.CorrelationID},
		{"strategy", stratEvent.Metadata.CorrelationID},
		{"risk", riskEvent.Metadata.CorrelationID},
		{"execution/event", execEvent.Metadata.CorrelationID},
		{"execution/intent", execEvent.ExecutionIntent.CorrelationID},
	} {
		if s.cid != traceID {
			t.Fatalf("[%s] CorrelationID: want %q, got %q", s.name, traceID, s.cid)
		}
	}

	// === CausationID DAG linkage across all 5 stages ===
	// Signal → Decision: decision.CausationID == signal.Metadata.ID
	assertCausation(t, "decision", decEvent.Metadata.CausationID, sigEvent.Metadata.ID)

	// Decision → Strategy: strategy.CausationID == decision.Metadata.ID
	assertCausation(t, "strategy", stratEvent.Metadata.CausationID, decEvent.Metadata.ID)

	// Strategy → Risk: risk.CausationID == strategy.Metadata.ID
	assertCausation(t, "risk", riskEvent.Metadata.CausationID, stratEvent.Metadata.ID)

	// Risk → Execution: execution.CausationID == risk.Metadata.ID
	assertCausation(t, "execution", execEvent.Metadata.CausationID, riskEvent.Metadata.ID)
	assertCausation(t, "execution/intent", execEvent.ExecutionIntent.CausationID, riskEvent.Metadata.ID)

	// Fan-out CausationIDs match parent event IDs.
	assertCausation(t, "signal→decision fan-out", sigFanMsg.CausationID, sigEvent.Metadata.ID)
	assertCausation(t, "decision→strategy fan-out", decFanMsg.CausationID, decEvent.Metadata.ID)
	assertCausation(t, "strategy→risk fan-out", stratFanMsg.CausationID, stratEvent.Metadata.ID)

	// ID uniqueness (DAG, no cycles).
	assertUniqueIDs(t, map[string]string{
		"signal": sigEvent.Metadata.ID, "decision": decEvent.Metadata.ID,
		"strategy": stratEvent.Metadata.ID, "risk": riskEvent.Metadata.ID,
		"execution": execEvent.Metadata.ID,
	})

	t.Log("[causal-chain/squeeze-breakout] PASS — full 5-stage DAG linkage validated")
}

// --- helpers ---

func assertCausation(t *testing.T, stage, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("[%s] CausationID: want %q, got %q", stage, want, got)
	}
	t.Logf("[%s] CausationID=%s ✓", stage, got)
}

func assertUniqueIDs(t *testing.T, ids map[string]string) {
	t.Helper()
	seen := map[string]string{}
	for stage, id := range ids {
		if id == "" {
			t.Fatalf("[%s] Metadata.ID is empty", stage)
		}
		if prev, ok := seen[id]; ok {
			t.Fatalf("Metadata.ID %q duplicated between %s and %s (DAG violated)", id, prev, stage)
		}
		seen[id] = stage
	}
}
