package derive

import (
	"testing"
	"time"
)

func TestVolumeSamplerActor_WindowFinalization_Publishes(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	volPID := e.Spawn(NewVolumeSamplerActor(VolumeSamplerConfig{
		Source:       "binancef",
		Symbol:       "btcusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubPID,
	}), "volume-sampler")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	// 2 trades: buy 100*1=100 notional, sell 200*2=400 notional.
	e.Send(volPID, tradeReceivedMessage{Event: makeTradeWithSide(base, 0, "100.00", "1.0", true)})
	e.Send(volPID, tradeReceivedMessage{Event: makeTradeWithSide(base, 10*time.Second, "200.00", "2.0", false)})

	// Trigger finalization.
	e.Send(volPID, tradeReceivedMessage{Event: makeTrade(base, 60*time.Second, "300.00", "1.0")})

	publisher.waitFor(t, 1, 2*time.Second)

	msg, ok := publisher.messages()[0].(publishVolumeMessage)
	if !ok {
		t.Fatalf("expected publishVolumeMessage, got %T", publisher.messages()[0])
	}

	v := msg.Event.Volume
	if v.Source != "binancef" || v.VenueSymbol() != "btcusdt" {
		t.Errorf("wrong source/symbol: %s/%s", v.Source, v.VenueSymbol())
	}
	if v.Timeframe != 60 {
		t.Errorf("timeframe: want 60, got %d", v.Timeframe)
	}
	if !v.Final {
		t.Error("expected final=true")
	}
	if v.TradeCount != 2 {
		t.Errorf("trade_count: want 2, got %d", v.TradeCount)
	}
	// BuyVol=100, SellVol=400, Total=500, VWAP=500/(1+2)=166.6667
	if v.BuyVolume != "100.00000000" {
		t.Errorf("buy_volume: want 100.00000000, got %s", v.BuyVolume)
	}
	if v.SellVolume != "400.00000000" {
		t.Errorf("sell_volume: want 400.00000000, got %s", v.SellVolume)
	}
	if v.TotalVolume != "500.00000000" {
		t.Errorf("total_volume: want 500.00000000, got %s", v.TotalVolume)
	}
	// VWAP = 500 / 3 = 166.66666666... (rounded to 8 decimals)
	if v.VWAP != "166.66666667" {
		t.Errorf("vwap: want 166.66666667, got %s", v.VWAP)
	}
	if prob := v.Validate(); prob != nil {
		t.Errorf("volume validation failed: %s", prob.Message)
	}
}

func TestVolumeSamplerActor_BuySellSplit_Correct(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	volPID := e.Spawn(NewVolumeSamplerActor(VolumeSamplerConfig{
		Source:       "binancef",
		Symbol:       "ethusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubPID,
	}), "volume-sampler")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	// All buys — sell should be 0.
	e.Send(volPID, tradeReceivedMessage{Event: makeTradeWithSide(base, 0, "50.00", "2.0", true)})
	e.Send(volPID, tradeReceivedMessage{Event: makeTradeWithSide(base, 5*time.Second, "60.00", "3.0", true)})
	e.Send(volPID, tradeReceivedMessage{Event: makeTrade(base, 60*time.Second, "70.00", "1.0")})

	publisher.waitFor(t, 1, 2*time.Second)

	v := publisher.messages()[0].(publishVolumeMessage).Event.Volume
	// BuyVol = 50*2 + 60*3 = 100 + 180 = 280
	if v.BuyVolume != "280.00000000" {
		t.Errorf("buy_volume: want 280.00000000, got %s", v.BuyVolume)
	}
	if v.SellVolume != "0.00000000" {
		t.Errorf("sell_volume: want 0.00000000, got %s", v.SellVolume)
	}
}

func TestVolumeSamplerActor_NoFinalization_BeforeWindowTransition(t *testing.T) {
	e := newTestEngine(t)

	publisher := newMsgCollector()
	pubPID := e.Spawn(publisher.producer(), "pub")

	volPID := e.Spawn(NewVolumeSamplerActor(VolumeSamplerConfig{
		Source:       "binancef",
		Symbol:       "btcusdt",
		Timeframe:    60 * time.Second,
		PublisherPID: pubPID,
	}), "volume-sampler")

	time.Sleep(50 * time.Millisecond)
	base := windowBase()

	for i := 0; i < 5; i++ {
		e.Send(volPID, tradeReceivedMessage{Event: makeTrade(base, time.Duration(i)*time.Second, "100.00", "1.0")})
	}

	time.Sleep(200 * time.Millisecond)
	if publisher.count() != 0 {
		t.Fatalf("expected 0 publish messages within same window, got %d", publisher.count())
	}
}
