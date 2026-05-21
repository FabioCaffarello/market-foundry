package executionclient_test

import (
	"encoding/json"
	"testing"

	"internal/application/executionclient"
)

func TestS466_LifecycleListQuery_SerializesFilters(t *testing.T) {
	q := executionclient.LifecycleListQuery{
		Source: "binancef",
		Symbol: "btcusdt",
	}

	data, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded executionclient.LifecycleListQuery
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Source != "binancef" {
		t.Fatalf("Source = %q, want %q", decoded.Source, "binancef")
	}
	if decoded.Symbol != "btcusdt" {
		t.Fatalf("Symbol = %q, want %q", decoded.Symbol, "btcusdt")
	}
}

func TestS466_LifecycleListQuery_EmptyOmitsFields(t *testing.T) {
	q := executionclient.LifecycleListQuery{}

	data, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// With omitempty, empty strings should not appear in JSON.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	if _, ok := raw["source"]; ok {
		t.Fatal("empty source should be omitted from JSON")
	}
	if _, ok := raw["symbol"]; ok {
		t.Fatal("empty symbol should be omitted from JSON")
	}
}
