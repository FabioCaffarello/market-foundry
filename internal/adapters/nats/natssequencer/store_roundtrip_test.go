//go:build integration

package natssequencer_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"internal/adapters/nats/natssequencer"
	"internal/shared/sequencer"
)

// store_roundtrip_test.go — integration coverage for Store
// against a real NATS+JetStream server. Validates the
// SEQUENCER_STATE_LATEST bucket lifecycle (declare → save →
// restore → resume) and the ADR-0020 owner isolation
// (LoadSnapshot returns only entries for this Store's
// ownerBinary, ignoring entries owned by other writers).
//
// Requires a running NATS server at localhost:4222 (or NATS_URL
// env var). Skipped automatically when NATS is unreachable.

func natsURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = "nats://localhost:4222"
	}
	host := "localhost:4222"
	if env := os.Getenv("NATS_URL"); env != "" {
		h := env[len("nats://"):]
		if h != "" {
			host = h
		}
	}
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		t.Skipf("NATS not reachable at %s: %v", host, err)
	}
	conn.Close()
	return url
}

func TestStore_SaveAndLoadRoundtrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := natssequencer.NewStore(natsURL(t), "ingest")
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Close()

	in := map[sequencer.StreamKey]int64{
		{Venue: "binance", Instrument: "btcusdt", EventType: "observation.trade"}:         1024,
		{Venue: "binance", Instrument: "ethusdt", EventType: "observation.trade"}:         512,
		{Venue: "binance", Instrument: "btcusdt", EventType: "observation.book.snapshot"}: 77,
	}
	if err := s.SaveSnapshot(ctx, in); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	out, err := s.LoadSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if len(out) != len(in) {
		t.Errorf("loaded %d entries, want %d", len(out), len(in))
	}
	for k, want := range in {
		if got := out[k]; got != want {
			t.Errorf("key %+v: got %d, want %d", k, got, want)
		}
	}
}

func TestStore_OwnerIsolation_LoadOnlyOwnEntries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Two stores sharing the same bucket but distinct owners.
	ingestStore := natssequencer.NewStore(natsURL(t), "ingest")
	if err := ingestStore.Start(ctx); err != nil {
		t.Fatalf("ingest Start: %v", err)
	}
	defer ingestStore.Close()

	deriveStore := natssequencer.NewStore(natsURL(t), "derive")
	if err := deriveStore.Start(ctx); err != nil {
		t.Fatalf("derive Start: %v", err)
	}
	defer deriveStore.Close()

	ingestKey := sequencer.StreamKey{Venue: "binance", Instrument: "btcusdt", EventType: "observation.trade"}
	deriveKey := sequencer.StreamKey{Venue: "binance", Instrument: "btcusdt", EventType: "evidence.candle"}

	if err := ingestStore.SaveSnapshot(ctx, map[sequencer.StreamKey]int64{ingestKey: 100}); err != nil {
		t.Fatalf("ingest Save: %v", err)
	}
	if err := deriveStore.SaveSnapshot(ctx, map[sequencer.StreamKey]int64{deriveKey: 200}); err != nil {
		t.Fatalf("derive Save: %v", err)
	}

	ingestSnap, err := ingestStore.LoadSnapshot(ctx)
	if err != nil {
		t.Fatalf("ingest Load: %v", err)
	}
	if _, found := ingestSnap[deriveKey]; found {
		t.Error("ingest store loaded a derive-owned key")
	}
	if got := ingestSnap[ingestKey]; got != 100 {
		t.Errorf("ingest own key = %d, want 100", got)
	}

	deriveSnap, err := deriveStore.LoadSnapshot(ctx)
	if err != nil {
		t.Fatalf("derive Load: %v", err)
	}
	if _, found := deriveSnap[ingestKey]; found {
		t.Error("derive store loaded an ingest-owned key")
	}
	if got := deriveSnap[deriveKey]; got != 200 {
		t.Errorf("derive own key = %d, want 200", got)
	}
}

func TestStore_LoadEmptyBucket_ReturnsEmptyMap(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := natssequencer.NewStore(natsURL(t), "unique-test-owner-empty")
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Close()

	out, err := s.LoadSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadSnapshot on empty bucket: %v", err)
	}
	if out == nil {
		t.Fatal("LoadSnapshot returned nil; want empty map")
	}
	if len(out) != 0 {
		t.Errorf("expected no entries for fresh owner; got %d", len(out))
	}
}

func TestStore_RestoreThenResumeFromSequencer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := natssequencer.NewStore(natsURL(t), "ingest")
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Close()

	k := sequencer.StreamKey{Venue: "binance", Instrument: "btcusdt", EventType: "observation.trade"}

	// Simulated prior run: 1000 sequences issued, snapshot persisted.
	prior := sequencer.New()
	for i := 0; i < 1000; i++ {
		prior.Next(k)
	}
	if err := s.SaveSnapshot(ctx, prior.Snapshot()); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	// Simulated restart: load snapshot, restore, resume.
	snap, err := s.LoadSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	resumed := sequencer.New()
	resumed.Restore(snap)

	// Next call must be strictly greater than the prior high-water mark.
	prevHigh := snap[k]
	next := resumed.Next(k)
	if next != prevHigh+1 {
		t.Fatalf("resumed Next = %d, want %d (prior+1)", next, prevHigh+1)
	}
}
