package signal

import (
	"strconv"
	"time"

	"internal/domain/signal"
)

// MACDSampler computes the MACD (Moving Average Convergence Divergence) indicator
// from a stream of close prices.
// Pure application logic — no I/O dependencies.
//
// Output semantics:
//   - Value is the MACD histogram (MACD line − signal line) as a decimal string.
//   - Positive histogram → bullish momentum divergence.
//   - Negative histogram → bearish momentum divergence.
//
// Parameters (standard MACD 12/26/9):
//   - fastPeriod  = 12  (EMA of close prices)
//   - slowPeriod  = 26  (EMA of close prices)
//   - signalPeriod = 9  (EMA of the MACD line)
//
// Warm-up requires slowPeriod + signalPeriod − 1 = 34 candles before the first
// complete signal (MACD line + signal line + histogram) is emitted.
type MACDSampler struct {
	source    string
	symbol    string
	timeframe int

	fastPeriod   int
	slowPeriod   int
	signalPeriod int

	// Warm-up phase 1: accumulate slowPeriod prices to seed both EMAs.
	prices []float64

	fastEMA   float64
	slowEMA   float64
	emaSeeded bool

	// Warm-up phase 2: accumulate signalPeriod MACD values to seed signal EMA.
	macdValues  []float64
	signalEMA   float64
	signalReady bool
}

func NewMACDSampler(source, symbol string, timeframe int) *MACDSampler {
	return &MACDSampler{
		source:       source,
		symbol:       symbol,
		timeframe:    timeframe,
		fastPeriod:   12,
		slowPeriod:   26,
		signalPeriod: 9,
	}
}

// AddClose processes a finalized candle close price.
// Returns a signal and true once enough data has been accumulated
// (slowPeriod + signalPeriod − 1 = 34 candles).
func (s *MACDSampler) AddClose(closePrice string, ts time.Time) (signal.Signal, bool) {
	price, err := strconv.ParseFloat(closePrice, 64)
	if err != nil {
		return signal.Signal{}, false
	}

	if !s.emaSeeded {
		s.prices = append(s.prices, price)
		if len(s.prices) < s.slowPeriod {
			return signal.Signal{}, false
		}

		// Seed both EMAs with simple moving averages over their respective windows.
		s.fastEMA = sma(s.prices[len(s.prices)-s.fastPeriod:])
		s.slowEMA = sma(s.prices)
		s.emaSeeded = true
		s.prices = nil

		macdLine := s.fastEMA - s.slowEMA
		return s.accumulateSignalEMA(macdLine, ts)
	}

	// EMA update: EMA_new = price * k + EMA_prev * (1 - k)
	fastK := 2.0 / float64(s.fastPeriod+1)
	slowK := 2.0 / float64(s.slowPeriod+1)

	s.fastEMA = price*fastK + s.fastEMA*(1-fastK)
	s.slowEMA = price*slowK + s.slowEMA*(1-slowK)

	macdLine := s.fastEMA - s.slowEMA

	if !s.signalReady {
		return s.accumulateSignalEMA(macdLine, ts)
	}

	// Signal EMA update.
	signalK := 2.0 / float64(s.signalPeriod+1)
	s.signalEMA = macdLine*signalK + s.signalEMA*(1-signalK)

	histogram := macdLine - s.signalEMA

	return s.buildSignal(macdLine, histogram, ts), true
}

// accumulateSignalEMA collects MACD line values until signalPeriod is reached,
// then seeds the signal EMA with the SMA of those values.
func (s *MACDSampler) accumulateSignalEMA(macdLine float64, ts time.Time) (signal.Signal, bool) {
	s.macdValues = append(s.macdValues, macdLine)
	if len(s.macdValues) < s.signalPeriod {
		return signal.Signal{}, false
	}

	s.signalEMA = sma(s.macdValues)
	s.signalReady = true
	s.macdValues = nil

	histogram := macdLine - s.signalEMA

	return s.buildSignal(macdLine, histogram, ts), true
}

func (s *MACDSampler) buildSignal(macdLine, histogram float64, ts time.Time) signal.Signal {
	return signal.Signal{
		Type:      "macd",
		Source:    s.source,
		Symbol:    s.symbol,
		Timeframe: s.timeframe,
		Value:     strconv.FormatFloat(histogram, 'f', 4, 64),
		Metadata: map[string]string{
			"fast_period":   strconv.Itoa(s.fastPeriod),
			"slow_period":   strconv.Itoa(s.slowPeriod),
			"signal_period": strconv.Itoa(s.signalPeriod),
			"fast_ema":      strconv.FormatFloat(s.fastEMA, 'f', 4, 64),
			"slow_ema":      strconv.FormatFloat(s.slowEMA, 'f', 4, 64),
			"macd_line":     strconv.FormatFloat(macdLine, 'f', 4, 64),
			"signal_line":   strconv.FormatFloat(s.signalEMA, 'f', 4, 64),
			"histogram":     strconv.FormatFloat(histogram, 'f', 4, 64),
		},
		Final:     true,
		Timestamp: ts,
	}
}
