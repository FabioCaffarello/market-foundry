package binancef

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"internal/domain/instrument"
	"internal/domain/observation"
	"internal/shared/events"
	"internal/shared/problem"
)

// AggTrade represents the raw Binance Futures aggTrade WebSocket message.
// See: https://developers.binance.com/docs/derivatives/usds-margined-futures/websocket-market-streams/Aggregate-Trade-Streams
type AggTrade struct {
	EventType    string `json:"e"` // "aggTrade"
	EventTime    int64  `json:"E"` // Event time (ms)
	Symbol       string `json:"s"` // Symbol (uppercase from Binance)
	AggTradeID   int64  `json:"a"` // Aggregate trade ID
	Price        string `json:"p"` // Price
	Quantity     string `json:"q"` // Quantity
	FirstTradeID int64  `json:"f"` // First trade ID
	LastTradeID  int64  `json:"l"` // Last trade ID
	TradeTime    int64  `json:"T"` // Trade time (ms)
	IsBuyerMaker bool   `json:"m"` // Is the buyer the maker?
}

const sourceName = "binancef"

// deliverySuffix matches Binance USDT-margined delivery futures
// symbol suffixes, e.g. "BTCUSDT_240329" → expiry 2024-03-29. The
// absence of this suffix on a binancef symbol means the contract is
// a perpetual swap.
var deliverySuffix = regexp.MustCompile(`_\d{6}$`)

// ParseAggTrade decodes a raw WebSocket message into an AggTrade.
func ParseAggTrade(data []byte) (AggTrade, *problem.Problem) {
	var raw AggTrade
	if err := json.Unmarshal(data, &raw); err != nil {
		return AggTrade{}, problem.Wrap(err, problem.InvalidArgument, "failed to parse aggTrade message")
	}
	if raw.EventType != "aggTrade" {
		return AggTrade{}, problem.New(problem.InvalidArgument, "unexpected event type: "+raw.EventType)
	}
	return raw, nil
}

// Normalize converts a Binance Futures AggTrade into a canonical
// ObservationTrade. The symbol parameter is the venue-native form
// supplied by the connector; it is parsed into a CanonicalInstrument
// here at the adapter / domain boundary, per ADR-0021.
func Normalize(raw AggTrade, symbol string) (observation.TradeReceivedEvent, *problem.Problem) {
	inst, prob := parseFuturesSymbol(symbol)
	if prob != nil {
		return observation.TradeReceivedEvent{}, prob
	}

	trade := observation.ObservationTrade{
		Source:     sourceName,
		Instrument: inst,
		Price:      raw.Price,
		Quantity:   raw.Quantity,
		TradeID:    formatTradeID(raw.AggTradeID),
		BuyerMaker: raw.IsBuyerMaker,
		Timestamp:  time.UnixMilli(raw.TradeTime).UTC(),
	}

	if prob := trade.Validate(); prob != nil {
		return observation.TradeReceivedEvent{}, prob
	}

	return observation.TradeReceivedEvent{
		Metadata: events.NewMetadata().WithOccurredAt(trade.Timestamp),
		Trade:    trade,
	}, nil
}

// parseFuturesSymbol translates a venue-native Binance USDT-margined
// Futures symbol into a CanonicalInstrument. The contract type is
// discriminated by the presence of a "_YYMMDD" expiry suffix:
//
//   - "btcusdt"           → ContractPerpetual
//   - "btcusdt_240329"    → ContractUSDTFutures, Expiry "240329"
//
// Since H-7.c (ADR-0021 erratum) the expiry digits are PRESERVED
// into the canonical Expiry field — the venue suffix is already the
// canonical YYMMDD form, so delivery futures with different expiries
// no longer collapse into the same canonical identity (gap G10).
// Enabling delivery symbols at ingest remains gated by the G11
// enablement gaps (ClickHouse expiry persistence, read-contract
// param).
//
// Anything without a USDT quote (after suffix stripping) is rejected:
// `binancef` is the USDT-margined family by definition, and a
// non-USDT symbol on this connector signals a misconfiguration that
// must not become an invalid canonical instrument silently.
func parseFuturesSymbol(symbol string) (instrument.CanonicalInstrument, *problem.Problem) {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	if s == "" {
		return instrument.CanonicalInstrument{}, problem.Validation(
			problem.InvalidArgument,
			"binancef symbol is invalid",
			problem.ValidationIssue{Field: "symbol", Message: "must not be empty"},
		)
	}

	contract := instrument.ContractPerpetual
	expiry := ""
	if loc := deliverySuffix.FindStringIndex(s); loc != nil {
		contract = instrument.ContractUSDTFutures
		expiry = s[loc[0]+1:] // digits after the '_' — already canonical YYMMDD
		s = s[:loc[0]]
	}

	const quote = "USDT"
	if !strings.HasSuffix(s, quote) || len(s) <= len(quote) {
		return instrument.CanonicalInstrument{}, problem.Validation(
			problem.InvalidArgument,
			"binancef symbol is invalid",
			problem.ValidationIssue{
				Field:   "symbol",
				Message: "must end with USDT (binancef is the USDT-margined family)",
				Value:   symbol,
			},
		)
	}
	base := s[:len(s)-len(quote)]
	if expiry != "" {
		return instrument.NewDelivery(base, quote, contract, expiry)
	}
	return instrument.New(base, quote, contract)
}

func formatTradeID(id int64) string {
	// Fast int-to-string without fmt import.
	if id == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	if id < 0 {
		buf = append(buf, '-')
		id = -id
	}
	start := len(buf)
	for id > 0 {
		buf = append(buf, byte('0'+id%10))
		id /= 10
	}
	// Reverse the digits.
	for i, j := start, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
