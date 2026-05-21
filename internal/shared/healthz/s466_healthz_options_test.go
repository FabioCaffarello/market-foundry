package healthz_test

import (
	"testing"
	"time"

	"internal/shared/healthz"
)

func TestS466_DefaultConstants(t *testing.T) {
	if healthz.DefaultIdleThreshold != 2*time.Minute {
		t.Fatalf("DefaultIdleThreshold = %v, want 2m", healthz.DefaultIdleThreshold)
	}
	if healthz.DefaultHeartbeatInterval != 30*time.Second {
		t.Fatalf("DefaultHeartbeatInterval = %v, want 30s", healthz.DefaultHeartbeatInterval)
	}
	if healthz.DefaultStartingThreshold != 30*time.Second {
		t.Fatalf("DefaultStartingThreshold = %v, want 30s", healthz.DefaultStartingThreshold)
	}
}

func TestS466_WithStartingThreshold_Applied(t *testing.T) {
	// Create a server with a very short starting threshold.
	s := healthz.NewHealthServer(":0", nil, nil,
		healthz.WithStartingThreshold(1*time.Millisecond),
	)
	// Server should exist without panic; configuration is accepted.
	if s == nil {
		t.Fatal("NewHealthServer returned nil")
	}
}

func TestS466_WithHeartbeatInterval_Applied(t *testing.T) {
	s := healthz.NewHealthServer(":0", nil, nil,
		healthz.WithHeartbeatInterval(10*time.Second),
	)
	if s == nil {
		t.Fatal("NewHealthServer returned nil")
	}
}
