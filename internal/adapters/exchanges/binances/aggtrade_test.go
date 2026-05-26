package binances_test

import (
	"testing"

	"internal/adapters/exchanges/binances"
	"internal/domain/instrument"
)

func TestParseAggTrade(t *testing.T) {
	raw := `{
		"e": "aggTrade",
		"E": 1710000000000,
		"s": "BTCUSDT",
		"a": 4839201,
		"p": "84521.30",
		"q": "0.150",
		"f": 100,
		"l": 105,
		"T": 1710000000123,
		"m": true
	}`

	agg, prob := binances.ParseAggTrade([]byte(raw))
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if agg.Price != "84521.30" {
		t.Fatalf("expected price 84521.30, got %s", agg.Price)
	}
	if agg.AggTradeID != 4839201 {
		t.Fatalf("expected agg trade id 4839201, got %d", agg.AggTradeID)
	}
	if !agg.IsBuyerMaker {
		t.Fatal("expected buyer maker to be true")
	}
}

func TestParseAggTrade_RejectsWrongType(t *testing.T) {
	raw := `{"e": "trade", "s": "BTCUSDT"}`
	_, prob := binances.ParseAggTrade([]byte(raw))
	if prob == nil {
		t.Fatal("expected error for non-aggTrade event type")
	}
}

func TestNormalize(t *testing.T) {
	agg := binances.AggTrade{
		EventType:    "aggTrade",
		EventTime:    1710000000000,
		Symbol:       "BTCUSDT",
		AggTradeID:   4839201,
		Price:        "84521.30",
		Quantity:     "0.150",
		FirstTradeID: 100,
		LastTradeID:  105,
		TradeTime:    1710000000123,
		IsBuyerMaker: false,
	}

	event, prob := binances.Normalize(agg, "btcusdt")
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if event.Trade.Source != "binances" {
		t.Fatalf("expected source binances, got %s", event.Trade.Source)
	}
	if event.Trade.Instrument.Base != "BTC" {
		t.Fatalf("expected base BTC, got %s", event.Trade.Instrument.Base)
	}
	if event.Trade.Instrument.Quote != "USDT" {
		t.Fatalf("expected quote USDT, got %s", event.Trade.Instrument.Quote)
	}
	if event.Trade.Instrument.Contract != instrument.ContractSpot {
		t.Fatalf("expected contract spot, got %s", event.Trade.Instrument.Contract)
	}
	if got := event.Trade.VenueSymbol(); got != "btcusdt" {
		t.Fatalf("expected venue symbol btcusdt, got %s", got)
	}
	if event.Trade.TradeID != "4839201" {
		t.Fatalf("expected trade id 4839201, got %s", event.Trade.TradeID)
	}
	if event.Trade.BuyerMaker {
		t.Fatal("expected buyer maker to be false")
	}
	if event.Trade.Timestamp.IsZero() {
		t.Fatal("expected non-zero timestamp")
	}
}

func TestNormalize_SourceIsAlwaysBinances(t *testing.T) {
	agg := binances.AggTrade{
		EventType:  "aggTrade",
		EventTime:  1710000000000,
		Symbol:     "ETHUSDT",
		AggTradeID: 42,
		Price:      "3000.00",
		Quantity:   "1.0",
		TradeTime:  1710000000000,
	}

	event, prob := binances.Normalize(agg, "ethusdt")
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if event.Trade.Source != "binances" {
		t.Fatalf("source must be binances, got %s", event.Trade.Source)
	}
}

func TestParseAggTrade_MalformedJSON(t *testing.T) {
	_, prob := binances.ParseAggTrade([]byte("{invalid json"))
	if prob == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParseAggTrade_EmptyPayload(t *testing.T) {
	_, prob := binances.ParseAggTrade([]byte(""))
	if prob == nil {
		t.Fatal("expected error for empty payload")
	}
}

func TestNormalize_PriceQuantityPreserved(t *testing.T) {
	agg := binances.AggTrade{
		EventType:  "aggTrade",
		EventTime:  1710000000000,
		Symbol:     "BTCUSDT",
		AggTradeID: 1,
		Price:      "84521.30000000",
		Quantity:   "0.00150000",
		TradeTime:  1710000000000,
	}

	event, prob := binances.Normalize(agg, "btcusdt")
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if event.Trade.Price != "84521.30000000" {
		t.Fatalf("price not preserved: expected 84521.30000000, got %s", event.Trade.Price)
	}
	if event.Trade.Quantity != "0.00150000" {
		t.Fatalf("quantity not preserved: expected 0.00150000, got %s", event.Trade.Quantity)
	}
}

func TestNormalize_SymbolFromParameter(t *testing.T) {
	agg := binances.AggTrade{
		EventType:  "aggTrade",
		EventTime:  1710000000000,
		Symbol:     "SOLUSDT",
		AggTradeID: 1,
		Price:      "100.00",
		Quantity:   "10.0",
		TradeTime:  1710000000000,
	}

	event, prob := binances.Normalize(agg, "solusdt")
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if event.Trade.Instrument.Base != "SOL" {
		t.Fatalf("expected base SOL from parameter solusdt, got %s", event.Trade.Instrument.Base)
	}
	if event.Trade.Instrument.Quote != "USDT" {
		t.Fatalf("expected quote USDT, got %s", event.Trade.Instrument.Quote)
	}
}

func TestNormalize_RejectsNonUSDTQuote(t *testing.T) {
	agg := binances.AggTrade{
		EventType:  "aggTrade",
		EventTime:  1710000000000,
		Symbol:     "BTCBUSD",
		AggTradeID: 1,
		Price:      "100.00",
		Quantity:   "1.0",
		TradeTime:  1710000000000,
	}
	_, prob := binances.Normalize(agg, "btcbusd")
	if prob == nil {
		t.Fatal("expected error for non-USDT quote on binances")
	}
}

func TestFormatStreamURL(t *testing.T) {
	url := binances.FormatStreamURL("btcusdt")
	expected := "wss://stream.binance.com:9443/ws/btcusdt@aggTrade"
	if url != expected {
		t.Fatalf("expected %s, got %s", expected, url)
	}
}
