package execution

import (
	"time"
)

// binanceSpotMainnetBaseURL is the Binance Spot mainnet REST endpoint.
// S433: Mainnet uses api.binance.com (vs testnet.binance.vision for testnet).
// Authentication scheme (HMAC-SHA256) and response format are identical.
const binanceSpotMainnetBaseURL = "https://api.binance.com"

// BinanceSpotMainnetAdapter implements ports.VenuePort for Binance Spot mainnet.
//
// S433: This adapter is structurally identical to BinanceSpotTestnetAdapter.
// The only difference is the base URL. All request construction, signing,
// response parsing, error classification, and fee normalization logic are
// inherited by embedding the testnet adapter and overriding the base URL.
//
// The adapter is always deployed behind DryRunSubmitter (dry_run=true is
// enforced for mainnet by config validation). Real order submission requires
// a future authorization ceremony that removes the dry_run enforcement.
type BinanceSpotMainnetAdapter = BinanceSpotTestnetAdapter

// NewBinanceSpotMainnetAdapter creates a venue adapter targeting Binance Spot mainnet.
// creds must contain API_KEY and API_SECRET.
// submitTimeout controls the HTTP client timeout per request.
func NewBinanceSpotMainnetAdapter(creds *CredentialSet, submitTimeout time.Duration) *BinanceSpotMainnetAdapter {
	adapter := NewBinanceSpotTestnetAdapter(creds, submitTimeout)
	adapter.baseURL = binanceSpotMainnetBaseURL
	return adapter
}
