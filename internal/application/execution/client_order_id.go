package execution

import (
	"crypto/sha256"
	"encoding/hex"

	domainexec "internal/domain/execution"
)

// ClientOrderID derives a deterministic client order ID from an ExecutionIntent.
// The derivation hashes the intent's DeduplicationKey() with SHA-256 and returns the
// first 32 hex characters. This guarantees:
//   - Same intent → same ID (deterministic)
//   - Different intents → different IDs (collision-resistant)
//   - Conforms to Binance newClientOrderId format: alphanumeric, max 36 chars
//
// The derivation uses no random or time-varying inputs beyond what is already
// encoded in the deduplication key (type, source, symbol, timeframe, unix timestamp).
func ClientOrderID(intent domainexec.ExecutionIntent) string {
	key := intent.DeduplicationKey()
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:16]) // 32 hex chars, fits Binance 36-char limit
}
