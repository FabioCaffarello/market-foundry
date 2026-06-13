//go:build integration

package natsinsights_test

// cross_venue_canary_test.go — H-8.c canary: the cross-venue fusion
// pipeline end-to-end against a real NATS server. A CrossVenueSnapshot
// published to INSIGHTS_EVENTS is consumed by the store-side consumer,
// materialized into the KV latest bucket, and read back through the
// gateway's KV-direct adapter — venues, spread and dominant intact.
//
// Requires a running NATS server (or NATS_URL); skipped when
// unreachable. Reuses canaryNATSURL from volume_profile_canary_test.go.

import (
	"context"
	"testing"
	"time"

	"internal/adapters/nats/natsinsights"
	"internal/application/insightsclient"
	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/shared/events"

	"log/slog"
)

func TestCrossVenuePipeline_PublishConsumeKVRead(t *testing.T) {
	url := canaryNATSURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	registry := natsinsights.DefaultRegistry()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("New: %v", prob)
	}

	kv := natsinsights.NewCrossVenueKVStore(url)
	if err := kv.Start(); err != nil {
		t.Fatalf("kv start: %v", err)
	}
	defer func() { _ = kv.Close() }()
	gw := natsinsights.NewGateway(nil, nil, kv)

	consumer := natsinsights.NewCrossVenueConsumer(url, natsinsights.StoreCrossVenueConsumer(), registry,
		func(ev insights.CrossVenueSampledEvent) {
			_, _ = kv.Put(context.Background(), ev.CrossVenueSnapshot)
		}, slog.Default())
	if err := consumer.Start(); err != nil {
		t.Fatalf("consumer start: %v", err)
	}
	defer func() { _ = consumer.Close() }()

	pub := natsinsights.NewPublisher(url, "h8c-canary", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("publisher start: %v", err)
	}
	defer func() { _ = pub.Close() }()

	openTime := time.Now().UTC().Truncate(time.Minute)
	cv := insights.CrossVenueSnapshot{
		Instrument: inst,
		Timeframe:  60,
		Venues: []insights.VenueRow{
			{Venue: "binancef", TradeCount: 5, Notional: "1000.00000000", LastPrice: "65000", HighPrice: "65020", LowPrice: "64990"},
			{Venue: "bybitf", TradeCount: 3, Notional: "2000.00000000", LastPrice: "65010", HighPrice: "65030", LowPrice: "65000"},
		},
		SpreadAbs:     "10.00000000",
		SpreadBps:     "1.5383",
		MidPrice:      "65005.00000000",
		DominantVenue: "bybitf",
		TradeCount:    8,
		OpenTime:      openTime,
		CloseTime:     openTime.Add(time.Minute),
		Final:         true,
	}

	if prob := pub.PublishCrossVenue(ctx, insights.CrossVenueSampledEvent{
		Metadata:           events.NewMetadata(),
		CrossVenueSnapshot: cv,
	}); prob != nil {
		t.Fatalf("publish: %v", prob)
	}

	deadline := time.Now().Add(15 * time.Second)
	var got *insights.CrossVenueSnapshot
	for time.Now().Before(deadline) {
		reply, prob := gw.GetLatestCrossVenue(ctx, insightsclient.CrossVenueLatestQuery{
			Instrument: inst, Timeframe: 60,
		})
		if prob == nil && reply.CrossVenueSnapshot != nil && reply.CrossVenueSnapshot.OpenTime.Equal(openTime) {
			got = reply.CrossVenueSnapshot
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	if got == nil {
		t.Fatal("cross-venue snapshot did not materialize through publish→consume→KV→read")
	}
	if got.Instrument != inst {
		t.Errorf("instrument mismatch: %+v", got.Instrument)
	}
	if len(got.Venues) != 2 || got.Venues[0].Venue != "binancef" || got.Venues[1].Venue != "bybitf" {
		t.Errorf("venues not preserved end-to-end: %+v", got.Venues)
	}
	if got.DominantVenue != "bybitf" || got.SpreadAbs != "10.00000000" {
		t.Errorf("consolidated metrics not preserved: dominant=%s spread=%s", got.DominantVenue, got.SpreadAbs)
	}
}
