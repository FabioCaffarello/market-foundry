package natsevidence

import (
	"internal/domain/instrument"

	"context"
	"log/slog"

	"internal/application/ports"
	"internal/shared/problem"
)

// CandleKVPriceSource implements ports.PriceSource by reading the Close price
// from the CANDLE_LATEST KV bucket. This closes the G1 gap identified in S384:
// DryRunSubmitter and PaperVenueAdapter receive realistic fill prices from the
// live candle projection instead of defaulting to "0".
//
// Best-effort semantics:
//   - Returns the last known close price as a decimal string.
//   - Returns ("0", nil) when no candle data exists (cold start, new symbol).
//   - Returns ("0", problem) on infrastructure errors — callers must not fail.
//   - Safe for concurrent use (delegates to CandleKVStore which is safe).
type CandleKVPriceSource struct {
	store  *CandleKVStore
	logger *slog.Logger
}

var _ ports.PriceSource = (*CandleKVPriceSource)(nil)

// NewCandleKVPriceSource creates a PriceSource backed by a CandleKVStore.
// The store must be started before calling LastPrice.
func NewCandleKVPriceSource(store *CandleKVStore, logger *slog.Logger) *CandleKVPriceSource {
	return &CandleKVPriceSource{store: store, logger: logger}
}

// LastPrice returns the most recent Close price for the given source/symbol/timeframe
// from the CANDLE_LATEST KV bucket. Falls back to "0" when unavailable.
func (p *CandleKVPriceSource) LastPrice(ctx context.Context, source string, inst instrument.CanonicalInstrument, timeframe int) (string, *problem.Problem) {
	candle, prob := p.store.Get(ctx, source, inst, timeframe)
	if prob != nil {
		if p.logger != nil {
			p.logger.Warn("price source: candle KV lookup failed",
				"source", source,
				"instrument", inst.Symbol(),
				"timeframe", timeframe,
				"error", prob.Message,
			)
		}
		return "0", prob
	}

	if candle == nil {
		return "0", nil
	}

	if candle.Close == "" {
		return "0", nil
	}

	return candle.Close, nil
}
