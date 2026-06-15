package delivery

import (
	"log/slog"
	"testing"

	"github.com/anthdm/hollywood/actor"
)

// TestHubMaxSessionsCap pins the subsystem-level bound (ADR-0028 I4):
// the hub admits up to MaxSessions concurrent sessions, rejects beyond
// (returning nil), frees a slot on Close, and treats Close idempotently
// (a double Close must not over-release the counter).
func TestHubMaxSessionsCap(t *testing.T) {
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	router := engine.Spawn(NewRouterActor(), "delivery-router-cap")
	hub := NewHub(engine, router, Config{QueueSize: 8, MaxSessions: 2}, slog.Default())

	h1 := hub.Admit(newRecordingConn())
	h2 := hub.Admit(newRecordingConn())
	if h1 == nil || h2 == nil {
		t.Fatal("first two admits (cap=2) should succeed")
	}
	if h3 := hub.Admit(newRecordingConn()); h3 != nil {
		t.Fatal("third admit should be rejected at cap=2")
	}

	// Freeing a slot lets a new session in.
	h1.Close()
	h4 := hub.Admit(newRecordingConn())
	if h4 == nil {
		t.Fatal("admit after a Close should succeed (slot freed)")
	}

	// Close is idempotent: a second Close must NOT free an extra slot.
	h1.Close()
	if h5 := hub.Admit(newRecordingConn()); h5 != nil {
		t.Fatal("double-Close must not free an extra slot (cap still 2)")
	}

	h2.Close()
	h4.Close()
}

// TestHubUnlimitedWhenMaxSessionsZero pins that MaxSessions == 0 disables
// the cap.
func TestHubUnlimitedWhenMaxSessionsZero(t *testing.T) {
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	router := engine.Spawn(NewRouterActor(), "delivery-router-unl")
	hub := NewHub(engine, router, Config{QueueSize: 8, MaxSessions: 0}, slog.Default())
	for i := 0; i < 5; i++ {
		if h := hub.Admit(newRecordingConn()); h == nil {
			t.Fatalf("admit %d should succeed (MaxSessions=0 = unlimited)", i)
		}
	}
}
