//go:build integration

package natsexecution_test

// key_cutover_canary_test.go — H-6.e.2 canary: canonical KV
// partition-key shape end-to-end against a real NATS server.
//
// Proves the load-bearing claims of the key cutover (wave prompt
// Decisão #3):
//
//   - KEY-1 (literal shape): a Put through the real store lands under
//     the literal canonical key "{source}.{subject_token}.{timeframe}"
//     — asserted via GetByKey with the hand-built literal, locking
//     the wire-visible shape (not just Put/Get symmetry).
//   - KEY-2 (no legacy writes): the legacy venue-native key shape
//     ("{source}.{venuesymbol}.{timeframe}") is NOT written for new
//     entries — pre-cutover keys are inert orphans, never refreshed.
//
// Requires NATS at localhost:4222 (or NATS_URL); skips otherwise.

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"internal/adapters/nats/natsexecution"
	"internal/domain/instrument"
)

func keyCanaryNATSURL(t *testing.T) string {
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

func TestKeyCutover_CanonicalShapeAndNoLegacyWrites(t *testing.T) {
	url := keyCanaryNATSURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	store := natsexecution.NewKVStore(url, testBucket(t))
	if err := store.Start(); err != nil {
		t.Fatalf("kv store start: %v", err)
	}
	defer func() { _ = store.Close() }()

	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("instrument.New: %v", prob)
	}

	intent := testIntent(t, time.Now().UTC())
	intent.Instrument = inst
	if _, prob := store.Put(ctx, intent); prob != nil {
		t.Fatalf("put: %v", prob)
	}

	// KEY-1: the literal canonical key shape is readable.
	canonicalKey := intent.Source + "." + inst.SubjectToken() + ".60"
	got, prob := store.GetByKey(ctx, canonicalKey)
	if prob != nil {
		t.Fatalf("GetByKey(%q): %v", canonicalKey, prob)
	}
	if got == nil {
		t.Fatalf("GetByKey(%q): expected entry, got nil — key shape drifted", canonicalKey)
	}

	// KEY-2: the legacy venue-native shape was NOT written.
	legacyKey := intent.Source + ".btcusdt.60"
	gotLegacy, prob := store.GetByKey(ctx, legacyKey)
	if prob != nil {
		t.Fatalf("GetByKey(%q): %v", legacyKey, prob)
	}
	if gotLegacy != nil {
		t.Fatalf("legacy key %q was written — cutover incomplete", legacyKey)
	}

	if tok := inst.SubjectToken(); tok != "btc_usdt_perpetual" {
		t.Errorf("token derivation drifted: %q", tok)
	}
}
