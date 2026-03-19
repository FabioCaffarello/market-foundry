package nats

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/observation"
	"internal/domain/signal"
	"internal/domain/strategy"
	"internal/shared/events"
	"internal/shared/problem"

	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ---------------------------------------------------------------------------
// Mock jetstream.Msg for consumer dispatch tests.
// ---------------------------------------------------------------------------

type mockMsg struct {
	data    []byte
	subject string
	acked   bool
	naked   bool
	termed  bool
}

func (m *mockMsg) Data() []byte                             { return m.data }
func (m *mockMsg) Subject() string                          { return m.subject }
func (m *mockMsg) Headers() natsgo.Header                   { return nil }
func (m *mockMsg) Reply() string                            { return "" }
func (m *mockMsg) Ack() error                               { m.acked = true; return nil }
func (m *mockMsg) DoubleAck(_ context.Context) error        { return nil }
func (m *mockMsg) Nak() error                               { m.naked = true; return nil }
func (m *mockMsg) NakWithDelay(_ time.Duration) error       { return nil }
func (m *mockMsg) InProgress() error                        { return nil }
func (m *mockMsg) Term() error                              { m.termed = true; return nil }
func (m *mockMsg) TermWithReason(_ string) error            { m.termed = true; return nil }
func (m *mockMsg) Metadata() (*jetstream.MsgMetadata, error) { return nil, nil }

var _ jetstream.Msg = (*mockMsg)(nil) // compile-time check

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// ---------------------------------------------------------------------------
// ObservationConsumer: valid event → handler called + Ack
// ---------------------------------------------------------------------------

