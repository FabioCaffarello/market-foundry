package ports

import (
	"context"

	"internal/shared/problem"
)

// PriceSource provides best-effort last-observed price for a symbol.
// Used by DryRunSubmitter and PaperVenueAdapter to produce realistic fill prices
// instead of hardcoded "0". Implementations read from CANDLE_LATEST KV bucket.
//
// Contract:
//   - Returns the last known close price as a decimal string (e.g., "50123.45").
//   - Returns ("0", nil) when no price data is available (cold start, new symbol).
//   - Returns ("0", problem) on infrastructure errors — callers must not fail on error.
//   - Implementations must be safe for concurrent use.
type PriceSource interface {
	LastPrice(ctx context.Context, source, symbol string, timeframe int) (string, *problem.Problem)
}
