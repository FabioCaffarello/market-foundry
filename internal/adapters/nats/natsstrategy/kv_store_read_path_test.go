package natsstrategy_test

import (
	"encoding/json"
	"testing"
	"time"

	"internal/domain/strategy"
)

// S367: Read-path verification — KV round-trip field preservation.
// These tests prove that JSON serialization/deserialization (the format used
// by KVStore.Put / KVStore.Get) preserves all strategy fields without loss.

func referenceStrategy(ts time.Time) strategy.Strategy {
	return strategy.Strategy{
		Type:       "mean_reversion_entry",
		Source:     "binancef",
		Symbol:     "btcusdt",
		Timeframe:  60,
		Direction:  strategy.DirectionLong,
		Confidence: "0.8500",
		Decisions: []strategy.DecisionInput{
			{
				Type:       "rsi_oversold",
				Outcome:    "triggered",
				Confidence: "0.85",
				Severity:   "high",
				Rationale:  "RSI below threshold",
				Timeframe:  60,
			},
		},
		Parameters: map[string]string{
			"entry":         "market",
			"target_offset": "0.02",
			"stop_offset":   "0.01",
		},
		Metadata: map[string]string{
			"resolver": "mean_reversion_entry",
		},
		Final:     true,
		Timestamp: ts,
	}
}

func TestKVRoundTrip_AllFieldsPreserved(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	original := referenceStrategy(now)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var restored strategy.Strategy
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Core identity fields.
	if restored.Type != original.Type {
		t.Errorf("type: want %q, got %q", original.Type, restored.Type)
	}
	if restored.Source != original.Source {
		t.Errorf("source: want %q, got %q", original.Source, restored.Source)
	}
	if restored.Symbol != original.Symbol {
		t.Errorf("symbol: want %q, got %q", original.Symbol, restored.Symbol)
	}
	if restored.Timeframe != original.Timeframe {
		t.Errorf("timeframe: want %d, got %d", original.Timeframe, restored.Timeframe)
	}

	// Directional fields.
	if restored.Direction != original.Direction {
		t.Errorf("direction: want %q, got %q", original.Direction, restored.Direction)
	}
	if restored.Confidence != original.Confidence {
		t.Errorf("confidence: want %q, got %q", original.Confidence, restored.Confidence)
	}

	// Decision inputs.
	if len(restored.Decisions) != len(original.Decisions) {
		t.Fatalf("decisions: want %d, got %d", len(original.Decisions), len(restored.Decisions))
	}
	d := restored.Decisions[0]
	if d.Type != "rsi_oversold" || d.Outcome != "triggered" || d.Severity != "high" || d.Rationale != "RSI below threshold" {
		t.Errorf("decision fields not preserved: %+v", d)
	}

	// Parameters.
	for k, v := range original.Parameters {
		if restored.Parameters[k] != v {
			t.Errorf("parameter %q: want %q, got %q", k, v, restored.Parameters[k])
		}
	}

	// Domain metadata (not event metadata).
	for k, v := range original.Metadata {
		if restored.Metadata[k] != v {
			t.Errorf("metadata %q: want %q, got %q", k, v, restored.Metadata[k])
		}
	}

	// Final flag and timestamp.
	if restored.Final != original.Final {
		t.Errorf("final: want %v, got %v", original.Final, restored.Final)
	}
	if !restored.Timestamp.Equal(original.Timestamp) {
		t.Errorf("timestamp: want %v, got %v", original.Timestamp, restored.Timestamp)
	}
}

func TestKVRoundTrip_PartitionKeyStable(t *testing.T) {
	now := time.Now().UTC()
	original := referenceStrategy(now)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var restored strategy.Strategy
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if restored.PartitionKey() != original.PartitionKey() {
		t.Errorf("partition key drift: want %q, got %q", original.PartitionKey(), restored.PartitionKey())
	}
}

func TestKVRoundTrip_DeduplicationKeyStable(t *testing.T) {
	now := time.Now().UTC()
	original := referenceStrategy(now)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var restored strategy.Strategy
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if restored.DeduplicationKey() != original.DeduplicationKey() {
		t.Errorf("dedup key drift: want %q, got %q", original.DeduplicationKey(), restored.DeduplicationKey())
	}
}

func TestKVRoundTrip_ValidationSurvives(t *testing.T) {
	now := time.Now().UTC()
	original := referenceStrategy(now)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var restored strategy.Strategy
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if prob := restored.Validate(); prob != nil {
		t.Fatalf("restored strategy fails validation: %s", prob.Message)
	}
}

func TestKVRoundTrip_MultiSymbolIsolation(t *testing.T) {
	now := time.Now().UTC()

	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	stored := make(map[string][]byte)

	for _, sym := range symbols {
		strat := referenceStrategy(now)
		strat.Symbol = sym
		data, err := json.Marshal(strat)
		if err != nil {
			t.Fatalf("marshal %s: %v", sym, err)
		}
		stored[strat.PartitionKey()] = data
	}

	if len(stored) != len(symbols) {
		t.Fatalf("expected %d partition keys, got %d", len(symbols), len(stored))
	}

	// Verify each entry restores to its own symbol.
	for key, data := range stored {
		var strat strategy.Strategy
		if err := json.Unmarshal(data, &strat); err != nil {
			t.Fatalf("unmarshal %s: %v", key, err)
		}
		if strat.PartitionKey() != key {
			t.Errorf("partition key mismatch: stored as %q, restored as %q", key, strat.PartitionKey())
		}
	}
}

func TestKVRoundTrip_EventMetadataNotPersisted(t *testing.T) {
	// S367 finding: KV store persists strategy.Strategy, not StrategyResolvedEvent.
	// Event metadata (correlation_id, causation_id) is NOT in the KV payload.
	// This test documents the gap explicitly.
	now := time.Now().UTC()
	strat := referenceStrategy(now)

	data, err := json.Marshal(strat)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	// Confirm event-level metadata fields are absent from KV payload.
	if _, exists := raw["correlation_id"]; exists {
		t.Error("correlation_id found in KV payload — expected absent (event metadata is not persisted)")
	}
	if _, exists := raw["causation_id"]; exists {
		t.Error("causation_id found in KV payload — expected absent (event metadata is not persisted)")
	}
	if _, exists := raw["occurred_at"]; exists {
		t.Error("occurred_at found in KV payload — expected absent (event metadata is not persisted)")
	}

	// The Strategy.Metadata field is domain metadata (e.g. resolver info), not event metadata.
	if raw["metadata"] == nil {
		t.Error("strategy domain metadata should be present")
	}
}
