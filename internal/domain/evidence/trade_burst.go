package evidence

import (
	"time"

	"internal/shared/problem"
)

// EvidenceTradeBurst is a per-window summary of trade activity for a specific symbol and timeframe.
// Emitted every finalized window. Captures metrics that candles do not: buy/sell volume split and burst detection.
// All monetary values are decimal strings to avoid IEEE 754 precision loss.
type EvidenceTradeBurst struct {
	Source     string    `json:"source"`      // Exchange identifier (e.g., "binancef")
	Symbol    string    `json:"symbol"`      // Instrument symbol, lowercase (e.g., "btcusdt")
	Timeframe int       `json:"timeframe"`   // Window duration in seconds (e.g., 60, 300)
	TradeCount int64    `json:"trade_count"` // Total trades in window
	BuyVolume  string   `json:"buy_volume"`  // Decimal string — volume where buyer is maker
	SellVolume string   `json:"sell_volume"` // Decimal string — volume where buyer is taker
	OpenTime  time.Time `json:"open_time"`   // Window start
	CloseTime time.Time `json:"close_time"`  // Window end
	Burst     bool      `json:"burst"`       // True if trade_count significantly exceeds baseline
	Final     bool      `json:"final"`       // True = window closed (immutable)
}

func (b EvidenceTradeBurst) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if b.Source == "" {
		issues = append(issues, problem.ValidationIssue{Field: "source", Message: "must not be empty"})
	}
	if b.Symbol == "" {
		issues = append(issues, problem.ValidationIssue{Field: "symbol", Message: "must not be empty"})
	}
	if b.Timeframe <= 0 {
		issues = append(issues, problem.ValidationIssue{Field: "timeframe", Message: "must be a positive integer"})
	}
	if b.BuyVolume == "" {
		issues = append(issues, problem.ValidationIssue{Field: "buy_volume", Message: "must not be empty"})
	}
	if b.SellVolume == "" {
		issues = append(issues, problem.ValidationIssue{Field: "sell_volume", Message: "must not be empty"})
	}
	if b.OpenTime.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "open_time", Message: "must not be zero"})
	}
	if b.CloseTime.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "close_time", Message: "must not be zero"})
	}
	if !b.OpenTime.IsZero() && !b.CloseTime.IsZero() && !b.CloseTime.After(b.OpenTime) {
		issues = append(issues, problem.ValidationIssue{Field: "close_time", Message: "must be after open_time"})
	}

	if len(issues) == 0 {
		return nil
	}
	return problem.Validation(problem.InvalidArgument, "evidence trade burst is invalid", issues...)
}
