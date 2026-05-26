package derive

import (
	"testing"
	"time"
)

func TestDrawdownLimitEvaluatorActor_Long_Approved(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	evalPID := e.Spawn(NewDrawdownLimitEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Instrument:       btcUSDTPerp(),
		Timeframe:        60 * time.Second,
		RiskPublisherPID: pubPID,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)

	e.Send(evalPID, strategyResolvedMessage{
		Symbol:             "btcusdt",
		StrategyType:       "mean_reversion_entry",
		StrategyDirection:  "long",
		StrategyConfidence: "0.7500",
		DecisionSeverity:   "low",
		DecisionRationale:  "RSI 28.50 below threshold",
		Timeframe:          60,
		Timestamp:          windowBase(),
		CorrelationID:      "corr-risk",
	})

	publisher.waitFor(t, 1, 2*time.Second)

	msg, ok := publisher.messages()[0].(publishRiskMessage)
	if !ok {
		t.Fatalf("expected publishRiskMessage, got %T", publisher.messages()[0])
	}

	r := msg.Event.RiskAssessment
	if r.Type != "drawdown_limit" {
		t.Errorf("risk type: want drawdown_limit, got %s", r.Type)
	}
	if r.Source != "binancef" {
		t.Errorf("source: want binancef, got %s", r.Source)
	}
	if r.VenueSymbol() != "btcusdt" {
		t.Errorf("symbol: want btcusdt, got %s", r.VenueSymbol())
	}
	if string(r.Disposition) != "approved" {
		t.Errorf("disposition: want approved, got %s", r.Disposition)
	}
	if !r.Final {
		t.Error("expected final=true")
	}
	if len(r.Strategies) != 1 {
		t.Fatalf("expected 1 strategy input, got %d", len(r.Strategies))
	}
	if r.Strategies[0].Type != "mean_reversion_entry" || r.Strategies[0].Direction != "long" {
		t.Errorf("strategy input: want mean_reversion_entry/long, got %s/%s", r.Strategies[0].Type, r.Strategies[0].Direction)
	}
	if r.Constraints.StopDistance == "" {
		t.Error("expected stop_distance constraint")
	}

	if r.Strategies[0].DecisionSeverity != "low" {
		t.Errorf("decision severity: want low, got %s", r.Strategies[0].DecisionSeverity)
	}
	if r.Strategies[0].DecisionRationale != "RSI 28.50 below threshold" {
		t.Errorf("decision rationale: want RSI 28.50 below threshold, got %s", r.Strategies[0].DecisionRationale)
	}

	if prob := r.Validate(); prob != nil {
		t.Errorf("risk validation failed: %s", prob.Message)
	}

	if msg.Event.Metadata.CorrelationID != "corr-risk" {
		t.Errorf("correlationID: want corr-risk, got %s", msg.Event.Metadata.CorrelationID)
	}
}

func TestDrawdownLimitEvaluatorActor_Flat_Approved(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	evalPID := e.Spawn(NewDrawdownLimitEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Instrument:       btcUSDTPerp(),
		Timeframe:        60 * time.Second,
		RiskPublisherPID: pubPID,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)

	e.Send(evalPID, strategyResolvedMessage{
		Symbol:             "btcusdt",
		StrategyType:       "mean_reversion_entry",
		StrategyDirection:  "flat",
		StrategyConfidence: "0.0000",
		DecisionSeverity:   "none",
		Timeframe:          60,
		Timestamp:          windowBase(),
	})

	publisher.waitFor(t, 1, 2*time.Second)

	r := publisher.messages()[0].(publishRiskMessage).Event.RiskAssessment
	if string(r.Disposition) != "approved" {
		t.Errorf("disposition: want approved, got %s", r.Disposition)
	}
	if r.Confidence != "1.0000" {
		t.Errorf("confidence: want 1.0000, got %s", r.Confidence)
	}
}

func TestDrawdownLimitEvaluatorActor_UnknownDirection_NoPublish(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	evalPID := e.Spawn(NewDrawdownLimitEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Instrument:       btcUSDTPerp(),
		Timeframe:        60 * time.Second,
		RiskPublisherPID: pubPID,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)

	e.Send(evalPID, strategyResolvedMessage{
		Symbol:             "btcusdt",
		StrategyType:       "mean_reversion_entry",
		StrategyDirection:  "unknown",
		StrategyConfidence: "0.5000",
		Timeframe:          60,
		Timestamp:          windowBase(),
	})

	time.Sleep(200 * time.Millisecond)
	if publisher.count() != 0 {
		t.Fatalf("expected no risk for unknown direction, got %d", publisher.count())
	}
}

func TestDrawdownLimitEvaluatorActor_FanOut_IncludesDecisionSeverity(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	scope := newMsgCollector()
	scopePID := e.Spawn(scope.producer(), "scope")

	evalPID := e.Spawn(NewDrawdownLimitEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Instrument:       btcUSDTPerp(),
		Timeframe:        60 * time.Second,
		RiskPublisherPID: pubPID,
		ScopePID:         scopePID,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)

	e.Send(evalPID, strategyResolvedMessage{
		Symbol:             "btcusdt",
		StrategyType:       "mean_reversion_entry",
		StrategyDirection:  "long",
		StrategyConfidence: "0.7500",
		DecisionSeverity:   "moderate",
		DecisionRationale:  "RSI 20.00 below threshold",
		Timeframe:          60,
		Timestamp:          windowBase(),
		CorrelationID:      "corr-fanout",
	})

	scope.waitFor(t, 1, 2*time.Second)

	fanout, ok := scope.messages()[0].(riskAssessedMessage)
	if !ok {
		t.Fatalf("expected riskAssessedMessage, got %T", scope.messages()[0])
	}

	if fanout.DecisionSeverity != "moderate" {
		t.Errorf("decision severity in fan-out: want moderate, got %s", fanout.DecisionSeverity)
	}
	if fanout.RiskDisposition != "approved" {
		t.Errorf("risk disposition in fan-out: want approved, got %s", fanout.RiskDisposition)
	}
	if fanout.StrategyDirection != "long" {
		t.Errorf("strategy direction in fan-out: want long, got %s", fanout.StrategyDirection)
	}
	if fanout.RiskType != "drawdown_limit" {
		t.Errorf("risk type in fan-out: want drawdown_limit, got %s", fanout.RiskType)
	}
}
