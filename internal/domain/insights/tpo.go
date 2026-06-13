package insights

import (
	"math/big"
	"strings"
	"time"

	"internal/domain/instrument"
	"internal/shared/problem"
)

// TPOMaxPeriods is the cap on time periods per TPO window — the classic
// market-profile letter range A..X (24). The sampler validates that
// timeframe / periodSeconds does not exceed this (Decisão T3).
const TPOMaxPeriods = 24

// DefaultValueAreaFraction is the share of TPO count that defines the
// value area (the conventional 70% band around the point of control).
const DefaultValueAreaFraction = 0.70

// PeriodLetter maps a zero-based period index to its market-profile
// letter ("A".."X"). Out-of-range indices return "" (the sampler caps
// at TPOMaxPeriods, so this only guards programmer error).
func PeriodLetter(index int) string {
	if index < 0 || index >= TPOMaxPeriods {
		return ""
	}
	return string(rune('A' + index))
}

// TPOPeriod is one time-slice of a TPO window, labelled by its letter
// (A..X). High/LowPrice are derived from the trade prints that fell in
// the period (trades-only, Decisão T2) — decimal strings to preserve
// exact precision (same convention as evidence/volume profile).
type TPOPeriod struct {
	Letter    string    `json:"letter"`     // "A".."X"
	StartTime time.Time `json:"start_time"` // period open (absolute)
	EndTime   time.Time `json:"end_time"`   // period close (absolute)
	HighPrice string    `json:"high_price"` // decimal — max trade price in period
	LowPrice  string    `json:"low_price"`  // decimal — min trade price in period
}

// TPOLevel is one price level of a TPO profile: WHICH periods traded at
// this level. Letters is the ascending, de-duplicated concatenation of
// the period labels that touched the level (e.g. "ACF"); Count is the
// number of distinct periods (= len(Letters)). PriceLevel is the
// canonical bucket lower bound (decimal string, via BucketLevel).
type TPOLevel struct {
	PriceLevel string `json:"price_level"` // decimal — bucket lower bound
	Letters    string `json:"letters"`     // ascending period labels, e.g. "ACF"
	Count      int    `json:"count"`       // distinct periods at this level
}

// TPOProfile (Time-Price Opportunity / market profile) is a per-window,
// timeframe-anchored (Decisão T1), trades-only (Decisão T2) view of
// WHICH price levels traded in WHICH time periods for one instrument
// and timeframe. It is decision-support (ADR-0027): it describes the
// time-at-price distribution, never what to do about it.
type TPOProfile struct {
	Source        string                         `json:"source"`
	Instrument    instrument.CanonicalInstrument `json:"instrument"`
	Timeframe     int                            `json:"timeframe"`      // window duration in seconds
	BucketSize    string                         `json:"bucket_size"`    // decimal — price bucket width
	PeriodSeconds int                            `json:"period_seconds"` // sub-period duration in seconds
	Periods       []TPOPeriod                    `json:"periods"`        // ascending by StartTime
	Levels        []TPOLevel                     `json:"levels"`         // ascending by PriceLevel

	// Derived metrics (computed in the snapshot, Decisão T4).
	POCPrice           string `json:"poc_price"`            // price level with most periods (Point of Control)
	ValueAreaHigh      string `json:"value_area_high"`      // upper bound of the ~70% value area
	ValueAreaLow       string `json:"value_area_low"`       // lower bound of the value area
	InitialBalanceHigh string `json:"initial_balance_high"` // high of the first two periods
	InitialBalanceLow  string `json:"initial_balance_low"`  // low of the first two periods
	RangeHigh          string `json:"range_high"`           // global high across periods
	RangeLow           string `json:"range_low"`            // global low across periods

	TradeCount int64         `json:"trade_count"`
	Overload   OverloadLevel `json:"overload"` // level-cap pressure (L0–L3), keyed on level count
	OpenTime   time.Time     `json:"open_time"`
	CloseTime  time.Time     `json:"close_time"`
	Final      bool          `json:"final"`
}

// VenueSymbol returns the lowercase venue-native symbol form (e.g.,
// "btcusdt") derived from the canonical instrument. TRANSITORY ADAPTER
// (shared shape with evidence / volume profile, sunset H-6.f.2).
func (tp TPOProfile) VenueSymbol() string {
	return strings.ToLower(string(tp.Instrument.Base) + string(tp.Instrument.Quote))
}

