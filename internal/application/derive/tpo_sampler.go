package derive

import (
	"math/big"
	"sort"
	"strings"
	"time"

	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/domain/observation"
)

// TPOSampler accumulates trades into a per-window TPO (Time-Price
// Opportunity) profile. Pure application logic, no I/O. Mirrors the
// VolumeProfileSampler window/finalize lifecycle; the TPO dimension is
// time-at-price: each trade marks its price level (insights.BucketLevel)
// with the LETTER of the sub-period it fell in (Decisões T1–T3).
//
// Timeframe-anchored (T1): the window is the timeframe, subdivided into
// periodSeconds-wide periods (A..X). Trades-only (T2): period high/low
// and level letters are derived from trade prints. Insights are
// decision-support only (ADR-0027).
type TPOSampler struct {
	source        string
	timeframe     time.Duration
	bucketSize    string
	periodSeconds int
	maxLevels     int

	instrument instrument.CanonicalInstrument
	levels     map[string]map[int]struct{} // price level → set of period indices
	periods    map[int]*tpoPeriodAccum     // period index → high/low (exact strings)
	tradeCount int64
	openTime   time.Time
	closeTime  time.Time
	active     bool
}

type tpoPeriodAccum struct {
	high string // decimal — max trade price in period
	low  string // decimal — min trade price in period
}

// NewTPOSampler creates a TPO sampler. periodSeconds is the sub-period
// width (<=0 derives ~12 periods from the timeframe, capped to fit the
// A..X range); maxLevels caps distinct price levels per window (<=0 uses
// insights.DefaultMaxBucketsPerWindow).
func NewTPOSampler(source string, timeframe time.Duration, bucketSize string, periodSeconds, maxLevels int) *TPOSampler {
	if maxLevels <= 0 {
		maxLevels = insights.DefaultMaxBucketsPerWindow
	}
	if periodSeconds <= 0 {
		// Default: ~12 periods per window, but never below 1s and never
		// so fine that the window would exceed the A..X period cap.
		secs := int(timeframe.Seconds())
		periodSeconds = secs / 12
		if periodSeconds < 1 {
			periodSeconds = 1
		}
		if min := minPeriodSeconds(secs); periodSeconds < min {
			periodSeconds = min
		}
	} else if min := minPeriodSeconds(int(timeframe.Seconds())); periodSeconds < min {
		periodSeconds = min
	}
	return &TPOSampler{
		source:        source,
		timeframe:     timeframe,
		bucketSize:    bucketSize,
		periodSeconds: periodSeconds,
		maxLevels:     maxLevels,
	}
}

// minPeriodSeconds is the smallest period width that keeps the window
// within the A..X cap (ceil(timeframe / TPOMaxPeriods)).
func minPeriodSeconds(timeframeSecs int) int {
	if timeframeSecs <= 0 {
		return 1
	}
	m := (timeframeSecs + insights.TPOMaxPeriods - 1) / insights.TPOMaxPeriods
	if m < 1 {
		m = 1
	}
	return m
}

// WindowFor computes the window boundaries for a given timestamp.
func (s *TPOSampler) WindowFor(ts time.Time) (openTime, closeTime time.Time) {
	secs := int64(s.timeframe.Seconds())
	windowStart := ts.Unix() / secs * secs
	openTime = time.Unix(windowStart, 0).UTC()
	closeTime = openTime.Add(s.timeframe)
	return
}

// periodIndex maps a timestamp to its zero-based period index within the
// current window, clamped to the A..X cap (late trades lump into X).
func (s *TPOSampler) periodIndex(ts time.Time) int {
	if s.periodSeconds <= 0 {
		return 0
	}
	elapsed := ts.Unix() - s.openTime.Unix()
	if elapsed < 0 {
		elapsed = 0
	}
	idx := int(elapsed / int64(s.periodSeconds))
	if idx >= insights.TPOMaxPeriods {
		idx = insights.TPOMaxPeriods - 1
	}
	return idx
}

// AddTrade processes a trade, finalizing and returning the previous
// window when the trade crosses a window boundary.
func (s *TPOSampler) AddTrade(trade observation.ObservationTrade) (finalized insights.TPOProfile, didFinalize bool) {
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
		s.levels = make(map[string]map[int]struct{})
		s.periods = make(map[int]*tpoPeriodAccum)
		s.tradeCount = 0
		s.active = true
	}

	s.admit(trade)
	return
}

