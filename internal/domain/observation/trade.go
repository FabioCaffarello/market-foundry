package observation

import (
	"time"

	"internal/shared/problem"
)

// ObservationTrade represents a normalized trade event captured from an external market source.
// Price and Quantity are decimal strings to avoid IEEE 754 precision loss.
type ObservationTrade struct {
	Source     string    `json:"source"`      // Exchange identifier (e.g., "binancef")
	Symbol    string    `json:"symbol"`      // Instrument symbol, lowercase (e.g., "btcusdt")
	Price     string    `json:"price"`       // Decimal string
	Quantity  string    `json:"quantity"`    // Decimal string
	TradeID   string    `json:"trade_id"`    // Source-assigned trade identifier
	BuyerMaker bool    `json:"buyer_maker"` // True if the buyer is the maker
	Timestamp time.Time `json:"timestamp"`   // Exchange-reported trade time
}

func (t ObservationTrade) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if t.Source == "" {
		issues = append(issues, problem.ValidationIssue{Field: "source", Message: "must not be empty"})
	}
	if t.Symbol == "" {
		issues = append(issues, problem.ValidationIssue{Field: "symbol", Message: "must not be empty"})
	}
	if t.Price == "" {
		issues = append(issues, problem.ValidationIssue{Field: "price", Message: "must not be empty"})
	}
	if t.Quantity == "" {
		issues = append(issues, problem.ValidationIssue{Field: "quantity", Message: "must not be empty"})
	}
	if t.TradeID == "" {
		issues = append(issues, problem.ValidationIssue{Field: "trade_id", Message: "must not be empty"})
	}
	if t.Timestamp.IsZero() {
		issues = append(issues, problem.ValidationIssue{Field: "timestamp", Message: "must not be zero"})
	}

	if len(issues) == 0 {
		return nil
	}
	return problem.Validation(problem.InvalidArgument, "observation trade is invalid", issues...)
}

// DeduplicationKey returns the NATS Msg-Id for JetStream deduplication.
func (t ObservationTrade) DeduplicationKey() string {
	return t.Source + ":" + t.TradeID
}
