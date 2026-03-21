package derive

import (
	"testing"
	"time"
)

func TestMeanReversionResolverActor_Triggered_LongDirection(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "rsi_oversold",
		DecisionOutcome:    "triggered",
		DecisionConfidence: "0.7500",
		Timeframe:          60,
		Timestamp:          windowBase(),
		CorrelationID:      "corr-strat",
	})

	publisher.waitFor(t, 1, 2*time.Second)

	msg, ok := publisher.messages()[0].(publishStrategyMessage)
	if !ok {
		t.Fatalf("expected publishStrategyMessage, got %T", publisher.messages()[0])
	}

	s := msg.Event.Strategy
	if s.Type != "mean_reversion_entry" {
		t.Errorf("strategy type: want mean_reversion_entry, got %s", s.Type)
	}
	if s.Source != "binancef" {
		t.Errorf("source: want binancef, got %s", s.Source)
	}
	if s.Symbol != "btcusdt" {
		t.Errorf("symbol: want btcusdt, got %s", s.Symbol)
	}
	if string(s.Direction) != "long" {
		t.Errorf("direction: want long, got %s", s.Direction)
	}
	if s.Confidence != "0.7500" {
		t.Errorf("confidence: want 0.7500, got %s", s.Confidence)
	}
	if !s.Final {
		t.Error("expected final=true")
	}
	if len(s.Decisions) != 1 {
		t.Fatalf("expected 1 decision input, got %d", len(s.Decisions))
	}
	if s.Decisions[0].Type != "rsi_oversold" || s.Decisions[0].Outcome != "triggered" {
		t.Errorf("decision input: want rsi_oversold/triggered, got %s/%s", s.Decisions[0].Type, s.Decisions[0].Outcome)
	}

	// Verify parameters are set for triggered.
	if s.Parameters["entry"] != "market" {
		t.Errorf("parameter entry: want market, got %s", s.Parameters["entry"])
	}
	if s.Parameters["target_offset"] != "0.02" {
		t.Errorf("parameter target_offset: want 0.02, got %s", s.Parameters["target_offset"])
	}
	if s.Parameters["stop_offset"] != "0.01" {
		t.Errorf("parameter stop_offset: want 0.01, got %s", s.Parameters["stop_offset"])
	}

	if prob := s.Validate(); prob != nil {
		t.Errorf("strategy validation failed: %s", prob.Message)
	}

	// Verify correlation ID propagated.
	if msg.Event.Metadata.CorrelationID != "corr-strat" {
		t.Errorf("correlationID: want corr-strat, got %s", msg.Event.Metadata.CorrelationID)
	}
}

func TestMeanReversionResolverActor_NotTriggered_FlatDirection(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "rsi_oversold",
		DecisionOutcome:    "not_triggered",
		DecisionConfidence: "0.8000",
		Timeframe:          60,
		Timestamp:          windowBase(),
	})

	publisher.waitFor(t, 1, 2*time.Second)

	s := publisher.messages()[0].(publishStrategyMessage).Event.Strategy
	if string(s.Direction) != "flat" {
		t.Errorf("direction: want flat, got %s", s.Direction)
	}
	if s.Confidence != "0.0000" {
		t.Errorf("confidence: want 0.0000, got %s", s.Confidence)
	}
	if s.Parameters != nil {
		t.Errorf("expected nil parameters for flat, got %v", s.Parameters)
	}
}

func TestMeanReversionResolverActor_Insufficient_FlatWithReason(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "rsi_oversold",
		DecisionOutcome:    "insufficient",
		DecisionConfidence: "0.0000",
		Timeframe:          60,
		Timestamp:          windowBase(),
	})

	publisher.waitFor(t, 1, 2*time.Second)

	s := publisher.messages()[0].(publishStrategyMessage).Event.Strategy
	if string(s.Direction) != "flat" {
		t.Errorf("direction: want flat, got %s", s.Direction)
	}
	if s.Metadata["reason"] != "insufficient_data" {
		t.Errorf("metadata.reason: want insufficient_data, got %s", s.Metadata["reason"])
	}
}

func TestMeanReversionResolverActor_UnknownOutcome_NoPublish(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "rsi_oversold",
		DecisionOutcome:    "unknown_outcome",
		DecisionConfidence: "0.5000",
		Timeframe:          60,
		Timestamp:          windowBase(),
	})

	time.Sleep(200 * time.Millisecond)
	if publisher.count() != 0 {
		t.Fatalf("expected no strategy for unknown outcome, got %d", publisher.count())
	}
}

