package execution

import (
	"time"
)

// binanceFuturesMainnetBaseURL is the Binance Futures mainnet REST endpoint.
// S433: Mainnet uses fapi.binance.com (vs testnet.binancefuture.com for testnet).
// Authentication scheme (HMAC-SHA256) and response format are identical.
const binanceFuturesMainnetBaseURL = "https://fapi.binance.com"

// BinanceFuturesMainnetAdapter implements ports.VenuePort for Binance Futures mainnet.
//
// S433: This adapter is structurally identical to BinanceFuturesTestnetAdapter.
// The only difference is the base URL. All request construction, signing,
// response parsing, error classification, and fee normalization logic are
// inherited by embedding the testnet adapter and overriding the base URL.
//
// The adapter is always deployed behind DryRunSubmitter (dry_run=true is
// enforced for mainnet by config validation). Real order submission requires
// a future authorization ceremony that removes the dry_run enforcement.
type BinanceFuturesMainnetAdapter = BinanceFuturesTestnetAdapter

// NewBinanceFuturesMainnetAdapter creates a venue adapter targeting Binance Futures mainnet.
// creds must contain API_KEY and API_SECRET.
// submitTimeout controls the HTTP client timeout per request.
func NewBinanceFuturesMainnetAdapter(creds *CredentialSet, submitTimeout time.Duration) *BinanceFuturesMainnetAdapter {
	adapter := NewBinanceFuturesTestnetAdapter(creds, submitTimeout)
	adapter.baseURL = binanceFuturesMainnetBaseURL
	return adapter
}
