package ingest_test

import (
	"testing"

	"internal/adapters/exchanges/binances"
	"internal/application/ingest"
)

// S397: Spot ingest binding seed — verify that the binding model correctly
// parses Spot topics and that the Spot exchange adapter produces canonical
// source identifiers.

func TestS397_ParseBindingTopic_SpotSource(t *testing.T) {
	target, prob := ingest.ParseBindingTopic("binances.btcusdt")
	if prob != nil {
		t.Fatalf("unexpected error parsing spot binding topic: %v", prob)
	}
	if target.Source != "binances" {
		t.Fatalf("expected source binances, got %s", target.Source)
	}
	if target.Symbol != "btcusdt" {
		t.Fatalf("expected symbol btcusdt, got %s", target.Symbol)
	}
}

func TestS397_ParseBindingTopic_SpotKey(t *testing.T) {
	target, prob := ingest.ParseBindingTopic("binances.ethusdt")
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if target.Key() != "binances.ethusdt" {
		t.Fatalf("expected key binances.ethusdt, got %s", target.Key())
	}
}

func TestS397_SpotNormalize_SourceIdentity(t *testing.T) {
	// Verify that the Spot adapter always stamps source=binances.
	agg := binances.AggTrade{
		EventType:  "aggTrade",
		EventTime:  1710000000000,
		Symbol:     "BTCUSDT",
		AggTradeID: 1,
		Price:      "84521.30",
		Quantity:   "0.150",
		TradeTime:  1710000000123,
	}

	event, prob := binances.Normalize(agg, "btcusdt")
	if prob != nil {
		t.Fatalf("unexpected error: %v", prob)
	}
	if event.Trade.Source != "binances" {
		t.Fatalf("spot adapter must stamp source=binances, got %s", event.Trade.Source)
	}
}

func TestS397_SpotAndFuturesSourcesAreDistinct(t *testing.T) {
	spotTarget, _ := ingest.ParseBindingTopic("binances.btcusdt")
	futuresTarget, _ := ingest.ParseBindingTopic("binancef.btcusdt")

	if spotTarget.Source == futuresTarget.Source {
		t.Fatal("spot and futures sources must be distinct")
	}
	if spotTarget.Key() == futuresTarget.Key() {
		t.Fatal("spot and futures binding keys must be distinct for the same symbol")
	}
}

func TestS397_SpotWebSocketURL_PointsToSpotEndpoint(t *testing.T) {
	url := binances.FormatStreamURL("btcusdt")
	expected := "wss://stream.binance.com:9443/ws/btcusdt@aggTrade"
	if url != expected {
		t.Fatalf("spot WS URL must point to stream.binance.com, got %s", url)
	}
}
