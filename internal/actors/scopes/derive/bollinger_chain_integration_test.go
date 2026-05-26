package derive

import (
	"strconv"
	"testing"
	"time"
)

// bollinger_chain_integration_test.go validates the Bollinger signal → decision
// actor chain wired through real actor instances with msgCollectors capturing output.

func TestActorChain_BollingerSignalSampler_ProducesSignal(t *testing.T) {
	e := newTestEngine(t)

	signalPub := newMsgCollector()
	scopeFanout := newMsgCollector()
	signalPubPID := e.Spawn(signalPub.producer(), "signal-pub")
	scopeFanoutPID := e.Spawn(scopeFanout.producer(), "scope-fanout")

	samplerPID := e.Spawn(NewBollingerSignalSamplerActor(SignalSamplerConfig{
		Source:             "binancef",
		Symbol:             "btcusdt",
		Timeframe:          60 * time.Second,
		SignalPublisherPID: signalPubPID,
		ScopePID:           scopeFanoutPID,
	}), "bollinger-sampler")

	time.Sleep(50 * time.Millisecond)

	// BollingerSampler needs 20 candles (period=20) before emitting a signal.
	// Feed 20 candles with varying prices to produce a meaningful signal.
	base := windowBase()
	for i := 0; i < 20; i++ {
		price := 100.0 + float64(i)*0.5
		e.Send(samplerPID, candleFinalizedMessage{
			Symbol:        "btcusdt",
			ClosePrice:    formatPrice(price),
			Timestamp:     base.Add(time.Duration(i) * time.Minute),
			CorrelationID: "boll-warmup",
		})
	}

	signalPub.waitFor(t, 1, 3*time.Second)
	scopeFanout.waitFor(t, 1, 3*time.Second)

	// Verify published signal.
	pubMsg := signalPub.messages()[0].(publishSignalMessage)
	sig := pubMsg.Event.Signal
	if sig.Type != "bollinger" {
		t.Fatalf("signal type: want bollinger, got %s", sig.Type)
	}
	if sig.VenueSymbol() != "btcusdt" {
		t.Fatalf("signal symbol: want btcusdt, got %s", sig.VenueSymbol())
	}
	if sig.Value == "" {
		t.Fatal("expected non-empty signal value (%B)")
	}
	if sig.Metadata["bandwidth"] == "" {
		t.Fatal("expected bandwidth in signal metadata")
	}
	if sig.Metadata["sma"] == "" {
		t.Fatal("expected sma in signal metadata")
	}
	if sig.Metadata["upper"] == "" {
		t.Fatal("expected upper in signal metadata")
	}
	if sig.Metadata["lower"] == "" {
		t.Fatal("expected lower in signal metadata")
	}
	if sig.Metadata["period"] != "20" {
		t.Errorf("expected period=20, got %s", sig.Metadata["period"])
	}

	// Verify fan-out message shape.
	fanoutMsg, ok := scopeFanout.messages()[0].(signalGeneratedMessage)
	if !ok {
		t.Fatalf("expected signalGeneratedMessage in fan-out, got %T", scopeFanout.messages()[0])
	}
	if fanoutMsg.SignalType != "bollinger" {
		t.Errorf("fan-out signal type: want bollinger, got %s", fanoutMsg.SignalType)
	}
	if fanoutMsg.CorrelationID != "boll-warmup" {
		t.Errorf("correlationID: want boll-warmup, got %s", fanoutMsg.CorrelationID)
	}
}

