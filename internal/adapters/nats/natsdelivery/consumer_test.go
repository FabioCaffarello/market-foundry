package natsdelivery

import (
	"encoding/json"
	"testing"

	"internal/adapters/nats/natsinsights"
	"internal/adapters/nats/natskit"
	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/shared/events"
	"internal/shared/problem"
)

// TestDecodeToJSON_PerFamily proves the multi-family decode path: a CBOR
// insights envelope of each family round-trips through decodeToJSON into
// valid JSON for the client wire. The subject-dispatch switch that
// selects the spec is exercised end-to-end by the integration canaries.
func TestDecodeToJSON_PerFamily(t *testing.T) {
	reg := natsinsights.DefaultRegistry()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("instrument.New: %v", prob)
	}

	t.Run("volume_profile", func(t *testing.T) {
		data := mustEncode(t, reg.VolumeProfileSampled, insights.VolumeProfileSampledEvent{
			Metadata: events.NewMetadata(),
			VolumeProfile: insights.VolumeProfile{
				Source: "s", Instrument: inst, Timeframe: 60, BucketSize: "1",
			},
		})
		out, prob := decodeToJSON[insights.VolumeProfileSampledEvent](reg.VolumeProfileSampled, data)
		assertValidJSON(t, out, prob)
	})

	t.Run("tpo", func(t *testing.T) {
		data := mustEncode(t, reg.TPOProfileSampled, insights.TPOProfileSampledEvent{
			Metadata:   events.NewMetadata(),
			TPOProfile: insights.TPOProfile{Source: "s", Instrument: inst, Timeframe: 60},
		})
		out, prob := decodeToJSON[insights.TPOProfileSampledEvent](reg.TPOProfileSampled, data)
		assertValidJSON(t, out, prob)
	})

	t.Run("cross_venue", func(t *testing.T) {
		data := mustEncode(t, reg.CrossVenueSampled, insights.CrossVenueSampledEvent{
			Metadata:           events.NewMetadata(),
			CrossVenueSnapshot: insights.CrossVenueSnapshot{Instrument: inst, Timeframe: 60},
		})
		out, prob := decodeToJSON[insights.CrossVenueSampledEvent](reg.CrossVenueSampled, data)
		assertValidJSON(t, out, prob)
	})
}

func mustEncode[T any](t *testing.T, spec natskit.EventSpec, payload T) []byte {
	t.Helper()
	data, prob := natskit.EncodeEvent(spec, "test", payload, "", "")
	if prob != nil {
		t.Fatalf("EncodeEvent: %v", prob)
	}
	return data
}

func assertValidJSON(t *testing.T, out []byte, prob *problem.Problem) {
	t.Helper()
	if prob != nil {
		t.Fatalf("decodeToJSON: %v", prob)
	}
	if len(out) == 0 || !json.Valid(out) {
		t.Fatalf("decodeToJSON produced invalid JSON: %q", out)
	}
}
