package executionclient_test

import (
	"internal/domain/instrument"

	"encoding/json"
	"testing"

	"internal/application/executionclient"
)

func TestS466_LifecycleListQuery_SerializesFilters(t *testing.T) {
	q := executionclient.LifecycleListQuery{
		Source:     "binancef",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
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
	if decoded.Instrument.SubjectToken() != "btc_usdt_perpetual" {
		t.Fatalf("Instrument token = %q, want %q", decoded.Instrument.SubjectToken(), "btc_usdt_perpetual")
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
