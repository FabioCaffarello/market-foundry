package signal

import (
	"strconv"
	"time"

	"internal/domain/instrument"
	"internal/domain/signal"
)

// RSISampler computes the Relative Strength Index from a stream of close prices.
// It uses Wilder's smoothed moving average with a default period of 14.
// Pure application logic — no I/O dependencies.
type RSISampler struct {
	source     string
	symbol     string
	instrument instrument.CanonicalInstrument
	timeframe  int
	period     int

	// Warm-up: collect first period+1 prices to compute initial averages.
	prices []float64

	// After warm-up: smoothed running averages + previous close.
	prevClose float64
	avgGain   float64
	avgLoss   float64
	warmedUp  bool
}

// NewRSISamplerForInstrument constructs an RSISampler from a canonical
// Instrument directly — no source-string reconstruction.
//
// Pre-flight 5 of H-6.c documented the silent-zero regression-shape
// from H-6.b' commit 37f8ddd: when the upstream caller passes a Source
// string outside the hardcoded binances/binancef mapping (synthetic
// labels, configctl variations), instrumentFromBinding returns a zero
// CanonicalInstrument silently. This constructor is the canonical
// pass-through path that the migrated derive actors will call after
// commit 6 of H-6.c.1.
//
// The `symbol` struct field is left as zero value — it is unused by
// the sampler body and only persists in the struct for legacy
// back-compat with NewRSISampler.
func NewRSISamplerForInstrument(source string, inst instrument.CanonicalInstrument, timeframe int) *RSISampler {
	return &RSISampler{
		source:     source,
		instrument: inst,
		timeframe:  timeframe,
		period:     14,
	}
}

// NewRSISampler is the legacy constructor that reconstructs Instrument
// from a (source, symbol) string pair via instrumentFromBinding.
//
// DEPRECATED (H-6.c.1 → sunset H-6.f). Use NewRSISamplerForInstrument
// when the caller already holds a canonical Instrument (e.g., derive
// actors after they consume BindingTarget). See ADR-0021.
func NewRSISampler(source, symbol string, timeframe int) *RSISampler {
	s := NewRSISamplerForInstrument(source, instrumentFromBinding(source, symbol), timeframe)
	s.symbol = symbol
	return s
}

// AddClose processes a finalized candle close price.
// Returns a signal and true once enough data has been accumulated (period+1 candles).
func (s *RSISampler) AddClose(closePrice string, ts time.Time) (signal.Signal, bool) {
	price, err := strconv.ParseFloat(closePrice, 64)
	if err != nil {
		return signal.Signal{}, false
	}

	if !s.warmedUp {
		s.prices = append(s.prices, price)
		if len(s.prices) <= s.period {
			return signal.Signal{}, false
		}

		// period+1 prices → period changes. Compute initial averages.
		var gainSum, lossSum float64
		for i := 1; i <= s.period; i++ {
			change := s.prices[i] - s.prices[i-1]
			if change > 0 {
				gainSum += change
			} else {
				lossSum += -change
			}
		}
		s.avgGain = gainSum / float64(s.period)
		s.avgLoss = lossSum / float64(s.period)
		s.prevClose = price
		s.warmedUp = true
		s.prices = nil

		return s.buildSignal(ts), true
	}

	// Wilder's smoothing.
	change := price - s.prevClose
	var gain, loss float64
	if change > 0 {
		gain = change
	} else {
		loss = -change
	}

	p := float64(s.period)
	s.avgGain = (s.avgGain*(p-1) + gain) / p
	s.avgLoss = (s.avgLoss*(p-1) + loss) / p
	s.prevClose = price

	return s.buildSignal(ts), true
}

func (s *RSISampler) buildSignal(ts time.Time) signal.Signal {
	rsi := s.computeRSI()
	return signal.Signal{
		Type:       "rsi",
		Source:     s.source,
		Instrument: s.instrument,
		Timeframe:  s.timeframe,
		Value:      strconv.FormatFloat(rsi, 'f', 4, 64),
		Metadata: map[string]string{
			"period":   strconv.Itoa(s.period),
			"avg_gain": strconv.FormatFloat(s.avgGain, 'f', 8, 64),
			"avg_loss": strconv.FormatFloat(s.avgLoss, 'f', 8, 64),
		},
		Final:     true,
		Timestamp: ts,
	}
}

func (s *RSISampler) computeRSI() float64 {
	if s.avgLoss == 0 {
		return 100.0
	}
	rs := s.avgGain / s.avgLoss
	return 100.0 - (100.0 / (1.0 + rs))
}
