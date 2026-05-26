package signal

import (
	"math"
	"strconv"
	"time"

	"internal/domain/instrument"
	"internal/domain/signal"
)

// EMACrossoverSampler computes two Exponential Moving Averages (fast and slow)
// from a stream of close prices and detects crossover events.
// Pure application logic — no I/O dependencies.
//
// Output semantics:
//   - "bullish"  — fast EMA is above slow EMA
//   - "bearish"  — fast EMA is below slow EMA
//   - "neutral"  — warm-up incomplete or EMAs equal within tolerance
type EMACrossoverSampler struct {
	source     string
	symbol     string
	instrument instrument.CanonicalInstrument
	timeframe  int

	fastPeriod int
	slowPeriod int

	// Warm-up: collect first slowPeriod prices to seed both EMAs.
	prices []float64

	fastEMA  float64
	slowEMA  float64
	warmedUp bool
}

// NewEMACrossoverSamplerForInstrument constructs an EMACrossoverSampler
// from a canonical Instrument directly. See NewRSISamplerForInstrument
// for the rationale (H-6.c.1; pre-flight 5 regression-shape avoidance).
func NewEMACrossoverSamplerForInstrument(source string, inst instrument.CanonicalInstrument, timeframe int) *EMACrossoverSampler {
	return &EMACrossoverSampler{
		source:     source,
		instrument: inst,
		timeframe:  timeframe,
		fastPeriod: 9,
		slowPeriod: 21,
	}
}

// NewEMACrossoverSampler is the legacy (source, symbol) constructor.
// DEPRECATED (H-6.c.1 → sunset H-6.f). Use NewEMACrossoverSamplerForInstrument.
func NewEMACrossoverSampler(source, symbol string, timeframe int) *EMACrossoverSampler {
	s := NewEMACrossoverSamplerForInstrument(source, instrumentFromBinding(source, symbol), timeframe)
	s.symbol = symbol
	return s
}

// AddClose processes a finalized candle close price.
// Returns a signal and true once enough data has been accumulated (slowPeriod candles).
func (s *EMACrossoverSampler) AddClose(closePrice string, ts time.Time) (signal.Signal, bool) {
	price, err := strconv.ParseFloat(closePrice, 64)
	if err != nil {
		return signal.Signal{}, false
	}

	if !s.warmedUp {
		s.prices = append(s.prices, price)
		if len(s.prices) < s.slowPeriod {
			return signal.Signal{}, false
		}

		// Seed both EMAs with simple moving averages over their respective windows.
		s.fastEMA = sma(s.prices[len(s.prices)-s.fastPeriod:])
		s.slowEMA = sma(s.prices)
		s.warmedUp = true
		s.prices = nil

		return s.buildSignal(price, ts), true
	}

	// EMA update: EMA_new = price * k + EMA_prev * (1 - k)
	fastK := 2.0 / float64(s.fastPeriod+1)
	slowK := 2.0 / float64(s.slowPeriod+1)

	s.fastEMA = price*fastK + s.fastEMA*(1-fastK)
	s.slowEMA = price*slowK + s.slowEMA*(1-slowK)

	return s.buildSignal(price, ts), true
}

func (s *EMACrossoverSampler) buildSignal(price float64, ts time.Time) signal.Signal {
	spread := s.fastEMA - s.slowEMA
	value := crossoverDirection(spread)

	return signal.Signal{
		Type:       "ema_crossover",
		Source:     s.source,
		Instrument: s.instrument,
		Timeframe:  s.timeframe,
		Value:      value,
		Metadata: map[string]string{
			"fast_period": strconv.Itoa(s.fastPeriod),
			"slow_period": strconv.Itoa(s.slowPeriod),
			"fast_ema":    strconv.FormatFloat(s.fastEMA, 'f', 4, 64),
			"slow_ema":    strconv.FormatFloat(s.slowEMA, 'f', 4, 64),
			"spread":      strconv.FormatFloat(spread, 'f', 4, 64),
		},
		Final:     true,
		Timestamp: ts,
	}
}

// crossoverDirection returns "bullish", "bearish", or "neutral" based on the spread
// between fast and slow EMA. A tolerance of 1e-8 prevents noise-level flips.
func crossoverDirection(spread float64) string {
	const tolerance = 1e-8
	if math.Abs(spread) < tolerance {
		return "neutral"
	}
	if spread > 0 {
		return "bullish"
	}
	return "bearish"
}

// sma returns the simple moving average of the given slice.
func sma(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
