package signal

import (
	"strconv"
	"time"

	"internal/domain/instrument"
	"internal/domain/signal"
)

// ATRSampler computes the Average True Range (ATR) indicator from a stream
// of high, low, and close prices.
// Pure application logic — no I/O dependencies.
//
// Output semantics:
//   - Value is the ATR as a decimal string (average of true ranges).
//   - Higher ATR → higher volatility.
//   - Lower ATR → lower volatility / consolidation.
//
// Parameters (standard ATR-14):
//   - period = 14 (smoothing window)
//
// Warm-up requires period + 1 candles (15) before the first signal is emitted,
// because True Range requires a previous close, and the initial ATR is seeded
// with the SMA of the first `period` true ranges.
type ATRSampler struct {
	source     string
	symbol     string
	instrument instrument.CanonicalInstrument
	timeframe  int

	period int

	// Warm-up: accumulate true ranges until period is reached.
	trueRanges []float64
	prevClose  float64
	hasPrev    bool

	atr      float64
	atrReady bool
}

func NewATRSampler(source, symbol string, timeframe int) *ATRSampler {
	return &ATRSampler{
		source:     source,
		symbol:     symbol,
		instrument: instrumentFromBinding(source, symbol),
		timeframe:  timeframe,
		period:     14,
	}
}

// AddCandle processes a finalized candle with high, low, and close prices.
// Returns a signal and true once enough data has been accumulated
// (period + 1 = 15 candles).
func (s *ATRSampler) AddCandle(high, low, close string, ts time.Time) (signal.Signal, bool) {
	h, errH := strconv.ParseFloat(high, 64)
	l, errL := strconv.ParseFloat(low, 64)
	c, errC := strconv.ParseFloat(close, 64)
	if errH != nil || errL != nil || errC != nil {
		return signal.Signal{}, false
	}

	if !s.hasPrev {
		s.prevClose = c
		s.hasPrev = true
		return signal.Signal{}, false
	}

	// True Range = max(high - low, |high - prevClose|, |low - prevClose|)
	tr := trueRange(h, l, s.prevClose)
	s.prevClose = c

	if !s.atrReady {
		s.trueRanges = append(s.trueRanges, tr)
		if len(s.trueRanges) < s.period {
			return signal.Signal{}, false
		}

		// Seed ATR with SMA of first `period` true ranges.
		s.atr = sma(s.trueRanges)
		s.atrReady = true
		s.trueRanges = nil

		return s.buildSignal(tr, ts), true
	}

	// Wilder smoothing: ATR = (prevATR * (period-1) + TR) / period
	s.atr = (s.atr*float64(s.period-1) + tr) / float64(s.period)

	return s.buildSignal(tr, ts), true
}

func (s *ATRSampler) buildSignal(tr float64, ts time.Time) signal.Signal {
	return signal.Signal{
		Type:       "atr",
		Source:     s.source,
		Instrument: s.instrument,
		Timeframe:  s.timeframe,
		Value:      strconv.FormatFloat(s.atr, 'f', 4, 64),
		Metadata: map[string]string{
			"period":     strconv.Itoa(s.period),
			"atr":        strconv.FormatFloat(s.atr, 'f', 4, 64),
			"true_range": strconv.FormatFloat(tr, 'f', 4, 64),
		},
		Final:     true,
		Timestamp: ts,
	}
}

func trueRange(high, low, prevClose float64) float64 {
	hl := high - low
	hc := high - prevClose
	if hc < 0 {
		hc = -hc
	}
	lc := low - prevClose
	if lc < 0 {
		lc = -lc
	}

	max := hl
	if hc > max {
		max = hc
	}
	if lc > max {
		max = lc
	}
	return max
}
