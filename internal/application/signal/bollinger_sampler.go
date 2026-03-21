package signal

import (
	"math"
	"strconv"
	"time"

	"internal/domain/signal"
)

// BollingerSampler computes Bollinger Bands from a stream of close prices.
// It uses a simple moving average (SMA) with configurable period and K multiplier.
// The primary output value is %B = (price - lower) / (upper - lower).
// Pure application logic — no I/O dependencies.
type BollingerSampler struct {
	source    string
	symbol    string
	timeframe int
	period    int
	k         float64

	// Rolling window of close prices.
	prices []float64
}

func NewBollingerSampler(source, symbol string, timeframe int) *BollingerSampler {
	return &BollingerSampler{
		source:    source,
		symbol:    symbol,
		timeframe: timeframe,
		period:    20,
		k:         2.0,
	}
}

// AddClose processes a finalized candle close price.
// Returns a signal and true once enough data has been accumulated (period candles).
func (s *BollingerSampler) AddClose(closePrice string, ts time.Time) (signal.Signal, bool) {
	price, err := strconv.ParseFloat(closePrice, 64)
	if err != nil {
		return signal.Signal{}, false
	}

	s.prices = append(s.prices, price)
	if len(s.prices) < s.period {
		return signal.Signal{}, false
	}

	// Keep only the last `period` prices.
	if len(s.prices) > s.period {
		s.prices = s.prices[len(s.prices)-s.period:]
	}

	sma := s.computeSMA()
	stddev := s.computeStdDev(sma)
	upper := sma + s.k*stddev
	lower := sma - s.k*stddev

	// %B: where current price sits relative to the bands.
	var pctB float64
	bandwidth := upper - lower
	if bandwidth > 0 {
		pctB = (price - lower) / bandwidth
	} else {
		pctB = 0.5 // bands collapsed — price is at midpoint by definition
	}

	return signal.Signal{
		Type:      "bollinger",
		Source:    s.source,
		Symbol:    s.symbol,
		Timeframe: s.timeframe,
		Value:     strconv.FormatFloat(pctB, 'f', 4, 64),
		Metadata: map[string]string{
			"period":    strconv.Itoa(s.period),
			"k":         strconv.FormatFloat(s.k, 'f', 1, 64),
			"sma":       strconv.FormatFloat(sma, 'f', 4, 64),
			"upper":     strconv.FormatFloat(upper, 'f', 4, 64),
			"lower":     strconv.FormatFloat(lower, 'f', 4, 64),
			"bandwidth": strconv.FormatFloat(bandwidth, 'f', 4, 64),
		},
		Final:     true,
		Timestamp: ts,
	}, true
}

func (s *BollingerSampler) computeSMA() float64 {
	var sum float64
	for _, p := range s.prices {
		sum += p
	}
	return sum / float64(len(s.prices))
}

func (s *BollingerSampler) computeStdDev(mean float64) float64 {
	var sumSq float64
	for _, p := range s.prices {
		diff := p - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(s.prices)))
}
