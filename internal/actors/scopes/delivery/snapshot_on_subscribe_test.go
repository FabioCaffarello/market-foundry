package delivery

import (
	"log/slog"
	"testing"
	"time"

	"github.com/anthdm/hollywood/actor"
)

// fakeSnapshotProvider returns a fixed frame for any subject (or nothing
// when frame is nil), exercising the SessionActor's snapshot-on-subscribe
// wiring without a real KV. The provider's own parsing/KV logic is tested
// in the natsdelivery package.
type fakeSnapshotProvider struct{ frame []byte }

func (f *fakeSnapshotProvider) Snapshot(string) ([]byte, bool) {
	if f.frame == nil {
		return nil, false
	}
	return f.frame, true
}

func TestSnapshotOnSubscribe(t *testing.T) {
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	router := engine.Spawn(NewRouterActor(), "delivery-router-snap")
	snap := []byte(`{"subject":"insights.events.volumeprofile.sampled.s.btc_usdt_perpetual.60","event":{"volume_profile":{}}}`)
	hub := NewHub(engine, router, Config{QueueSize: 8, SnapshotProvider: &fakeSnapshotProvider{frame: snap}}, slog.Default())

	conn := newRecordingConn()
	h := hub.Admit(conn)
	defer h.Close()
	h.Subscribe("insights.events.volumeprofile.sampled.s.btc_usdt_perpetual.60")

	select {
	case got := <-conn.frames:
		if string(got) != string(snap) {
			t.Fatalf("snapshot frame mismatch:\n got %s\nwant %s", got, snap)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("expected a snapshot frame on subscribe")
	}
}

func TestNoSnapshotWhenProviderNil(t *testing.T) {
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	router := engine.Spawn(NewRouterActor(), "delivery-router-nosnap")
	hub := NewHub(engine, router, Config{QueueSize: 8}, slog.Default()) // no provider

	conn := newRecordingConn()
	h := hub.Admit(conn)
	defer h.Close()
	h.Subscribe("insights.events.volumeprofile.sampled.s.btc_usdt_perpetual.60")

	select {
	case got := <-conn.frames:
		t.Fatalf("unexpected frame with nil provider: %s", got)
	case <-time.After(200 * time.Millisecond):
		// expected: no snapshot offered
	}
}
