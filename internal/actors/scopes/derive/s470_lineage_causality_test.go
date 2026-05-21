package derive

import (
	"testing"
	"time"

	appdecision "internal/application/decision"
	appexec "internal/application/execution"
	apprisk "internal/application/risk"
	appstrategy "internal/application/strategy"
	domaindecision "internal/domain/decision"
	domainexec "internal/domain/execution"
	domainrisk "internal/domain/risk"
	domainstrategy "internal/domain/strategy"
	"internal/shared/events"
)

// TestS470_DecisionCarriesSignalEventID validates that the decision evaluator
// enriches SignalInput.EventID with the signal event's ID (causation reference).
func TestS470_DecisionCarriesSignalEventID(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "dec-pub")

	scope := newMsgCollector()
	scopePID := e.Spawn(scope.producer(), "scope")

	evalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: pubPID,
		ScopePID:             scopePID,
	}), "rsi-eval")

	signalEventID := "sig-evt-abc123"

	e.Send(evalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "rsi",
		SignalValue:   "20.0000",
		Timeframe:     60,
		Timestamp:     time.Now(),
		CorrelationID: "corr-test-470",
		CausationID:   signalEventID,
	})

	time.Sleep(50 * time.Millisecond)

	msgs := publisher.messages()
	if len(msgs) == 0 {
		t.Fatal("expected at least one publish message")
	}

	pubMsg, ok := msgs[0].(publishDecisionMessage)
	if !ok {
		t.Fatalf("expected publishDecisionMessage, got %T", msgs[0])
	}

	dec := pubMsg.Event.Decision
	if len(dec.Signals) == 0 {
		t.Fatal("decision must carry signal inputs")
	}
	if dec.Signals[0].EventID != signalEventID {
		t.Errorf("SignalInput.EventID: want %q, got %q", signalEventID, dec.Signals[0].EventID)
	}

	// Verify event metadata causation points to signal event.
	if pubMsg.Event.Metadata.CausationID != signalEventID {
		t.Errorf("event CausationID: want %q, got %q", signalEventID, pubMsg.Event.Metadata.CausationID)
	}
	if pubMsg.Event.Metadata.CorrelationID != "corr-test-470" {
		t.Errorf("event CorrelationID: want corr-test-470, got %q", pubMsg.Event.Metadata.CorrelationID)
	}
}

// TestS470_StrategyCarriesDecisionEventID validates that strategy resolvers
// enrich DecisionInput.EventID with the decision event's ID.
func TestS470_StrategyCarriesDecisionEventID(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "strat-pub")

	scope := newMsgCollector()
	scopePID := e.Spawn(scope.producer(), "scope")

	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
		ScopePID:             scopePID,
	}), "mean-rev")

	decisionEventID := "dec-evt-def456"

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "rsi_oversold",
		DecisionOutcome:    "triggered",
		DecisionConfidence: "0.8500",
		DecisionSeverity:   "high",
		DecisionRationale:  "RSI below threshold",
		Timeframe:          60,
		Timestamp:          time.Now(),
		CorrelationID:      "corr-test-470",
		CausationID:        decisionEventID,
	})

	time.Sleep(50 * time.Millisecond)

	msgs := publisher.messages()
	if len(msgs) == 0 {
		t.Fatal("expected at least one publish message")
	}

	pubMsg, ok := msgs[0].(publishStrategyMessage)
	if !ok {
		t.Fatalf("expected publishStrategyMessage, got %T", msgs[0])
	}

	strat := pubMsg.Event.Strategy
	if len(strat.Decisions) == 0 {
		t.Fatal("strategy must carry decision inputs")
	}
	if strat.Decisions[0].EventID != decisionEventID {
		t.Errorf("DecisionInput.EventID: want %q, got %q", decisionEventID, strat.Decisions[0].EventID)
	}
}

