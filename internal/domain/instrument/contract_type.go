package instrument

import "internal/shared/problem"

// ContractType classifies how an instrument settles. Per ADR-0021,
// the canonical model discriminates four contract types covering
// the products the foundry handles in the near-to-medium term.
type ContractType string

const (
	// ContractSpot is physical settlement with no leverage by
	// default (matches Binance Spot, Coinbase, Kraken).
	ContractSpot ContractType = "spot"

	// ContractUSDTFutures is USDT-margined linear futures with
	// explicit expiry (matches Binance USDT quarterlies).
	ContractUSDTFutures ContractType = "usdtfutures"

	// ContractCoinFutures is coin-margined inverse futures
	// (matches Kraken Futures, Binance Coin quarterlies).
	ContractCoinFutures ContractType = "coinfutures"

	// ContractPerpetual is perpetual swap with no expiry and
	// funding rate (matches Hyperliquid, Bybit Perp, Binance
	// USDM perpetual).
	ContractPerpetual ContractType = "perpetual"
)

// ValidContractType reports whether c is a recognized contract
// type value. Returns false for any string not in the declared
// enum.
func ValidContractType(c ContractType) bool {
	switch c {
	case ContractSpot, ContractUSDTFutures, ContractCoinFutures, ContractPerpetual:
		return true
	default:
		return false
	}
}

// String returns the contract type's string form.
func (c ContractType) String() string { return string(c) }

// Validate reports whether the contract type value is recognized.
func (c ContractType) Validate() *problem.Problem {
	if !ValidContractType(c) {
		return problem.Validation(
			problem.InvalidArgument,
			"contract type is invalid",
			problem.ValidationIssue{
				Field:   "contract",
				Message: "must be one of: spot, usdtfutures, coinfutures, perpetual",
				Value:   string(c),
			},
		)
	}
	return nil
}
