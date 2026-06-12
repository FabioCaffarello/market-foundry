package bybitf_test

import (
	"testing"

	"internal/adapters/exchanges/bybitf"
	"internal/domain/instrument"
)

const sampleFrame = `{"topic":"publicTrade.BTCUSDT","type":"snapshot","ts":1710000001000,"data":[
	{"T":1710000001001,"s":"BTCUSDT","S":"Buy","v":"0.010","p":"65000.50","i":"trade-1"},
	{"T":1710000001002,"s":"BTCUSDT","S":"Sell","v":"0.020","p":"65000.40","i":"trade-2"}
]}`

func TestNormalize_PerpetualBatch(t *testing.T) {
	frame, ok, prob := bybitf.ParsePublicTrade([]byte(sampleFrame))
	if prob != nil || !ok {
		t.Fatalf("parse: ok=%v prob=%v", ok, prob)
	}

	events, prob := bybitf.Normalize(frame, "btcusdt")
	if prob != nil {
		t.Fatalf("normalize: %v", prob)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	want, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("New: %v", prob)
	}
	for i, ev := range events {
		if ev.Trade.Source != "bybitf" {
			t.Errorf("event %d source = %q, want bybitf", i, ev.Trade.Source)
		}
		if ev.Trade.Instrument != want {
			t.Errorf("event %d instrument = %+v, want %+v (linear = perpetual)", i, ev.Trade.Instrument, want)
		}
	}
	// Taker-side inversion, same contract as the spot adapter.
	if events[0].Trade.BuyerMaker || !events[1].Trade.BuyerMaker {
		t.Error("BuyerMaker mapping: want S=Buy→false, S=Sell→true")
	}
}

// Delivery futures (dash-separated expiry) stay gated by G10 until
// expiry modeling lands in H-7.c — rejected at the parser, never a
// silently-wrong perpetual identity.
func TestNormalize_DeliveryFuturesRejected(t *testing.T) {
	frame := bybitf.PublicTradeFrame{
		Topic: "publicTrade.BTCUSDT-29MAR24",
		Data: []bybitf.PublicTrade{
			{TradeTime: 1710000001001, Symbol: "BTCUSDT-29MAR24", Side: "Buy", Quantity: "1", Price: "1", TradeID: "x"},
		},
	}
	if _, prob := bybitf.Normalize(frame, "btcusdt-29mar24"); prob == nil {
		t.Fatal("delivery symbol must be rejected (G10 gate)")
	}
}

func TestNormalize_RejectsNonUSDTQuote(t *testing.T) {
	frame := bybitf.PublicTradeFrame{
		Topic: "publicTrade.BTCUSD",
		Data: []bybitf.PublicTrade{
			{TradeTime: 1710000001001, Symbol: "BTCUSD", Side: "Buy", Quantity: "1", Price: "1", TradeID: "x"},
		},
	}
	if _, prob := bybitf.Normalize(frame, "btcusd"); prob == nil {
		t.Fatal("non-USDT (inverse-style) symbol must be rejected on the linear adapter")
	}
}

func TestParsePublicTrade_ControlFramesSkipped(t *testing.T) {
	for _, raw := range []string{
		`{"success":true,"ret_msg":"subscribe","op":"subscribe"}`,
		`{"topic":"tickers.BTCUSDT","type":"snapshot","ts":1,"data":{"symbol":"BTCUSDT"}}`,
	} {
		if _, ok, prob := bybitf.ParsePublicTrade([]byte(raw)); ok || prob != nil {
			t.Errorf("frame %q must skip silently (ok=%v prob=%v)", raw, ok, prob)
		}
	}
}

func TestCapabilities_Declaration(t *testing.T) {
	c := bybitf.Capabilities()
	if prob := c.Validate(); prob != nil {
		t.Fatalf("declaration incoherent: %v", prob)
	}
	if c.Venue != instrument.VenueBybitFutures {
		t.Errorf("venue = %q, want %q", c.Venue, instrument.VenueBybitFutures)
	}
	if !c.Allows("observation.trade", instrument.ContractPerpetual) {
		t.Error("observation.trade/perpetual must be declared")
	}
	for _, undeclared := range []instrument.ContractType{
		instrument.ContractSpot,
		instrument.ContractUSDTFutures,
		instrument.ContractCoinFutures,
	} {
		if c.Allows("observation.trade", undeclared) {
			t.Errorf("%s must NOT be declared on the linear perpetual adapter", undeclared)
		}
	}
	if _, ok := c.Notes["delivery"]; !ok {
		t.Error("delivery G10 gating note must be carried in the declaration")
	}
}

func TestWSClient_TopicAndURL(t *testing.T) {
	client := bybitf.NewWSClient("ethusdt", func([]byte) {}, nil)
	if got := client.StreamURL(); got != "wss://stream.bybit.com/v5/public/linear#publicTrade.ETHUSDT" {
		t.Errorf("StreamURL = %q", got)
	}
}
