// Package insights provides the decision-support analytics domain
// per ADR-0027. Insights describe market structure (volume profile,
// TPO, cross-venue fusion); they are read-only over the pipeline and
// NEVER emit buy/sell directives. The package is in the innermost
// layer (internal/domain/) and pure value-typed: no time, no random,
// no I/O — compatible with the check determinism analyzer
// (ADR-0019 INV-D1).
//
// ADR-0027 boundary: this package MUST NOT import the directive chain
// (internal/domain/{strategy,decision,risk,execution}); the
// `check insights` analyzer enforces that statically.
package insights

import (
	"strings"
	"time"

	"internal/domain/instrument"
	"internal/shared/problem"
)

// PriceBucket is one price level of a volume profile: the notional
// buy/sell volume that traded within the bucket's price range.
// PriceLevel is the canonical lower bound of the bucket (decimal
// string, computed via BucketLevel). All notional values are decimal
// strings to avoid IEEE 754 precision loss (same convention as
// evidence).
type PriceBucket struct {
	PriceLevel string `json:"price_level"` // decimal — bucket lower bound
	BuyVolume  string `json:"buy_volume"`  // decimal — Σ price×qty, taker buy
	SellVolume string `json:"sell_volume"` // decimal — Σ price×qty, taker sell
}

// VolumeProfile (VPVR — Volume Profile Visible Range) is a per-window,
// price-bucketed view of traded notional for one instrument and
// timeframe. It is decision-support (ADR-0027): it describes WHERE
// volume traded, never what to do about it.
//
// Computed from trade prints only (the foundry does not ingest
// order-book depth — ADR-0027 I4).
type VolumeProfile struct {
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`   // window duration in seconds
	BucketSize string                         `json:"bucket_size"` // decimal — price bucket width
	Buckets    []PriceBucket                  `json:"buckets"`     // ascending by PriceLevel
	TradeCount int64                          `json:"trade_count"`
	Overload   OverloadLevel                  `json:"overload"` // bucket-cap pressure level (L0–L3)
	OpenTime   time.Time                      `json:"open_time"`
	CloseTime  time.Time                      `json:"close_time"`
	Final      bool                           `json:"final"`
}

// VenueSymbol returns the lowercase venue-native symbol form
// (e.g., "btcusdt") derived from the canonical instrument.
//
// TRANSITORY ADAPTER (sunset H-6.f.2, shared shape with evidence).
// See ADR-0021. Lossy for delivery futures by design.
func (vp VolumeProfile) VenueSymbol() string {
	return strings.ToLower(string(vp.Instrument.Base) + string(vp.Instrument.Quote))
}

// Validate enforces the VolumeProfile invariants. A profile with zero
// buckets is valid (an empty window — no trades yet); each present
// bucket must be fully populated.
func (vp VolumeProfile) Validate() *problem.Problem {
	if vp.Source == "" {
		return problem.New(problem.InvalidArgument, "source is required")
	}
	if vp.Instrument.IsZero() {
		return problem.New(problem.InvalidArgument, "instrument is required")
	}
	if prob := vp.Instrument.Validate(); prob != nil {
		return prob
	}
	if vp.Timeframe <= 0 {
		return problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if vp.BucketSize == "" {
		return problem.New(problem.InvalidArgument, "bucket_size is required")
	}
	if prob := vp.Overload.Validate(); prob != nil {
		return prob
	}
	for i, b := range vp.Buckets {
		if b.PriceLevel == "" || b.BuyVolume == "" || b.SellVolume == "" {
			return problem.Validation(
				problem.InvalidArgument,
				"volume profile bucket is incomplete",
				problem.ValidationIssue{
					Field:   "buckets",
					Message: "each bucket needs price_level, buy_volume, sell_volume",
					Value:   vp.PriceLevel(i),
				},
			)
		}
	}
	if vp.OpenTime.IsZero() {
		return problem.New(problem.InvalidArgument, "open_time is required")
	}
	if vp.CloseTime.IsZero() {
		return problem.New(problem.InvalidArgument, "close_time is required")
	}
	if !vp.CloseTime.After(vp.OpenTime) {
		return problem.New(problem.InvalidArgument, "close_time must be after open_time")
	}
	return nil
}

// PriceLevel returns the price level of bucket i, or "" if out of
// range — a small helper used in validation messages.
func (vp VolumeProfile) PriceLevel(i int) string {
	if i < 0 || i >= len(vp.Buckets) {
		return ""
	}
	return vp.Buckets[i].PriceLevel
}
