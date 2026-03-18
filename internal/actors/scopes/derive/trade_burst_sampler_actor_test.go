package derive

import (
	"testing"
	"time"
)

func TestTradeBurstSamplerActor_WindowFinalization_Publishes(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	burstPID := e.Spawn(NewTradeBurstSamplerActor(TradeBurstSamplerConfig{
		Source:       "binancef",
		Symbol:       "btcusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubPID,
	}), "burst-sampler")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	// 2 trades in window 0: one buy, one sell.
	e.Send(burstPID, tradeReceivedMessage{Event: makeTradeWithSide(base, 0, "100.00", "1.0", true)})
	e.Send(burstPID, tradeReceivedMessage{Event: makeTradeWithSide(base, 5*time.Second, "101.00", "2.0", false)})

	// Trigger finalization.
	e.Send(burstPID, tradeReceivedMessage{Event: makeTrade(base, 60*time.Second, "110.00", "1.0")})

	publisher.waitFor(t, 1, 2*time.Second)

	msg, ok := publisher.messages()[0].(publishTradeBurstMessage)
	if !ok {
		t.Fatalf("expected publishTradeBurstMessage, got %T", publisher.messages()[0])
	}

	b := msg.Event.TradeBurst
	if b.Source != "binancef" || b.Symbol != "btcusdt" {
		t.Errorf("wrong source/symbol: %s/%s", b.Source, b.Symbol)
	}
	if b.Timeframe != 60 {
		t.Errorf("timeframe: want 60, got %d", b.Timeframe)
	}
	if !b.Final {
		t.Error("expected final=true")
	}
	if b.TradeCount != 2 {
		t.Errorf("trade_count: want 2, got %d", b.TradeCount)
	}
	// BuyVolume = 100*1 = 100, SellVolume = 101*2 = 202.
	if b.BuyVolume != "100.00000000" {
		t.Errorf("buy_volume: want 100.00000000, got %s", b.BuyVolume)
	}
	if b.SellVolume != "202.00000000" {
		t.Errorf("sell_volume: want 202.00000000, got %s", b.SellVolume)
	}
	if prob := b.Validate(); prob != nil {
		t.Errorf("trade burst validation failed: %s", prob.Message)
	}
}

func TestTradeBurstSamplerActor_BurstDetection(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	burstPID := e.Spawn(NewTradeBurstSamplerActor(TradeBurstSamplerConfig{
		Source:       "binancef",
		Symbol:       "btcusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubPID,
	}), "burst-sampler")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	// Window 0: 2 trades (baseline).
	e.Send(burstPID, tradeReceivedMessage{Event: makeTrade(base, 0, "100.00", "1.0")})
	e.Send(burstPID, tradeReceivedMessage{Event: makeTrade(base, 10*time.Second, "101.00", "1.0")})

	// Window 1: 5 trades (>2x baseline → burst).
	for i := 0; i < 5; i++ {
		e.Send(burstPID, tradeReceivedMessage{Event: makeTrade(base, 60*time.Second+time.Duration(i)*time.Second, "110.00", "1.0")})
	}

	// Window 2: trigger finalization of window 1.
	e.Send(burstPID, tradeReceivedMessage{Event: makeTrade(base, 120*time.Second, "120.00", "1.0")})

	publisher.waitFor(t, 2, 2*time.Second)

	msgs := publisher.messages()

	// Window 0 (no previous baseline → no burst).
	b0 := msgs[0].(publishTradeBurstMessage).Event.TradeBurst
	if b0.Burst {
		t.Error("window 0 should not be burst (no baseline)")
	}

	// Window 1 (5 > 2*2=4 → burst).
	b1 := msgs[1].(publishTradeBurstMessage).Event.TradeBurst
	if !b1.Burst {
		t.Error("window 1 should be burst (5 > 2*2)")
	}
}

func TestTradeBurstSamplerActor_NoFinalization_BeforeWindowTransition(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	burstPID := e.Spawn(NewTradeBurstSamplerActor(TradeBurstSamplerConfig{
		Source:       "binancef",
		Symbol:       "btcusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubPID,
	}), "burst-sampler")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	// Multiple trades within the same window — no finalization.
	for i := 0; i < 10; i++ {
		e.Send(burstPID, tradeReceivedMessage{Event: makeTrade(base, time.Duration(i)*time.Second, "100.00", "1.0")})
	}

	time.Sleep(200 * time.Millisecond)
	if publisher.count() != 0 {
		t.Fatalf("expected 0 publish messages within same window, got %d", publisher.count())
	}
}
