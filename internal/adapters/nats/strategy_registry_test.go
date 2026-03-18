package nats

import (
	"testing"
)

func TestDefaultStrategyRegistry_StreamName(t *testing.T) {
	reg := DefaultStrategyRegistry()
	if reg.MeanReversionEntryResolved.Stream.Name != "STRATEGY_EVENTS" {
		t.Fatalf("expected STRATEGY_EVENTS, got %s", reg.MeanReversionEntryResolved.Stream.Name)
	}
}

func TestDefaultStrategyRegistry_EventSubject(t *testing.T) {
	reg := DefaultStrategyRegistry()
	expected := "strategy.events.mean_reversion_entry.resolved"
	if reg.MeanReversionEntryResolved.Subject != expected {
		t.Fatalf("expected %s, got %s", expected, reg.MeanReversionEntryResolved.Subject)
	}
}

func TestDefaultStrategyRegistry_QuerySubject(t *testing.T) {
	reg := DefaultStrategyRegistry()
	expected := "strategy.query.mean_reversion_entry.latest"
	if reg.MeanReversionEntryLatest.Subject != expected {
		t.Fatalf("expected %s, got %s", expected, reg.MeanReversionEntryLatest.Subject)
	}
}

func TestStrategyRegistry_LatestSpecByType_Known(t *testing.T) {
	reg := DefaultStrategyRegistry()
	spec, ok := reg.LatestSpecByType("mean_reversion_entry")
	if !ok {
		t.Fatal("expected mean_reversion_entry to be registered")
	}
	if spec.Subject != "strategy.query.mean_reversion_entry.latest" {
		t.Fatalf("unexpected subject: %s", spec.Subject)
	}
}

func TestStrategyRegistry_LatestSpecByType_Unknown(t *testing.T) {
	reg := DefaultStrategyRegistry()
	_, ok := reg.LatestSpecByType("unknown_strategy")
	if ok {
		t.Fatal("expected unknown strategy type to return false")
	}
}

func TestStoreMeanReversionEntryStrategyConsumer(t *testing.T) {
	spec := StoreMeanReversionEntryStrategyConsumer()
	if spec.Durable != "store-strategy-mean-reversion-entry" {
		t.Fatalf("expected durable store-strategy-mean-reversion-entry, got %s", spec.Durable)
	}
	if spec.Event.Stream.Name != "STRATEGY_EVENTS" {
		t.Fatalf("expected stream STRATEGY_EVENTS, got %s", spec.Event.Stream.Name)
	}
}
