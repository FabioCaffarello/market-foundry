package delivery

import (
	"log/slog"
	"os"
	"strconv"

	deliverydomain "internal/domain/delivery"
)

// Config tunes the delivery subsystem (ADR-0028 I4). Use DefaultConfig
// and override; a non-positive QueueSize falls back to the default.
type Config struct {
	QueueSize int
	Policy    deliverydomain.BackpressurePolicy
}

// DefaultConfig is the bounded, DropNewest baseline.
func DefaultConfig() Config {
	return Config{QueueSize: DefaultOutboundQueue, Policy: deliverydomain.DropNewest}
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
	return cfg
}
