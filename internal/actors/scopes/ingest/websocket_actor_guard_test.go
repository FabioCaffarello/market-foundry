package ingest

import (
	"log/slog"
	"testing"
	"time"

	"internal/adapters/exchanges/binancef"
	"internal/adapters/exchanges/binances"
	"internal/domain/instrument"
	"internal/domain/observation"
	"internal/shared/events"
	"internal/shared/metrics"
)

func tradeEventFor(t *testing.T, source string, contract instrument.ContractType) observation.TradeReceivedEvent {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", contract)
	if prob != nil {
		t.Fatalf("New: %v", prob)
	}
	trade := observation.ObservationTrade{
		Source:     source,
		Instrument: inst,
		Price:      "50000",
		Quantity:   "0.01",
		TradeID:    "1",
		Timestamp:  time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC),
	}
	return observation.TradeReceivedEvent{
		Metadata: events.NewMetadata().WithOccurredAt(trade.Timestamp),
		Trade:    trade,
	}
}

// ── ADR-0022 R3 guard (H-7.a) ─────────────────────────────────────

// Declared pairs pass through without touching the counter.
func TestDeclared_DeclaredPairPasses(t *testing.T) {
	a := &WebSocketAdapterActor{logger: slog.Default()}
	caps := binancef.Capabilities()

	before := metrics.AdapterUndeclaredEventCount("binancef", eventTypeTrade, "perpetual")
	if !a.declared(caps, tradeEventFor(t, "binancef", instrument.ContractPerpetual)) {
		t.Fatal("declared pair (observation.trade, perpetual) must pass the R3 guard")
	}
	if got := metrics.AdapterUndeclaredEventCount("binancef", eventTypeTrade, "perpetual"); got != before {
		t.Errorf("counter moved for a declared pair: %v -> %v", before, got)
	}
}

// Undeclared pairs are rejected AND counted — the silent-zero shape
// is forbidden: rejection must be observable (ADR-0022 R3).
func TestDeclared_UndeclaredPairRejectedAndCounted(t *testing.T) {
	a := &WebSocketAdapterActor{logger: slog.Default()}
	// Spot adapter receiving a perpetual instrument: undeclared.
	caps := binances.Capabilities()

	before := metrics.AdapterUndeclaredEventCount("binance", eventTypeTrade, "perpetual")
	if a.declared(caps, tradeEventFor(t, "binances", instrument.ContractPerpetual)) {
		t.Fatal("undeclared pair (observation.trade, perpetual on spot adapter) must be rejected")
	}
	after := metrics.AdapterUndeclaredEventCount("binance", eventTypeTrade, "perpetual")
	if after != before+1 {
		t.Errorf("undeclared rejection must increment the counter: %v -> %v", before, after)
	}
}
