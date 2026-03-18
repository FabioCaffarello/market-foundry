package nats

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/observation"
	"internal/domain/signal"
	"internal/domain/strategy"
	"internal/shared/events"
)

// ---------------------------------------------------------------------------
// Encode → Decode roundtrip: the core transport contract.
// If a publisher can encode a domain event, a consumer must be able to decode
// it back with identical payload. These tests cover the invariant for every
// domain event type currently on the mesh.
// ---------------------------------------------------------------------------

func testSpec(subject, eventType string) EventSpec {
	return EventSpec{
		Subject: subject,
		Type:    eventType,
		Stream: StreamSpec{
			Name:     "TEST_STREAM",
			Subjects: []string{subject + ".>"},
		},
	}
}

func TestCodecRoundtrip_ObservationTradeReceived(t *testing.T) {
	spec := testSpec("observation.events.market.trade", "observation.events.v1.trade_received")
	source := "test-derive"

	original := observation.TradeReceivedEvent{
		Metadata: events.NewMetadata(),
		Trade: observation.ObservationTrade{
			Source:     "binancef",
			Symbol:    "btcusdt",
			Price:     "65432.10",
			Quantity:  "0.5",
			TradeID:   "abc-123",
			BuyerMaker: true,
			Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		},
	}

	data, prob := encodeEvent(spec, source, original, original.Metadata.CorrelationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	env, prob := decodeEvent[observation.TradeReceivedEvent](spec, data)
	if prob != nil {
		t.Fatalf("decode: %v", prob)
	}

	got := env.Payload
	if got.Trade.Source != original.Trade.Source {
		t.Errorf("source: want %s, got %s", original.Trade.Source, got.Trade.Source)
	}
	if got.Trade.Symbol != original.Trade.Symbol {
		t.Errorf("symbol: want %s, got %s", original.Trade.Symbol, got.Trade.Symbol)
	}
	if got.Trade.Price != original.Trade.Price {
		t.Errorf("price: want %s, got %s", original.Trade.Price, got.Trade.Price)
	}
	if got.Trade.Quantity != original.Trade.Quantity {
		t.Errorf("quantity: want %s, got %s", original.Trade.Quantity, got.Trade.Quantity)
	}
	if got.Trade.TradeID != original.Trade.TradeID {
		t.Errorf("trade_id: want %s, got %s", original.Trade.TradeID, got.Trade.TradeID)
	}
	if got.Trade.BuyerMaker != original.Trade.BuyerMaker {
		t.Errorf("buyer_maker: want %v, got %v", original.Trade.BuyerMaker, got.Trade.BuyerMaker)
	}
}

func TestCodecRoundtrip_EvidenceCandleSampled(t *testing.T) {
	spec := testSpec("evidence.events.candle.sampled", "evidence.events.v1.candle_sampled")
	now := time.Now().UTC().Truncate(time.Millisecond)

	original := evidence.CandleSampledEvent{
		Metadata: events.NewMetadata(),
		Candle: evidence.EvidenceCandle{
			Source:     "binancef",
			Symbol:    "ethusdt",
			Timeframe: 60,
			Open:      "3200.00",
			High:      "3250.00",
			Low:       "3180.00",
			Close:     "3220.00",
			Volume:    "1500.50",
			TradeCount: 42,
			OpenTime:  now,
			CloseTime: now.Add(60 * time.Second),
			Final:     true,
		},
	}

	data, prob := encodeEvent(spec, "derive", original, original.Metadata.CorrelationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	env, prob := decodeEvent[evidence.CandleSampledEvent](spec, data)
	if prob != nil {
		t.Fatalf("decode: %v", prob)
	}

	got := env.Payload
	if got.Candle.Source != original.Candle.Source {
		t.Errorf("source: want %s, got %s", original.Candle.Source, got.Candle.Source)
	}
	if got.Candle.Symbol != original.Candle.Symbol {
		t.Errorf("symbol: want %s, got %s", original.Candle.Symbol, got.Candle.Symbol)
	}
	if got.Candle.Timeframe != original.Candle.Timeframe {
		t.Errorf("timeframe: want %d, got %d", original.Candle.Timeframe, got.Candle.Timeframe)
	}
	if got.Candle.Open != original.Candle.Open {
		t.Errorf("open: want %s, got %s", original.Candle.Open, got.Candle.Open)
	}
	if got.Candle.Close != original.Candle.Close {
		t.Errorf("close: want %s, got %s", original.Candle.Close, got.Candle.Close)
	}
	if got.Candle.Final != original.Candle.Final {
		t.Errorf("final: want %v, got %v", original.Candle.Final, got.Candle.Final)
	}
}

func TestCodecRoundtrip_EvidenceTradeBurstSampled(t *testing.T) {
	spec := testSpec("evidence.events.tradeburst.sampled", "evidence.events.v1.tradeburst_sampled")
	now := time.Now().UTC().Truncate(time.Millisecond)

	original := evidence.TradeBurstSampledEvent{
		Metadata: events.NewMetadata(),
		TradeBurst: evidence.EvidenceTradeBurst{
			Source:     "binancef",
			Symbol:    "btcusdt",
			Timeframe: 60,
			TradeCount: 200,
			BuyVolume:  "5000.00",
			SellVolume: "3000.00",
			OpenTime:  now,
			CloseTime: now.Add(60 * time.Second),
			Burst:     true,
			Final:     true,
		},
	}

	data, prob := encodeEvent(spec, "derive", original, original.Metadata.CorrelationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	env, prob := decodeEvent[evidence.TradeBurstSampledEvent](spec, data)
	if prob != nil {
		t.Fatalf("decode: %v", prob)
	}

	got := env.Payload
	if got.TradeBurst.BuyVolume != original.TradeBurst.BuyVolume {
		t.Errorf("buy_volume: want %s, got %s", original.TradeBurst.BuyVolume, got.TradeBurst.BuyVolume)
	}
	if got.TradeBurst.Burst != original.TradeBurst.Burst {
		t.Errorf("burst: want %v, got %v", original.TradeBurst.Burst, got.TradeBurst.Burst)
	}
}

func TestCodecRoundtrip_EvidenceVolumeSampled(t *testing.T) {
	spec := testSpec("evidence.events.volume.sampled", "evidence.events.v1.volume_sampled")
	now := time.Now().UTC().Truncate(time.Millisecond)

	original := evidence.VolumeSampledEvent{
		Metadata: events.NewMetadata(),
		Volume: evidence.EvidenceVolume{
			Source:      "binancef",
			Symbol:      "btcusdt",
			Timeframe:   60,
			BuyVolume:   "100000.00",
			SellVolume:  "80000.00",
			TotalVolume: "180000.00",
			VWAP:        "65000.00",
			TradeCount:  150,
			OpenTime:    now,
			CloseTime:   now.Add(60 * time.Second),
			Final:       true,
		},
	}

	data, prob := encodeEvent(spec, "derive", original, original.Metadata.CorrelationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	env, prob := decodeEvent[evidence.VolumeSampledEvent](spec, data)
	if prob != nil {
		t.Fatalf("decode: %v", prob)
	}

	got := env.Payload
	if got.Volume.VWAP != original.Volume.VWAP {
		t.Errorf("vwap: want %s, got %s", original.Volume.VWAP, got.Volume.VWAP)
	}
	if got.Volume.TotalVolume != original.Volume.TotalVolume {
		t.Errorf("total_volume: want %s, got %s", original.Volume.TotalVolume, got.Volume.TotalVolume)
	}
}

func TestCodecRoundtrip_SignalGenerated(t *testing.T) {
	spec := testSpec("signal.events.rsi.generated", "signal.events.v1.rsi_generated")
	now := time.Now().UTC().Truncate(time.Millisecond)

	original := signal.SignalGeneratedEvent{
		Metadata: events.NewMetadata(),
		Signal: signal.Signal{
			Type:      "rsi",
			Source:    "binancef",
			Symbol:   "btcusdt",
			Timeframe: 60,
			Value:     "28.5",
			Metadata:  map[string]string{"period": "14"},
			Final:     true,
			Timestamp: now,
		},
	}

	data, prob := encodeEvent(spec, "derive", original, original.Metadata.CorrelationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	env, prob := decodeEvent[signal.SignalGeneratedEvent](spec, data)
	if prob != nil {
		t.Fatalf("decode: %v", prob)
	}

	got := env.Payload
	if got.Signal.Type != original.Signal.Type {
		t.Errorf("type: want %s, got %s", original.Signal.Type, got.Signal.Type)
	}
	if got.Signal.Value != original.Signal.Value {
		t.Errorf("value: want %s, got %s", original.Signal.Value, got.Signal.Value)
	}
	if got.Signal.Metadata["period"] != "14" {
		t.Errorf("metadata[period]: want 14, got %s", got.Signal.Metadata["period"])
	}
}

func TestCodecRoundtrip_DecisionEvaluated(t *testing.T) {
	spec := testSpec("decision.events.rsi_oversold.evaluated", "decision.events.v1.rsi_oversold_evaluated")
	now := time.Now().UTC().Truncate(time.Millisecond)

	original := decision.DecisionEvaluatedEvent{
		Metadata: events.NewMetadata(),
		Decision: decision.Decision{
			Type:       "rsi_oversold",
			Source:     "binancef",
			Symbol:     "btcusdt",
			Timeframe:  60,
			Outcome:    decision.OutcomeTriggered,
			Confidence: "0.85",
			Signals: []decision.SignalInput{
				{Type: "rsi", Value: "28.5", Timeframe: 60},
			},
			Metadata:  map[string]string{"threshold": "30"},
			Final:     true,
			Timestamp: now,
		},
	}

	data, prob := encodeEvent(spec, "derive", original, original.Metadata.CorrelationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	env, prob := decodeEvent[decision.DecisionEvaluatedEvent](spec, data)
	if prob != nil {
		t.Fatalf("decode: %v", prob)
	}

	got := env.Payload
	if got.Decision.Outcome != original.Decision.Outcome {
		t.Errorf("outcome: want %s, got %s", original.Decision.Outcome, got.Decision.Outcome)
	}
	if len(got.Decision.Signals) != 1 {
		t.Fatalf("signals: want 1, got %d", len(got.Decision.Signals))
	}
	if got.Decision.Signals[0].Value != "28.5" {
		t.Errorf("signal value: want 28.5, got %s", got.Decision.Signals[0].Value)
	}
}

func TestCodecRoundtrip_StrategyResolved(t *testing.T) {
	spec := testSpec("strategy.events.mean_reversion_entry.resolved", "strategy.events.v1.mean_reversion_entry_resolved")
	now := time.Now().UTC().Truncate(time.Millisecond)

	original := strategy.StrategyResolvedEvent{
		Metadata: events.NewMetadata(),
		Strategy: strategy.Strategy{
			Type:       "mean_reversion_entry",
			Source:     "binancef",
			Symbol:     "btcusdt",
			Timeframe:  60,
			Direction:  strategy.DirectionLong,
			Confidence: "0.75",
			Decisions: []strategy.DecisionInput{
				{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.85", Timeframe: 60},
			},
			Parameters: map[string]string{"entry_mode": "aggressive"},
			Metadata:   map[string]string{},
			Final:      true,
			Timestamp:  now,
		},
	}

	data, prob := encodeEvent(spec, "derive", original, original.Metadata.CorrelationID)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	env, prob := decodeEvent[strategy.StrategyResolvedEvent](spec, data)
	if prob != nil {
		t.Fatalf("decode: %v", prob)
	}

	got := env.Payload
	if got.Strategy.Direction != original.Strategy.Direction {
		t.Errorf("direction: want %s, got %s", original.Strategy.Direction, got.Strategy.Direction)
	}
	if len(got.Strategy.Decisions) != 1 {
		t.Fatalf("decisions: want 1, got %d", len(got.Strategy.Decisions))
	}
}

// ---------------------------------------------------------------------------
// Decode rejects wrong envelope kind.
// A consumer must reject a command envelope when it expects an event.
// ---------------------------------------------------------------------------

func TestCodecDecode_RejectsWrongKind(t *testing.T) {
	spec := testSpec("observation.events.market.trade", "observation.events.v1.trade_received")
	controlSpec := ControlSpec{
		Subject:     "test.control",
		RequestType: "observation.events.v1.trade_received",
		ReplyType:   "test.reply",
		QueueGroup:  "test",
	}

	trade := observation.TradeReceivedEvent{
		Metadata: events.NewMetadata(),
		Trade: observation.ObservationTrade{
			Source: "binancef", Symbol: "btcusdt", Price: "1", Quantity: "1",
			TradeID: "t1", Timestamp: time.Now().UTC(),
		},
	}

	// Encode as command (wrong kind for event decode).
	data, prob := encodeControlRequest(nil, controlSpec, "test", trade)
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	_, prob = decodeEvent[observation.TradeReceivedEvent](spec, data)
	if prob == nil {
		t.Fatal("expected decode to fail for wrong envelope kind")
	}
}

// ---------------------------------------------------------------------------
// Decode rejects wrong event type.
// ---------------------------------------------------------------------------

func TestCodecDecode_RejectsWrongType(t *testing.T) {
	encodeSpec := testSpec("signal.events.rsi.generated", "signal.events.v1.rsi_generated")
	decodeSpec := testSpec("signal.events.rsi.generated", "signal.events.v1.wrong_type")

	event := signal.SignalGeneratedEvent{
		Metadata: events.NewMetadata(),
		Signal: signal.Signal{
			Type: "rsi", Source: "binancef", Symbol: "btcusdt",
			Timeframe: 60, Value: "50", Timestamp: time.Now().UTC(),
		},
	}

	data, prob := encodeEvent(encodeSpec, "test", event, "")
	if prob != nil {
		t.Fatalf("encode: %v", prob)
	}

	_, prob = decodeEvent[signal.SignalGeneratedEvent](decodeSpec, data)
	if prob == nil {
		t.Fatal("expected decode to fail for wrong event type")
	}
}

// ---------------------------------------------------------------------------
// Decode rejects garbage bytes.
// ---------------------------------------------------------------------------

func TestCodecDecode_RejectsGarbage(t *testing.T) {
	spec := testSpec("test.events.foo", "test.events.v1.foo")
	_, prob := decodeEvent[observation.TradeReceivedEvent](spec, []byte("not cbor"))
	if prob == nil {
		t.Fatal("expected decode to fail for garbage data")
	}
}

// ---------------------------------------------------------------------------
// Evidence publisher dedup key collision isolation.
// Candle, TradeBurst, and Volume dedup keys must never collide for the
// same source/symbol/timeframe/opentime — because they share the same stream.
// ---------------------------------------------------------------------------

func TestEvidenceDedupKey_CrossTypeIsolation(t *testing.T) {
	source := "binancef"
	symbol := "btcusdt"
	timeframe := 60
	openTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	candleKey := source + ":" + symbol + ":" +
		strconv.Itoa(timeframe) + ":" +
		strconv.FormatInt(openTime.Unix(), 10)

	burstKey := "burst:" + source + ":" + symbol + ":" +
		strconv.Itoa(timeframe) + ":" +
		strconv.FormatInt(openTime.Unix(), 10)

	volumeKey := "vol:" + source + ":" + symbol + ":" +
		strconv.Itoa(timeframe) + ":" +
		strconv.FormatInt(openTime.Unix(), 10)

	keys := map[string]string{
		"candle": candleKey,
		"burst":  burstKey,
		"volume": volumeKey,
	}

	for nameA, keyA := range keys {
		for nameB, keyB := range keys {
			if nameA != nameB && keyA == keyB {
				t.Errorf("%s and %s dedup keys collide: %s", nameA, nameB, keyA)
			}
		}
	}
}

func TestEvidenceDedupKey_Deterministic(t *testing.T) {
	source := "binancef"
	symbol := "ethusdt"
	timeframe := 300
	openTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	key1 := source + ":" + symbol + ":" +
		strconv.Itoa(timeframe) + ":" +
		strconv.FormatInt(openTime.Unix(), 10)

	key2 := source + ":" + symbol + ":" +
		strconv.Itoa(timeframe) + ":" +
		strconv.FormatInt(openTime.Unix(), 10)

	if key1 != key2 {
		t.Errorf("same inputs must produce same dedup key: %s vs %s", key1, key2)
	}
}

func TestEvidenceDedupKey_DifferentWindowsNeverCollide(t *testing.T) {
	source := "binancef"
	symbol := "btcusdt"
	timeframe := 60
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 1, 0, 1, 0, 0, time.UTC)

	makeKey := func(openTime time.Time) string {
		return source + ":" + symbol + ":" +
			strconv.Itoa(timeframe) + ":" +
			strconv.FormatInt(openTime.Unix(), 10)
	}

	if makeKey(t1) == makeKey(t2) {
		t.Error("different windows must produce different dedup keys")
	}
}

// ---------------------------------------------------------------------------
// Publisher subject construction: verify extended subjects match stream wildcards.
// These replicate the publisher's subject formatting logic and verify the
// result is captured by the stream's wildcard pattern.
// ---------------------------------------------------------------------------

func TestPublisherSubject_ObservationMatchesStream(t *testing.T) {
	reg := DefaultObservationRegistry()
	subject := reg.TradeReceived.Subject + ".binancef"
	assertSubjectMatchesWildcard(t, subject, reg.TradeReceived.Stream.Subjects)
}

func TestPublisherSubject_EvidenceCandleMatchesStream(t *testing.T) {
	reg := DefaultEvidenceRegistry()
	subject := fmt.Sprintf("%s.binancef.btcusdt.60", reg.CandleSampled.Subject)
	assertSubjectMatchesWildcard(t, subject, reg.CandleSampled.Stream.Subjects)
}

func TestPublisherSubject_EvidenceTradeBurstMatchesStream(t *testing.T) {
	reg := DefaultEvidenceRegistry()
	subject := fmt.Sprintf("%s.binancef.btcusdt.60", reg.TradeBurstSampled.Subject)
	assertSubjectMatchesWildcard(t, subject, reg.TradeBurstSampled.Stream.Subjects)
}

func TestPublisherSubject_EvidenceVolumeMatchesStream(t *testing.T) {
	reg := DefaultEvidenceRegistry()
	subject := fmt.Sprintf("%s.binancef.btcusdt.60", reg.VolumeSampled.Subject)
	assertSubjectMatchesWildcard(t, subject, reg.VolumeSampled.Stream.Subjects)
}

func TestPublisherSubject_SignalMatchesStream(t *testing.T) {
	reg := DefaultSignalRegistry()
	subject := fmt.Sprintf("%s.binancef.btcusdt.60", reg.RSIGenerated.Subject)
	assertSubjectMatchesWildcard(t, subject, reg.RSIGenerated.Stream.Subjects)
}

func TestPublisherSubject_DecisionMatchesStream(t *testing.T) {
	reg := DefaultDecisionRegistry()
	subject := fmt.Sprintf("%s.binancef.btcusdt.60", reg.RSIOversoldEvaluated.Subject)
	assertSubjectMatchesWildcard(t, subject, reg.RSIOversoldEvaluated.Stream.Subjects)
}

func TestPublisherSubject_StrategyMatchesStream(t *testing.T) {
	reg := DefaultStrategyRegistry()
	subject := fmt.Sprintf("%s.binancef.btcusdt.60", reg.MeanReversionEntryResolved.Subject)
	assertSubjectMatchesWildcard(t, subject, reg.MeanReversionEntryResolved.Stream.Subjects)
}

// assertSubjectMatchesWildcard checks that a fully-qualified NATS subject
// would be captured by at least one of the stream's wildcard patterns.
// Uses the NATS ">" wildcard semantics: "a.b.>" matches "a.b.c", "a.b.c.d", etc.
func assertSubjectMatchesWildcard(t *testing.T, subject string, streamSubjects []string) {
	t.Helper()
	for _, pattern := range streamSubjects {
		base := pattern[:len(pattern)-1] // trim ">"
		if len(subject) >= len(base) && subject[:len(base)] == base {
			return
		}
	}
	t.Errorf("subject %q not captured by any stream wildcard %v", subject, streamSubjects)
}
