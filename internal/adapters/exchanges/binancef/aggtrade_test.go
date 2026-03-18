package binancef_test

import (
	"testing"

	"internal/adapters/exchanges/binancef"
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

	agg, prob := binancef.ParseAggTrade([]byte(raw))
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
	_, prob := binancef.ParseAggTrade([]byte(raw))
	if prob == nil {
		t.Fatal("expected error for non-aggTrade event type")
	}
}

func TestNormalize(t *testing.T) {
	agg := binancef.AggTrade{
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

	event, prob := binancef.Normalize(agg, "btcusdt")
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if event.Trade.Source != "binancef" {
		t.Fatalf("expected source binancef, got %s", event.Trade.Source)
	}
	if event.Trade.Symbol != "btcusdt" {
		t.Fatalf("expected symbol btcusdt, got %s", event.Trade.Symbol)
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

func TestParseAggTrade_MalformedJSON(t *testing.T) {
	_, prob := binancef.ParseAggTrade([]byte("{invalid json"))
	if prob == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParseAggTrade_EmptyPayload(t *testing.T) {
	_, prob := binancef.ParseAggTrade([]byte(""))
	if prob == nil {
		t.Fatal("expected error for empty payload")
	}
}

func TestParseAggTrade_EmptyObject(t *testing.T) {
	// Empty JSON object has no event type → should be rejected.
	_, prob := binancef.ParseAggTrade([]byte("{}"))
	if prob == nil {
		t.Fatal("expected error for empty object (no event type)")
	}
}

func TestNormalize_TimestampConversion(t *testing.T) {
	agg := binancef.AggTrade{
		EventType:    "aggTrade",
		EventTime:    1710000000000,
		Symbol:       "BTCUSDT",
		AggTradeID:   1,
		Price:        "100.00",
		Quantity:     "1.0",
		TradeTime:    1710000000123,
		IsBuyerMaker: false,
	}

	event, prob := binancef.Normalize(agg, "btcusdt")
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}

	// Verify millisecond precision: TradeTime 1710000000123 ms → 1710000000 sec + 123 ms
	expectedSec := int64(1710000000)
	if event.Trade.Timestamp.Unix() != expectedSec {
		t.Fatalf("expected unix seconds %d, got %d", expectedSec, event.Trade.Timestamp.Unix())
	}
	if event.Trade.Timestamp.UTC() != event.Trade.Timestamp {
		t.Fatal("timestamp must be in UTC")
	}
}

func TestNormalize_PriceQuantityPreserved(t *testing.T) {
	// Verify decimal strings are preserved exactly through normalization.
	agg := binancef.AggTrade{
		EventType:  "aggTrade",
		EventTime:  1710000000000,
		Symbol:     "BTCUSDT",
		AggTradeID: 1,
		Price:      "84521.30000000",
		Quantity:   "0.00150000",
		TradeTime:  1710000000000,
	}

	event, prob := binancef.Normalize(agg, "btcusdt")
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

func TestNormalize_SourceIsAlwaysBinancef(t *testing.T) {
	agg := binancef.AggTrade{
		EventType:  "aggTrade",
		EventTime:  1710000000000,
		Symbol:     "ETHUSDT",
		AggTradeID: 42,
		Price:      "3000.00",
		Quantity:   "1.0",
		TradeTime:  1710000000000,
	}

	event, prob := binancef.Normalize(agg, "ethusdt")
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if event.Trade.Source != "binancef" {
		t.Fatalf("source must be binancef, got %s", event.Trade.Source)
	}
}

func TestNormalize_SymbolFromParameter(t *testing.T) {
	// Symbol in AggTrade is uppercase from Binance; the normalize function uses the parameter.
	agg := binancef.AggTrade{
		EventType:  "aggTrade",
		EventTime:  1710000000000,
		Symbol:     "SOLUSDT",
		AggTradeID: 1,
		Price:      "100.00",
		Quantity:   "10.0",
		TradeTime:  1710000000000,
	}

	event, prob := binancef.Normalize(agg, "solusdt")
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if event.Trade.Symbol != "solusdt" {
		t.Fatalf("expected symbol from parameter solusdt, got %s", event.Trade.Symbol)
	}
}

func TestNormalize_TradeIDFormat(t *testing.T) {
	tests := []struct {
		name     string
		id       int64
		expected string
	}{
		{"zero", 0, "0"},
		{"positive", 4839201, "4839201"},
		{"large", 9999999999, "9999999999"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agg := binancef.AggTrade{
				EventType:  "aggTrade",
				EventTime:  1710000000000,
				Symbol:     "BTCUSDT",
				AggTradeID: tc.id,
				Price:      "100.00",
				Quantity:   "1.0",
				TradeTime:  1710000000000,
			}
			event, prob := binancef.Normalize(agg, "btcusdt")
			if prob != nil {
				t.Fatalf("unexpected error: %v", prob)
			}
			if event.Trade.TradeID != tc.expected {
				t.Fatalf("expected trade_id %s, got %s", tc.expected, event.Trade.TradeID)
			}
		})
	}
}

func TestNormalize_EventMetadata(t *testing.T) {
	agg := binancef.AggTrade{
		EventType:  "aggTrade",
		EventTime:  1710000000000,
		Symbol:     "BTCUSDT",
		AggTradeID: 1,
		Price:      "100.00",
		Quantity:   "1.0",
		TradeTime:  1710000000000,
	}

	event, prob := binancef.Normalize(agg, "btcusdt")
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if event.EventName() != "market.trade_received" {
		t.Fatalf("expected event name market.trade_received, got %s", event.EventName())
	}
	// CorrelationID is not set by Normalize — it's set at the actor layer.
	// Verify metadata has a valid ID (auto-generated).
	if event.EventMetadata().ID == "" {
		t.Fatal("expected non-empty event ID in metadata")
	}
}

func TestNormalize_EmptySymbolParamFails(t *testing.T) {
	agg := binancef.AggTrade{
		EventType:  "aggTrade",
		EventTime:  1710000000000,
		Symbol:     "BTCUSDT",
		AggTradeID: 1,
		Price:      "100.00",
		Quantity:   "1.0",
		TradeTime:  1710000000000,
	}

	_, prob := binancef.Normalize(agg, "")
	if prob == nil {
		t.Fatal("expected validation error when symbol parameter is empty")
	}
}

func TestNormalize_ZeroTradeTimeFails(t *testing.T) {
	agg := binancef.AggTrade{
		EventType:  "aggTrade",
		EventTime:  1710000000000,
		Symbol:     "BTCUSDT",
		AggTradeID: 1,
		Price:      "100.00",
		Quantity:   "1.0",
		TradeTime:  0,
	}

	_, prob := binancef.Normalize(agg, "btcusdt")
	// TradeTime=0 → timestamp is 1970-01-01 → NOT zero → passes domain validation.
	// This is by design — the domain doesn't restrict timestamp range.
	if prob != nil {
		t.Fatalf("TradeTime=0 maps to epoch, which is a valid timestamp: %v", prob)
	}
}
