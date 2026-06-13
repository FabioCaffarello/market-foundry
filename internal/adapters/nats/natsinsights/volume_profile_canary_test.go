//go:build integration

package natsinsights_test

// volume_profile_canary_test.go — H-8.a canary: the insights volume
// profile pipeline end-to-end against a real NATS server.
//
// Proves the load-bearing path of the delivery: a VolumeProfile
// published to INSIGHTS_EVENTS is consumed by the store-side
// consumer, materialized into the KV latest bucket, and read back
// through the gateway's KV-direct read adapter — with the canonical
// instrument and all price buckets intact.
//
// Requires a running NATS server at localhost:4222 (or NATS_URL).
// Skipped automatically when NATS is unreachable.

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"internal/adapters/nats/natsinsights"
	"internal/application/insightsclient"
	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/shared/events"

	"log/slog"
)

func canaryNATSURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = "nats://localhost:4222"
	}
	host := "localhost:4222"
	if env := os.Getenv("NATS_URL"); env != "" {
		if h := strings.TrimPrefix(env, "nats://"); h != "" {
			host = h
		}
	}
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		t.Skipf("NATS not reachable at %s: %v", host, err)
	}
	_ = conn.Close()
	return url
}

func TestVolumeProfilePipeline_PublishConsumeKVRead(t *testing.T) {
	url := canaryNATSURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	registry := natsinsights.DefaultRegistry()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("New: %v", prob)
	}

	// KV store (store side) + gateway reader over it.
	kv := natsinsights.NewVolumeProfileKVStore(url)
	if err := kv.Start(); err != nil {
		t.Fatalf("kv start: %v", err)
	}
	defer func() { _ = kv.Close() }()
	gw := natsinsights.NewGateway(kv, nil, nil)

	// Consumer materializes published profiles into the KV bucket
	// (the store projection's behavior, inline here).
	consumer := natsinsights.NewVolumeProfileConsumer(url, natsinsights.StoreVolumeProfileConsumer(), registry,
		func(ev insights.VolumeProfileSampledEvent) {
			_, _ = kv.Put(context.Background(), ev.VolumeProfile)
		}, slog.Default())
	if err := consumer.Start(); err != nil {
		t.Fatalf("consumer start: %v", err)
	}
	defer func() { _ = consumer.Close() }()

	pub := natsinsights.NewPublisher(url, "h8a-canary", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("publisher start: %v", err)
	}
	defer func() { _ = pub.Close() }()

	// Per-run unique window (avoid dedup against earlier runs inside
	// the stream TTL).
	openTime := time.Now().UTC().Truncate(time.Minute)
	vp := insights.VolumeProfile{
		Source:     "binancef",
		Instrument: inst,
		Timeframe:  60,
		BucketSize: "1",
		Buckets: []insights.PriceBucket{
			{PriceLevel: "65000", BuyVolume: "1200.50000000", SellVolume: "800.00000000"},
			{PriceLevel: "65010", BuyVolume: "300.00000000", SellVolume: "450.25000000"},
		},
		TradeCount: 7,
		Overload:   insights.OverloadL0,
		OpenTime:   openTime,
		CloseTime:  openTime.Add(time.Minute),
		Final:      true,
	}

	if prob := pub.PublishVolumeProfile(ctx, insights.VolumeProfileSampledEvent{
		Metadata:      events.NewMetadata(),
		VolumeProfile: vp,
	}); prob != nil {
		t.Fatalf("publish: %v", prob)
	}

	// Poll the gateway read path until the profile materializes.
	deadline := time.Now().Add(15 * time.Second)
	var got *insights.VolumeProfile
	for time.Now().Before(deadline) {
		reply, prob := gw.GetLatestVolumeProfile(ctx, insightsclient.VolumeProfileLatestQuery{
			Source: "binancef", Instrument: inst, Timeframe: 60,
		})
		if prob == nil && reply.VolumeProfile != nil && reply.VolumeProfile.OpenTime.Equal(openTime) {
			got = reply.VolumeProfile
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	if got == nil {
		t.Fatal("volume profile did not materialize through publish→consume→KV→read")
	}
	if got.Instrument != inst {
		t.Errorf("instrument mismatch: %+v", got.Instrument)
	}
	if len(got.Buckets) != 2 || got.Buckets[0].PriceLevel != "65000" || got.Buckets[1].PriceLevel != "65010" {
		t.Errorf("buckets not preserved end-to-end: %+v", got.Buckets)
	}
	if got.TradeCount != 7 {
		t.Errorf("trade count = %d, want 7", got.TradeCount)
	}
}
