package evidence

import (
	"strings"
	"time"

	"internal/domain/instrument"
	"internal/shared/problem"
)

// EvidenceTradeBurst is a per-window summary of trade activity for a specific instrument and timeframe.
// Emitted every finalized window. Captures metrics that candles do not: buy/sell volume split and burst detection.
// All monetary values are decimal strings to avoid IEEE 754 precision loss.
//
// Per ADR-0021, the canonical instrument identity is carried in the
// Instrument field. Migrated from Symbol string in H-6.b.
type EvidenceTradeBurst struct {
	Source     string                         `json:"source"`      // Exchange identifier (e.g., "binancef")
	Instrument instrument.CanonicalInstrument `json:"instrument"`  // Canonical instrument identity (ADR-0021)
	Timeframe  int                            `json:"timeframe"`   // Window duration in seconds (e.g., 60, 300)
	TradeCount int64                          `json:"trade_count"` // Total trades in window
	BuyVolume  string                         `json:"buy_volume"`  // Decimal string — volume where buyer is maker
	SellVolume string                         `json:"sell_volume"` // Decimal string — volume where buyer is taker
	OpenTime   time.Time                      `json:"open_time"`   // Window start
	CloseTime  time.Time                      `json:"close_time"`  // Window end
	Burst      bool                           `json:"burst"`       // True if trade_count significantly exceeds baseline
	Final      bool                           `json:"final"`       // True = window closed (immutable)
}

// VenueSymbol returns the lowercase venue-native symbol form
// (e.g., "btcusdt") derived from the canonical instrument.
//
// TRANSITORY ADAPTER (H-6.b): exists to keep KV partition keys,
// log labels, and publisher subject composition compiling while
// the migration to CanonicalInstrument propagates. Slated for
// sunset in H-6.f.
//
// Known limitation: lossy for delivery futures. See ADR-0021 +
// H-6.a precedent on ObservationTrade.VenueSymbol().
func (b EvidenceTradeBurst) VenueSymbol() string {
	return strings.ToLower(string(b.Instrument.Base) + string(b.Instrument.Quote))
}

func (b EvidenceTradeBurst) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if b.Source == "" {
		issues = append(issues, problem.ValidationIssue{Field: "source", Message: "must not be empty"})
	}
	if b.Instrument.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "instrument", Message: "must not be zero"})
	} else if prob := b.Instrument.Validate(); prob != nil {
		return prob
	}
	if b.Timeframe <= 0 {
		issues = append(issues, problem.ValidationIssue{Field: "timeframe", Message: "must be a positive integer"})
	}
	if b.BuyVolume == "" {
		issues = append(issues, problem.ValidationIssue{Field: "buy_volume", Message: "must not be empty"})
	}
	if b.SellVolume == "" {
		issues = append(issues, problem.ValidationIssue{Field: "sell_volume", Message: "must not be empty"})
	}
	if b.OpenTime.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "open_time", Message: "must not be zero"})
	}
	if b.CloseTime.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "close_time", Message: "must not be zero"})
	}
	if !b.OpenTime.IsZero() && !b.CloseTime.IsZero() && !b.CloseTime.After(b.OpenTime) {
		issues = append(issues, problem.ValidationIssue{Field: "close_time", Message: "must be after open_time"})
	}

	if len(issues) == 0 {
		return nil
	}
	return problem.Validation(problem.InvalidArgument, "evidence trade burst is invalid", issues...)
}
