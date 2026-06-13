//go:build integration

package delivery_test

// delivery_ws_canary_test.go — H-11.a canary: the delivery pipeline
// end-to-end against a real NATS server.
//
// Proves the load-bearing path: a VolumeProfile published to
// INSIGHTS_EVENTS is consumed by the durable delivery consumer
// ('deliver-insights'), fanned out by the router, matched by a
// subscribed session, and written to the client connection as a JSON
// frame — connect → subscribe → receive one volume profile.
//
// A per-run unique source makes the session's subscription match ONLY
// this run's event, so historical events replayed by the durable never
// flood (or DropNewest-evict) the asserted frame.
//
// Requires a running NATS server at localhost:4222 (or NATS_URL).
// Skipped automatically when NATS is unreachable.

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"log/slog"

	"internal/actors/scopes/delivery"
	"internal/adapters/nats/natsinsights"
	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/shared/events"

	"github.com/anthdm/hollywood/actor"
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

// canaryConn records delivered frames on a channel (a fake client).
type canaryConn struct{ frames chan []byte }

func newCanaryConn() *canaryConn { return &canaryConn{frames: make(chan []byte, 64)} }

func (c *canaryConn) Send(frame []byte) error {
	cp := make([]byte, len(frame))
	copy(cp, frame)
	c.frames <- cp
	return nil
}

func (c *canaryConn) Close() error { return nil }

func TestDeliveryWS_PublishSubscribeReceive(t *testing.T) {
	url := canaryNATSURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	rt, err := delivery.Start(engine, url, delivery.DefaultConfig(), slog.Default())
	if err != nil {
		t.Fatalf("delivery start: %v", err)
	}
	defer func() { _ = rt.Close() }()

	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("instrument.New: %v", prob)
	}

	// Per-run unique source → unique subject → the session matches only
	// this event (no history flood / DropNewest eviction of the assert).
	source := fmt.Sprintf("h11acanary%d", time.Now().UnixNano())
	pattern := "insights.events.volumeprofile.sampled." + source + ".>"

	conn := newCanaryConn()
	handle := rt.Hub.Admit(conn)
	defer handle.Close()
	handle.Subscribe(pattern)
	// Subscribe is async via the actor mailbox; let it register before
	// publishing.
	time.Sleep(200 * time.Millisecond)

	pub := natsinsights.NewPublisher(url, source, natsinsights.DefaultRegistry())
	if err := pub.Start(); err != nil {
		t.Fatalf("publisher start: %v", err)
	}
	defer func() { _ = pub.Close() }()

	openTime := time.Now().UTC().Truncate(time.Minute)
	vp := insights.VolumeProfile{
		Source:     source,
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

	var raw []byte
	select {
	case raw = <-conn.frames:
	case <-time.After(20 * time.Second):
		t.Fatal("did not receive the volume profile frame over delivery")
	}

	var frame struct {
		Subject string `json:"subject"`
		Event   struct {
			VolumeProfile struct {
				Source    string    `json:"source"`
				Timeframe int       `json:"timeframe"`
				OpenTime  time.Time `json:"open_time"`
				Buckets   []struct {
					PriceLevel string `json:"price_level"`
				} `json:"buckets"`
			} `json:"volume_profile"`
		} `json:"event"`
	}
	if err := json.Unmarshal(raw, &frame); err != nil {
		t.Fatalf("decode delivered frame: %v (raw=%s)", err, raw)
	}
	if !strings.HasPrefix(frame.Subject, "insights.events.volumeprofile.sampled."+source+".") {
		t.Errorf("delivered subject = %q, want volumeprofile family for source %q", frame.Subject, source)
	}
	if frame.Event.VolumeProfile.Source != source {
		t.Errorf("delivered source = %q, want %q", frame.Event.VolumeProfile.Source, source)
	}
	if !frame.Event.VolumeProfile.OpenTime.Equal(openTime) {
		t.Errorf("delivered open_time = %v, want %v", frame.Event.VolumeProfile.OpenTime, openTime)
	}
	if len(frame.Event.VolumeProfile.Buckets) != 2 ||
		frame.Event.VolumeProfile.Buckets[0].PriceLevel != "65000" ||
		frame.Event.VolumeProfile.Buckets[1].PriceLevel != "65010" {
		t.Errorf("delivered buckets not preserved: %+v", frame.Event.VolumeProfile.Buckets)
	}
}
