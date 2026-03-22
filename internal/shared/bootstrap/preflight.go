package bootstrap

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"internal/shared/settings"
)

// PreflightCheck represents a single startup precondition to validate before
// opening connections or performing I/O.
type PreflightCheck struct {
	Name  string
	Check func() error
}

// RunPreflight executes all checks sequentially. On the first failure it logs the
// error and exits the process. This is intentionally fail-fast: a missing
// precondition means the binary cannot operate, so there is no value in
// continuing to the next check or attempting partial startup.
func RunPreflight(serviceName string, logger *slog.Logger, checks []PreflightCheck) {
	for _, c := range checks {
		if err := c.Check(); err != nil {
			logger.Error(fmt.Sprintf("%s startup blocked: preflight check %q failed", serviceName, c.Name),
				"check", c.Name,
				"error", err,
			)
			os.Exit(1)
		}
	}
	logger.Info(fmt.Sprintf("%s preflight passed", serviceName), "checks", len(checks))
}

// NATSEnabledCheck returns a preflight check that verifies NATS is enabled in config.
// Use this for binaries that hard-require NATS (all except migrate).
func NATSEnabledCheck(config settings.AppConfig) PreflightCheck {
	return PreflightCheck{
		Name: "nats-enabled",
		Check: func() error {
			if !config.NATS.Enabled {
				return fmt.Errorf("nats.enabled must be true — this binary requires NATS connectivity")
			}
			return nil
		},
	}
}

// NATSURLFormatCheck returns a preflight check that validates the NATS URL has
// a parseable scheme and host. This catches obviously malformed URLs before any
// connection attempt, producing an actionable error message instead of an opaque
// dial failure.
func NATSURLFormatCheck(config settings.AppConfig) PreflightCheck {
	return PreflightCheck{
		Name: "nats-url-format",
		Check: func() error {
			if !config.NATS.Enabled {
				return nil // skip if NATS is not enabled
			}
			return ValidateNATSURL(config.NATS.URL)
		},
	}
}

// ValidateNATSURL checks that a NATS URL has a valid scheme and non-empty host.
// Accepted schemes: nats, nats+tls, tls, wss (standard NATS client schemes).
func ValidateNATSURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("nats.url is empty")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("nats.url is not a valid URL: %w", err)
	}

	switch u.Scheme {
	case "nats", "tls", "wss":
		// accepted schemes
	default:
		return fmt.Errorf("nats.url has unexpected scheme %q; expected nats://, tls://, or wss://", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("nats.url has no host component: %s", raw)
	}

	return nil
}