func TestObservationConsumer_OnMessage_ValidEvent(t *testing.T) {
	reg := DefaultObservationRegistry()
	spec := DeriveObservationConsumer()

	event := observation.TradeReceivedEvent{
		Metadata: events.NewMetadata(),
		Trade: observation.ObservationTrade{
			Source: "binancef", Symbol: "btcusdt", Price: "65000.00",
			Quantity: "0.1", TradeID: "t-1", BuyerMaker: false,
			Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		},
	}

	data, prob := encodeEvent(reg.TradeReceived, "ingest", event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	var received observation.TradeReceivedEvent
	handler := func(e observation.TradeReceivedEvent) { received = e }

	consumer := NewObservationConsumer("nats://unused", spec, reg, handler, testLogger())
	msg := &mockMsg{data: data, subject: "observation.events.market.trade.binancef"}
	consumer.onMessage(msg)

	if !msg.acked {
		t.Error("expected Ack after successful processing")
	}
	if received.Trade.TradeID != "t-1" {
		t.Errorf("handler received wrong trade: got %s", received.Trade.TradeID)
	}
}

// ---------------------------------------------------------------------------
// ObservationConsumer: garbage data → Term (permanent error)
// ---------------------------------------------------------------------------

func TestObservationConsumer_OnMessage_GarbageData(t *testing.T) {
	reg := DefaultObservationRegistry()
	spec := DeriveObservationConsumer()

	handler := func(_ observation.TradeReceivedEvent) {
		t.Fatal("handler should not be called for invalid data")
	}

	consumer := NewObservationConsumer("nats://unused", spec, reg, handler, testLogger())
	msg := &mockMsg{data: []byte("not valid cbor"), subject: "observation.events.market.trade.binancef"}
	consumer.onMessage(msg)

	if !msg.termed {
		t.Error("expected Term for permanent decode error")
	}
	if msg.acked {
		t.Error("must not Ack a failed decode")
	}
}

// ---------------------------------------------------------------------------
// EvidenceConsumer (candle): valid event → handler called + Ack
// ---------------------------------------------------------------------------

func TestEvidenceConsumer_OnMessage_ValidCandle(t *testing.T) {
	reg := DefaultEvidenceRegistry()
	spec := StoreCandleConsumer()
	now := time.Now().UTC().Truncate(time.Millisecond)

	event := evidence.CandleSampledEvent{
		Metadata: events.NewMetadata(),
		Candle: evidence.EvidenceCandle{
			Source: "binancef", Symbol: "ethusdt", Timeframe: 60,
			Open: "3200", High: "3250", Low: "3180", Close: "3220", Volume: "500",
			TradeCount: 10, OpenTime: now, CloseTime: now.Add(time.Minute), Final: true,
		},
	}

	data, prob := encodeEvent(reg.CandleSampled, "derive", event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	var received evidence.CandleSampledEvent
	handler := func(e evidence.CandleSampledEvent) { received = e }

	consumer := NewEvidenceConsumer("nats://unused", spec, reg, handler, testLogger())
	msg := &mockMsg{data: data, subject: "evidence.events.candle.sampled.binancef.ethusdt.60"}
	consumer.onMessage(msg)

	if !msg.acked {
		t.Error("expected Ack")
	}
	if received.Candle.Symbol != "ethusdt" {
		t.Errorf("handler received wrong candle: got %s", received.Candle.Symbol)
	}
}

// ---------------------------------------------------------------------------
// EvidenceConsumer: garbage → Term
// ---------------------------------------------------------------------------

func TestEvidenceConsumer_OnMessage_GarbageData(t *testing.T) {
	reg := DefaultEvidenceRegistry()
	spec := StoreCandleConsumer()

	handler := func(_ evidence.CandleSampledEvent) {
		t.Fatal("handler should not be called")
	}

	consumer := NewEvidenceConsumer("nats://unused", spec, reg, handler, testLogger())
	msg := &mockMsg{data: []byte("garbage"), subject: "test"}
	consumer.onMessage(msg)

	if !msg.termed {
		t.Error("expected Term for decode error")
	}
}

// ---------------------------------------------------------------------------
// TradeBurstConsumer: valid event → Ack
// ---------------------------------------------------------------------------

func TestTradeBurstConsumer_OnMessage_ValidEvent(t *testing.T) {
	reg := DefaultEvidenceRegistry()
	spec := StoreTradeBurstConsumer()
	now := time.Now().UTC().Truncate(time.Millisecond)

	event := evidence.TradeBurstSampledEvent{
		Metadata: events.NewMetadata(),
		TradeBurst: evidence.EvidenceTradeBurst{
			Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
			TradeCount: 100, BuyVolume: "5000", SellVolume: "3000",
			OpenTime: now, CloseTime: now.Add(time.Minute), Final: true,
		},
	}

	data, prob := encodeEvent(reg.TradeBurstSampled, "derive", event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	var received evidence.TradeBurstSampledEvent
	handler := func(e evidence.TradeBurstSampledEvent) { received = e }

	consumer := NewTradeBurstConsumer("nats://unused", spec, reg, handler, testLogger())
	msg := &mockMsg{data: data, subject: "evidence.events.tradeburst.sampled.binancef.btcusdt.60"}
	consumer.onMessage(msg)

	if !msg.acked {
		t.Error("expected Ack")
	}
	if received.TradeBurst.TradeCount != 100 {
		t.Errorf("expected 100 trades, got %d", received.TradeBurst.TradeCount)
	}
}

// ---------------------------------------------------------------------------
// VolumeConsumer: valid event → Ack
// ---------------------------------------------------------------------------

func TestVolumeConsumer_OnMessage_ValidEvent(t *testing.T) {
	reg := DefaultEvidenceRegistry()
	spec := StoreVolumeConsumer()
	now := time.Now().UTC().Truncate(time.Millisecond)

	event := evidence.VolumeSampledEvent{
		Metadata: events.NewMetadata(),
		Volume: evidence.EvidenceVolume{
			Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
			BuyVolume: "100000", SellVolume: "80000", TotalVolume: "180000", VWAP: "65000",
			TradeCount: 150, OpenTime: now, CloseTime: now.Add(time.Minute), Final: true,
		},
	}

	data, prob := encodeEvent(reg.VolumeSampled, "derive", event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	var received evidence.VolumeSampledEvent
	handler := func(e evidence.VolumeSampledEvent) { received = e }

	consumer := NewVolumeConsumer("nats://unused", spec, reg, handler, testLogger())
	msg := &mockMsg{data: data, subject: "evidence.events.volume.sampled.binancef.btcusdt.60"}
	consumer.onMessage(msg)

	if !msg.acked {
		t.Error("expected Ack")
	}
	if received.Volume.VWAP != "65000" {
		t.Errorf("expected VWAP 65000, got %s", received.Volume.VWAP)
	}
}

// ---------------------------------------------------------------------------
// SignalConsumer: valid event → Ack
// ---------------------------------------------------------------------------

func TestSignalConsumer_OnMessage_ValidEvent(t *testing.T) {
	reg := DefaultSignalRegistry()
	spec := StoreRSISignalConsumer()
	now := time.Now().UTC().Truncate(time.Millisecond)

	event := signal.SignalGeneratedEvent{
		Metadata: events.NewMetadata(),
		Signal: signal.Signal{
			Type: "rsi", Source: "binancef", Symbol: "btcusdt",
			Timeframe: 60, Value: "28.5",
			Metadata: map[string]string{"period": "14"},
			Final: true, Timestamp: now,
		},
	}

	data, prob := encodeEvent(spec.Event, "derive", event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	var received signal.SignalGeneratedEvent
	handler := func(e signal.SignalGeneratedEvent) { received = e }

	consumer := NewSignalConsumer("nats://unused", spec, reg, handler, testLogger())
	msg := &mockMsg{data: data, subject: "signal.events.rsi.generated.binancef.btcusdt.60"}
	consumer.onMessage(msg)

	if !msg.acked {
		t.Error("expected Ack")
	}
	if received.Signal.Value != "28.5" {
		t.Errorf("expected value 28.5, got %s", received.Signal.Value)
	}
}

// ---------------------------------------------------------------------------
// DecisionConsumer: valid event → Ack
// ---------------------------------------------------------------------------

func TestDecisionConsumer_OnMessage_ValidEvent(t *testing.T) {
	reg := DefaultDecisionRegistry()
	spec := StoreRSIOversoldDecisionConsumer()
	now := time.Now().UTC().Truncate(time.Millisecond)

	event := decision.DecisionEvaluatedEvent{
		Metadata: events.NewMetadata(),
		Decision: decision.Decision{
			Type: "rsi_oversold", Source: "binancef", Symbol: "btcusdt",
			Timeframe: 60, Outcome: decision.OutcomeTriggered, Confidence: "0.85",
			Signals:   []decision.SignalInput{{Type: "rsi", Value: "28.5", Timeframe: 60}},
			Final:     true, Timestamp: now,
		},
	}

	data, prob := encodeEvent(spec.Event, "derive", event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	var received decision.DecisionEvaluatedEvent
	handler := func(e decision.DecisionEvaluatedEvent) { received = e }

	consumer := NewDecisionConsumer("nats://unused", spec, reg, handler, testLogger())
	msg := &mockMsg{data: data, subject: "decision.events.rsi_oversold.evaluated.binancef.btcusdt.60"}
	consumer.onMessage(msg)

	if !msg.acked {
		t.Error("expected Ack")
	}
	if received.Decision.Outcome != decision.OutcomeTriggered {
		t.Errorf("expected triggered, got %s", received.Decision.Outcome)
	}
}

// ---------------------------------------------------------------------------
// StrategyConsumer: valid event → Ack
// ---------------------------------------------------------------------------

func TestStrategyConsumer_OnMessage_ValidEvent(t *testing.T) {
	reg := DefaultStrategyRegistry()
	spec := StoreMeanReversionEntryStrategyConsumer()
	now := time.Now().UTC().Truncate(time.Millisecond)

	event := strategy.StrategyResolvedEvent{
		Metadata: events.NewMetadata(),
		Strategy: strategy.Strategy{
			Type: "mean_reversion_entry", Source: "binancef", Symbol: "btcusdt",
			Timeframe: 60, Direction: strategy.DirectionLong, Confidence: "0.75",
			Decisions: []strategy.DecisionInput{{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Timeframe: 60}},
			Final:     true, Timestamp: now,
		},
	}

	data, prob := encodeEvent(spec.Event, "derive", event, event.Metadata.CorrelationID, event.Metadata.CausationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	var received strategy.StrategyResolvedEvent
	handler := func(e strategy.StrategyResolvedEvent) { received = e }

	consumer := NewStrategyConsumer("nats://unused", spec, reg, handler, testLogger())
	msg := &mockMsg{data: data, subject: "strategy.events.mean_reversion_entry.resolved.binancef.btcusdt.60"}
	consumer.onMessage(msg)

	if !msg.acked {
		t.Error("expected Ack")
	}
	if received.Strategy.Direction != strategy.DirectionLong {
		t.Errorf("expected long, got %s", received.Strategy.Direction)
	}
}

// ---------------------------------------------------------------------------
// terminateOrNak: InvalidArgument → Term, other → Nak
// ---------------------------------------------------------------------------

func TestTerminateOrNak_InvalidArgument_Terms(t *testing.T) {
	reg := DefaultObservationRegistry()
	spec := DeriveObservationConsumer()
	consumer := NewObservationConsumer("nats://unused", spec, reg, nil, testLogger())

	msg := &mockMsg{}
	prob := problem.New(problem.InvalidArgument, "bad data")
	consumer.terminateOrNak(msg, prob)

	if !msg.termed {
		t.Error("expected Term for InvalidArgument")
	}
	if msg.naked {
		t.Error("must not Nak for InvalidArgument")
	}
}

func TestTerminateOrNak_TransientError_Naks(t *testing.T) {
	reg := DefaultObservationRegistry()
	spec := DeriveObservationConsumer()
	consumer := NewObservationConsumer("nats://unused", spec, reg, nil, testLogger())

	msg := &mockMsg{}
	prob := problem.New(problem.Unavailable, "transient error")
	consumer.terminateOrNak(msg, prob)

	if msg.termed {
		t.Error("must not Term for transient error")
	}
	if !msg.naked {
		t.Error("expected Nak for transient error")
	}
}

func TestTerminateOrNak_InternalError_Naks(t *testing.T) {
	reg := DefaultObservationRegistry()
	spec := DeriveObservationConsumer()
	consumer := NewObservationConsumer("nats://unused", spec, reg, nil, testLogger())

	msg := &mockMsg{}
	prob := problem.New(problem.Internal, "internal error")
	consumer.terminateOrNak(msg, prob)

	if msg.termed {
		t.Error("must not Term for internal error")
	}
	if !msg.naked {
		t.Error("expected Nak for internal error")
	}
}

// ---------------------------------------------------------------------------
// Consumer Close: closing an unstarted consumer must not panic.
// ---------------------------------------------------------------------------

func TestConsumer_Close_Unstarted(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{"observation", func() error {
			return NewObservationConsumer("nats://u", DeriveObservationConsumer(), DefaultObservationRegistry(), nil, testLogger()).Close()
		}},
		{"evidence", func() error {
			return NewEvidenceConsumer("nats://u", StoreCandleConsumer(), DefaultEvidenceRegistry(), nil, testLogger()).Close()
		}},
		{"trade_burst", func() error {
			return NewTradeBurstConsumer("nats://u", StoreTradeBurstConsumer(), DefaultEvidenceRegistry(), nil, testLogger()).Close()
		}},
		{"volume", func() error {
			return NewVolumeConsumer("nats://u", StoreVolumeConsumer(), DefaultEvidenceRegistry(), nil, testLogger()).Close()
		}},
		{"signal", func() error {
			return NewSignalConsumer("nats://u", StoreRSISignalConsumer(), DefaultSignalRegistry(), nil, testLogger()).Close()
		}},
		{"decision", func() error {
			return NewDecisionConsumer("nats://u", StoreRSIOversoldDecisionConsumer(), DefaultDecisionRegistry(), nil, testLogger()).Close()
		}},
		{"strategy", func() error {
			return NewStrategyConsumer("nats://u", StoreMeanReversionEntryStrategyConsumer(), DefaultStrategyRegistry(), nil, testLogger()).Close()
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err != nil {
				t.Errorf("close unstarted: %v", err)
			}
		})
	}
}
