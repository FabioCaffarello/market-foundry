package derive

import (
	"testing"
	"time"
)

func TestRSISignalSamplerActor_WarmupPeriod_NoSignal(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	scope := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")
	scopePID := e.Spawn(scope.producer(), "scope")

	rsiPID := e.Spawn(NewRSISignalSamplerActor(SignalSamplerConfig{
		Source:             "binancef",
		Symbol:             "btcusdt",
		Timeframe:          60 * time.Second,
		SignalPublisherPID: pubPID,
		ScopePID:           scopePID,
	}), "rsi-sampler")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	// RSI requires period+1=15 candle closes to produce first signal.
	// Send only 14 → no signal should be emitted.
	for i := 0; i < 14; i++ {
		e.Send(rsiPID, candleFinalizedMessage{
			Symbol:     "btcusdt",
			Timeframe:  60,
			ClosePrice: "100.00",
			Timestamp:  base.Add(time.Duration(i) * time.Minute),
		})
	}

	time.Sleep(200 * time.Millisecond)
	if publisher.count() != 0 {
		t.Fatalf("expected no signals during warmup, got %d", publisher.count())
	}
	if scope.count() != 0 {
		t.Fatalf("expected no scope fan-out during warmup, got %d", scope.count())
	}
}

func TestRSISignalSamplerActor_AfterWarmup_ProducesSignalAndFansOut(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	scope := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")
	scopePID := e.Spawn(scope.producer(), "scope")

	rsiPID := e.Spawn(NewRSISignalSamplerActor(SignalSamplerConfig{
		Source:             "binancef",
		Symbol:             "btcusdt",
		Timeframe:          60 * time.Second,
		SignalPublisherPID: pubPID,
		ScopePID:           scopePID,
	}), "rsi-sampler")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	// Send 15 candles (period+1) — first signal at candle 15.
	for i := 0; i < 15; i++ {
		e.Send(rsiPID, candleFinalizedMessage{
			Symbol:        "btcusdt",
			Timeframe:     60,
			ClosePrice:    "100.00",
			Timestamp:     base.Add(time.Duration(i) * time.Minute),
			CorrelationID: "corr-test",
		})
	}

	publisher.waitFor(t, 1, 2*time.Second)
	scope.waitFor(t, 1, 2*time.Second)

	// Verify publishSignalMessage.
	pubMsg, ok := publisher.messages()[0].(publishSignalMessage)
	if !ok {
		t.Fatalf("expected publishSignalMessage, got %T", publisher.messages()[0])
	}
	sig := pubMsg.Event.Signal
	if sig.Type != "rsi" {
		t.Errorf("signal type: want rsi, got %s", sig.Type)
	}
	if sig.Source != "binancef" {
		t.Errorf("signal source: want binancef, got %s", sig.Source)
	}
	if sig.VenueSymbol() != "btcusdt" {
		t.Errorf("signal symbol: want btcusdt, got %s", sig.VenueSymbol())
	}
	if sig.Timeframe != 60 {
		t.Errorf("signal timeframe: want 60, got %d", sig.Timeframe)
	}
	if !sig.Final {
		t.Error("expected final=true")
	}
	if sig.Value == "" {
		t.Error("signal value should not be empty")
	}
	if prob := sig.Validate(); prob != nil {
		t.Errorf("signal validation failed: %s", prob.Message)
	}

	// Verify signalGeneratedMessage to scope.
	scopeMsg, ok := scope.messages()[0].(signalGeneratedMessage)
	if !ok {
		t.Fatalf("expected signalGeneratedMessage, got %T", scope.messages()[0])
	}
	if scopeMsg.Symbol != "btcusdt" {
		t.Errorf("scope symbol: want btcusdt, got %s", scopeMsg.Symbol)
	}
	if scopeMsg.SignalType != "rsi" {
		t.Errorf("scope signal type: want rsi, got %s", scopeMsg.SignalType)
	}
	if scopeMsg.CorrelationID != "corr-test" {
		t.Errorf("scope correlationID: want corr-test, got %s", scopeMsg.CorrelationID)
	}
}

func TestRSISignalSamplerActor_SubsequentCandles_ProduceSignals(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	rsiPID := e.Spawn(NewRSISignalSamplerActor(SignalSamplerConfig{
		Source:             "binancef",
		Symbol:             "btcusdt",
		Timeframe:          60 * time.Second,
		SignalPublisherPID: pubPID,
	}), "rsi-sampler")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	// Send 17 candles: 15 for warm-up, 2 more → 3 total signals.
	for i := 0; i < 17; i++ {
		e.Send(rsiPID, candleFinalizedMessage{
			Symbol:     "btcusdt",
			Timeframe:  60,
			ClosePrice: "100.00",
			Timestamp:  base.Add(time.Duration(i) * time.Minute),
		})
	}

	publisher.waitFor(t, 3, 2*time.Second)
	if publisher.count() != 3 {
		t.Errorf("expected 3 signals (15th, 16th, 17th candle), got %d", publisher.count())
	}
}

func TestRSISignalSamplerActor_NilScopePID_PublishesWithoutFanout(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	rsiPID := e.Spawn(NewRSISignalSamplerActor(SignalSamplerConfig{
		Source:             "binancef",
		Symbol:             "btcusdt",
		Timeframe:          60 * time.Second,
		SignalPublisherPID: pubPID,
		ScopePID:           nil,
	}), "rsi-sampler")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	for i := 0; i < 15; i++ {
		e.Send(rsiPID, candleFinalizedMessage{
			Symbol:     "btcusdt",
			Timeframe:  60,
			ClosePrice: "100.00",
			Timestamp:  base.Add(time.Duration(i) * time.Minute),
		})
	}

	publisher.waitFor(t, 1, 2*time.Second)

	// No panic from nil ScopePID, signal was published.
	sig := publisher.messages()[0].(publishSignalMessage).Event.Signal
	if sig.Type != "rsi" {
		t.Errorf("expected rsi, got %s", sig.Type)
	}
}