// admit marks the trade's price level with its period letter and updates
// the period's high/low. Honors the overload policy: at L3 a brand-new
// price level is dropped (existing levels still accumulate letters).
func (s *TPOSampler) admit(trade observation.ObservationTrade) {
	level, prob := insights.BucketLevel(trade.Price, s.bucketSize)
	if prob != nil {
		return // malformed price — skip silently (validated upstream)
	}
	idx := s.periodIndex(trade.Timestamp)

	set, exists := s.levels[level]
	if !exists {
		overload := insights.ClassifyOverload(len(s.levels), s.maxLevels)
		if !overload.AdmitsNewLevel() {
			return // L3: drop new price level
		}
		set = make(map[int]struct{})
		s.levels[level] = set
	}
	set[idx] = struct{}{}

	acc, ok := s.periods[idx]
	if !ok {
		s.periods[idx] = &tpoPeriodAccum{high: trade.Price, low: trade.Price}
	} else {
		if decGreater(trade.Price, acc.high) {
			acc.high = trade.Price
		}
		if decGreater(acc.low, trade.Price) {
			acc.low = trade.Price
		}
	}
	s.tradeCount++
}

func (s *TPOSampler) Active() bool { return s.active }

// Snapshot returns the current profile without finalizing.
func (s *TPOSampler) Snapshot() (insights.TPOProfile, bool) {
	if !s.active {
		return insights.TPOProfile{}, false
	}
	return s.snapshot(false), true
}

func (s *TPOSampler) snapshot(final bool) insights.TPOProfile {
	// Periods ascending by index (→ letter order).
	pIdxs := make([]int, 0, len(s.periods))
	for idx := range s.periods {
		pIdxs = append(pIdxs, idx)
	}
	sort.Ints(pIdxs)
	periods := make([]insights.TPOPeriod, 0, len(pIdxs))
	for _, idx := range pIdxs {
		acc := s.periods[idx]
		start := s.openTime.Add(time.Duration(idx*s.periodSeconds) * time.Second)
		periods = append(periods, insights.TPOPeriod{
			Letter:    insights.PeriodLetter(idx),
			StartTime: start,
			EndTime:   start.Add(time.Duration(s.periodSeconds) * time.Second),
			HighPrice: acc.high,
			LowPrice:  acc.low,
		})
	}

	// Levels ascending by numeric price.
	lvlKeys := make([]string, 0, len(s.levels))
	for k := range s.levels {
		lvlKeys = append(lvlKeys, k)
	}
	sort.Slice(lvlKeys, func(i, j int) bool {
		a, _ := new(big.Rat).SetString(lvlKeys[i])
		b, _ := new(big.Rat).SetString(lvlKeys[j])
		return a.Cmp(b) < 0
	})
	levels := make([]insights.TPOLevel, 0, len(lvlKeys))
	for _, lvl := range lvlKeys {
		set := s.levels[lvl]
		idxs := make([]int, 0, len(set))
		for idx := range set {
			idxs = append(idxs, idx)
		}
		sort.Ints(idxs)
		var sb strings.Builder
		for _, idx := range idxs {
			sb.WriteString(insights.PeriodLetter(idx))
		}
		levels = append(levels, insights.TPOLevel{
			PriceLevel: lvl,
			Letters:    sb.String(),
			Count:      len(idxs),
		})
	}

	poc := insights.PointOfControl(levels)
	vah, val := insights.ValueArea(levels, insights.DefaultValueAreaFraction)
	ibHigh, ibLow := insights.InitialBalance(periods, 2)
	rHigh, rLow := insights.PriceRange(periods)

	return insights.TPOProfile{
		Source:             s.source,
		Instrument:         s.instrument,
		Timeframe:          int(s.timeframe.Seconds()),
		BucketSize:         s.bucketSize,
		PeriodSeconds:      s.periodSeconds,
		Periods:            periods,
		Levels:             levels,
		POCPrice:           poc,
		ValueAreaHigh:      vah,
		ValueAreaLow:       val,
		InitialBalanceHigh: ibHigh,
		InitialBalanceLow:  ibLow,
		RangeHigh:          rHigh,
		RangeLow:           rLow,
		TradeCount:         s.tradeCount,
		Overload:           insights.ClassifyOverload(len(s.levels), s.maxLevels),
		OpenTime:           s.openTime,
		CloseTime:          s.closeTime,
		Final:              final,
	}
}

func (s *TPOSampler) reset() {
	s.active = false
	s.instrument = instrument.CanonicalInstrument{}
	s.levels = nil
	s.periods = nil
	s.tradeCount = 0
	s.openTime = time.Time{}
	s.closeTime = time.Time{}
}

// decGreater reports whether decimal string a > b. Unparseable operands
// compare as not-greater (defensive; trades are validated upstream).
func decGreater(a, b string) bool {
	ra, ok := new(big.Rat).SetString(a)
	if !ok {
		return false
	}
	rb, ok := new(big.Rat).SetString(b)
	if !ok {
		return false
	}
	return ra.Cmp(rb) > 0
}
