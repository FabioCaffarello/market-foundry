package replay_test

import (
	"fmt"
	"time"

	envv1 "internal/shared/contracts/envelope/v1"
	"internal/shared/random"
)

// goldenReplayCyclePath is the canonical relative path of the
// replay-cycle scope's golden fixture, anchored at this package's
// testdata directory.
const goldenReplayCyclePath = "testdata/golden/replay-cycle/synthetic-100.jsonl"

// generateSyntheticReplayCycleEvents returns the deterministic
// 100-event sequence used by the replay-cycle golden. The
// distribution mirrors the agreement reached in PAUSE-AND-REPORT #5
// before commit 9 landed:
//
//   - 3 event types (observation.trade / observation.book.snapshot /
//     observation.funding), 2 venues (binance / binancef), 5
//     instruments (btcusdt / ethusdt / solusdt / bnbusdt / xrpusdt).
//   - Approximately 60% of events have a non-nil TsExchange; the
//     other ~40% have TsExchange == nil (covering the explicit
//     "ts_exchange":null encoding path).
//   - Approximately 50% of events have a non-empty Payload; the
//     other ~50% have Payload that the recorder normalizes to
//     []byte{} (covering the canonical "payload":"" empty
//     encoding path).
//   - The Seq value of event i is exactly i, so event 0 has Seq=0
//     (covering the first-of-a-stream legitimate zero) and events
//     1..99 have Seq>0. Per ADR-0020 a real Sequencer would
//     allocate per-stream-key counters; the fixture intentionally
//     decouples from Sequencer semantics because the golden tests
//     the replay layer's record/play byte-stability, not Sequencer
//     ordering.
//
// Determinism comes from random.SeededSource(42); calling this
// function from any process produces the same byte-for-byte
// sequence. Changing the seed or the distribution constants is a
// fixture-format-breaking change — `make golden-regen
// SCOPE=replay-cycle` must follow.
func generateSyntheticReplayCycleEvents() []envv1.CanonicalEvent {
	const baseNanos int64 = 1700000000000000000

	rng := random.NewSeededSource(42)

	types := []string{
		"observation.trade",
		"observation.book.snapshot",
		"observation.funding",
	}
	venues := []string{"binance", "binancef"}
	instruments := []string{"btcusdt", "ethusdt", "solusdt", "bnbusdt", "xrpusdt"}

	events := make([]envv1.CanonicalEvent, 0, 100)
	for i := 0; i < 100; i++ {
		typIdx := rng.Int63() % int64(len(types))
		venueIdx := rng.Int63() % int64(len(venues))
		instrIdx := rng.Int63() % int64(len(instruments))

		var tsExchange *time.Time
		if rng.Float64() < 0.6 {
			ts := time.Unix(0, baseNanos+int64(i)*int64(time.Millisecond)).UTC()
			tsExchange = &ts
		}

		var payload []byte
		if rng.Float64() < 0.5 {
			payload = []byte(fmt.Sprintf("p%d", i))
		}
		// else: payload remains nil; recorder normalizes to []byte{}
		// which serializes as "payload":"" in the fixture.

		events = append(events, envv1.CanonicalEvent{
			Type:           types[typIdx],
			Version:        1,
			Venue:          venues[venueIdx],
			Instrument:     instruments[instrIdx],
			TsExchange:     tsExchange,
			TsIngest:       time.Unix(0, baseNanos+int64(i)*int64(time.Microsecond)).UTC(),
			Seq:            int64(i),
			IdempotencyKey: fmt.Sprintf("k-%04d", i),
			Payload:        payload,
		})
	}
	return events
}
