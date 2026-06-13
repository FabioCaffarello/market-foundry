//go:build integration

package natsinsights_test

// tpo_canary_test.go — H-8.b canary: the insights TPO pipeline
// end-to-end against a real NATS server.
//
// Proves the load-bearing path: a TPOProfile published to
// INSIGHTS_EVENTS is consumed by the store-side consumer, materialized
// into the KV latest bucket, and read back through the gateway's
// KV-direct read adapter — with the canonical instrument, periods,
// levels and POC intact.
//
// Requires a running NATS server at localhost:4222 (or NATS_URL).
// Skipped automatically when NATS is unreachable. Reuses canaryNATSURL
// from volume_profile_canary_test.go (same package).

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

func TestTPOProfilePipeline_PublishConsumeKVRead(t *testing.T) {
	url := canaryNATSURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	registry := natsinsights.DefaultRegistry()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("New: %v", prob)
	}

	kv := natsinsights.NewTPOKVStore(url)
	if err := kv.Start(); err != nil {
		t.Fatalf("kv start: %v", err)
	}
	defer func() { _ = kv.Close() }()
	gw := natsinsights.NewGateway(nil, kv)

	consumer := natsinsights.NewTPOConsumer(url, natsinsights.StoreTPOConsumer(), registry,
		func(ev insights.TPOProfileSampledEvent) {
			_, _ = kv.Put(context.Background(), ev.TPOProfile)
		}, slog.Default())
	if err := consumer.Start(); err != nil {
		t.Fatalf("consumer start: %v", err)
	}
	defer func() { _ = consumer.Close() }()

	pub := natsinsights.NewPublisher(url, "h8b-canary", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("publisher start: %v", err)
	}
	defer func() { _ = pub.Close() }()

	openTime := time.Now().UTC().Truncate(time.Hour)
	tp := insights.TPOProfile{
		Source:        "binancef",
		Instrument:    inst,
		Timeframe:     3600,
		BucketSize:    "1",
		PeriodSeconds: 600,
		Periods: []insights.TPOPeriod{
			{Letter: "A", StartTime: openTime, EndTime: openTime.Add(10 * time.Minute), HighPrice: "65020", LowPrice: "65000"},
			{Letter: "B", StartTime: openTime.Add(10 * time.Minute), EndTime: openTime.Add(20 * time.Minute), HighPrice: "65040", LowPrice: "65010"},
		},
		Levels: []insights.TPOLevel{
			{PriceLevel: "65000", Letters: "A", Count: 1},
			{PriceLevel: "65010", Letters: "AB", Count: 2},
		},
		POCPrice:   "65010",
		TradeCount: 9,
		Overload:   insights.OverloadL0,
		OpenTime:   openTime,
		CloseTime:  openTime.Add(time.Hour),
		Final:      true,
	}

	if prob := pub.PublishTPOProfile(ctx, insights.TPOProfileSampledEvent{
		Metadata:   events.NewMetadata(),
		TPOProfile: tp,
	}); prob != nil {
		t.Fatalf("publish: %v", prob)
	}

	deadline := time.Now().Add(15 * time.Second)
	var got *insights.TPOProfile
	for time.Now().Before(deadline) {
		reply, prob := gw.GetLatestTPOProfile(ctx, insightsclient.TPOProfileLatestQuery{
			Source: "binancef", Instrument: inst, Timeframe: 3600,
		})
		if prob == nil && reply.TPOProfile != nil && reply.TPOProfile.OpenTime.Equal(openTime) {
			got = reply.TPOProfile
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	if got == nil {
		t.Fatal("tpo profile did not materialize through publish→consume→KV→read")
	}
	if got.Instrument != inst {
		t.Errorf("instrument mismatch: %+v", got.Instrument)
	}
	if len(got.Periods) != 2 || got.Periods[0].Letter != "A" || got.Periods[1].Letter != "B" {
		t.Errorf("periods not preserved end-to-end: %+v", got.Periods)
	}
	if len(got.Levels) != 2 || got.Levels[1].Letters != "AB" {
		t.Errorf("levels not preserved end-to-end: %+v", got.Levels)
	}
	if got.POCPrice != "65010" {
		t.Errorf("poc = %s, want 65010", got.POCPrice)
	}
}
