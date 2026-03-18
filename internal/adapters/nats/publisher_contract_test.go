package nats

import (
	"context"
	"testing"
	"time"

	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/observation"
	"internal/domain/signal"
	"internal/domain/strategy"
	"internal/shared/events"
	"internal/shared/problem"
)

// ---------------------------------------------------------------------------
// Publisher nil guard: calling Publish on an unstarted publisher must return
// an Unavailable problem without panic.
// ---------------------------------------------------------------------------

func TestObservationPublisher_NilGuard(t *testing.T) {
	pub := NewObservationPublisher("nats://unused", "test", DefaultObservationRegistry())
	// Not started — js is nil.
	prob := pub.PublishTrade(context.Background(), observation.TradeReceivedEvent{
		Metadata: events.NewMetadata(),
		Trade: observation.ObservationTrade{
			Source: "binancef", Symbol: "btcusdt", Price: "1", Quantity: "1",
			TradeID: "t1", Timestamp: time.Now().UTC(),
		},
	})
	if prob == nil {
		t.Fatal("expected problem from unstarted publisher")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}

func TestEvidencePublisher_NilGuard_Candle(t *testing.T) {
	pub := NewEvidencePublisher("nats://unused", "test", DefaultEvidenceRegistry())
	now := time.Now().UTC()
	prob := pub.PublishCandle(context.Background(), evidence.CandleSampledEvent{
		Metadata: events.NewMetadata(),
		Candle: evidence.EvidenceCandle{
			Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
			Open: "1", High: "2", Low: "0.5", Close: "1.5", Volume: "100",
			OpenTime: now, CloseTime: now.Add(time.Minute),
		},
	})
	if prob == nil {
		t.Fatal("expected problem from unstarted publisher")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}

func TestEvidencePublisher_NilGuard_TradeBurst(t *testing.T) {
	pub := NewEvidencePublisher("nats://unused", "test", DefaultEvidenceRegistry())
	now := time.Now().UTC()
	prob := pub.PublishTradeBurst(context.Background(), evidence.TradeBurstSampledEvent{
		Metadata: events.NewMetadata(),
		TradeBurst: evidence.EvidenceTradeBurst{
			Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
			BuyVolume: "100", SellVolume: "50",
			OpenTime: now, CloseTime: now.Add(time.Minute),
		},
	})
	if prob == nil {
		t.Fatal("expected problem from unstarted publisher")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}

func TestEvidencePublisher_NilGuard_Volume(t *testing.T) {
	pub := NewEvidencePublisher("nats://unused", "test", DefaultEvidenceRegistry())
	now := time.Now().UTC()
	prob := pub.PublishVolume(context.Background(), evidence.VolumeSampledEvent{
		Metadata: events.NewMetadata(),
		Volume: evidence.EvidenceVolume{
			Source: "binancef", Symbol: "btcusdt", Timeframe: 60,
			BuyVolume: "100", SellVolume: "50", TotalVolume: "150", VWAP: "65000",
			OpenTime: now, CloseTime: now.Add(time.Minute),
		},
	})
	if prob == nil {
		t.Fatal("expected problem from unstarted publisher")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}

func TestSignalPublisher_NilGuard(t *testing.T) {
	pub := NewSignalPublisher("nats://unused", "test", DefaultSignalRegistry())
	prob := pub.PublishSignal(context.Background(), signal.SignalGeneratedEvent{
		Metadata: events.NewMetadata(),
		Signal: signal.Signal{
			Type: "rsi", Source: "binancef", Symbol: "btcusdt",
			Timeframe: 60, Value: "30", Timestamp: time.Now().UTC(),
		},
	})
	if prob == nil {
		t.Fatal("expected problem from unstarted publisher")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}

func TestDecisionPublisher_NilGuard(t *testing.T) {
	pub := NewDecisionPublisher("nats://unused", "test", DefaultDecisionRegistry())
	prob := pub.PublishDecision(context.Background(), decision.DecisionEvaluatedEvent{
		Metadata: events.NewMetadata(),
		Decision: decision.Decision{
			Type: "rsi_oversold", Source: "binancef", Symbol: "btcusdt",
			Timeframe: 60, Outcome: decision.OutcomeTriggered, Confidence: "0.8",
			Timestamp: time.Now().UTC(),
		},
	})
	if prob == nil {
		t.Fatal("expected problem from unstarted publisher")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}

func TestStrategyPublisher_NilGuard(t *testing.T) {
	pub := NewStrategyPublisher("nats://unused", "test", DefaultStrategyRegistry())
	prob := pub.PublishStrategy(context.Background(), strategy.StrategyResolvedEvent{
		Metadata: events.NewMetadata(),
		Strategy: strategy.Strategy{
			Type: "mean_reversion_entry", Source: "binancef", Symbol: "btcusdt",
			Timeframe: 60, Direction: strategy.DirectionLong, Confidence: "0.7",
			Decisions: []strategy.DecisionInput{{Type: "rsi_oversold", Outcome: "triggered", Confidence: "0.8", Timeframe: 60}},
			Timestamp: time.Now().UTC(),
		},
	})
	if prob == nil {
		t.Fatal("expected problem from unstarted publisher")
	}
	if prob.Code != problem.Unavailable {
		t.Errorf("expected Unavailable, got %s", prob.Code)
	}
}

// ---------------------------------------------------------------------------
// Nil publisher: calling Publish on a nil pointer must not panic.
// ---------------------------------------------------------------------------

func TestObservationPublisher_NilPointer(t *testing.T) {
	var pub *ObservationPublisher
	prob := pub.PublishTrade(context.Background(), observation.TradeReceivedEvent{})
	if prob == nil {
		t.Fatal("expected problem from nil publisher")
	}
}

func TestEvidencePublisher_NilPointer(t *testing.T) {
	var pub *EvidencePublisher
	prob := pub.PublishCandle(context.Background(), evidence.CandleSampledEvent{})
	if prob == nil {
		t.Fatal("expected problem from nil publisher")
	}
}

func TestSignalPublisher_NilPointer(t *testing.T) {
	var pub *SignalPublisher
	prob := pub.PublishSignal(context.Background(), signal.SignalGeneratedEvent{})
	if prob == nil {
		t.Fatal("expected problem from nil publisher")
	}
}

func TestDecisionPublisher_NilPointer(t *testing.T) {
	var pub *DecisionPublisher
	prob := pub.PublishDecision(context.Background(), decision.DecisionEvaluatedEvent{})
	if prob == nil {
		t.Fatal("expected problem from nil publisher")
	}
}

func TestStrategyPublisher_NilPointer(t *testing.T) {
	var pub *StrategyPublisher
	prob := pub.PublishStrategy(context.Background(), strategy.StrategyResolvedEvent{})
	if prob == nil {
		t.Fatal("expected problem from nil publisher")
	}
}

// ---------------------------------------------------------------------------
// specForType routing: signal, decision, strategy publishers must route
// known types to the correct spec and reject unknown types.
// ---------------------------------------------------------------------------

func TestSignalPublisher_SpecForType(t *testing.T) {
	pub := NewSignalPublisher("nats://unused", "test", DefaultSignalRegistry())

	t.Run("rsi routes to RSIGenerated spec", func(t *testing.T) {
		spec := pub.specForType("rsi")
		if spec == nil {
			t.Fatal("expected spec for rsi")
		}
		if spec.Subject != pub.registry.RSIGenerated.Subject {
			t.Errorf("subject mismatch: want %s, got %s", pub.registry.RSIGenerated.Subject, spec.Subject)
		}
	})

	t.Run("unknown type returns nil", func(t *testing.T) {
		if pub.specForType("macd") != nil {
			t.Error("expected nil for unknown signal type")
		}
	})

	t.Run("empty type returns nil", func(t *testing.T) {
		if pub.specForType("") != nil {
			t.Error("expected nil for empty signal type")
		}
	})
}

func TestDecisionPublisher_SpecForType(t *testing.T) {
	pub := NewDecisionPublisher("nats://unused", "test", DefaultDecisionRegistry())

	t.Run("rsi_oversold routes to RSIOversoldEvaluated spec", func(t *testing.T) {
		spec := pub.specForType("rsi_oversold")
		if spec == nil {
			t.Fatal("expected spec for rsi_oversold")
		}
		if spec.Subject != pub.registry.RSIOversoldEvaluated.Subject {
			t.Errorf("subject mismatch: want %s, got %s", pub.registry.RSIOversoldEvaluated.Subject, spec.Subject)
		}
	})

	t.Run("unknown type returns nil", func(t *testing.T) {
		if pub.specForType("bollinger_squeeze") != nil {
			t.Error("expected nil for unknown decision type")
		}
	})
}

func TestStrategyPublisher_SpecForType(t *testing.T) {
	pub := NewStrategyPublisher("nats://unused", "test", DefaultStrategyRegistry())

	t.Run("mean_reversion_entry routes to correct spec", func(t *testing.T) {
		spec := pub.specForType("mean_reversion_entry")
		if spec == nil {
			t.Fatal("expected spec for mean_reversion_entry")
		}
		if spec.Subject != pub.registry.MeanReversionEntryResolved.Subject {
			t.Errorf("subject mismatch: want %s, got %s", pub.registry.MeanReversionEntryResolved.Subject, spec.Subject)
		}
	})

	t.Run("unknown type returns nil", func(t *testing.T) {
		if pub.specForType("momentum_breakout") != nil {
			t.Error("expected nil for unknown strategy type")
		}
	})
}

// ---------------------------------------------------------------------------
// Publisher Close: closing an unstarted publisher must not panic.
// ---------------------------------------------------------------------------

func TestPublisher_Close_Unstarted(t *testing.T) {
	t.Run("observation", func(t *testing.T) {
		pub := NewObservationPublisher("nats://unused", "test", DefaultObservationRegistry())
		if err := pub.Close(); err != nil {
			t.Errorf("close unstarted: %v", err)
		}
	})
	t.Run("evidence", func(t *testing.T) {
		pub := NewEvidencePublisher("nats://unused", "test", DefaultEvidenceRegistry())
		if err := pub.Close(); err != nil {
			t.Errorf("close unstarted: %v", err)
		}
	})
	t.Run("signal", func(t *testing.T) {
		pub := NewSignalPublisher("nats://unused", "test", DefaultSignalRegistry())
		if err := pub.Close(); err != nil {
			t.Errorf("close unstarted: %v", err)
		}
	})
	t.Run("decision", func(t *testing.T) {
		pub := NewDecisionPublisher("nats://unused", "test", DefaultDecisionRegistry())
		if err := pub.Close(); err != nil {
			t.Errorf("close unstarted: %v", err)
		}
	})
	t.Run("strategy", func(t *testing.T) {
		pub := NewStrategyPublisher("nats://unused", "test", DefaultStrategyRegistry())
		if err := pub.Close(); err != nil {
			t.Errorf("close unstarted: %v", err)
		}
	})
}
