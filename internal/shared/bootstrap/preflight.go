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

// MainnetCredentialCheck returns a preflight check that verifies all required
// mainnet credentials are resolvable before the binary proceeds to adapter bootstrap.
// S434: This catches missing or misconfigured secret manager entries at startup,
// producing an actionable error instead of a late failure in buildVenueAdapterByType.
//
// The check is a no-op when no mainnet adapter is configured, ensuring zero
// overhead for testnet and paper_simulator deployments.
func MainnetCredentialCheck(config settings.AppConfig, resolve func(venueType, key string) string) PreflightCheck {
	return PreflightCheck{
		Name: "mainnet-credentials",
		Check: func() error {
			segs := config.Venue.EnabledSegments()
			for _, seg := range segs {
				adapter := config.Venue.AdapterForSegment(seg)
				if !adapter.IsMainnet() {
					continue
				}
				for _, key := range []string{"API_KEY", "API_SECRET"} {
					val := resolve(string(adapter), key)
					if val == "" {
						return fmt.Errorf(
							"mainnet credential missing: segment=%s adapter=%s key=%s — set via secret manager or environment",
							seg, adapter, key,
						)
					}
				}
			}
			return nil
		},
	}
}

// CredentialPathCheck returns a preflight check that verifies the file
// credential provider's base path exists and is a directory.
// S439: This check is a no-op when the credential provider is not "file".
// When the provider is "file", the base path must exist before any
// credential resolution attempt — a missing directory means secrets
// were not mounted, and the binary must fail fast.
func CredentialPathCheck(config settings.AppConfig) PreflightCheck {
	return PreflightCheck{
		Name: "credential-path",
		Check: func() error {
			if config.Venue.CredentialProviderName() != "file" {
				return nil
			}
			path := config.Venue.CredentialPath
			info, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf(
						"credential_path %q does not exist — mount secrets directory before starting (provider: file)",
						path,
					)
				}
				return fmt.Errorf("credential_path %q is not accessible: %w", path, err)
			}
			if !info.IsDir() {
				return fmt.Errorf("credential_path %q is not a directory", path)
			}
			return nil
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
