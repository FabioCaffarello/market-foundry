package derive

import (
	"testing"
	"time"
)

func TestRSIOversoldEvaluatorActor_LowRSI_TriggeredWithFanout(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	scope := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")
	scopePID := e.Spawn(scope.producer(), "scope")

	evalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: pubPID,
		ScopePID:             scopePID,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)

	// RSI = 20.0 → below 30 threshold → triggered.
	e.Send(evalPID, signalGeneratedMessage{
		Symbol:        "btcusdt",
		SignalType:    "rsi",
		SignalValue:   "20.0000",
		Timeframe:     60,
		Timestamp:     windowBase(),
		CorrelationID: "corr-1",
	})

	publisher.waitFor(t, 1, 2*time.Second)
	scope.waitFor(t, 1, 2*time.Second)

	// Verify publishDecisionMessage.
	pubMsg, ok := publisher.messages()[0].(publishDecisionMessage)
	if !ok {
		t.Fatalf("expected publishDecisionMessage, got %T", publisher.messages()[0])
	}
	dec := pubMsg.Event.Decision
	if dec.Type != "rsi_oversold" {
		t.Errorf("decision type: want rsi_oversold, got %s", dec.Type)
	}
	if dec.Source != "binancef" {
		t.Errorf("source: want binancef, got %s", dec.Source)
	}
	if dec.Symbol != "btcusdt" {
		t.Errorf("symbol: want btcusdt, got %s", dec.Symbol)
	}
	if string(dec.Outcome) != "triggered" {
		t.Errorf("outcome: want triggered, got %s", dec.Outcome)
	}
	if !dec.Final {
		t.Error("expected final=true")
	}
	if len(dec.Signals) != 1 {
		t.Fatalf("expected 1 signal input, got %d", len(dec.Signals))
	}
	if dec.Signals[0].Type != "rsi" || dec.Signals[0].Value != "20.0000" {
		t.Errorf("signal input: want rsi/20.0000, got %s/%s", dec.Signals[0].Type, dec.Signals[0].Value)
	}
	if prob := dec.Validate(); prob != nil {
		t.Errorf("decision validation failed: %s", prob.Message)
	}

	// Verify decisionEvaluatedMessage to scope.
	scopeMsg, ok := scope.messages()[0].(decisionEvaluatedMessage)
	if !ok {
		t.Fatalf("expected decisionEvaluatedMessage, got %T", scope.messages()[0])
	}
	if scopeMsg.Symbol != "btcusdt" {
		t.Errorf("scope symbol: want btcusdt, got %s", scopeMsg.Symbol)
	}
	if scopeMsg.DecisionOutcome != "triggered" {
		t.Errorf("scope outcome: want triggered, got %s", scopeMsg.DecisionOutcome)
	}
	if scopeMsg.CorrelationID != "corr-1" {
		t.Errorf("scope correlationID: want corr-1, got %s", scopeMsg.CorrelationID)
	}
}

func TestRSIOversoldEvaluatorActor_HighRSI_NotTriggered(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	evalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: pubPID,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)

	// RSI = 75.0 → above 30 threshold → not_triggered.
	e.Send(evalPID, signalGeneratedMessage{
		Symbol:      "btcusdt",
		SignalType:  "rsi",
		SignalValue: "75.0000",
		Timeframe:   60,
		Timestamp:   windowBase(),
	})

	publisher.waitFor(t, 1, 2*time.Second)

	dec := publisher.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "not_triggered" {
		t.Errorf("outcome: want not_triggered, got %s", dec.Outcome)
	}
}

func TestRSIOversoldEvaluatorActor_InvalidSignalValue_NoPanic(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	evalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: pubPID,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)

	// Invalid (non-numeric) signal value → evaluator returns false, no publish.
	e.Send(evalPID, signalGeneratedMessage{
		Symbol:      "btcusdt",
		SignalType:  "rsi",
		SignalValue: "not-a-number",
		Timeframe:   60,
		Timestamp:   windowBase(),
	})

	time.Sleep(200 * time.Millisecond)
	if publisher.count() != 0 {
		t.Fatalf("expected no decision for invalid signal value, got %d", publisher.count())
	}
}

func TestRSIOversoldEvaluatorActor_NilScopePID_PublishesWithoutFanout(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	evalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: pubPID,
		ScopePID:             nil,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)

	e.Send(evalPID, signalGeneratedMessage{
		Symbol:      "btcusdt",
		SignalType:  "rsi",
		SignalValue: "20.0000",
		Timeframe:   60,
		Timestamp:   windowBase(),
	})

	publisher.waitFor(t, 1, 2*time.Second)
	dec := publisher.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "triggered" {
		t.Errorf("outcome: want triggered, got %s", dec.Outcome)
	}
}

func TestRSIOversoldEvaluatorActor_BoundaryRSI_AtThreshold(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	evalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: pubPID,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)

	// RSI exactly at threshold (30.0) → not_triggered (>= threshold).
	e.Send(evalPID, signalGeneratedMessage{
		Symbol:      "btcusdt",
		SignalType:  "rsi",
		SignalValue: "30.0000",
		Timeframe:   60,
		Timestamp:   windowBase(),
	})

	publisher.waitFor(t, 1, 2*time.Second)
	dec := publisher.messages()[0].(publishDecisionMessage).Event.Decision
	if string(dec.Outcome) != "not_triggered" {
		t.Errorf("outcome at threshold: want not_triggered, got %s", dec.Outcome)
	}
}

func TestRSIOversoldEvaluatorActor_MultipleSignals_IndependentEvaluation(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	evalPID := e.Spawn(NewRSIOversoldEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: pubPID,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	// First: low RSI → triggered.
	e.Send(evalPID, signalGeneratedMessage{
		Symbol: "btcusdt", SignalType: "rsi", SignalValue: "15.0000",
		Timeframe: 60, Timestamp: base,
	})
	// Second: high RSI → not_triggered.
	e.Send(evalPID, signalGeneratedMessage{
		Symbol: "btcusdt", SignalType: "rsi", SignalValue: "80.0000",
		Timeframe: 60, Timestamp: base.Add(time.Minute),
	})

	publisher.waitFor(t, 2, 2*time.Second)

	msgs := publisher.messages()
	d0 := msgs[0].(publishDecisionMessage).Event.Decision
	d1 := msgs[1].(publishDecisionMessage).Event.Decision

	if string(d0.Outcome) != "triggered" {
		t.Errorf("first outcome: want triggered, got %s", d0.Outcome)
	}
	if string(d1.Outcome) != "not_triggered" {
		t.Errorf("second outcome: want not_triggered, got %s", d1.Outcome)
	}
}
