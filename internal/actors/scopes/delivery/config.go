package delivery

import (
	"log/slog"
	"os"
	"strconv"

	"internal/application/ports"
	deliverydomain "internal/domain/delivery"
)

// DefaultMaxSessions bounds the total number of concurrent WebSocket
// delivery sessions the hub admits (ADR-0028 I4 — a subsystem-level
// bound complementing the per-session bounded buffer). A generous
// default that loopback usage never reaches in practice; 0 = unlimited.
const DefaultMaxSessions = 1024

// Config tunes the delivery subsystem (ADR-0028 I4). Use DefaultConfig
// and override; a non-positive QueueSize falls back to the default.
type Config struct {
	QueueSize   int
	Policy      deliverydomain.BackpressurePolicy
	MaxSessions int // 0 = unlimited
	// SnapshotProvider, when non-nil, supplies a KV-latest snapshot sent
	// to a client on subscribe (before live deltas, H-11.f). nil = off.
	SnapshotProvider ports.SnapshotProvider
}

// DefaultConfig is the bounded, DropNewest baseline.
func DefaultConfig() Config {
	return Config{QueueSize: DefaultOutboundQueue, Policy: deliverydomain.DropNewest, MaxSessions: DefaultMaxSessions}
}

// ConfigFromEnv builds a delivery Config from the baseline plus optional
// operator overrides (kept off the shared settings schema — the gateway
// owns this surface):
//
//	MARKETFOUNDRY_DELIVERY_QUEUE_SIZE   — positive integer
//	MARKETFOUNDRY_DELIVERY_BACKPRESSURE — drop_newest | drop_oldest
//
// Invalid values are logged and ignored (the default stands).
func ConfigFromEnv(logger *slog.Logger) Config {
	cfg := DefaultConfig()
	if v := os.Getenv("MARKETFOUNDRY_DELIVERY_QUEUE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.QueueSize = n
		} else if logger != nil {
			logger.Warn("invalid MARKETFOUNDRY_DELIVERY_QUEUE_SIZE; using default", "value", v)
		}
	}
	if v := os.Getenv("MARKETFOUNDRY_DELIVERY_BACKPRESSURE"); v != "" {
		if p, prob := deliverydomain.ParseBackpressurePolicy(v); prob == nil {
			cfg.Policy = p
		} else if logger != nil {
			logger.Warn("invalid MARKETFOUNDRY_DELIVERY_BACKPRESSURE; using default", "value", v)
		}
	}
	if v := os.Getenv("MARKETFOUNDRY_DELIVERY_MAX_SESSIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 { // 0 = unlimited
			cfg.MaxSessions = n
		} else if logger != nil {
			logger.Warn("invalid MARKETFOUNDRY_DELIVERY_MAX_SESSIONS; using default", "value", v)
		}
	}
	return cfg
}
