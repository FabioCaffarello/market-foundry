package bootstrap

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"internal/shared/healthz"
	"internal/shared/settings"
)

// NATSReadinessCheck returns a readiness check that verifies TCP connectivity
// to the configured NATS server. If NATS is disabled in config, the check
// returns an error indicating that NATS is not enabled.
func NATSReadinessCheck(config settings.AppConfig) healthz.ReadinessCheck {
	return healthz.ReadinessCheck{
		Name: "nats",
		Check: func(ctx context.Context) error {
			if !config.NATS.Enabled {
				return fmt.Errorf("nats is not enabled")
			}
			return dialNATS(config.NATS.URL)
		},
	}
}

// dialNATS performs a TCP dial to the NATS server to verify connectivity.
func dialNATS(natsURL string) error {
	u, err := url.Parse(natsURL)
	if err != nil {
		return fmt.Errorf("parse nats url: %w", err)
	}
	host := u.Host
	if host == "" {
		host = u.Opaque
	}
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return fmt.Errorf("nats dial: %w", err)
	}
	_ = conn.Close()
	return nil
}