// Validate enforces the TPOProfile invariants. A profile with zero
// periods/levels is valid (an empty window — no trades yet); each
// present period/level must be fully populated.
func (tp TPOProfile) Validate() *problem.Problem {
	if tp.Source == "" {
		return problem.New(problem.InvalidArgument, "source is required")
	}
	if tp.Instrument.IsZero() {
		return problem.New(problem.InvalidArgument, "instrument is required")
	}
	if prob := tp.Instrument.Validate(); prob != nil {
		return prob
	}
	if tp.Timeframe <= 0 {
		return problem.New(problem.InvalidArgument, "timeframe must be positive")
	}
	if tp.PeriodSeconds <= 0 {
		return problem.New(problem.InvalidArgument, "period_seconds must be positive")
	}
	if tp.BucketSize == "" {
		return problem.New(problem.InvalidArgument, "bucket_size is required")
	}
	if len(tp.Periods) > TPOMaxPeriods {
		return problem.New(problem.InvalidArgument, "period count exceeds the A..X cap")
	}
	if prob := tp.Overload.Validate(); prob != nil {
		return prob
	}
	for _, p := range tp.Periods {
		if p.Letter == "" || p.HighPrice == "" || p.LowPrice == "" {
			return problem.Validation(
				problem.InvalidArgument,
				"tpo period is incomplete",
				problem.ValidationIssue{
					Field:   "periods",
					Message: "each period needs letter, high_price, low_price",
					Value:   p.Letter,
				},
			)
		}
	}
	for _, l := range tp.Levels {
		if l.PriceLevel == "" || l.Letters == "" || l.Count == 0 {
			return problem.Validation(
				problem.InvalidArgument,
				"tpo level is incomplete",
				problem.ValidationIssue{
					Field:   "levels",
					Message: "each level needs price_level, letters, count>0",
					Value:   l.PriceLevel,
				},
			)
		}
	}
	if tp.OpenTime.IsZero() {
		return problem.New(problem.InvalidArgument, "open_time is required")
	}
	if tp.CloseTime.IsZero() {
		return problem.New(problem.InvalidArgument, "close_time is required")
	}
	if !tp.CloseTime.After(tp.OpenTime) {
		return problem.New(problem.InvalidArgument, "close_time must be after open_time")
	}
	return nil
}

// PointOfControl returns the price level with the most distinct periods
// (the classic market-profile POC). Ties resolve to the lowest price.
// Returns "" for an empty level set. Levels must be ascending by price.
func PointOfControl(levels []TPOLevel) string {
	best := ""
	bestCount := -1
	for _, l := range levels {
		if l.Count > bestCount {
			bestCount = l.Count
			best = l.PriceLevel
		}
	}
	return best
}

// ValueArea returns the high/low price bounds of the value area — the
// smallest contiguous (by price order) band of levels around the POC
// whose cumulative TPO count reaches `fraction` of the total. Levels
// MUST be ascending by price. Returns ("","") for an empty set.
//
// Standard market-profile expansion: start at the POC, repeatedly add
// the neighbour (above or below) with the greater count until the
// target is met; ties extend upward.
func ValueArea(levels []TPOLevel, fraction float64) (high, low string) {
	n := len(levels)
	if n == 0 {
		return "", ""
	}
	total := 0
	pocIdx := 0
	for i, l := range levels {
		total += l.Count
		if l.Count > levels[pocIdx].Count {
			pocIdx = i
		}
	}
	target := int(float64(total)*fraction + 0.9999999) // ceil
	if target < 1 {
		target = 1
	}

	lo, hi := pocIdx, pocIdx
	running := levels[pocIdx].Count
	for running < target && (lo > 0 || hi < n-1) {
		below, above := -1, -1
		if lo > 0 {
			below = levels[lo-1].Count
		}
		if hi < n-1 {
			above = levels[hi+1].Count
		}
		if above >= below {
			hi++
			running += levels[hi].Count
		} else {
			lo--
			running += levels[lo].Count
		}
	}
	return levels[hi].PriceLevel, levels[lo].PriceLevel
}

// InitialBalance returns the high/low across the first n periods (the
// market-profile "initial balance", conventionally the first two
// periods A+B). Periods must be ordered by time. Returns ("","") when
// there are no periods.
func InitialBalance(periods []TPOPeriod, n int) (high, low string) {
	if len(periods) == 0 || n <= 0 {
		return "", ""
	}
	if n > len(periods) {
		n = len(periods)
	}
	return periodsHighLow(periods[:n])
}

// PriceRange returns the global high/low across all periods.
func PriceRange(periods []TPOPeriod) (high, low string) {
	return periodsHighLow(periods)
}

// periodsHighLow computes the max HighPrice and min LowPrice across the
// given periods using exact decimal comparison.
func periodsHighLow(periods []TPOPeriod) (high, low string) {
	for _, p := range periods {
		if p.HighPrice != "" && (high == "" || decimalGreater(p.HighPrice, high)) {
			high = p.HighPrice
		}
		if p.LowPrice != "" && (low == "" || decimalGreater(low, p.LowPrice)) {
			low = p.LowPrice
		}
	}
	return high, low
}

// decimalGreater reports whether decimal string a > b. Unparseable
// operands compare as not-greater (defensive; the sampler only feeds
// canonical decimals).
func decimalGreater(a, b string) bool {
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