func TestActorChain_BollingerSignal_To_BollingerSqueezeDecision(t *testing.T) {
	e := newTestEngine(t)

	// Set up collectors.
	signalPub := newMsgCollector()
	decisionPub := newMsgCollector()
	signalFanout := newMsgCollector()
	decFanout := newMsgCollector()

	signalPubPID := e.Spawn(signalPub.producer(), "signal-pub")
	decisionPubPID := e.Spawn(decisionPub.producer(), "decision-pub")
	signalFanoutPID := e.Spawn(signalFanout.producer(), "signal-fanout")
	decFanoutPID := e.Spawn(decFanout.producer(), "dec-fanout")

	// Wire actors.
	samplerPID := e.Spawn(NewBollingerSignalSamplerActor(SignalSamplerConfig{
		Source:             "binancef",
		Symbol:             "btcusdt",
		Timeframe:          60 * time.Second,
		SignalPublisherPID: signalPubPID,
		ScopePID:           signalFanoutPID,
	}), "bollinger-sampler")

	decisionEvalPID := e.Spawn(NewBollingerSqueezeEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "bollinger-squeeze-eval")

	time.Sleep(50 * time.Millisecond)

	// Feed 20 candles with tight prices (low bandwidth → squeeze).
	base := windowBase()
	for i := 0; i < 20; i++ {
		// Very tight range: 100.00 to 100.19 — bandwidth will be small relative to SMA.
		price := 100.0 + float64(i)*0.01
		e.Send(samplerPID, candleFinalizedMessage{
			Symbol:        "btcusdt",
			ClosePrice:    formatPrice(price),
			Timestamp:     base.Add(time.Duration(i) * time.Minute),
			CorrelationID: "boll-squeeze-corr-1",
		})
	}

	signalPub.waitFor(t, 1, 3*time.Second)
	signalFanout.waitFor(t, 1, 3*time.Second)

	// Forward signal to decision evaluator (simulating SourceScopeActor routing).
	fanoutMsg := signalFanout.messages()[0].(signalGeneratedMessage)
	e.Send(decisionEvalPID, fanoutMsg)

	decisionPub.waitFor(t, 1, 3*time.Second)
	decFanout.waitFor(t, 1, 3*time.Second)

	// Verify decision output.
	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if dec.Type != "bollinger_squeeze" {
		t.Fatalf("decision type: want bollinger_squeeze, got %s", dec.Type)
	}
	if string(dec.Outcome) != "triggered" {
		t.Fatalf("decision outcome: want triggered (tight bands → squeeze), got %s", dec.Outcome)
	}
	if string(dec.Severity) == "" {
		t.Fatal("expected decision severity to be set")
	}
	if dec.Confidence == "" {
		t.Fatal("expected decision confidence to be set")
	}

	// Verify fan-out message carries the decision.
	decFanoutMsg, ok := decFanout.messages()[0].(decisionEvaluatedMessage)
	if !ok {
		t.Fatalf("expected decisionEvaluatedMessage, got %T", decFanout.messages()[0])
	}
	if decFanoutMsg.DecisionType != "bollinger_squeeze" {
		t.Errorf("fan-out decision type: want bollinger_squeeze, got %s", decFanoutMsg.DecisionType)
	}
	if decFanoutMsg.CorrelationID != "boll-squeeze-corr-1" {
		t.Errorf("correlationID: want boll-squeeze-corr-1, got %s", decFanoutMsg.CorrelationID)
	}
}

func TestActorChain_BollingerSignal_WideBands_NotTriggered(t *testing.T) {
	e := newTestEngine(t)

	decisionPub := newMsgCollector()
	decFanout := newMsgCollector()
	decisionPubPID := e.Spawn(decisionPub.producer(), "decision-pub")
	decFanoutPID := e.Spawn(decFanout.producer(), "dec-fanout")

	decisionEvalPID := e.Spawn(NewBollingerSqueezeEvaluatorActor(DecisionEvaluatorConfig{
		Source:               "binancef",
		Symbol:               "btcusdt",
		Timeframe:            60 * time.Second,
		DecisionPublisherPID: decisionPubPID,
		ScopePID:             decFanoutPID,
	}), "bollinger-squeeze-eval")

	time.Sleep(50 * time.Millisecond)

	// Inject a bollinger signal with wide bandwidth (no squeeze).
	e.Send(decisionEvalPID, signalGeneratedMessage{
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
		CorrelationID: "boll-wide-corr-1",
	})

	decisionPub.waitFor(t, 1, 2*time.Second)
	decFanout.waitFor(t, 1, 2*time.Second)

	dec := decisionPub.messages()[0].(publishDecisionMessage).Event.Decision
	if dec.Type != "bollinger_squeeze" {
		t.Fatalf("decision type: want bollinger_squeeze, got %s", dec.Type)
	}
	if string(dec.Outcome) != "not_triggered" {
		t.Fatalf("decision outcome: want not_triggered (wide bands), got %s", dec.Outcome)
	}
}

// formatPrice formats a float64 price as a string for candleFinalizedMessage.
func formatPrice(price float64) string {
	return strconv.FormatFloat(price, 'f', 4, 64)
}
