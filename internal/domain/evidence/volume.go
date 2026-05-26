package evidence

import (
	"strings"
	"time"

	"internal/domain/instrument"
	"internal/shared/problem"
)

// EvidenceVolume is a per-window volume profile for a specific instrument and timeframe.
// It provides notional buy/sell volume, total volume, and volume-weighted average price (VWAP).
// All monetary values are decimal strings to avoid IEEE 754 precision loss.
//
// Per ADR-0021, the canonical instrument identity is carried in the
// Instrument field. Migrated from Symbol string in H-6.b.
type EvidenceVolume struct {
	Source      string                         `json:"source"`
	Instrument  instrument.CanonicalInstrument `json:"instrument"`
	Timeframe   int                            `json:"timeframe"`    // window duration in seconds
	BuyVolume   string                         `json:"buy_volume"`   // decimal — notional buy volume (Σ price×qty where buyer_is_maker)
	SellVolume  string                         `json:"sell_volume"`  // decimal — notional sell volume (Σ price×qty where !buyer_is_maker)
	TotalVolume string                         `json:"total_volume"` // decimal — BuyVolume + SellVolume
	VWAP        string                         `json:"vwap"`         // decimal — volume-weighted average price (TotalVolume / TotalQuantity)
	TradeCount  int64                          `json:"trade_count"`
	OpenTime    time.Time                      `json:"open_time"`
	CloseTime   time.Time                      `json:"close_time"`
	Final       bool                           `json:"final"`
}

// VenueSymbol returns the lowercase venue-native symbol form
// (e.g., "btcusdt") derived from the canonical instrument.
//
// TRANSITORY ADAPTER (H-6.b → sunset H-6.f). See ADR-0021 +
// H-6.a precedent on ObservationTrade.VenueSymbol() for the
// shape and the lossy-for-delivery-futures limitation.
func (v EvidenceVolume) VenueSymbol() string {
	return strings.ToLower(string(v.Instrument.Base) + string(v.Instrument.Quote))
}

func (v EvidenceVolume) Validate() *problem.Problem {
	if v.Source == "" {
		return problem.New(problem.InvalidArgument, "source is required")
	}
	if v.Instrument.IsZero() {
		return problem.New(problem.InvalidArgument, "instrument is required")
	}
	if prob := v.Instrument.Validate(); prob != nil {
		return prob
	}
	if v.Timeframe <= 0 {
		return problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if v.BuyVolume == "" {
		return problem.New(problem.InvalidArgument, "buy_volume is required")
	}
	if v.SellVolume == "" {
		return problem.New(problem.InvalidArgument, "sell_volume is required")
	}
	if v.TotalVolume == "" {
		return problem.New(problem.InvalidArgument, "total_volume is required")
	}
	if v.VWAP == "" {
		return problem.New(problem.InvalidArgument, "vwap is required")
	}
	if v.OpenTime.IsZero() {
		return problem.New(problem.InvalidArgument, "open_time is required")
	}
	if v.CloseTime.IsZero() {
		return problem.New(problem.InvalidArgument, "close_time is required")
	}
	if !v.CloseTime.After(v.OpenTime) {
		return problem.New(problem.InvalidArgument, "close_time must be after open_time")
	}
	return nil
}
