package signal

import (
	"strconv"
	"time"

	"internal/domain/signal"
)

// VWAPSampler computes the Volume Weighted Average Price from a stream of
// close prices and volumes.
// Pure application logic — no I/O dependencies.
//
// Output semantics:
//   - Value is the VWAP deviation ratio: (close − VWAP) / VWAP as a decimal string.
//   - Positive ratio → price is above VWAP (potential resistance zone).
//   - Negative ratio → price is below VWAP (potential support zone).
//
// Unlike other signal families that only consume close prices, VWAP requires
// both close price and volume — a fundamentally different input shape that
// proves the signal contract supports heterogeneous evidence consumption.
//
// Parameters:
//   - period = 20 (rolling window of candles for VWAP computation)
//
// Warm-up requires `period` candles before the first signal is emitted.
type VWAPSampler struct {
	source    string
	symbol    string
	timeframe int

	period int

	// Rolling windows for price and volume.
	closes  []float64
	volumes []float64
}

func NewVWAPSampler(source, symbol string, timeframe int) *VWAPSampler {
	return &VWAPSampler{
		source:    source,
		symbol:    symbol,
		timeframe: timeframe,
		period:    20,
	}
}

// AddCandle processes a finalized candle with close price and volume.
// Returns a signal and true once enough data has been accumulated (period candles).
//
// This method signature differs from other samplers' AddClose — VWAP is the
// first signal family to require volume, expanding the evidence input surface.
func (s *VWAPSampler) AddCandle(closePrice, volume string, ts time.Time) (signal.Signal, bool) {
	price, err := strconv.ParseFloat(closePrice, 64)
	if err != nil {
		return signal.Signal{}, false
	}
	vol, err := strconv.ParseFloat(volume, 64)
	if err != nil {
		return signal.Signal{}, false
	}

	s.closes = append(s.closes, price)
	s.volumes = append(s.volumes, vol)

	if len(s.closes) < s.period {
		return signal.Signal{}, false
	}

	// Keep only the last `period` entries.
	if len(s.closes) > s.period {
		s.closes = s.closes[len(s.closes)-s.period:]
		s.volumes = s.volumes[len(s.volumes)-s.period:]
	}

	vwap, totalVolume := s.computeVWAP()

	// Deviation ratio: how far price is from VWAP, normalized.
	var deviation float64
	if vwap > 0 {
		deviation = (price - vwap) / vwap
	}

	return signal.Signal{
		Type:      "vwap",
		Source:    s.source,
		Symbol:    s.symbol,
		Timeframe: s.timeframe,
		Value:     strconv.FormatFloat(deviation, 'f', 6, 64),
		Metadata: map[string]string{
			"period":       strconv.Itoa(s.period),
			"vwap":         strconv.FormatFloat(vwap, 'f', 4, 64),
			"close":        strconv.FormatFloat(price, 'f', 4, 64),
			"total_volume": strconv.FormatFloat(totalVolume, 'f', 4, 64),
			"deviation":    strconv.FormatFloat(deviation, 'f', 6, 64),
		},
		Final:     true,
		Timestamp: ts,
	}, true
}

// computeVWAP calculates VWAP = Σ(close × volume) / Σ(volume) over the rolling window.
func (s *VWAPSampler) computeVWAP() (vwap, totalVolume float64) {
	var pv float64
	for i := range s.closes {
		pv += s.closes[i] * s.volumes[i]
		totalVolume += s.volumes[i]
	}
	if totalVolume > 0 {
		vwap = pv / totalVolume
	}
	return vwap, totalVolume
}
