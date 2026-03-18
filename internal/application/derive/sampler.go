package derive

import (
	"math/big"
	"time"

	"internal/domain/evidence"
	"internal/domain/observation"
)

// CandleSampler accumulates trades into a single OHLCV candle for a fixed timeframe.
// It is pure application logic with no I/O dependencies.
type CandleSampler struct {
	source    string
	symbol    string
	timeframe time.Duration

	open       *big.Float
	high       *big.Float
	low        *big.Float
	close      *big.Float
	volume     *big.Float
	tradeCount int64
	openTime   time.Time
	closeTime  time.Time
	active     bool
}

// NewCandleSampler creates a sampler for the given source, symbol, and timeframe.
func NewCandleSampler(source, symbol string, timeframe time.Duration) *CandleSampler {
	return &CandleSampler{
		source:    source,
		symbol:    symbol,
		timeframe: timeframe,
	}
}

// WindowFor computes the window boundaries for a given timestamp.
func (s *CandleSampler) WindowFor(ts time.Time) (openTime, closeTime time.Time) {
	secs := int64(s.timeframe.Seconds())
	windowStart := ts.Unix() / secs * secs
	openTime = time.Unix(windowStart, 0).UTC()
	closeTime = openTime.Add(s.timeframe)
	return
}

// AddTrade processes a trade. If the trade belongs to a new window, the current
// window is finalized and returned. If no window was finalized, the returned
// candle has Final == false and should be ignored.
func (s *CandleSampler) AddTrade(trade observation.ObservationTrade) (finalized evidence.EvidenceCandle, didFinalize bool) {
	price, _, _ := big.NewFloat(0).Parse(trade.Price, 10)
	qty, _, _ := big.NewFloat(0).Parse(trade.Quantity, 10)

	openTime, closeTime := s.WindowFor(trade.Timestamp)

	// If the trade belongs to a different window than the current one, finalize the current window.
	if s.active && openTime != s.openTime {
		finalized = s.snapshot(true)
		didFinalize = true
		s.reset()
	}

	if !s.active {
		s.openTime = openTime
		s.closeTime = closeTime
		s.open = new(big.Float).Copy(price)
		s.high = new(big.Float).Copy(price)
		s.low = new(big.Float).Copy(price)
		s.close = new(big.Float).Copy(price)
		s.volume = new(big.Float).Mul(price, qty)
		s.tradeCount = 1
		s.active = true
		return
	}

	// Update OHLCV within the current window.
	if price.Cmp(s.high) > 0 {
		s.high.Copy(price)
	}
	if price.Cmp(s.low) < 0 {
		s.low.Copy(price)
	}
	s.close.Copy(price)
	s.volume.Add(s.volume, new(big.Float).Mul(price, qty))
	s.tradeCount++

	return
}

// Snapshot returns the current candle state without finalizing.
func (s *CandleSampler) Snapshot() (evidence.EvidenceCandle, bool) {
	if !s.active {
		return evidence.EvidenceCandle{}, false
	}
	return s.snapshot(false), true
}

func (s *CandleSampler) snapshot(final bool) evidence.EvidenceCandle {
	return evidence.EvidenceCandle{
		Source:     s.source,
		Symbol:     s.symbol,
		Timeframe:  int(s.timeframe.Seconds()),
		Open:       s.open.Text('f', 8),
		High:       s.high.Text('f', 8),
		Low:        s.low.Text('f', 8),
		Close:      s.close.Text('f', 8),
		Volume:     s.volume.Text('f', 8),
		TradeCount: s.tradeCount,
		OpenTime:   s.openTime,
		CloseTime:  s.closeTime,
		Final:      final,
	}
}

func (s *CandleSampler) reset() {
	s.active = false
	s.open = nil
	s.high = nil
	s.low = nil
	s.close = nil
	s.volume = nil
	s.tradeCount = 0
	s.openTime = time.Time{}
	s.closeTime = time.Time{}
}

// Active reports whether the sampler has an open window.
func (s *CandleSampler) Active() bool {
	return s.active
}
