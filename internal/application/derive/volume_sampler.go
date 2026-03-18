package derive

import (
	"math/big"
	"time"

	"internal/domain/evidence"
	"internal/domain/observation"
)

// VolumeSampler accumulates trades into a per-window volume profile.
// It tracks buy/sell notional volume and computes VWAP (volume-weighted average price).
// It is pure application logic with no I/O dependencies.
type VolumeSampler struct {
	source    string
	symbol    string
	timeframe time.Duration

	buyVolume  *big.Float // notional: Σ(price × qty) where BuyerMaker
	sellVolume *big.Float // notional: Σ(price × qty) where !BuyerMaker
	totalQty   *big.Float // raw quantity: Σ(qty) — VWAP denominator
	tradeCount int64
	openTime   time.Time
	closeTime  time.Time
	active     bool
}

func NewVolumeSampler(source, symbol string, timeframe time.Duration) *VolumeSampler {
	return &VolumeSampler{
		source:    source,
		symbol:    symbol,
		timeframe: timeframe,
	}
}

func (s *VolumeSampler) WindowFor(ts time.Time) (openTime, closeTime time.Time) {
	secs := int64(s.timeframe.Seconds())
	windowStart := ts.Unix() / secs * secs
	openTime = time.Unix(windowStart, 0).UTC()
	closeTime = openTime.Add(s.timeframe)
	return
}

func (s *VolumeSampler) AddTrade(trade observation.ObservationTrade) (finalized evidence.EvidenceVolume, didFinalize bool) {
	qty, _, _ := big.NewFloat(0).Parse(trade.Quantity, 10)
	price, _, _ := big.NewFloat(0).Parse(trade.Price, 10)
	notional := new(big.Float).Mul(price, qty)

	openTime, closeTime := s.WindowFor(trade.Timestamp)

	if s.active && openTime != s.openTime {
		finalized = s.snapshot(true)
		didFinalize = true
		s.reset()
	}

	if !s.active {
		s.openTime = openTime
		s.closeTime = closeTime
		s.buyVolume = new(big.Float)
		s.sellVolume = new(big.Float)
		s.totalQty = new(big.Float)
		s.tradeCount = 0
		s.active = true
	}

	s.tradeCount++
	s.totalQty.Add(s.totalQty, qty)
	if trade.BuyerMaker {
		s.buyVolume.Add(s.buyVolume, notional)
	} else {
		s.sellVolume.Add(s.sellVolume, notional)
	}

	return
}

func (s *VolumeSampler) Active() bool {
	return s.active
}

func (s *VolumeSampler) snapshot(final bool) evidence.EvidenceVolume {
	totalVol := new(big.Float).Add(s.buyVolume, s.sellVolume)

	vwap := new(big.Float)
	if s.totalQty.Sign() > 0 {
		vwap.Quo(totalVol, s.totalQty)
	}

	return evidence.EvidenceVolume{
		Source:      s.source,
		Symbol:      s.symbol,
		Timeframe:   int(s.timeframe.Seconds()),
		BuyVolume:   s.buyVolume.Text('f', 8),
		SellVolume:  s.sellVolume.Text('f', 8),
		TotalVolume: totalVol.Text('f', 8),
		VWAP:        vwap.Text('f', 8),
		TradeCount:  s.tradeCount,
		OpenTime:    s.openTime,
		CloseTime:   s.closeTime,
		Final:       final,
	}
}

func (s *VolumeSampler) reset() {
	s.active = false
	s.tradeCount = 0
	s.buyVolume = nil
	s.sellVolume = nil
	s.totalQty = nil
	s.openTime = time.Time{}
	s.closeTime = time.Time{}
}
