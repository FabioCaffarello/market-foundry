//go:build integration

package delivery_test

// delivery_snapshot_canary_test.go — H-11.f canary: snapshot-then-delta
// end-to-end against a real NATS server.
//
// Proves the load-bearing path: a client subscribing to a fully-specified
// insights subject receives the current KV-latest as a SNAPSHOT frame
// first, then live deltas. The snapshot is seeded straight into the KV
// bucket (so it is visible only via the SnapshotProvider, not the event
// stream); the delta is published to INSIGHTS_EVENTS (so it flows via the
// durable consumer). They are distinguished by open_time.
//
// Requires a running NATS server at localhost:4222 (or NATS_URL).

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"log/slog"

	"internal/actors/scopes/delivery"
	"internal/adapters/nats/natsdelivery"
	"internal/adapters/nats/natsinsights"
	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/shared/events"

	"github.com/anthdm/hollywood/actor"
)

func decodeVPOpenTime(t *testing.T, raw []byte) (string, time.Time) {
	t.Helper()
	var w struct {
		Event struct {
			VolumeProfile struct {
				Source   string    `json:"source"`
				OpenTime time.Time `json:"open_time"`
			} `json:"volume_profile"`
		} `json:"event"`
	}
	if err := json.Unmarshal(raw, &w); err != nil {
		t.Fatalf("decode frame: %v (raw=%s)", err, raw)
	}
	return w.Event.VolumeProfile.Source, w.Event.VolumeProfile.OpenTime
}

func TestDeliverySnapshotThenDelta(t *testing.T) {
	url := canaryNATSURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("instrument.New: %v", prob)
	}
	source := fmt.Sprintf("h11fsnap%d", time.Now().UnixNano())
	tf := 60
	subject := "insights.events.volumeprofile.sampled." + source + "." + inst.SubjectToken() + "." + strconv.Itoa(tf)

	// 1. Seed the KV-latest with the snapshot VP (older open_time).
	kv := natsinsights.NewVolumeProfileKVStore(url)
	if err := kv.Start(); err != nil {
		t.Fatalf("kv start: %v", err)
	}
	defer func() { _ = kv.Close() }()
	snapOpen := time.Now().UTC().Truncate(time.Minute).Add(-time.Hour)
	if _, prob := kv.Put(ctx, insights.VolumeProfile{
		Source: source, Instrument: inst, Timeframe: tf, BucketSize: "1",
		OpenTime: snapOpen, CloseTime: snapOpen.Add(time.Minute), Final: true,
		Buckets: []insights.PriceBucket{{PriceLevel: "100", BuyVolume: "1", SellVolume: "1"}},
	}); prob != nil {
		t.Fatalf("kv put: %v", prob)
	}

	// 2. Delivery runtime with the KV-backed snapshot provider.
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	cfg := delivery.DefaultConfig()
	cfg.SnapshotProvider = natsdelivery.NewKVSnapshotProvider(natsinsights.NewGateway(kv, nil, nil))
	rt, err := delivery.Start(engine, url, cfg, slog.Default())
	if err != nil {
		t.Fatalf("delivery start: %v", err)
	}
	defer func() { _ = rt.Close() }()

	// 3. Subscribe → first frame must be the KV snapshot (snapOpen).
	conn := newCanaryConn()
	h := rt.Hub.Admit(conn)
	defer h.Close()
	h.Subscribe(subject)

	select {
	case raw := <-conn.frames:
		src, ot := decodeVPOpenTime(t, raw)
		if src != source || !ot.Equal(snapOpen) {
			t.Fatalf("first frame not the snapshot: source=%q open_time=%v (want %q / %v)", src, ot, source, snapOpen)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("did not receive the snapshot frame on subscribe")
	}

	// 4. Publish a live delta (newer open_time) → second frame must be it.
	pub := natsinsights.NewPublisher(url, source, natsinsights.DefaultRegistry())
	if err := pub.Start(); err != nil {
		t.Fatalf("publisher start: %v", err)
	}
	defer func() { _ = pub.Close() }()
	deltaOpen := time.Now().UTC().Truncate(time.Minute)
	if prob := pub.PublishVolumeProfile(ctx, insights.VolumeProfileSampledEvent{
		Metadata: events.NewMetadata(),
		VolumeProfile: insights.VolumeProfile{
			Source: source, Instrument: inst, Timeframe: tf, BucketSize: "1",
			OpenTime: deltaOpen, CloseTime: deltaOpen.Add(time.Minute), Final: true,
		},
	}); prob != nil {
		t.Fatalf("publish delta: %v", prob)
	}

	select {
	case raw := <-conn.frames:
		src, ot := decodeVPOpenTime(t, raw)
		if src != source || !ot.Equal(deltaOpen) {
			t.Fatalf("second frame not the delta: source=%q open_time=%v (want %q / %v)", src, ot, source, deltaOpen)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("did not receive the live delta frame after the snapshot")
	}
}
