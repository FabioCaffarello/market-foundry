package delivery

import (
	"log/slog"
	"testing"
	"time"

	"github.com/anthdm/hollywood/actor"
)

// TestOfferDropNewest pins the bounded-buffer backpressure (ADR-0028
// I4): once the outbound buffer is full, the newest frames are dropped
// and counted; nothing blocks. Deterministic — no engine, no goroutine.
func TestOfferDropNewest(t *testing.T) {
	a := &SessionActor{out: make(chan []byte, 2), logger: slog.Default()}

	a.offer([]byte("a"))
	a.offer([]byte("b"))
	// buffer (cap 2) is now full; the next offer must drop, not block.
	a.offer([]byte("c"))

	if got := len(a.out); got != 2 {
		t.Fatalf("buffered = %d, want 2", got)
	}
	if a.dropped != 1 {
		t.Fatalf("dropped = %d, want 1", a.dropped)
	}
}

// recordingConn is a fake WSConn that records frames on a channel.
type recordingConn struct {
	frames chan []byte
}

func newRecordingConn() *recordingConn {
	return &recordingConn{frames: make(chan []byte, 16)}
}

func (c *recordingConn) Send(frame []byte) error {
	c.frames <- frame
	return nil
}

func (c *recordingConn) Close() error { return nil }

// TestFanoutMatchesSubscriptionOnly proves the end-to-end actor wiring
// without NATS: a subscribed client receives only events whose subject
// matches its subscription. Ordering is deterministic — the
// non-matching event is broadcast before the matching one, so if the
// filter were broken the client would receive the non-matching frame
// first. Reading exactly one frame and asserting it is the matching one
// therefore catches a filter regression without a flaky "wait for
// nothing".
func TestFanoutMatchesSubscriptionOnly(t *testing.T) {
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	router := engine.Spawn(NewRouterActor(), "delivery-router")
	hub := NewHub(engine, router, slog.Default())

	conn := newRecordingConn()
	h := hub.Admit(conn)
	defer h.Close()

	h.Subscribe("insights.events.volumeprofile.sampled.>")

	const tpoSubject = "insights.events.tpo.sampled.binances.btc_usdt_spot.60"
	const vpSubject = "insights.events.volumeprofile.sampled.binances.btc_usdt_spot.60"

	// Non-matching first, then matching. FIFO mailbox ordering means a
	// broken filter would surface the TPO frame before the VP frame.
	engine.Send(router, eventReceivedMessage{Subject: tpoSubject, Payload: []byte("TPO")})
	engine.Send(router, eventReceivedMessage{Subject: vpSubject, Payload: []byte("VP")})

	select {
	case got := <-conn.frames:
		if string(got) != "VP" {
			t.Fatalf("first delivered frame = %q, want VP (TPO should have been filtered)", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for the matching frame")
	}

	// No further frame should arrive (TPO was filtered).
	select {
	case got := <-conn.frames:
		t.Fatalf("unexpected extra frame %q", got)
	case <-time.After(150 * time.Millisecond):
	}
}
