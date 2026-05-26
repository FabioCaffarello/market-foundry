package evidence

import (
	"strings"
	"time"

	"internal/domain/instrument"
	"internal/shared/problem"
)

// EvidenceCandle represents a sampled OHLCV candle for a specific instrument and timeframe.
// All monetary values (Open, High, Low, Close, Volume) are decimal strings to avoid IEEE 754 precision loss.
//
// Per ADR-0021, the canonical instrument identity is carried in the
// Instrument field. Migrated from Symbol string in H-6.b.
type EvidenceCandle struct {
	Source     string                         `json:"source"`      // Exchange identifier (e.g., "binancef")
	Instrument instrument.CanonicalInstrument `json:"instrument"`  // Canonical instrument identity (ADR-0021)
	Timeframe  int                            `json:"timeframe"`   // Window duration in seconds (e.g., 60, 300)
	Open       string                         `json:"open"`        // Decimal string
	High       string                         `json:"high"`        // Decimal string
	Low        string                         `json:"low"`         // Decimal string
	Close      string                         `json:"close"`       // Decimal string
	Volume     string                         `json:"volume"`      // Decimal string — total traded volume in window
	TradeCount int64                          `json:"trade_count"` // Number of trades in the window
	OpenTime   time.Time                      `json:"open_time"`   // Window start: floor(first_trade_ts / timeframe) * timeframe
	CloseTime  time.Time                      `json:"close_time"`  // Window end: open_time + timeframe
	Final      bool                           `json:"final"`       // True = window closed (immutable); false = interim/realtime
}

// VenueSymbol returns the lowercase venue-native symbol form
// (e.g., "btcusdt") derived from the canonical instrument.
//
// TRANSITORY ADAPTER (H-6.b): exists to keep KV partition keys,
// log labels, and ClickHouse rows compiling while the migration
// to CanonicalInstrument propagates. Slated for sunset in H-6.f,
// when the last venue-native string reader is removed in favor
// of canonical reasoning or explicit venue routing metadata.
//
// Known limitation: lossy for delivery futures (BTCUSDT_240329
// collapses to "btcusdt"). Acceptable in H-6.b because no
// delivery-futures contracts ride the current routing path;
// H-6.e revisits routing key shape.
func (c EvidenceCandle) VenueSymbol() string {
	return strings.ToLower(string(c.Instrument.Base) + string(c.Instrument.Quote))
}

func (c EvidenceCandle) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if c.Source == "" {
		issues = append(issues, problem.ValidationIssue{Field: "source", Message: "must not be empty"})
	}
	if c.Instrument.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "instrument", Message: "must not be zero"})
	} else if prob := c.Instrument.Validate(); prob != nil {
		return prob
	}
	if c.Timeframe <= 0 {
		issues = append(issues, problem.ValidationIssue{Field: "timeframe", Message: "must be a positive integer"})
	}
	if c.Open == "" {
		issues = append(issues, problem.ValidationIssue{Field: "open", Message: "must not be empty"})
	}
	if c.High == "" {
		issues = append(issues, problem.ValidationIssue{Field: "high", Message: "must not be empty"})
	}
	if c.Low == "" {
		issues = append(issues, problem.ValidationIssue{Field: "low", Message: "must not be empty"})
	}
	if c.Close == "" {
		issues = append(issues, problem.ValidationIssue{Field: "close", Message: "must not be empty"})
	}
	if c.Volume == "" {
		issues = append(issues, problem.ValidationIssue{Field: "volume", Message: "must not be empty"})
	}
	if c.OpenTime.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "open_time", Message: "must not be zero"})
	}
	if c.CloseTime.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "close_time", Message: "must not be zero"})
	}
	if !c.OpenTime.IsZero() && !c.CloseTime.IsZero() && !c.CloseTime.After(c.OpenTime) {
		issues = append(issues, problem.ValidationIssue{Field: "close_time", Message: "must be after open_time"})
	}

	if len(issues) == 0 {
		return nil
	}
	return problem.Validation(problem.InvalidArgument, "evidence candle is invalid", issues...)
}
