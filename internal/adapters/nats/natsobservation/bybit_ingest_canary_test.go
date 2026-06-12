//go:build integration

package natsobservation_test

// bybit_ingest_canary_test.go — H-7.b canary: the Bybit adapters'
// full normalize→publish path against a real NATS server.
//
// Proves the load-bearing claims of the H-7.b delivery:
//
//   - CANARY-1 (batch): one Bybit v5 publicTrade frame carrying N
//     trades produces N distinct JetStream messages (the per-trade
//     DeduplicationKey — source:TradeID — keeps batch siblings from
//     collapsing inside the 2-minute duplicate window).
//   - CANARY-2 (both sources): bybits (spot) and bybitf (linear
//     perpetual) route on their own source-suffixed subjects through
//     the same wildcard filter shape consumers use, with the
//     canonical instrument (base/quote/contract) inside the payload.
//
// Requires a running NATS server at localhost:4222 (or NATS_URL).
// Skipped automatically when NATS is unreachable.

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"internal/adapters/exchanges/bybitf"
	"internal/adapters/exchanges/bybits"
	"internal/adapters/nats/natskit"
	"internal/adapters/nats/natsobservation"

	"github.com/nats-io/nats.go/jetstream"
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

// Frames carry per-run unique trade IDs: the ObservationTrade dedup
// key is source:TradeID, so a rerun with fixed IDs inside JetStream's
// 2-minute duplicate window would be silently deduplicated and the
// DeliverNew consumer would see nothing (exactly what happened to the
// first draft of this canary).
func canaryFrames(runID int64) (spot, linear string) {
	spot = fmt.Sprintf(`{"topic":"publicTrade.BTCUSDT","type":"snapshot","ts":1770000001000,"data":[
	{"T":1770000001001,"s":"BTCUSDT","S":"Buy","v":"0.010","p":"65000.50","i":"h7b-spot-1-%d"},
	{"T":1770000001002,"s":"BTCUSDT","S":"Sell","v":"0.020","p":"65000.40","i":"h7b-spot-2-%d"}
]}`, runID, runID)
	linear = fmt.Sprintf(`{"topic":"publicTrade.ETHUSDT","type":"snapshot","ts":1770000002000,"data":[
	{"T":1770000002001,"s":"ETHUSDT","S":"Buy","v":"1.5","p":"3500.10","i":"h7b-linear-1-%d"}
]}`, runID)
	return spot, linear
}

func TestBybitIngest_NormalizePublishRoundtrip(t *testing.T) {
	url := canaryNATSURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	registry := natsobservation.DefaultRegistry()
	pub := natsobservation.NewPublisher(url, "h7b-canary", registry)
	if err := pub.Start(); err != nil {
		t.Fatalf("publisher start: %v", err)
	}
	defer func() { _ = pub.Close() }()

	nc, err := natskit.Connect(url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer nc.Close()
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}

	stream, err := js.Stream(ctx, registry.TradeReceived.Stream.Name)
	if err != nil {
		t.Fatalf("stream lookup: %v", err)
	}

	// DeliverNew BEFORE publishing: the canary only sees its own
	// messages, immune to residue from earlier runs inside the TTL.
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		FilterSubject: registry.TradeReceived.Subject + ".>",
		DeliverPolicy: jetstream.DeliverNewPolicy,
	})
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}

	spotFrame, linearFrame := canaryFrames(time.Now().UnixNano())

	// CANARY-1: bybits batch frame → 2 events → 2 publishes.
	frame, ok, prob := bybits.ParsePublicTrade([]byte(spotFrame))
	if prob != nil || !ok {
		t.Fatalf("bybits parse: ok=%v prob=%v", ok, prob)
	}
	spotEvents, prob := bybits.Normalize(frame, "btcusdt")
	if prob != nil {
		t.Fatalf("bybits normalize: %v", prob)
	}
	if len(spotEvents) != 2 {
		t.Fatalf("expected 2 spot events, got %d", len(spotEvents))
	}
	for _, ev := range spotEvents {
		if prob := pub.PublishTrade(ctx, ev); prob != nil {
			t.Fatalf("publish bybits trade: %v", prob)
		}
	}

	// CANARY-2: bybitf frame on the same stream.
	lframe, ok, prob := bybitf.ParsePublicTrade([]byte(linearFrame))
	if prob != nil || !ok {
		t.Fatalf("bybitf parse: ok=%v prob=%v", ok, prob)
	}
	linearEvents, prob := bybitf.Normalize(lframe, "ethusdt")
	if prob != nil {
		t.Fatalf("bybitf normalize: %v", prob)
	}
	if prob := pub.PublishTrade(ctx, linearEvents[0]); prob != nil {
		t.Fatalf("publish bybitf trade: %v", prob)
	}

	spotSubject := registry.TradeReceived.Subject + ".bybits"
	linearSubject := registry.TradeReceived.Subject + ".bybitf"

	type recv struct {
		subject string
		payload string
	}
	var got []recv
	deadline := time.Now().Add(15 * time.Second)
	for len(got) < 3 && time.Now().Before(deadline) {
		batch, err := consumer.Fetch(3, jetstream.FetchMaxWait(2*time.Second))
		if err != nil {
			continue
		}
		for msg := range batch.Messages() {
			got = append(got, recv{subject: msg.Subject(), payload: string(msg.Data())})
			_ = msg.Ack()
		}
	}

	spotCount, linearCount := 0, 0
	for _, r := range got {
		switch r.subject {
		// Envelope payload is CBOR (application/cbor) — string values
		// are embedded raw, so plain substring checks see them.
		case spotSubject:
			spotCount++
			if !strings.Contains(r.payload, "BTC") || !strings.Contains(r.payload, "spot") || !strings.Contains(r.payload, "bybits") {
				t.Errorf("spot payload missing canonical instrument fields: %q", r.payload)
			}
		case linearSubject:
			linearCount++
			if !strings.Contains(r.payload, "ETH") || !strings.Contains(r.payload, "perpetual") || !strings.Contains(r.payload, "bybitf") {
				t.Errorf("linear payload missing canonical instrument fields: %q", r.payload)
			}
		}
	}

	if spotCount != 2 {
		t.Errorf("CANARY-1: expected 2 bybits messages (batch must not collapse in the dedup window), got %d (received: %v)", spotCount, got)
	}
	if linearCount != 1 {
		t.Errorf("CANARY-2: expected 1 bybitf message, got %d (received: %v)", linearCount, got)
	}
}
