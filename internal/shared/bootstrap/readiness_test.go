package bootstrap

import (
	"context"
	"testing"

	"internal/shared/settings"
)

func TestNATSReadinessCheckDisabled(t *testing.T) {
	config := settings.AppConfig{
		NATS: settings.NATSConfig{Enabled: false},
	}
	check := NATSReadinessCheck(config)
	if check.Name != "nats" {
		t.Fatalf("expected check name 'nats', got %q", check.Name)
	}
	err := check.Check(context.Background())
	if err == nil {
		t.Fatal("expected error when NATS is disabled")
	}
}

func TestDialNATSInvalidURL(t *testing.T) {
	err := dialNATS("://bad")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestDialNATSUnreachable(t *testing.T) {
	err := dialNATS("nats://127.0.0.1:4")
	if err == nil {
		t.Fatal("expected error for unreachable host")
	}
}
