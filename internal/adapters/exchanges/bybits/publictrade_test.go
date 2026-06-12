package bybits_test

import (
	"testing"
	"time"

	"internal/adapters/exchanges/bybits"
	"internal/domain/instrument"
)

// Frame shape per Bybit v5 publicTrade docs: data[] batches N trades.
const sampleFrame = `{"topic":"publicTrade.BTCUSDT","type":"snapshot","ts":1710000001000,"data":[
	{"T":1710000001001,"s":"BTCUSDT","S":"Buy","v":"0.010","p":"65000.50","i":"trade-1"},
	{"T":1710000001002,"s":"BTCUSDT","S":"Sell","v":"0.020","p":"65000.40","i":"trade-2"}
]}`

func TestParsePublicTrade_DataFrame(t *testing.T) {
	frame, ok, prob := bybits.ParsePublicTrade([]byte(sampleFrame))
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if !ok {
		t.Fatal("expected data frame, got skip")
	}
	if len(frame.Data) != 2 {
		t.Fatalf("expected 2 trades in frame, got %d", len(frame.Data))
	}
	if frame.Data[0].TradeID != "trade-1" || frame.Data[1].Side != "Sell" {
		t.Errorf("frame fields mismatch: %+v", frame.Data)
	}
}

// Control frames (subscribe acks, pongs) and non-trade topics are
// expected multiplexed traffic on the v5 socket — skipped silently,
// never errors (unlike Binance's URL-per-stream model).
func TestParsePublicTrade_ControlAndNonTradeFramesSkipped(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"subscribe_ack", `{"success":true,"ret_msg":"subscribe","conn_id":"x","op":"subscribe"}`},
		{"pong", `{"success":true,"ret_msg":"pong","op":"ping"}`},
		{"orderbook_topic", `{"topic":"orderbook.50.BTCUSDT","type":"delta","ts":1710000001000,"data":{}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok, prob := bybits.ParsePublicTrade([]byte(tc.raw))
			if prob != nil {
				t.Fatalf("control/non-trade frame must skip silently, got problem: %v", prob)
			}
			if ok {
				t.Fatal("control/non-trade frame must not parse as data")
			}
		})
	}
}

func TestParsePublicTrade_Malformed(t *testing.T) {
	for _, raw := range []string{`not-json`, `{"topic":"publicTrade.BTCUSDT","data":[]}`} {
		if _, ok, prob := bybits.ParsePublicTrade([]byte(raw)); prob == nil || ok {
			t.Errorf("malformed payload %q must return a problem", raw)
		}
	}
}

func TestNormalize_BatchAndTakerSideInversion(t *testing.T) {
	frame, ok, prob := bybits.ParsePublicTrade([]byte(sampleFrame))
	if prob != nil || !ok {
		t.Fatalf("parse: ok=%v prob=%v", ok, prob)
	}

	events, prob := bybits.Normalize(frame, "btcusdt")
	if prob != nil {
		t.Fatalf("normalize: %v", prob)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events (one per trade in frame), got %d", len(events))
	}

	want, prob := instrument.New("BTC", "USDT", instrument.ContractSpot)
	if prob != nil {
		t.Fatalf("New: %v", prob)
	}
	for i, ev := range events {
		if ev.Trade.Source != "bybits" {
			t.Errorf("event %d source = %q, want bybits", i, ev.Trade.Source)
		}
		if ev.Trade.Instrument != want {
			t.Errorf("event %d instrument = %+v, want %+v", i, ev.Trade.Instrument, want)
		}
	}

	// Bybit S is the TAKER side: "Buy" taker → buyer NOT maker;
	// "Sell" taker → buyer was the resting maker.
	if events[0].Trade.BuyerMaker {
		t.Error("S=Buy (taker buy) must map to BuyerMaker=false")
	}
	if !events[1].Trade.BuyerMaker {
		t.Error("S=Sell (taker sell) must map to BuyerMaker=true")
	}

	if got, want := events[0].Trade.Timestamp, time.UnixMilli(1710000001001).UTC(); !got.Equal(want) {
		t.Errorf("timestamp = %v, want %v", got, want)
	}
}

func TestNormalize_RejectsNonUSDTQuote(t *testing.T) {
	frame := bybits.PublicTradeFrame{
		Topic: "publicTrade.BTCEUR",
		Data: []bybits.PublicTrade{
			{TradeTime: 1710000001001, Symbol: "BTCEUR", Side: "Buy", Quantity: "1", Price: "1", TradeID: "x"},
		},
	}
	if _, prob := bybits.Normalize(frame, "btceur"); prob == nil {
		t.Fatal("non-USDT quote must be rejected (bybits is USDT-quoted spot only)")
	}
	if _, prob := bybits.Normalize(frame, ""); prob == nil {
		t.Fatal("empty symbol must be rejected")
	}
}

func TestCapabilities_Declaration(t *testing.T) {
	c := bybits.Capabilities()
	if prob := c.Validate(); prob != nil {
		t.Fatalf("declaration incoherent: %v", prob)
	}
	if c.Venue != instrument.VenueBybit {
		t.Errorf("venue = %q, want %q", c.Venue, instrument.VenueBybit)
	}
	if !c.Allows("observation.trade", instrument.ContractSpot) {
		t.Error("observation.trade/spot must be declared")
	}
	if c.Allows("observation.trade", instrument.ContractPerpetual) {
		t.Error("perpetual must NOT be declared on the spot adapter")
	}
	if c.Allows("observation.orderbook", instrument.ContractSpot) {
		t.Error("orderbook must NOT be declared (trades-only parsing surface)")
	}
}

func TestWSClient_TopicAndURL(t *testing.T) {
	client := bybits.NewWSClient("btcusdt", func([]byte) {}, nil)
	if got := client.StreamURL(); got != "wss://stream.bybit.com/v5/public/spot#publicTrade.BTCUSDT" {
		t.Errorf("StreamURL = %q", got)
	}
	if client.Symbol() != "btcusdt" {
		t.Errorf("Symbol = %q", client.Symbol())
	}
}
