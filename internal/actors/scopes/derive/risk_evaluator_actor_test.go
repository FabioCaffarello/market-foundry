package derive

import (
	"testing"
	"time"
)

func TestPositionExposureEvaluatorActor_Long_Approved(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	evalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: pubPID,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)

	e.Send(evalPID, strategyResolvedMessage{
		Symbol:             "btcusdt",
		StrategyType:       "mean_reversion_entry",
		StrategyDirection:  "long",
		StrategyConfidence: "0.7500",
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
	if r.Type != "position_exposure" {
		t.Errorf("risk type: want position_exposure, got %s", r.Type)
	}
	if r.Source != "binancef" {
		t.Errorf("source: want binancef, got %s", r.Source)
	}
	if r.Symbol != "btcusdt" {
		t.Errorf("symbol: want btcusdt, got %s", r.Symbol)
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
	if r.Constraints.MaxPositionSize == "" {
		t.Error("expected max_position_size constraint")
	}

	if prob := r.Validate(); prob != nil {
		t.Errorf("risk validation failed: %s", prob.Message)
	}

	// Verify correlation ID propagated.
	if msg.Event.Metadata.CorrelationID != "corr-risk" {
		t.Errorf("correlationID: want corr-risk, got %s", msg.Event.Metadata.CorrelationID)
	}
}

func TestPositionExposureEvaluatorActor_Flat_Approved(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	evalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: pubPID,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)

	e.Send(evalPID, strategyResolvedMessage{
		Symbol:             "btcusdt",
		StrategyType:       "mean_reversion_entry",
		StrategyDirection:  "flat",
		StrategyConfidence: "0.0000",
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

func TestPositionExposureEvaluatorActor_UnknownDirection_NoPublish(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	evalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
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

func TestPositionExposureEvaluatorActor_SequentialStrategies_Independent(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	evalPID := e.Spawn(NewPositionExposureEvaluatorActor(RiskEvaluatorConfig{
		Source:           "binancef",
		Symbol:           "btcusdt",
		Timeframe:        60 * time.Second,
		RiskPublisherPID: pubPID,
	}), "evaluator")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	// Long → Approved.
	e.Send(evalPID, strategyResolvedMessage{
		Symbol: "btcusdt", StrategyType: "mean_reversion_entry", StrategyDirection: "long",
		StrategyConfidence: "0.8000", Timeframe: 60, Timestamp: base,
	})
	// Flat → Approved.
	e.Send(evalPID, strategyResolvedMessage{
		Symbol: "btcusdt", StrategyType: "mean_reversion_entry", StrategyDirection: "flat",
		StrategyConfidence: "0.0000", Timeframe: 60, Timestamp: base.Add(time.Minute),
	})

	publisher.waitFor(t, 2, 2*time.Second)

	msgs := publisher.messages()
	r0 := msgs[0].(publishRiskMessage).Event.RiskAssessment
	r1 := msgs[1].(publishRiskMessage).Event.RiskAssessment

	if string(r0.Disposition) != "approved" {
		t.Errorf("first disposition: want approved, got %s", r0.Disposition)
	}
	if string(r1.Disposition) != "approved" {
		t.Errorf("second disposition: want approved, got %s", r1.Disposition)
	}
}