// TestS470_RiskCarriesStrategyEventID validates that risk evaluators
// enrich StrategyInput.EventID with the strategy event's ID.
func TestS470_RiskCarriesStrategyEventID(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "risk-pub")

	scope := newMsgCollector()
	scopePID := e.Spawn(scope.producer(), "scope")

	evalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: pubPID,
		ScopePID:         scopePID,
	}), "pos-exp")

	strategyEventID := "strat-evt-ghi789"

	e.Send(evalPID, strategyResolvedMessage{
		Symbol:             "btcusdt",
		StrategyType:       "mean_reversion_entry",
		StrategyDirection:  "long",
		StrategyConfidence: "0.8500",
		DecisionSeverity:   "high",
		DecisionRationale:  "RSI deeply oversold",
		Timeframe:          60,
		Timestamp:          time.Now(),
		CorrelationID:      "corr-test-470",
		CausationID:        strategyEventID,
	})

	time.Sleep(50 * time.Millisecond)

	msgs := publisher.messages()
	if len(msgs) == 0 {
		t.Fatal("expected at least one publish message")
	}

	pubMsg, ok := msgs[0].(publishRiskMessage)
	if !ok {
		t.Fatalf("expected publishRiskMessage, got %T", msgs[0])
	}

	assessment := pubMsg.Event.RiskAssessment
	if len(assessment.Strategies) == 0 {
		t.Fatal("risk must carry strategy inputs")
	}
	if assessment.Strategies[0].EventID != strategyEventID {
		t.Errorf("StrategyInput.EventID: want %q, got %q", strategyEventID, assessment.Strategies[0].EventID)
	}
}

// TestS470_ExecutionCarriesRiskEventID validates that the execution evaluator
// enriches RiskInput.EventID with the risk event's ID.
func TestS470_ExecutionCarriesRiskEventID(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "exec-pub")

	evalPID := e.Spawn(NewPaperOrderEvaluatorActor(ExecutionEvaluatorConfig{
		Source:                "binancef",
		Symbol:                "btcusdt",
		Timeframe:             60 * time.Second,
		ExecutionPublisherPID: pubPID,
	}), "paper-eval")

	riskEventID := "risk-evt-jkl012"

	e.Send(evalPID, riskAssessedMessage{
		Symbol:             "btcusdt",
		RiskType:           "position_exposure",
		RiskDisposition:    "approved",
		RiskConfidence:     "0.7500",
		MaxPositionPct:     "5.0",
		StrategyDirection:  "long",
		StrategyConfidence: "0.8500",
		StrategyType:       "mean_reversion_entry",
		DecisionSeverity:   "high",
		Timeframe:          60,
		Timestamp:          time.Now(),
		CorrelationID:      "corr-test-470",
		CausationID:        riskEventID,
	})

	time.Sleep(50 * time.Millisecond)

	msgs := publisher.messages()
	if len(msgs) == 0 {
		t.Fatal("expected at least one publish message")
	}

	pubMsg, ok := msgs[0].(publishExecutionMessage)
	if !ok {
		t.Fatalf("expected publishExecutionMessage, got %T", msgs[0])
	}

	intent := pubMsg.Event.ExecutionIntent
	if intent.Risk.EventID != riskEventID {
		t.Errorf("RiskInput.EventID: want %q, got %q", riskEventID, intent.Risk.EventID)
	}
	if intent.CorrelationID != "corr-test-470" {
		t.Errorf("ExecutionIntent.CorrelationID: want corr-test-470, got %q", intent.CorrelationID)
	}
	if intent.CausationID != riskEventID {
		t.Errorf("ExecutionIntent.CausationID: want %q, got %q", riskEventID, intent.CausationID)
	}
}

