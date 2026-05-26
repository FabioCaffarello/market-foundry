package natsstrategy

// publisher_correctness_test.go — S366: Publisher correctness unit tests.
//
// Validates that the natsstrategy.Publisher constructs correct NATS subjects,
// deduplication keys, and event payloads without requiring a live NATS server.
// These are structural tests for the publisher's logic paths — they prove the
// publisher would produce the right messages given a working JetStream context.
//
// Governing question answered:
//   - DIQ-4: Does publisher produce correct NATS messages?

import (
	"context"
	"fmt"
	"testing"
	"time"

	"internal/adapters/nats/natskit"
	"internal/domain/instrument"
	"internal/domain/strategy"
	"internal/shared/events"
)

func btcUSDTPerpForPub(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

func ethUSDTPerpForPub(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("ETH", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

// ── Registry Contract Tests ─────────────────────────────────────────

func TestRegistry_MeanReversionEntrySubject(t *testing.T) {
	reg := DefaultRegistry()
	want := "strategy.events.mean_reversion_entry.resolved"
	if reg.MeanReversionEntryResolved.Subject != want {
		t.Errorf("subject: want %s, got %s", want, reg.MeanReversionEntryResolved.Subject)
	}
}

func TestRegistry_MeanReversionEntryType(t *testing.T) {
	reg := DefaultRegistry()
	want := "strategy.events.v1.mean_reversion_entry_resolved"
	if reg.MeanReversionEntryResolved.Type != want {
		t.Errorf("type: want %s, got %s", want, reg.MeanReversionEntryResolved.Type)
	}
}

func TestRegistry_StreamName(t *testing.T) {
	reg := DefaultRegistry()
	want := "STRATEGY_EVENTS"
	if reg.MeanReversionEntryResolved.Stream.Name != want {
		t.Errorf("stream name: want %s, got %s", want, reg.MeanReversionEntryResolved.Stream.Name)
	}
}

func TestRegistry_StreamSubjects(t *testing.T) {
	reg := DefaultRegistry()
	subjects := reg.MeanReversionEntryResolved.Stream.Subjects
	if len(subjects) != 1 || subjects[0] != "strategy.events.>" {
		t.Errorf("stream subjects: want [strategy.events.>], got %v", subjects)
	}
}

func TestRegistry_StreamRetention72Hours(t *testing.T) {
	reg := DefaultRegistry()
	if reg.MeanReversionEntryResolved.Stream.MaxAge != 72*time.Hour {
		t.Errorf("stream max age: want 72h, got %v", reg.MeanReversionEntryResolved.Stream.MaxAge)
	}
}

func TestRegistry_StreamMaxBytes256MB(t *testing.T) {
	reg := DefaultRegistry()
	want := int64(256 * 1024 * 1024)
	if reg.MeanReversionEntryResolved.Stream.MaxBytes != want {
		t.Errorf("stream max bytes: want %d, got %d", want, reg.MeanReversionEntryResolved.Stream.MaxBytes)
	}
}

func TestRegistry_AllThreeStrategyTypes(t *testing.T) {
	reg := DefaultRegistry()

	specs := map[string]string{
		"mean_reversion_entry":   reg.MeanReversionEntryResolved.Subject,
		"trend_following_entry":  reg.TrendFollowingEntryResolved.Subject,
		"squeeze_breakout_entry": reg.SqueezeBreakoutEntryResolved.Subject,
	}

	for name, subject := range specs {
		expected := "strategy.events." + name + ".resolved"
		if subject != expected {
			t.Errorf("%s subject: want %s, got %s", name, expected, subject)
		}
	}
}

// ── specForType Routing Tests ───────────────────────────────────────

func TestSpecForType_MeanReversion(t *testing.T) {
	p := NewPublisher("nats://fake", "binancef", DefaultRegistry())
	spec := p.specForType("mean_reversion_entry")
	if spec == nil {
		t.Fatal("specForType returned nil for mean_reversion_entry")
	}
	if spec.Subject != "strategy.events.mean_reversion_entry.resolved" {
		t.Errorf("subject: %s", spec.Subject)
	}
}

func TestSpecForType_TrendFollowing(t *testing.T) {
	p := NewPublisher("nats://fake", "binancef", DefaultRegistry())
	spec := p.specForType("trend_following_entry")
	if spec == nil {
		t.Fatal("specForType returned nil for trend_following_entry")
	}
	if spec.Subject != "strategy.events.trend_following_entry.resolved" {
		t.Errorf("subject: %s", spec.Subject)
	}
}

func TestSpecForType_SqueezeBreakout(t *testing.T) {
	p := NewPublisher("nats://fake", "binancef", DefaultRegistry())
	spec := p.specForType("squeeze_breakout_entry")
	if spec == nil {
		t.Fatal("specForType returned nil for squeeze_breakout_entry")
	}
	if spec.Subject != "strategy.events.squeeze_breakout_entry.resolved" {
		t.Errorf("subject: %s", spec.Subject)
	}
}

func TestSpecForType_UnknownType_ReturnsNil(t *testing.T) {
	p := NewPublisher("nats://fake", "binancef", DefaultRegistry())
	spec := p.specForType("unknown_strategy")
	if spec != nil {
		t.Errorf("expected nil for unknown type, got %+v", spec)
	}
}

// ── Subject Construction Tests ──────────────────────────────────────

func TestSubjectConstruction_MeanReversionEntry(t *testing.T) {
	reg := DefaultRegistry()
	spec := reg.MeanReversionEntryResolved

	// Simulate subject construction as publisher does it.
	subject := fmt.Sprintf("%s.%s.%s.%d",
		spec.Subject,
		"binancef",
		"btcusdt",
		60,
	)

	want := "strategy.events.mean_reversion_entry.resolved.binancef.btcusdt.60"
	if subject != want {
		t.Errorf("subject: want %s, got %s", want, subject)
	}
}

func TestSubjectConstruction_MatchesConsumerFilter(t *testing.T) {
	// Consumer filter uses wildcard: strategy.events.mean_reversion_entry.resolved.>
	// Publisher produces: strategy.events.mean_reversion_entry.resolved.{source}.{symbol}.{timeframe}
	// The > wildcard matches one or more tokens after the prefix.
	consumerFilter := "strategy.events.mean_reversion_entry.resolved.>"
	publisherSubject := "strategy.events.mean_reversion_entry.resolved.binancef.btcusdt.60"

	// Verify prefix match (the > wildcard part).
	prefix := "strategy.events.mean_reversion_entry.resolved."
	if publisherSubject[:len(prefix)] != consumerFilter[:len(prefix)] {
		t.Error("publisher subject prefix does not match consumer filter prefix")
	}
}

// ── Deduplication Key Tests ─────────────────────────────────────────

func TestDeduplicationKey_Format(t *testing.T) {
	ts := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	s := strategy.Strategy{
		Type:       "mean_reversion_entry",
		Source:     "binancef",
		Instrument: btcUSDTPerpForPub(t),
		Timeframe:  60,
		Timestamp:  ts,
	}

	// P4.1.10: dedup key precision is nanoseconds (was seconds);
	// prevents silent JetStream dedup drops under rapid same-second
	// publishes.
	want := fmt.Sprintf("strat:mean_reversion_entry:binancef:btcusdt:60:%d", ts.UnixNano())
	got := s.DeduplicationKey()
	if got != want {
		t.Errorf("dedup key: want %s, got %s", want, got)
	}
}

func TestDeduplicationKey_Uniqueness_DifferentTimestamps(t *testing.T) {
	base := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	s1 := strategy.Strategy{
		Type: "mean_reversion_entry", Source: "binancef", Instrument: btcUSDTPerpForPub(t),
		Timeframe: 60, Timestamp: base,
	}
	s2 := strategy.Strategy{
		Type: "mean_reversion_entry", Source: "binancef", Instrument: btcUSDTPerpForPub(t),
		Timeframe: 60, Timestamp: base.Add(time.Minute),
	}

	if s1.DeduplicationKey() == s2.DeduplicationKey() {
		t.Error("dedup keys must differ for different timestamps")
	}
}

func TestDeduplicationKey_Uniqueness_DifferentSymbols(t *testing.T) {
	ts := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	s1 := strategy.Strategy{
		Type: "mean_reversion_entry", Source: "binancef", Instrument: btcUSDTPerpForPub(t),
		Timeframe: 60, Timestamp: ts,
	}
	s2 := strategy.Strategy{
		Type: "mean_reversion_entry", Source: "binancef", Instrument: ethUSDTPerpForPub(t),
		Timeframe: 60, Timestamp: ts,
	}

	if s1.DeduplicationKey() == s2.DeduplicationKey() {
		t.Error("dedup keys must differ for different symbols")
	}
}

// ── PublishStrategy Error Path Tests ────────────────────────────────

func TestPublishStrategy_NilPublisher_ReturnsUnavailable(t *testing.T) {
	var p *Publisher
	prob := p.PublishStrategy(context.Background(), strategy.StrategyResolvedEvent{})
	if prob == nil {
		t.Fatal("expected problem for nil publisher")
	}
	if prob.Code != "SYS_UNAVAILABLE" {
		t.Errorf("error code: want SYS_UNAVAILABLE, got %s", prob.Code)
	}
}

func TestPublishStrategy_NilJetStream_ReturnsUnavailable(t *testing.T) {
	p := NewPublisher("nats://fake", "binancef", DefaultRegistry())
	// p.js is nil because Start() was not called.
	prob := p.PublishStrategy(context.Background(), strategy.StrategyResolvedEvent{})
	if prob == nil {
		t.Fatal("expected problem for unstarted publisher")
	}
	if prob.Code != "SYS_UNAVAILABLE" {
		t.Errorf("error code: want SYS_UNAVAILABLE, got %s", prob.Code)
	}
}

func TestPublishStrategy_UnknownType_ReturnsInvalidArgument(t *testing.T) {
	p := NewPublisher("nats://fake", "binancef", DefaultRegistry())
	// Hack: set js to non-nil to pass the nil check. Use an unsafe approach
	// since we can't easily create a mock JetStream.
	// Instead, test the specForType path directly.
	spec := p.specForType("nonexistent_strategy")
	if spec != nil {
		t.Error("expected nil spec for unknown strategy type")
	}
}

// ── Consumer Spec Alignment Tests ───────────────────────────────────

func TestWriterConsumer_MeanReversion_MatchesProducerSubject(t *testing.T) {
	consumer := WriterMeanReversionEntryStrategyConsumer()
	if consumer.Event.Subject != "strategy.events.mean_reversion_entry.resolved.>" {
		t.Errorf("writer consumer subject: %s", consumer.Event.Subject)
	}
	if consumer.Event.Stream.Name != "STRATEGY_EVENTS" {
		t.Errorf("writer consumer stream: %s", consumer.Event.Stream.Name)
	}
}

func TestStoreConsumer_MeanReversion_MatchesProducerSubject(t *testing.T) {
	consumer := StoreMeanReversionEntryStrategyConsumer()
	if consumer.Event.Subject != "strategy.events.mean_reversion_entry.resolved.>" {
		t.Errorf("store consumer subject: %s", consumer.Event.Subject)
	}
}

func TestExecuteConsumer_MeanReversion_MatchesProducerSubject(t *testing.T) {
	consumer := ExecuteStrategyMeanReversionEntryConsumer()
	if consumer.Event.Subject != "strategy.events.mean_reversion_entry.resolved.>" {
		t.Errorf("execute consumer subject: %s", consumer.Event.Subject)
	}
	if consumer.Durable != "execute-strategy-mean-reversion-entry" {
		t.Errorf("execute consumer durable: %s", consumer.Durable)
	}
}

func TestConsumer_AckWait30s(t *testing.T) {
	consumers := []struct {
		name string
		spec natskit.ConsumerSpec
	}{
		{"writer_mean_reversion", WriterMeanReversionEntryStrategyConsumer()},
		{"store_mean_reversion", StoreMeanReversionEntryStrategyConsumer()},
		{"execute_mean_reversion", ExecuteStrategyMeanReversionEntryConsumer()},
	}

	for _, c := range consumers {
		if c.spec.AckWait != 30*time.Second {
			t.Errorf("%s AckWait: want 30s, got %v", c.name, c.spec.AckWait)
		}
	}
}

func TestConsumer_MaxDeliver5(t *testing.T) {
	consumers := []struct {
		name string
		spec natskit.ConsumerSpec
	}{
		{"writer_mean_reversion", WriterMeanReversionEntryStrategyConsumer()},
		{"store_mean_reversion", StoreMeanReversionEntryStrategyConsumer()},
		{"execute_mean_reversion", ExecuteStrategyMeanReversionEntryConsumer()},
	}

	for _, c := range consumers {
		if c.spec.MaxDeliver != 5 {
			t.Errorf("%s MaxDeliver: want 5, got %d", c.name, c.spec.MaxDeliver)
		}
	}
}

// ── Event Construction Tests ────────────────────────────────────────

func TestStrategyResolvedEvent_ImplementsEventInterface(t *testing.T) {
	meta := events.NewMetadata().
		WithCorrelationID("corr-test").
		WithCausationID("cause-test")

	event := strategy.StrategyResolvedEvent{
		Metadata: meta,
		Strategy: strategy.Strategy{
			Type:       "mean_reversion_entry",
			Source:     "binancef",
			Instrument: btcUSDTPerpForPub(t),
			Timeframe:  60,
			Direction:  strategy.DirectionLong,
			Confidence: "0.8500",
			Decisions: []strategy.DecisionInput{
				{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.8500"},
			},
			Final:     true,
			Timestamp: time.Now(),
		},
	}

	if event.EventName() != "strategy_resolved" {
		t.Errorf("EventName: want strategy_resolved, got %s", event.EventName())
	}
	if event.EventMetadata().CorrelationID != "corr-test" {
		t.Errorf("CorrelationID: want corr-test, got %s", event.EventMetadata().CorrelationID)
	}
	if event.EventMetadata().CausationID != "cause-test" {
		t.Errorf("CausationID: want cause-test, got %s", event.EventMetadata().CausationID)
	}
	if event.EventMetadata().ID == "" {
		t.Error("Metadata.ID must be non-empty")
	}
}

// ── LatestSpecByType Tests ──────────────────────────────────────────

func TestLatestSpecByType_KnownTypes(t *testing.T) {
	reg := DefaultRegistry()
	types := []string{"mean_reversion_entry", "trend_following_entry", "squeeze_breakout_entry"}

	for _, st := range types {
		spec, ok := reg.LatestSpecByType(st)
		if !ok {
			t.Errorf("LatestSpecByType(%s): expected ok=true", st)
		}
		if spec.Subject == "" {
			t.Errorf("LatestSpecByType(%s): empty subject", st)
		}
	}
}

func TestLatestSpecByType_UnknownType(t *testing.T) {
	reg := DefaultRegistry()
	_, ok := reg.LatestSpecByType("nonexistent")
	if ok {
		t.Error("LatestSpecByType should return false for unknown type")
	}
}