func TestMeanReversionResolverActor_SeverityAndRationale_Propagated(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "rsi_oversold",
		DecisionOutcome:    "triggered",
		DecisionConfidence: "0.8500",
		DecisionSeverity:   "high",
		DecisionRationale:  "RSI 5.00 below oversold threshold 30.0 (distance 83.3%); severity high",
		Timeframe:          60,
		Timestamp:          windowBase(),
		CorrelationID:      "corr-sev",
	})

	publisher.waitFor(t, 1, 2*time.Second)

	msg := publisher.messages()[0].(publishStrategyMessage)
	s := msg.Event.Strategy

	// Severity and rationale must appear in DecisionInput.
	if len(s.Decisions) != 1 {
		t.Fatalf("expected 1 decision input, got %d", len(s.Decisions))
	}
	if s.Decisions[0].Severity != "high" {
		t.Errorf("decision severity: want high, got %s", s.Decisions[0].Severity)
	}
	if s.Decisions[0].Rationale == "" {
		t.Error("expected decision rationale to be non-empty")
	}
	// Rationale should also appear in strategy metadata.
	if s.Metadata["decision_rationale"] == "" {
		t.Error("expected decision_rationale in strategy metadata")
	}
}

func TestMeanReversionResolverActor_NilScopePID_PublishesWithoutFanout(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
		ScopePID:             nil,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "rsi_oversold",
		DecisionOutcome:    "triggered",
		DecisionConfidence: "0.7500",
		Timeframe:          60,
		Timestamp:          windowBase(),
	})

	publisher.waitFor(t, 1, 2*time.Second)
	s := publisher.messages()[0].(publishStrategyMessage).Event.Strategy
	if string(s.Direction) != "long" {
		t.Errorf("direction: want long, got %s", s.Direction)
	}
}

func TestMeanReversionResolverActor_FanOut_IncludesDecisionContext(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	scope := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")
	scopePID := e.Spawn(scope.producer(), "scope")

	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
		ScopePID:             scopePID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "rsi_oversold",
		DecisionOutcome:    "triggered",
		DecisionConfidence: "0.8000",
		DecisionSeverity:   "moderate",
		DecisionRationale:  "RSI 20.00 below threshold",
		Timeframe:          60,
		Timestamp:          windowBase(),
		CorrelationID:      "corr-fanout",
	})

	publisher.waitFor(t, 1, 2*time.Second)
	scope.waitFor(t, 1, 2*time.Second)

	fanout, ok := scope.messages()[0].(strategyResolvedMessage)
	if !ok {
		t.Fatalf("expected strategyResolvedMessage, got %T", scope.messages()[0])
	}
	if fanout.DecisionSeverity != "moderate" {
		t.Errorf("decision severity in fan-out: want moderate, got %s", fanout.DecisionSeverity)
	}
	if fanout.DecisionRationale != "RSI 20.00 below threshold" {
		t.Errorf("decision rationale in fan-out: want RSI 20.00 below threshold, got %s", fanout.DecisionRationale)
	}
	if fanout.StrategyDirection != "long" {
		t.Errorf("strategy direction in fan-out: want long, got %s", fanout.StrategyDirection)
	}
	if fanout.CorrelationID != "corr-fanout" {
		t.Errorf("correlationID in fan-out: want corr-fanout, got %s", fanout.CorrelationID)
	}
}

func TestMeanReversionResolverActor_InvalidConfidence_NoPublish(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "rsi_oversold",
		DecisionOutcome:    "triggered",
		DecisionConfidence: "not-a-number",
		Timeframe:          60,
		Timestamp:          windowBase(),
	})

	time.Sleep(200 * time.Millisecond)
	if publisher.count() != 0 {
		t.Fatalf("expected no strategy for invalid confidence, got %d", publisher.count())
	}
}

func TestMeanReversionResolverActor_SequentialDecisions_Independent(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	// Triggered → Long.
	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
		DecisionConfidence: "0.8000", Timeframe: 60, Timestamp: base,
	})
	// Not triggered → Flat.
	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "not_triggered",
		DecisionConfidence: "0.9000", Timeframe: 60, Timestamp: base.Add(time.Minute),
	})

	publisher.waitFor(t, 2, 2*time.Second)

	msgs := publisher.messages()
	s0 := msgs[0].(publishStrategyMessage).Event.Strategy
	s1 := msgs[1].(publishStrategyMessage).Event.Strategy

	if string(s0.Direction) != "long" {
		t.Errorf("first direction: want long, got %s", s0.Direction)
	}
	if string(s1.Direction) != "flat" {
		t.Errorf("second direction: want flat, got %s", s1.Direction)
	}
}