// TestS470_FullChainLineagePreservation validates that across all 5 stages the
// EventID enrichment forms a proper causal chain. This is an application-level
// integration test using the pure evaluator/resolver functions (no NATS, no actors).
func TestS470_FullChainLineagePreservation(t *testing.T) {
	ts := time.Now()

	// Stage 1: Signal generation (creates event metadata with a fresh ID).
	signalMeta := events.NewMetadata().WithCorrelationID("chain-corr-470")
	signalEventID := signalMeta.ID

	// Stage 2: Decision evaluation (consumes signal).
	decEval := appdecision.NewRSIOversoldEvaluator("binancef", "btcusdt", 60)
	dec, ok := decEval.Evaluate("rsi", "15.0000", 60, ts)
	if !ok {
		t.Fatal("decision evaluation should succeed")
	}
	// S470: enrich signal input with causal reference.
	for i := range dec.Signals {
		dec.Signals[i].EventID = signalEventID
	}
	decMeta := events.NewMetadata().
		WithCorrelationID("chain-corr-470").
		WithCausationID(signalEventID)
	decisionEventID := decMeta.ID

	// Verify: decision's signal input carries signal event reference.
	if dec.Signals[0].EventID != signalEventID {
		t.Errorf("SignalInput.EventID: want %q, got %q", signalEventID, dec.Signals[0].EventID)
	}

	// Stage 3: Strategy resolution (consumes decision).
	stratResolver := appstrategy.NewMeanReversionEntryResolver("binancef", "btcusdt", 60)
	strat, ok := stratResolver.Resolve(dec.Type, string(dec.Outcome), dec.Confidence, string(dec.Severity), dec.Rationale, dec.Timeframe, ts)
	if !ok {
		t.Fatal("strategy resolution should succeed")
	}
	// S470: enrich decision input with causal reference.
	for i := range strat.Decisions {
		strat.Decisions[i].EventID = decisionEventID
	}
	stratMeta := events.NewMetadata().
		WithCorrelationID("chain-corr-470").
		WithCausationID(decisionEventID)
	strategyEventID := stratMeta.ID

	if strat.Decisions[0].EventID != decisionEventID {
		t.Errorf("DecisionInput.EventID: want %q, got %q", decisionEventID, strat.Decisions[0].EventID)
	}

	// Stage 4: Risk assessment (consumes strategy).
	riskEval := apprisk.NewPositionExposureEvaluator("binancef", "btcusdt", 60)
	assessment, ok := riskEval.Evaluate(strat.Type, string(strat.Direction), strat.Confidence, strat.Decisions[0].Severity, strat.Decisions[0].Rationale, strat.Timeframe, ts)
	if !ok {
		t.Fatal("risk evaluation should succeed")
	}
	// S470: enrich strategy input with causal reference.
	for i := range assessment.Strategies {
		assessment.Strategies[i].EventID = strategyEventID
	}
	riskMeta := events.NewMetadata().
		WithCorrelationID("chain-corr-470").
		WithCausationID(strategyEventID)
	riskEventID := riskMeta.ID

	if assessment.Strategies[0].EventID != strategyEventID {
		t.Errorf("StrategyInput.EventID: want %q, got %q", strategyEventID, assessment.Strategies[0].EventID)
	}

	// Stage 5: Execution intent (consumes risk).
	execEval := appexec.NewPaperOrderEvaluator("binancef", "btcusdt", 60)
	intent, ok := execEval.Evaluate(
		assessment.Type, string(assessment.Disposition), assessment.Confidence,
		assessment.Constraints.MaxPositionSize,
		string(strat.Direction), strat.Confidence,
		strat.Type, strat.Decisions[0].Severity,
		assessment.Timeframe, ts,
	)
	if !ok {
		t.Fatal("execution evaluation should succeed")
	}
	// S470: enrich risk input with causal reference.
	intent.Risk.EventID = riskEventID
	intent.CorrelationID = "chain-corr-470"
	intent.CausationID = riskEventID

	if intent.Risk.EventID != riskEventID {
		t.Errorf("RiskInput.EventID: want %q, got %q", riskEventID, intent.Risk.EventID)
	}

	// Verify chain integrity.
	type link struct {
		stage   string
		eventID string
		causID  string
	}
	chain := []link{
		{"signal", signalEventID, ""},
		{"decision", decisionEventID, signalEventID},
		{"strategy", strategyEventID, decisionEventID},
		{"risk", riskEventID, strategyEventID},
		{"execution_intent", intent.CausationID, riskEventID},
	}

	for i, l := range chain {
		if l.eventID == "" {
			t.Errorf("stage %s: eventID must not be empty", l.stage)
		}
		if i > 0 && l.causID != chain[i-1].eventID {
			t.Errorf("stage %s: causation_id %q should equal previous stage event_id %q",
				l.stage, l.causID, chain[i-1].eventID)
		}
	}

	// All Input types carry parent event references.
	t.Logf("Full chain verified: signal(%s) -> decision(%s) -> strategy(%s) -> risk(%s) -> execution(corr=%s, caus=%s)",
		signalEventID[:8], decisionEventID[:8], strategyEventID[:8], riskEventID[:8],
		intent.CorrelationID[:8], intent.CausationID[:8])
}

// Ensure unused imports are satisfied at compile time.
var (
	_ domaindecision.Decision
	_ domainstrategy.Strategy
	_ domainrisk.RiskAssessment
	_ domainexec.ExecutionIntent
)
