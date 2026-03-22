package bootstrap

import (
	"testing"

	"internal/shared/settings"
)

func TestNATSEnabledCheck(t *testing.T) {
	t.Run("passes when enabled", func(t *testing.T) {
		cfg := settings.AppConfig{NATS: settings.NATSConfig{Enabled: true, URL: "nats://localhost:4222"}}
		check := NATSEnabledCheck(cfg)
		if err := check.Check(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("fails when disabled", func(t *testing.T) {
		cfg := settings.AppConfig{NATS: settings.NATSConfig{Enabled: false}}
		check := NATSEnabledCheck(cfg)
		if err := check.Check(); err == nil {
			t.Fatal("expected error for disabled NATS")
		}
	})
}

func TestNATSURLFormatCheck(t *testing.T) {
	t.Run("skips when NATS disabled", func(t *testing.T) {
		cfg := settings.AppConfig{NATS: settings.NATSConfig{Enabled: false}}
		check := NATSURLFormatCheck(cfg)
		if err := check.Check(); err != nil {
			t.Fatalf("expected no error when NATS disabled, got %v", err)
		}
	})

	t.Run("passes valid nats URL", func(t *testing.T) {
		cfg := settings.AppConfig{NATS: settings.NATSConfig{Enabled: true, URL: "nats://localhost:4222"}}
		check := NATSURLFormatCheck(cfg)
		if err := check.Check(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("fails on empty URL", func(t *testing.T) {
		cfg := settings.AppConfig{NATS: settings.NATSConfig{Enabled: true, URL: ""}}
		check := NATSURLFormatCheck(cfg)
		if err := check.Check(); err == nil {
			t.Fatal("expected error for empty URL")
		}
	})

	t.Run("fails on bad scheme", func(t *testing.T) {
		cfg := settings.AppConfig{NATS: settings.NATSConfig{Enabled: true, URL: "http://localhost:4222"}}
		check := NATSURLFormatCheck(cfg)
		if err := check.Check(); err == nil {
			t.Fatal("expected error for http scheme")
		}
	})
}

func TestValidateNATSURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid nats", "nats://localhost:4222", false},
		{"valid tls", "tls://nats.example.com:4222", false},
		{"valid wss", "wss://nats.example.com:443", false},
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"http scheme", "http://localhost:4222", true},
		{"no host", "nats://", true},
		{"no scheme", "localhost:4222", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNATSURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNATSURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
