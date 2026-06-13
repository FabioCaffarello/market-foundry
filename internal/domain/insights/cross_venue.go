package insights

import (
	"math/big"
	"strings"
	"time"

	"internal/domain/instrument"
	"internal/shared/problem"
)

// VenueRow is one venue's trade summary within a cross-venue window.
// Venue is the source identifier (e.g. "binancef", "bybitf"); decimals
// are strings (exact, same convention as evidence/volume profile).
type VenueRow struct {
	Venue      string `json:"venue"`
	TradeCount int64  `json:"trade_count"`
	Notional   string `json:"notional"`   // decimal — Σ price×qty in window
	LastPrice  string `json:"last_price"` // decimal — last trade price in window
	HighPrice  string `json:"high_price"` // decimal — max trade price
	LowPrice   string `json:"low_price"`  // decimal — min trade price
}

// CrossVenueSnapshot fuses trade flow for ONE canonical instrument
// across venues within a timeframe window (decision-support, ADR-0027).
// The venue is the fused dimension; the canonical instrument
// (venue-agnostic per ADR-0021) is the join key. Trades-only (I4).
type CrossVenueSnapshot struct {
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"` // window duration in seconds
	Venues     []VenueRow                     `json:"venues"`    // ascending by Venue

	// Consolidated metrics across the venues' last prices.
	SpreadAbs     string `json:"spread_abs"`     // max(last) - min(last)
	SpreadBps     string `json:"spread_bps"`     // spread / mid * 10000
	MidPrice      string `json:"mid_price"`      // (max(last) + min(last)) / 2
	DominantVenue string `json:"dominant_venue"` // venue with the most notional

	TradeCount int64     `json:"trade_count"` // total across venues
	OpenTime   time.Time `json:"open_time"`
	CloseTime  time.Time `json:"close_time"`
	Final      bool      `json:"final"`
}

// VenueSymbol returns the lowercase venue-native symbol form derived
// from the canonical instrument. TRANSITORY ADAPTER (shared shape).
func (s CrossVenueSnapshot) VenueSymbol() string {
	return strings.ToLower(string(s.Instrument.Base) + string(s.Instrument.Quote))
}

// Validate enforces the snapshot invariants. A window with zero venues
// is invalid (the sampler only emits when ≥1 venue traded); each venue
// row must be fully populated.
func (s CrossVenueSnapshot) Validate() *problem.Problem {
	if s.Instrument.IsZero() {
		return problem.New(problem.InvalidArgument, "instrument is required")
	}
	if prob := s.Instrument.Validate(); prob != nil {
		return prob
	}
	if s.Timeframe <= 0 {
		return problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if len(s.Venues) == 0 {
		return problem.New(problem.InvalidArgument, "cross-venue snapshot needs at least one venue")
	}
	for _, v := range s.Venues {
		if v.Venue == "" || v.Notional == "" || v.LastPrice == "" || v.HighPrice == "" || v.LowPrice == "" {
			return problem.Validation(
				problem.InvalidArgument,
				"cross-venue venue row is incomplete",
				problem.ValidationIssue{
					Field:   "venues",
					Message: "each venue row needs venue, notional, last/high/low price",
					Value:   v.Venue,
				},
			)
		}
	}
	if s.OpenTime.IsZero() {
		return problem.New(problem.InvalidArgument, "open_time is required")
	}
	if s.CloseTime.IsZero() {
		return problem.New(problem.InvalidArgument, "close_time is required")
	}
	if !s.CloseTime.After(s.OpenTime) {
		return problem.New(problem.InvalidArgument, "close_time must be after open_time")
	}
	return nil
}

// ConsolidatedSpread computes the absolute spread, spread in basis
// points, and mid price across the venues' last prices. With a single
// venue the spread is zero and the mid is that venue's last price.
// Returns ("0","0","") when no venue has a parseable last price.
func ConsolidatedSpread(venues []VenueRow) (spreadAbs, spreadBps, mid string) {
	var min, max *big.Rat
	for _, v := range venues {
		r, ok := new(big.Rat).SetString(v.LastPrice)
		if !ok {
			continue
		}
		if min == nil || r.Cmp(min) < 0 {
			min = new(big.Rat).Set(r)
		}
		if max == nil || r.Cmp(max) > 0 {
			max = new(big.Rat).Set(r)
		}
	}
	if min == nil || max == nil {
		return "0", "0", ""
	}

	spread := new(big.Rat).Sub(max, min)
	midR := new(big.Rat).Add(max, min)
	midR.Quo(midR, big.NewRat(2, 1))

	spreadAbs = spread.FloatString(8)
	mid = midR.FloatString(8)

	if midR.Sign() == 0 {
		spreadBps = "0"
		return
	}
	bps := new(big.Rat).Quo(spread, midR)
	bps.Mul(bps, big.NewRat(10000, 1))
	spreadBps = bps.FloatString(4)
	return
}

// DominantVenue returns the venue with the greatest notional. Ties
// resolve to the first encountered (the caller passes venues ascending
// by name, so ties resolve to the alphabetically-first venue). Returns
// "" for an empty set.
func DominantVenue(venues []VenueRow) string {
	best := ""
	var bestNotional *big.Rat
	for _, v := range venues {
		r, ok := new(big.Rat).SetString(v.Notional)
		if !ok {
			continue
		}
		if bestNotional == nil || r.Cmp(bestNotional) > 0 {
			bestNotional = new(big.Rat).Set(r)
			best = v.Venue
		}
	}
	return best
}
