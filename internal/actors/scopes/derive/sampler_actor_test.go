package derive

import (
	"testing"
	"time"

	"internal/domain/evidence"
)

func TestSamplerActor_WindowFinalization_PublishesAndFansOut(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	scope := newMsgCollector()

	pubPID := e.Spawn(publisher.producer(), "pub")
	scopePID := e.Spawn(scope.producer(), "scope")

	samplerPID := e.Spawn(NewSamplerActor(SamplerConfig{
		Source:       "binancef",
		Symbol:       "btcusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubPID,
		ScopePID:     scopePID,
	}), "sampler")

	time.Sleep(50 * time.Millisecond) // allow Started to run

	base := windowBase()

	// Trade in window 0 (0:00:00 – 0:00:59).
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 0, "100.00", "1.0")})
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 10*time.Second, "105.00", "0.5")})

	// No finalization yet — still in window 0.
	time.Sleep(100 * time.Millisecond)
	if publisher.count() != 0 {
		t.Fatal("expected no publish before window transition")
	}

	// Trade in window 1 → finalizes window 0.
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 60*time.Second, "110.00", "2.0")})

	publisher.waitFor(t, 1, 2*time.Second)
	scope.waitFor(t, 1, 2*time.Second)

	// Verify publishCandleMessage.
	pubMsgs := publisher.messages()
	candleMsg, ok := pubMsgs[0].(publishCandleMessage)
	if !ok {
		t.Fatalf("expected publishCandleMessage, got %T", pubMsgs[0])
	}
	c := candleMsg.Event.Candle
	if c.Source != "binancef" {
		t.Errorf("source: want binancef, got %s", c.Source)
	}
	if c.Symbol != "btcusdt" {
		t.Errorf("symbol: want btcusdt, got %s", c.Symbol)
	}
	if c.Timeframe != 60 {
		t.Errorf("timeframe: want 60, got %d", c.Timeframe)
	}
	if !c.Final {
		t.Error("expected final=true")
	}
	if c.TradeCount != 2 {
		t.Errorf("trade_count: want 2, got %d", c.TradeCount)
	}
	if c.Open != "100.00000000" {
		t.Errorf("open: want 100.00000000, got %s", c.Open)
	}
	if c.High != "105.00000000" {
		t.Errorf("high: want 105.00000000, got %s", c.High)
	}
	if c.Low != "100.00000000" {
		t.Errorf("low: want 100.00000000, got %s", c.Low)
	}
	if c.Close != "105.00000000" {
		t.Errorf("close: want 105.00000000, got %s", c.Close)
	}

	// Verify candleFinalizedMessage to scope.
	scopeMsgs := scope.messages()
	fanout, ok := scopeMsgs[0].(candleFinalizedMessage)
	if !ok {
		t.Fatalf("expected candleFinalizedMessage, got %T", scopeMsgs[0])
	}
	if fanout.Symbol != "btcusdt" {
		t.Errorf("fanout symbol: want btcusdt, got %s", fanout.Symbol)
	}
	if fanout.Timeframe != 60 {
		t.Errorf("fanout timeframe: want 60, got %d", fanout.Timeframe)
	}
	if fanout.ClosePrice != "105.00000000" {
		t.Errorf("fanout close_price: want 105.00000000, got %s", fanout.ClosePrice)
	}
}

func TestSamplerActor_NilScopePID_PublishesWithoutFanout(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	samplerPID := e.Spawn(NewSamplerActor(SamplerConfig{
		Source:       "binancef",
		Symbol:       "btcusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubPID,
		ScopePID:     nil, // no scope
	}), "sampler")

	time.Sleep(50 * time.Millisecond)

	base := windowBase()
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 0, "100.00", "1.0")})
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 60*time.Second, "110.00", "1.0")})

	publisher.waitFor(t, 1, 2*time.Second)

	candleMsg, ok := publisher.messages()[0].(publishCandleMessage)
	if !ok {
		t.Fatalf("expected publishCandleMessage, got %T", publisher.messages()[0])
	}
	if !candleMsg.Event.Candle.Final {
		t.Error("expected final=true")
	}
}

func TestSamplerActor_MultipleWindows_ProducesMultipleCandles(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	samplerPID := e.Spawn(NewSamplerActor(SamplerConfig{
		Source:       "binancef",
		Symbol:       "btcusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubPID,
		ScopePID:     nil,
	}), "sampler")

	time.Sleep(50 * time.Millisecond)

	base := windowBase()
	// 3 windows → 2 finalizations (window 0 finalized by trade in window 1, window 1 by trade in window 2).
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 0, "100.00", "1.0")})
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 60*time.Second, "200.00", "1.0")})
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 120*time.Second, "300.00", "1.0")})

	publisher.waitFor(t, 2, 2*time.Second)

	msgs := publisher.messages()
	c0 := msgs[0].(publishCandleMessage).Event.Candle
	c1 := msgs[1].(publishCandleMessage).Event.Candle

	if c0.Open != "100.00000000" || c1.Open != "200.00000000" {
		t.Errorf("window sequence mismatch: c0.Open=%s, c1.Open=%s", c0.Open, c1.Open)
	}

	// Verify candles have correct OpenTime monotonicity.
	if !c1.OpenTime.After(c0.OpenTime) {
		t.Errorf("expected c1.OpenTime > c0.OpenTime: %v vs %v", c1.OpenTime, c0.OpenTime)
	}
}

