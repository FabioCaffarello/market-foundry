package derive

import (
	"testing"
	"time"
)

func TestTrendFollowingResolverActor_Triggered_LongDirection(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "ema_crossover",
		DecisionOutcome:    "triggered",
		DecisionConfidence: "0.7500",
		Timeframe:          60,
		Timestamp:          windowBase(),
		CorrelationID:      "corr-trend",
	})

	publisher.waitFor(t, 1, 2*time.Second)

	msg, ok := publisher.messages()[0].(publishStrategyMessage)
	if !ok {
		t.Fatalf("expected publishStrategyMessage, got %T", publisher.messages()[0])
	}

	s := msg.Event.Strategy
	if s.Type != "trend_following_entry" {
		t.Errorf("strategy type: want trend_following_entry, got %s", s.Type)
	}
	if s.Source != "binancef" {
		t.Errorf("source: want binancef, got %s", s.Source)
	}
	if s.VenueSymbol() != "btcusdt" {
		t.Errorf("symbol: want btcusdt, got %s", s.VenueSymbol())
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
	if s.Decisions[0].Type != "ema_crossover" || s.Decisions[0].Outcome != "triggered" {
		t.Errorf("decision input: want ema_crossover/triggered, got %s/%s", s.Decisions[0].Type, s.Decisions[0].Outcome)
	}

	// Verify parameters are set for triggered.
	if s.Parameters["entry"] != "market" {
		t.Errorf("parameter entry: want market, got %s", s.Parameters["entry"])
	}
	if s.Parameters["trailing_stop_pct"] != "0.03" {
		t.Errorf("parameter trailing_stop_pct: want 0.03, got %s", s.Parameters["trailing_stop_pct"])
	}
	if s.Parameters["take_profit_pct"] != "0.05" {
		t.Errorf("parameter take_profit_pct: want 0.05, got %s", s.Parameters["take_profit_pct"])
	}

	if prob := s.Validate(); prob != nil {
		t.Errorf("strategy validation failed: %s", prob.Message)
	}

	// Verify correlation ID propagated.
	if msg.Event.Metadata.CorrelationID != "corr-trend" {
		t.Errorf("correlationID: want corr-trend, got %s", msg.Event.Metadata.CorrelationID)
	}
}

func TestTrendFollowingResolverActor_NotTriggered_FlatDirection(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "ema_crossover",
		DecisionOutcome:    "not_triggered",
		DecisionConfidence: "0.7500",
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

func TestTrendFollowingResolverActor_Insufficient_FlatWithReason(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "ema_crossover",
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

func TestTrendFollowingResolverActor_UnknownOutcome_NoPublish(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "ema_crossover",
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

func TestTrendFollowingResolverActor_SeverityAndRationale_Propagated(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	resolverPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "ema_crossover",
		DecisionOutcome:    "triggered",
		DecisionConfidence: "0.7500",
		DecisionSeverity:   "moderate",
		DecisionRationale:  "EMA crossover bullish: fast EMA above slow EMA on 60s timeframe",
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
	if s.Decisions[0].Severity != "moderate" {
		t.Errorf("decision severity: want moderate, got %s", s.Decisions[0].Severity)
	}
	if s.Decisions[0].Rationale == "" {
		t.Error("expected decision rationale to be non-empty")
	}
	// Rationale should also appear in strategy metadata.
	if s.Metadata["decision_rationale"] == "" {
		t.Error("expected decision_rationale in strategy metadata")
	}
}

func TestTrendFollowingResolverActor_FanOut_IncludesDecisionContext(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	scope := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")
	scopePID := e.Spawn(scope.producer(), "scope")

	resolverPID := e.Spawn(NewTrendFollowingEntryResolverActor(StrategyResolverConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		StrategyPublisherPID: pubPID,
		ScopePID:             scopePID,
	}), "resolver")

	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol:             "btcusdt",
		DecisionType:       "ema_crossover",
		DecisionOutcome:    "triggered",
		DecisionConfidence: "0.7500",
		DecisionSeverity:   "moderate",
		DecisionRationale:  "EMA crossover bullish confirmation",
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
	if fanout.StrategyType != "trend_following_entry" {
		t.Errorf("strategy type in fan-out: want trend_following_entry, got %s", fanout.StrategyType)
	}
	if fanout.DecisionSeverity != "moderate" {
		t.Errorf("decision severity in fan-out: want moderate, got %s", fanout.DecisionSeverity)
	}
	if fanout.DecisionRationale != "EMA crossover bullish confirmation" {
		t.Errorf("decision rationale in fan-out: want EMA crossover bullish confirmation, got %s", fanout.DecisionRationale)
	}
	if fanout.StrategyDirection != "long" {
		t.Errorf("strategy direction in fan-out: want long, got %s", fanout.StrategyDirection)
	}
	if fanout.CorrelationID != "corr-fanout" {
		t.Errorf("correlationID in fan-out: want corr-fanout, got %s", fanout.CorrelationID)
	}
}
