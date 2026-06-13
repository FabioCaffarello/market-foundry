package derive

import (
	"math/big"
	"sort"
	"time"

	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/domain/observation"
)

// VolumeProfileSampler accumulates trades into a per-window,
// price-bucketed volume profile (VPVR). Pure application logic, no
// I/O. Mirrors VolumeSampler's window/finalize lifecycle; the new
// dimension is price binning (insights.BucketLevel) with a bounded
// bucket map (insights overload — Decisão #5 of PROGRAM-0005).
//
// Insights are decision-support only (ADR-0027): this sampler emits
// VolumeProfile, never a directive.
type VolumeProfileSampler struct {
	source     string
	timeframe  time.Duration
	bucketSize string
	maxBuckets int

	instrument instrument.CanonicalInstrument
	buckets    map[string]*bucketAccum // key: canonical price level
	tradeCount int64
	openTime   time.Time
	closeTime  time.Time
	active     bool
}

type bucketAccum struct {
	buy  *big.Float
	sell *big.Float
}

// NewVolumeProfileSampler creates a sampler. bucketSize is the price
// bucket width (decimal string); maxBuckets caps distinct price
// levels per window (<=0 uses insights.DefaultMaxBucketsPerWindow).
func NewVolumeProfileSampler(source string, timeframe time.Duration, bucketSize string, maxBuckets int) *VolumeProfileSampler {
	if maxBuckets <= 0 {
		maxBuckets = insights.DefaultMaxBucketsPerWindow
	}
	return &VolumeProfileSampler{
		source:     source,
		timeframe:  timeframe,
		bucketSize: bucketSize,
		maxBuckets: maxBuckets,
	}
}

// WindowFor computes the window boundaries for a given timestamp.
func (s *VolumeProfileSampler) WindowFor(ts time.Time) (openTime, closeTime time.Time) {
	secs := int64(s.timeframe.Seconds())
	windowStart := ts.Unix() / secs * secs
	openTime = time.Unix(windowStart, 0).UTC()
	closeTime = openTime.Add(s.timeframe)
	return
}

// AddTrade processes a trade, finalizing and returning the previous
// window when the trade crosses a window boundary.
func (s *VolumeProfileSampler) AddTrade(trade observation.ObservationTrade) (finalized insights.VolumeProfile, didFinalize bool) {
	openTime, closeTime := s.WindowFor(trade.Timestamp)

	if s.active && openTime != s.openTime {
		finalized = s.snapshot(true)
		didFinalize = true
		s.reset()
	}

	if !s.active {
		s.openTime = openTime
		s.closeTime = closeTime
		s.instrument = trade.Instrument
		s.buckets = make(map[string]*bucketAccum)
		s.tradeCount = 0
		s.active = true
	}

	s.admit(trade)
	return
}

// admit bins the trade into its price bucket, honoring the overload
// policy: at L3 a brand-new price level is dropped (existing buckets
// still accumulate) — bounded memory under a pathological tick stream.
func (s *VolumeProfileSampler) admit(trade observation.ObservationTrade) {
	level, prob := insights.BucketLevel(trade.Price, s.bucketSize)
	if prob != nil {
		return // malformed price — skip silently (already validated upstream)
	}

	acc, exists := s.buckets[level]
	if !exists {
		overload := insights.ClassifyOverload(len(s.buckets), s.maxBuckets)
		if !overload.AdmitsNewLevel() {
			return // L3: drop new price level
		}
		acc = &bucketAccum{buy: new(big.Float), sell: new(big.Float)}
		s.buckets[level] = acc
	}

	qty, _, _ := big.NewFloat(0).Parse(trade.Quantity, 10)
	price, _, _ := big.NewFloat(0).Parse(trade.Price, 10)
	notional := new(big.Float).Mul(price, qty)
	if trade.BuyerMaker {
		acc.buy.Add(acc.buy, notional)
	} else {
		acc.sell.Add(acc.sell, notional)
	}
	s.tradeCount++
}

func (s *VolumeProfileSampler) Active() bool { return s.active }

// Snapshot returns the current profile without finalizing.
func (s *VolumeProfileSampler) Snapshot() (insights.VolumeProfile, bool) {
	if !s.active {
		return insights.VolumeProfile{}, false
	}
	return s.snapshot(false), true
}

func (s *VolumeProfileSampler) snapshot(final bool) insights.VolumeProfile {
	levels := make([]string, 0, len(s.buckets))
	for k := range s.buckets {
		levels = append(levels, k)
	}
	// Deterministic ascending numeric order by price level.
	sort.Slice(levels, func(i, j int) bool {
		a, _ := new(big.Rat).SetString(levels[i])
		b, _ := new(big.Rat).SetString(levels[j])
		return a.Cmp(b) < 0
	})

	buckets := make([]insights.PriceBucket, 0, len(levels))
	for _, lvl := range levels {
		acc := s.buckets[lvl]
		buckets = append(buckets, insights.PriceBucket{
			PriceLevel: lvl,
			BuyVolume:  acc.buy.Text('f', 8),
			SellVolume: acc.sell.Text('f', 8),
		})
	}

	return insights.VolumeProfile{
		Source:     s.source,
		Instrument: s.instrument,
		Timeframe:  int(s.timeframe.Seconds()),
		BucketSize: s.bucketSize,
		Buckets:    buckets,
		TradeCount: s.tradeCount,
		Overload:   insights.ClassifyOverload(len(s.buckets), s.maxBuckets),
		OpenTime:   s.openTime,
		CloseTime:  s.closeTime,
		Final:      final,
	}
}

func (s *VolumeProfileSampler) reset() {
	s.active = false
	s.instrument = instrument.CanonicalInstrument{}
	s.buckets = nil
	s.tradeCount = 0
	s.openTime = time.Time{}
	s.closeTime = time.Time{}
}
