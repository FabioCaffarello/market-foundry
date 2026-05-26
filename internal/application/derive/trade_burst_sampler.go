package derive

import (
	"math/big"
	"time"

	"internal/domain/evidence"
	"internal/domain/instrument"
	"internal/domain/observation"
)

// burstThresholdRatio is the multiplier over the previous window's trade count
// that triggers the Burst flag. 2.0 means "more than double the previous window".
const burstThresholdRatio = 2.0

// TradeBurstSampler accumulates trades into a per-window activity summary.
// It tracks buy/sell volume separately and detects bursts by comparing
// the current window's trade count to the previous window's count.
// It is pure application logic with no I/O dependencies.
type TradeBurstSampler struct {
	source    string
	symbol    string
	timeframe time.Duration

	// Current window state.
	instrument instrument.CanonicalInstrument
	tradeCount int64
	buyVolume  *big.Float
	sellVolume *big.Float
	openTime   time.Time
	closeTime  time.Time
	active     bool

	// Baseline: previous window's trade count for burst detection.
	prevTradeCount int64
}

// NewTradeBurstSampler creates a sampler for the given source, symbol, and timeframe.
func NewTradeBurstSampler(source, symbol string, timeframe time.Duration) *TradeBurstSampler {
	return &TradeBurstSampler{
		source:    source,
		symbol:    symbol,
		timeframe: timeframe,
	}
}

// WindowFor computes the window boundaries for a given timestamp.
func (s *TradeBurstSampler) WindowFor(ts time.Time) (openTime, closeTime time.Time) {
	secs := int64(s.timeframe.Seconds())
	windowStart := ts.Unix() / secs * secs
	openTime = time.Unix(windowStart, 0).UTC()
	closeTime = openTime.Add(s.timeframe)
	return
}

// AddTrade processes a trade. If the trade belongs to a new window, the current
// window is finalized and returned. Same finalization pattern as CandleSampler.
func (s *TradeBurstSampler) AddTrade(trade observation.ObservationTrade) (finalized evidence.EvidenceTradeBurst, didFinalize bool) {
	qty, _, _ := big.NewFloat(0).Parse(trade.Quantity, 10)
	price, _, _ := big.NewFloat(0).Parse(trade.Price, 10)
	vol := new(big.Float).Mul(price, qty)

	openTime, closeTime := s.WindowFor(trade.Timestamp)

	// If the trade belongs to a different window, finalize the current one.
	if s.active && openTime != s.openTime {
		finalized = s.snapshot(true)
		didFinalize = true
		s.prevTradeCount = s.tradeCount
		s.reset()
	}

	if !s.active {
		s.openTime = openTime
		s.closeTime = closeTime
		s.instrument = trade.Instrument
		s.buyVolume = new(big.Float)
		s.sellVolume = new(big.Float)
		s.tradeCount = 0
		s.active = true
	}

	s.tradeCount++
	if trade.BuyerMaker {
		s.buyVolume.Add(s.buyVolume, vol)
	} else {
		s.sellVolume.Add(s.sellVolume, vol)
	}

	return
}

// Active reports whether the sampler has an open window.
func (s *TradeBurstSampler) Active() bool {
	return s.active
}

func (s *TradeBurstSampler) snapshot(final bool) evidence.EvidenceTradeBurst {
	burst := false
	if s.prevTradeCount > 0 {
		burst = float64(s.tradeCount) > burstThresholdRatio*float64(s.prevTradeCount)
	}

	return evidence.EvidenceTradeBurst{
		Source:     s.source,
		Instrument: s.instrument,
		Timeframe:  int(s.timeframe.Seconds()),
		TradeCount: s.tradeCount,
		BuyVolume:  s.buyVolume.Text('f', 8),
		SellVolume: s.sellVolume.Text('f', 8),
		OpenTime:   s.openTime,
		CloseTime:  s.closeTime,
		Burst:      burst,
		Final:      final,
	}
}

func (s *TradeBurstSampler) reset() {
	s.active = false
	s.instrument = instrument.CanonicalInstrument{}
	s.tradeCount = 0
	s.buyVolume = nil
	s.sellVolume = nil
	s.openTime = time.Time{}
	s.closeTime = time.Time{}
}