func TestSamplerActor_CandleValidation_CorrectOHLCV(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	samplerPID := e.Spawn(NewSamplerActor(SamplerConfig{
		Source:       "binancef",
		Symbol:       "ethusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubPID,
		ScopePID:     nil,
	}), "sampler")

	time.Sleep(50 * time.Millisecond)

	base := windowBase()
	// Specific OHLCV: O=50, H=80, L=40, C=70 (3 trades)
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 0, "50.00", "1.0")})
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 10*time.Second, "80.00", "1.0")})
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 20*time.Second, "40.00", "1.0")})
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 30*time.Second, "70.00", "1.0")})
	// Trigger finalization.
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 60*time.Second, "100.00", "1.0")})

	publisher.waitFor(t, 1, 2*time.Second)
	c := publisher.messages()[0].(publishCandleMessage).Event.Candle

	assertCandleField(t, "open", c.Open, "50.00000000")
	assertCandleField(t, "high", c.High, "80.00000000")
	assertCandleField(t, "low", c.Low, "40.00000000")
	assertCandleField(t, "close", c.Close, "70.00000000")
	if c.TradeCount != 4 {
		t.Errorf("trade_count: want 4, got %d", c.TradeCount)
	}
	if prob := c.Validate(); prob != nil {
		t.Errorf("candle validation failed: %s", prob.Message)
	}
}

func assertCandleField(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("candle.%s: want %s, got %s", field, want, got)
	}
}

func TestSamplerActor_CorrelationID_PropagatedToFanout(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	scope := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")
	scopePID := e.Spawn(scope.producer(), "scope")

	samplerPID := e.Spawn(NewSamplerActor(SamplerConfig{
		Source:       "binancef",
		Symbol:       "btcusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubPID,
		ScopePID:     scopePID,
	}), "sampler")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	// The correlation ID should come from the trade that triggers finalization
	// (the trade that opens the new window).
	trade2 := makeTrade(base, 60*time.Second, "110.00", "1.0")
	corrID := trade2.Metadata.CorrelationID

	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 0, "100.00", "1.0")})
	e.Send(samplerPID, tradeReceivedMessage{Event: trade2})

	publisher.waitFor(t, 1, 2*time.Second)
	scope.waitFor(t, 1, 2*time.Second)

	candleMsg := publisher.messages()[0].(publishCandleMessage)
	if candleMsg.Event.Metadata.CorrelationID != corrID {
		t.Errorf("publisher correlationID: want %s, got %s", corrID, candleMsg.Event.Metadata.CorrelationID)
	}

	fanout := scope.messages()[0].(candleFinalizedMessage)
	if fanout.CorrelationID != corrID {
		t.Errorf("scope correlationID: want %s, got %s", corrID, fanout.CorrelationID)
	}
}

func TestSamplerActor_SymbolIsolation_DifferentSymbolsNoBleed(t *testing.T) {
	e := newTestEngine(t)

	pubBTC := newMsgCollector()
	pubETH := newMsgCollector()
	pubBTCPID := e.Spawn(pubBTC.producer(), "pub-btc")
	pubETHPID := e.Spawn(pubETH.producer(), "pub-eth")

	btcPID := e.Spawn(NewSamplerActor(SamplerConfig{
		Source:       "binancef",
		Symbol:       "btcusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubBTCPID,
	}), "sampler-btc")

	ethPID := e.Spawn(NewSamplerActor(SamplerConfig{
		Source:       "binancef",
		Symbol:       "ethusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubETHPID,
	}), "sampler-eth")

	time.Sleep(50 * time.Millisecond)

	base := windowBase()

	// Feed only BTC trades.
	e.Send(btcPID, tradeReceivedMessage{Event: makeTrade(base, 0, "100.00", "1.0")})
	e.Send(btcPID, tradeReceivedMessage{Event: makeTrade(base, 60*time.Second, "200.00", "1.0")})

	pubBTC.waitFor(t, 1, 2*time.Second)

	// ETH publisher should have received nothing.
	time.Sleep(100 * time.Millisecond)
	if pubETH.count() != 0 {
		t.Fatal("ETH publisher received messages despite no ETH trades — cross-symbol bleed")
	}

	// Verify BTC candle has correct symbol.
	c := pubBTC.messages()[0].(publishCandleMessage).Event.Candle
	if c.Symbol != "btcusdt" {
		t.Errorf("expected btcusdt, got %s", c.Symbol)
	}

	_ = ethPID // used to prevent compiler error
}

func TestSamplerActor_FinalizedCandle_PassesValidation(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	samplerPID := e.Spawn(NewSamplerActor(SamplerConfig{
		Source:       "binancef",
		Symbol:       "btcusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubPID,
	}), "sampler")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 0, "100.00", "1.0")})
	e.Send(samplerPID, tradeReceivedMessage{Event: makeTrade(base, 60*time.Second, "200.00", "1.0")})

	publisher.waitFor(t, 1, 2*time.Second)

	candle := publisher.messages()[0].(publishCandleMessage).Event.Candle
	if prob := candle.Validate(); prob != nil {
		t.Errorf("finalized candle fails domain validation: %s", prob.Message)
	}

	// Verify it would pass store final-gate.
	if !candle.Final {
		t.Error("expected Final=true on finalized candle")
	}

	// Verify OpenTime < CloseTime (window invariant).
	if !candle.CloseTime.After(candle.OpenTime) {
		t.Errorf("expected CloseTime > OpenTime: %v vs %v", candle.CloseTime, candle.OpenTime)
	}

	// Verify the output is a valid evidence.EvidenceCandle (struct contract).
	_ = evidence.EvidenceCandle(candle)
}
