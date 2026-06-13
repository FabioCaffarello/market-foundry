package derive

import (
	"math/big"
	"sort"
	"time"

	"internal/domain/insights"
	"internal/domain/observation"
)

// CrossVenueFusion accumulates trades for MANY canonical instruments
// into per-window, per-venue cross-venue snapshots (H-8.c). Unlike the
// per-source samplers (VolumeProfile/TPO — one actor per source), one
// fusion handles ALL instruments for a single timeframe and fuses
// across venues (the venue is the trade's Source; the canonical
// instrument is the join key, venue-agnostic per ADR-0021).
//
// Pure application logic, no I/O. Windowed by timeframe (Decisão C2):
// a window finalizes when a new trade for that instrument crosses the
// boundary (same finalize-on-next-trade semantics as the samplers).
type CrossVenueFusion struct {
	timeframe time.Duration
	windows   map[string]*cvWindow // key: canonical instrument Symbol()
}

type cvWindow struct {
	instrument observation.ObservationTrade // carries the canonical instrument
	openTime   time.Time
	closeTime  time.Time
	venues     map[string]*cvVenueAccum // key: source (venue)
	tradeCount int64
}

type cvVenueAccum struct {
	tradeCount int64
	notional   *big.Float
	last       string // last trade price by arrival order
	high       string
	low        string
}

// NewCrossVenueFusion creates a fusion for one timeframe.
func NewCrossVenueFusion(timeframe time.Duration) *CrossVenueFusion {
	return &CrossVenueFusion{
		timeframe: timeframe,
		windows:   make(map[string]*cvWindow),
	}
}

func (f *CrossVenueFusion) windowFor(ts time.Time) (openTime, closeTime time.Time) {
	secs := int64(f.timeframe.Seconds())
	windowStart := ts.Unix() / secs * secs
	openTime = time.Unix(windowStart, 0).UTC()
	closeTime = openTime.Add(f.timeframe)
	return
}

// AddTrade folds a trade into its instrument's window, finalizing and
// returning the previous window when the trade crosses the boundary.
func (f *CrossVenueFusion) AddTrade(trade observation.ObservationTrade) (finalized insights.CrossVenueSnapshot, didFinalize bool) {
	key := trade.Instrument.Symbol()
	openTime, closeTime := f.windowFor(trade.Timestamp)

	w := f.windows[key]
	if w != nil && !w.openTime.Equal(openTime) {
		finalized = f.snapshot(w, true)
		didFinalize = true
		w = nil
	}
	if w == nil {
		w = &cvWindow{
			instrument: trade,
			openTime:   openTime,
			closeTime:  closeTime,
			venues:     make(map[string]*cvVenueAccum),
		}
		f.windows[key] = w
	}

	f.admit(w, trade)
	return
}

func (f *CrossVenueFusion) admit(w *cvWindow, trade observation.ObservationTrade) {
	acc := w.venues[trade.Source]
	if acc == nil {
		acc = &cvVenueAccum{notional: new(big.Float)}
		w.venues[trade.Source] = acc
	}
	acc.tradeCount++
	w.tradeCount++

	price, _, _ := big.NewFloat(0).Parse(trade.Price, 10)
	qty, _, _ := big.NewFloat(0).Parse(trade.Quantity, 10)
	acc.notional.Add(acc.notional, new(big.Float).Mul(price, qty))

	acc.last = trade.Price
	if acc.high == "" || decGreater(trade.Price, acc.high) {
		acc.high = trade.Price
	}
	if acc.low == "" || decGreater(acc.low, trade.Price) {
		acc.low = trade.Price
	}
}

func (f *CrossVenueFusion) snapshot(w *cvWindow, final bool) insights.CrossVenueSnapshot {
	venueKeys := make([]string, 0, len(w.venues))
	for k := range w.venues {
		venueKeys = append(venueKeys, k)
	}
	sort.Strings(venueKeys)

	rows := make([]insights.VenueRow, 0, len(venueKeys))
	for _, vk := range venueKeys {
		acc := w.venues[vk]
		rows = append(rows, insights.VenueRow{
			Venue:      vk,
			TradeCount: acc.tradeCount,
			Notional:   acc.notional.Text('f', 8),
			LastPrice:  acc.last,
			HighPrice:  acc.high,
			LowPrice:   acc.low,
		})
	}

	spreadAbs, spreadBps, mid := insights.ConsolidatedSpread(rows)
	return insights.CrossVenueSnapshot{
		Instrument:    w.instrument.Instrument,
		Timeframe:     int(f.timeframe.Seconds()),
		Venues:        rows,
		SpreadAbs:     spreadAbs,
		SpreadBps:     spreadBps,
		MidPrice:      mid,
		DominantVenue: insights.DominantVenue(rows),
		TradeCount:    w.tradeCount,
		OpenTime:      w.openTime,
		CloseTime:     w.closeTime,
		Final:         final,
	}
}
