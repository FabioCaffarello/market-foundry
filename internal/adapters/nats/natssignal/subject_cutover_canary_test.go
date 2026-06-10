//go:build integration

package natssignal_test

// subject_cutover_canary_test.go — H-6.e canary: canonical subject
// token end-to-end against a real NATS server.
//
// Proves the two load-bearing claims of the H-6.e atomic cutover
// (wave prompt Decisões #1/#6):
//
//   - CANARY-1 (canonical path): a signal published through the real
//     Publisher routes with the SubjectToken()-derived {symbol} token
//     (btc_usdt_perpetual) and is received through the same wildcard
//     filter shape the consumers use (`<template>.>`), proving stream
//     subject patterns and consumer filters needed no change.
//   - CANARY-2 (mixed-state): a legacy-token message
//     (…binancef.btcusdt.60 — the pre-cutover grammar) published onto
//     the same stream is received side by side with the canonical
//     one. Mixed-state until the 72h TTL is by design (H-6.d
//     precedent); this is its literal proof at the routing layer.
//
// Requires a running NATS server at localhost:4222 (or NATS_URL).
// Skipped automatically when NATS is unreachable.

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"internal/adapters/nats/natskit"
	"internal/adapters/nats/natssignal"
	"internal/domain/instrument"
	"internal/domain/signal"
	"internal/shared/events"

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
		if h := env[len("nats://"):]; h != "" {
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

func TestSubjectCutover_CanonicalTokenAndMixedState(t *testing.T) {
	url := canaryNATSURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	registry := natssignal.DefaultRegistry()

	pub := natssignal.NewPublisher(url, "h6e-canary", registry)
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

	stream, err := js.Stream(ctx, registry.RSIGenerated.Stream.Name)
	if err != nil {
		t.Fatalf("stream lookup: %v", err)
	}

	// DeliverNew BEFORE publishing: the canary only sees its own two
	// messages, immune to residue from earlier runs within the 72h TTL.
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		FilterSubject: registry.RSIGenerated.Subject + ".>",
		DeliverPolicy: jetstream.DeliverNewPolicy,
	})
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}

	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("instrument.New: %v", prob)
	}

	// CANARY-1: canonical-token publish through the real Publisher.
	event := signal.SignalGeneratedEvent{
		Metadata: events.NewMetadata(),
		Signal: signal.Signal{
			Type:       "rsi",
			Source:     "binancef",
			Instrument: inst,
			Timeframe:  60,
			Value:      "42.0",
			Final:      true,
			Timestamp:  time.Now().UTC(),
		},
	}
	if prob := pub.PublishSignal(ctx, event); prob != nil {
		t.Fatalf("publish canonical: %v", prob)
	}

	// CANARY-2: legacy-token message (pre-cutover grammar) on the
	// same stream — simulates rows still inside the TTL window.
	legacySubject := registry.RSIGenerated.Subject + ".binancef.btcusdt.60"
	if _, err := js.Publish(ctx, legacySubject, []byte(`{"legacy":true}`)); err != nil {
		t.Fatalf("publish legacy: %v", err)
	}

	wantCanonical := registry.RSIGenerated.Subject + ".binancef." + inst.SubjectToken() + ".60"
	got := map[string]bool{}
	deadline := time.Now().Add(15 * time.Second)
	for len(got) < 2 && time.Now().Before(deadline) {
		batch, err := consumer.Fetch(2, jetstream.FetchMaxWait(2*time.Second))
		if err != nil {
			continue
		}
		for msg := range batch.Messages() {
			got[msg.Subject()] = true
			_ = msg.Ack()
		}
	}

	if !got[wantCanonical] {
		t.Errorf("canonical-token subject not received: want %q, got %v", wantCanonical, got)
	}
	if !got[legacySubject] {
		t.Errorf("legacy-token subject not received (mixed-state broken): want %q, got %v", legacySubject, got)
	}
	if tok := inst.SubjectToken(); tok != "btc_usdt_perpetual" {
		t.Errorf("token derivation drifted: %q", tok)
	}
}
